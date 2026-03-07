package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config представляет полную конфигурацию SOVA
type Config struct {
	// Основные настройки
	Mode       string `json:"mode"`        // "local", "remote", "server"
	ListenAddr string `json:"listen_addr"` // Адрес SOCKS5 прокси
	ListenPort int    `json:"listen_port"` // Порт SOCKS5 прокси

	// Настройки удалённого сервера
	ServerAddr string `json:"server_addr"` // Адрес удалённого SOVA сервера
	ServerPort int    `json:"server_port"` // Порт сервера

	// Шифрование
	Encryption EncryptionConfig `json:"encryption"`

	// Стелс
	Stealth StealthConfig `json:"stealth"`

	// API управления
	API APIConfig `json:"api"`

	// DNS
	DNS DNSConfig `json:"dns"`

	// Логирование
	LogLevel string `json:"log_level"` // "debug", "info", "warn", "error"
	LogFile  string `json:"log_file"`

	// Управление модулями (включить/выключить)
	Features FeaturesConfig `json:"features"`

	// Транспорт
	Transport TransportConfig2 `json:"transport"`

	mu sync.RWMutex `json:"-"`
}

// EncryptionConfig настройки шифрования
type EncryptionConfig struct {
	Algorithm  string `json:"algorithm"`   // "aes-256-gcm", "chacha20-poly1305"
	PQEnabled  bool   `json:"pq_enabled"`  // Пост-квантовая криптография Kyber1024+Dilithium
	ZKPEnabled bool   `json:"zkp_enabled"` // Zero-Knowledge Proof аутентификация
}

// StealthConfig настройки стелс-режима
type StealthConfig struct {
	Enabled        bool   `json:"enabled"`
	Profile        string `json:"profile"`         // "chrome", "youtube", "cloud_api", "random"
	JitterMs       int    `json:"jitter_ms"`        // Среднее время jitter
	PaddingEnabled bool   `json:"padding_enabled"`
	DecoyEnabled   bool   `json:"decoy_enabled"`
	TLSFingerprint string `json:"tls_fingerprint"` // "chrome", "firefox", "safari", "random"
}

// APIConfig настройки управляющего REST API
type APIConfig struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Host    string `json:"host"`
	AuthKey string `json:"auth_key"` // Ключ для доступа к API (пустой = без авторизации)
}

// DNSConfig настройки DNS-over-SOVA
type DNSConfig struct {
	Enabled  bool   `json:"enabled"`
	Port     int    `json:"port"`
	Upstream string `json:"upstream"` // Upstream DNS (8.8.8.8:53, 1.1.1.1:53)
}

// FeaturesConfig управление модулями
type FeaturesConfig struct {
	Compression    bool `json:"compression"`     // Gzip сжатие трафика
	ConnectionPool bool `json:"connection_pool"`  // Переиспользование соединений
	SmartRouting   bool `json:"smart_routing"`    // Оптимизация маршрутов
	MeshNetwork    bool `json:"mesh_network"`     // Mesh-сеть между нодами
	OfflineFirst   bool `json:"offline_first"`    // Offline-first архитектура
	AIAdapter      bool `json:"ai_adapter"`       // AI-адаптивное переключение
	Dashboard      bool `json:"dashboard"`        // Веб-дашборд
	AutoProxy      bool `json:"auto_proxy"`       // Авто-настройка системного прокси
}

