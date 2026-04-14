package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sova/common"
	"strconv"
	"strings"
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
	runtimeCfg  *common.Config
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
func NewServerAPI(serverKeys *common.ServerKeys, rateLimiter *RateLimiter, logger *Logger, connMonitor *ConnectionMonitor, runtimeCfg *common.Config) *ServerAPI {
	return &ServerAPI{
		users:       make(map[string]*common.UserCredentials),
		sessions:    make(map[string]*Session),
		serverKeys:  serverKeys,
		stats:       &ServerStats{Uptime: time.Since(time.Now())},
		RateLimiter: rateLimiter,
		Logger:      logger,
		ConnMonitor: connMonitor,
		runtimeCfg:  runtimeCfg,
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

func (api *ServerAPI) buildSOVAProfile() *common.JSONConfig {
	server := os.Getenv("SERVER_DOMAIN")
	if server == "" {
		server = os.Getenv("SOVA_DOMAIN")
	}
	if server == "" && api.runtimeCfg != nil && api.runtimeCfg.ServerAddr != "" {
		server = api.runtimeCfg.ServerAddr
	}
	if server == "" {
		server = "sova.example.com"
	}

	serverPort := 443
	psk := common.DefaultPSK
	sniList := []string{"www.google.com", "cdn.cloudflare.com"}
	transports := []string{"tcp-sova"}
	fragmentSize := 2
	fragmentJitter := 25
	websocketPath := ""

	if api.runtimeCfg != nil {
		if api.runtimeCfg.ServerPort > 0 {
			serverPort = api.runtimeCfg.ServerPort
		}
		if api.runtimeCfg.PSK != "" {
			psk = api.runtimeCfg.PSK
		}
		if len(api.runtimeCfg.Transport.SNIList) > 0 {
			sniList = append([]string{}, api.runtimeCfg.Transport.SNIList...)
		}
		if api.runtimeCfg.Stealth.JitterMs >= 0 {
			fragmentJitter = api.runtimeCfg.Stealth.JitterMs
		}
		if api.runtimeCfg.Transport.Mode == "websocket" || api.runtimeCfg.Transport.Mode == "auto" {
			transports = []string{"tcp-sova", "ws-sova"}
			websocketPath = "/sova-ws"
		}
	}

	return &common.JSONConfig{
		Protocol:       "sova",
		Version:        common.Version,
		Server:         server,
		ServerPort:     serverPort,
		ServerPubKey:   base64.StdEncoding.EncodeToString(api.serverKeys.PublicKey),
		PSK:            psk,
		Transports:     transports,
		SNIList:        sniList,
		FragmentSize:   fragmentSize,
		FragmentJitter: fragmentJitter,
		WebSocketPath:  websocketPath,
		LocalProxy:     "http://127.0.0.1:1080",
	}
}

func (api *ServerAPI) buildSOVAShareLink(profile *common.JSONConfig) string {
	query := url.Values{}
	if profile.PSK != "" {
		query.Set("psk", profile.PSK)
	}
	if profile.FragmentSize > 0 {
		query.Set("frag", strconv.Itoa(profile.FragmentSize))
	}
	if profile.FragmentJitter > 0 {
		query.Set("jitter", strconv.Itoa(profile.FragmentJitter))
	}
	if len(profile.SNIList) > 0 {
		query.Set("sni", strings.Join(profile.SNIList, ","))
	}
	if profile.WebSocketPath != "" {
		query.Set("ws", profile.WebSocketPath)
	}
	return fmt.Sprintf("sova://%s:%d?%s#SOVA", profile.Server, profile.ServerPort, query.Encode())
}

func (api *ServerAPI) exportNativeConfig(profile *common.JSONConfig) string {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (api *ServerAPI) exportSingBoxPatchConfig(profile *common.JSONConfig) string {
	return fmt.Sprintf(`{
  "outbounds": [{
    "type": "sova",
    "tag": "SOVA-NATIVE",
    "server": "%s",
    "server_port": %d,
    "psk": "%s",
    "sni_list": ["%s"],
    "fragment_size": %d,
    "fragment_jitter": %d
  }]
}`,
		profile.Server,
		profile.ServerPort,
		profile.PSK,
		strings.Join(profile.SNIList, `","`),
		profile.FragmentSize,
		profile.FragmentJitter,
	)
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

	config := api.buildSOVAProfile()

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

	config := api.buildSOVAProfile()

	var exportedConfig string
	switch clientType {
	case "native", "profile", "sova":
		exportedConfig = api.exportNativeConfig(config)
	case "share_link":
		exportedConfig = api.buildSOVAShareLink(config)
	case "singbox_patch":
		exportedConfig = api.exportSingBoxPatchConfig(config)
	default:
		http.Error(w, "Unsupported client type. Supported: native, profile, sova, share_link, singbox_patch", http.StatusBadRequest)
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
		Name        string `json:"name"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}

	profile := api.buildSOVAProfile()
	encodedProfile, _ := common.EncodeConfig(profile)
	confs := []ProxyConf{
		{Name: "sova_share_link", URL: api.buildSOVAShareLink(profile), Description: "Native SOVA share link for external integrators"},
		{Name: "sova_profile_base64", URL: encodedProfile, Description: "Base64-encoded native SOVA profile"},
		{Name: "sova_local_proxy", URL: profile.LocalProxy, Description: "Local SOVA Proxy endpoint for browsers and apps"},
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
