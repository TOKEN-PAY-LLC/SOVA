package common

import (
	"testing"
)

func TestNewAIAdapter(t *testing.T) {
	ai := NewAIAdapter()
	if ai == nil {
		t.Fatal("NewAIAdapter returned nil")
	}
	if len(ai.History) != 0 {
		t.Fatalf("expected 0 events, got %d", len(ai.History))
	}
	if len(ai.Strategies) == 0 {
		t.Fatal("expected default strategies")
	}
}

func TestRecordEvent(t *testing.T) {
	ai := NewAIAdapter()

	ai.RecordEvent("rtt_high", 150.0)
	ai.RecordEvent("packet_loss_high", 0.08)
	ai.RecordEvent("rst_detected", 1.0)

	if len(ai.History) != 3 {
		t.Fatalf("expected 3 events, got %d", len(ai.History))
	}

	if ai.History[0].Type != "rtt_high" {
		t.Fatalf("expected rtt_high, got %s", ai.History[0].Type)
	}
	if ai.History[0].Value != 150.0 {
		t.Fatalf("expected 150.0, got %f", ai.History[0].Value)
	}
}

func TestAnalyzeAndAdapt(t *testing.T) {
	ai := NewAIAdapter()

	// Record events that should trigger strategies
	for i := 0; i < 5; i++ {
		ai.RecordEvent("rtt_high", 200.0)
	}

	actions := ai.AnalyzeAndAdapt()
	// Should get some actions based on strategies
	// The exact actions depend on probability, so just check it doesn't panic
	_ = actions
}

func TestPredictNextAction(t *testing.T) {
	ai := NewAIAdapter()

	// With no events, prediction should still work
	action := ai.PredictNextAction()
	if action == "" {
		t.Fatal("PredictNextAction returned empty string")
	}

	// With events
	for i := 0; i < 10; i++ {
		ai.RecordEvent("rst_detected", 1.0)
	}
	action = ai.PredictNextAction()
	if action == "" {
		t.Fatal("PredictNextAction with events returned empty")
	}
}

func TestAdaptiveSwitcher(t *testing.T) {
	switcher := NewAdaptiveSwitcher()
	if switcher == nil {
		t.Fatal("NewAdaptiveSwitcher returned nil")
	}
	if switcher.CurrentMode != WebMirrorMode {
		t.Fatalf("expected WebMirrorMode, got %d", switcher.CurrentMode)
	}
	if switcher.Metrics == nil {
		t.Fatal("Metrics is nil")
	}
	if switcher.AI == nil {
		t.Fatal("AI is nil")
	}
}

func TestExecuteAction(t *testing.T) {
	switcher := NewAdaptiveSwitcher()

	switcher.ExecuteAction("switch_to_quic")
	if switcher.CurrentMode != CloudCarrierMode {
		t.Fatalf("expected CloudCarrierMode after switch_to_quic")
	}

	switcher.ExecuteAction("switch_to_websocket")
	if switcher.CurrentMode != ShadowWebSocketMode {
		t.Fatalf("expected ShadowWebSocketMode after switch_to_websocket")
	}

	// These should not panic
	switcher.ExecuteAction("fragment_packets")
	switcher.ExecuteAction("jitter_timing")
	switcher.ExecuteAction("change_sni")
	switcher.ExecuteAction("update_cdn_list")
	switcher.ExecuteAction("unknown_action")
}

func TestNetworkMetrics(t *testing.T) {
	metrics := &NetworkMetrics{}
	if metrics.RTT != 0 {
		t.Fatal("expected zero RTT")
	}
	if metrics.PacketLoss != 0 {
		t.Fatal("expected zero PacketLoss")
	}
}
