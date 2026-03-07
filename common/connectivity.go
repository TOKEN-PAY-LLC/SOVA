package common

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"
)

// MeshNode представляет узел в mesh-сети SOVA
type MeshNode struct {
	ID            string        // Уникальный ID узла
	Address       string        // IP:port адрес
	PublicKey     []byte        // Публичный ключ для шифрования
	SignalStrength int          // Сила сигнала (0-100)
	LastSeen      time.Time     // Последний контакт
	Distance      int           // Хопы до узла
	IsReachable   bool          // Доступен ли узел
	Capabilities  []string      // ["gateway", "relay", "client"]
}

// ConnectivityDetector обнаруживает доступные каналы связи
type ConnectivityDetector struct {
	mu               sync.RWMutex
	meshNodes        map[string]*MeshNode
	cellularTowers   []*CellularTower
	internetGateway  *InternetGateway
	localNetworks    []*LocalNetwork
	isOnline         bool
	offlineMode      bool
	lastCheckTime    time.Time
	checkInterval    time.Duration
}

// CellularTower представляет сотовую вышку
type CellularTower struct {
	CellID        string
	Operator      string // MTK, Beeline, MegaFon, Rostelecom и т.д.
	Technology    string // 5G, 4G (LTE), 3G, 2G
	SignalStrength int    // -140 to -44 (dBm)
	LAC           int     // Location Area Code
	Latitude      float64
	Longitude     float64
	Distance      float64
}

// InternetGateway представляет точку выхода в интернет
type InternetGateway struct {
	Type           string // "direct", "proxy", "mesh", "cellular"
	Address        string
	Latency        time.Duration
	Bandwidth      int64 // в бит/сек
	ReliabilityScore float64 // 0-1
	IsActive       bool
}

// LocalNetwork представляет локальную сеть (Wi-Fi, Bluetooth, NFC)
type LocalNetwork struct {
	Type         string // "wifi", "bluetooth", "nfc", "zigbee"
	SSID         string // для Wi-Fi
	Signal       int
	IsConnected  bool
	Speed        int64
	Range        int // дальность в метрах
}

// NewConnectivityDetector создает детектор связи
func NewConnectivityDetector() *ConnectivityDetector {
	return &ConnectivityDetector{
		meshNodes:     make(map[string]*MeshNode),
		cellularTowers: make([]*CellularTower, 0),
		localNetworks: make([]*LocalNetwork, 0),
		checkInterval: 5 * time.Second,
		isOnline:      true,
		offlineMode:   false,
	}
}

// StartMonitoring запускает监视связи
func (cd *ConnectivityDetector) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(cd.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cd.detectAllChannels()
			cd.updateConnectivityStatus()
		}
	}
}

// detectAllChannels обнаруживает все доступные каналы связи
func (cd *ConnectivityDetector) detectAllChannels() {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	// Проверка интернета
	cd.detectInternet()

	// Сканирование Wi-Fi сетей
	cd.scanLocalNetworks()

	// Обнаружение mesh-узлов
	cd.discoverMeshNodes()

	// Сканирование сотовых вышек
	cd.scanCellularTowers()
}

// detectInternet проверяет подключение к интернету
func (cd *ConnectivityDetector) detectInternet() {
	// Пытаемся подключиться к нескольким известным серверам
	testServers := []string{
		"8.8.8.8:53",      // Google DNS
		"1.1.1.1:53",      // Cloudflare DNS
		"9.9.9.9:53",      // Quad9 DNS
		"208.67.222.222:53", // OpenDNS
	}

	for _, server := range testServers {
		conn, err := net.DialTimeout("udp", server, 2*time.Second)
		if err == nil {
			conn.Close()
			cd.isOnline = true
			cd.offlineMode = false
			return
		}
	}

	cd.isOnline = false
	cd.offlineMode = true
}

// scanLocalNetworks сканирует доступные локальные сети
func (cd *ConnectivityDetector) scanLocalNetworks() {
	// Получаем список всех интерфейсов
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error scanning interfaces: %v", err)
		return
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for range addrs {
			// Добавляем Wi-Fi сеть
			network := &LocalNetwork{
				Type:        "wifi",
				SSID:        iface.Name,
				Signal:      rand.Intn(100), // Имитируем сигнал
				IsConnected: true,
				Speed:       150_000_000, // 150 Mbps
				Range:       100,
			}
			cd.localNetworks = append(cd.localNetworks, network)
		}
	}
}

