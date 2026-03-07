package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"sova/common"
)

func main() {
	ui := common.NewUI(true)

	// CLI команды
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "help", "-h", "--help":
			ui.PrintBannerQuiet()
			printServerHelp()
			return
		case "version", "-v", "--version":
			fmt.Printf("SOVA Server v%s\n", common.Version)
			return
		case "config":
			handleServerConfig(ui)
			return
		}
	}

	// Запуск сервера
	startServer(ui)
}

func startServer(ui *common.UI) {
	ui.PrintBanner()

	// Загрузка конфигурации
	ui.PrintStatus("Loading server configuration...", common.Cyan)
	cfg, err := common.LoadConfig(common.GetConfigPath())
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Config error: %v, using defaults", err))
		cfg = common.DefaultConfig()
	}
	cfg.Mode = "server"
	ui.PrintSuccess("Configuration loaded")

	// Показать конфигурацию
	ui.PrintConfig(cfg)

	// Инициализация криптографии
	ui.PrintStatus("Initializing server cryptography...", common.Cyan)
	serverKeys, err := common.GenerateServerKeys()
	if err != nil {
		ui.ExitWithError(err)
	}
	if err := common.InitMasterKey(); err != nil {
		ui.ExitWithError(err)
	}
	if cfg.Encryption.PQEnabled {
		if err := common.InitPQKeys(); err != nil {
			ui.PrintWarning(fmt.Sprintf("PQ crypto: %v (continuing without PQ)", err))
		} else {
			ui.PrintSuccess("Post-quantum crypto initialized (Kyber1024 + Dilithium)")
		}
	}
	ui.PrintSuccess(fmt.Sprintf("Server public key: %x", serverKeys.PublicKey[:16]))

	// Инициализация middleware
	rateLimiter := NewRateLimiter(100)
	logger := NewLogger(1000)
	connMonitor := NewConnectionMonitor()

	// Запуск REST API + Dashboard
	if cfg.API.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting management API on %s:%d...", cfg.API.Host, cfg.API.Port), common.Cyan)
		api := NewServerAPI(serverKeys, rateLimiter, logger, connMonitor)
		api.StartAPI(cfg.API.Port)
		ui.PrintSuccess(fmt.Sprintf("Dashboard: http://%s:%d/", cfg.API.Host, cfg.API.Port))
		ui.PrintSuccess(fmt.Sprintf("API: http://%s:%d/api/", cfg.API.Host, cfg.API.Port))
	}

	// AI Adapter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Features.AIAdapter {
		ui.PrintStatus("Starting AI adaptive engine...", common.Cyan)
		switcher := common.NewAdaptiveSwitcher()
		go switcher.MonitorNetwork(ctx)
		ui.PrintSuccess("AI adapter active")
	}

	// Mesh + OfflineFirst
	if cfg.Features.MeshNetwork || cfg.Features.OfflineFirst {
		ui.PrintStatus("Initializing mesh network...", common.Cyan)
		offlineArch := common.NewOfflineFirstArchitecture("sova-server-1")
		if err := offlineArch.Start(ctx); err != nil {
			ui.PrintWarning(fmt.Sprintf("Offline-first: %v", err))
		} else {
			ui.PrintSuccess("Mesh + Connectivity + OfflineFirst active")
		}
	}

	// DNS-over-SOVA
	if cfg.DNS.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting DNS-over-SOVA on :%d...", cfg.DNS.Port), common.Cyan)
		dns := common.NewDNSResolver(cfg.DNS.Upstream)
		go dns.ListenAndServe(fmt.Sprintf(":%d", cfg.DNS.Port))
		ui.PrintSuccess(fmt.Sprintf("DNS resolver on :%d (upstream: %s)", cfg.DNS.Port, cfg.DNS.Upstream))
	}

	// Запуск relay сервера — основной обработчик клиентских подключений
	relayAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	ui.PrintStatus(fmt.Sprintf("Starting SOVA relay on %s...", relayAddr), common.Green)

	relay := NewRelayServer(relayAddr)
	if err := relay.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("relay server failed: %v", err))
	}

	ui.AnimateConnection()

	ui.PrintSection("SOVA Server Active")
	ui.PrintKeyValue("Relay:", relayAddr)
	if cfg.API.Enabled {
		ui.PrintKeyValue("API:", fmt.Sprintf("http://%s:%d/api/", cfg.API.Host, cfg.API.Port))
		ui.PrintKeyValue("Dashboard:", fmt.Sprintf("http://%s:%d/", cfg.API.Host, cfg.API.Port))
	}
	ui.PrintKeyValue("Protocol:", "SOVA v"+common.Version)
	ui.PrintKeyValue("Encryption:", cfg.Encryption.Algorithm)
	ui.PrintKeyValue("PQ Crypto:", boolStr(cfg.Encryption.PQEnabled))
	ui.PrintKeyValue("Stealth:", boolStr(cfg.Stealth.Enabled))
	ui.PrintKeyValue("AI Adapter:", boolStr(cfg.Features.AIAdapter))
	fmt.Println()
	ui.PrintDivider()
	fmt.Printf("%s  Clients connect: sova connect <server-ip>:%d%s\n", common.Dim, cfg.ServerPort, common.Reset)
	fmt.Printf("%s  Press Ctrl+C to stop%s\n", common.Dim, common.Reset)
	fmt.Println()

	// Ожидание сигнала
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			ui.PrintStatus("Shutting down SOVA server...", common.Yellow)
			relay.Stop()
			cancel()
			stats := relay.GetStats()
			ui.PrintSection("Server Session Summary")
			ui.PrintKeyValue("Total connections:", fmt.Sprintf("%d", stats["total_connections"]))
			ui.PrintKeyValue("Traffic relayed ↑:", formatBytes(stats["bytes_up"]))
			ui.PrintKeyValue("Traffic relayed ↓:", formatBytes(stats["bytes_down"]))
			fmt.Println()
			ui.PrintSuccess("SOVA server stopped.")
			return

		case <-ticker.C:
			stats := relay.GetStats()
			if stats["total_connections"] > 0 {
				ui.PrintStatus(fmt.Sprintf("Relay: active=%d total=%d ↑%s ↓%s",
					stats["active_connections"],
					stats["total_connections"],
					formatBytes(stats["bytes_up"]),
					formatBytes(stats["bytes_down"]),
				), common.Dim+common.Purple)
			}
		}
	}
}

