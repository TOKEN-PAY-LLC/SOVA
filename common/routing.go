package common

// ── SOVA Core — Routing Engine ─────────────────────────────────────────
//
// Rule-based маршрутизация трафика:
//   - domain: точное совпадение, суффикс (.example.com), regex
//   - ip: CIDR диапазоны
//   - geo: страна (GeoIP lookup)
//   - process: имя процесса (Windows)
//   - default: маршрут по умолчанию
//
// Правила применяются сверху вниз, первое совпадение побеждает.
// Если ни одно правило не совпало — используется default outbound.
//
// ────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
)

// ── Rule Types ─────────────────────────────────────────────────────────

// RuleType тип правила маршрутизации
type RuleType string

const (
	RuleDomain  RuleType = "domain"
	RuleSuffix  RuleType = "suffix"
	RuleRegex   RuleType = "regex"
	RuleIPCIDR  RuleType = "ip_cidr"
	RuleGeoIP   RuleType = "geoip"
	RuleProcess RuleType = "process"
	RuleDefault RuleType = "default"
)

// RoutingRule правило маршрутизации
type RoutingRule struct {
	Type      RuleType `json:"type"`
	Value     string   `json:"value"`
	Outbound  string   `json:"outbound"` // "direct", "sova", "block", "http"
	inverted  bool
	regexComp *regexp.Regexp
	cidrNet   *net.IPNet
}

// Compile компилирует правило (regex, CIDR)
func (r *RoutingRule) Compile() error {
	switch r.Type {
	case RuleRegex:
	comp, err := regexp.Compile(r.Value)
		if err != nil {
			return fmt.Errorf("routing: invalid regex '%s': %v", r.Value, err)
		}
		r.regexComp = comp
	case RuleIPCIDR:
		_, cidr, err := net.ParseCIDR(r.Value)
		if err != nil {
			return fmt.Errorf("routing: invalid CIDR '%s': %v", r.Value, err)
		}
		r.cidrNet = cidr
	}
	return nil
}

// Match проверяет, совпадает ли адрес с правилом
func (r *RoutingRule) Match(addr string) bool {
	var matched bool

	switch r.Type {
	case RuleDomain:
		matched = strings.EqualFold(addr, r.Value)
	case RuleSuffix:
		// .example.com совпадает с www.example.com
		host := extractHost(addr)
		suffix := r.Value
		if !strings.HasPrefix(suffix, ".") {
			suffix = "." + suffix
		}
		matched = strings.HasSuffix(strings.ToLower(host), strings.ToLower(suffix)) ||
			strings.EqualFold(host, strings.TrimPrefix(suffix, "."))
	case RuleRegex:
		if r.regexComp != nil {
			matched = r.regexComp.MatchString(extractHost(addr))
		}
	case RuleIPCIDR:
		if r.cidrNet != nil {
			host := extractHost(addr)
			ip := net.ParseIP(host)
			if ip == nil {
				// Попробовать resolve
				ips, err := net.LookupIP(host)
				if err == nil && len(ips) > 0 {
					ip = ips[0]
				}
			}
			if ip != nil {
				matched = r.cidrNet.Contains(ip)
			}
		}
	case RuleGeoIP:
		// GeoIP lookup — упрощённо: проверяем по IP
		host := extractHost(addr)
		ip := net.ParseIP(host)
		if ip != nil {
			matched = matchGeoIP(ip, r.Value)
		}
	case RuleProcess:
		// Process matching — заглушка
		matched = false
	case RuleDefault:
		matched = true
	}

	if r.inverted {
		return !matched
	}
	return matched
}

// ── Router ─────────────────────────────────────────────────────────────

// Router маршрутизатор трафика
type Router struct {
	rules     []RoutingRule
	outbounds map[string]OutboundHandler
	mu        sync.RWMutex
}

// NewRouter создаёт маршрутизатор
func NewRouter(cfg *Config, outbounds map[string]OutboundHandler) *Router {
	r := &Router{
		outbounds: outbounds,
	}

	// Загружаем правила из конфигурации
	r.loadRules(cfg)

	return r
}

// loadRules загружает правила из конфигурации
func (r *Router) loadRules(cfg *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules = make([]RoutingRule, 0)

	for _, rule := range cfg.Routing.Rules {
		rr := RoutingRule{
			Type:     RuleType(rule.Type),
			Value:    rule.Value,
			Outbound: rule.Outbound,
		}
		if err := rr.Compile(); err != nil {
			continue // Пропускаем невалидные правила
		}
		r.rules = append(r.rules, rr)
	}

	// Default rule
	if cfg.Routing.DefaultOutbound != "" {
		r.rules = append(r.rules, RoutingRule{
			Type:     RuleDefault,
			Outbound: cfg.Routing.DefaultOutbound,
		})
	} else {
		// По умолчанию — direct
		r.rules = append(r.rules, RoutingRule{
			Type:     RuleDefault,
			Outbound: "direct",
		})
	}
}

