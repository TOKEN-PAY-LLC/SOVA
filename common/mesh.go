package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// MeshNetwork управляет peer-to-peer mesh-сетью SOVA
type MeshNetwork struct {
	mu              sync.RWMutex
	nodeID          string
	peers           map[string]*Peer
	routingTable    map[string]*RoutingEntry
	messageQueue    chan *MeshMessage
	broadcastChan   chan *MeshMessage
	isActive        bool
	capabilities    []string // "gateway", "relay", "client"
	maxPeers        int
}

// Peer представляет одного участника mesh-сети
type Peer struct {
	ID              string
	Address         string
	PublicKey       []byte
	LastHeartbeat   time.Time
	Distance        int       // количество хопов
	Bandwidth       int64     // в бит/сек
	IsReliable      bool      // стабильно ли соединение
	Messages        int       // количество relay-сообщений
	EncryptionLevel int       // 1-5, выше - лучше
}

// RoutingEntry закэширует маршруты в mesh-сети
type RoutingEntry struct {
	DestinationID  string
	NextHopID      string
	Metric         int
	LastUpdated    time.Time
	Reliability    float64
}

// MeshMessage расшифровывается и передается между узлами
type MeshMessage struct {
	ID            string
	SourceID      string
	DestinationID string
	Type          string // "data", "heartbeat", "routing", "barrier"
	Payload       []byte
	TTL           int
	Timestamp     time.Time
	EncryptedFlag bool
}

// NewMeshNetwork создает новую mesh-сеть
func NewMeshNetwork(nodeID string, capabilities []string) *MeshNetwork {
	return &MeshNetwork{
		nodeID:       nodeID,
		peers:        make(map[string]*Peer),
		routingTable: make(map[string]*RoutingEntry),
		messageQueue: make(chan *MeshMessage, 1000),
		broadcastChan: make(chan *MeshMessage, 100),
		isActive:     false,
		capabilities: capabilities,
		maxPeers:     50,
	}
}

// Start запускает mesh-сеть
func (mn *MeshNetwork) Start() error {
	mn.mu.Lock()
	mn.isActive = true
	mn.mu.Unlock()

	// Запускаем обработчик сообщений
	go mn.processMessages()

	// Запускаем broadcast
	go mn.broadcastHeartbeats()

	// Запускаем очистку неактивных сверстников
	go mn.cleanupInactivePeers()

	log.Printf("[Mesh] Сеть запущена с узлом %s", mn.nodeID)
	return nil
}

// AddPeer добавляет нового сверстника в сеть
func (mn *MeshNetwork) AddPeer(peer *Peer) error {
	mn.mu.Lock()
	defer mn.mu.Unlock()

	if len(mn.peers) >= mn.maxPeers {
		return fmt.Errorf("максимальное количество peer-ов достигнуто")
	}

	if _, exists := mn.peers[peer.ID]; exists {
		return fmt.Errorf("peer %s уже добавлен", peer.ID)
	}

	peer.LastHeartbeat = time.Now()
	mn.peers[peer.ID] = peer

	// Добавляем в таблицу маршрутизации
	mn.routingTable[peer.ID] = &RoutingEntry{
		DestinationID: peer.ID,
		NextHopID:     peer.ID,
		Metric:        1,
		LastUpdated:   time.Now(),
		Reliability:   0.8,
	}

	log.Printf("[Mesh] Добавлен peer: %s (%s)", peer.ID, peer.Address)
	return nil
}

// SendMessage отправляет сообщение через mesh-сеть
func (mn *MeshNetwork) SendMessage(destID string, data []byte) error {
	mn.mu.RLock()
	if !mn.isActive {
		mn.mu.RUnlock()
		return fmt.Errorf("mesh-сеть не активна")
	}
	mn.mu.RUnlock()

	msgID := mn.generateMessageID()
	msg := &MeshMessage{
		ID:            msgID,
		SourceID:      mn.nodeID,
		DestinationID: destID,
		Type:          "data",
		Payload:       data,
		TTL:           32,
		Timestamp:     time.Now(),
		EncryptedFlag: true,
	}

	select {
	case mn.messageQueue <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("очередь сообщений переполнена")
	}
}

// BroadcastMessage отправляет сообщение всем узлам
func (mn *MeshNetwork) BroadcastMessage(data []byte) error {
	mn.mu.RLock()
	peers := len(mn.peers)
	mn.mu.RUnlock()

	if peers == 0 {
		return fmt.Errorf("нет доступных peer-ов")
	}

	msgID := mn.generateMessageID()
	msg := &MeshMessage{
		ID:            msgID,
		SourceID:      mn.nodeID,
		DestinationID: "broadcast",
		Type:          "data",
		Payload:       data,
		TTL:           32,
		Timestamp:     time.Now(),
		EncryptedFlag: true,
	}

	select {
	case mn.broadcastChan <- msg:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("широкоречевой канал переполнен")
	}
}

