package common

// ── SOVA Core — Autonomous Proxy Engine ────────────────────────────────
//
// SOVA Core — собственное ядро прокси-сервера, аналог xray/sing-box,
// но полностью автономное и использующее только SOVA Protocol.
//
// Архитектура:
//
//   Inbound (HTTP CONNECT, TUN) → Router → Outbound (SOVA Direct, SOVA Server, Block)
//
// Inbound — приём трафика от приложений:
//   - HTTP CONNECT Proxy (основной, для браузеров/системного прокси)
//   - TUN Device (для перехвата всего трафика на уровне ОС)
//
// Outbound — отправка трафика к цели:
//   - direct — прямое подключение (для локального/белого трафика)
//   - sova — через SOVA сервер (TLS + PQ handshake + encrypted frames)
//   - block — блокировка (для рекламы/трекеров)
//   - http — через HTTP CONNECT прокси
//
// Router — маршрутизация по правилам:
//   - domain — по домену (точное совпадение, суффикс, regex)
//   - ip — по IP/CIDR
//   - geo — по стране (GeoIP)
//   - process — по имени процесса (на Windows)
//   - default — маршрут по умолчанию
//
// ────────────────────────────────────────────────────────────────────────

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ── Core Engine ────────────────────────────────────────────────────────

// SOVACore — главное ядро SOVA
type SOVACore struct {
	config    *Config
	router    *Router
	inbound   InboundHandler
	outbounds map[string]OutboundHandler
	ui        *UI

	// Stats
	activeConns    int64
	totalConns     int64
	totalBytesUp   int64
	totalBytesDown int64

	// Lifecycle
	mu      sync.RWMutex
	running bool
	cancel  context.CancelFunc
}

// NewSOVACore создаёт новое ядро SOVA
func NewSOVACore(cfg *Config, ui *UI) *SOVACore {
	core := &SOVACore{
		config:    cfg,
		outbounds: make(map[string]OutboundHandler),
		ui:        ui,
	}

	// Инициализируем outbound обработчики
	core.setupOutbounds()

	// Инициализируем роутер
	core.router = NewRouter(cfg, core.outbounds)

	return core
}

// setupOutbounds создаёт outbound обработчики
func (c *SOVACore) setupOutbounds() {
	// Direct — прямое подключение
	c.outbounds["direct"] = &DirectOutbound{}

	// Block — блокировка
	c.outbounds["block"] = &BlockOutbound{}

	// SOVA — подключение через SOVA сервер
	if c.config.ServerAddr != "" || c.config.UpstreamProxy != "" {
		c.outbounds["sova"] = NewSOVAOutbound(c.config)
	}

	// HTTP — через HTTP CONNECT прокси
	if c.config.UpstreamProxy != "" && strings.HasPrefix(c.config.UpstreamProxy, "http") {
		dialer, _ := CreateUpstreamDialer(c.config.UpstreamProxy)
		if dialer != nil {
			c.outbounds["http"] = &HTTPOutbound{dialer: dialer}
		}
	}
}

// Start запускает ядро SOVA
func (c *SOVACore) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	// Инициализация криптографии
	if err := InitMasterKey(); err != nil {
		return fmt.Errorf("SOVA Core: master key init: %v", err)
	}
	if c.config.Encryption.PQEnabled {
		if err := InitPQKeys(); err != nil {
			if c.ui != nil {
				c.ui.PrintWarning(fmt.Sprintf("PQ crypto init: %v", err))
			}
		}
	}

	// Создаём inbound обработчик
	listenAddr := c.config.ListenAddress()
	c.inbound = NewHTTPConnectInbound(listenAddr, c.router, c.ui)

	// Запускаем inbound
	if err := c.inbound.Start(); err != nil {
		return fmt.Errorf("SOVA Core: inbound start: %v", err)
	}

	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())

	// Запускаем keepalive для SOVA outbound
	if sova, ok := c.outbounds["sova"]; ok {
		go sova.(*SOVAOutbound).keepAlive(ctx)
	}

	c.running = true

	if c.ui != nil {
		c.ui.PrintSuccess(fmt.Sprintf("SOVA Core v%s started on %s", Version, listenAddr))
	}

	return nil
}

