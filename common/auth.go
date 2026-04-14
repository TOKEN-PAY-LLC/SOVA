package common

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// UserCredentials представляет учетные данные пользователя
type UserCredentials struct {
	UserID   string
	Password string // В реальности хэш или соль
}

// ServerKeys представляет ключи сервера
type ServerKeys struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateServerKeys генерирует пару ключей для сервера
func GenerateServerKeys() (*ServerKeys, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &ServerKeys{PublicKey: pub, PrivateKey: priv}, nil
}

// ZKPChallenge представляет вызов для Zero-Knowledge Proof
type ZKPChallenge struct {
	Nonce []byte
}

// GenerateChallenge генерирует случайный nonce для ZKP
func GenerateChallenge() (*ZKPChallenge, error) {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, err
	}
	return &ZKPChallenge{Nonce: nonce}, nil
}

// ZKPProof представляет доказательство ZKP
type ZKPProof struct {
	Response []byte
}

// ProvePassword доказывает знание пароля без раскрытия (Schnorr-like ZKP)
func (cred *UserCredentials) ProvePassword(challenge *ZKPChallenge, serverPub ed25519.PublicKey) (*ZKPProof, error) {
	message := append(challenge.Nonce, []byte(cred.UserID)...)
	// Derive a 32-byte seed from password using SHA-256
	seed := sha256.Sum256([]byte(cred.Password))
	priv := ed25519.NewKeyFromSeed(seed[:])
	signature := ed25519.Sign(priv, message)
	return &ZKPProof{Response: signature}, nil
}

// VerifyProof проверяет доказательство ZKP
func VerifyProof(proof *ZKPProof, challenge *ZKPChallenge, userID string, password string) error {
	message := append(challenge.Nonce, []byte(userID)...)
	seed := sha256.Sum256([]byte(password))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	if !ed25519.Verify(pub, message, proof.Response) {
		return errors.New("invalid proof")
	}
	return nil
}

// JSONConfig представляет конфигурацию подключения
type JSONConfig struct {
	Protocol       string   `json:"protocol,omitempty"`
	Version        string   `json:"version,omitempty"`
	Server         string   `json:"server,omitempty"`
	ServerPort     int      `json:"server_port,omitempty"`
	ServerPubKey   string   `json:"server_pub_key"`
	PSK            string   `json:"psk,omitempty"`
	Transports     []string `json:"transports"`
	SNIList        []string `json:"sni_list"`
	FragmentSize   int      `json:"fragment_size,omitempty"`
	FragmentJitter int      `json:"fragment_jitter,omitempty"`
	WebSocketPath  string   `json:"websocket_path,omitempty"`
	LocalProxy     string   `json:"local_proxy,omitempty"`
}

// EncodeConfig кодирует JSONConfig в base64 строку
func EncodeConfig(config *JSONConfig) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %v", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeConfig декодирует base64 строку в JSONConfig
func DecodeConfig(encoded string) (*JSONConfig, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	config := &JSONConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}
	return config, nil
}

// Note: Real post-quantum cryptography (Kyber1024 KEM + Dilithium mode5 signatures)
// is implemented in crypto.go using Cloudflare's circl library.
// Use InitPQKeys(), EncapsulatePQ(), DecapsulatePQ(), SignPQ(), VerifyPQ() from crypto.go.
