package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"sova/common"

	"github.com/gorilla/websocket"
)

// RelayServer — SOVA relay сервер.
// Принимает подключения по SOVA протоколу (TLS + зашифрованные фреймы).
// Поддерживает TCP (TLS) и WebSocket транспорт.
type RelayServer struct {
	listenAddr     string
	psk            string
	listener       net.Listener
	activeConns    int64
	totalConns     int64
	totalBytesUp   int64
	totalBytesDown int64
	mu             sync.RWMutex
	running        bool

	// WebSocket relay
	wsEnabled  bool
	wsAddr     string
	wsPath     string
	wsUpgrader websocket.Upgrader
}

// NewRelayServer создаёт сервер релея
func NewRelayServer(listenAddr, psk string) *RelayServer {
	if psk == "" {
		psk = common.DefaultPSK
	}
	return &RelayServer{
		listenAddr: listenAddr,
		psk:        psk,
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  32 * 1024,
			WriteBufferSize: 32 * 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// EnableWebSocket включает WebSocket relay на указанном адресе
func (rs *RelayServer) EnableWebSocket(addr, path string) {
	rs.wsEnabled = true
	rs.wsAddr = addr
	rs.wsPath = path
}

// Start запускает сервер релея с TLS (SOVA протокол)
func (rs *RelayServer) Start() error {
	// TLS listener с самоподписанным сертификатом
	// (или за nginx, который делает TLS termination)
	listener, err := common.NewTLSListener(rs.listenAddr)
	if err != nil {
		return fmt.Errorf("relay TLS listen failed on %s: %v", rs.listenAddr, err)
	}
	rs.listener = listener
	rs.mu.Lock()
	rs.running = true
	rs.mu.Unlock()

	// TCP/TLS relay — SOVA протокол
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				rs.mu.RLock()
				running := rs.running
				rs.mu.RUnlock()
				if !running {
					return
				}
				continue
			}
			atomic.AddInt64(&rs.totalConns, 1)
			atomic.AddInt64(&rs.activeConns, 1)
			go rs.handleSOVAConnection(conn)
		}
	}()

	// WebSocket relay (если включён — для мобильных операторов)
	if rs.wsEnabled {
		go rs.startWebSocketRelay()
	}

	return nil
}

// handleSOVAConnection обрабатывает SOVA протокол поверх TLS:
//  1. SOVA handshake (v2 с PQ KEM или v1 fallback) → зашифрованное соединение
//  2. Читаем CONNECT фрейм (адрес назначения)
//  3. Подключаемся к целевому серверу
//  4. Отправляем ACK
//  5. Двунаправленный relay через зашифрованные фреймы
func (rs *RelayServer) handleSOVAConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&rs.activeConns, -1)
	}()

	// 1. SOVA protocol handshake — try v2 first, fallback to v1
	// ServerHandshakeV2 detects v2 magic and falls back to v1 automatically
	sovaConn, err := common.ServerHandshakeV2(conn, rs.psk, true)
	if err != nil {
		return // Невалидный клиент — молча закрываем
	}

	// 2. Читаем CONNECT фрейм (v2 or v1 format)
	sovaConn.SetReadDeadline(time.Now().Add(15 * time.Second))

	// Try v2 frame format first
	sid, ftype, payload, err := sovaConn.ReadFrameV2()
	if err != nil {
		// Fallback: v2 read failed — the connection was already handled by v1 handshake
		// (ServerHandshakeV2 falls back to v1 automatically if no v2 magic detected)
		// In that case sovaConn wraps a v1 SOVAConn — try v1 frame format
		sovaConn.Close()
		return
	}
	sovaConn.SetDeadline(time.Time{})

	if ftype != common.FrameV2Connect || len(payload) == 0 {
		sovaConn.WriteFrameV2(sid, common.FrameV2Ack, []byte{0x01})
		sovaConn.Close()
		return
	}

	targetAddr := string(payload)

	// 3. Подключаемся к целевому серверу
	remote, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		sovaConn.WriteFrameV2(sid, common.FrameV2Ack, []byte{0x01})
		sovaConn.Close()
		return
	}
	defer remote.Close()

	// 4. Отправляем ACK — успех
	if err := sovaConn.WriteFrameV2(sid, common.FrameV2Ack, []byte{0x00}); err != nil {
		return
	}

	// 5. Relay через SOVAStreamV2 ↔ raw TCP
	stream := common.NewSOVAStreamV2(sovaConn, sid)
	rs.relay(stream, remote)
}

