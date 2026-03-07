//go:build windows

package common

import (
	"os"
	"syscall"
	"unsafe"
)

func enableVTInput() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetMode := kernel32.NewProc("GetConsoleMode")
	procSetMode := kernel32.NewProc("SetConsoleMode")
	h := syscall.Handle(os.Stdin.Fd())
	var mode uint32
	procGetMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	mode |= 0x0200 // ENABLE_VIRTUAL_TERMINAL_INPUT
	procSetMode.Call(uintptr(h), uintptr(mode))
}
