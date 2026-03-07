package common

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// OfflineFirstArchitecture управляет режимом без интернета
type OfflineFirstArchitecture struct {
	mu                 sync.RWMutex
	meshNetwork        *MeshNetwork
	connectivity       *ConnectivityDetector
	localCache         map[string][]byte
	routingCache       map[string]*CachedRoute
	peerDiscovery      *PeerDiscoveryService
	resourceManager    *ResourceManager
	isOffline          bool
	lastOnlineTime     time.Time
	offlineStartTime   time.Time
	survivability      float64 // 0-1, как долго система может работать офлайн
}

// CachedRoute хранит кэшированные маршруты
type CachedRoute struct {
	DestinationID string
	Route         *Route
	CachedTime    time.Time
	ExpiresAt     time.Time
	Confidence    float64 // 0-1
}

// PeerDiscoveryService обнаруживает новые peer-ы в офлайн-режиме
type PeerDiscoveryService struct {
	mu               sync.RWMutex
	discoveredPeers  map[string]*DiscoveredPeer
	scanInterval     time.Duration
	signalStrengthDB map[string]int
}

// DiscoveredPeer представляет найденный peer коротким диапазоне
type DiscoveredPeer struct {
	ID              string
	SignalStrength  int
	DiscoveredTime  time.Time
	Type            string // "bluetooth", "nfc", "shortrange_radio"
	DataRate        int64
	EncryptionReady bool
}

// ResourceManager управляет ресурсами в офлайн-режиме
type ResourceManager struct {
	mu               sync.RWMutex
	batteryLevel     float64       // 0-100
	cpuUsage         float64       // 0-100
	memoryUsage      float64       // 0-100
	storageAvailable int64         // в байтах
	powerSaveMode    bool
	criticalMode     bool          // режим критической нехватки ресурсов
}

// NewOfflineFirstArchitecture создает архитектуру для работы без интернета
func NewOfflineFirstArchitecture(nodeID string) *OfflineFirstArchitecture {
	return &OfflineFirstArchitecture{
		meshNetwork:    NewMeshNetwork(nodeID, []string{"relay", "cache"}),
		connectivity:   NewConnectivityDetector(),
		localCache:     make(map[string][]byte),
		routingCache:   make(map[string]*CachedRoute),
		peerDiscovery:  NewPeerDiscoveryService(),
		resourceManager: NewResourceManager(),
		isOffline:       false,
	}
}

// Start запускает архитектуру
func (ofa *OfflineFirstArchitecture) Start(ctx context.Context) error {
	// Запускаем сетевые компоненты
	if err := ofa.meshNetwork.Start(); err != nil {
		return fmt.Errorf("ошибка запуска mesh: %v", err)
	}

	// Запускаем мониторинг подключения
	go ofa.connectivity.StartMonitoring(ctx)

	// Запускаем обнаружение peer-ов
	go ofa.peerDiscovery.Start(ctx)

	// Запускаем монитор ресурсов
	go ofa.resourceManager.MonitorResources(ctx)

	// Запускаем адаптивную маршрутизацию
	go ofa.adaptiveRouting(ctx)

	log.Printf("[OfflineFirst] Архитектура запущена для узла")
	return nil
}

// adaptiveRouting выполняет адаптивную маршрутизацию
func (ofa *OfflineFirstArchitecture) adaptiveRouting(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ofa.updateConnectivityStatus()
			ofa.optimizeRouting()
		}
	}
}

// updateConnectivityStatus обновляет статус подключения
func (ofa *OfflineFirstArchitecture) updateConnectivityStatus() {
	ofa.mu.Lock()
	defer ofa.mu.Unlock()

	isOnline := ofa.connectivity.IsOnline()
	wasOffline := ofa.isOffline

	ofa.isOffline = !isOnline

	if wasOffline && !ofa.isOffline {
		// Вернулось подключение к интернету
		ofa.lastOnlineTime = time.Now()
		log.Printf("[OfflineFirst] Восстановлено подключение к интернету")
	} else if !wasOffline && ofa.isOffline {
		// Потеряли подключение к интернету
		ofa.offlineStartTime = time.Now()
		log.Printf("[OfflineFirst] Потеряна связь с интернетом, переходим в offline-режим")
	}
}

