package common

import (
	"bytes"
	"testing"
)

func TestNewStealthEngine(t *testing.T) {
	se := NewStealthEngine()
	if se == nil {
		t.Fatal("expected non-nil stealth engine")
	}
	if !se.enabled {
		t.Fatal("should be enabled by default")
	}
	if se.profile != ProfileHTTPS {
		t.Fatalf("default profile should be https_browsing, got %s", se.profile)
	}
	if len(se.mimicPatterns) < 3 {
		t.Fatal("expected at least 3 mimic patterns")
	}
}

func TestSetProfile(t *testing.T) {
	se := NewStealthEngine()
	se.SetProfile(ProfileVideo)
	if se.profile != ProfileVideo {
		t.Fatalf("expected video_streaming, got %s", se.profile)
	}
}

func TestIntelligentPadder_Pad(t *testing.T) {
	padder := &IntelligentPadder{
		targetSizes: []int{64, 128, 256, 512, 1024, 1460},
	}

	tests := []struct {
		name string
		data []byte
	}{
		{"tiny", []byte("hi")},
		{"small", []byte("hello world from SOVA")},
		{"medium", bytes.Repeat([]byte("A"), 200)},
		{"large", bytes.Repeat([]byte("B"), 1400)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := padder.PadPacket(tt.data)
			if len(padded) < len(tt.data) {
				t.Fatalf("padded shorter than original: %d < %d", len(padded), len(tt.data))
			}

			extracted, err := padder.UnpadPacket(padded)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(tt.data, extracted) {
				t.Fatalf("roundtrip mismatch for %s", tt.name)
			}
		})
	}
}

func TestAdaptiveJitter_NextDelay(t *testing.T) {
	jitter := &AdaptiveJitter{
		baseInterval: 50000000,  // 50ms
		deviation:    20000000,  // 20ms
	}

	delays := make([]int64, 100)
	for i := range delays {
		d := jitter.NextDelay()
		delays[i] = d.Milliseconds()
		if d < 0 {
			t.Fatalf("negative delay: %v", d)
		}
	}

	// Проверяем что есть вариативность
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Fatal("all delays identical — no jitter")
	}
}

func TestFragmentByProfile(t *testing.T) {
	se := NewStealthEngine()
	data := bytes.Repeat([]byte("X"), 5000)

	profiles := []TrafficProfile{ProfileHTTPS, ProfileVideo, ProfileCloudAPI}
	for _, p := range profiles {
		se.SetProfile(p)
		frags := se.fragmentByProfile(data)
		if len(frags) == 0 {
			t.Fatalf("profile %s produced 0 fragments", p)
		}
		// Reconstruct
		var rebuilt []byte
		for _, f := range frags {
			rebuilt = append(rebuilt, f...)
		}
		if !bytes.Equal(data, rebuilt) {
			t.Fatalf("profile %s: fragments don't reconstruct original", p)
		}
	}
}

func TestIsDecoyPacketStealth(t *testing.T) {
	if IsDecoyPacket(nil) {
		t.Fatal("nil should not be decoy")
	}
	if IsDecoyPacket([]byte{0x00}) {
		t.Fatal("0x00 should not be decoy")
	}
	if !IsDecoyPacket([]byte{0xFE, 0xAA, 0xBB}) {
		t.Fatal("0xFE should be decoy")
	}
}

func TestTLSFingerprint(t *testing.T) {
	se := NewStealthEngine()
	if len(se.fingerprint.CipherSuites) == 0 {
		t.Fatal("expected cipher suites")
	}
	if len(se.fingerprint.Extensions) == 0 {
		t.Fatal("expected extensions")
	}
}

func BenchmarkPadPacketStealth(b *testing.B) {
	padder := &IntelligentPadder{
		targetSizes: []int{64, 128, 256, 512, 1024, 1460},
	}
	data := []byte("benchmark stealth padding")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		padder.PadPacket(data)
	}
}

func BenchmarkJitter(b *testing.B) {
	jitter := &AdaptiveJitter{
		baseInterval: 50000000,
		deviation:    20000000,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jitter.NextDelay()
	}
}
