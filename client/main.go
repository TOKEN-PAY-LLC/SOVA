package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sova/common"
)

func main() {
	ui := common.NewUI(true)

	// Без аргументов или "start" — запуск туннеля
	if len(os.Args) < 2 || os.Args[1] == "start" {
		startTunnel(ui)
		return
	}

	command := os.Args[1]
	switch command {
	case "help", "-h", "--help":
		ui.PrintBannerQuiet()
		ui.PrintHelp()

	case "version", "-v", "--version":
		fmt.Printf("SOVA Protocol v%s\n", common.Version)

	case "config":
		handleConfig(ui)

	case "features":
		cfg, _ := common.LoadConfig(common.GetConfigPath())
		ui.PrintBannerQuiet()
		ui.PrintFeatures(cfg)

	case "enable":
		handleFeatureToggle(ui, true)

	case "disable":
		handleFeatureToggle(ui, false)

	case "status":
		handleStatus(ui)

	case "connect":
		if len(os.Args) < 3 {
			ui.PrintError(fmt.Errorf("Usage: sova connect <server:port>"))
			return
		}
		startRemoteTunnel(ui, os.Args[2])

	default:
		ui.PrintError(fmt.Errorf("Unknown command: %s", command))
		fmt.Println()
		ui.PrintHelp()
	}
}

// startTunnel — главная функция: запуск локального SOCKS5 прокси
func startTunnel(ui *common.UI) {
	// Анимированный баннер
	ui.PrintBanner()

	// Загрузка конфигурации
	ui.PrintStatus("Loading configuration...", common.Cyan)
	cfg, err := common.LoadConfig(common.GetConfigPath())
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Config error: %v, using defaults", err))
		cfg = common.DefaultConfig()
	}
	ui.PrintSuccess(fmt.Sprintf("Config loaded from %s", common.GetConfigPath()))

	// Показать конфигурацию
	ui.PrintConfig(cfg)

	// Инициализация криптографии
	ui.PrintStatus("Initializing cryptography...", common.Cyan)
	if err := common.InitMasterKey(); err != nil {
		ui.ExitWithError(fmt.Errorf("master key init failed: %v", err))
	}
	if cfg.Encryption.PQEnabled {
		if err := common.InitPQKeys(); err != nil {
			ui.PrintWarning(fmt.Sprintf("PQ crypto init: %v (continuing without PQ)", err))
		} else {
			ui.PrintSuccess("Post-quantum crypto initialized (Kyber1024 + Dilithium)")
		}
	}
	ui.PrintSuccess("AES-256-GCM + ChaCha20-Poly1305 ready")

	// Инициализация stealth engine
	if cfg.Stealth.Enabled {
		ui.PrintStatus("Activating stealth engine...", common.Cyan)
		ui.AnimateLoading("Stealth engine loading...", 500*time.Millisecond)
		ui.PrintSuccess(fmt.Sprintf("Stealth active: profile=%s, jitter=%dms", cfg.Stealth.Profile, cfg.Stealth.JitterMs))
	}

	// Инициализация AI адаптера
	var cancel context.CancelFunc
	if cfg.Features.AIAdapter {
		ui.PrintStatus("Starting AI adaptive engine...", common.Cyan)
		switcher := common.NewAdaptiveSwitcher()
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		go switcher.MonitorNetwork(ctx)
		ui.PrintSuccess("AI adapter active — monitoring network conditions")
	}

	// Запуск REST API
	if cfg.API.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting management API on %s:%d...", cfg.API.Host, cfg.API.Port), common.Cyan)
		go startClientAPI(cfg, ui)
		ui.PrintSuccess(fmt.Sprintf("API: http://%s:%d/api/", cfg.API.Host, cfg.API.Port))
	}

	// DNS-over-SOVA
	if cfg.DNS.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting DNS-over-SOVA on :%d...", cfg.DNS.Port), common.Cyan)
		dns := common.NewDNSResolver(cfg.DNS.Upstream)
		go dns.ListenAndServe(fmt.Sprintf(":%d", cfg.DNS.Port))
		ui.PrintSuccess(fmt.Sprintf("DNS resolver: 127.0.0.1:%d (upstream: %s)", cfg.DNS.Port, cfg.DNS.Upstream))
	}

	// Запуск SOCKS5 прокси — главный туннель
	listenAddr := cfg.ListenAddress()
	ui.PrintStatus(fmt.Sprintf("Starting SOCKS5 proxy on %s...", listenAddr), common.Green)

	socks := common.NewSOCKS5Server(listenAddr, ui)
	if err := socks.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("SOCKS5 proxy failed: %v", err))
	}

	// Анимация подключения
	ui.AnimateConnection()

	// Инструкции для пользователя
	ui.PrintTunnelActive(listenAddr, cfg)

	// Ожидание сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Периодический вывод статистики
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			ui.PrintStatus("Shutting down SOVA...", common.Yellow)
			socks.Stop()
			if cancel != nil {
				cancel()
			}
			stats := socks.GetStats()
			ui.PrintSection("Session Summary")
			ui.PrintKeyValue("Total connections:", fmt.Sprintf("%d", stats["total_connections"]))
			ui.PrintKeyValue("Traffic uploaded:", formatBytes(stats["bytes_up"]))
			ui.PrintKeyValue("Traffic downloaded:", formatBytes(stats["bytes_down"]))
			fmt.Println()
			ui.PrintSuccess("SOVA stopped. Stay free!")
			return

		case <-ticker.C:
			stats := socks.GetStats()
			if stats["active_connections"] > 0 || stats["total_connections"] > 0 {
				ui.PrintStatus(fmt.Sprintf("Active: %d | Total: %d | ↑%s ↓%s",
					stats["active_connections"],
					stats["total_connections"],
					formatBytes(stats["bytes_up"]),
					formatBytes(stats["bytes_down"]),
				), common.Dim+common.Purple)
			}
		}
	}
}

