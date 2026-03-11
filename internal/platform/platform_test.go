package platform

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Parallel()

	info := Detect()
	if info.Arch != runtime.GOARCH {
		t.Fatalf("Detect().Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}

	wantOS := runtime.GOOS
	if runtime.GOOS == "darwin" {
		wantOS = "macos"
	}

	if info.OS != wantOS {
		t.Fatalf("Detect().OS = %q, want %q", info.OS, wantOS)
	}
}

func TestNormalizeOS(t *testing.T) {
	t.Parallel()

	if got := normalizeOS("darwin"); got != "macos" {
		t.Fatalf("normalizeOS(%q) = %q, want %q", "darwin", got, "macos")
	}

	if got := normalizeOS("linux"); got != "linux" {
		t.Fatalf("normalizeOS(%q) = %q, want %q", "linux", got, "linux")
	}
}