// Stop останавливает ядро
func (c *SOVACore) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.running = false
	if c.cancel != nil {
		c.cancel()
	}
	if c.inbound != nil {
		c.inbound.Stop()
	}
}

// GetStats возвращает статистику
func (c *SOVACore) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&c.activeConns),
		"total_connections":  atomic.LoadInt64(&c.totalConns),
		"bytes_up":           atomic.LoadInt64(&c.totalBytesUp),
		"bytes_down":         atomic.LoadInt64(&c.totalBytesDown),
	}
}

// RouteConnection маршрутизирует соединение через роутер
func (c *SOVACore) RouteConnection(targetAddr string) (net.Conn, error) {
	outbound := c.router.Resolve(targetAddr)
	if outbound == nil {
		outbound = c.outbounds["direct"]
	}

	conn, err := outbound.Dial("tcp", targetAddr)
	if err != nil {
		return nil, err
	}

	atomic.AddInt64(&c.totalConns, 1)
	atomic.AddInt64(&c.activeConns, 1)

	return &trackedConn{
		Conn:   conn,
		core:   c,
		onClose: func() {
			atomic.AddInt64(&c.activeConns, -1)
		},
	}, nil
}

// trackedConn — соединение с отслеживанием статистики
type trackedConn struct {
	net.Conn
	core    *SOVACore
	onClose func()
}

func (tc *trackedConn) Read(b []byte) (int, error) {
	n, err := tc.Conn.Read(b)
	if n > 0 {
		atomic.AddInt64(&tc.core.totalBytesDown, int64(n))
	}
	return n, err
}

func (tc *trackedConn) Write(b []byte) (int, error) {
	n, err := tc.Conn.Write(b)
	if n > 0 {
		atomic.AddInt64(&tc.core.totalBytesUp, int64(n))
	}
	return n, err
}

func (tc *trackedConn) Close() error {
	if tc.onClose != nil {
		tc.onClose()
	}
	return tc.Conn.Close()
}

// ── Inbound Handler Interface ─────────────────────────────────────────

// InboundHandler принимает подключения от приложений
type InboundHandler interface {
	Start() error
	Stop()
}

// HTTPConnectInbound — HTTP CONNECT прокси (основной inbound)
type HTTPConnectInbound struct {
	listenAddr string
	router     *Router
	ui         *UI
	listener   net.Listener
	running    bool
	mu         sync.RWMutex
}

// NewHTTPConnectInbound создаёт HTTP CONNECT inbound
func NewHTTPConnectInbound(listenAddr string, router *Router, ui *UI) *HTTPConnectInbound {
	return &HTTPConnectInbound{
		listenAddr: listenAddr,
		router:     router,
		ui:         ui,
	}
}

// Start запускает HTTP CONNECT прокси
func (h *HTTPConnectInbound) Start() error {
	listener, err := net.Listen("tcp", h.listenAddr)
	if err != nil {
		return fmt.Errorf("SOVA Core inbound listen: %v", err)
	}
	h.listener = listener
	h.mu.Lock()
	h.running = true
	h.mu.Unlock()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				h.mu.RLock()
				running := h.running
				h.mu.RUnlock()
				if !running {
					return
				}
				continue
			}
			go h.handleConnection(conn)
		}
	}()

	return nil
}

// Stop останавливает inbound
func (h *HTTPConnectInbound) Stop() {
	h.mu.Lock()
	h.running = false
	h.mu.Unlock()
	if h.listener != nil {
		h.listener.Close()
	}
}

