package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/dilithium/dilithium5"
)

// MasterKey хранит симметричный ключ сервера, генерируется один раз
var MasterKey []byte

// PQMasterKey хранит публичный ключ сервера для пост-квантового шифрования
var PQMasterPublicKey kem.PublicKey
var PQMasterPrivateKey kem.PrivateKey

// PQSignPublicKey сохраняет публичный ключ подписи Dilithium
var PQSignPublicKey sign.PublicKey
var PQSignPrivateKey sign.PrivateKey

// MasterKey хранит симметричный ключ сервера, генерируется один раз
var MasterKey []byte

// InitMasterKey создает ключ, если он ещё не инициализирован
func InitMasterKey() error {
	if len(MasterKey) == 0 {
		key := make([]byte, 32)
		_, err := rand.Read(key)
		if err != nil {
			return err
		}
		MasterKey = key
	}
	return nil
}

// DeriveSessionKey получает ключ сессии из master и nonce
func DeriveSessionKey(nonce []byte) ([]byte, error) {
	if len(MasterKey) != 32 {
		return nil, errors.New("master key not initialized")
	}
	h := hmac.New(sha256.New, MasterKey)
	h.Write(nonce)
	return h.Sum(nil)[:32], nil
}

// EncryptData шифрует данные AES-GCM с ключом сессии
func EncryptData(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptData расшифровывает AES-GCM
func DecryptData(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}
// InitPQKeys инициализирует пост-квантовые ключи сервера
func InitPQKeys() error {
	// Инициализировать Kyber1024 для KEM
	pk, sk, err := kyber1024.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	PQMasterPublicKey = pk
	PQMasterPrivateKey = sk

	// Инициализировать Dilithium5 для подписей
	spk, ssk, err := dilithium5.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	PQSignPublicKey = spk
	PQSignPrivateKey = ssk

	return nil
}

// EncapsulatePQ выполняет инкапсуляцию для PQ-КЕМ
// Возвращает шифротекст и общий секрет
func EncapsulatePQ() ([]byte, []byte, error) {
	if PQMasterPublicKey == nil {
		return nil, nil, errors.New("PQ public key not initialized")
	}
	
	// Используем Kyber1024 для инкапсуляции
	encapKey, err := PQMasterPublicKey.Encaps(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	
	return encapKey.Ciphertext(), encapKey.SharedSecret(), nil
}

// DecapsulatePQ дешифрует инкапсулированный общий секрет
func DecapsulatePQ(ciphertext []byte) ([]byte, error) {
	if PQMasterPrivateKey == nil {
		return nil, errors.New("PQ private key not initialized")
	}
	
	// Используем Kyber1024 для декапсуляции
	sharedSecret, err := PQMasterPrivateKey.Decaps(ciphertext)
	if err != nil {
		return nil, err
	}
	
	return sharedSecret, nil
}

// SignPQ подписывает данные используя Dilithium5
func SignPQ(message []byte) ([]byte, error) {
	if PQSignPrivateKey == nil {
		return nil, errors.New("PQ signing key not initialized")
	}
	
	return PQSignPrivateKey.Sign(rand.Reader, message)
}

// VerifyPQ проверяет подпись Dilithium5
func VerifyPQ(publicKey sign.PublicKey, message, signature []byte) bool {
	if publicKey == nil {
		return false
	}
	
	return publicKey.Verify(message, signature)
}

// GetPQPublicKeysBytes возвращает сериализованные пубичные ключи для отправки клиенту
func GetPQPublicKeysBytes() (kyberPK []byte, dilPK []byte, err error) {
	if PQMasterPublicKey == nil || PQSignPublicKey == nil {
		return nil, nil, errors.New("PQ keys not initialized")
	}
	
	kyberPK = PQMasterPublicKey.Bytes()
	dilPK = PQSignPublicKey.Bytes()
	
	return kyberPK, dilPK, nil
}