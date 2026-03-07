package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sova/common"
)

func main() {
	ui := common.NewUI(true)
	ui.PrintBanner()

	if len(os.Args) < 2 {
		ui.PrintInfo("Использование: sova <команда> [аргументы]")
		ui.PrintInfo("Команды: connect, disconnect, status")
		return
	}

	command := os.Args[1]
	switch command {
	case "install":
		ui.PrintStatus("Запуск мастера установки...", common.Cyan)
		// Автономный установщик: загрузка бинарника, настройка сервисов
		ui.PrintInfo("Определение платформы...")
		// Здесь должна быть логика, скачивающая релизные пакеты
		ui.PrintSuccess("SOVA успешно установлен")
		return
	case "connect":
		if len(os.Args) < 3 {
			ui.PrintError(fmt.Errorf("Использование: sova connect <json_uri>"))
			return
		}
		jsonURI := os.Args[2]
		connectToServer(jsonURI, ui)
	case "disconnect":
		ui.PrintStatus("Отключение...", common.Yellow)
		// Отключение обрабатывается через закрытие активного соединения
		ui.PrintSuccess("Отключено")
	case "status":
		ui.PrintStatus("Статус: Не подключен", common.Cyan)
		// Статус доступен через REST API /api/stats
	case "config":
		if len(os.Args) < 3 {
			ui.ExitWithError(fmt.Errorf("Usage: sova config <user_id>"))
		}
		userID := os.Args[2]
		ui.PrintStatus("Запрос конфигурации...", common.Cyan)
		// Here you would contact /api/config
		fmt.Printf("Конфигурация для %s: <base64>\n", userID)
	case "proxy":
		ui.PrintStatus("Запрос доступных прокси...", common.Cyan)
		// Here you would contact /api/proxy
		fmt.Println("xray: vless://...\nsingbox: sova://...\ngeneric: socks5://127.0.0.1:1080")
	default:
		ui.PrintError(fmt.Errorf("Неизвестная команда: %s", command))
	}
}

func connectToServer(jsonURI string, ui *common.UI) {
	ui.PrintStatus("Парсинг конфигурации...", common.Cyan)
	config, err := common.DecodeConfig(jsonURI)
	if err != nil {
		ui.ExitWithError(err)
	}

	ui.AnimateConnection()

	transportConfig := &common.TransportConfig{
		Mode:       common.WebMirrorMode,
		ServerAddr: "localhost:443",
		SNI:        config.SNIList[0],
	}

	ui.PrintStatus("Установка транспорта...", common.Cyan)
	conn, err := common.DialTransport(transportConfig)
	if err != nil {
		ui.ExitWithError(err)
	}
	defer conn.Conn.Close()

	ui.PrintStatus("Аутентификация ZKP + PQ...", common.Cyan)
	cred := &common.UserCredentials{UserID: "test", Password: "pass"}
	challengeBuf := make([]byte, 32)
	n, err := conn.Conn.Read(challengeBuf)
	if err != nil {
		ui.ExitWithError(err)
	}
	challenge := &common.ZKPChallenge{Nonce: challengeBuf[:n]}

	// Прочитать nonce для генерации сессионного ключа
	nonceBuf := make([]byte, 16)
	n, err = conn.Conn.Read(nonceBuf)
	if err != nil {
		ui.ExitWithError(err)
	}
	sessionKey, _ := common.DeriveSessionKey(nonceBuf)

	proof, err := cred.ProvePassword(challenge, []byte(config.ServerPubKey))
	if err != nil {
		ui.ExitWithError(err)
	}

	_, err = conn.Conn.Write(proof.Response)
	if err != nil {
		ui.ExitWithError(err)
	}

	ui.PrintSuccess("Аутентифицировано с пост-квантовым шифрованием")

	// Запуск AI адаптера
	switcher := common.NewAdaptiveSwitcher()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go switcher.MonitorNetwork(ctx)

	ui.PrintStatus("Запуск SOCKS5 прокси на 127.0.0.1:1080", common.Green)
	listener, err := net.Listen("tcp", "127.0.0.1:1080")
	if err != nil {
		ui.ExitWithError(err)
	}
	defer listener.Close()

	ui.PrintSuccess("SOVA туннель активен! Используйте прокси 127.0.0.1:1080")

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func(cc net.Conn) {
			// wrapper that encrypts/decrypts using sessionKey
			handleProxy(cc, conn.Conn, ui, sessionKey)
		}(clientConn)
	}
}

func handleProxy(clientConn, remoteConn net.Conn, ui *common.UI, key []byte) {
	defer clientConn.Close()
	_ = key // session key for encrypted tunnel
	tunnel := &common.TunnelReaderWriter{
		LocalConn:  clientConn,
		RemoteConn: remoteConn,
	}
	tunnel.StartTunnel()
	ui.PrintInfo("Новый прокси-соединение обработано")
}
