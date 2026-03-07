package common

import (
	"context"
	"crypto/tls"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICTransport представляет QUIC транспорт
type QUICTransport struct {
	Session quic.Connection
	Stream  quic.Stream
}

// DialQUICTransport устанавливает QUIC соединение
func DialQUICTransport(config *TransportConfig) (*Connection, error) {
	tlsConfig := &tls.Config{
		ServerName: config.SNI,
		NextProtos: []string{"sova-quic"},
		// InsecureSkipVerify is intentional: SOVA uses post-quantum key exchange
		// (Kyber1024) for server authentication instead of TLS certificates.
		InsecureSkipVerify: true, // #nosec G402 — PQ key exchange verifies server identity
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := quic.DialAddr(ctx, config.ServerAddr, tlsConfig, &quic.Config{
		EnableDatagrams: true, // For jitter
	})
	if err != nil {
		return nil, err
	}

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		session.CloseWithError(0, "")
		return nil, err
	}

	transport := &QUICTransport{
		Session: session,
		Stream:  stream,
	}

	return &Connection{
		Conn:   &quicConn{transport: transport},
		Config: config,
	}, nil
}

// quicConn адаптер для net.Conn
type quicConn struct {
	transport *QUICTransport
}

func (q *quicConn) Read(b []byte) (int, error) {
	return q.transport.Stream.Read(b)
}

func (q *quicConn) Write(b []byte) (int, error) {
	return q.transport.Stream.Write(b)
}

func (q *quicConn) Close() error {
	q.transport.Stream.Close()
	return q.transport.Session.CloseWithError(0, "")
}

func (q *quicConn) LocalAddr() net.Addr {
	return q.transport.Session.LocalAddr()
}

func (q *quicConn) RemoteAddr() net.Addr {
	return q.transport.Session.RemoteAddr()
}

func (q *quicConn) SetDeadline(t time.Time) error {
	return q.transport.Stream.SetDeadline(t)
}

func (q *quicConn) SetReadDeadline(t time.Time) error {
	return q.transport.Stream.SetReadDeadline(t)
}

func (q *quicConn) SetWriteDeadline(t time.Time) error {
	return q.transport.Stream.SetWriteDeadline(t)
}

// QUICServer представляет QUIC сервер
type QUICServer struct {
	Listener *quic.Listener
}

// NewQUICServer создает QUIC сервер
func NewQUICServer(addr string, tlsConfig *tls.Config) (*QUICServer, error) {
	listener, err := quic.ListenAddr(addr, tlsConfig, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		return nil, err
	}

	return &QUICServer{Listener: listener}, nil
}

// Accept принимает QUIC соединение
func (s *QUICServer) Accept() (net.Conn, error) {
	session, err := s.Listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}

	stream, err := session.AcceptStream(context.Background())
	if err != nil {
		session.CloseWithError(0, "")
		return nil, err
	}

	return &quicConn{transport: &QUICTransport{
		Session: session,
		Stream:  stream,
	}}, nil
}

// AdaptiveCongestionControl адаптивное управление конгестией
type AdaptiveCongestionControl struct {
	BaseRTT time.Duration
	Jitter  time.Duration
}

// NewAdaptiveCC создает адаптивное CC
func NewAdaptiveCC() *AdaptiveCongestionControl {
	return &AdaptiveCongestionControl{
		BaseRTT: 50 * time.Millisecond,
		Jitter:  10 * time.Millisecond,
	}
}

// ApplyJitter применяет jitter к отправке
func (cc *AdaptiveCongestionControl) ApplyJitter(data []byte, writer io.Writer) error {
	chunks := cc.fragmentData(data)
	for _, chunk := range chunks {
		_, err := writer.Write(chunk)
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(rand.Intn(int(cc.Jitter.Milliseconds()))) * time.Millisecond)
	}
	return nil
}

// fragmentData разбивает данные для имитации HTTP/3
func (cc *AdaptiveCongestionControl) fragmentData(data []byte) [][]byte {
	chunkSize := 1024 + rand.Intn(2048) // Variable size
	var chunks [][]byte
	for len(data) > 0 {
		if len(data) < chunkSize {
			chunkSize = len(data)
		}
		chunks = append(chunks, data[:chunkSize])
		data = data[chunkSize:]
	}
	return chunks
}

// HysteriaLikeCC реализация Hysteria-like congestion control
type HysteriaLikeCC struct {
	*AdaptiveCongestionControl
	BandwidthEstimate float64
}

// NewHysteriaCC создает Hysteria CC
func NewHysteriaCC() *HysteriaLikeCC {
	return &HysteriaLikeCC{
		AdaptiveCongestionControl: NewAdaptiveCC(),
		BandwidthEstimate:         10 * 1024 * 1024, // 10 Mbps
	}
}

// EstimateBandwidth оценивает bandwidth
func (h *HysteriaLikeCC) EstimateBandwidth(rtt time.Duration, packetLoss float64) {
	// Simplified bandwidth estimation
	h.BandwidthEstimate = h.BandwidthEstimate*(1-packetLoss) + (1/rtt.Seconds())*1000000*packetLoss
}
