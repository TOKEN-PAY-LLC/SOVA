package common

// ── SOVA Core — Stream Multiplexer ────────────────────────────────────
//
// Мультиплексирование потоков поверх одного SOVA соединения.
// Позволяет открывать несколько логических потоков (streams)
// через одно TCP/TLS соединение — как HTTP/2 мультиплексирование.
//
// Преимущества:
//   - Один handshake на несколько соединений
//   - Меньше TCP соединений = меньше подозрений DPI
//   - Head-of-line blocking устранён через stream isolation
//   - Connection pooling встроен
//
// Протокол (поверх SOVA Protocol v2):
//   MUX_OPEN:  [StreamID:4][0x06][TargetLen:2][Target:N]
//   MUX_DATA:  [StreamID:4][0x08][Data:N]
//   MUX_CLOSE: [StreamID:4][0x07]
//   MUX_WIN:   [StreamID:4][0x09][WindowUpdate:4]
//
// ────────────────────────────────────────────────────────────────────────

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ── MuxSession ─────────────────────────────────────────────────────────

// MuxSession управляет мультиплексированными потоками поверх SOVAV2Conn
type MuxSession struct {
	conn      *SOVAV2Conn
	streams   map[uint32]*MuxStream
	nextID    uint32
	mu        sync.RWMutex
	closed    bool
	onClose   func()
}

// NewMuxSession создаёт новую мультиплексную сессию
func NewMuxSession(conn *SOVAV2Conn) *MuxSession {
	return &MuxSession{
		conn:    conn,
		streams: make(map[uint32]*MuxStream),
		nextID:  1, // 0 = session control
	}
}

// OpenStream открывает новый мультиплексированный поток
func (ms *MuxSession) OpenStream(target string) (*MuxStream, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return nil, errors.New("mux: session closed")
	}

	streamID := atomic.AddUint32(&ms.nextID, 2) // Клиентские ID нечётные
	stream := &MuxStream{
		id:        streamID,
		session:   ms,
		target:    target,
		readChan:  make(chan []byte, 64),
		closeChan: make(chan struct{}),
		window:    256 * 1024, // 256KB initial window
	}

	ms.streams[streamID] = stream

	// Отправляем MUX_OPEN
	if err := ms.conn.WriteFrameV2(streamID, FrameV2MuxOpen, []byte(target)); err != nil {
		delete(ms.streams, streamID)
		return nil, fmt.Errorf("mux: open stream: %v", err)
	}

	return stream, nil
}

// AcceptStream принимает входящий поток (серверная сторона)
func (ms *MuxSession) AcceptStream() (*MuxStream, error) {
	for {
		sid, ftype, payload, err := ms.conn.ReadFrameV2()
		if err != nil {
			return nil, err
		}

		switch ftype {
		case FrameV2MuxOpen:
			stream := &MuxStream{
				id:        sid,
				session:   ms,
				target:    string(payload),
				readChan:  make(chan []byte, 64),
				closeChan: make(chan struct{}),
				window:    256 * 1024,
			}
			ms.mu.Lock()
			ms.streams[sid] = stream
			ms.mu.Unlock()
			return stream, nil

		case FrameV2MuxData:
			ms.mu.RLock()
			stream, ok := ms.streams[sid]
			ms.mu.RUnlock()
			if ok && stream != nil {
				select {
				case stream.readChan <- payload:
				default:
					// Buffer full — drop (flow control)
				}
			}

		case FrameV2MuxClose:
			ms.mu.Lock()
			stream, ok := ms.streams[sid]
			if ok {
				close(stream.closeChan)
				delete(ms.streams, sid)
			}
			ms.mu.Unlock()

		case FrameV2MuxWin:
			// Window update
			ms.mu.RLock()
			stream, ok := ms.streams[sid]
			ms.mu.RUnlock()
			if ok && len(payload) == 4 {
				delta := uint32(payload[0])<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])
				atomic.AddUint32(&stream.window, delta)
			}

		case FrameV2Keepalive, FrameV2Padding:
			continue

		case FrameV2Close:
			ms.Close()
			return nil, io.EOF
		}
	}
}

