package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sova/common"
	"strconv"
	"sync"
	"time"
)

// ServerAPI представляет REST API сервера
type ServerAPI struct {
	mu          sync.RWMutex
	users       map[string]*common.UserCredentials
	sessions    map[string]*Session
	serverKeys  *common.ServerKeys
	stats       *ServerStats
	RateLimiter *RateLimiter
	Logger      *Logger
	ConnMonitor *ConnectionMonitor
}

// Session представляет активную сессию
type Session struct {
	UserID    string
	StartTime time.Time
	BytesUp   int64
	BytesDown int64
}

// ServerStats статистика сервера
type ServerStats struct {
	TotalUsers     int
	ActiveSessions int
	TotalBytes     int64
	Uptime         time.Duration
}

// NewServerAPI создает новый API
func NewServerAPI(serverKeys *common.ServerKeys, rateLimiter *RateLimiter, logger *Logger, connMonitor *ConnectionMonitor) *ServerAPI {
	return &ServerAPI{
		users:       make(map[string]*common.UserCredentials),
		sessions:    make(map[string]*Session),
		serverKeys:  serverKeys,
		stats:       &ServerStats{Uptime: time.Since(time.Now())},
		RateLimiter: rateLimiter,
		Logger:      logger,
		ConnMonitor: connMonitor,
	}
}

// RegisterUser регистрирует пользователя
func (api *ServerAPI) RegisterUser(userID, password string) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	if _, exists := api.users[userID]; exists {
		return fmt.Errorf("user already exists")
	}

	api.users[userID] = &common.UserCredentials{
		UserID:   userID,
		Password: password,
	}
	api.stats.TotalUsers++
	return nil
}

// StartSession начинает сессию
func (api *ServerAPI) StartSession(userID string) (*Session, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if _, exists := api.users[userID]; !exists {
		return nil, fmt.Errorf("user not found")
	}

	sessionID := fmt.Sprintf("%s_%d", userID, time.Now().Unix())
	session := &Session{
		UserID:    userID,
		StartTime: time.Now(),
	}
	api.sessions[sessionID] = session
	api.stats.ActiveSessions++
	return session, nil
}

// UpdateStats обновляет статистику сессии
func (api *ServerAPI) UpdateStats(sessionID string, bytesUp, bytesDown int64) {
	api.mu.Lock()
	defer api.mu.Unlock()

	if session, exists := api.sessions[sessionID]; exists {
		session.BytesUp += bytesUp
		session.BytesDown += bytesDown
		api.stats.TotalBytes += bytesUp + bytesDown
	}
}

// GetStats возвращает статистику
func (api *ServerAPI) GetStats() *ServerStats {
	api.mu.RLock()
	defer api.mu.RUnlock()

	stats := *api.stats
	stats.Uptime = time.Since(time.Now().Add(-stats.Uptime))
	return &stats
}

// HTTP handlers

// handleRegister обрабатывает регистрацию пользователя
func (api *ServerAPI) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Logger.Log("ERROR", "Invalid JSON in register request", getClientIP(r), "")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := api.RegisterUser(req.UserID, req.Password); err != nil {
		api.Logger.Log("WARN", fmt.Sprintf("User registration failed: %v", err), getClientIP(r), req.UserID)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	api.Logger.Log("INFO", "User registered successfully", getClientIP(r), req.UserID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "user registered"})
}

// handleStats возвращает статистику сервера
func (api *ServerAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := api.GetStats()
	api.Logger.Log("INFO", "Stats requested", getClientIP(r), "")
	json.NewEncoder(w).Encode(stats)
}

// handleConfig возвращает конфигурацию для клиента
func (api *ServerAPI) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		api.Logger.Log("WARN", "Config request missing user_id", getClientIP(r), "")
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}

	config := &common.JSONConfig{
		ServerPubKey: base64.StdEncoding.EncodeToString(api.serverKeys.PublicKey),
		Transports:   []string{"web_mirror"},
		SNIList:      []string{"sova.example.com"},
	}

	encoded, err := common.EncodeConfig(config)
	if err != nil {
		api.Logger.Log("ERROR", "Config encoding failed", getClientIP(r), userID)
		http.Error(w, "Config encoding failed", http.StatusInternalServerError)
		return
	}

	api.Logger.Log("INFO", "Config provided", getClientIP(r), userID)
	json.NewEncoder(w).Encode(map[string]string{"config": encoded})
}