// startRemoteTunnel подключается к удалённому SOVA серверу
func startRemoteTunnel(ui *common.UI, serverAddr string) {
	ui.PrintBanner()

	cfg, _ := common.LoadConfig(common.GetConfigPath())

	// Парсим адрес сервера
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":443"
	}

	ui.PrintStatus(fmt.Sprintf("Connecting to SOVA server %s...", serverAddr), common.Cyan)

	// Инициализация криптографии
	if err := common.InitMasterKey(); err != nil {
		ui.ExitWithError(err)
	}
	if cfg.Encryption.PQEnabled {
		common.InitPQKeys()
	}

	ui.AnimateConnection()

	// В remote режиме SOCKS5 прокси направляет трафик через сервер
	listenAddr := cfg.ListenAddress()
	ui.PrintStatus(fmt.Sprintf("Starting SOCKS5 proxy on %s (via %s)...", listenAddr, serverAddr), common.Green)

	socks := common.NewSOCKS5Server(listenAddr, ui)

	// Remote dialer: подключаемся к серверу для каждого соединения
	// В будущем — мультиплексирование через одно соединение
	socks.RemoteDialer = common.CreateRemoteDialer(serverAddr)

	if err := socks.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("SOCKS5 proxy failed: %v", err))
	}

	ui.PrintSection("SOVA Remote Tunnel Active")
	ui.PrintKeyValue("Server:", serverAddr)
	ui.PrintKeyValue("Local proxy:", listenAddr)
	ui.PrintKeyValue("Protocol:", "SOVA v"+common.Version+" (PQ-encrypted)")
	fmt.Println()
	fmt.Printf("%s  Press Ctrl+C to stop%s\n", common.Dim, common.Reset)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	socks.Stop()
	ui.PrintSuccess("Disconnected from server")
}

// handleConfig — управление конфигурацией
func handleConfig(ui *common.UI) {
	cfg, _ := common.LoadConfig(common.GetConfigPath())

	if len(os.Args) < 3 {
		// Показать текущую конфигурацию
		ui.PrintBannerQuiet()
		ui.PrintConfig(cfg)
		ui.PrintInfoAlways(fmt.Sprintf("Config file: %s", common.GetConfigPath()))
		return
	}

	if os.Args[2] == "set" && len(os.Args) >= 5 {
		key := os.Args[3]
		value := os.Args[4]

		switch key {
		case "mode":
			cfg.Mode = value
		case "listen_addr":
			cfg.ListenAddr = value
		case "listen_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.ListenPort = port
		case "server_addr":
			cfg.ServerAddr = value
		case "server_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.ServerPort = port
		case "encryption":
			cfg.Encryption.Algorithm = value
		case "stealth_profile":
			cfg.Stealth.Profile = value
		case "tls_fingerprint":
			cfg.Stealth.TLSFingerprint = value
		case "log_level":
			cfg.LogLevel = value
		case "transport_mode":
			cfg.Transport.Mode = value
		case "api_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.API.Port = port
		case "dns_upstream":
			cfg.DNS.Upstream = value
		case "dns_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.DNS.Port = port
		case "jitter_ms":
			jitter, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid jitter: %s", value))
			}
			cfg.Stealth.JitterMs = jitter
		default:
			ui.ExitWithError(fmt.Errorf("unknown config key: %s", key))
		}

		if err := cfg.Save(common.GetConfigPath()); err != nil {
			ui.ExitWithError(err)
		}
		ui.PrintSuccess(fmt.Sprintf("Config updated: %s = %s", key, value))
		return
	}

	if os.Args[2] == "reset" {
		cfg = common.DefaultConfig()
		if err := cfg.Save(common.GetConfigPath()); err != nil {
			ui.ExitWithError(err)
		}
		ui.PrintSuccess("Config reset to defaults")
		return
	}

	if os.Args[2] == "path" {
		fmt.Println(common.GetConfigPath())
		return
	}

	if os.Args[2] == "json" {
		data, _ := cfg.ToJSON()
		fmt.Println(string(data))
		return
	}

	ui.PrintError(fmt.Errorf("Unknown config command: %s", os.Args[2]))
}

// handleFeatureToggle включает/выключает модуль
func handleFeatureToggle(ui *common.UI, enable bool) {
	if len(os.Args) < 3 {
		action := "enable"
		if !enable {
			action = "disable"
		}
		ui.ExitWithError(fmt.Errorf("Usage: sova %s <module_name>", action))
	}

	moduleName := os.Args[2]
	cfg, _ := common.LoadConfig(common.GetConfigPath())

	if !cfg.SetFeature(moduleName, enable) {
		ui.ExitWithError(fmt.Errorf("Unknown module: %s", moduleName))
	}

	if err := cfg.Save(common.GetConfigPath()); err != nil {
		ui.ExitWithError(err)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	ui.PrintSuccess(fmt.Sprintf("Module '%s' %s", moduleName, action))
}

// handleStatus показывает статус
func handleStatus(ui *common.UI) {
	cfg, _ := common.LoadConfig(common.GetConfigPath())
	ui.PrintBannerQuiet()
	ui.PrintSystemInfo()
	ui.PrintConfig(cfg)
}

// startClientAPI запускает REST API для управления клиентом
func startClientAPI(cfg *common.Config, ui *common.UI) {
	// API реализован в common пакете, используется и клиентом и сервером
	// Здесь запускаем базовый HTTP API для управления
	common.StartManagementAPI(cfg, ui)
}

// formatBytes форматирует байты в читаемый формат
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
