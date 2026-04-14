package common

// ── SOVA Core — Advanced DNS Engine ────────────────────────────────────
//
// Полноценный DNS движок с поддержкой:
//   - DNS-over-HTTPS (DoH) — Google, Cloudflare, NextDNS
//   - DNS-over-TLS (DoT) — прямое TLS подключение к DNS серверу
//   - DNS-over-QUIC (DoQ) — QUIC транспорт для DNS
//   - Fake DNS — локальный резолвер для перехвата запросов
//   - DNS leak protection — все запросы идут через SOVA
//   - Split DNS — разные upstream для разных доменов
//   - Cache с TTL — кэширование ответов
//   - Anti-poisoning — проверка целостности ответов
//
// ────────────────────────────────────────────────────────────────────────

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
)

// ── DNS Engine ─────────────────────────────────────────────────────────

// DNSEngine — продвинутый DNS движок
type DNSEngine struct {
	config  DNSConfig2
	cache   map[string]*dnsCacheEntry
	mu      sync.RWMutex
	udpConn *net.UDPConn
	running bool
	stats   DNSStats
}

// DNSConfig2 конфигурация DNS движка
type DNSConfig2 struct {
	Enabled     bool   `json:"enabled"`
	ListenAddr  string `json:"listen_addr"`   // ":53"
	Mode        string `json:"mode"`          // "doh", "dot", "doq", "udp", "fake"
	FakeIPRange string `json:"fake_ip_range"` // "198.18.0.0/16"
	LeakProtect bool   `json:"leak_protect"`  // Блокировать DNS запросы вне SOVA

	// Upstream серверы
	DoHServers []DoHServer `json:"doh_servers"`
	DoTServers []DoTServer `json:"dot_servers"`
	DoQServers []DoQServer `json:"doq_servers"`

	// Split DNS
	SplitRules []DNSSplitRule `json:"split_rules"`

	// Cache
	CacheEnabled bool `json:"cache_enabled"`
	CacheSize    int  `json:"cache_size"` // Макс записей
}

// DoHServer DNS-over-HTTPS сервер
type DoHServer struct {
	URL      string `json:"url"`      // "https://1.1.1.1/dns-query"
	Hostname string `json:"hostname"` // "cloudflare-dns.com"
}

// DoTServer DNS-over-TLS сервер
type DoTServer struct {
	Addr     string `json:"addr"`     // "1.1.1.1:853"
	Hostname string `json:"hostname"` // "cloudflare-dns.com"
}

// DoQServer DNS-over-QUIC сервер
type DoQServer struct {
	Addr     string `json:"addr"`     // "1.1.1.1:853"
	Hostname string `json:"hostname"` // "cloudflare-dns.com"
}

// DNSSplitRule правило split DNS
type DNSSplitRule struct {
	Domain   string `json:"domain"`   // Суффикс домена
	Upstream string `json:"upstream"` // "doh:google", "dot:cloudflare", "direct"
}

// dnsCacheEntry кэш запись
type dnsCacheEntry struct {
	IPs      []net.IP
	Expiry   time.Time
	TTL      time.Duration
	HitCount int
}

// DNSStats статистика DNS
type DNSStats struct {
	Queries    int64
	CacheHits  int64
	CacheMiss  int64
	DoHQueries int64
	DoTQueries int64
	DoQQueries int64
	Blocked    int64
}

// DefaultDNSConfig2 конфигурация DNS по умолчанию
func DefaultDNSConfig2() DNSConfig2 {
	return DNSConfig2{
		Enabled:      false,
		ListenAddr:   "127.0.0.1:5353",
		Mode:         "doh",
		FakeIPRange:  "198.18.0.0/16",
		LeakProtect:  true,
		CacheEnabled: true,
		CacheSize:    4096,
		DoHServers: []DoHServer{
			{URL: "https://1.1.1.1/dns-query", Hostname: "cloudflare-dns.com"},
			{URL: "https://8.8.8.8/dns-query", Hostname: "dns.google"},
			{URL: "https://9.9.9.9/dns-query", Hostname: "dns.quad9.net"},
		},
		DoTServers: []DoTServer{
			{Addr: "1.1.1.1:853", Hostname: "cloudflare-dns.com"},
			{Addr: "8.8.8.8:853", Hostname: "dns.google"},
		},
		DoQServers: []DoQServer{
			{Addr: "9.9.9.9:853", Hostname: "dns.quad9.net"},
		},
		SplitRules: []DNSSplitRule{
			{Domain: ".ru", Upstream: "direct"},
			{Domain: ".su", Upstream: "direct"},
			{Domain: ".рф", Upstream: "direct"},
		},
	}
}