// discoverMeshNodes обнаруживает соседние mesh-узлы
func (cd *ConnectivityDetector) discoverMeshNodes() {
	// Мультикаст запрос для обнаружения узлов
	addr, err := net.ResolveUDPAddr("udp", "224.0.0.1:9999")
	if err != nil {
		return
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	buffer := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return
	}

	var nodeInfo struct {
		NodeID       string
		PublicKey    []byte
		Capabilities []string
	}

	if err := json.Unmarshal(buffer[:n], &nodeInfo); err == nil {
		distance := cd.calculateHopDistance(remoteAddr.IP)
		node := &MeshNode{
			ID:             nodeInfo.NodeID,
			Address:        remoteAddr.String(),
			PublicKey:      nodeInfo.PublicKey,
			SignalStrength: 80 + rand.Intn(20),
			LastSeen:       time.Now(),
			Distance:       distance,
			IsReachable:    true,
			Capabilities:   nodeInfo.Capabilities,
		}
		cd.meshNodes[nodeInfo.NodeID] = node
	}
}

// scanCellularTowers обнаруживает доступные вышки
func (cd *ConnectivityDetector) scanCellularTowers() {
	// Имитируем сканирование GSM/LTE вышек
	// В реальности используется Radio Resource Control (RRC) запросы
	operators := []string{"МТК", "Beeline", "MegaFon", "Rostelecom"}
	technologies := []string{"5G", "4G", "3G", "2G"}

	for i := 0; i < rand.Intn(5)+1; i++ {
		tower := &CellularTower{
			CellID:         fmt.Sprintf("CELL_%d", rand.Int63()),
			Operator:       operators[rand.Intn(len(operators))],
			Technology:     technologies[rand.Intn(len(technologies))],
			SignalStrength: -140 + rand.Intn(100),
			LAC:            rand.Intn(65535),
			Latitude:       55.7558 + (rand.Float64()-0.5)*0.1, // Москва
			Longitude:      37.6173 + (rand.Float64()-0.5)*0.1,
			Distance:       float64(rand.Intn(5000) + 500),
		}
		cd.cellularTowers = append(cd.cellularTowers, tower)
	}

	// Сортируем по сигналу
	sort.Slice(cd.cellularTowers, func(i, j int) bool {
		return cd.cellularTowers[i].SignalStrength > cd.cellularTowers[j].SignalStrength
	})
}

// calculateHopDistance расчитывает расстояние между узлами
func (cd *ConnectivityDetector) calculateHopDistance(ip net.IP) int {
	ttl := 64
	testConn, err := net.Dial("udp", ip.String()+":53")
	if err != nil {
		return 255
	}
	defer testConn.Close()

	// Простой расчет на основе IP
	distance := ttl - int(ip[3])
	if distance < 1 {
		distance = 1
	}
	return distance
}

// updateConnectivityStatus обновляет статус подключения
func (cd *ConnectivityDetector) updateConnectivityStatus() {
	cd.lastCheckTime = time.Now()

	// Если есть интернет - используем его
	if cd.isOnline && len(cd.cellularTowers) > 0 {
		cd.offlineMode = false
		return
	}

	// Если нет интернета но есть mesh-узлы с выходом - используем их
	for _, node := range cd.meshNodes {
		if node.IsReachable && len(node.Capabilities) > 0 {
			if contains(node.Capabilities, "gateway") {
				cd.offlineMode = false
				return
			}
		}
	}

	// Переходим в офлайн-режим
	if len(cd.meshNodes) > 0 || len(cd.cellularTowers) > 0 {
		cd.offlineMode = true
	}
}

// GetBestRoute выбирает лучший маршрут подключения
func (cd *ConnectivityDetector) GetBestRoute() *RoutingDecision {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	decision := &RoutingDecision{
		Timestamp: time.Now(),
		Routes:    make([]*Route, 0),
	}

	// Маршрут 1: Прямой интернет
	if cd.isOnline {
		decision.Routes = append(decision.Routes, &Route{
			Type:       "internet",
			Priority:   100,
			Reliability: 0.95,
		})
	}

	// Маршрут 2: Сотовая сеть
	if len(cd.cellularTowers) > 0 {
		bestTower := cd.cellularTowers[0]
		decision.Routes = append(decision.Routes, &Route{
			Type:       "cellular",
			Priority:   80,
			Reliability: cd.calculateCellularReliability(bestTower),
			Details:    fmt.Sprintf("%s (%s)", bestTower.Operator, bestTower.Technology),
		})
	}

	// Маршрут 3: Mesh-сеть
	for _, node := range cd.meshNodes {
		if node.IsReachable {
			decision.Routes = append(decision.Routes, &Route{
				Type:       "mesh",
				Priority:   60 - node.Distance*5,
				Reliability: float64(node.SignalStrength) / 100,
				Details:    fmt.Sprintf("Node %s (%d hops)", node.ID, node.Distance),
			})
		}
	}

	// Маршрут 4: Локальная сеть
	if len(cd.localNetworks) > 0 {
		for _, net := range cd.localNetworks {
			if net.IsConnected {
				decision.Routes = append(decision.Routes, &Route{
					Type:       "local",
					Priority:   40,
					Reliability: float64(net.Signal) / 100,
					Details:    fmt.Sprintf("%s (%s)", net.Type, net.SSID),
				})
			}
		}
	}

	// Сортируем по приоритету
	sort.Slice(decision.Routes, func(i, j int) bool {
		return decision.Routes[i].Priority > decision.Routes[j].Priority
	})

	if len(decision.Routes) > 0 {
		decision.BestRoute = decision.Routes[0]
	}

	return decision
}