func handleServerConfig(ui *common.UI) {
	cfg, _ := common.LoadConfig(common.GetConfigPath())
	if len(os.Args) < 3 {
		ui.PrintBannerQuiet()
		ui.PrintConfig(cfg)
		return
	}

	if os.Args[2] == "set" && len(os.Args) >= 5 {
		key := os.Args[3]
		value := os.Args[4]
		switch key {
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.ServerPort = port
		case "api_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid port: %s", value))
			}
			cfg.API.Port = port
		default:
			// Попробовать как feature toggle
			if value == "true" || value == "false" {
				cfg.SetFeature(key, value == "true")
			} else {
				ui.ExitWithError(fmt.Errorf("unknown config key: %s", key))
			}
		}
		cfg.Save(common.GetConfigPath())
		ui.PrintSuccess(fmt.Sprintf("Config updated: %s = %s", key, value))
	}
}

func printServerHelp() {
	fmt.Println(common.Cyan + "  sova-server" + common.Reset + "                  Start SOVA relay server")
	fmt.Println(common.Cyan + "  sova-server config" + common.Reset + "           Show configuration")
	fmt.Println(common.Cyan + "  sova-server config set <k> <v>" + common.Reset + " Update config")
	fmt.Println(common.Cyan + "  sova-server help" + common.Reset + "             Show this help")
	fmt.Println(common.Cyan + "  sova-server version" + common.Reset + "          Show version")
	fmt.Println()
}

func boolStr(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

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
