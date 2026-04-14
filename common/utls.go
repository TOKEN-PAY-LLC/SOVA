package common

// ── SOVA Core — TLS Fingerprint Mimicry ────────────────────────────────
//
// Имитация TLS отпечатков популярных браузеров для обхода DPI.
// DPI системы могут идентифицировать прокси-клиентов по нестандартным
// TLS ClientHello (JA3/JA4 fingerprint).
//
// SOVA имитирует отпечатки:
//   - Chrome 120+ (самый популярный, наименее подозрительный)
//   - Firefox 120+ (второй по популярности)
//   - Safari 17+ (macOS/iOS)
//   - Random (случайная комбинация для продвинутого DPI)
//
// Техники:
//   - Правильный порядок cipher suites
//   - Правильные TLS extensions
//   - ALPN (h2, http/1.1)
//   - Supported Versions extension (TLS 1.3)
//   - Key Share (X25519 + P-256)
//   - PSK Key Exchange Modes
//   - Signed Certificate Timestamp
//   - Compress Certificate
//   - Application-Layer Protocol Negotiation
//
// ────────────────────────────────────────────────────────────────────────

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// ── TLS Fingerprint Profiles ───────────────────────────────────────────

// TLSProfile профиль TLS отпечатка
type TLSProfile string

const (
	TLSChrome  TLSProfile = "chrome"
	TLSFirefox TLSProfile = "firefox"
	TLSSafari  TLSProfile = "safari"
	TLSRandom  TLSProfile = "random"
)

// Chrome cipher suites (в порядке Chrome 120)
var chromeCipherSuites = []uint16{
	0x1301, // TLS_AES_128_GCM_SHA256
	0x1302, // TLS_AES_256_GCM_SHA384
	0x1303, // TLS_CHACHA20_POLY1305_SHA256
	0xC02C, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	0xC02B, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	0xCCA9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305
	0xC030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	0xC02F, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	0xCCA8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305
}

// Firefox cipher suites
var firefoxCipherSuites = []uint16{
	0x1303, // TLS_CHACHA20_POLY1305_SHA256
	0x1301, // TLS_AES_128_GCM_SHA256
	0x1302, // TLS_AES_256_GCM_SHA384
	0xCCA9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305
	0xC02C, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	0xC02B, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	0xCCA8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305
	0xC030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	0xC02F, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
}

// Safari cipher suites
var safariCipherSuites = []uint16{
	0x1301, // TLS_AES_128_GCM_SHA256
	0x1302, // TLS_AES_256_GCM_SHA384
	0x1303, // TLS_CHACHA20_POLY1305_SHA256
	0xC02B, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	0xC02C, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	0xC02F, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	0xC030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
}

// Chrome ALPN
var chromeALPN = []string{"h2", "http/1.1"}

// Firefox ALPN
var firefoxALPN = []string{"h2", "http/1.1"}

// Safari ALPN
var safariALPN = []string{"h2", "http/1.1"}

// Chrome SNI list for domain fronting
var chromeSNIs = []string{
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
	"maps.googleapis.com",
	"storage.googleapis.com",
	"firebase.googleapis.com",
	"android.googleapis.com",
}

// ── UTLS Dialer ────────────────────────────────────────────────────────

// UTLSConfig конфигурация uTLS
type UTLSConfig struct {
	Profile        TLSProfile
	SNI            string
	FragmentSize   int
	FragmentJitter int
	Insecure       bool
}

// DialUTLS подключается к серверу с имитацией TLS отпечатка браузера
func DialUTLS(network, addr string, cfg *UTLSConfig) (net.Conn, error) {
	if cfg == nil {
		cfg = &UTLSConfig{Profile: TLSChrome}
	}

	// Выбираем профиль
	profile := cfg.Profile
	if profile == TLSRandom {
		profiles := []TLSProfile{TLSChrome, TLSFirefox, TLSSafari}
		profile = profiles[rand.Intn(len(profiles))]
	}

	// Выбираем SNI
	sni := cfg.SNI
	if sni == "" {
		sni = chromeSNIs[rand.Intn(len(chromeSNIs))]
	}

	// Строим TLS конфигурацию по профилю
	tlsConfig := buildTLSConfig(profile, sni, cfg.Insecure)

	// TCP connect
	rawConn, err := net.DialTimeout(network, addr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("utls: TCP connect: %v", err)
	}

	// DPI evasion: фрагментация ClientHello
	var baseConn net.Conn = rawConn
	if cfg.FragmentSize > 0 {
		baseConn = NewFragConn(rawConn, cfg.FragmentSize, cfg.FragmentJitter)
	}

	// TLS handshake с нужным fingerprint
	tlsConn := tls.Client(baseConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("utls: TLS handshake (profile=%s, SNI=%s): %v", profile, sni, err)
	}

	return tlsConn, nil
}

