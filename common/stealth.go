package common

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mrand "math/rand"
	"net"
	"sync"
	"time"
)

// StealthEngine движок скрытности — делает трафик SOVA неотличимым от обычного
type StealthEngine struct {
	mu            sync.RWMutex
	profile       TrafficProfile
	mimicPatterns []MimicPattern
	jitterGen     *AdaptiveJitter
	padder        *IntelligentPadder
	fingerprint   *TLSFingerprint
	enabled       bool
}

// TrafficProfile профиль трафика для мимикрии
type TrafficProfile string

const (
	ProfileHTTPS    TrafficProfile = "https_browsing"
	ProfileVideo    TrafficProfile = "video_streaming"
	ProfileWebRTC   TrafficProfile = "webrtc_call"
	ProfileCloudAPI TrafficProfile = "cloud_api"
)

// MimicPattern паттерн мимикрии трафика
type MimicPattern struct {
	Name          string
	AvgPacketSize int
	PacketJitter  int
	IntervalMs    int
	IntervalDev   int
	BurstSize     int
	BurstProb     float64
}

// AdaptiveJitter адаптивный генератор задержек
type AdaptiveJitter struct {
	baseInterval time.Duration
	deviation    time.Duration
	burstMode    bool
	burstCount   int
}

// IntelligentPadder умное дополнение пакетов
type IntelligentPadder struct {
	targetSizes []int // типичные размеры HTTP-пакетов
}

// TLSFingerprint маскировка TLS отпечатка
type TLSFingerprint struct {
	ClientHellos [][]byte
	Extensions   []uint16
	CipherSuites []uint16
}

// NewStealthEngine создает движок скрытности
func NewStealthEngine() *StealthEngine {
	return &StealthEngine{
		profile: ProfileHTTPS,
		mimicPatterns: []MimicPattern{
			{
				Name:          "Chrome HTTPS Browsing",
				AvgPacketSize: 1380,
				PacketJitter:  200,
				IntervalMs:    50,
				IntervalDev:   30,
				BurstSize:     8,
				BurstProb:     0.15,
			},
			{
				Name:          "YouTube Video Stream",
				AvgPacketSize: 1460,
				PacketJitter:  40,
				IntervalMs:    10,
				IntervalDev:   5,
				BurstSize:     32,
				BurstProb:     0.8,
			},
			{
				Name:          "Cloud API Calls",
				AvgPacketSize: 512,
				PacketJitter:  300,
				IntervalMs:    200,
				IntervalDev:   150,
				BurstSize:     3,
				BurstProb:     0.05,
			},
		},
		jitterGen: &AdaptiveJitter{
			baseInterval: 50 * time.Millisecond,
			deviation:    20 * time.Millisecond,
		},
		padder: &IntelligentPadder{
			targetSizes: []int{
				64, 128, 256, 512, 576, 1024, 1380, 1460, 1500,
			},
		},
		fingerprint: &TLSFingerprint{
			CipherSuites: []uint16{
				0x1301, 0x1302, 0x1303, // TLS 1.3
				0xc02b, 0xc02f, 0xc02c, 0xc030, // ECDHE
				0xcca9, 0xcca8, // ChaCha20
			},
			Extensions: []uint16{
				0x0000, 0x0017, 0x0023, 0x000d, 0x0005,
				0x0012, 0x0010, 0xff01, 0x002b, 0x000a,
			},
		},
		enabled: true,
	}
}

// SetProfile устанавливает профиль мимикрии
func (se *StealthEngine) SetProfile(profile TrafficProfile) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.profile = profile
}

// PadPacket дополняет пакет до типичного размера HTTP
func (ip *IntelligentPadder) PadPacket(data []byte) []byte {
	dataLen := len(data)

	// Находим ближайший типичный размер, который >= dataLen + 4 (header)
	targetSize := 0
	for _, size := range ip.targetSizes {
		if size >= dataLen+4 {
			targetSize = size
			break
		}
	}
	if targetSize == 0 {
		// Больше максимального — округляем вверх до кратного MTU
		targetSize = ((dataLen + 4 + 1459) / 1460) * 1460
	}

	paddingLen := targetSize - dataLen - 4
	if paddingLen < 0 {
		paddingLen = 0
	}

	// [2 байта: длина данных][данные][рандомный padding][2 байта: длина padding]
	result := make([]byte, 0, targetSize)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(dataLen))
	result = append(result, lenBuf...)
	result = append(result, data...)

	if paddingLen > 2 {
		padding := make([]byte, paddingLen-2)
		rand.Read(padding)
		result = append(result, padding...)
	}

	padLenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(padLenBuf, uint16(paddingLen))
	result = append(result, padLenBuf...)

	return result
}

