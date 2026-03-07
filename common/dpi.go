package common

// ── DPI Evasion Engine ─────────────────────────────────────────────────
//
// Техники обхода Deep Packet Inspection (DPI) мобильных операторов:
//
// 1. TLS ClientHello фрагментация — разбиваем первый TCP-пакет (ClientHello)
//    на фрагменты по 1–3 байта. DPI системы не могут прочитать SNI.
//
// 2. SNI spoofing — в TLS ClientHello указываем легитимный домен
//    (google.com, cloudflare.com), а не реальный адрес сервера.
//
// 3. Timing jitter — случайные задержки между фрагментами,
//    нарушающие паттерн-анализ.
//
// 4. Random padding — каждый фрейм содержит случайное количество
//    мусорных байт для обмана traffic analysis.
//
// 5. Self-signed TLS — генерация сертификата на лету для standalone режима.
//
// ────────────────────────────────────────────────────────────────────────

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// ── DPI Config ──────────────────────────────────────────────────────────

// DPIConfig настройки обхода DPI
type DPIConfig struct {
	Enabled             bool
	FragmentClientHello bool     // разбить TLS ClientHello на мелкие TCP-сегменты
	FragmentSize        int      // размер каждого фрагмента (1–5 байт)
	FragmentJitterMs    int      // jitter между фрагментами (мс)
	SNIList             []string // домены для подделки SNI
	PaddingEnabled      bool     // random padding в фреймах
}

// DefaultDPIConfig возвращает настройки DPI обхода по умолчанию
func DefaultDPIConfig() *DPIConfig {
	return &DPIConfig{
		Enabled:             true,
		FragmentClientHello: true,
		FragmentSize:        2,
		FragmentJitterMs:    25,
		SNIList: []string{
			"www.google.com",
			"cdn.cloudflare.com",
			"ajax.googleapis.com",
			"fonts.googleapis.com",
			"www.youtube.com",
			"play.google.com",
			"www.gstatic.com",
			"update.googleapis.com",
			"clients1.google.com",
			"www.googleapis.com",
		},
		PaddingEnabled: true,
	}
}

// DPIConfigFromConfig создаёт DPIConfig из основной конфигурации SOVA
func DPIConfigFromConfig(cfg *Config) *DPIConfig {
	dpi := DefaultDPIConfig()
	dpi.Enabled = cfg.Stealth.Enabled
	dpi.FragmentJitterMs = cfg.Stealth.JitterMs
	dpi.PaddingEnabled = cfg.Stealth.PaddingEnabled
	if len(cfg.Transport.SNIList) > 0 {
		dpi.SNIList = cfg.Transport.SNIList
	}
	return dpi
}

// ── FragConn: TCP-фрагментатор ─────────────────────────────────────────
// Разбивает первую запись (TLS ClientHello) на мелкие TCP-сегменты.
// После первой записи данные идут нормально.
// Это ломает DPI, который пытается прочитать SNI из цельного ClientHello.

type FragConn struct {
	net.Conn
	fragSize int
	fragDone bool
	jitterMs int
}

// NewFragConn оборачивает соединение в фрагментатор
func NewFragConn(conn net.Conn, fragSize, jitterMs int) *FragConn {
	if fragSize < 1 {
		fragSize = 2
	}
	return &FragConn{
		Conn:     conn,
		fragSize: fragSize,
		jitterMs: jitterMs,
	}
}

func (fc *FragConn) Write(p []byte) (int, error) {
	// Фрагментируем только первую запись (TLS ClientHello)
	if fc.fragDone {
		return fc.Conn.Write(p)
	}
	fc.fragDone = true

	total := 0
	for len(p) > 0 {
		sz := fc.fragSize
		if sz > len(p) {
			sz = len(p)
		}
		n, err := fc.Conn.Write(p[:sz])
		total += n
		if err != nil {
			return total, err
		}
		p = p[sz:]

		// Случайная задержка между фрагментами
		if len(p) > 0 && fc.jitterMs > 0 {
			jitter := 1 + int(secureRandByte())%fc.jitterMs
			time.Sleep(time.Duration(jitter) * time.Millisecond)
		}
	}
	return total, nil
}

// ── SOVA Dialer ─────────────────────────────────────────────────────────