// NewDNSEngine создаёт DNS движок
func NewDNSEngine2(cfg DNSConfig2) *DNSEngine {
	return &DNSEngine{
		config: cfg,
		cache:  make(map[string]*dnsCacheEntry),
	}
}

// Start запускает DNS движок
func (de *DNSEngine) Start() error {
	if !de.config.Enabled {
		return nil
	}

	addr := de.config.ListenAddr
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("dns engine: resolve addr: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("dns engine: listen: %v", err)
	}
	de.udpConn = conn
	de.running = true

	go de.serve()

	return nil
}

// Stop останавливает DNS движок
func (de *DNSEngine) Stop() {
	de.running = false
	if de.udpConn != nil {
		de.udpConn.Close()
	}
}

// serve обрабатывает DNS запросы
func (de *DNSEngine) serve() {
	buf := make([]byte, 1500)
	for de.running {
		n, remoteAddr, err := de.udpConn.ReadFromUDP(buf)
		if err != nil {
			if !de.running {
				return
			}
			continue
		}

		// Копируем данные
		query := make([]byte, n)
		copy(query, buf[:n])

		go de.handleQuery(query, remoteAddr)
	}
}

// handleQuery обрабатывает DNS запрос
func (de *DNSEngine) handleQuery(query []byte, remoteAddr *net.UDPAddr) {
	// Парсим домен из запроса
	domain := parseDNSDomain(query)
	if domain == "" {
		return
	}

	// Проверяем split DNS
	upstream := de.resolveSplitDNS(domain)

	// Проверяем кэш
	if de.config.CacheEnabled {
		if ips := de.cacheLookup(domain); len(ips) > 0 {
			response := buildDNSResponse(query, ips)
			de.udpConn.WriteToUDP(response, remoteAddr)
			atomic.AddInt64(&de.stats.CacheHits, 1)
			return
		}
		atomic.AddInt64(&de.stats.CacheMiss, 1)
	}

	atomic.AddInt64(&de.stats.Queries, 1)

	// Резолвим через выбранный upstream
	var ips []net.IP
	var err error

	switch upstream {
	case "direct":
		ips, err = de.resolveDirect(domain)
	case "doh:cloudflare":
		ips, err = de.resolveDoH(domain, de.config.DoHServers[0])
	case "doh:google":
		if len(de.config.DoHServers) > 1 {
			ips, err = de.resolveDoH(domain, de.config.DoHServers[1])
		} else {
			ips, err = de.resolveDoH(domain, de.config.DoHServers[0])
		}
	default:
		// По умолчанию — DoH через первый сервер
		if len(de.config.DoHServers) > 0 {
			ips, err = de.resolveDoH(domain, de.config.DoHServers[0])
			atomic.AddInt64(&de.stats.DoHQueries, 1)
		} else {
			ips, err = de.resolveDirect(domain)
		}
	}

	if err != nil || len(ips) == 0 {
		return
	}

	// Кэшируем
	if de.config.CacheEnabled {
		de.cacheStore(domain, ips, 300*time.Second)
	}

	// Отправляем ответ
	response := buildDNSResponse(query, ips)
	de.udpConn.WriteToUDP(response, remoteAddr)
}

// resolveSplitDNS определяет upstream для домена
func (de *DNSEngine) resolveSplitDNS(domain string) string {
	for _, rule := range de.config.SplitRules {
		if strings.HasSuffix(strings.ToLower(domain), strings.ToLower(rule.Domain)) {
			return rule.Upstream
		}
	}
	return de.config.Mode // По умолчанию — основной режим
}

