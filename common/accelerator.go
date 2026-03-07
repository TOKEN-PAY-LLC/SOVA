package common

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TrafficAccelerator ускоряет трафик через сжатие, пулинг и умную маршрутизацию
type TrafficAccelerator struct {
	mu              sync.RWMutex
	pool            *ConnectionPool
	compressor      *TrafficCompressor
	optimizer       *RouteOptimizer
	stats           AcceleratorStats
	enabled         bool
}

// AcceleratorStats статистика ускорения
type AcceleratorStats struct {
	BytesSaved      atomic.Int64
	BytesTotal      atomic.Int64
	CompressionRate float64
	AvgLatencyMs    float64
	ConnectionsPool int
	ActiveStreams    int
}

// ConnectionPool пул переиспользуемых соединений
type ConnectionPool struct {
	mu       sync.Mutex
	conns    map[string][]*PooledConn
	maxIdle  int
	maxTotal int
	idleTime time.Duration
}

// PooledConn соединение из пула
type PooledConn struct {
	Conn      net.Conn
	Target    string
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
}

// TrafficCompressor сжатие трафика
type TrafficCompressor struct {
	level     int
	minSize   int // минимальный размер для сжатия
	algorithm string
}

// RouteOptimizer оптимизатор маршрутов
type RouteOptimizer struct {
	mu     sync.RWMutex
	routes map[string]*RouteInfo
}

// RouteInfo информация о маршруте
type RouteInfo struct {
	Target      string
	AvgLatency  time.Duration
	PacketLoss  float64
	Bandwidth   float64 // bytes/sec
	LastChecked time.Time
	Score       float64
	Failures    int
}

// NewTrafficAccelerator создает акселератор
func NewTrafficAccelerator() *TrafficAccelerator {
	return &TrafficAccelerator{
		pool: &ConnectionPool{
			conns:    make(map[string][]*PooledConn),
			maxIdle:  32,
			maxTotal: 128,
			idleTime: 90 * time.Second,
		},
		compressor: &TrafficCompressor{
			level:     flate.BestSpeed,
			minSize:   256,
			algorithm: "gzip",
		},
		optimizer: &RouteOptimizer{
			routes: make(map[string]*RouteInfo),
		},
		enabled: true,
	}
}

// Compress сжимает данные для передачи
func (tc *TrafficCompressor) Compress(data []byte) ([]byte, error) {
	if len(data) < tc.minSize {
		// Маркер: не сжато (0x00) + оригинальные данные
		return append([]byte{0x00}, data...), nil
	}

	var buf bytes.Buffer
	buf.WriteByte(0x01) // Маркер: сжато

	w, err := gzip.NewWriterLevel(&buf, tc.level)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	compressed := buf.Bytes()
	// Используем сжатие только если оно выгодно
	if len(compressed) >= len(data)+1 {
		return append([]byte{0x00}, data...), nil
	}
	return compressed, nil
}

// Decompress восстанавливает данные
func (tc *TrafficCompressor) Decompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if data[0] == 0x00 {
		return data[1:], nil
	}

	r, err := gzip.NewReader(bytes.NewReader(data[1:]))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

// GetConn получает соединение из пула или создает новое
func (cp *ConnectionPool) GetConn(target string, dial func() (net.Conn, error)) (net.Conn, error) {
	cp.mu.Lock()

	// Ищем свободное соединение
	if conns, ok := cp.conns[target]; ok {
		for i, pc := range conns {
			if !pc.InUse && time.Since(pc.LastUsed) < cp.idleTime {
				pc.InUse = true
				pc.LastUsed = time.Now()
				cp.mu.Unlock()
				// Проверяем что соединение живо
				if err := pc.Conn.SetDeadline(time.Now().Add(100 * time.Millisecond)); err == nil {
					pc.Conn.SetDeadline(time.Time{})
					return pc.Conn, nil
				}
				// Мертвое соединение — удаляем
				pc.Conn.Close()
				cp.mu.Lock()
				cp.conns[target] = append(conns[:i], conns[i+1:]...)
				cp.mu.Unlock()
				break
			}
		}
	}
	cp.mu.Unlock()

	// Создаем новое
	conn, err := dial()
	if err != nil {
		return nil, err
	}

	cp.mu.Lock()
	cp.conns[target] = append(cp.conns[target], &PooledConn{
		Conn:      conn,
		Target:    target,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		InUse:     true,
	})
	cp.mu.Unlock()

	return conn, nil
}

