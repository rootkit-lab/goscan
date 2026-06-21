package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// NotifyAvailable reports whether native desktop notifications can be sent.
func NotifyAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath("notify-send")
		return err == nil
	case "darwin":
		return true
	default:
		return false
	}
}

// Notify shows a native desktop notification when supported.
func Notify(title, body, iconPath string) error {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		title = "goscan"
	}
	switch runtime.GOOS {
	case "linux":
		return notifyLinux(title, body, iconPath)
	case "darwin":
		script := fmt.Sprintf(
			`display notification %q with title %q`,
			body, title,
		)
		return exec.Command("osascript", "-e", script).Run()
	default:
		return fmt.Errorf("notificações nativas não suportadas em %s", runtime.GOOS)
	}
}

// PlaySound plays a short alert (kind: "env" | "script_ok").
func PlaySound(kind string) error {
	switch runtime.GOOS {
	case "linux":
		return playLinux(kind)
	case "darwin":
		sound := "/System/Library/Sounds/Ping.aiff"
		if kind == "script_ok" {
			sound = "/System/Library/Sounds/Glass.aiff"
		}
		return exec.Command("afplay", sound).Run()
	default:
		return fmt.Errorf("som nativo não suportado em %s", runtime.GOOS)
	}
}

// ResolveIcon returns the first existing notification icon path.
func ResolveIcon(candidates ...string) string {
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func notifyLinux(title, body, iconPath string) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("notify-send em falta: %w", err)
	}
	args := []string{"-a", "goscan", "-t", "8000"}
	if iconPath != "" {
		args = append(args, "-i", iconPath)
	}
	args = append(args, title, body)
	cmd := exec.Command("notify-send", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("notify-send: %s: %w", msg, err)
		}
		return err
	}
	return nil
}

func playLinux(kind string) error {
	if kind == "env" {
		if err := playBeep(880, 120); err == nil {
			time.Sleep(130 * time.Millisecond)
			return playBeep(1100, 140)
		}
		if err := canberra("complete"); err == nil {
			time.Sleep(130 * time.Millisecond)
			return canberra("complete")
		}
		return paplaySound(
			"/usr/share/sounds/freedesktop/stereo/complete.oga",
			"/usr/share/sounds/gnome/default/alerts/complete.oga",
		)
	}
	if err := playBeep(660, 90); err == nil {
		time.Sleep(100 * time.Millisecond)
		return playBeep(880, 110)
	}
	if err := canberra("bell"); err == nil {
		return nil
	}
	return paplaySound(
		"/usr/share/sounds/freedesktop/stereo/bell.oga",
		"/usr/share/sounds/gnome/default/alerts/glass.oga",
	)
}

func playBeep(freq, ms int) error {
	if _, err := exec.LookPath("play"); err != nil {
		return err
	}
	dur := fmt.Sprintf("%.3f", float64(ms)/1000.0)
	cmd := exec.Command("play", "-q", "-n", "synth", dur, "sine", fmt.Sprintf("%d", freq))
	return cmd.Run()
}

func canberra(name string) error {
	if _, err := exec.LookPath("canberra-gtk-play"); err != nil {
		return err
	}
	return exec.Command("canberra-gtk-play", "-i", name).Run()
}

func paplaySound(paths ...string) error {
	if _, err := exec.LookPath("paplay"); err != nil {
		return fmt.Errorf("sem play/canberra-gtk-play/paplay")
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := exec.Command("paplay", p).Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("nenhum som do sistema encontrado")
}

// DefaultIconCandidates returns common goscan icon paths.
func DefaultIconCandidates(repoRoot string) []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(repoRoot, "assets/icon/goscan.png"),
		filepath.Join(home, ".local/share/icons/hicolor/256x256/apps/goscan.png"),
		"/usr/share/icons/hicolor/256x256/apps/goscan.png",
		"/usr/share/pixmaps/goscan.png",
	}
}