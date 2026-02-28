package banner

import (
	"testing"
)

func TestPrint_DoesNotPanic(t *testing.T) {
	// Smoke test: calling Print should not panic regardless of host config.
	Print("0.0.0.0", 9800, "ClawIDE dev (commit: none, built: unknown)")
}

func TestPrint_SpecificHost(t *testing.T) {
	Print("192.168.1.42", 3000, "ClawIDE v1.2.3 (commit: abc, built: 2025-01-01)")
}

func TestDetectLANIP(t *testing.T) {
	ip := detectLANIP()
	// We can't assert a specific IP, but it should be non-empty on most dev machines.
	t.Logf("Detected LAN IP: %q", ip)
}

func TestRenderTerminalQR(t *testing.T) {
	qr := renderTerminalQR("http://localhost:9800")
	if qr == "" {
		t.Fatal("Expected non-empty QR code string")
	}
	t.Logf("QR output:\n%s", qr)
}
