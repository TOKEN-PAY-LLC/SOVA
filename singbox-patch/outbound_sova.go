//go:build ignore

package sova_patch

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// ══════════════════════════════════════════════════════════════════════════
// SOVA Wire Protocol v1 (встроенная реализация)
// ══════════════════════════════════════════════════════════════════════════

const (
	sovaProtoMagic   uint32 = 0x534F5641 // "SOVA" in ASCII hex
	sovaProtoVersion byte   = 0x01

	sovaFrameConnect   byte = 0x01
	sovaFrameData      byte = 0x02
	sovaFrameClose     byte = 0x03
	sovaFrameKeepalive byte = 0x04
	sovaFrameAck       byte = 0x05

	sovaMaxFramePayload = 60000
	sovaHandshakeSize   = 4 + 1 + 16 + 32 // magic + version + clientSalt + random
	sovaServerAckSize   = 1 + 16          // ack + serverSalt

	sovaDefaultPSK = "sova-protocol-v1-key-2026"
)

// ── sovaConn: зашифрованное фреймовое соединение ─────────────────────────

type sovaConn struct {
	conn    net.Conn
	aead    cipher.AEAD
	readMu  sync.Mutex
	writeMu sync.Mutex
	nonce   uint64
}

func newSovaConn(conn net.Conn, key []byte) (*sovaConn, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sova: AES init: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sova: GCM init: %v", err)
	}
	return &sovaConn{conn: conn, aead: gcm}, nil
}

func (sc *sovaConn) writeFrame(ftype byte, payload []byte) error {
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()

	// Random padding 4–64 байт
	padLen := 4 + int(sovaSecureRandByte()%61)
	padding := make([]byte, padLen)
	rand.Read(padding)

	// Plaintext: [PadLen:1][Type:1][Payload:N][Padding:P]
	plain := make([]byte, 0, 2+len(payload)+padLen)
	plain = append(plain, byte(padLen), ftype)
	plain = append(plain, payload...)
	plain = append(plain, padding...)

	// Nonce (12 байт, counter)
	nonce := make([]byte, sc.aead.NonceSize())
	nVal := atomic.AddUint64(&sc.nonce, 1)
	binary.BigEndian.PutUint64(nonce[4:], nVal)

	// AES-256-GCM seal
	sealed := sc.aead.Seal(nil, nonce, plain, nil)

	// Wire: [Nonce:12][Length:2][Sealed:N]
	wire := make([]byte, len(nonce)+2+len(sealed))
	copy(wire, nonce)
	binary.BigEndian.PutUint16(wire[len(nonce):], uint16(len(sealed)))
	copy(wire[len(nonce)+2:], sealed)

	_, err := sc.conn.Write(wire)
	return err
}

func (sc *sovaConn) readFrame() (byte, []byte, error) {
	sc.readMu.Lock()
	defer sc.readMu.Unlock()

	nonce := make([]byte, sc.aead.NonceSize())
	if _, err := io.ReadFull(sc.conn, nonce); err != nil {
		return 0, nil, err
	}

	lb := make([]byte, 2)
	if _, err := io.ReadFull(sc.conn, lb); err != nil {
		return 0, nil, err
	}
	n := int(binary.BigEndian.Uint16(lb))
	if n > sovaMaxFramePayload+200 {
		return 0, nil, errors.New("sova: frame too large")
	}

	ct := make([]byte, n)
	if _, err := io.ReadFull(sc.conn, ct); err != nil {
		return 0, nil, err
	}

	plain, err := sc.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("sova: decrypt: %v", err)
	}
	if len(plain) < 2 {
		return 0, nil, errors.New("sova: frame too small")
	}

	padLen := int(plain[0])
	ftype := plain[1]
	data := plain[2:]
	if padLen > len(data) {
		return 0, nil, errors.New("sova: bad padding")
	}
	payload := data[:len(data)-padLen]

	return ftype, append([]byte{}, payload...), nil
}

func (sc *sovaConn) close() error { return sc.conn.Close() }

// ── sovaStream: net.Conn-совместимая обёртка ─────────────────────────────

type sovaStream struct {
	sc      *sovaConn
	readBuf []byte
	closed  bool
	mu      sync.Mutex
}

func newSovaStream(sc *sovaConn) *sovaStream {
	return &sovaStream{sc: sc}
}

