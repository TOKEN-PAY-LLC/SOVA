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
	"github.com/cloudflare/circl/sign/dilithium/mode5"
	"golang.org/x/crypto/chacha20poly1305"
)

// MasterKey хранит симметричный ключ сервера, генерируется один раз
var MasterKey []byte

// PQ KEM keys (Kyber1024)
var pqKEMScheme = kyber1024.Scheme()
var PQMasterPublicKey []byte  // marshalled public key
var PQMasterPrivateKey []byte // marshalled private key
var pqKEMPublicKey kem.PublicKey
var pqKEMPrivateKey kem.PrivateKey

// PQ Sign keys (Dilithium mode5)
var PQSignPublicKey *mode5.PublicKey
var PQSignPrivateKey *mode5.PrivateKey

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
	pk, sk, _ := pqKEMScheme.GenerateKeyPair()
	pqKEMPublicKey = pk
	pqKEMPrivateKey = sk

	pkBytes, err := pk.MarshalBinary()
	if err != nil {
		return err
	}
	skBytes, err := sk.MarshalBinary()
	if err != nil {
		return err
	}
	PQMasterPublicKey = pkBytes
	PQMasterPrivateKey = skBytes

	// Инициализировать Dilithium mode5 для подписей
	spk, ssk, err := mode5.GenerateKey(rand.Reader)
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
	if pqKEMPublicKey == nil {
		return nil, nil, errors.New("PQ public key not initialized")
	}

	ct, ss, err := pqKEMScheme.Encapsulate(pqKEMPublicKey)
	if err != nil {
		return nil, nil, err
	}

	return ct, ss, nil
}

// DecapsulatePQ дешифрует инкапсулированный общий секрет
func DecapsulatePQ(ciphertext []byte) ([]byte, error) {
	if pqKEMPrivateKey == nil {
		return nil, errors.New("PQ private key not initialized")
	}

	ss, err := pqKEMScheme.Decapsulate(pqKEMPrivateKey, ciphertext)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

// SignPQ подписывает данные используя Dilithium mode5
func SignPQ(message []byte) ([]byte, error) {
	if PQSignPrivateKey == nil {
		return nil, errors.New("PQ signing key not initialized")
	}

	sig := make([]byte, mode5.SignatureSize)
	mode5.SignTo(PQSignPrivateKey, message, sig)
	return sig, nil
}

// VerifyPQ проверяет подпись Dilithium mode5
func VerifyPQ(publicKey *mode5.PublicKey, message, signature []byte) bool {
	if publicKey == nil {
		return false
	}

	return mode5.Verify(publicKey, message, signature)
}

// GetPQPublicKeysBytes возвращает сериализованные публичные ключи для отправки клиенту
func GetPQPublicKeysBytes() (kyberPK []byte, dilPK []byte, err error) {
	if PQMasterPublicKey == nil || PQSignPublicKey == nil {
		return nil, nil, errors.New("PQ keys not initialized")
	}

	dilBytes, err := PQSignPublicKey.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}

	return PQMasterPublicKey, dilBytes, nil
}

// EncryptChaCha20 шифрует данные с ChaCha20-Poly1305
func EncryptChaCha20(key, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptChaCha20 дешифрует данные с ChaCha20-Poly1305
func DecryptChaCha20(key, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short for chacha20")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aead.Open(nil, nonce, ct, nil)
}

// GenerateRandomBytes генерирует криптографически безопасные случайные байты
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DeriveKey выводит ключ из пароля используя HKDF
func DeriveKey(secret, salt, info []byte) ([]byte, error) {
	h := hmac.New(sha256.New, salt)
	h.Write(secret)
	h.Write(info)
	return h.Sum(nil)[:32], nil
}
