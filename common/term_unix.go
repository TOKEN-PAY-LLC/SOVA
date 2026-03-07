//go:build !windows

package common

// EnableVTMode is a no-op on Unix — terminals support VT natively
func EnableVTMode() {}

func enableVTInput() {}