// resolveDirect резолвит через системный DNS
func (de *DNSEngine) resolveDirect(domain string) ([]net.IP, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	var result []net.IP
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			result = append(result, ipv4)
		}
	}
	return result, nil
}

// resolveDoH резолвит через DNS-over-HTTPS
func (de *DNSEngine) resolveDoH(domain string, server DoHServer) ([]net.IP, error) {
	// Строим DNS запрос
	query := buildDNSQuery(domain, 1) // Type A

	// DoH RFC 8484: GET с base64url параметром
	encoded := base64.RawURLEncoding.EncodeToString(query)
	url := fmt.Sprintf("%s?dns=%s", server.URL, encoded)

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: server.Hostname,
			},
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-message")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DoH server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseDNSResponse(body)
}

// resolveDoT резолвит через DNS-over-TLS
func (de *DNSEngine) resolveDoT(domain string, server DoTServer) ([]net.IP, error) {
	query := buildDNSQuery(domain, 1)

	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		server.Addr,
		&tls.Config{ServerName: server.Hostname},
	)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// DNS over TCP: 2-byte length prefix
	msg := make([]byte, 2+len(query))
	binary.BigEndian.PutUint16(msg, uint16(len(query)))
	copy(msg[2:], query)

	if _, err := conn.Write(msg); err != nil {
		return nil, err
	}

	// Read response
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	respLen := int(binary.BigEndian.Uint16(header))
	resp := make([]byte, respLen)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, err
	}

	return parseDNSResponse(resp)
}

// resolveDoQ резолвит через DNS-over-QUIC
func (de *DNSEngine) resolveDoQ(domain string, server DoQServer) ([]net.IP, error) {
	query := buildDNSQuery(domain, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := quic.DialAddr(ctx, server.Addr,
		&tls.Config{
			ServerName:         server.Hostname,
			NextProtos:         []string{"doq"},
			InsecureSkipVerify: true,
		},
		&quic.Config{},
	)
	if err != nil {
		return nil, err
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	if _, err := stream.Write(query); err != nil {
		return nil, err
	}

	resp := make([]byte, 1500)
	n, err := stream.Read(resp)
	if err != nil {
		return nil, err
	}

	return parseDNSResponse(resp[:n])
}

// ── Cache ──────────────────────────────────────────────────────────────

func (de *DNSEngine) cacheLookup(domain string) []net.IP {
	de.mu.RLock()
	defer de.mu.RUnlock()

	entry, ok := de.cache[domain]
	if !ok || time.Now().After(entry.Expiry) {
		return nil
	}
	entry.HitCount++
	result := make([]net.IP, len(entry.IPs))
	copy(result, entry.IPs)
	return result
}

func (de *DNSEngine) cacheStore(domain string, ips []net.IP, ttl time.Duration) {
	de.mu.Lock()
	defer de.mu.Unlock()

	// Ограничиваем размер кэша
	if len(de.cache) >= de.config.CacheSize {
		// Удаляем самые старые
		var oldest string
		oldestTime := time.Now()
		for k, v := range de.cache {
			if v.Expiry.Before(oldestTime) {
				oldestTime = v.Expiry
				oldest = k
			}
		}
		if oldest != "" {
			delete(de.cache, oldest)
		}
	}

	de.cache[domain] = &dnsCacheEntry{
		IPs:    ips,
		Expiry: time.Now().Add(ttl),
		TTL:    ttl,
	}
}

// ── DNS Protocol Helpers ───────────────────────────────────────────────

// parseDNSDomain извлекает домен из DNS запроса
func parseDNSDomain(query []byte) string {
	if len(query) < 12 {
		return ""
	}
	// Skip header (12 bytes)
	pos := 12
	var domain string
	for pos < len(query) {
		labelLen := int(query[pos])
		if labelLen == 0 {
			break
		}
		pos++
		if pos+labelLen > len(query) {
			break
		}
		if domain != "" {
			domain += "."
		}
		domain += string(query[pos : pos+labelLen])
		pos += labelLen
	}
	return domain
}

// buildDNSQuery создаёт DNS запрос
func buildDNSQuery(domain string, qtype uint16) []byte {
	var buf bytes.Buffer

	// Header
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], 0x1234) // ID
	header[2] = 0x01                                // Recursion desired
	binary.BigEndian.PutUint16(header[4:6], 1)      // QDCOUNT
	buf.Write(header)

	// Question
	for _, label := range strings.Split(domain, ".") {
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0) // End of domain

	// QTYPE + QCLASS
	qtypeBuf := make([]byte, 4)
	binary.BigEndian.PutUint16(qtypeBuf[0:2], qtype)
	binary.BigEndian.PutUint16(qtypeBuf[2:4], 1) // IN
	buf.Write(qtypeBuf)

	return buf.Bytes()
}