// optimizeRouting оптимизирует маршруты на основе доступности
func (ofa *OfflineFirstArchitecture) optimizeRouting() {
	decision := ofa.connectivity.GetBestRoute()

	ofa.mu.Lock()
	defer ofa.mu.Unlock()

	// Обновляем кэш маршрутов
	for _, route := range decision.Routes {
		key := fmt.Sprintf("route_%s", route.Type)
		ofa.routingCache[key] = &CachedRoute{
			DestinationID: route.Type,
			Route:         route,
			CachedTime:    time.Now(),
			ExpiresAt:     time.Now().Add(10 * time.Minute),
			Confidence:    route.Reliability,
		}
	}
}

// RequestData запрашивает данные от сети или кэша
func (ofa *OfflineFirstArchitecture) RequestData(key string) ([]byte, error) {
	ofa.mu.RLock()

	// Проверяем локальный кэш первым
	if data, exists := ofa.localCache[key]; exists {
		ofa.mu.RUnlock()
		return data, nil
	}

	// Если есть интернет - запрашиваем с сервера
	if !ofa.isOffline {
		ofa.mu.RUnlock()
		// TODO: запрос с сервера
		return nil, fmt.Errorf("данные недоступны")
	}

	ofa.mu.RUnlock()

	// Ищем данные в mesh-сети
	peers := ofa.meshNetwork.GetPeers()
	if len(peers) > 0 {
		return ofa.downloadFromPeer(peers[0], key)
	}

	return nil, fmt.Errorf("данные недоступны")
}

// downloadFromPeer скачивает данные от peer-а
func (ofa *OfflineFirstArchitecture) downloadFromPeer(peer *Peer, key string) ([]byte, error) {
	// Отправляем запрос
	requestData := fmt.Sprintf("GET:%s", key)
	err := ofa.meshNetwork.SendMessage(peer.ID, []byte(requestData))
	if err != nil {
		return nil, err
	}

	// Ожидаем ответ с таймаутом
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("таймаут ответа от peer")
		case <-time.After(100 * time.Millisecond):
			// Проверяем, получили ли ответ
			continue
		}
	}
}

// CacheData кэширует данные локально
func (ofa *OfflineFirstArchitecture) CacheData(key string, data []byte) error {
	ofa.mu.Lock()
	defer ofa.mu.Unlock()

	size := int64(len(data))
	if size > ofa.resourceManager.GetAvailableStorage() {
		return fmt.Errorf("недостаточно место для кэша")
	}

	ofa.localCache[key] = data
	return nil
}

// GetOfflineStatus возвращает статус офлайн-режима
func (ofa *OfflineFirstArchitecture) GetOfflineStatus() map[string]interface{} {
	ofa.mu.RLock()
	defer ofa.mu.RUnlock()

	uptime := time.Since(ofa.offlineStartTime)
	if !ofa.isOffline {
		uptime = 0
	}

	return map[string]interface{}{
		"is_offline":        ofa.isOffline,
		"offline_duration":  uptime.String(),
		"mesh_peers":        len(ofa.meshNetwork.GetPeers()),
		"cache_size":        ofa.getCacheSize(),
		"survivability":     ofa.survivability,
		"cellular_towers":   len(ofa.connectivity.GetCellularTowers()),
		"mesh_nodes":        len(ofa.connectivity.GetMeshNodes()),
	}
}

// getCacheSize возвращает размер кэша
func (ofa *OfflineFirstArchitecture) getCacheSize() int64 {
	var size int64
	for _, data := range ofa.localCache {
		size += int64(len(data))
	}
	return size
}

// NewPeerDiscoveryService создает сервис обнаружения peer-ов
func NewPeerDiscoveryService() *PeerDiscoveryService {
	return &PeerDiscoveryService{
		discoveredPeers:  make(map[string]*DiscoveredPeer),
		scanInterval:     3 * time.Second,
		signalStrengthDB: make(map[string]int),
	}
}

// Start запускает сканирование peer-ов
func (pds *PeerDiscoveryService) Start(ctx context.Context) {
	ticker := time.NewTicker(pds.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pds.scanForPeers()
		}
	}
}

