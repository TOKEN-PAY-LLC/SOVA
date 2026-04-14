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

	case "core":
		startCore(ui)

	case "help", "-h", "--help":
		ui.PrintBannerQuiet()
		ui.PrintHelp()

	case "version", "-v", "--version":
		fmt.Printf("SOVA Core v%s\n", common.Version)

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
		// Проверяем текущий статус системного прокси
		proxyStatus := common.T("[OFF]", "[ВЫКЛ]")
		if common.IsSystemProxySet() {
			proxyStatus = common.T("[ON]", "[ВКЛ]")
		}

		// Upstream proxy info
		cfg, _ := common.LoadConfig(common.GetConfigPath())
		upstreamDesc := common.T("Not set — direct traffic", "Не задан — прямой трафик")
		if cfg.UpstreamProxy != "" {
			upstreamDesc = cfg.UpstreamProxy
		}

		items := []common.MenuItem{
			{LabelEN: "Start SOVA Core", LabelRU: "Запустить SOVA Core", DescEN: "Core engine + routing + auto-proxy", DescRU: "Ядро + маршрутизация + авто-прокси"},
			{LabelEN: "Start Tunnel (Legacy)", LabelRU: "Запустить туннель", DescEN: "SOVA Proxy + auto-route", DescRU: "SOVA Proxy + авто-маршрут"},
			{LabelEN: "Connect to Server", LabelRU: "Подключиться к серверу", DescEN: "Via remote SOVA server", DescRU: "Через удалённый сервер"},
			{LabelEN: "SOVA Gateway", LabelRU: "Шлюз SOVA", DescEN: upstreamDesc, DescRU: upstreamDesc},
			{LabelEN: "System Proxy " + proxyStatus, LabelRU: "Системный прокси " + proxyStatus, DescEN: "Route ALL traffic through SOVA", DescRU: "Весь трафик через SOVA"},
			{LabelEN: "Configuration", LabelRU: "Конфигурация", DescEN: "View & edit settings", DescRU: "Настройки"},
			{LabelEN: "Modules", LabelRU: "Модули", DescEN: "Toggle features on/off", DescRU: "Вкл/выкл модули"},
			{LabelEN: "Status", LabelRU: "Статус", DescEN: "System info", DescRU: "Информация о системе"},
			{LabelEN: "Help", LabelRU: "Справка", DescEN: "Commands & API", DescRU: "Команды и API"},
			{LabelEN: "Exit", LabelRU: "Выход", DescEN: "", DescRU: ""},
		}

		choice := common.RunMenu("SOVA Core v"+common.Version, "SOVA Core v"+common.Version, items)
		switch choice {
		case 0:
			startCore(ui)
			return
		case 1:
			startTunnel(ui)
			return
		case 2:
			menuConnect(ui)
		case 3:
			menuUpstreamProxy(ui)
		case 4:
			menuSystemProxy(ui)
		case 5:
			menuConfig(ui)
		case 6:
			menuModules(ui)
		case 7:
			handleStatus(ui)
			waitEnter()
		case 8:
			ui.PrintHelp()
			waitEnter()
		case -1, 9:
			fmt.Println()
			ui.PrintSuccess(common.T("Goodbye! Stay free!", "До свидания! Оставайтесь свободными!"))
			return
		}
	}
}

func menuConnect(ui *common.UI) {
	fmt.Printf("\n  %s%s%s ", common.Purple7, common.T("Server address (host:port): ", "Адрес сервера (host:port): "), common.Reset)
	reader := bufio.NewReader(os.Stdin)
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}
	startRemoteTunnel(ui, addr)
}