// DialSOVAServer подключается к SOVA серверу с полным стеком протокола:
//  1. TCP connect
//  2. TLS с поддельным SNI + фрагментация ClientHello (обход DPI)
//  3. SOVA handshake (вывод сессионного ключа)
//  4. Возвращает SOVAConn, готовый к зашифрованной фреймовой коммуникации
func DialSOVAServer(serverAddr, psk string, dpiCfg *DPIConfig) (*SOVAConn, error) {
	if dpiCfg == nil {
		dpiCfg = DefaultDPIConfig()
	}

	// 1. TCP connect
	rawConn, err := net.DialTimeout("tcp", serverAddr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("sova: TCP connect to %s failed: %v", serverAddr, err)
	}

	// 2. Оборачиваем в фрагментатор (DPI evasion)
	var baseConn net.Conn = rawConn
	if dpiCfg.Enabled && dpiCfg.FragmentClientHello {
		baseConn = NewFragConn(rawConn, dpiCfg.FragmentSize, dpiCfg.FragmentJitterMs)
	}

	// 3. TLS с поддельным SNI
	sni := "www.google.com"
	if len(dpiCfg.SNIList) > 0 {
		sni = dpiCfg.SNIList[int(secureRandByte())%len(dpiCfg.SNIList)]
	}

	tlsConn := tls.Client(baseConn, &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true, // Аутентификация через SOVA handshake, не через CA
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	})
	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("sova: TLS handshake failed (SNI=%s): %v", sni, err)
	}

	// 4. SOVA protocol handshake (вывод общего сессионного ключа AES-256-GCM)
	sovaConn, err := ClientHandshake(tlsConn, psk)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("sova: protocol handshake failed: %v", err)
	}

	return sovaConn, nil
}

// CreateSOVARemoteDialer создаёт dialer, который маршрутизирует трафик
// через SOVA сервер с полным стеком протокола (TLS + DPI evasion + encryption).
// Заменяет старый CreateRemoteDialer.
func CreateSOVARemoteDialer(serverAddr, psk string, dpiCfg *DPIConfig) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		// Подключаемся к SOVA серверу с DPI evasion + TLS + SOVA protocol
		sovaConn, err := DialSOVAServer(serverAddr, psk, dpiCfg)
		if err != nil {
			return nil, err
		}

		// Отправляем CONNECT фрейм с адресом назначения
		if err := sovaConn.WriteFrame(&Frame{
			Type:    FrameConnect,
			Payload: []byte(addr),
		}); err != nil {
			sovaConn.Close()
			return nil, fmt.Errorf("sova: CONNECT write failed: %v", err)
		}

		// Читаем ACK от сервера
		ackFrame, err := sovaConn.ReadFrame()
		if err != nil {
			sovaConn.Close()
			return nil, fmt.Errorf("sova: no ACK from server: %v", err)
		}
		if ackFrame.Type != FrameAck || len(ackFrame.Payload) == 0 || ackFrame.Payload[0] != 0x00 {
			sovaConn.Close()
			return nil, fmt.Errorf("sova: server refused connection to %s", addr)
		}

		// Возвращаем SOVAStream как net.Conn — прозрачный byte-stream
		// поверх зашифрованных фреймов
		return NewSOVAStream(sovaConn), nil
	}
}

// ── Self-signed TLS Certificate (для standalone SOVA сервера) ──────────

// GenerateSelfSignedTLSConfig генерирует TLS конфиг с самоподписанным сертификатом.
// Используется когда SOVA сервер работает без nginx.
func GenerateSelfSignedTLSConfig() (*tls.Config, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("sova tls: key generation: %v", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"Cloudflare Inc"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{
			"www.google.com",
			"cdn.cloudflare.com",
			"ajax.googleapis.com",
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("sova tls: cert creation: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("sova tls: key marshal: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("sova tls: key pair: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// ── TLS Listener (для SOVA сервера) ────────────────────────────────────

// NewTLSListener создаёт TLS listener для SOVA сервера
func NewTLSListener(addr string) (net.Listener, error) {
	tlsCfg, err := GenerateSelfSignedTLSConfig()
	if err != nil {
		return nil, err
	}

	listener, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("sova: TLS listen on %s failed: %v", addr, err)
	}
	return listener, nil
}
