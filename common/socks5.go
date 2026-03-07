package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// SOCKS5Server представляет SOCKS5 прокси-сервер
type SOCKS5Server struct {
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

// SOCKS5 constants
const (
	socks5Version    = 0x05
	socks5AuthNone   = 0x00
	socks5CmdConnect = 0x01
	socks5CmdBind    = 0x02
	socks5CmdUDP     = 0x03
	socks5AtypIPv4   = 0x01
	socks5AtypDomain = 0x03
	socks5AtypIPv6   = 0x04
)

// NewSOCKS5Server создает SOCKS5 сервер
func NewSOCKS5Server(listenAddr string, ui *UI) *SOCKS5Server {
	return &SOCKS5Server{
		ListenAddr: listenAddr,
		UI:         ui,
		RemoteDialer: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 10*time.Second)
		},
	}
}

// Start запускает SOCKS5 сервер
func (s *SOCKS5Server) Start() error {
	listener, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("SOCKS5 listen failed: %v", err)
	}
	s.listener = listener
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	if s.UI != nil {
		s.UI.PrintSuccess(fmt.Sprintf("SOCKS5 прокси запущен на %s", s.ListenAddr))
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				s.mu.RLock()
				running := s.running
				s.mu.RUnlock()
				if !running {
					return
				}
				continue
			}
			atomic.AddInt64(&s.totalConns, 1)
			atomic.AddInt64(&s.activeConns, 1)
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// Stop останавливает SOCKS5 сервер
func (s *SOCKS5Server) Stop() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	if s.listener != nil {
		s.listener.Close()
	}
}

// GetStats возвращает статистику
func (s *SOCKS5Server) GetStats() map[string]int64 {
	return map[string]int64{
		"active_connections": atomic.LoadInt64(&s.activeConns),
		"total_connections":  atomic.LoadInt64(&s.totalConns),
		"bytes_up":           atomic.LoadInt64(&s.totalBytesUp),
		"bytes_down":         atomic.LoadInt64(&s.totalBytesDown),
	}
}

func (s *SOCKS5Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&s.activeConns, -1)
	}()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Handshake
	if err := s.handleHandshake(conn); err != nil {
		return
	}

	// Request
	targetAddr, err := s.handleRequest(conn)
	if err != nil {
		return
	}

	// Connect to target
	conn.SetDeadline(time.Time{})
	remote, err := s.RemoteDialer("tcp", targetAddr)
	if err != nil {
		s.sendReply(conn, 0x05) // Connection refused
		return
	}
	defer remote.Close()

	// Send success reply
	s.sendReply(conn, 0x00)

	// Tunnel
	s.tunnel(conn, remote)
}

func (s *SOCKS5Server) handleHandshake(conn net.Conn) error {
	// Read version and number of methods
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	if header[0] != socks5Version {
		return errors.New("unsupported SOCKS version")
	}

	numMethods := int(header[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Accept no-auth method
	_, err := conn.Write([]byte{socks5Version, socks5AuthNone})
	return err
}

func (s *SOCKS5Server) handleRequest(conn net.Conn) (string, error) {
	// Read request header: VER CMD RSV ATYP
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", err
	}

	if header[0] != socks5Version {
		return "", errors.New("unsupported version")
	}
	if header[1] != socks5CmdConnect {
		s.sendReply(conn, 0x07) // Command not supported
		return "", errors.New("unsupported command")
	}

	var host string
	switch header[3] {
	case socks5AtypIPv4:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()

	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domain := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		host = string(domain)

	case socks5AtypIPv6:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return "", err
		}
		host = net.IP(addr).String()

	default:
		return "", errors.New("unsupported address type")
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)

	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

func (s *SOCKS5Server) sendReply(conn net.Conn, status byte) {
	reply := []byte{
		socks5Version, status, 0x00, socks5AtypIPv4,
		0, 0, 0, 0, // Bind addr
		0, 0, // Bind port
	}
	conn.Write(reply)
}

func (s *SOCKS5Server) tunnel(local, remote net.Conn) {
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
			atomic.AddInt64(&s.totalBytesUp, int64(wn))
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
			atomic.AddInt64(&s.totalBytesDown, int64(wn))
		}
		done <- struct{}{}
	}()

	<-done
}

// CreateRemoteDialer создаёт dialer, который маршрутизирует трафик через удалённый SOVA сервер
func CreateRemoteDialer(serverAddr string) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		// Подключаемся к удалённому SOVA серверу
		serverConn, err := net.DialTimeout("tcp", serverAddr, 15*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SOVA server %s: %v", serverAddr, err)
		}

		// Отправляем целевой адрес серверу в формате: длина(1 байт) + адрес
		addrBytes := []byte(addr)
		if len(addrBytes) > 255 {
			serverConn.Close()
			return nil, fmt.Errorf("target address too long")
		}

		header := make([]byte, 1+len(addrBytes))
		header[0] = byte(len(addrBytes))
		copy(header[1:], addrBytes)

		if _, err := serverConn.Write(header); err != nil {
			serverConn.Close()
			return nil, fmt.Errorf("failed to send target address: %v", err)
		}

		// Читаем ответ (1 байт: 0 = успех, 1 = ошибка)
		resp := make([]byte, 1)
		serverConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if _, err := serverConn.Read(resp); err != nil {
			serverConn.Close()
			return nil, fmt.Errorf("no response from server: %v", err)
		}
		serverConn.SetReadDeadline(time.Time{})

		if resp[0] != 0 {
			serverConn.Close()
			return nil, fmt.Errorf("server refused connection to %s", addr)
		}

		return serverConn, nil
	}
}