// handleExportConfig экспортирует конфиг для различных клиентов
func (api *ServerAPI) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientType := r.URL.Query().Get("client")
	userID := r.URL.Query().Get("user_id")

	if clientType == "" || userID == "" {
		http.Error(w, "client and user_id required", http.StatusBadRequest)
		return
	}

	config := &common.JSONConfig{
		ServerPubKey: base64.StdEncoding.EncodeToString(api.serverKeys.PublicKey),
		Transports:   []string{"web_mirror", "cloud_carrier", "shadow_websocket"},
		SNIList:      []string{"sova.example.com", "cdn.cloudflare.com", "aws.amazon.com"},
	}

	var exportedConfig string
	switch clientType {
	case "xray":
		exportedConfig = api.exportV2RayConfig(config, userID)
	case "v2ray":
		exportedConfig = api.exportV2RayConfig(config, userID)
	case "singbox":
		exportedConfig = api.exportSingBoxConfig(config, userID)
	case "clash":
		exportedConfig = api.exportClashConfig(config, userID)
	case "nekoray":
		exportedConfig = api.exportNekoRayConfig(config, userID)
	default:
		http.Error(w, "Unsupported client type. Supported: xray, v2ray, singbox, clash, nekoray", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"config": exportedConfig})
}

// handleProxyConfig возвращает конфигурации для прокси-клиентов
func (api *ServerAPI) handleProxyConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Example output
	type ProxyConf struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	confs := []ProxyConf{
		{Name: "xray", URL: "vless://" + "your-sova-server.com"},
		{Name: "singbox", URL: "sova://" + "your-sova-server.com"},
		{Name: "generic", URL: "sova://127.0.0.1:1080"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(confs)
}

// registerAPIRoutes добавляет эндпоинты
func (api *ServerAPI) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/register", api.handleRegister)
	mux.HandleFunc("/api/stats", api.handleStats)
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/api/export", api.handleExportConfig)
	mux.HandleFunc("/api/proxy", api.handleProxyConfig)
}

// Экспорт конфигураций для различных клиентов

func (api *ServerAPI) exportClashConfig(config *common.JSONConfig, userID string) string {
	return fmt.Sprintf(`{
  "proxies": [{
    "name": "SOVA-%s",
    "type": "http",
    "server": "127.0.0.1",
    "port": 1080,
    "username": "%s",
    "password": "sova_password"
  }],
  "proxy-groups": [{
    "name": "SOVA",
    "type": "select",
    "proxies": ["SOVA-%s"]
  }]
}`, userID, userID, userID)
}

func (api *ServerAPI) exportV2RayConfig(config *common.JSONConfig, userID string) string {
	return fmt.Sprintf(`{
  "inbounds": [{
    "port": 1080,
    "protocol": "http",
    "settings": {
      "auth": "password",
      "accounts": [{
        "user": "%s",
        "pass": "sova_password"
      }]
    }
  }],
  "outbounds": [{
    "protocol": "vless",
    "settings": {
      "vnext": [{
        "address": "sova.example.com",
        "port": 443,
        "users": [{
          "id": "%s",
          "encryption": "none"
        }]
      }]
    },
    "streamSettings": {
      "network": "tcp",
      "security": "tls",
      "tlsSettings": {
        "serverName": "sova.example.com"
      }
    }
  }]
}`, userID, userID)
}

func (api *ServerAPI) exportSingBoxConfig(config *common.JSONConfig, userID string) string {
	return fmt.Sprintf(`{
  "inbounds": [{
    "type": "http",
    "tag": "sova-in",
    "listen": "::",
    "listen_port": 1080,
    "users": [{
      "username": "%s",
      "password": "sova_password"
    }]
  }],
  "outbounds": [{
    "type": "vless",
    "tag": "sova-out",
    "server": "sova.example.com",
    "server_port": 443,
    "uuid": "%s",
    "tls": {
      "enabled": true,
      "server_name": "sova.example.com"
    }
  }]
}`, userID, userID)
}

func (api *ServerAPI) exportNekoRayConfig(config *common.JSONConfig, userID string) string {
	return fmt.Sprintf(`{
  "outbounds": [{
    "protocol": "vless",
    "settings": {
      "vnext": [{
        "address": "sova.example.com",
        "port": 443,
        "users": [{
          "id": "%s",
          "encryption": "none"
        }]
      }]
    },
    "streamSettings": {
      "network": "tcp",
      "security": "tls",
      "tlsSettings": {
        "serverName": "sova.example.com"
      }
    },
    "tag": "SOVA-%s"
  }]
}`, userID, userID)
}

// StartAPI запускает HTTP API сервер с дашбордом
func (api *ServerAPI) StartAPI(port int) {
	mux := http.NewServeMux()

	// API маршруты
	mux.HandleFunc("/api/register", api.handleRegister)
	mux.HandleFunc("/api/stats", api.handleStats)
	mux.HandleFunc("/api/config", api.handleConfig)
	mux.HandleFunc("/api/export", api.handleExportConfig)
	mux.HandleFunc("/api/proxy", api.handleProxyConfig)

	// Веб-дашборд
	dashboard := NewDashboard(api)
	dashboard.RegisterDashboardRoutes(mux)

	// Применить middleware
	handler := api.Logger.Middleware(api.RateLimiter.Middleware(mux))

	go http.ListenAndServe(":"+strconv.Itoa(port), handler)
	fmt.Printf("SOVA Dashboard: http://localhost:%d\n", port)
	fmt.Printf("SOVA API:       http://localhost:%d/api/\n", port)
}
