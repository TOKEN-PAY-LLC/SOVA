package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// RelayServer принимает подключения от SOVA клиентов и релеит трафик
type RelayServer struct {
	listenAddr   string
	listener     net.Listener
	activeConns  int64
	totalConns   int64
	totalBytesUp int64
	totalBytesDown int64
	mu           sync.RWMutex
	running      bool
}

// NewRelayServer создаёт сервер релея
func NewRelayServer(listenAddr string) *RelayServer {
	return &RelayServer{
		listenAddr: listenAddr,
	}
}

// Start запускает сервер релея
func (rs *RelayServer) Start() error {
	listener, err := net.Listen("tcp", rs.listenAddr)
	if err != nil {
		return fmt.Errorf("relay listen failed on %s: %v", rs.listenAddr, err)
	}
	rs.listener = listener
	rs.mu.Lock()
	rs.running = true
	rs.mu.Unlock()

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
			go rs.handleConnection(conn)
		}
	}()

	return nil
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

func (rs *RelayServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&rs.activeConns, -1)
	}()

	// Читаем целевой адрес от клиента
	// Протокол: 1 байт длина + адрес (host:port)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	lenBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return
	}

	addrLen := int(lenBuf[0])
	if addrLen == 0 || addrLen > 255 {
		conn.Write([]byte{1}) // ошибка
		return
	}

	addrBuf := make([]byte, addrLen)
	if _, err := io.ReadFull(conn, addrBuf); err != nil {
		conn.Write([]byte{1}) // ошибка
		return
	}

	targetAddr := string(addrBuf)
	conn.SetReadDeadline(time.Time{})

	// Подключаемся к целевому серверу
	remote, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		conn.Write([]byte{1}) // ошибка — не удалось подключиться
		return
	}
	defer remote.Close()

	// Отправляем клиенту успех
	if _, err := conn.Write([]byte{0}); err != nil {
		return
	}

	// Двунаправленный relay
	rs.relay(conn, remote)
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
