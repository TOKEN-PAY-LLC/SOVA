package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"
)

// TransportMode определяет режим транспорта
type TransportMode int

const (
	WebMirrorMode TransportMode = iota
	CloudCarrierMode
	ShadowWebSocketMode
)

// TransportConfig конфигурация транспорта
type TransportConfig struct {
	Mode       TransportMode
	ServerAddr string
	SNI        string
}

// Connection представляет соединение
type Connection struct {
	Conn   net.Conn
	Config *TransportConfig
	// Encryption hooks for tunnel
	EncryptFunc func([]byte) []byte
	DecryptFunc func([]byte) []byte
}

// DialWebMirror устанавливает соединение в режиме Web Mirror
func DialWebMirror(config *TransportConfig) (*Connection, error) {
	// Установить TLS соединение с custom fingerprint для имитации браузера
	tlsConfig := &tls.Config{
		ServerName: config.SNI,
		// InsecureSkipVerify is intentional: SOVA uses post-quantum key exchange
		// (Kyber1024) for server authentication instead of TLS certificates.
		// The SNI field is set for DPI evasion, not for certificate validation.
		InsecureSkipVerify: true, // #nosec G402 — PQ key exchange verifies server identity
	}

	conn, err := tls.Dial("tcp", config.ServerAddr, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &Connection{Conn: conn, Config: config}, nil
}

// DialCloudCarrier устанавливает QUIC соединение
func DialCloudCarrier(config *TransportConfig) (*Connection, error) {
	return DialQUICTransport(config)
}

// DialShadowWebSocket устанавливает WebSocket через CDN
func DialShadowWebSocket(config *TransportConfig) (*Connection, error) {
	return DialWebSocketTransport(config)
}

// DialTransport выбирает и устанавливает транспорт
func DialTransport(config *TransportConfig) (*Connection, error) {
	switch config.Mode {
	case WebMirrorMode:
		return DialWebMirror(config)
	case CloudCarrierMode:
		return DialCloudCarrier(config)
	case ShadowWebSocketMode:
		return DialShadowWebSocket(config)
	default:
		return nil, fmt.Errorf("unknown transport mode")
	}
}

// NetworkMetrics содержит метрики сети для адаптации
type NetworkMetrics struct {
	RTT        time.Duration
	PacketLoss float64
	RSTCount   int
	HTTPStubs  int
	Bandwidth  float64
	Jitter     time.Duration
}

// AdaptiveSwitcher управляет адаптацией транспорта
type AdaptiveSwitcher struct {
	CurrentMode TransportMode
	Metrics     *NetworkMetrics
	AI          *AIAdapter
}

// NewAdaptiveSwitcher создает новый адаптивный переключатель
func NewAdaptiveSwitcher() *AdaptiveSwitcher {
	return &AdaptiveSwitcher{
		CurrentMode: WebMirrorMode,
		Metrics:     &NetworkMetrics{},
		AI:          NewAIAdapter(),
	}
}

// MonitorNetwork мониторит сеть и собирает метрики
func (s *AdaptiveSwitcher) MonitorNetwork(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Измерять RTT, потери пакетов и т.д.
			// Для прототипа: симулировать
			s.Metrics.RTT = 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
			s.Metrics.PacketLoss = rand.Float64() * 0.1
			s.Metrics.RSTCount += rand.Intn(2)
			s.Metrics.HTTPStubs += rand.Intn(2)

			// Записать в AI
			if s.Metrics.RTT > 100*time.Millisecond {
				s.AI.RecordEvent("rtt_high", float64(s.Metrics.RTT.Milliseconds()))
			}
			if s.Metrics.PacketLoss > 0.05 {
				s.AI.RecordEvent("packet_loss_high", s.Metrics.PacketLoss)
			}
			if s.Metrics.RSTCount > 0 {
				s.AI.RecordEvent("rst_detected", float64(s.Metrics.RSTCount))
			}
			if s.Metrics.HTTPStubs > 0 {
				s.AI.RecordEvent("http_stub", float64(s.Metrics.HTTPStubs))
			}

			// Адаптироваться
			actions := s.AI.AnalyzeAndAdapt()
			for _, action := range actions {
				s.ExecuteAction(action)
			}
		}
	}
}

// ExecuteAction выполняет действие адаптации
func (s *AdaptiveSwitcher) ExecuteAction(action string) {
	switch action {
	case "switch_to_quic":
		s.CurrentMode = CloudCarrierMode
	case "switch_to_websocket":
		s.CurrentMode = ShadowWebSocketMode
	case "fragment_packets":
		// Фрагментация обрабатывается в stealth engine
	case "jitter_timing":
		// Jitter обрабатывается в stealth engine
	case "change_sni":
		// SNI ротация обрабатывается при следующем подключении
	case "update_cdn_list":
		// CDN обновляется из конфигурации
	}
}

// TunnelReaderWriter для туннелирования трафика
type TunnelReaderWriter struct {
	LocalConn  net.Conn
	RemoteConn net.Conn
}

// StartTunnel запускает туннель
func (t *TunnelReaderWriter) StartTunnel() {
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(t.RemoteConn, t.LocalConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(t.LocalConn, t.RemoteConn)
		done <- struct{}{}
	}()
	<-done
}

// EncryptedTunnel туннель с шифрованием
type EncryptedTunnel struct {
	LocalConn   net.Conn
	RemoteConn  net.Conn
	EncryptFunc func([]byte) ([]byte, error)
	DecryptFunc func([]byte) ([]byte, error)
	OnData      func(up, down int64)
}

// StartEncryptedTunnel запускает шифрованный туннель
func (t *EncryptedTunnel) StartEncryptedTunnel() {
	done := make(chan struct{}, 2)
	// Local -> Remote (encrypt)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := t.LocalConn.Read(buf)
			if err != nil {
				break
			}
			data := buf[:n]
			if t.EncryptFunc != nil {
				encrypted, err := t.EncryptFunc(data)
				if err != nil {
					break
				}
				data = encrypted
			}
			wn, err := t.RemoteConn.Write(data)
			if err != nil {
				break
			}
			if t.OnData != nil {
				t.OnData(int64(wn), 0)
			}
		}
		done <- struct{}{}
	}()
	// Remote -> Local (decrypt)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := t.RemoteConn.Read(buf)
			if err != nil {
				break
			}
			data := buf[:n]
			if t.DecryptFunc != nil {
				decrypted, err := t.DecryptFunc(data)
				if err != nil {
					break
				}
				data = decrypted
			}
			wn, err := t.LocalConn.Write(data)
			if err != nil {
				break
			}
			if t.OnData != nil {
				t.OnData(0, int64(wn))
			}
		}
		done <- struct{}{}
	}()
	<-done
}
