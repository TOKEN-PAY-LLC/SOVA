package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CreateUpstreamDialer creates a dialer that routes traffic through an upstream proxy.
// Supports: socks5://host:port, http://host:port, or plain host:port (assumes SOCKS5).
func CreateUpstreamDialer(proxyURL string) (func(network, addr string) (net.Conn, error), error) {
	// Plain host:port — assume SOCKS5
	if !strings.Contains(proxyURL, "://") {
		if !strings.Contains(proxyURL, ":") {
			return nil, fmt.Errorf("invalid proxy address: %s (expected host:port)", proxyURL)
		}
		return createSOCKS5Dialer(proxyURL), nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %v", err)
	}

	switch u.Scheme {
	case "socks5", "socks":
		return createSOCKS5Dialer(u.Host), nil
	case "http", "https":
		return createHTTPConnectDialer(u.Host), nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s (use socks5:// or http://)", u.Scheme)
	}
}

// createSOCKS5Dialer chains through an upstream SOCKS5 proxy
func createSOCKS5Dialer(proxyAddr string) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout("tcp", proxyAddr, 15*time.Second)
		if err != nil {
			return nil, fmt.Errorf("upstream SOCKS5 connect failed (%s): %v", proxyAddr, err)
		}

		// SOCKS5 handshake: version 5, 1 method, no-auth
		if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
			conn.Close()
			return nil, fmt.Errorf("upstream handshake write failed: %v", err)
		}

		resp := make([]byte, 2)
		if _, err := io.ReadFull(conn, resp); err != nil {
			conn.Close()
			return nil, fmt.Errorf("upstream handshake read failed: %v", err)
		}
		if resp[0] != 0x05 || resp[1] != 0x00 {
			conn.Close()
			return nil, errors.New("upstream SOCKS5 rejected auth method")
		}

		// Parse target
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			conn.Close()
			return nil, err
		}
		port, _ := strconv.Atoi(portStr)

		// Build CONNECT request
		req := []byte{0x05, 0x01, 0x00} // VER, CMD=CONNECT, RSV

		ip := net.ParseIP(host)
		if ip != nil && ip.To4() != nil {
			req = append(req, 0x01) // IPv4
			req = append(req, ip.To4()...)
		} else if ip != nil {
			req = append(req, 0x04) // IPv6
			req = append(req, ip.To16()...)
		} else {
			req = append(req, 0x03) // Domain
			req = append(req, byte(len(host)))
			req = append(req, []byte(host)...)
		}

		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, uint16(port))
		req = append(req, portBytes...)

		if _, err := conn.Write(req); err != nil {
			conn.Close()
			return nil, fmt.Errorf("upstream CONNECT write failed: %v", err)
		}

		// Read response header (4 bytes: VER REP RSV ATYP)
		reply := make([]byte, 4)
		if _, err := io.ReadFull(conn, reply); err != nil {
			conn.Close()
			return nil, fmt.Errorf("upstream CONNECT read failed: %v", err)
		}
		if reply[1] != 0x00 {
			conn.Close()
			return nil, fmt.Errorf("upstream SOCKS5 refused connection (status %d)", reply[1])
		}

		// Drain remaining bind address bytes
		switch reply[3] {
		case 0x01: // IPv4 + port
			io.ReadFull(conn, make([]byte, 4+2))
		case 0x03: // Domain + port
			lenBuf := make([]byte, 1)
			io.ReadFull(conn, lenBuf)
			io.ReadFull(conn, make([]byte, int(lenBuf[0])+2))
		case 0x04: // IPv6 + port
			io.ReadFull(conn, make([]byte, 16+2))
		}

		return conn, nil
	}
}

// createHTTPConnectDialer chains through an upstream HTTP CONNECT proxy
func createHTTPConnectDialer(proxyAddr string) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout("tcp", proxyAddr, 15*time.Second)
		if err != nil {
			return nil, fmt.Errorf("upstream HTTP proxy connect failed (%s): %v", proxyAddr, err)
		}

		// Send CONNECT
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: Keep-Alive\r\n\r\n", addr, addr)
		if _, err := conn.Write([]byte(connectReq)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("HTTP CONNECT write failed: %v", err)
		}

		// Read response (look for "200")
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(buf)
		conn.SetReadDeadline(time.Time{})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("HTTP CONNECT read failed: %v", err)
		}

		respLine := string(buf[:n])
		if !strings.Contains(respLine, "200") {
			conn.Close()
			return nil, fmt.Errorf("HTTP CONNECT rejected: %s", strings.TrimSpace(strings.Split(respLine, "\r\n")[0]))
		}

		return conn, nil
	}
}