// ReturnConn возвращает соединение в пул
func (cp *ConnectionPool) ReturnConn(target string, conn net.Conn) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if conns, ok := cp.conns[target]; ok {
		for _, pc := range conns {
			if pc.Conn == conn {
				pc.InUse = false
				pc.LastUsed = time.Now()
				return
			}
		}
	}
}

// CleanIdle удаляет протухшие соединения
func (cp *ConnectionPool) CleanIdle() int {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cleaned := 0
	for target, conns := range cp.conns {
		var active []*PooledConn
		for _, pc := range conns {
			if !pc.InUse && time.Since(pc.LastUsed) > cp.idleTime {
				pc.Conn.Close()
				cleaned++
			} else {
				active = append(active, pc)
			}
		}
		cp.conns[target] = active
	}
	return cleaned
}

// UpdateRoute обновляет статистику маршрута
func (ro *RouteOptimizer) UpdateRoute(target string, latency time.Duration, loss float64, bandwidth float64) {
	ro.mu.Lock()
	defer ro.mu.Unlock()

	info, ok := ro.routes[target]
	if !ok {
		info = &RouteInfo{Target: target}
		ro.routes[target] = info
	}

	// Экспоненциальное скользящее среднее
	alpha := 0.3
	info.AvgLatency = time.Duration(float64(info.AvgLatency)*(1-alpha) + float64(latency)*alpha)
	info.PacketLoss = info.PacketLoss*(1-alpha) + loss*alpha
	info.Bandwidth = info.Bandwidth*(1-alpha) + bandwidth*alpha
	info.LastChecked = time.Now()

	// Score: ниже = лучше (latency_ms * (1 + loss) / bandwidth_mbps)
	latMs := float64(info.AvgLatency.Milliseconds())
	bwMbps := info.Bandwidth / 1024 / 1024
	if bwMbps < 0.001 {
		bwMbps = 0.001
	}
	info.Score = latMs * (1 + info.PacketLoss*10) / bwMbps
}

// BestRoute возвращает лучший маршрут
func (ro *RouteOptimizer) BestRoute() *RouteInfo {
	ro.mu.RLock()
	defer ro.mu.RUnlock()

	var best *RouteInfo
	for _, info := range ro.routes {
		if best == nil || info.Score < best.Score {
			best = info
		}
	}
	return best
}

// AcceleratedWrite пишет данные со сжатием
func (ta *TrafficAccelerator) AcceleratedWrite(conn net.Conn, data []byte) (int, error) {
	ta.stats.BytesTotal.Add(int64(len(data)))

	compressed, err := ta.compressor.Compress(data)
	if err != nil {
		return conn.Write(data)
	}

	saved := int64(len(data)) - int64(len(compressed))
	if saved > 0 {
		ta.stats.BytesSaved.Add(saved)
	}

	// Записываем длину + данные
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(compressed)))
	if _, err := conn.Write(header); err != nil {
		return 0, err
	}
	return conn.Write(compressed)
}

// AcceleratedRead читает данные с распаковкой
func (ta *TrafficAccelerator) AcceleratedRead(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	size := binary.BigEndian.Uint32(header)
	if size > 16*1024*1024 { // 16MB max
		return nil, fmt.Errorf("packet too large: %d", size)
	}

	data := make([]byte, size)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, err
	}

	return ta.compressor.Decompress(data)
}

// GetStats возвращает статистику
func (ta *TrafficAccelerator) GetStats() map[string]interface{} {
	total := ta.stats.BytesTotal.Load()
	saved := ta.stats.BytesSaved.Load()
	ratio := float64(0)
	if total > 0 {
		ratio = float64(saved) / float64(total) * 100
	}

	return map[string]interface{}{
		"bytes_total":      total,
		"bytes_saved":      saved,
		"compression_pct":  fmt.Sprintf("%.1f%%", ratio),
		"pool_idle_conns":  ta.pool.maxIdle,
		"pool_max_conns":   ta.pool.maxTotal,
	}
}
