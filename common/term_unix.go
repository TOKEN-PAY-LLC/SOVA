//go:build !windows

package common

func enableVTInput() {
	// Unix terminals support VT escape sequences natively
}
