package common

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketTransport представляет WebSocket транспорт через CDN
type WebSocketTransport struct {
	Conn *websocket.Conn
}

// DialWebSocketTransport устанавливает WebSocket через CDN
func DialWebSocketTransport(config *TransportConfig) (*Connection, error) {
	// Выбрать CDN IP (Cloudflare, etc.)
	cdnIPs := []string{
		"104.16.0.0/20",  // Cloudflare
		"173.245.48.0/20",
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"141.101.64.0/18",
		"108.162.192.0/18",
		"190.93.240.0/20",
		"188.114.96.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"162.158.0.0/15",
		"104.24.0.0/14",
		"172.64.0.0/13",
		"131.0.255.0/16", // Amazon CloudFront
	}

	// Ротация IP для обхода блокировок
	cdnIP := cdnIPs[rand.Intn(len(cdnIPs))]

	u := url.URL{
		Scheme: "wss",
		Host:   cdnIP + ":443",
		Path:   "/sova-tunnel",
	}

	headers := http.Header{}
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	headers.Set("Origin", "https://"+config.SNI)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			ServerName: config.SNI,
			InsecureSkipVerify: true, // For CDN
		},
	}

	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return nil, fmt.Errorf("WebSocket dial failed: %v", err)
	}

	transport := &WebSocketTransport{Conn: conn}

	return &Connection{
		Conn: &wsConn{transport: transport},
		Config: config,
	}, nil
}

// wsConn адаптер для net.Conn
type wsConn struct {
	transport *WebSocketTransport
}

func (w *wsConn) Read(b []byte) (int, error) {
	_, data, err := w.transport.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(b, data)
	return len(data), nil
}

func (w *wsConn) Write(b []byte) (int, error) {
	err := w.transport.Conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *wsConn) Close() error {
	return w.transport.Conn.Close()
}

func (w *wsConn) LocalAddr() net.Addr {
	return w.transport.Conn.LocalAddr()
}

func (w *wsConn) RemoteAddr() net.Addr {
	return w.transport.Conn.RemoteAddr()
}

func (w *wsConn) SetDeadline(t time.Time) error {
	return w.transport.Conn.SetReadDeadline(t)
}

func (w *wsConn) SetReadDeadline(t time.Time) error {
	return w.transport.Conn.SetReadDeadline(t)
}

func (w *wsConn) SetWriteDeadline(t time.Time) error {
	return w.transport.Conn.SetWriteDeadline(t)
}

// CDNWorkerIntegration интеграция с Cloudflare Workers
type CDNWorkerIntegration struct {
	WorkerURL string
}

// NewCDNWorker создает интеграцию
func NewCDNWorker(workerURL string) *CDNWorkerIntegration {
	return &CDNWorkerIntegration{WorkerURL: workerURL}
}

// DeployWorkerScript скрипт для Cloudflare Worker
func (c *CDNWorkerIntegration) DeployWorkerScript() string {
	return `
addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request))
})

async function handleRequest(request) {
  // Проксировать SOVA трафик на реальный сервер
  const sovaServer = 'https://your-sova-server.com'

  const newRequest = new Request(sovaServer + request.url.pathname, {
    method: request.method,
    headers: request.headers,
    body: request.body
  })

  return fetch(newRequest)
}
`
}

// ServerlessFunctionIntegration для других CDN
type ServerlessFunctionIntegration struct {
	FunctionURL string
}

// NewServerlessFunction создает интеграцию
func NewServerlessFunction(functionURL string) *ServerlessFunctionIntegration {
	return &ServerlessFunctionIntegration{FunctionURL: functionURL}
}

// LambdaScript для AWS Lambda
func (s *ServerlessFunctionIntegration) LambdaScript() string {
	return `
exports.handler = async (event) => {
    // Проксировать на SOVA сервер
    const sovaServer = 'https://your-sova-server.com'
    
    const response = await fetch(sovaServer + event.path, {
        method: event.httpMethod,
        headers: event.headers,
        body: event.body
    })
    
    return {
        statusCode: response.status,
        headers: Object.fromEntries(response.headers),
        body: await response.text()
    }
}
`
}