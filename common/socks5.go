package common

import (
	"fmt"
	"net"
	"time"
)

// CreateRemoteDialer создаёт dialer, который маршрутизирует трафик через удалённый SOVA сервер
func CreateRemoteDialer(serverAddr string) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		// Подключаемся к удалённому SOVA серверу
		serverConn, err := net.DialTimeout("tcp", serverAddr, 15*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SOVA server %s: %v", serverAddr, err)
		}

		// Отправляем целевой адрес серверу в формате: длина(1 байт) + адрес
		addrBytes := []byte(addr)
		if len(addrBytes) > 255 {
			serverConn.Close()
			return nil, fmt.Errorf("target address too long")
		}

		header := make([]byte, 1+len(addrBytes))
		header[0] = byte(len(addrBytes))
		copy(header[1:], addrBytes)

		if _, err := serverConn.Write(header); err != nil {
			serverConn.Close()
			return nil, fmt.Errorf("failed to send target address: %v", err)
		}

		// Читаем ответ (1 байт: 0 = успех, 1 = ошибка)
		resp := make([]byte, 1)
		serverConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if _, err := serverConn.Read(resp); err != nil {
			serverConn.Close()
			return nil, fmt.Errorf("no response from server: %v", err)
		}
		serverConn.SetReadDeadline(time.Time{})

		if resp[0] != 0 {
			serverConn.Close()
			return nil, fmt.Errorf("server refused connection to %s", addr)
		}

		return serverConn, nil
	}
}
