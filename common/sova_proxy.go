package common

// ── SOVA Proxy — собственный прокси-сервер SOVA ────────────────────────
//
// Полностью автономный входной шлюз SOVA.
// SOVA сам себе прокси, сам себе протокол, сам себе маршрутизатор.
//
// Локальный прокси принимает подключения от браузеров/приложений:
//   - HTTP CONNECT (нативная поддержка в браузерах и системном прокси)
//   - Plain HTTP proxy requests
//
// Маршрутизация:
//   - Напрямую (direct) — для локального режима
//   - Через SOVA сервер (RemoteDialer) — для удалённого режима
//     (TLS + DPI evasion + AES-256-GCM зашифрованные фреймы)
//
// ────────────────────────────────────────────────────────────────────────

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SOVAProxy — собственный прокси-сервер SOVA
type SOVAProxy struct {
	ListenAddr     string
	RemoteDialer   func(network, addr string) (net.Conn, error)
	UI             *UI
	listener       net.Listener
	activeConns    int64
	totalConns     int64
	totalBytesUp   int64
	totalBytesDown int64
	mu             sync.RWMutex
	running        bool
}

// NewSOVAProxy создаёт новый SOVA прокси-сервер
func NewSOVAProxy(listenAddr string, ui *UI) *SOVAProxy {
	return &SOVAProxy{
		ListenAddr: listenAddr,
		UI:         ui,
		RemoteDialer: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 10*time.Second)
		},
	}
}

// Start запускает SOVA прокси
func (p *SOVAProxy) Start() error {
	listener, err := net.Listen("tcp", p.ListenAddr)
	if err != nil {
		return fmt.Errorf("SOVA proxy listen failed: %v", err)
	}
	p.listener = listener
	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	if p.UI != nil {
		p.UI.PrintSuccess(fmt.Sprintf("SOVA прокси запущен на %s", p.ListenAddr))
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				p.mu.RLock()
				running := p.running
				p.mu.RUnlock()
				if !running {
					return
				}
				continue
			}
			atomic.AddInt64(&p.totalConns, 1)
			atomic.AddInt64(&p.activeConns, 1)
			go p.handleConnection(conn)
		}
	}()

	return nil
}

// Stop останавливает SOVA прокси
func (p *SOVAProxy) Stop() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
	if p.listener != nil {
		p.listener.Close()
	}
}

// GetStats возвращает статистику
func (p *SOVAProxy) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&p.activeConns),
		"total_connections":  atomic.LoadInt64(&p.totalConns),
		"bytes_up":           atomic.LoadInt64(&p.totalBytesUp),
		"bytes_down":         atomic.LoadInt64(&p.totalBytesDown),
	}
}

// handleConnection определяет тип подключения и обрабатывает его.
// SOVA Proxy принимает только HTTP CONNECT и plain HTTP запросы.
func (p *SOVAProxy) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&p.activeConns, -1)
	}()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Peek первый байт для авто-детекта протокола
	buf := make([]byte, 1)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}

	if buf[0] == 0x05 {
		return
	}

	// HTTP CONNECT — основной режим SOVA прокси
	p.handleHTTPConnect(conn, buf[0])
}

// ── HTTP CONNECT — основной протокол SOVA прокси ────────────────────────

func (p *SOVAProxy) handleHTTPConnect(conn net.Conn, firstByte byte) {
	// Дочитываем HTTP запрос (первый байт уже прочитан)
	reader := bufio.NewReader(io.MultiReader(
		strings.NewReader(string(firstByte)),
		conn,
	))

	req, err := http.ReadRequest(reader)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	if req.Method == "CONNECT" {
		// CONNECT host:port — туннелирование
		p.handleConnectMethod(conn, req.Host)
	} else {
		// Обычный HTTP запрос — проксируем
		p.handlePlainHTTP(conn, req, reader)
	}
}

func (p *SOVAProxy) handleConnectMethod(conn net.Conn, targetAddr string) {
	// Нормализуем адрес
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":443"
	}

	conn.SetDeadline(time.Time{})

	// Подключаемся к целевому серверу (напрямую или через SOVA сервер)
	remote, err := p.RemoteDialer("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	// Отправляем 200 — туннель установлен
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Двунаправленный relay
	p.tunnel(conn, remote)
}

func (p *SOVAProxy) handlePlainHTTP(conn net.Conn, req *http.Request, reader *bufio.Reader) {
	// Определяем целевой адрес из Host
	targetAddr := req.Host
	if !strings.Contains(targetAddr, ":") {
		targetAddr = targetAddr + ":80"
	}

	conn.SetDeadline(time.Time{})

	// Подключаемся к целевому серверу
	remote, err := p.RemoteDialer("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer remote.Close()

	// Пересылаем оригинальный запрос
	req.Write(remote)

	// Relay ответ обратно клиенту
	p.tunnel(conn, remote)
}

// ── Tunnel ──────────────────────────────────────────────────────────────

func (p *SOVAProxy) tunnel(local, remote net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := local.Read(buf)
			if err != nil {
				break
			}
			wn, err := remote.Write(buf[:n])
			if err != nil {
				break
			}
			atomic.AddInt64(&p.totalBytesUp, int64(wn))
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := remote.Read(buf)
			if err != nil {
				break
			}
			wn, err := local.Write(buf[:n])
			if err != nil {
				break
			}
			atomic.AddInt64(&p.totalBytesDown, int64(wn))
		}
		done <- struct{}{}
	}()

	<-done
}