func menuUpstreamProxy(ui *common.UI) {
	cfg, _ := common.LoadConfig(common.GetConfigPath())

	fmt.Println()
	if cfg.UpstreamProxy != "" {
		fmt.Printf("  %s%s%s %s%s%s\n", common.Purple7, common.Bold,
			common.T("Current gateway:", "Текущий шлюз:"), common.Reset+common.Gold+common.Bold, cfg.UpstreamProxy, common.Reset)
	} else {
		fmt.Printf("  %s%s%s\n", common.Yellow,
			common.T("No upstream gateway set — traffic goes direct from your IP", "Шлюз не задан — трафик идёт напрямую с вашего IP"), common.Reset)
	}

	fmt.Println()
	fmt.Printf("  %s%s%s\n", common.Dim,
		common.T("Formats: sova://host:port  http://host:port  host:port",
			"Форматы: sova://host:port  http://host:port  host:port"), common.Reset)
	fmt.Printf("  %s%s%s\n", common.Dim,
		common.T("Enter 'clear' to remove upstream gateway",
			"Введите 'clear' чтобы убрать шлюз"), common.Reset)
	fmt.Println()

	fmt.Printf("  %s%s%s ", common.Purple7,
		common.T("SOVA gateway: ", "Шлюз SOVA: "), common.Reset)
	reader := bufio.NewReader(os.Stdin)
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return
	}

	if addr == "clear" || addr == "none" || addr == "off" || addr == "0" {
		cfg.UpstreamProxy = ""
		if err := cfg.Save(common.GetConfigPath()); err != nil {
			ui.PrintError(err)
			return
		}
		ui.PrintSuccess(common.T("Upstream gateway removed — direct mode", "Шлюз убран — прямой режим"))
		waitEnter()
		return
	}

	// Validate the proxy URL
	_, err := common.CreateUpstreamDialer(addr)
	if err != nil {
		ui.PrintError(fmt.Errorf(common.T("Invalid gateway: %v", "Ошибка шлюза: %v"), err))
		waitEnter()
		return
	}

	cfg.UpstreamProxy = addr
	if err := cfg.Save(common.GetConfigPath()); err != nil {
		ui.PrintError(err)
		waitEnter()
		return
	}
	ui.PrintSuccess(fmt.Sprintf(common.T(
		"Upstream gateway set: %s — tunnel will route ALL traffic through it",
		"Шлюз установлен: %s — туннель направит ВЕСЬ трафик через него",
	), addr))
	waitEnter()
}

func menuSystemProxy(ui *common.UI) {
	cfg, _ := common.LoadConfig(common.GetConfigPath())
	listenAddr := cfg.ListenAddress()

	if common.IsSystemProxySet() {
		// Прокси включен — предлагаем выключить
		fmt.Printf("\n  %s%s%s\n", common.Gold+common.Bold,
			common.T("System proxy is ON — disabling...", "Системный прокси ВКЛ — отключаем..."), common.Reset)
		if err := common.ClearSystemProxy(); err != nil {
			ui.PrintError(fmt.Errorf(common.T("Failed to clear proxy: %v", "Ошибка сброса прокси: %v"), err))
		} else {
			ui.PrintSuccess(common.T("System proxy disabled", "Системный прокси отключён"))
		}
	} else {
		// Прокси выключен — включаем
		fmt.Printf("\n  %s%s%s\n", common.Cyan+common.Bold,
			common.T("Enabling system proxy → "+listenAddr, "Включаем системный прокси → "+listenAddr), common.Reset)
		if err := common.SetSystemProxy(listenAddr); err != nil {
			ui.PrintError(fmt.Errorf(common.T("Failed to set proxy: %v", "Ошибка установки прокси: %v"), err))
		} else {
			ui.PrintSuccess(common.T(
				"System proxy ON — ALL traffic routed through SOVA ("+listenAddr+")",
				"Системный прокси ВКЛ — ВЕСЬ трафик через SOVA ("+listenAddr+")",
			))
		}
	}
	waitEnter()
}