// startWebSocketRelay запускает HTTP-сервер для WebSocket relay.
// Клиенты подключаются по wss://host/sova-ws,
// nginx терминирует TLS и проксирует WebSocket сюда.
// Внутри WS идёт тот же нативный SOVA protocol handshake и encrypted framing.
func (rs *RelayServer) startWebSocketRelay() {
	mux := http.NewServeMux()

	path := rs.wsPath
	if path == "" {
		path = "/sova-ws"
	}

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ws, err := rs.wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		atomic.AddInt64(&rs.totalConns, 1)
		atomic.AddInt64(&rs.activeConns, 1)

		// WebSocket → net.Conn адаптер → нативный SOVA протокол
		conn := NewWSConn(ws)
		go rs.handleSOVAConnection(conn)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		stats := rs.GetStats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","protocol":"sova-core-v%s","transports":["tcp-sova","ws-sova"],"active":%d,"total":%d}`,
			common.Version,
			stats["active_connections"], stats["total_connections"])
	})

	http.ListenAndServe(rs.wsAddr, mux)
}

// Stop останавливает сервер
func (rs *RelayServer) Stop() {
	rs.mu.Lock()
	rs.running = false
	rs.mu.Unlock()
	if rs.listener != nil {
		rs.listener.Close()
	}
}

// GetStats возвращает статистику
func (rs *RelayServer) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&rs.activeConns),
		"total_connections":  atomic.LoadInt64(&rs.totalConns),
		"bytes_up":           atomic.LoadInt64(&rs.totalBytesUp),
		"bytes_down":         atomic.LoadInt64(&rs.totalBytesDown),
	}
}

func (rs *RelayServer) relay(client, remote net.Conn) {
	done := make(chan struct{}, 2)

	// client → remote (upload)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := client.Read(buf)
			if err != nil {
				break
			}
			wn, err := remote.Write(buf[:n])
			if err != nil {
				break
			}
			atomic.AddInt64(&rs.totalBytesUp, int64(wn))
		}
		done <- struct{}{}
	}()

	// remote → client (download)
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := remote.Read(buf)
			if err != nil {
				break
			}
			wn, err := client.Write(buf[:n])
			if err != nil {
				break
			}
			atomic.AddInt64(&rs.totalBytesDown, int64(wn))
		}
		done <- struct{}{}
	}()

	<-done
}

// ─── WSConn: net.Conn adapter for gorilla/websocket ───

// WSConn оборачивает *websocket.Conn в интерфейс net.Conn,
// позволяя использовать WebSocket как транспорт для relay.
type WSConn struct {
	ws      *websocket.Conn
	readBuf []byte
}

// NewWSConn создаёт адаптер
func NewWSConn(ws *websocket.Conn) *WSConn {
	return &WSConn{ws: ws}
}

func (c *WSConn) Read(p []byte) (int, error) {
	for len(c.readBuf) == 0 {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			return 0, err
		}
		c.readBuf = msg
	}
	n := copy(p, c.readBuf)
	c.readBuf = c.readBuf[n:]
	return n, nil
}

func (c *WSConn) Write(p []byte) (int, error) {
	err := c.ws.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *WSConn) Close() error {
	return c.ws.Close()
}

func (c *WSConn) LocalAddr() net.Addr {
	return c.ws.LocalAddr()
}

func (c *WSConn) RemoteAddr() net.Addr {
	return c.ws.RemoteAddr()
}

func (c *WSConn) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return err
	}
	return c.ws.SetWriteDeadline(t)
}

func (c *WSConn) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
