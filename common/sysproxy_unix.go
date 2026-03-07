//go:build !windows

package common

import (
	"fmt"
	"os/exec"
	"strings"
)

var savedProxyStateUnix string

// SetSystemProxy настраивает системный прокси через gsettings (GNOME) или env
func SetSystemProxy(proxyAddr string) error {
	parts := strings.Split(proxyAddr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid proxy address: %s", proxyAddr)
	}
	host := parts[0]
	port := parts[1]

	// Пробуем gsettings (GNOME/Ubuntu)
	if path, err := exec.LookPath("gsettings"); err == nil && path != "" {
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
		exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", port).Run()
		return nil
	}

	return fmt.Errorf("auto-proxy not supported on this system; configure SOCKS5 manually: %s", proxyAddr)
}

// ClearSystemProxy восстанавливает оригинальные настройки прокси
func ClearSystemProxy() error {
	if path, err := exec.LookPath("gsettings"); err == nil && path != "" {
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
		return nil
	}
	return nil
}

// IsSystemProxySet проверяет, установлен ли системный прокси
func IsSystemProxySet() bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "manual")
}