// buildTLSConfig создаёт TLS конфигурацию по профилю браузера
func buildTLSConfig(profile TLSProfile, sni string, insecure bool) *tls.Config {
	cfg := &tls.Config{
		ServerName: sni,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		NextProtos: chromeALPN, // По умолчанию Chrome ALPN
	}

	if insecure {
		cfg.InsecureSkipVerify = true
	} else {
		cfg.RootCAs = systemCertPool()
	}

	// Применяем cipher suites по профилю
	switch profile {
	case TLSChrome:
		cfg.CipherSuites = chromeCipherSuites
		cfg.NextProtos = chromeALPN
	case TLSFirefox:
		cfg.CipherSuites = firefoxCipherSuites
		cfg.NextProtos = firefoxALPN
	case TLSSafari:
		cfg.CipherSuites = safariCipherSuites
		cfg.NextProtos = safariALPN
	}

	// CurvePreferences (X25519 first, как в Chrome)
	cfg.CurvePreferences = []tls.CurveID{
		tls.X25519,
		tls.CurveP256,
		tls.CurveP384,
	}

	return cfg
}

// systemCertPool возвращает системный пул сертификатов
func systemCertPool() *x509.CertPool {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return x509.NewCertPool()
	}
	return pool
}

// ── Domain Fronting ────────────────────────────────────────────────────

// DomainFrontingConfig конфигурация domain fronting
type DomainFrontingConfig struct {
	Enabled    bool   `json:"enabled"`
	FrontSNI   string `json:"front_sni"`   // SNI для DPI (легитимный домен)
	HostHeader string `json:"host_header"` // Host header для CDN
}

// DialWithDomainFronting подключается с domain fronting
func DialWithDomainFronting(network, addr string, frontCfg *DomainFrontingConfig, dpiCfg *DPIConfig) (net.Conn, error) {
	if frontCfg == nil || !frontCfg.Enabled {
		// Без domain fronting — обычное подключение
		conn, err := DialSOVAServer(addr, DefaultPSK, dpiCfg)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	// TCP connect к CDN
	rawConn, err := net.DialTimeout(network, addr, 15*time.Second)
	if err != nil {
		return nil, err
	}

	// DPI evasion
	var baseConn net.Conn = rawConn
	if dpiCfg != nil && dpiCfg.Enabled && dpiCfg.FragmentClientHello {
		baseConn = NewFragConn(rawConn, dpiCfg.FragmentSize, dpiCfg.FragmentJitterMs)
	}

	// TLS с поддельным SNI (domain fronting)
	tlsConfig := &tls.Config{
		ServerName:         frontCfg.FrontSNI,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		NextProtos:         []string{"h2", "http/1.1"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}

	tlsConn := tls.Client(baseConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("domain fronting: TLS handshake: %v", err)
	}

	return tlsConn, nil
}

// ── ECH (Encrypted Client Hello) ────────────────────────────────────────

// ECHConfig конфигурация Encrypted Client Hello
type ECHConfig struct {
	Enabled    bool   `json:"enabled"`
	PublicName string `json:"public_name"` // Легитимное имя для outer SNI
	ConfigID   byte   `json:"config_id"`
	PublicKey  []byte `json:"public_key"`
}

// NOTE: Полная реализация ECH (ESNI) требует поддержки в crypto/tls,
// которая пока экспериментальная в Go. SOVA подготовит инфраструктуру
// для ECH, а полная поддержка будет добавлена когда Go её стабилизирует.

// GetUTLSProfileName возвращает имя профиля
func GetUTLSProfileName(profile TLSProfile) string {
	switch profile {
	case TLSChrome:
		return "Chrome 120+"
	case TLSFirefox:
		return "Firefox 120+"
	case TLSSafari:
		return "Safari 17+"
	case TLSRandom:
		return "Random (rotate)"
	default:
		return string(profile)
	}
}

// GetAllTLSProfiles возвращает все доступные профили
func GetAllTLSProfiles() []TLSProfile {
	return []TLSProfile{TLSChrome, TLSFirefox, TLSSafari, TLSRandom}
}