func menuConfig(ui *common.UI) {
	for {
		items := []common.MenuItem{
			{LabelEN: "Show Configuration", LabelRU: "Показать конфигурацию", DescEN: "Current settings", DescRU: "Текущие настройки"},
			{LabelEN: "Edit Setting", LabelRU: "Изменить параметр", DescEN: "key = value", DescRU: "ключ = значение"},
			{LabelEN: "Reset to Defaults", LabelRU: "Сбросить настройки", DescEN: "Restore defaults", DescRU: "Восстановить по умолчанию"},
			{LabelEN: "Export JSON", LabelRU: "Экспорт JSON", DescEN: "Config as JSON", DescRU: "Конфиг в JSON"},
			{LabelEN: "Config Path", LabelRU: "Путь к конфигу", DescEN: "File location", DescRU: "Расположение файла"},
			{LabelEN: "Back", LabelRU: "Назад", DescEN: "", DescRU: ""},
		}

		choice := common.RunMenu("Configuration", "Конфигурация", items)
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
				ui.PrintSuccess(common.T("Config reset to defaults", "Конфигурация сброшена"))
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

	fmt.Printf("\n  %s%s%s", common.Purple7, common.T("Key: ", "Ключ: "), common.Reset)
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	fmt.Printf("  %s%s%s", common.Purple7, common.T("Value: ", "Значение: "), common.Reset)
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	cfg, _ := common.LoadConfig(common.GetConfigPath())
	applied := applyConfigSetting(cfg, key, value)
	if !applied {
		ui.PrintError(fmt.Errorf(common.T("Unknown key: %s", "Неизвестный ключ: %s"), key))
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
	case "upstream_proxy":
		cfg.UpstreamProxy = value
	case "tls_profile":
		cfg.TLSProfile = value
	case "protocol_version":
		v, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		cfg.ProtocolVersion = v
	case "default_outbound":
		cfg.Routing.DefaultOutbound = value
	case "dns_mode":
		cfg.DNSEngine.Mode = value
	case "dns_listen":
		cfg.DNSEngine.ListenAddr = value
	case "dns_leak_protect":
		cfg.DNSEngine.LeakProtect = value == "true" || value == "1" || value == "on"
	case "domain_front_sni":
		cfg.DomainFronting.FrontSNI = value
		cfg.DomainFronting.Enabled = value != ""
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
				DescRU:  "Enter для переключения",
			})
		}
		items = append(items, common.MenuItem{LabelEN: "Back", LabelRU: "Назад"})

		choice := common.RunMenu("Modules", "Модули", items)
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
	fmt.Printf("\n  %s%s%s", common.Dim, common.T("Press Enter to continue...", "Нажмите Enter..."), common.Reset)
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// startTunnel — главная функция: запуск SOVA прокси
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

	// Запуск SOVA прокси — главный туннель
	listenAddr := cfg.ListenAddress()
	ui.PrintStatus(fmt.Sprintf("Starting SOVA proxy on %s...", listenAddr), common.Green)

	// Protocol v2 info
	if cfg.ProtocolVersion == 2 {
		ui.PrintSuccess("SOVA Protocol v2: X25519+Kyber1024 hybrid KEM, HKDF, ChaCha20-Poly1305")
	}

	proxy := common.NewSOVAProxy(listenAddr, ui)

	// Upstream gateway chaining — маршрутизация трафика через другой шлюз SOVA/HTTP
	if cfg.UpstreamProxy != "" {
		ui.PrintStatus(fmt.Sprintf("Chaining through upstream gateway %s...", cfg.UpstreamProxy), common.Cyan)
		dialer, err := common.CreateUpstreamDialer(cfg.UpstreamProxy)
		if err != nil {
			ui.PrintError(fmt.Errorf("upstream gateway failed: %v", err))
			ui.PrintWarning("Falling back to direct connections (foreign sites may not work)")
		} else {
			proxy.RemoteDialer = dialer
			ui.PrintSuccess(fmt.Sprintf("Traffic chained through %s — IP changed!", cfg.UpstreamProxy))
		}
	} else {
		ui.PrintWarning(common.T(
			"No upstream gateway configured — traffic exits from YOUR IP. Set upstream_proxy for multi-hop mode.",
			"Шлюз не настроен — трафик идёт с ВАШЕГО IP. Настройте upstream_proxy для multi-hop режима.",
		))
	}

	if err := proxy.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("SOVA proxy failed: %v", err))
	}

	// Авто-настройка системного прокси
	autoProxy := cfg.Features.AutoProxy
	if autoProxy {
		ui.PrintStatus("Configuring system proxy...", common.Cyan)
		if err := common.SetSystemProxy(listenAddr); err != nil {
			ui.PrintWarning(fmt.Sprintf("Auto-proxy failed: %v", err))
			autoProxy = false
		} else {
			ui.PrintSuccess("System proxy configured — ALL traffic routed through SOVA")
		}
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

			// Восстановить системный прокси
			if autoProxy {
				ui.PrintStatus("Restoring system proxy settings...", common.Yellow)
				if err := common.ClearSystemProxy(); err != nil {
					ui.PrintWarning(fmt.Sprintf("Failed to restore proxy: %v", err))
				} else {
					ui.PrintSuccess("System proxy restored")
				}
			}

			proxy.Stop()
			if cancel != nil {
				cancel()
			}
			stats := proxy.GetStats()
			ui.PrintSection("Session Summary")
			ui.PrintKeyValue("Total connections:", fmt.Sprintf("%d", stats["total_connections"]))
			ui.PrintKeyValue("Traffic uploaded:", formatBytes(stats["bytes_up"]))
			ui.PrintKeyValue("Traffic downloaded:", formatBytes(stats["bytes_down"]))
			fmt.Println()
			ui.PrintSuccess("SOVA stopped. Stay free!")
			return

		case <-ticker.C:
			stats := proxy.GetStats()
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

	// SOVA прокси направляет трафик через удалённый сервер
	listenAddr := cfg.ListenAddress()
	ui.PrintStatus(fmt.Sprintf("Starting SOVA proxy on %s (via %s)...", listenAddr, serverAddr), common.Green)

	proxy := common.NewSOVAProxy(listenAddr, ui)

	// SOVA protocol dialer: TLS + DPI evasion + encrypted frames
	psk := cfg.PSK
	if psk == "" {
		psk = common.DefaultPSK
	}
	dpiCfg := common.DPIConfigFromConfig(cfg)

	ui.PrintStatus("SOVA Protocol: TLS + AES-256-GCM + DPI evasion", common.Cyan)
	if dpiCfg.FragmentClientHello {
		ui.PrintSuccess(fmt.Sprintf("DPI bypass: ClientHello fragmentation (%d bytes), SNI spoofing, jitter %dms",
			dpiCfg.FragmentSize, dpiCfg.FragmentJitterMs))
	}

	proxy.RemoteDialer = common.CreateSOVARemoteDialer(serverAddr, psk, dpiCfg)

	if err := proxy.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("SOVA proxy failed: %v", err))
	}

	// Авто-настройка системного прокси
	autoProxy := cfg.Features.AutoProxy
	if autoProxy {
		ui.PrintStatus("Configuring system proxy...", common.Cyan)
		if err := common.SetSystemProxy(listenAddr); err != nil {
			ui.PrintWarning(fmt.Sprintf("Auto-proxy failed: %v", err))
			autoProxy = false
		} else {
			ui.PrintSuccess("System proxy configured — ALL traffic via SOVA server")
		}
	}

	ui.PrintSection("🦉 SOVA Remote Tunnel Active")
	ui.PrintKeyValue("Server:", common.Gold+common.Bold+serverAddr+common.Reset)
	ui.PrintKeyValue("Local proxy:", listenAddr)
	protoVer := "v1 (TLS + AES-256-GCM)"
	if cfg.ProtocolVersion == 2 {
		protoVer = "v2 (X25519+Kyber1024 + HKDF + ChaCha20)"
	}
	ui.PrintKeyValue("Protocol:", "SOVA "+protoVer)
	ui.PrintKeyValue("DPI Evasion:", "ClientHello frag + SNI spoof + jitter")
	ui.PrintKeyValue("Transport:", "TLS 1.3 (looks like HTTPS to ISP)")
	if autoProxy {
		ui.PrintKeyValue("System proxy:", common.Gold+common.Bold+"ON — all traffic routed"+common.Reset)
	}
	fmt.Println()
	fmt.Printf("%s  Press Ctrl+C to stop%s\n", common.Dim, common.Reset)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	if autoProxy {
		ui.PrintStatus("Restoring system proxy...", common.Yellow)
		common.ClearSystemProxy()
		ui.PrintSuccess("System proxy restored")
	}
	proxy.Stop()
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
		case "upstream_proxy":
			cfg.UpstreamProxy = value
		case "tls_profile":
			cfg.TLSProfile = value
		case "protocol_version":
			v, err := strconv.Atoi(value)
			if err != nil {
				ui.ExitWithError(fmt.Errorf("invalid version: %s", value))
			}
			cfg.ProtocolVersion = v
		case "default_outbound":
			cfg.Routing.DefaultOutbound = value
		case "dns_mode":
			cfg.DNSEngine.Mode = value
		case "domain_front_sni":
			cfg.DomainFronting.FrontSNI = value
			cfg.DomainFronting.Enabled = value != ""
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
	common.StartManagementAPI(cfg, ui)
}

