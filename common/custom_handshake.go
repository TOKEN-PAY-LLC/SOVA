package common

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// CustomTLSConfig представляет custom TLS config для SOVA
type CustomTLSConfig struct {
	*tls.Config
	SovaExtensions []uint16 // Custom extensions для отличия SOVA
}

// IsSovaHandshake проверяет, является ли handshake SOVA
func IsSovaHandshake(conn net.Conn) bool {
	// Читаем ClientHello (упрощенная проверка)
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}

	// Проверяем на наличие custom extension (например, 0xFFFF)
	// Это упрощенная логика; в реальности парсить TLS handshake
	for i := 0; i < n-2; i++ {
		if buf[i] == 0xFF && buf[i+1] == 0xFF {
			return true
		}
	}
	return false
}

// DialWithCustomHandshake подключается с custom ClientHello
func DialWithCustomHandshake(addr string, config *CustomTLSConfig) (*tls.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Создаем custom ClientHello с дополнительными extensions
	// Для прототипа используем стандартный, но добавляем флаг
	tlsConn := tls.Client(conn, config.Config)
	// TODO: Переопределить handshake для добавления custom extensions

	err = tlsConn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// FragmentClientHello фрагментирует ClientHello для обхода DPI
func FragmentClientHello(conn net.Conn, hello []byte) error {
	// Разбиваем на фрагменты
	fragmentSize := 100
	for i := 0; i < len(hello); i += fragmentSize {
		end := i + fragmentSize
		if end > len(hello) {
			end = len(hello)
		}
		_, err := conn.Write(hello[i:end])
		if err != nil {
			return err
		}
		// Добавляем jitter
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
	return nil
}

// PacketMorphing применяет морфинг пакетов
func PacketMorphing(data []byte) []byte {
	// Добавляем случайный padding
	padding := make([]byte, rand.Intn(50))
	for i := range padding {
		padding[i] = byte(rand.Intn(256))
	}
	return append(data, padding...)
}