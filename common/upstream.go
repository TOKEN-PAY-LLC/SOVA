package common

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CreateUpstreamDialer creates a dialer that routes traffic through an upstream gateway.
// Supports: sova://host:port, http://host:port, https://host:port, or plain host:port (assumes SOVA).
func CreateUpstreamDialer(proxyURL string) (func(network, addr string) (net.Conn, error), error) {
	// Plain host:port — assume native SOVA gateway
	if !strings.Contains(proxyURL, "://") {
		if !strings.Contains(proxyURL, ":") {
			return nil, fmt.Errorf("invalid gateway address: %s (expected host:port)", proxyURL)
		}
		return createSOVADialer(proxyURL, DefaultPSK, DefaultDPIConfig()), nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway URL: %v", err)
	}

	switch u.Scheme {
	case "sova":
		psk := u.Query().Get("psk")
		if psk == "" {
			psk = DefaultPSK
		}
		dpiCfg := DefaultDPIConfig()
		if frag := u.Query().Get("frag"); frag != "" {
			if n, err := strconv.Atoi(frag); err == nil && n > 0 {
				dpiCfg.FragmentSize = n
			}
		}
		if jitter := u.Query().Get("jitter"); jitter != "" {
			if n, err := strconv.Atoi(jitter); err == nil && n >= 0 {
				dpiCfg.FragmentJitterMs = n
			}
		}
		if sni := u.Query().Get("sni"); sni != "" {
			dpiCfg.SNIList = strings.Split(sni, ",")
		}
		if u.Query().Get("stealth") == "off" {
			dpiCfg.Enabled = false
			dpiCfg.FragmentClientHello = false
		}
		return createSOVADialer(u.Host, psk, dpiCfg), nil
	case "http", "https":
		return createHTTPConnectDialer(u.Host), nil
	default:
		return nil, fmt.Errorf("unsupported gateway scheme: %s (use sova:// or http://)", u.Scheme)
	}
}

// createSOVADialer chains through an upstream SOVA gateway
func createSOVADialer(serverAddr, psk string, dpiCfg *DPIConfig) func(network, addr string) (net.Conn, error) {
	return CreateSOVARemoteDialer(serverAddr, psk, dpiCfg)
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