// TransportConfig2 настройки транспорта (2 чтобы не конфликтовать с TransportConfig)
type TransportConfig2 struct {
	Mode       string   `json:"mode"`        // "web_mirror", "quic", "websocket", "auto"
	SNIList    []string `json:"sni_list"`    // Список SNI для маскировки
	CDNList    []string `json:"cdn_list"`    // Список CDN для WebSocket
	Fallback   bool     `json:"fallback"`    // Автопереключение при блокировке
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Mode:       "local",
		ListenAddr: "127.0.0.1",
		ListenPort: 1080,
		ServerAddr: "",
		ServerPort: 443,
		Encryption: EncryptionConfig{
			Algorithm:  "aes-256-gcm",
			PQEnabled:  true,
			ZKPEnabled: true,
		},
		Stealth: StealthConfig{
			Enabled:        true,
			Profile:        "chrome",
			JitterMs:       50,
			PaddingEnabled: true,
			DecoyEnabled:   false,
			TLSFingerprint: "chrome",
		},
		API: APIConfig{
			Enabled: true,
			Port:    8080,
			Host:    "127.0.0.1",
		},
		DNS: DNSConfig{
			Enabled:  false,
			Port:     5353,
			Upstream: "8.8.8.8:53",
		},
		LogLevel: "info",
		Features: FeaturesConfig{
			Compression:    true,
			ConnectionPool: true,
			SmartRouting:   true,
			MeshNetwork:    false,
			OfflineFirst:   false,
			AIAdapter:      true,
			Dashboard:      true,
			AutoProxy:      false,
		},
		Transport: TransportConfig2{
			Mode:     "auto",
			SNIList:  []string{"www.google.com", "cdn.cloudflare.com", "aws.amazon.com"},
			CDNList:  []string{"cdn.cloudflare.com", "fastly.net"},
			Fallback: true,
		},
	}
}

// GetConfigDir возвращает путь к директории конфигурации
func GetConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sova")
}

// GetConfigPath возвращает путь к файлу конфигурации
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

// LoadConfig загружает конфигурацию из файла; создаёт default если не существует
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if saveErr := cfg.Save(path); saveErr != nil {
				return cfg, nil // Не критично, работаем с default
			}
			return cfg, nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %v", err)
	}
	return cfg, nil
}

// Save сохраняет конфигурацию в файл
func (c *Config) Save(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SetFeature включает/выключает модуль. Возвращает false если модуль не найден.
func (c *Config) SetFeature(name string, enabled bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch name {
	case "compression":
		c.Features.Compression = enabled
	case "connection_pool":
		c.Features.ConnectionPool = enabled
	case "smart_routing":
		c.Features.SmartRouting = enabled
	case "mesh_network":
		c.Features.MeshNetwork = enabled
	case "offline_first":
		c.Features.OfflineFirst = enabled
	case "ai_adapter":
		c.Features.AIAdapter = enabled
	case "dashboard":
		c.Features.Dashboard = enabled
	case "auto_proxy":
		c.Features.AutoProxy = enabled
	case "stealth":
		c.Stealth.Enabled = enabled
	case "dns":
		c.DNS.Enabled = enabled
	case "api":
		c.API.Enabled = enabled
	case "pq_crypto":
		c.Encryption.PQEnabled = enabled
	case "zkp":
		c.Encryption.ZKPEnabled = enabled
	case "decoy":
		c.Stealth.DecoyEnabled = enabled
	case "padding":
		c.Stealth.PaddingEnabled = enabled
	default:
		return false
	}
	return true
}

// GetFeatureStatus возвращает статус всех модулей
func (c *Config) GetFeatureStatus() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]bool{
		"compression":     c.Features.Compression,
		"connection_pool": c.Features.ConnectionPool,
		"smart_routing":   c.Features.SmartRouting,
		"mesh_network":    c.Features.MeshNetwork,
		"offline_first":   c.Features.OfflineFirst,
		"ai_adapter":      c.Features.AIAdapter,
		"dashboard":       c.Features.Dashboard,
		"auto_proxy":      c.Features.AutoProxy,
		"stealth":         c.Stealth.Enabled,
		"dns":             c.DNS.Enabled,
		"api":             c.API.Enabled,
		"pq_crypto":       c.Encryption.PQEnabled,
		"zkp":             c.Encryption.ZKPEnabled,
		"decoy":           c.Stealth.DecoyEnabled,
		"padding":         c.Stealth.PaddingEnabled,
	}
}

// ToJSON сериализует конфигурацию в JSON
func (c *Config) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.MarshalIndent(c, "", "  ")
}

// UpdateFromJSON обновляет конфигурацию из JSON
func (c *Config) UpdateFromJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return json.Unmarshal(data, c)
}

// ListenAddress возвращает полный адрес прослушивания
func (c *Config) ListenAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("%s:%d", c.ListenAddr, c.ListenPort)
}

// ServerAddress возвращает полный адрес сервера
func (c *Config) ServerAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("%s:%d", c.ServerAddr, c.ServerPort)
}
