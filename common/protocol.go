package common

// ── SOVA Wire Protocol v1 ──────────────────────────────────────────────
//
// SOVA — полностью автономный зашифрованный протокол туннелирования.
//
// Архитектура:
//   [App] → SOVA Proxy (local) → SOVA Client ─→ [TLS + SOVA Protocol] ─→ SOVA Server → [Internet]
//
// Транспорт:
//   TLS 1.3 с поддельным SNI (выглядит как HTTPS для DPI)
//   + ClientHello фрагментация (обход DPI мобильных операторов)
//   + timing jitter между фрагментами
//   + random padding в каждом фрейме
//
// Формат фрейма на проводе:
//   [Nonce:12][Length:2][AES-256-GCM(PadLen:1 | Type:1 | Payload:N | Padding:P)]
//
// Типы фреймов:
//   0x01 CONNECT   — открыть соединение (payload = адрес:порт)
//   0x02 DATA      — данные
//   0x03 CLOSE     — закрыть соединение
//   0x04 KEEPALIVE — пинг
//   0x05 ACK       — подтверждение (payload[0]: 0=ok, 1=fail)
//
// Хендшейк (поверх TLS):
//   Client → Server: Magic(4) + Version(1) + ClientSalt(16) + Random(32)
//   Server → Client: ACK(1) + ServerSalt(16)
//   SessionKey = SHA-256(PSK ‖ ClientSalt ‖ ServerSalt)
//
// ────────────────────────────────────────────────────────────────────────

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Protocol constants
const (
	ProtoMagic   uint32 = 0x534F5641 // "SOVA" in ASCII hex
	ProtoVersion byte   = 0x01

	FrameConnect   byte = 0x01
	FrameData      byte = 0x02
	FrameClose     byte = 0x03
	FrameKeepalive byte = 0x04
	FrameAck       byte = 0x05

	MaxFramePayload = 60000
	HandshakeSize   = 4 + 1 + 16 + 32 // magic + version + clientSalt + random
	ServerAckSize   = 1 + 16          // ack + serverSalt

	DefaultPSK = "sova-protocol-v1-key-2026"
)

// ── Frame ───────────────────────────────────────────────────────────────

// Frame — единица данных SOVA протокола
type Frame struct {
	Type    byte
	Payload []byte
}

// ── SOVAConn — зашифрованное фреймовое соединение ──────────────────────

// SOVAConn оборачивает net.Conn в AES-256-GCM зашифрованный фреймовый протокол
type SOVAConn struct {
	conn    net.Conn
	aead    cipher.AEAD
	readMu  sync.Mutex
	writeMu sync.Mutex
	nonce   uint64
}

// DeriveSOVASessionKey — получение 256-бит ключа из PSK + client salt + server salt
func DeriveSOVASessionKey(psk string, clientSalt, serverSalt []byte) []byte {
	h := sha256.New()
	h.Write([]byte(psk))
	h.Write(clientSalt)
	h.Write(serverSalt)
	return h.Sum(nil)
}

// NewSOVAConn создаёт зашифрованное соединение поверх существующего
func NewSOVAConn(conn net.Conn, key []byte) (*SOVAConn, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sova: AES init: %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("sova: GCM init: %v", err)
	}
	return &SOVAConn{conn: conn, aead: aead}, nil
}