func (s *sovaStream) Read(p []byte) (int, error) {
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}
	for {
		ftype, payload, err := s.sc.readFrame()
		if err != nil {
			return 0, err
		}
		switch ftype {
		case sovaFrameData:
			n := copy(p, payload)
			if n < len(payload) {
				s.readBuf = payload[n:]
			}
			return n, nil
		case sovaFrameClose:
			return 0, io.EOF
		case sovaFrameKeepalive:
			continue
		default:
			continue
		}
	}
}

func (s *sovaStream) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		chunk := p
		if len(chunk) > sovaMaxFramePayload {
			chunk = p[:sovaMaxFramePayload]
		}
		if err := s.sc.writeFrame(sovaFrameData, chunk); err != nil {
			return total, err
		}
		total += len(chunk)
		p = p[len(chunk):]
	}
	return total, nil
}

func (s *sovaStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.sc.writeFrame(sovaFrameClose, nil)
	return s.sc.close()
}

func (s *sovaStream) LocalAddr() net.Addr                { return s.sc.conn.LocalAddr() }
func (s *sovaStream) RemoteAddr() net.Addr               { return s.sc.conn.RemoteAddr() }
func (s *sovaStream) SetDeadline(t time.Time) error      { return s.sc.conn.SetDeadline(t) }
func (s *sovaStream) SetReadDeadline(t time.Time) error  { return s.sc.conn.SetReadDeadline(t) }
func (s *sovaStream) SetWriteDeadline(t time.Time) error { return s.sc.conn.SetWriteDeadline(t) }

// ── SOVA Handshake ───────────────────────────────────────────────────────

func sovaClientHandshake(conn net.Conn, psk string) (*sovaConn, error) {
	clientSalt := make([]byte, 16)
	randomPad := make([]byte, 32)
	rand.Read(clientSalt)
	rand.Read(randomPad)

	// Send: Magic(4) + Version(1) + ClientSalt(16) + Random(32)
	hs := make([]byte, sovaHandshakeSize)
	binary.BigEndian.PutUint32(hs[0:4], sovaProtoMagic)
	hs[4] = sovaProtoVersion
	copy(hs[5:21], clientSalt)
	copy(hs[21:53], randomPad)

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write(hs); err != nil {
		return nil, fmt.Errorf("sova handshake: write: %v", err)
	}

	// Read: ACK(1) + ServerSalt(16)
	resp := make([]byte, sovaServerAckSize)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, fmt.Errorf("sova handshake: server response: %v", err)
	}
	conn.SetDeadline(time.Time{})

	if resp[0] != 0x00 {
		return nil, errors.New("sova handshake: server rejected")
	}
	serverSalt := resp[1:17]

	// SessionKey = SHA-256(PSK || ClientSalt || ServerSalt)
	h := sha256.New()
	h.Write([]byte(psk))
	h.Write(clientSalt)
	h.Write(serverSalt)
	key := h.Sum(nil)

	return newSovaConn(conn, key)
}

// ── DPI Evasion: TCP фрагментатор ClientHello ────────────────────────────

type sovaFragConn struct {
	net.Conn
	fragSize int
	fragDone bool
	jitterMs int
}

func newFragConn(conn net.Conn, fragSize, jitterMs int) *sovaFragConn {
	if fragSize < 1 {
		fragSize = 2
	}
	return &sovaFragConn{Conn: conn, fragSize: fragSize, jitterMs: jitterMs}
}

func (fc *sovaFragConn) Write(p []byte) (int, error) {
	if fc.fragDone {
		return fc.Conn.Write(p)
	}
	fc.fragDone = true

	total := 0
	for len(p) > 0 {
		sz := fc.fragSize
		if sz > len(p) {
			sz = len(p)
		}
		n, err := fc.Conn.Write(p[:sz])
		total += n
		if err != nil {
			return total, err
		}
		p = p[sz:]
		if len(p) > 0 && fc.jitterMs > 0 {
			jitter := 1 + int(sovaSecureRandByte())%fc.jitterMs
			time.Sleep(time.Duration(jitter) * time.Millisecond)
		}
	}
	return total, nil
}

func sovaSecureRandByte() byte {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]
}

// ══════════════════════════════════════════════════════════════════════════
// sing-box Outbound Adapter
// ══════════════════════════════════════════════════════════════════════════

var _ adapter.Outbound = (*SOVA)(nil)