// handleConnection обрабатывает входящее подключение
func (h *HTTPConnectInbound) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Peek первый байт
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}

	// SOCKS5 — отклоняем (SOVA не использует SOCKS5)
	if buf[0] == 0x05 {
		return
	}

	// HTTP запрос
	reader := bufio.NewReader(io.MultiReader(
		strings.NewReader(string(buf[0])),
		conn,
	))

	req, err := http.ReadRequest(reader)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	if req.Method == "CONNECT" {
		h.handleConnect(conn, req.Host)
	} else {
		h.handlePlainHTTP(conn, req, reader)
	}
}

// handleConnect обрабатывает HTTP CONNECT
func (h *HTTPConnectInbound) handleConnect(conn net.Conn, targetAddr string) {
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":443"
	}
	conn.SetDeadline(time.Time{})

	// Маршрутизация через роутер
	outbound := h.router.Resolve(targetAddr)
	if outbound == nil {
		outbound = &DirectOutbound{}
	}

	remote, err := outbound.Dial("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional relay
	relay(conn, remote)
}

// handlePlainHTTP обрабатывает обычный HTTP запрос
func (h *HTTPConnectInbound) handlePlainHTTP(conn net.Conn, req *http.Request, reader *bufio.Reader) {
	targetAddr := req.Host
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":80"
	}
	conn.SetDeadline(time.Time{})

	outbound := h.router.Resolve(targetAddr)
	if outbound == nil {
		outbound = &DirectOutbound{}
	}

	remote, err := outbound.Dial("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	req.Write(remote)
	relay(conn, remote)
}

// relay — двунаправленная пересылка
func relay(local, remote net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(remote, local)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(local, remote)
		done <- struct{}{}
	}()

	<-done
}

// ── Outbound Handler Interface ─────────────────────────────────────────

// OutboundHandler отправляет трафик к цели
type OutboundHandler interface {
	Dial(network, addr string) (net.Conn, error)
}

// DirectOutbound — прямое подключение
type DirectOutbound struct{}

func (d *DirectOutbound) Dial(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, 10*time.Second)
}

// BlockOutbound — блокировка соединения
type BlockOutbound struct{}

func (b *BlockOutbound) Dial(network, addr string) (net.Conn, error) {
	return nil, fmt.Errorf("sova-core: blocked: %s", addr)
}

// SOVAOutbound — подключение через SOVA сервер
type SOVAOutbound struct {
	config    *Config
	dialer    func(network, addr string) (net.Conn, error)
	mu        sync.RWMutex
	connPool  map[string]net.Conn
}

// NewSOVAOutbound создаёт SOVA outbound
func NewSOVAOutbound(cfg *Config) *SOVAOutbound {
	o := &SOVAOutbound{
		config:   cfg,
		connPool: make(map[string]net.Conn),
	}

	// Определяем dialer
	if cfg.UpstreamProxy != "" {
		dialer, err := CreateUpstreamDialer(cfg.UpstreamProxy)
		if err == nil && dialer != nil {
			o.dialer = dialer
			return o
		}
	}

	if cfg.ServerAddr != "" {
		psk := cfg.PSK
		if psk == "" {
			psk = DefaultPSK
		}
		dpiCfg := DPIConfigFromConfig(cfg)
		o.dialer = CreateSOVARemoteDialer(cfg.ServerAddress(), psk, dpiCfg)
	}

	return o
}

func (o *SOVAOutbound) Dial(network, addr string) (net.Conn, error) {
	if o.dialer == nil {
		return nil, fmt.Errorf("sova-core: no SOVA server configured")
	}
	return o.dialer(network, addr)
}

func (o *SOVAOutbound) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Cleanup stale connections
			o.mu.Lock()
			for k, conn := range o.connPool {
				if conn == nil {
					delete(o.connPool, k)
				}
			}
			o.mu.Unlock()
		}
	}
}

// HTTPOutbound — через HTTP CONNECT прокси
type HTTPOutbound struct {
	dialer func(network, addr string) (net.Conn, error)
}

func (h *HTTPOutbound) Dial(network, addr string) (net.Conn, error) {
	if h.dialer == nil {
		return nil, fmt.Errorf("sova-core: HTTP outbound not configured")
	}
	return h.dialer(network, addr)
}
