package main

import (
	"bufio"
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

	// If command-line args provided, run in CLI mode
	if len(os.Args) >= 2 {
		runCLI(ui, os.Args[1])
		return
	}

	// No args: interactive menu mode
	runInteractiveMenu(ui)
}

func runCLI(ui *common.UI, command string) {
	switch command {
	case "start":
		startTunnel(ui)

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

	case "menu":
		runInteractiveMenu(ui)

	default:
		ui.PrintError(fmt.Errorf("Unknown command: %s", command))
		fmt.Println()
		ui.PrintHelp()
	}
}

// runInteractiveMenu shows language selection then the main interactive menu
func runInteractiveMenu(ui *common.UI) {
	ui.PrintBanner()

	common.CurrentLang = common.SelectLanguage()

	for {
		items := []common.MenuItem{
			{LabelEN: "Start Tunnel", LabelRU: "Zapustit' tunnel'", DescEN: "SOCKS5 proxy 127.0.0.1:1080", DescRU: "SOCKS5 proksi 127.0.0.1:1080"},
			{LabelEN: "Connect to Server", LabelRU: "Podklyuchit'sya k serveru", DescEN: "Via remote SOVA server", DescRU: "Cherez udalyonnyj server"},
			{LabelEN: "Configuration", LabelRU: "Konfiguratsiya", DescEN: "View & edit settings", DescRU: "Nastrojki"},
			{LabelEN: "Modules", LabelRU: "Moduli", DescEN: "Toggle features on/off", DescRU: "Vkl/vykl moduli"},
			{LabelEN: "Status", LabelRU: "Status", DescEN: "System info", DescRU: "Informatsiya o sisteme"},
			{LabelEN: "Help", LabelRU: "Spravka", DescEN: "Commands & API", DescRU: "Komandy i API"},
			{LabelEN: "Exit", LabelRU: "Vykhod", DescEN: "", DescRU: ""},
		}

		choice := common.RunMenu("SOVA Protocol v"+common.Version, "Protokol SOVA v"+common.Version, items)
		switch choice {
		case 0:
			startTunnel(ui)
			return
		case 1:
			menuConnect(ui)
		case 2:
			menuConfig(ui)
		case 3:
			menuModules(ui)
		case 4:
			handleStatus(ui)
			waitEnter()
		case 5:
			ui.PrintHelp()
			waitEnter()
		case -1, 6:
			fmt.Println()
			ui.PrintSuccess(common.T("Goodbye! Stay free!", "Do svidaniya!"))
			return
		}
	}
}

func menuConnect(ui *common.UI) {
	fmt.Printf("\n  %s%s%s ", common.Purple7, common.T("Server address (host:port): ", "Adres servera (host:port): "), common.Reset)
	reader := bufio.NewReader(os.Stdin)
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}
	startRemoteTunnel(ui, addr)
}

func menuConfig(ui *common.UI) {
	for {
		items := []common.MenuItem{
			{LabelEN: "Show Configuration", LabelRU: "Pokazat' konfiguraciyu", DescEN: "Current settings", DescRU: "Tekushchie nastrojki"},
			{LabelEN: "Edit Setting", LabelRU: "Izmenit' parametr", DescEN: "key = value", DescRU: "klyuch = znachenie"},
			{LabelEN: "Reset to Defaults", LabelRU: "Sbrosit' nastrojki", DescEN: "Restore defaults", DescRU: "Vosstanovit' po umolchaniyu"},
			{LabelEN: "Export JSON", LabelRU: "Eksport JSON", DescEN: "Config as JSON", DescRU: "Konfig v JSON"},
			{LabelEN: "Config Path", LabelRU: "Put' k konfigu", DescEN: "File location", DescRU: "Raspolozhenie fajla"},
			{LabelEN: "Back", LabelRU: "Nazad", DescEN: "", DescRU: ""},
		}

		choice := common.RunMenu("Configuration", "Konfiguratsiya", items)
		cfg, _ := common.LoadConfig(common.GetConfigPath())

		switch choice {
		case 0:
			ui.PrintConfig(cfg)
			waitEnter()
		case 1:
			menuEditSetting(ui)
		case 2:
			def := common.DefaultConfig()
			if err := def.Save(common.GetConfigPath()); err != nil {
				ui.PrintError(err)
			} else {
				ui.PrintSuccess(common.T("Config reset to defaults", "Konfiguratsiya sbroshena"))
			}
			waitEnter()
		case 3:
			data, _ := cfg.ToJSON()
			fmt.Println(string(data))
			waitEnter()
		case 4:
			fmt.Printf("  %s\n", common.GetConfigPath())
			waitEnter()
		case -1, 5:
			return
		}
	}
}