// GetMeshNodes возвращает доступные mesh-узлы
func (cd *ConnectivityDetector) GetMeshNodes() []*MeshNode {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	nodes := make([]*MeshNode, 0)
	for _, node := range cd.meshNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetCellularTowers возвращает обнаруженные вышки
func (cd *ConnectivityDetector) GetCellularTowers() []*CellularTower {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.cellularTowers
}

// IsOnline возвращает статус подключения к интернету
func (cd *ConnectivityDetector) IsOnline() bool {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.isOnline
}

// IsOfflineModeActive возвращает статус офлайн-режима
func (cd *ConnectivityDetector) IsOfflineModeActive() bool {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.offlineMode
}

// calculateCellularReliability вычисляет надежность сотовой сети
func (cd *ConnectivityDetector) calculateCellularReliability(tower *CellularTower) float64 {
	// Чем крепче сигнал, тем выше надежность
	// -44 дБм = отличный сигнал (1.0)
	// -140 дБм = плохой сигнал (0.1)
	signalRange := float64(-140 - (-44))
	signalNormalized := float64(tower.SignalStrength - (-140)) / signalRange
	if signalNormalized < 0 {
		signalNormalized = 0
	}
	if signalNormalized > 1 {
		signalNormalized = 1
	}
	return 0.1 + (signalNormalized * 0.85)
}

// RoutingDecision содержит рекомендацию маршрута
type RoutingDecision struct {
	Timestamp time.Time
	Routes    []*Route
	BestRoute *Route
}

// Route описывает доступный маршрут
type Route struct {
	Type        string
	Priority    int
	Reliability float64
	Details     string
}

// AdaptiveRouter автоматически переключает маршруты
type AdaptiveRouter struct {
	detector          *ConnectivityDetector
	currentRoute      *Route
	failoverAttempts  int
	maxFailoverTries  int
	mu                sync.RWMutex
}

// NewAdaptiveRouter создает адаптивный маршрутизатор
func NewAdaptiveRouter(detector *ConnectivityDetector) *AdaptiveRouter {
	return &AdaptiveRouter{
		detector:         detector,
		maxFailoverTries: 3,
	}
}

// SwitchRoute переключается на следующий лучший маршрут
func (ar *AdaptiveRouter) SwitchRoute() (*Route, error) {
	decision := ar.detector.GetBestRoute()

	ar.mu.Lock()
	defer ar.mu.Unlock()

	if len(decision.Routes) == 0 {
		return nil, fmt.Errorf("no routes available")
	}

	ar.currentRoute = decision.BestRoute
	ar.failoverAttempts++

	return ar.currentRoute, nil
}

// GetCurrentRoute возвращает текущий маршрут
func (ar *AdaptiveRouter) GetCurrentRoute() *Route {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return ar.currentRoute
}

// EstimateLatency оценивает задержку до целевого сервера через разные маршруты
func (ar *AdaptiveRouter) EstimateLatency(targetAddr string) time.Duration {
	decision := ar.detector.GetBestRoute()
	if len(decision.Routes) == 0 {
		return time.Duration(math.MaxInt64)
	}

	baseLatency := 10 * time.Millisecond
	for _, route := range decision.Routes {
		switch route.Type {
		case "internet":
			return baseLatency + time.Duration(rand.Intn(50))*time.Millisecond
		case "cellular":
			return baseLatency + time.Duration(100+rand.Intn(200))*time.Millisecond
		case "mesh":
			hopsLatency := time.Duration(route.Priority/20) * time.Millisecond
			return baseLatency + hopsLatency + time.Duration(rand.Intn(100))*time.Millisecond
		case "local":
			return baseLatency + time.Duration(rand.Intn(20))*time.Millisecond
		}
	}
	return baseLatency
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