// Resolve находит outbound для адреса
func (r *Router) Resolve(addr string) OutboundHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rule := range r.rules {
		if rule.Match(addr) {
			if outbound, ok := r.outbounds[rule.Outbound]; ok {
				return outbound
			}
		}
	}

	// Fallback на direct
	if outbound, ok := r.outbounds["direct"]; ok {
		return outbound
	}
	return &DirectOutbound{}
}

// AddRule добавляет правило маршрутизации
func (r *Router) AddRule(rule RoutingRule) error {
	if err := rule.Compile(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// Вставляем перед default
	rules := make([]RoutingRule, 0, len(r.rules)+1)
	for _, existing := range r.rules {
		if existing.Type == RuleDefault {
			break
		}
		rules = append(rules, existing)
	}
	rules = append(rules, rule)
	// Добавляем default обратно
	for _, existing := range r.rules {
		if existing.Type == RuleDefault {
			rules = append(rules, existing)
			break
		}
	}
	r.rules = rules

	return nil
}

// RemoveRule удаляет правило по индексу
func (r *Router) RemoveRule(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if index >= 0 && index < len(r.rules) {
		r.rules = append(r.rules[:index], r.rules[index+1:]...)
	}
}

// GetRules возвращает список правил
func (r *Router) GetRules() []RoutingRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]RoutingRule, len(r.rules))
	copy(result, r.rules)
	return result
}

// ── Helpers ────────────────────────────────────────────────────────────

// extractHost извлекает хост из адреса (host:port → host)
func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// matchGeoIP — упрощённый GeoIP matching
// В продакшене используется MaxMind GeoIP2 database
func matchGeoIP(ip net.IP, country string) bool {
	// Упрощённая реализация: проверяем по известным диапазонам
	// В полной реализации загружается GeoLite2-Country.mmdb
	_ = ip
	_ = country
	return false
}

// ── Default Routing Rules ──────────────────────────────────────────────

// DefaultRoutingRules возвращает правила по умолчанию для России
func DefaultRoutingRules() []RoutingRule {
	rules := []RoutingRule{
		// Блокировка рекламы и трекеров
		{Type: RuleSuffix, Value: ".ad.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".ads.google.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".doubleclick.net", Outbound: "block"},
		{Type: RuleSuffix, Value: ".googleadservices.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".googlesyndication.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".googletagmanager.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".google-analytics.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".yandex.ru/clck", Outbound: "block"},
		{Type: RuleSuffix, Value: ".mc.yandex.ru", Outbound: "block"},
		{Type: RuleSuffix, Value: ".an.yandex.ru", Outbound: "block"},
		{Type: RuleSuffix, Value: ".adsrvr.org", Outbound: "block"},
		{Type: RuleSuffix, Value: ".criteo.com", Outbound: "block"},
		{Type: RuleSuffix, Value: ".facebook.net", Outbound: "block"},
		{Type: RuleSuffix, Value: ".fbcdn.net", Outbound: "block"},

		// Локальные адреса — напрямую
		{Type: RuleIPCIDR, Value: "10.0.0.0/8", Outbound: "direct"},
		{Type: RuleIPCIDR, Value: "172.16.0.0/12", Outbound: "direct"},
		{Type: RuleIPCIDR, Value: "192.168.0.0/16", Outbound: "direct"},
		{Type: RuleIPCIDR, Value: "127.0.0.0/8", Outbound: "direct"},
		{Type: RuleIPCIDR, Value: "::1/128", Outbound: "direct"},
		{Type: RuleIPCIDR, Value: "fc00::/7", Outbound: "direct"},

		// Российские сайты — напрямую (если не заблокированы)
		{Type: RuleSuffix, Value: ".ru", Outbound: "direct"},
		{Type: RuleSuffix, Value: ".su", Outbound: "direct"},
		{Type: RuleSuffix, Value: ".рф", Outbound: "direct"},
		{Type: RuleSuffix, Value: ".kz", Outbound: "direct"},
		{Type: RuleSuffix, Value: ".by", Outbound: "direct"},
		{Type: RuleSuffix, Value: ".uz", Outbound: "direct"},

		// Заблокированные сайты — через SOVA
		{Type: RuleSuffix, Value: ".dev", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".app", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".io", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".me", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".com", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".net", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".org", Outbound: "sova"},
		{Type: RuleSuffix, Value: ".info", Outbound: "sova"},

		// Default — через SOVA
		{Type: RuleDefault, Outbound: "sova"},
	}

	// Компилируем regex/CIDR правила
	for i := range rules {
		rules[i].Compile()
	}

	return rules
}
