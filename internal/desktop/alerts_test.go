package desktop

import "testing"

func TestNotifyAvailableLinux(t *testing.T) {
	// Smoke: must not panic on any platform.
	_ = NotifyAvailable()
}

func TestResolveIconEmpty(t *testing.T) {
	if got := ResolveIcon(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