// startCore — запуск SOVA Core engine (новый движок v2)
func startCore(ui *common.UI) {
	ui.PrintBanner()

	cfg, err := common.LoadConfig(common.GetConfigPath())
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Config error: %v, using defaults", err))
		cfg = common.DefaultConfig()
	}

	// Убеждаемся что Protocol v2
	cfg.ProtocolVersion = 2
	ui.PrintSuccess("SOVA Core v2 — Protocol v2 enabled")

	// Инициализация криптографии
	ui.PrintStatus("Initializing SOVA Core cryptography...", common.Cyan)
	if err := common.InitMasterKey(); err != nil {
		ui.ExitWithError(fmt.Errorf("master key init failed: %v", err))
	}
	if cfg.Encryption.PQEnabled {
		if err := common.InitPQKeys(); err != nil {
			ui.PrintWarning(fmt.Sprintf("PQ crypto init: %v (continuing without PQ)", err))
		} else {
			ui.PrintSuccess("Post-quantum crypto: Kyber1024 KEM + Dilithium5 signatures")
		}
	}
	ui.PrintSuccess("Hybrid KEM: X25519 + Kyber1024 | AEAD: ChaCha20-Poly1305 | HKDF-SHA256")

	// Stealth
	if cfg.Stealth.Enabled {
		ui.PrintStatus("Activating stealth engine...", common.Cyan)
		ui.PrintSuccess(fmt.Sprintf("Stealth: profile=%s, jitter=%dms", cfg.Stealth.Profile, cfg.Stealth.JitterMs))
	}

	// DNS Engine
	if cfg.DNSEngine.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting DNS engine (%s) on %s...", cfg.DNSEngine.Mode, cfg.DNSEngine.ListenAddr), common.Cyan)
		dnsEngine := common.NewDNSEngine2(cfg.DNSEngine)
		if err := dnsEngine.Start(); err != nil {
			ui.PrintWarning(fmt.Sprintf("DNS engine: %v", err))
		} else {
			ui.PrintSuccess(fmt.Sprintf("DNS engine: %s mode on %s (leak protect: %v)", cfg.DNSEngine.Mode, cfg.DNSEngine.ListenAddr, cfg.DNSEngine.LeakProtect))
		}
	}

	// API
	if cfg.API.Enabled {
		ui.PrintStatus(fmt.Sprintf("Starting management API on %s:%d...", cfg.API.Host, cfg.API.Port), common.Cyan)
		go startClientAPI(cfg, ui)
		ui.PrintSuccess(fmt.Sprintf("API: http://%s:%d/api/", cfg.API.Host, cfg.API.Port))
	}

	// SOVA Core Engine
	ui.PrintStatus("Starting SOVA Core engine...", common.Green)

	core := common.NewSOVACore(cfg, ui)
	if err := core.Start(); err != nil {
		ui.ExitWithError(fmt.Errorf("SOVA Core failed: %v", err))
	}

	listenAddr := cfg.ListenAddress()

	// TLS Profile
	ui.PrintSuccess(fmt.Sprintf("TLS fingerprint: %s", common.GetUTLSProfileName(common.TLSProfile(cfg.TLSProfile))))

	// Domain Fronting
	if cfg.DomainFronting.Enabled {
		ui.PrintSuccess(fmt.Sprintf("Domain fronting: SNI=%s", cfg.DomainFronting.FrontSNI))
	}

	// Auto proxy
	autoProxy := cfg.Features.AutoProxy
	if autoProxy {
		ui.PrintStatus("Configuring system proxy...", common.Cyan)
		if err := common.SetSystemProxy(listenAddr); err != nil {
			ui.PrintWarning(fmt.Sprintf("Auto-proxy failed: %v", err))
			autoProxy = false
		} else {
			ui.PrintSuccess("System proxy configured — ALL traffic routed through SOVA Core")
		}
	}

	ui.AnimateConnection()

	// Информация
	ui.PrintSection("🦉 SOVA Core Active")
	ui.PrintKeyValue("Core version:", "v"+common.Version)
	ui.PrintKeyValue("Protocol:", "SOVA v2 (X25519+Kyber1024 + HKDF + ChaCha20)")
	ui.PrintKeyValue("Local proxy:", listenAddr)
	ui.PrintKeyValue("Routing:", cfg.Routing.DefaultOutbound)
	ui.PrintKeyValue("TLS profile:", cfg.TLSProfile)
	if cfg.DomainFronting.Enabled {
		ui.PrintKeyValue("Domain fronting:", cfg.DomainFronting.FrontSNI)
	}
	if cfg.DNSEngine.Enabled {
		ui.PrintKeyValue("DNS engine:", cfg.DNSEngine.Mode+" on "+cfg.DNSEngine.ListenAddr)
	}
	if autoProxy {
		ui.PrintKeyValue("System proxy:", common.Gold+common.Bold+"ON — all traffic routed"+common.Reset)
	}
	fmt.Println()
	fmt.Printf("%s  Press Ctrl+C to stop%s\n", common.Dim, common.Reset)

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			ui.PrintStatus("Shutting down SOVA Core...", common.Yellow)
			if autoProxy {
				common.ClearSystemProxy()
				ui.PrintSuccess("System proxy restored")
			}
			core.Stop()
			stats := core.GetStats()
			ui.PrintSection("Session Summary")
			ui.PrintKeyValue("Total connections:", fmt.Sprintf("%d", stats["total_connections"]))
			ui.PrintKeyValue("Traffic up:", formatBytes(stats["bytes_up"]))
			ui.PrintKeyValue("Traffic down:", formatBytes(stats["bytes_down"]))
			fmt.Println()
			ui.PrintSuccess("SOVA Core stopped. Stay free!")
			return
		case <-ticker.C:
			stats := core.GetStats()
			if stats["active_connections"] > 0 || stats["total_connections"] > 0 {
				ui.PrintStatus(fmt.Sprintf("Core: Active %d | Total %d | ↑%s ↓%s",
					stats["active_connections"],
					stats["total_connections"],
					formatBytes(stats["bytes_up"]),
					formatBytes(stats["bytes_down"])), common.Dim+common.Purple)
			}
		}
	}
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
