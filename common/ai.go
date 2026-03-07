package common

import (
	"fmt"
	"math/rand"
	"time"
)

// AIAdapter представляет AI-подобный адаптер для умного поведения
type AIAdapter struct {
	History     []NetworkEvent
	Strategies  []AdaptationStrategy
	CurrentStrat int
}

// NetworkEvent событие сети
type NetworkEvent struct {
	Timestamp time.Time
	Type      string // "rtt_high", "packet_loss", "rst_detected", "http_stub"
	Value     float64
}

// AdaptationStrategy стратегия адаптации
type AdaptationStrategy struct {
	Name        string
	Conditions  []string
	Actions     []string
	Probability float64
}

// NewAIAdapter создает новый AI адаптер
func NewAIAdapter() *AIAdapter {
	return &AIAdapter{
		History: make([]NetworkEvent, 0),
		Strategies: []AdaptationStrategy{
			{
				Name:        "Switch to QUIC on high RTT",
				Conditions:  []string{"rtt_high"},
				Actions:     []string{"switch_to_quic", "change_sni"},
				Probability: 0.8,
			},
			{
				Name:        "Fragment packets on DPI detection",
				Conditions:  []string{"rst_detected", "http_stub"},
				Actions:     []string{"fragment_packets", "jitter_timing"},
				Probability: 0.9,
			},
			{
				Name:        "Use CDN WebSocket on persistent blocks",
				Conditions:  []string{"packet_loss_high", "rst_detected"},
				Actions:     []string{"switch_to_websocket", "update_cdn_list"},
				Probability: 0.7,
			},
		},
		CurrentStrat: 0,
	}
}

// RecordEvent записывает событие
func (ai *AIAdapter) RecordEvent(eventType string, value float64) {
	event := NetworkEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Value:     value,
	}
	ai.History = append(ai.History, event)
	fmt.Printf("AI: Recorded event %s: %.2f\n", eventType, value)
}

// AnalyzeAndAdapt анализирует историю и адаптируется
func (ai *AIAdapter) AnalyzeAndAdapt() []string {
	actions := make([]string, 0)

	// Простая логика: проверить последние события
	recentEvents := ai.getRecentEvents(10 * time.Second)

	for _, strat := range ai.Strategies {
		if ai.matchesConditions(strat.Conditions, recentEvents) && rand.Float64() < strat.Probability {
			actions = append(actions, strat.Actions...)
			fmt.Printf("AI: Applying strategy '%s'\n", strat.Name)
			break
		}
	}

	return actions
}

// getRecentEvents получает недавние события
func (ai *AIAdapter) getRecentEvents(duration time.Duration) []NetworkEvent {
	cutoff := time.Now().Add(-duration)
	events := make([]NetworkEvent, 0)
	for _, e := range ai.History {
		if e.Timestamp.After(cutoff) {
			events = append(events, e)
		}
	}
	return events
}

// matchesConditions проверяет условия
func (ai *AIAdapter) matchesConditions(conditions []string, events []NetworkEvent) bool {
	for _, cond := range conditions {
		found := false
		for _, e := range events {
			if e.Type == cond {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// PredictNextAction предсказывает следующее действие (простая ML-подобная логика)
func (ai *AIAdapter) PredictNextAction() string {
	// Упрощенная: на основе частоты событий
	rttCount := 0
	rstCount := 0
	for _, e := range ai.History {
		if e.Type == "rtt_high" {
			rttCount++
		}
		if e.Type == "rst_detected" {
			rstCount++
		}
	}

	if rstCount > rttCount {
		return "switch_to_websocket"
	}
	return "fragment_packets"
}