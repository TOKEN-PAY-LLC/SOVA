package common

// ── SOVA Protocol v2 — Post-Quantum Native Core ────────────────────────
//
// SOVA Core v2 — полностью автономный зашифрованный протокол туннелирования
// с интегрированной пост-квантовой криптографией и мультиплексированием.
//
// Ключевые отличия от v1:
//   - X25519 + Kyber1024 гибридный KEM (как в Chrome/Cloudflare)
//   - Dilithium5 подпись сервера (вместо PSK-only)
//   - ChaCha20-Poly1305 как основной AEAD (быстрее на мобильных)
//   - HKDF-based key derivation (вместо простого SHA-256)
//   - Мультиплексирование потоков (mux frames)
//   - Domain fronting / ECH support
//   - Ротация ключей сессии (key rotation)
//
// Архитектура:
//   [App] → SOVA Core (Inbound) → Router → Outbound → [SOVA Server] → [Internet]
//
// Handshake v2:
//   Client → Server:
//     Magic(4) + Version(1) + ClientRandom(32) +
//     X25519_PubKey(32) + Kyber1024_CT(1568)
//   Server → Client:
//     ACK(1) + ServerRandom(32) + X25519_PubKey(32) +
//     Dilithium5_Signature(2420)
//
//   SharedSecret = X25519(shared) || Kyber1024(shared)
//   SessionKey = HKDF-SHA256(SharedSecret, ClientRandom || ServerRandom, "sova-v2")
//
// Формат фрейма v2:
//   [StreamID:4][Type:1][Length:3][AEAD(Nonce:12 | Payload:N | Tag:16 | Padding:P)]
//
// Типы фреймов v2:
//   0x01 CONNECT   — открыть поток (payload = адрес:порт)
//   0x02 DATA      — данные потока
//   0x03 CLOSE     — закрыть поток
//   0x04 KEEPALIVE — пинг
//   0x05 ACK       — подтверждение
//   0x06 MUX_OPEN  — открыть мультиплексированный поток
//   0x07 MUX_CLOSE — закрыть мультиплексированный поток
//   0x08 MUX_DATA  — данные мультиплексированного потока
//   0x09 MUX_WIN   — window update
//   0x0A KEY_ROT   — ротация ключей
//   0x0B PADDING   — decoy/padding фрейм
//
// ────────────────────────────────────────────────────────────────────────

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// Protocol v2 constants
const (
	ProtoV2Magic   uint32 = 0x534F5632 // "SOV2"
	ProtoV2Version byte   = 0x02

	// Frame types v2
	FrameV2Connect   byte = 0x01
	FrameV2Data      byte = 0x02
	FrameV2Close     byte = 0x03
	FrameV2Keepalive byte = 0x04
	FrameV2Ack       byte = 0x05
	FrameV2MuxOpen   byte = 0x06
	FrameV2MuxClose  byte = 0x07
	FrameV2MuxData   byte = 0x08
	FrameV2MuxWin    byte = 0x09
	FrameV2KeyRot    byte = 0x0A
	FrameV2Padding   byte = 0x0B

	// Sizes
	MaxFrameV2Payload = 65535
	KeyRotationBytes  = 1 << 30 // Ротация ключей каждые 1GB
	AEADNonceSize     = 12
	AEADTagSize       = 16

	// Handshake v2 sizes
	ClientHSV2Size = 4 + 1 + 32 + 32 + 1568 // magic + ver + random + x25519pub + kyberCT
	ServerHSV2Size = 1 + 32 + 32 + 2420     // ack + random + x25519pub + dilithiumSig

	// Default stream ID for non-multiplexed connections
	DefaultStreamID = 1
)

// ── SOVAV2Conn — зашифрованное мультиплексное соединение v2 ────────────

type SOVAV2Conn struct {
	conn       net.Conn
	aead       cipher.AEAD
	readMu     sync.Mutex
	writeMu    sync.Mutex
	nonce      uint64
	bytesSent  uint64
	rotCounter uint64
	key        []byte
	useChaCha  bool
}

// NewSOVAV2Conn создаёт v2 соединение с выбранным AEAD
func NewSOVAV2Conn(conn net.Conn, key []byte, useChaCha bool) (*SOVAV2Conn, error) {
	var aead cipher.AEAD
	var err error

	if useChaCha && len(key) == chacha20poly1305.KeySize {
		aead, err = chacha20poly1305.NewX(key)
	} else if len(key) == 32 {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("sova-v2: AES init: %v", err)
		}
		aead, err = cipher.NewGCM(block)
	} else {
		return nil, fmt.Errorf("sova-v2: invalid key size %d", len(key))
	}

	if err != nil {
		return nil, fmt.Errorf("sova-v2: AEAD init: %v", err)
	}

	return &SOVAV2Conn{
		conn:      conn,
		aead:      aead,
		key:       key,
		useChaCha: useChaCha,
	}, nil
}