// SOVA outbound для sing-box — подключается к SOVA серверу по нативному протоколу
type SOVA struct {
	myOutboundAdapter
	dialer     N.Dialer
	serverAddr M.Socksaddr
	psk        string
	sniList    []string
	fragSize   int
	fragJitter int
	tlsEnabled bool
	tlsSNI     string
}

// NewSOVA создаёт SOVA outbound
func NewSOVA(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.SOVAOutboundOptions) (*SOVA, error) {
	outbound := &SOVA{
		myOutboundAdapter: myOutboundAdapter{
			protocol:     C.TypeSOVA,
			network:      []string{N.NetworkTCP},
			router:       router,
			logger:       logger,
			tag:          tag,
			dependencies: withDialerDependency(options.DialerOptions),
		},
		serverAddr: options.ServerOptions.Build(),
		psk:        options.PSK,
		sniList:    options.SNIList,
		fragSize:   options.FragmentSize,
		fragJitter: options.FragmentJitter,
	}

	if outbound.psk == "" {
		outbound.psk = sovaDefaultPSK
	}
	if len(outbound.sniList) == 0 {
		outbound.sniList = []string{
			"www.google.com",
			"cdn.cloudflare.com",
			"ajax.googleapis.com",
			"www.youtube.com",
			"www.gstatic.com",
		}
	}
	if outbound.fragSize == 0 {
		outbound.fragSize = 2
	}
	if outbound.fragJitter == 0 {
		outbound.fragJitter = 25
	}

	// TLS settings (для WS через nginx)
	if options.TLS != nil && options.TLS.Enabled {
		outbound.tlsEnabled = true
		outbound.tlsSNI = options.TLS.ServerName
	}

	outboundDialer, err := dialer.New(router, options.DialerOptions)
	if err != nil {
		return nil, err
	}
	outbound.dialer = outboundDialer

	return outbound, nil
}

// DialContext — основной метод: подключение к SOVA серверу и relay
func (s *SOVA) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	s.logger.InfoContext(ctx, "SOVA connecting to ", destination)

	// 1. TCP подключение к серверу
	conn, err := s.dialer.DialContext(ctx, N.NetworkTCP, s.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("sova: TCP connect to %s failed: %v", s.serverAddr, err)
	}

	// 2. DPI evasion: фрагментация ClientHello
	var baseConn net.Conn = conn
	if s.fragSize > 0 {
		baseConn = newFragConn(conn, s.fragSize, s.fragJitter)
	}

	// 3. TLS с поддельным SNI
	sni := s.sniList[int(sovaSecureRandByte())%len(s.sniList)]
	tlsConn := tls.Client(baseConn, &tls.Config{
		ServerName: sni,
		// SOVA использует собственный handshake для аутентификации,
		// TLS сертификат самоподписанный — проверяется через SOVA PSK
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("sova: TLS handshake failed (SNI=%s): %v", sni, err)
	}

	// 4. SOVA protocol handshake (вывод сессионного ключа AES-256-GCM)
	sovaC, err := sovaClientHandshake(tlsConn, s.psk)
	if err != nil {
		tlsConn.Close()
		return nil, fmt.Errorf("sova: protocol handshake failed: %v", err)
	}

	// 5. Отправляем CONNECT фрейм с адресом назначения
	targetAddr := destination.String()
	if err := sovaC.writeFrame(sovaFrameConnect, []byte(targetAddr)); err != nil {
		sovaC.close()
		return nil, fmt.Errorf("sova: CONNECT frame failed: %v", err)
	}

	// 6. Читаем ACK
	ftype, payload, err := sovaC.readFrame()
	if err != nil {
		sovaC.close()
		return nil, fmt.Errorf("sova: no ACK from server: %v", err)
	}
	if ftype != sovaFrameAck || len(payload) == 0 || payload[0] != 0x00 {
		sovaC.close()
		return nil, fmt.Errorf("sova: server refused connection to %s", targetAddr)
	}

	s.logger.InfoContext(ctx, "SOVA tunnel established to ", destination)

	// 7. Возвращаем SOVAStream как net.Conn
	return newSovaStream(sovaC), nil
}

// ListenPacket — SOVA пока не поддерживает UDP
func (s *SOVA) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, fmt.Errorf("sova: UDP not supported yet")
}

// NewConnection — для sing-box NewConnection interface
func (s *SOVA) NewConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext) error {
	return NewConnection(ctx, s, conn, metadata)
}
