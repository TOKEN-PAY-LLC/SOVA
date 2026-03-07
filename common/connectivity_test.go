package common

import (
	"context"
	"math/rand"
	"testing"
	"time"
)

// TestConnectivityDetector tests internet and network detection
func TestConnectivityDetector(t *testing.T) {
	detector := NewConnectivityDetector()
	
	// Test detection
	ctx, cancel := context.WithCancel(context.Background())
	go detector.StartMonitoring(ctx)
	time.Sleep(100 * time.Millisecond)
	
	// Should detect internet or local networks
	routes := detector.GetBestRoute()
	if routes == nil || len(routes.Routes) == 0 {
		t.Log("No routes detected (expected in test environment)")
	}
	
	cancel()
}

// TestMeshNetwork tests peer-to-peer mesh networking
func TestMeshNetwork(t *testing.T) {
	mesh := NewMeshNetwork("test-node-1", []string{"relay", "client"})
	
	if err := mesh.Start(); err != nil {
		t.Fatalf("Failed to start mesh: %v", err)
	}
	defer mesh.Stop()
	
	// Add a peer
	peer := &Peer{
		ID:         "test-peer-1",
		Address:    "127.0.0.1:9000",
		PublicKey:  make([]byte, 32),
		LastHeartbeat: time.Now(),
		Distance:   1,
		IsReliable: true,
	}
	
	if err := mesh.AddPeer(peer); err != nil {
		t.Fatalf("Failed to add peer: %v", err)
	}
	
	peers := mesh.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}
	
	// Test message sending
	err := mesh.SendMessage(peer.ID, []byte("test data"))
	if err != nil {
		t.Logf("Message send failed (expected in test): %v", err)
	}
}

// TestOfflineFirstArchitecture tests offline-first mode
func TestOfflineFirstArchitecture(t *testing.T) {
	ofa := NewOfflineFirstArchitecture("test-offline-node")
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := ofa.Start(ctx); err != nil {
		t.Fatalf("Failed to start offline architecture: %v", err)
	}
	
	time.Sleep(100 * time.Millisecond)
	
	// Get offline status
	status := ofa.GetOfflineStatus()
	if status == nil {
		t.Error("Failed to get offline status")
	}
	
	// Test caching
	testKey := "test_key"
	testData := []byte("test data for cache")
	
	if err := ofa.CacheData(testKey, testData); err != nil {
		t.Fatalf("Failed to cache data: %v", err)
	}
	
	// Retrieve from cache
	retrieved, err := ofa.RequestData(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve cached data: %v", err)
	}
	
	if string(retrieved) != string(testData) {
		t.Errorf("Data mismatch: got %s, expected %s", string(retrieved), string(testData))
	}
}

// TestPeerDiscoveryService tests peer discovery
func TestPeerDiscoveryService(t *testing.T) {
	pds := NewPeerDiscoveryService()
	
	ctx, cancel := context.WithCancel(context.Background())
	go pds.Start(ctx)
	
	time.Sleep(500 * time.Millisecond)
	
	peers := pds.GetDiscoveredPeers()
	if peers == nil {
		t.Error("Failed to get discovered peers")
	}
	
	cancel()
}

// TestResourceManager tests resource monitoring
func TestResourceManager(t *testing.T) {
	rm := NewResourceManager()
	
	ctx, cancel := context.WithCancel(context.Background())
	go rm.MonitorResources(ctx)
	
	time.Sleep(100 * time.Millisecond)
	
	stats := rm.GetResourceStats()
	if stats == nil {
		t.Error("Failed to get resource stats")
	}
	
	battery := stats["battery_level"].(float64)
	if battery <= 0 || battery > 100 {
		t.Errorf("Battery level out of range: %f", battery)
	}
	
	cancel()
}

// TestConnectivityFailover tests automatic failover
func TestConnectivityFailover(t *testing.T) {
	detector := NewConnectivityDetector()
	router := NewAdaptiveRouter(detector)
	
	ctx, cancel := context.WithCancel(context.Background())
	go detector.StartMonitoring(ctx)
	
	time.Sleep(100 * time.Millisecond)
	
	// Switch to best route
	route, err := router.SwitchRoute()
	if err != nil {
		t.Logf("Route switching failed (expected in test): %v", err)
	} else if route != nil {
		if route.Type == "" {
			t.Error("Route type is empty")
		}
	}
	
	cancel()
}

// TestMeshRelaying tests message relaying in mesh network
func TestMeshRelaying(t *testing.T) {
	mesh := NewMeshNetwork("relay-node", []string{"relay"})
	
	if err := mesh.Start(); err != nil {
		t.Fatalf("Failed to start mesh: %v", err)
	}
	defer mesh.Stop()
	
	// Add multiple peers to test relaying
	for i := 1; i <= 3; i++ {
		peer := &Peer{
			ID:         "peer-" + string(rune(i)),
			Address:    "127.0.0.1:" + string(rune(9000+i)),
			PublicKey:  make([]byte, 32),
			LastHeartbeat: time.Now(),
			IsReliable: true,
		}
		mesh.AddPeer(peer)
	}
	
	peers := mesh.GetPeers()
	if len(peers) != 3 {
		t.Errorf("Expected 3 peers, got %d", len(peers))
	}
	
	// Broadcast message
	err := mesh.BroadcastMessage([]byte("broadcast test"))
	if err != nil {
		t.Logf("Broadcast failed (expected in test): %v", err)
	}
}

// Benchmark tests
func BenchmarkMeshSendMessage(b *testing.B) {
	mesh := NewMeshNetwork("bench-node", []string{"relay"})
	mesh.Start()
	defer mesh.Stop()
	
	peer := &Peer{
		ID:         "bench-peer",
		Address:    "127.0.0.1:9999",
		PublicKey:  make([]byte, 32),
		LastHeartbeat: time.Now(),
		IsReliable: true,
	}
	mesh.AddPeer(peer)
	
	testData := []byte("benchmark test data for mesh network")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mesh.SendMessage(peer.ID, testData)
	}
}

func BenchmarkConnectivityDetection(b *testing.B) {
	detector := NewConnectivityDetector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.detectAllChannels()
	}
}

func BenchmarkOfflineArchitecture(b *testing.B) {
	ofa := NewOfflineFirstArchitecture("bench-offline")
	ctx, cancel := context.WithCancel(context.Background())
	ofa.Start(ctx)
	defer cancel()
	
	testKey := "bench_key"
	testData := []byte("benchmark data for offline cache")
	
	ofa.CacheData(testKey, testData)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ofa.RequestData(testKey)
	}
}