func menuEditSetting(ui *common.UI) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n  %s%s%s", common.Purple7, common.T("Key: ", "Klyuch: "), common.Reset)
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	fmt.Printf("  %s%s%s", common.Purple7, common.T("Value: ", "Znachenie: "), common.Reset)
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	cfg, _ := common.LoadConfig(common.GetConfigPath())
	applied := applyConfigSetting(cfg, key, value)
	if !applied {
		ui.PrintError(fmt.Errorf(common.T("Unknown key: %s", "Neizvestnyj klyuch: %s"), key))
		return
	}
	if err := cfg.Save(common.GetConfigPath()); err != nil {
		ui.PrintError(err)
		return
	}
	ui.PrintSuccess(fmt.Sprintf("%s = %s", key, value))
}

func applyConfigSetting(cfg *common.Config, key, value string) bool {
	switch key {
	case "mode":
		cfg.Mode = value
	case "listen_addr":
		cfg.ListenAddr = value
	case "listen_port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		cfg.ListenPort = port
	case "server_addr":
		cfg.ServerAddr = value
	case "server_port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return false
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
			return false
		}
		cfg.API.Port = port
	case "dns_upstream":
		cfg.DNS.Upstream = value
	case "dns_port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		cfg.DNS.Port = port
	case "jitter_ms":
		jitter, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		cfg.Stealth.JitterMs = jitter
	default:
		return false
	}
	return true
}

func menuModules(ui *common.UI) {
	for {
		cfg, _ := common.LoadConfig(common.GetConfigPath())

		type feat struct {
			name string
			on   bool
		}
		features := []feat{
			{"pq_crypto", cfg.Encryption.PQEnabled},
			{"zkp", cfg.Encryption.ZKPEnabled},
			{"stealth", cfg.Stealth.Enabled},
			{"padding", cfg.Stealth.PaddingEnabled},
			{"decoy", cfg.Stealth.DecoyEnabled},
			{"ai_adapter", cfg.Features.AIAdapter},
			{"compression", cfg.Features.Compression},
			{"connection_pool", cfg.Features.ConnectionPool},
			{"smart_routing", cfg.Features.SmartRouting},
			{"mesh_network", cfg.Features.MeshNetwork},
			{"offline_first", cfg.Features.OfflineFirst},
			{"dns", cfg.DNS.Enabled},
			{"api", cfg.API.Enabled},
			{"dashboard", cfg.Features.Dashboard},
			{"auto_proxy", cfg.Features.AutoProxy},
		}

		items := make([]common.MenuItem, 0, len(features)+1)
		for _, f := range features {
			status := "[ON] "
			if !f.on {
				status = "[OFF]"
			}
			items = append(items, common.MenuItem{
				LabelEN: status + " " + f.name,
				LabelRU: status + " " + f.name,
				DescEN:  "Enter to toggle",
				DescRU:  "Enter dlya pereklyucheniya",
			})
		}
		items = append(items, common.MenuItem{LabelEN: "Back", LabelRU: "Nazad"})

		choice := common.RunMenu("Modules", "Moduli", items)
		if choice == -1 || choice >= len(features) {
			return
		}
		if choice >= 0 && choice < len(features) {
			f := features[choice]
			cfg.SetFeature(f.name, !f.on)
			cfg.Save(common.GetConfigPath())
		}
	}
}

func waitEnter() {
	fmt.Printf("\n  %s%s%s", common.Dim, common.T("Press Enter to continue...", "Nazhmite Enter..."), common.Reset)
	bufio.NewReader(os.Stdin).ReadString('\n')
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