// processMessages обрабатывает входящие сообщения
func (mn *MeshNetwork) processMessages() {
	for msg := range mn.messageQueue {
		if msg.TTL <= 0 {
			continue // Сообщение истекло
		}

		mn.mu.RLock()
		dest, exists := mn.peers[msg.DestinationID]
		mn.mu.RUnlock()

		if !exists {
			// Маршрутизируем через промежуточный узел
			mn.relayMessage(msg)
			continue
		}

		// Отправляем сообщение
		mn.forwardToPeer(dest, msg)
	}
}

// relayMessage релирует сообщение к следующему узлу
func (mn *MeshNetwork) relayMessage(msg *MeshMessage) {
	msg.TTL--

	mn.mu.RLock()
	route, exists := mn.routingTable[msg.DestinationID]
	mn.mu.RUnlock()

	if !exists {
		return // Нет маршрута
	}

	mn.mu.RLock()
	nextPeer, exists := mn.peers[route.NextHopID]
	mn.mu.RUnlock()

	if exists && nextPeer.IsReliable {
		mn.forwardToPeer(nextPeer, msg)
		nextPeer.Messages++
	}
}

// forwardToPeer отправляет сообщение одному peer-у
func (mn *MeshNetwork) forwardToPeer(peer *Peer, msg *MeshMessage) {
	// Шифруем сообщение перед отправкой
	if msg.EncryptedFlag {
		encData, err := EncryptData(peer.PublicKey[:32], msg.Payload)
		if err != nil {
			log.Printf("[Mesh] Ошибка шифрования: %v", err)
			return
		}
		msg.Payload = encData
	}

	// Отправляем UDP пакет
	addr, err := net.ResolveUDPAddr("udp", peer.Address)
	if err != nil {
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		peer.IsReliable = false
		return
	}
	defer conn.Close()

	data, _ := json.Marshal(msg)
	conn.Write(data)
	peer.IsReliable = true
}

// broadcastHeartbeats отправляет периодические сигналы жизни
func (mn *MeshNetwork) broadcastHeartbeats() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mn.mu.RLock()
		if !mn.isActive {
			mn.mu.RUnlock()
			return
		}

		peers := len(mn.peers)
		mn.mu.RUnlock()

		if peers > 0 {
			heartbeat := map[string]interface{}{
				"node_id":       mn.nodeID,
				"capabilities": mn.capabilities,
				"peer_count":   peers,
				"timestamp":    time.Now().Unix(),
			}

			data, _ := json.Marshal(heartbeat)
			mn.BroadcastMessage(data)
		}
	}
}

// cleanupInactivePeers удаляет неактивные peer-ы
func (mn *MeshNetwork) cleanupInactivePeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mn.mu.Lock()

		now := time.Now()
		for id, peer := range mn.peers {
			if now.Sub(peer.LastHeartbeat) > 60*time.Second {
				delete(mn.peers, id)
				delete(mn.routingTable, id)
				log.Printf("[Mesh] Удален неактивный peer: %s", id)
			}
		}

		mn.mu.Unlock()
	}
}

// GetPeers возвращает список активных peer-ов
func (mn *MeshNetwork) GetPeers() []*Peer {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	peers := make([]*Peer, 0)
	for _, peer := range mn.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetNetworkStats возвращает статистику сети
func (mn *MeshNetwork) GetNetworkStats() map[string]interface{} {
	mn.mu.RLock()
	defer mn.mu.RUnlock()

	totalMessages := 0
	totalBandwidth := int64(0)

	for _, peer := range mn.peers {
		totalMessages += peer.Messages
		totalBandwidth += peer.Bandwidth
	}

	return map[string]interface{}{
		"node_id":          mn.nodeID,
		"peer_count":       len(mn.peers),
		"total_messages":   totalMessages,
		"total_bandwidth":  totalBandwidth,
		"routing_entries":  len(mn.routingTable),
		"is_active":        mn.isActive,
		"capabilities":    mn.capabilities,
	}
}

// generateMessageID генерирует уникальный ID сообщения
func (mn *MeshNetwork) generateMessageID() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%d", mn.nodeID, time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Stop останавливает mesh-сеть
func (mn *MeshNetwork) Stop() {
	mn.mu.Lock()
	mn.isActive = false
	mn.mu.Unlock()

	close(mn.messageQueue)
	close(mn.broadcastChan)

	log.Printf("[Mesh] Сеть остановлена")
}