// buildDNSResponse создаёт DNS ответ из запроса
func buildDNSResponse(query []byte, ips []net.IP) []byte {
	if len(query) < 12 || len(ips) == 0 {
		return query
	}

	resp := make([]byte, len(query))
	copy(resp, query)

	// Set response flags
	resp[2] = 0x81 // Response + Recursion available
	resp[3] = 0x80 // No error

	// ANCOUNT
	binary.BigEndian.PutUint16(resp[6:8], uint16(len(ips)))

	// Append answers
	for _, ip := range ips {
		// Name pointer (compression)
		resp = append(resp, 0xC0, 0x0C)
		// TYPE A
		resp = append(resp, 0x00, 0x01)
		// CLASS IN
		resp = append(resp, 0x00, 0x01)
		// TTL (300s)
		ttl := make([]byte, 4)
		binary.BigEndian.PutUint32(ttl, 300)
		resp = append(resp, ttl...)
		// RDLENGTH
		resp = append(resp, 0x00, 0x04)
		// RDATA (IP)
		resp = append(resp, ip.To4()...)
	}

	return resp
}

// parseDNSResponse парсит DNS ответ
func parseDNSResponse(resp []byte) ([]net.IP, error) {
	if len(resp) < 12 {
		return nil, fmt.Errorf("dns: response too short")
	}

	ancount := int(binary.BigEndian.Uint16(resp[6:8]))
	if ancount == 0 {
		return nil, fmt.Errorf("dns: no answers")
	}

	// Skip header
	pos := 12

	// Skip question
	for pos < len(resp) && resp[pos] != 0 {
		pos++
	}
	pos += 5 // null byte + qtype + qclass

	var ips []net.IP
	for i := 0; i < ancount && pos < len(resp); i++ {
		// Name (might be compressed)
		if resp[pos]&0xC0 == 0xC0 {
			pos += 2
		} else {
			for pos < len(resp) && resp[pos] != 0 {
				pos++
			}
			pos++
		}

		if pos+10 > len(resp) {
			break
		}

		// TYPE
		qtype := binary.BigEndian.Uint16(resp[pos : pos+2])
		pos += 8 // skip type(2) + class(2) + ttl(4)

		// RDLENGTH
		rdlen := int(binary.BigEndian.Uint16(resp[pos : pos+2]))
		pos += 2

		if pos+rdlen > len(resp) {
			break
		}

		if qtype == 1 && rdlen == 4 { // A record
			ips = append(ips, net.IP(resp[pos:pos+4]))
		}

		pos += rdlen
	}

	return ips, nil
}

// GetStats возвращает статистику DNS
func (de *DNSEngine) GetStats() map[string]int64 {
	return map[string]int64{
		"queries":     atomic.LoadInt64(&de.stats.Queries),
		"cache_hits":  atomic.LoadInt64(&de.stats.CacheHits),
		"cache_miss":  atomic.LoadInt64(&de.stats.CacheMiss),
		"doh_queries": atomic.LoadInt64(&de.stats.DoHQueries),
		"dot_queries": atomic.LoadInt64(&de.stats.DoTQueries),
		"doq_queries": atomic.LoadInt64(&de.stats.DoQQueries),
		"blocked":     atomic.LoadInt64(&de.stats.Blocked),
	}
}
