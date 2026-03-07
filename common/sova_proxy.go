package common

// ── SOVA Proxy — собственный прокси-сервер SOVA ────────────────────────
//
// Полностью автономный прокси. НЕ SOCKS5, НЕ HTTP-прокси стороннего формата.
// SOVA сам себе прокси, сам себе протокол, сам себе маршрутизатор.
//
// Локальный прокси принимает подключения от браузеров/приложений:
//   - HTTP CONNECT (нативная поддержка в браузерах и системном прокси Windows)
//   - Авто-детект устаревших SOCKS5 клиентов для обратной совместимости
//
// Маршрутизация:
//   - Напрямую (direct) — для локального режима
//   - Через SOVA сервер (RemoteDialer) — для удалённого режима
//     (TLS + DPI evasion + AES-256-GCM зашифрованные фреймы)
//
// ────────────────────────────────────────────────────────────────────────

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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
// Авто-детект: первый байт 0x05 = legacy SOCKS5, иначе HTTP CONNECT.
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
		// Legacy SOCKS5 совместимость (для curl и др.)
		p.handleLegacySocks(conn, buf[0])
	} else {
		// HTTP CONNECT — основной режим SOVA прокси
		p.handleHTTPConnect(conn, buf[0])
	}
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

// ── Legacy SOCKS5 совместимость ─────────────────────────────────────────
// Для приложений (curl, etc.) которые используют SOCKS5.
// SOVA обрабатывает их прозрачно, но это НЕ основной протокол.

func (p *SOVAProxy) handleLegacySocks(conn net.Conn, versionByte byte) {
	// versionByte уже прочитан (0x05)
	// Читаем количество методов аутентификации
	numBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, numBuf); err != nil {
		return
	}
	methods := make([]byte, numBuf[0])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}

	// Принимаем no-auth
	conn.Write([]byte{0x05, 0x00})

	// Читаем CONNECT запрос: VER CMD RSV ATYP
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}
	if header[1] != 0x01 { // Только CONNECT
		p.socksReply(conn, 0x07)
		return
	}

	// Парсим адрес
	var host string
	switch header[3] {
	case 0x01: // IPv4
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		host = net.IP(addr).String()
	case 0x03: // Domain
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return
		}
		host = string(domain)
	case 0x04: // IPv6
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		host = net.IP(addr).String()
	default:
		p.socksReply(conn, 0x08)
		return
	}

	// Порт
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBuf)
	targetAddr := net.JoinHostPort(host, strconv.Itoa(int(port)))

	conn.SetDeadline(time.Time{})

	// Подключаемся
	remote, err := p.RemoteDialer("tcp", targetAddr)
	if err != nil {
		p.socksReply(conn, 0x05)
		return
	}
	defer remote.Close()

	// Успех
	p.socksReply(conn, 0x00)

	// Tunnel
	p.tunnel(conn, remote)
}

func (p *SOVAProxy) socksReply(conn net.Conn, status byte) {
	conn.Write([]byte{0x05, status, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
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
