package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sova/common"
)

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Port       int                                `json:"port"`
	API        APIConfig                          `json:"api"`
	Security   SecurityConfig                     `json:"security"`
	Users      map[string]*common.UserCredentials `json:"users"`
	Transports []string                           `json:"transports"`
	SNIList    []string                           `json:"sni_list"`
	WebSocket  WebSocketConfig                    `json:"websocket"`
}

// WebSocketConfig настройки WebSocket relay для обхода мобильных ISP
type WebSocketConfig struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Path    string `json:"path"`
}

// APIConfig конфигурация API
type APIConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// SecurityConfig конфигурация безопасности
type SecurityConfig struct {
	EnablePQ     bool     `json:"enable_pq"`
	AllowedUsers []string `json:"allowed_users"`
	RateLimit    int      `json:"rate_limit"`
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(filename string) (*ServerConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &ServerConfig{}
	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig сохраняет конфигурацию в файл
func SaveConfig(config *ServerConfig, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Port: 443,
		API: APIConfig{
			Enabled: true,
			Port:    8080,
		},
		Security: SecurityConfig{
			EnablePQ:     true,
			AllowedUsers: []string{},
			RateLimit:    100,
		},
		Users:      make(map[string]*common.UserCredentials),
		Transports: []string{"web_mirror", "cloud_carrier", "shadow_websocket"},
		SNIList:    []string{"sova.example.com", "cdn.cloudflare.com", "aws.amazon.com"},
		WebSocket: WebSocketConfig{
			Enabled: true,
			Port:    9444,
			Path:    "/sova-ws",
		},
	}
}

// ValidateConfig валидирует конфигурацию
func ValidateConfig(config *ServerConfig) error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	if config.API.Enabled && (config.API.Port < 1 || config.API.Port > 65535) {
		return fmt.Errorf("invalid API port: %d", config.API.Port)
	}
	if config.Security.RateLimit < 0 {
		return fmt.Errorf("invalid rate limit: %d", config.Security.RateLimit)
	}
	return nil
}

// ClientConfig конфигурация клиента
type ClientConfig struct {
	ServerAddr  string   `json:"server_addr"`
	UserID      string   `json:"user_id"`
	Password    string   `json:"password"`
	ProxyPort   int      `json:"proxy_port"`
	Transports  []string `json:"transports"`
	SNIList     []string `json:"sni_list"`
	AutoConnect bool     `json:"auto_connect"`
}

// LoadClientConfig загружает клиентскую конфигурацию
func LoadClientConfig(filename string) (*ClientConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &ClientConfig{}
	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// SaveClientConfig сохраняет клиентскую конфигурацию
func SaveClientConfig(config *ClientConfig, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// DefaultClientConfig конфигурация клиента по умолчанию
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerAddr:  "sova.example.com:443",
		UserID:      "user",
		Password:    "password",
		ProxyPort:   1080,
		Transports:  []string{"web_mirror"},
		SNIList:     []string{"sova.example.com"},
		AutoConnect: false,
	}
}
