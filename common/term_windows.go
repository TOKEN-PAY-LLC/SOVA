//go:build windows

package common

import (
	"os"
	"syscall"
	"unsafe"
)

// EnableVTMode enables Virtual Terminal processing on Windows console.
// MUST be called before ANY ANSI escape output, otherwise PS 5.1 shows raw codes.
func EnableVTMode() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetMode := kernel32.NewProc("GetConsoleMode")
	procSetMode := kernel32.NewProc("SetConsoleMode")

	// STDOUT: enable ENABLE_VIRTUAL_TERMINAL_PROCESSING (0x0004)
	stdout := syscall.Handle(os.Stdout.Fd())
	var outMode uint32
	procGetMode.Call(uintptr(stdout), uintptr(unsafe.Pointer(&outMode)))
	outMode |= 0x0004
	procSetMode.Call(uintptr(stdout), uintptr(outMode))

	// STDERR: same
	stderr := syscall.Handle(os.Stderr.Fd())
	var errMode uint32
	procGetMode.Call(uintptr(stderr), uintptr(unsafe.Pointer(&errMode)))
	errMode |= 0x0004
	procSetMode.Call(uintptr(stderr), uintptr(errMode))
}

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