// WriteFrameV2 шифрует и отправляет v2 фрейм с StreamID
func (sc *SOVAV2Conn) WriteFrameV2(streamID uint32, ftype byte, payload []byte) error {
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()

	// Random padding 4–64 байт для обмана traffic analysis
	padLen := 4 + int(secureRandByte()%61)
	padding := make([]byte, padLen)
	rand.Read(padding)

	// Plaintext: [StreamID:4][Type:1][PadLen:1][Payload:N][Padding:P]
	plain := make([]byte, 0, 6+len(payload)+padLen)
	plain = append(plain, byte(streamID>>24), byte(streamID>>16), byte(streamID>>8), byte(streamID))
	plain = append(plain, ftype, byte(padLen))
	plain = append(plain, payload...)
	plain = append(plain, padding...)

	// Nonce (12 байт, counter-based)
	nonce := make([]byte, sc.aead.NonceSize())
	nVal := atomic.AddUint64(&sc.nonce, 1)
	binary.BigEndian.PutUint64(nonce[4:], nVal)

	// AEAD encrypt
	sealed := sc.aead.Seal(nil, nonce, plain, nil)

	// Wire: [Nonce:12][Length:3][Sealed:N]
	wire := make([]byte, AEADNonceSize+3+len(sealed))
	copy(wire, nonce)
	wire[AEADNonceSize] = byte(len(sealed) >> 16)
	wire[AEADNonceSize+1] = byte(len(sealed) >> 8)
	wire[AEADNonceSize+2] = byte(len(sealed))
	copy(wire[AEADNonceSize+3:], sealed)

	_, err := sc.conn.Write(wire)
	if err == nil {
		atomic.AddUint64(&sc.bytesSent, uint64(len(wire)))
	}
	return err
}

// ReadFrameV2 читает и расшифровывает v2 фрейм
func (sc *SOVAV2Conn) ReadFrameV2() (streamID uint32, ftype byte, payload []byte, err error) {
	sc.readMu.Lock()
	defer sc.readMu.Unlock()

	// Nonce
	nonce := make([]byte, sc.aead.NonceSize())
	if _, err = io.ReadFull(sc.conn, nonce); err != nil {
		return 0, 0, nil, err
	}

	// 3-byte length
	lb := make([]byte, 3)
	if _, err = io.ReadFull(sc.conn, lb); err != nil {
		return 0, 0, nil, err
	}
	n := int(lb[0])<<16 | int(lb[1])<<8 | int(lb[2])
	if n > MaxFrameV2Payload+200+AEADTagSize {
		return 0, 0, nil, errors.New("sova-v2: frame too large")
	}

	// Ciphertext
	ct := make([]byte, n)
	if _, err = io.ReadFull(sc.conn, ct); err != nil {
		return 0, 0, nil, err
	}

	// Decrypt
	plain, err := sc.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("sova-v2: decrypt: %v", err)
	}
	if len(plain) < 6 {
		return 0, 0, nil, errors.New("sova-v2: frame too small")
	}

	streamID = binary.BigEndian.Uint32(plain[0:4])
	ftype = plain[4]
	padLen := int(plain[5])
	data := plain[6:]
	if padLen > len(data) {
		return 0, 0, nil, errors.New("sova-v2: bad padding")
	}
	payload = data[:len(data)-padLen]

	return streamID, ftype, append([]byte{}, payload...), nil
}

func (sc *SOVAV2Conn) Close() error                       { return sc.conn.Close() }
func (sc *SOVAV2Conn) SetDeadline(t time.Time) error      { return sc.conn.SetDeadline(t) }
func (sc *SOVAV2Conn) SetReadDeadline(t time.Time) error  { return sc.conn.SetReadDeadline(t) }
func (sc *SOVAV2Conn) SetWriteDeadline(t time.Time) error { return sc.conn.SetWriteDeadline(t) }
func (sc *SOVAV2Conn) RemoteAddr() net.Addr               { return sc.conn.RemoteAddr() }
func (sc *SOVAV2Conn) LocalAddr() net.Addr                { return sc.conn.LocalAddr() }