// UnpadPacket извлекает данные из дополненного пакета
func (ip *IntelligentPadder) UnpadPacket(packet []byte) ([]byte, error) {
	if len(packet) < 4 {
		return packet, nil
	}

	dataLen := int(binary.BigEndian.Uint16(packet[:2]))
	if dataLen+2 > len(packet) {
		return packet, nil
	}

	return packet[2 : 2+dataLen], nil
}

// NextDelay генерирует задержку, имитирующую реальный трафик
func (aj *AdaptiveJitter) NextDelay() time.Duration {
	// Нормальное распределение вокруг базового интервала
	u1 := mrand.Float64()
	u2 := mrand.Float64()
	// Box-Muller transform
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

	delay := float64(aj.baseInterval) + z*float64(aj.deviation)
	if delay < 0 {
		delay = float64(aj.baseInterval) * 0.1
	}

	// Иногда burst — без задержки
	if aj.burstMode && aj.burstCount > 0 {
		aj.burstCount--
		return time.Duration(mrand.Intn(2)) * time.Millisecond
	}

	// Случайный burst
	if mrand.Float64() < 0.1 {
		aj.burstMode = true
		aj.burstCount = 3 + mrand.Intn(8)
	}

	return time.Duration(delay)
}

// StealthWrite пишет данные с полной мимикрией
func (se *StealthEngine) StealthWrite(conn net.Conn, data []byte) error {
	if !se.enabled {
		_, err := conn.Write(data)
		return err
	}

	// 1. Дополняем до типичного размера
	padded := se.padder.PadPacket(data)

	// 2. Разбиваем на фрагменты, имитирующие выбранный профиль
	fragments := se.fragmentByProfile(padded)

	// 3. Отправляем с реалистичными задержками
	for _, frag := range fragments {
		delay := se.jitterGen.NextDelay()
		time.Sleep(delay)
		if _, err := conn.Write(frag); err != nil {
			return err
		}
	}

	return nil
}

// fragmentByProfile разбивает данные по профилю
func (se *StealthEngine) fragmentByProfile(data []byte) [][]byte {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var pattern MimicPattern
	switch se.profile {
	case ProfileVideo:
		pattern = se.mimicPatterns[1]
	case ProfileCloudAPI:
		pattern = se.mimicPatterns[2]
	default:
		pattern = se.mimicPatterns[0]
	}

	var fragments [][]byte
	for len(data) > 0 {
		size := pattern.AvgPacketSize + mrand.Intn(pattern.PacketJitter*2) - pattern.PacketJitter
		if size <= 0 {
			size = 64
		}
		if size > len(data) {
			size = len(data)
		}
		fragments = append(fragments, data[:size])
		data = data[size:]
	}
	return fragments
}

// GenerateDecoyTraffic генерирует фоновый трафик-обманку
func (se *StealthEngine) GenerateDecoyTraffic(conn net.Conn, stop chan struct{}) {
	ticker := time.NewTicker(time.Duration(500+mrand.Intn(2000)) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			// Генерируем пакет, похожий на HTTP keep-alive или heartbeat
			decoySize := 32 + mrand.Intn(128)
			decoy := make([]byte, decoySize)
			rand.Read(decoy)
			// Маркер decoy-пакета (первый байт 0xFE)
			decoy[0] = 0xFE
			conn.Write(decoy)
			ticker.Reset(time.Duration(500+mrand.Intn(2000)) * time.Millisecond)
		}
	}
}

// IsDecoyPacket проверяет, является ли пакет обманкой
func IsDecoyPacket(data []byte) bool {
	return len(data) > 0 && data[0] == 0xFE
}
