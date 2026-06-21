import { api } from "@/lib/api";

export type AlertPrefs = {
  notifyEnvFound: boolean;
  notifyScriptOk: boolean;
  soundEnvFound: boolean;
  soundScriptOk: boolean;
};

let audioCtx: AudioContext | null = null;
let audioUnlocked = false;

function ctx(): AudioContext {
  if (!audioCtx) audioCtx = new AudioContext();
  return audioCtx;
}

/** Call once after user interaction so WebAudio fallback can play during scan. */
export function unlockWebAudio() {
  if (audioUnlocked) return;
  audioUnlocked = true;
  try {
    const ac = ctx();
    if (ac.state === "suspended") {
      void ac.resume();
    }
  } catch {
    /* ignore */
  }
}

function beepWeb(freq: number, ms: number, gain = 0.08) {
  try {
    const ac = ctx();
    if (ac.state === "suspended") {
      void ac.resume();
    }
    const osc = ac.createOscillator();
    const amp = ac.createGain();
    osc.type = "sine";
    osc.frequency.value = freq;
    amp.gain.value = gain;
    osc.connect(amp);
    amp.connect(ac.destination);
    osc.start();
    setTimeout(() => {
      osc.stop();
      osc.disconnect();
      amp.disconnect();
    }, ms);
  } catch {
    /* ignore audio errors */
  }
}

function playEnvFoundSoundWeb() {
  beepWeb(880, 120);
  setTimeout(() => beepWeb(1100, 140), 130);
}

function playScriptOkSoundWeb() {
  beepWeb(660, 90);
  setTimeout(() => beepWeb(880, 110), 100);
}

async function playNativeSound(kind: "env" | "script_ok"): Promise<boolean> {
  try {
    await api.playAlertSound(kind);
    return true;
  } catch {
    return false;
  }
}

export async function playEnvFoundSound() {
  if (await playNativeSound("env")) return;
  playEnvFoundSoundWeb();
}

export async function playScriptOkSound() {
  if (await playNativeSound("script_ok")) return;
  playScriptOkSoundWeb();
}

async function showWebNotification(title: string, body: string) {
  if (!("Notification" in window)) return false;
  try {
    if (Notification.permission === "default") {
      await Notification.requestPermission();
    }
    if (Notification.permission === "granted") {
      new Notification(title, { body, silent: true });
      return true;
    }
  } catch {
    /* ignore */
  }
  return false;
}

export async function showDesktopNotification(title: string, body: string) {
  try {
    const ok = await api.desktopNotify(title, body);
    if (ok) return;
  } catch {
    /* fallback below */
  }
  await showWebNotification(title, body);
}

export function alertEnvFound(prefs: AlertPrefs, domain: string, path: string) {
  if (prefs.soundEnvFound) void playEnvFoundSound();
  if (prefs.notifyEnvFound) {
    void showDesktopNotification("Novo .env", `${domain}${path}`);
  }
}

export function alertScriptOk(prefs: AlertPrefs, domain: string, scriptLabel: string, summary: string) {
  if (prefs.soundScriptOk) void playScriptOkSound();
  if (prefs.notifyScriptOk) {
    const body = summary ? `${scriptLabel} — ${summary}` : scriptLabel;
    void showDesktopNotification(`${domain} · OK`, body);
  }
}

export function alertPrefsFromSettings(s: {
  notifyEnvFound?: boolean;
  notifyScriptOk?: boolean;
  soundEnvFound?: boolean;
  soundScriptOk?: boolean;
}): AlertPrefs {
  return {
    notifyEnvFound: s.notifyEnvFound ?? true,
    notifyScriptOk: s.notifyScriptOk ?? true,
    soundEnvFound: s.soundEnvFound ?? true,
    soundScriptOk: s.soundScriptOk ?? true
  };
}

export type AlertTestResult = {
  soundOk: boolean;
  notifyOk: boolean;
  notifyAvailable: boolean;
  error?: string;
};

export async function testEnvAlert(prefs: AlertPrefs): Promise<AlertTestResult> {
  unlockWebAudio();
  try {
    return await api.testEnvAlert(prefs.notifyEnvFound, prefs.soundEnvFound);
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    return { soundOk: false, notifyOk: false, notifyAvailable: false, error: msg };
  }
}