// scanForPeers сканирует ближайшие peer-ы (Bluetooth, NFC и т.д.)
func (pds *PeerDiscoveryService) scanForPeers() {
	pds.mu.Lock()
	defer pds.mu.Unlock()

	// Имитируем сканирование Bluetooth
	numPeers := rand.Intn(5)
	for i := 0; i < numPeers; i++ {
		peerID := fmt.Sprintf("peer_%d", rand.Int63())
		signal := -80 + rand.Intn(40) // -80 до -40 dBm

		peer := &DiscoveredPeer{
			ID:              peerID,
			SignalStrength:  signal,
			DiscoveredTime:  time.Now(),
			Type:            "bluetooth",
			DataRate:        1_000_000, // 1 Mbps
			EncryptionReady: true,
		}

		pds.discoveredPeers[peerID] = peer
	}

	// Удаляем старые peer-ы
	now := time.Now()
	for id, peer := range pds.discoveredPeers {
		if now.Sub(peer.DiscoveredTime) > 30*time.Second {
			delete(pds.discoveredPeers, id)
		}
	}
}

// GetDiscoveredPeers возвращает найденные peer-ы
func (pds *PeerDiscoveryService) GetDiscoveredPeers() []*DiscoveredPeer {
	pds.mu.RLock()
	defer pds.mu.RUnlock()

	peers := make([]*DiscoveredPeer, 0)
	for _, peer := range pds.discoveredPeers {
		peers = append(peers, peer)
	}
	return peers
}

// NewResourceManager создает менеджер ресурсов
func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		batteryLevel:     100,
		cpuUsage:         50,
		memoryUsage:      40,
		storageAvailable: 5_000_000_000, // 5 GB
		powerSaveMode:    false,
		criticalMode:     false,
	}
}

// MonitorResources мониторит ресурсы системы
func (rm *ResourceManager) MonitorResources(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.updateResources()
		}
	}
}

// updateResources обновляет показатели ресурсов
func (rm *ResourceManager) updateResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Имитируем снижение батареи в офлайн-режиме
	if rm.batteryLevel > 5 {
		rm.batteryLevel -= 0.2
	}

	// Обновляем CPU и память
	rm.cpuUsage = float64(40+rand.Intn(30)) / 100
	rm.memoryUsage = float64(30+rand.Intn(40)) / 100

	// Активируем режим экономии батареи при 20%
	if rm.batteryLevel < 20 {
		rm.powerSaveMode = true
	}

	// Критический режим при 5%
	if rm.batteryLevel < 5 {
		rm.criticalMode = true
	}
}

// GetResourceStats возвращает статистику ресурсов
func (rm *ResourceManager) GetResourceStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return map[string]interface{}{
		"battery_level":      rm.batteryLevel,
		"cpu_usage":          rm.cpuUsage * 100,
		"memory_usage":       rm.memoryUsage * 100,
		"storage_available":  rm.storageAvailable,
		"power_save_mode":    rm.powerSaveMode,
		"critical_mode":      rm.criticalMode,
	}
}

// GetAvailableStorage возвращает доступное место
func (rm *ResourceManager) GetAvailableStorage() int64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.storageAvailable
}

// CalculateSurvivability вычисляет, как долго система может работать офлайн
func (ofa *OfflineFirstArchitecture) CalculateSurvivability() float64 {
	ofa.mu.RLock()
	defer ofa.mu.RUnlock()

	resourceStats := ofa.resourceManager.GetResourceStats()
	battery := resourceStats["battery_level"].(float64)
	meshPeers := float64(len(ofa.meshNetwork.GetPeers()))

	// Формула: (батарея / 100) * (1 + log(mesh_peers)) * (1 - ресурс_использование)
	cpuUsage := resourceStats["cpu_usage"].(float64) / 100
	survivability := (battery / 100) * (1 + math.Log(max(1, meshPeers))) * (1 - cpuUsage)

	if survivability > 1.0 {
		survivability = 1.0
	}
	if survivability < 0 {
		survivability = 0
	}

	return survivability
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