// ReadLoop запускает цикл чтения (клиентская сторона)
func (ms *MuxSession) ReadLoop() {
	for {
		sid, ftype, payload, err := ms.conn.ReadFrameV2()
		if err != nil {
			ms.Close()
			return
		}

		switch ftype {
		case FrameV2MuxData:
			ms.mu.RLock()
			stream, ok := ms.streams[sid]
			ms.mu.RUnlock()
			if ok && stream != nil {
				select {
				case stream.readChan <- payload:
				default:
				}
			}

		case FrameV2MuxClose:
			ms.mu.Lock()
			stream, ok := ms.streams[sid]
			if ok {
				close(stream.closeChan)
				delete(ms.streams, sid)
			}
			ms.mu.Unlock()

		case FrameV2MuxWin:
			ms.mu.RLock()
			stream, ok := ms.streams[sid]
			ms.mu.RUnlock()
			if ok && len(payload) == 4 {
				delta := uint32(payload[0])<<24 | uint32(payload[1])<<16 | uint32(payload[2])<<8 | uint32(payload[3])
				atomic.AddUint32(&stream.window, delta)
			}

		case FrameV2Close:
			ms.Close()
			return

		case FrameV2Keepalive, FrameV2Padding:
			continue
		}
	}
}

// Close закрывает сессию
func (ms *MuxSession) Close() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return
	}
	ms.closed = true

	for _, stream := range ms.streams {
		close(stream.closeChan)
	}
	ms.streams = make(map[uint32]*MuxStream)

	ms.conn.WriteFrameV2(0, FrameV2Close, nil)
	ms.conn.Close()

	if ms.onClose != nil {
		ms.onClose()
	}
}

// GetStreamCount возвращает количество активных потоков
func (ms *MuxSession) GetStreamCount() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.streams)
}

// ── MuxStream ──────────────────────────────────────────────────────────

// MuxStream — один мультиплексированный поток
type MuxStream struct {
	id        uint32
	session   *MuxSession
	target    string
	readChan  chan []byte
	closeChan chan struct{}
	readBuf   []byte
	window    uint32
	closed    bool
	mu        sync.Mutex
}

// Read читает данные из потока
func (ms *MuxStream) Read(p []byte) (int, error) {
	// Сначала из буфера
	if len(ms.readBuf) > 0 {
		n := copy(p, ms.readBuf)
		ms.readBuf = ms.readBuf[n:]
		return n, nil
	}

	select {
	case data, ok := <-ms.readChan:
		if !ok {
			return 0, io.EOF
		}
		n := copy(p, data)
		if n < len(data) {
			ms.readBuf = data[n:]
		}
		// Window update
		ms.sendWindowUpdate(uint32(len(data)))
		return n, nil
	case <-ms.closeChan:
		return 0, io.EOF
	}
}

// Write записывает данные в поток
func (ms *MuxStream) Write(p []byte) (int, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return 0, errors.New("mux: stream closed")
	}

	// Разбиваем на чанки
	total := 0
	for len(p) > 0 {
		chunk := p
		if len(chunk) > MaxFrameV2Payload {
			chunk = p[:MaxFrameV2Payload]
		}
		if err := ms.session.conn.WriteFrameV2(ms.id, FrameV2MuxData, chunk); err != nil {
			return total, err
		}
		total += len(chunk)
		p = p[len(chunk):]
	}
	return total, nil
}

// Close закрывает поток
func (ms *MuxStream) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return nil
	}
	ms.closed = true

	ms.session.conn.WriteFrameV2(ms.id, FrameV2MuxClose, nil)

	ms.session.mu.Lock()
	delete(ms.session.streams, ms.id)
	ms.session.mu.Unlock()

	return nil
}

// sendWindowUpdate отправляет window update
func (ms *MuxStream) sendWindowUpdate(delta uint32) {
	payload := make([]byte, 4)
	payload[0] = byte(delta >> 24)
	payload[1] = byte(delta >> 16)
	payload[2] = byte(delta >> 8)
	payload[3] = byte(delta)
	ms.session.conn.WriteFrameV2(ms.id, FrameV2MuxWin, payload)
}

// net.Conn compatibility
func (ms *MuxStream) LocalAddr() net.Addr                { return ms.session.conn.LocalAddr() }
func (ms *MuxStream) RemoteAddr() net.Addr               { return ms.session.conn.RemoteAddr() }
func (ms *MuxStream) SetDeadline(t time.Time) error      { return nil }
func (ms *MuxStream) SetReadDeadline(t time.Time) error  { return nil }
func (ms *MuxStream) SetWriteDeadline(t time.Time) error { return nil }

// ID возвращает идентификатор потока
func (ms *MuxStream) ID() uint32 { return ms.id }

// Target возвращает целевой адрес потока
func (ms *MuxStream) Target() string { return ms.target }
