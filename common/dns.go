package common

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// DoSOVAResolver DNS-over-SOVA резолвер для обхода DNS-блокировок
type DoSOVAResolver struct {
	mu             sync.RWMutex
	cache          map[string]*DNSCacheEntry
	upstreamDNS    []string
	cacheTTL       time.Duration
	fallbackDNS    []string
	encryptedMode  bool
}

// DNSCacheEntry запись DNS кэша
type DNSCacheEntry struct {
	IPs       []net.IP
	ExpiresAt time.Time
	Source    string
}

// NewDoSOVAResolver создает DNS резолвер
func NewDoSOVAResolver() *DoSOVAResolver {
	return &DoSOVAResolver{
		cache: make(map[string]*DNSCacheEntry),
		upstreamDNS: []string{
			"1.1.1.1:53",       // Cloudflare
			"8.8.8.8:53",       // Google
			"9.9.9.9:53",       // Quad9
			"208.67.222.222:53", // OpenDNS
		},
		fallbackDNS: []string{
			"1.0.0.1:53",
			"8.8.4.4:53",
			"149.112.112.112:53",
		},
		cacheTTL:      5 * time.Minute,
		encryptedMode: true,
	}
}

// Resolve разрешает доменное имя с обходом блокировок
func (r *DoSOVAResolver) Resolve(domain string) ([]net.IP, error) {
	domain = strings.TrimSuffix(strings.ToLower(domain), ".")

	// Проверить кэш
	if entry := r.getFromCache(domain); entry != nil {
		return entry.IPs, nil
	}

	// Попробовать все upstream DNS
	for _, dns := range r.upstreamDNS {
		ips, err := r.queryDNS(domain, dns)
		if err == nil && len(ips) > 0 {
			r.putToCache(domain, ips, "upstream:"+dns)
			return ips, nil
		}
	}

	// Fallback DNS
	for _, dns := range r.fallbackDNS {
		ips, err := r.queryDNS(domain, dns)
		if err == nil && len(ips) > 0 {
			r.putToCache(domain, ips, "fallback:"+dns)
			return ips, nil
		}
	}

	// System resolver как последний вариант
	ips, err := r.systemResolve(domain)
	if err == nil && len(ips) > 0 {
		r.putToCache(domain, ips, "system")
		return ips, nil
	}

	return nil, fmt.Errorf("DNS resolution failed for %s", domain)
}

func (r *DoSOVAResolver) queryDNS(domain, server string) ([]net.IP, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", server)
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}
	return ips, nil
}

func (r *DoSOVAResolver) systemResolve(domain string) ([]net.IP, error) {
	addrs, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

func (r *DoSOVAResolver) getFromCache(domain string) *DNSCacheEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, ok := r.cache[domain]
	if !ok {
		return nil
	}
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry
}

func (r *DoSOVAResolver) putToCache(domain string, ips []net.IP, source string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cache[domain] = &DNSCacheEntry{
		IPs:       ips,
		ExpiresAt: time.Now().Add(r.cacheTTL),
		Source:    source,
	}
}

// GetCacheStats возвращает статистику кэша
func (r *DoSOVAResolver) GetCacheStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := 0
	expired := 0
	now := time.Now()
	for _, entry := range r.cache {
		if now.Before(entry.ExpiresAt) {
			active++
		} else {
			expired++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(r.cache),
		"active_entries":  active,
		"expired_entries": expired,
	}
}

// ClearCache очищает DNS кэш
func (r *DoSOVAResolver) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*DNSCacheEntry)
}

// CreateSOVADialer создает net.Dialer с DNS-over-SOVA
func (r *DoSOVAResolver) CreateSOVADialer(timeout time.Duration) *net.Dialer {
	return &net.Dialer{
		Timeout: timeout,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// Используем первый upstream DNS
				d := net.Dialer{Timeout: 3 * time.Second}
				return d.DialContext(ctx, "udp", r.upstreamDNS[0])
			},
		},
	}
}
