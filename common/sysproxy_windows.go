//go:build windows

package common

import (
	"fmt"
	"os/exec"
	"strings"
)

// SystemProxyState хранит оригинальные настройки прокси для восстановления
type SystemProxyState struct {
	WasEnabled    bool
	OriginalProxy string
}

var savedProxyState *SystemProxyState

// SetSystemProxy настраивает системный прокси Windows через реестр
func SetSystemProxy(proxyAddr string) error {
	// Сохраняем текущие настройки для восстановления
	savedProxyState = &SystemProxyState{}

	// Читаем текущий ProxyEnable
	out, err := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable").CombinedOutput()
	if err == nil && strings.Contains(string(out), "0x1") {
		savedProxyState.WasEnabled = true
	}

	// Читаем текущий ProxyServer
	out, err = exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer").CombinedOutput()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "ProxyServer") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					savedProxyState.OriginalProxy = parts[len(parts)-1]
				}
			}
		}
	}

	// Устанавливаем SOCKS прокси
	// Windows Internet Settings: socks=addr:port
	socksProxy := fmt.Sprintf("socks=%s", proxyAddr)

	// Включаем прокси
	err = exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
	if err != nil {
		return fmt.Errorf("failed to enable proxy: %v", err)
	}

	// Устанавливаем адрес прокси
	err = exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyServer", "/t", "REG_SZ", "/d", socksProxy, "/f").Run()
	if err != nil {
		return fmt.Errorf("failed to set proxy address: %v", err)
	}

	// Добавляем bypass для локальных адресов
	err = exec.Command("reg", "add",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyOverride", "/t", "REG_SZ", "/d",
		"localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;<local>",
		"/f").Run()
	if err != nil {
		return fmt.Errorf("failed to set proxy bypass: %v", err)
	}

	// Уведомляем систему об изменении настроек интернета
	exec.Command("rundll32.exe", "wininet.dll,InternetSetOptionW", "39", "0", "0").Run()

	return nil
}

// ClearSystemProxy восстанавливает оригинальные настройки прокси
func ClearSystemProxy() error {
	if savedProxyState == nil {
		// Просто выключаем прокси
		return exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	}

	if savedProxyState.WasEnabled && savedProxyState.OriginalProxy != "" {
		// Восстанавливаем оригинальный прокси
		exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyServer", "/t", "REG_SZ", "/d", savedProxyState.OriginalProxy, "/f").Run()
	} else {
		// Выключаем прокси
		exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	}

	// Уведомляем систему
	exec.Command("rundll32.exe", "wininet.dll,InternetSetOptionW", "39", "0", "0").Run()

	savedProxyState = nil
	return nil
}

// IsSystemProxySet проверяет, установлен ли системный прокси
func IsSystemProxySet() bool {
	out, err := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", "ProxyEnable").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "0x1")
}
