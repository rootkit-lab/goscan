export type AlertPrefs = {
  notifyEnvFound: boolean;
  notifyScriptOk: boolean;
  soundEnvFound: boolean;
  soundScriptOk: boolean;
};

let audioCtx: AudioContext | null = null;

function ctx(): AudioContext {
  if (!audioCtx) audioCtx = new AudioContext();
  return audioCtx;
}

function beep(freq: number, ms: number, gain = 0.08) {
  try {
    const ac = ctx();
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

export function playEnvFoundSound() {
  beep(880, 120);
  setTimeout(() => beep(1100, 140), 130);
}

export function playScriptOkSound() {
  beep(660, 90);
  setTimeout(() => beep(880, 110), 100);
}

export async function showDesktopNotification(title: string, body: string) {
  if (!("Notification" in window)) return;
  try {
    if (Notification.permission === "default") {
      await Notification.requestPermission();
    }
    if (Notification.permission === "granted") {
      new Notification(title, { body, silent: true });
    }
  } catch {
    /* ignore notification errors */
  }
}

export function alertEnvFound(prefs: AlertPrefs, domain: string, path: string) {
  if (prefs.soundEnvFound) playEnvFoundSound();
  if (prefs.notifyEnvFound) {
    void showDesktopNotification("Novo .env", `${domain}${path}`);
  }
}

export function alertScriptOk(prefs: AlertPrefs, domain: string, scriptLabel: string, summary: string) {
  if (prefs.soundScriptOk) playScriptOkSound();
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
    soundEnvFound: s.soundEnvFound ?? false,
    soundScriptOk: s.soundScriptOk ?? true
  };
}
