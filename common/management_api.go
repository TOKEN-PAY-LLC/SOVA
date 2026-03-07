package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// StartManagementAPI запускает REST API для управления SOVA
func StartManagementAPI(cfg *Config, ui *UI) {
	mux := http.NewServeMux()
	startTime := time.Now()

	// GET /api/status — статус системы
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version":    Version,
			"mode":       cfg.Mode,
			"uptime":     time.Since(startTime).String(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
			"goroutines": runtime.NumGoroutine(),
			"listen":     cfg.ListenAddress(),
		})
	})

	// GET /api/config — текущая конфигурация
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			data, err := cfg.ToJSON()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(data)

		case http.MethodPut:
			var update Config
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
				return
			}
			data, _ := json.Marshal(&update)
			if err := cfg.UpdateFromJSON(data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cfg.Save(GetConfigPath())
			json.NewEncoder(w).Encode(map[string]string{"status": "config updated"})

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /api/features — статус всех модулей
	mux.HandleFunc("/api/features", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(cfg.GetFeatureStatus())
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// POST /api/features/{name}/enable  и  /api/features/{name}/disable
	mux.HandleFunc("/api/feature/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if !cfg.SetFeature(req.Name, req.Enabled) {
			http.Error(w, fmt.Sprintf("Unknown feature: %s", req.Name), http.StatusBadRequest)
			return
		}
		cfg.Save(GetConfigPath())

		action := "enabled"
		if !req.Enabled {
			action = "disabled"
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"feature": req.Name,
			"action":  action,
		})
	})

	// GET /api/health — проверка здоровья
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": Version,
			"uptime":  time.Since(startTime).Seconds(),
		})
	})

	// GET /api/system — системная информация
	mux.HandleFunc("/api/system", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
			"go_version":   runtime.Version(),
			"cpus":         runtime.NumCPU(),
			"goroutines":   runtime.NumGoroutine(),
			"memory_alloc": mem.Alloc,
			"memory_sys":   mem.Sys,
			"gc_runs":      mem.NumGC,
		})
	})

	// POST /api/config/set — установить одно значение конфигурации
	mux.HandleFunc("/api/config/set", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		switch req.Key {
		case "listen_port":
			port, err := strconv.Atoi(req.Value)
			if err != nil {
				http.Error(w, "invalid port", http.StatusBadRequest)
				return
			}
			cfg.ListenPort = port
		case "listen_addr":
			cfg.ListenAddr = req.Value
		case "mode":
			cfg.Mode = req.Value
		case "encryption":
			cfg.Encryption.Algorithm = req.Value
		case "stealth_profile":
			cfg.Stealth.Profile = req.Value
		case "log_level":
			cfg.LogLevel = req.Value
		case "server_addr":
			cfg.ServerAddr = req.Value
		case "server_port":
			port, err := strconv.Atoi(req.Value)
			if err != nil {
				http.Error(w, "invalid port", http.StatusBadRequest)
				return
			}
			cfg.ServerPort = port
		default:
			http.Error(w, fmt.Sprintf("unknown key: %s", req.Key), http.StatusBadRequest)
			return
		}

		cfg.Save(GetConfigPath())
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"key":    req.Key,
			"value":  req.Value,
		})
	})

	// Корневой маршрут — список доступных API
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "SOVA Management API",
			"version": Version,
			"endpoints": []string{
				"GET  /api/status     — System status",
				"GET  /api/health     — Health check",
				"GET  /api/config     — Current config",
				"PUT  /api/config     — Update full config",
				"POST /api/config/set — Set single config value",
				"GET  /api/features   — List all features",
				"POST /api/feature/   — Enable/disable feature",
				"GET  /api/system     — System information",
			},
		})
	})

	addr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	http.ListenAndServe(addr, mux)
}