// WriteFrame шифрует и отправляет фрейм с рандомным padding
func (sc *SOVAConn) WriteFrame(f *Frame) error {
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()

	// Random padding 4–64 байт для обмана traffic analysis
	padLen := 4 + int(secureRandByte()%61)
	padding := make([]byte, padLen)
	rand.Read(padding)

	// Plaintext: [PadLen:1][Type:1][Payload:N][Padding:P]
	plain := make([]byte, 0, 2+len(f.Payload)+padLen)
	plain = append(plain, byte(padLen), f.Type)
	plain = append(plain, f.Payload...)
	plain = append(plain, padding...)

	// Nonce (12 байт, counter-based для каждого фрейма)
	nonce := make([]byte, sc.aead.NonceSize())
	nVal := atomic.AddUint64(&sc.nonce, 1)
	binary.BigEndian.PutUint64(nonce[4:], nVal)

	// AES-256-GCM encrypt
	sealed := sc.aead.Seal(nil, nonce, plain, nil)

	// Wire format: [Nonce:12][Length:2][Sealed:N]
	wire := make([]byte, len(nonce)+2+len(sealed))
	copy(wire, nonce)
	binary.BigEndian.PutUint16(wire[len(nonce):], uint16(len(sealed)))
	copy(wire[len(nonce)+2:], sealed)

	_, err := sc.conn.Write(wire)
	return err
}

// ReadFrame читает и расшифровывает фрейм
func (sc *SOVAConn) ReadFrame() (*Frame, error) {
	sc.readMu.Lock()
	defer sc.readMu.Unlock()

	// Nonce
	nonce := make([]byte, sc.aead.NonceSize())
	if _, err := io.ReadFull(sc.conn, nonce); err != nil {
		return nil, err
	}

	// Length
	lb := make([]byte, 2)
	if _, err := io.ReadFull(sc.conn, lb); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(lb))
	if n > MaxFramePayload+200 {
		return nil, errors.New("sova: frame too large")
	}

	// Ciphertext
	ct := make([]byte, n)
	if _, err := io.ReadFull(sc.conn, ct); err != nil {
		return nil, err
	}

	// Decrypt
	plain, err := sc.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("sova: decrypt: %v", err)
	}
	if len(plain) < 2 {
		return nil, errors.New("sova: frame too small")
	}

	padLen := int(plain[0])
	ftype := plain[1]
	data := plain[2:]
	if padLen > len(data) {
		return nil, errors.New("sova: bad padding")
	}
	payload := data[:len(data)-padLen]

	return &Frame{Type: ftype, Payload: append([]byte{}, payload...)}, nil
}

func (sc *SOVAConn) Close() error                       { return sc.conn.Close() }
func (sc *SOVAConn) SetDeadline(t time.Time) error      { return sc.conn.SetDeadline(t) }
func (sc *SOVAConn) SetReadDeadline(t time.Time) error  { return sc.conn.SetReadDeadline(t) }
func (sc *SOVAConn) SetWriteDeadline(t time.Time) error { return sc.conn.SetWriteDeadline(t) }
func (sc *SOVAConn) RemoteAddr() net.Addr               { return sc.conn.RemoteAddr() }
func (sc *SOVAConn) LocalAddr() net.Addr                { return sc.conn.LocalAddr() }

func secureRandByte() byte {
	b := make([]byte, 1)
	rand.Read(b)
	return b[0]
}

// ── Handshake ───────────────────────────────────────────────────────────

// ClientHandshake выполняет клиентскую часть SOVA-хендшейка поверх TLS.
// Обе стороны получают общий сессионный ключ для AES-256-GCM.
func ClientHandshake(conn net.Conn, psk string) (*SOVAConn, error) {
	clientSalt := make([]byte, 16)
	randomPad := make([]byte, 32)
	rand.Read(clientSalt)
	rand.Read(randomPad)

	// Send: Magic(4) + Version(1) + ClientSalt(16) + Random(32)
	hs := make([]byte, HandshakeSize)
	binary.BigEndian.PutUint32(hs[0:4], ProtoMagic)
	hs[4] = ProtoVersion
	copy(hs[5:21], clientSalt)
	copy(hs[21:53], randomPad)

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write(hs); err != nil {
		return nil, fmt.Errorf("sova handshake: write: %v", err)
	}

	// Read: ACK(1) + ServerSalt(16)
	resp := make([]byte, ServerAckSize)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, fmt.Errorf("sova handshake: server response: %v", err)
	}
	conn.SetDeadline(time.Time{})

	if resp[0] != 0x00 {
		return nil, errors.New("sova handshake: server rejected")
	}
	serverSalt := resp[1:17]

	key := DeriveSOVASessionKey(psk, clientSalt, serverSalt)
	return NewSOVAConn(conn, key)
}

