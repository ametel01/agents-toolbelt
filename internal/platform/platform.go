// Package platform provides normalized runtime platform detection.
package platform

import "runtime"

// Info describes the current operating system and CPU architecture.
type Info struct {
	OS   string
	Arch string
}

// Detect returns the normalized runtime platform information.
func Detect() Info {
	return Info{
		OS:   normalizeOS(runtime.GOOS),
		Arch: runtime.GOARCH,
	}
}

func normalizeOS(os string) string {
	if os == "darwin" {
		return "macos"
	}

	return os
}
