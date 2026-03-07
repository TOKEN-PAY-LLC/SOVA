package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"sova/common"
	"time"
)

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SOVA Protocol"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func main() {
	ui := common.NewUI(true)
	ui.PrintBanner()

	// CLI helper
	if len(os.Args) > 1 && os.Args[1] == "install" {
		ui.PrintStatus("Установка SOVA сервера...", common.Cyan)
		// В реальности скачиваем/настраиваем, генерируем ключи
		srvKeys, _ := common.GenerateServerKeys()
		fmt.Printf("Серверные ключи сгенерированы: %x\n", srvKeys.PublicKey)
		fmt.Println("JSON-ссылка: https://sova.io/link/" + "dummy123")
		return
	}

	ui.PrintStatus("Генерация ключей сервера...", common.Cyan)
	serverKeys, err := common.GenerateServerKeys()
	if err != nil {
		ui.ExitWithError(err)
	}
	if err := common.InitMasterKey(); err != nil {
		ui.ExitWithError(err)
	}
	ui.PrintStatus("Инициализация пост-квантовых ключей...", common.Cyan)
	if err := common.InitPQKeys(); err != nil {
		ui.ExitWithError(err)
	}
	ui.PrintSuccess(fmt.Sprintf("Публичный ключ сервера: %x", serverKeys.PublicKey))

	// Инициализация middleware
	rateLimiter := NewRateLimiter(100) // 100 requests per minute
	logger := NewLogger(1000)          // Keep last 1000 logs
	connMonitor := NewConnectionMonitor()

	// Инициализация API с middleware
	api := NewServerAPI(serverKeys, rateLimiter, logger, connMonitor)
	api.StartAPI(8080) // API на порту 8080

	ui.PrintStatus("Генерация сертификата...", common.Cyan)
	cert, err := generateSelfSignedCert()
	if err != nil {
		ui.ExitWithError(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		// TODO: Добавить custom handshake detection
	}

	ui.PrintStatus("Запуск AI адаптера...", common.Cyan)
	switcher := common.NewAdaptiveSwitcher()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go switcher.MonitorNetwork(ctx)

	ui.PrintStatus("Запуск сервера на :443", common.Green)
	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		ui.ExitWithError(err)
	}
	defer listener.Close()

	ui.PrintSuccess("SOVA сервер запущен и готов к подключениям")

	for {
		conn, err := listener.Accept()
		if err != nil {
			ui.PrintError(err)
			continue
		}

		go sovaHandler(conn, ui, api)
	}
}

func sovaHandler(conn net.Conn, ui *common.UI, api *ServerAPI) {
	defer conn.Close()
	clientIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	connID := fmt.Sprintf("%s_%d", clientIP, time.Now().Unix())

	ui.PrintInfo("Новое SOVA соединение от " + clientIP)

	// Генерация challenge для ZKP
	challenge, _ := common.GenerateChallenge()
	conn.Write(challenge.Nonce)

	// Новая сессионная симметричная ключ-шем
	nonce := make([]byte, 16)
	io.ReadFull(rand.Reader, nonce)
	sessionKey, _ := common.DeriveSessionKey(nonce)
	// отправить nonce клиенту для синхронизации
	conn.Write(nonce)

	// Получить proof
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		ui.PrintError(err)
		api.Logger.Log("ERROR", "Failed to read proof", clientIP, "")
		return
	}
	proof := &common.ZKPProof{Response: buf[:n]}

	// Проверить (упрощенная логика)
	cred := &common.UserCredentials{UserID: "test", Password: "pass"}
	if err := common.VerifyProof(proof, challenge, cred.UserID, api.serverKeys.PublicKey); err != nil {
		ui.PrintError(fmt.Errorf("Аутентификация не удалась"))
		api.Logger.Log("WARN", "Authentication failed", clientIP, cred.UserID)
		return
	}
	ui.PrintSuccess("Аутентифицировано")
	api.Logger.Log("INFO", "User authenticated", clientIP, cred.UserID)

	// Начать сессию
	session, err := api.StartSession(cred.UserID)
	if err != nil {
		ui.PrintError(err)
		api.Logger.Log("ERROR", "Failed to start session", clientIP, cred.UserID)
		return
	}

	// Добавить соединение в монитор
	api.ConnMonitor.AddConnection(connID, clientIP, cred.UserID)

	// Установить туннель с симметричным шифрованием и статистикой
	tunnel := &common.TunnelReaderWriter{
		LocalConn:  conn, // В реальности локальный прокси
		RemoteConn: conn, // Эхо для прототипа
		OnData: func(up, down int64) {
			api.ConnMonitor.UpdateConnection(connID, up, down)
		},
		EncryptFunc: func(data []byte) []byte {
			ciphertext, _ := common.EncryptData(sessionKey, data)
			return ciphertext
		},
		DecryptFunc: func(data []byte) []byte {
			plaintext, _ := common.DecryptData(sessionKey, data)
			return plaintext
		},
	}
	tunnel.StartTunnel()

	// Завершить сессию
	api.ConnMonitor.RemoveConnection(connID)
	api.UpdateStats(fmt.Sprintf("%s_%d", session.UserID, session.StartTime.Unix()), 0, 0)
	api.Logger.Log("INFO", "Connection closed", clientIP, cred.UserID)
}