// ServerHandshake выполняет серверную часть SOVA-хендшейка.
func ServerHandshake(conn net.Conn, psk string) (*SOVAConn, error) {
	hs := make([]byte, HandshakeSize)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.ReadFull(conn, hs); err != nil {
		return nil, fmt.Errorf("sova handshake: read: %v", err)
	}

	magic := binary.BigEndian.Uint32(hs[0:4])
	if magic != ProtoMagic {
		conn.Write([]byte{0x01})
		return nil, errors.New("sova handshake: bad magic")
	}
	if hs[4] != ProtoVersion {
		conn.Write([]byte{0x01})
		return nil, fmt.Errorf("sova handshake: unsupported version %d", hs[4])
	}

	clientSalt := hs[5:21]

	serverSalt := make([]byte, 16)
	rand.Read(serverSalt)

	// ACK(1) + ServerSalt(16)
	resp := make([]byte, ServerAckSize)
	resp[0] = 0x00
	copy(resp[1:], serverSalt)

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write(resp); err != nil {
		return nil, fmt.Errorf("sova handshake: ACK write: %v", err)
	}
	conn.SetDeadline(time.Time{})

	key := DeriveSOVASessionKey(psk, clientSalt, serverSalt)
	return NewSOVAConn(conn, key)
}

// ── SOVAStream: net.Conn-совместимая обёртка над SOVAConn ──────────────
// Мост между фреймовым протоколом и byte-stream интерфейсом,
// который ожидают SOVA тоннель и стандартный Go I/O.

type SOVAStream struct {
	sc      *SOVAConn
	readBuf []byte
	closed  bool
	mu      sync.Mutex
}

// NewSOVAStream оборачивает SOVAConn для потокового I/O
func NewSOVAStream(sc *SOVAConn) *SOVAStream {
	return &SOVAStream{sc: sc}
}

func (s *SOVAStream) Read(p []byte) (int, error) {
	if len(s.readBuf) > 0 {
		n := copy(p, s.readBuf)
		s.readBuf = s.readBuf[n:]
		return n, nil
	}
	for {
		frame, err := s.sc.ReadFrame()
		if err != nil {
			return 0, err
		}
		switch frame.Type {
		case FrameData:
			n := copy(p, frame.Payload)
			if n < len(frame.Payload) {
				s.readBuf = frame.Payload[n:]
			}
			return n, nil
		case FrameClose:
			return 0, io.EOF
		case FrameKeepalive:
			continue
		default:
			continue
		}
	}
}

func (s *SOVAStream) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		chunk := p
		if len(chunk) > MaxFramePayload {
			chunk = p[:MaxFramePayload]
		}
		if err := s.sc.WriteFrame(&Frame{Type: FrameData, Payload: chunk}); err != nil {
			return total, err
		}
		total += len(chunk)
		p = p[len(chunk):]
	}
	return total, nil
}

func (s *SOVAStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.sc.WriteFrame(&Frame{Type: FrameClose})
	return s.sc.Close()
}

func (s *SOVAStream) LocalAddr() net.Addr                { return s.sc.LocalAddr() }
func (s *SOVAStream) RemoteAddr() net.Addr               { return s.sc.RemoteAddr() }
func (s *SOVAStream) SetDeadline(t time.Time) error      { return s.sc.SetDeadline(t) }
func (s *SOVAStream) SetReadDeadline(t time.Time) error  { return s.sc.SetReadDeadline(t) }
func (s *SOVAStream) SetWriteDeadline(t time.Time) error { return s.sc.SetWriteDeadline(t) }
