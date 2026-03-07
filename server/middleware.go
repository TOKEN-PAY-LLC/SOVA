package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter ограничивает скорость запросов
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*ClientLimiter
	rate     int // requests per minute
	interval time.Duration
}

// ClientLimiter лимитер для клиента
type ClientLimiter struct {
	requests []time.Time
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(rate int) *RateLimiter {
	return &RateLimiter{
		clients:  make(map[string]*ClientLimiter),
		rate:     rate,
		interval: time.Minute,
	}
}

// Allow проверяет, разрешен ли запрос
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limiter, exists := rl.clients[clientIP]
	if !exists {
		limiter = &ClientLimiter{requests: []time.Time{}}
		rl.clients[clientIP] = limiter
	}

	// Очистить старые запросы
	cutoff := now.Add(-rl.interval)
	validRequests := []time.Time{}
	for _, req := range limiter.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	limiter.requests = validRequests

	if len(limiter.requests) >= rl.rate {
		return false
	}

	limiter.requests = append(limiter.requests, now)
	return true
}

// Logger логирует запросы
type Logger struct {
	mu     sync.Mutex
	logs   []LogEntry
	maxLogs int
}

// LogEntry запись лога
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	ClientIP  string
	UserID    string
}

// NewLogger создает новый логгер
func NewLogger(maxLogs int) *Logger {
	return &Logger{
		logs:     []LogEntry{},
		maxLogs:  maxLogs,
	}
}

// Log добавляет запись в лог
func (l *Logger) Log(level, message, clientIP, userID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		ClientIP:  clientIP,
		UserID:    userID,
	}

	l.logs = append(l.logs, entry)
	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[1:]
	}

	fmt.Printf("[%s] %s: %s (IP: %s, User: %s)\n", entry.Timestamp.Format("15:04:05"), level, message, clientIP, userID)
}

// GetLogs возвращает логи
func (l *Logger) GetLogs(limit int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 || limit > len(l.logs) {
		limit = len(l.logs)
	}

	start := len(l.logs) - limit
	return l.logs[start:]
}

// ConnectionMonitor мониторит соединения
type ConnectionMonitor struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionInfo
}

// ConnectionInfo информация о соединении
type ConnectionInfo struct {
	ClientIP    string
	UserID      string
	StartTime   time.Time
	BytesUp     int64
	BytesDown   int64
	LastActivity time.Time
}

// NewConnectionMonitor создает монитор соединений
func NewConnectionMonitor() *ConnectionMonitor {
	return &ConnectionMonitor{
		connections: make(map[string]*ConnectionInfo),
	}
}

// AddConnection добавляет соединение
func (cm *ConnectionMonitor) AddConnection(connID, clientIP, userID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[connID] = &ConnectionInfo{
		ClientIP:     clientIP,
		UserID:       userID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}
}

// UpdateConnection обновляет статистику соединения
func (cm *ConnectionMonitor) UpdateConnection(connID string, bytesUp, bytesDown int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.connections[connID]; exists {
		conn.BytesUp += bytesUp
		conn.BytesDown += bytesDown
		conn.LastActivity = time.Now()
	}
}

// RemoveConnection удаляет соединение
func (cm *ConnectionMonitor) RemoveConnection(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.connections, connID)
}

// GetActiveConnections возвращает активные соединения
func (cm *ConnectionMonitor) GetActiveConnections() map[string]*ConnectionInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]*ConnectionInfo)
	for id, conn := range cm.connections {
		result[id] = conn
	}
	return result
}

// Middleware для HTTP API
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		if !rl.Allow(clientIP) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		l.Log("INFO", fmt.Sprintf("%s %s", r.Method, r.URL.Path), clientIP, "")

		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		l.Log("INFO", fmt.Sprintf("Request completed in %v", duration), clientIP, "")
	})
}

// getClientIP получает IP клиента
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}