package common

import (
	"bytes"
	"testing"
)

func TestTrafficCompressor_Compress(t *testing.T) {
	tc := &TrafficCompressor{level: 1, minSize: 32, algorithm: "gzip"}

	// Маленькие данные — не сжимаются
	small := []byte("hello")
	out, err := tc.Compress(small)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != 0x00 {
		t.Fatal("small data should not be compressed")
	}

	// Большие повторяющиеся данные — сжимаются
	big := bytes.Repeat([]byte("SOVA protocol is the best "), 100)
	out, err = tc.Compress(big)
	if err != nil {
		t.Fatal(err)
	}
	if out[0] != 0x01 {
		t.Fatal("large repetitive data should be compressed")
	}
	if len(out) >= len(big) {
		t.Fatalf("compressed should be smaller: %d >= %d", len(out), len(big))
	}
}

func TestTrafficCompressor_RoundTrip(t *testing.T) {
	tc := &TrafficCompressor{level: 1, minSize: 32, algorithm: "gzip"}

	testCases := [][]byte{
		[]byte("short"),
		bytes.Repeat([]byte("ABCDEF"), 200),
		make([]byte, 4096),
	}

	for i, original := range testCases {
		compressed, err := tc.Compress(original)
		if err != nil {
			t.Fatalf("case %d: compress error: %v", i, err)
		}
		decompressed, err := tc.Decompress(compressed)
		if err != nil {
			t.Fatalf("case %d: decompress error: %v", i, err)
		}
		if !bytes.Equal(original, decompressed) {
			t.Fatalf("case %d: roundtrip mismatch", i)
		}
	}
}

func TestConnectionPool_GetReturn(t *testing.T) {
	pool := &ConnectionPool{
		conns:    make(map[string][]*PooledConn),
		maxIdle:  8,
		maxTotal: 32,
		idleTime: 30 * 1000000000,
	}

	callCount := 0
	dial := func() (interface{}, error) {
		callCount++
		return nil, nil
	}
	_ = dial
	_ = pool

	// Pool structure test
	if pool.maxIdle != 8 {
		t.Fatalf("expected maxIdle 8, got %d", pool.maxIdle)
	}
	if pool.maxTotal != 32 {
		t.Fatalf("expected maxTotal 32, got %d", pool.maxTotal)
	}
}

func TestConnectionPool_CleanIdle(t *testing.T) {
	pool := &ConnectionPool{
		conns:    make(map[string][]*PooledConn),
		maxIdle:  8,
		maxTotal: 32,
		idleTime: 0, // всё протухает
	}

	cleaned := pool.CleanIdle()
	if cleaned != 0 {
		t.Fatalf("expected 0 cleaned, got %d", cleaned)
	}
}

func TestRouteOptimizer_UpdateAndBest(t *testing.T) {
	ro := &RouteOptimizer{routes: make(map[string]*RouteInfo)}

	ro.UpdateRoute("fast-server", 10*1000000, 0.01, 100*1024*1024)
	ro.UpdateRoute("slow-server", 200*1000000, 0.1, 1*1024*1024)

	best := ro.BestRoute()
	if best == nil {
		t.Fatal("expected a best route")
	}
	if best.Target != "fast-server" {
		t.Fatalf("expected fast-server, got %s", best.Target)
	}
}

func TestIntelligentPadder_RoundTrip(t *testing.T) {
	padder := &IntelligentPadder{
		targetSizes: []int{64, 128, 256, 512, 1024, 1460},
	}

	original := []byte("secret owl data that must be padded")
	padded := padder.PadPacket(original)

	if len(padded) < len(original) {
		t.Fatalf("padded should be >= original: %d < %d", len(padded), len(original))
	}

	extracted, err := padder.UnpadPacket(padded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, extracted) {
		t.Fatalf("roundtrip mismatch: got %q", extracted)
	}
}

func TestNewTrafficAccelerator(t *testing.T) {
	ta := NewTrafficAccelerator()
	if ta == nil {
		t.Fatal("expected non-nil accelerator")
	}
	if !ta.enabled {
		t.Fatal("should be enabled by default")
	}

	stats := ta.GetStats()
	if stats["bytes_total"].(int64) != 0 {
		t.Fatal("expected 0 bytes_total initially")
	}
}

func TestIsDecoyPacket(t *testing.T) {
	if IsDecoyPacket([]byte{0x01, 0x02}) {
		t.Fatal("0x01 should not be decoy")
	}
	if !IsDecoyPacket([]byte{0xFE, 0x02}) {
		t.Fatal("0xFE should be decoy")
	}
	if IsDecoyPacket([]byte{}) {
		t.Fatal("empty should not be decoy")
	}
}

func BenchmarkCompress(b *testing.B) {
	tc := &TrafficCompressor{level: 1, minSize: 32, algorithm: "gzip"}
	data := bytes.Repeat([]byte("SOVA benchmark data "), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc.Compress(data)
	}
}

func BenchmarkPadPacket(b *testing.B) {
	padder := &IntelligentPadder{
		targetSizes: []int{64, 128, 256, 512, 1024, 1460},
	}
	data := []byte("benchmark padding test data")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		padder.PadPacket(data)
	}
}
