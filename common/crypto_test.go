package common

import (
	"bytes"
	"testing"
)

func TestInitMasterKey(t *testing.T) {
	MasterKey = nil
	if err := InitMasterKey(); err != nil {
		t.Fatalf("InitMasterKey failed: %v", err)
	}
	if len(MasterKey) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(MasterKey))
	}

	// Second call should not change the key
	original := make([]byte, 32)
	copy(original, MasterKey)
	if err := InitMasterKey(); err != nil {
		t.Fatalf("second InitMasterKey failed: %v", err)
	}
	if !bytes.Equal(original, MasterKey) {
		t.Fatal("InitMasterKey regenerated key on second call")
	}
}

func TestDeriveSessionKey(t *testing.T) {
	MasterKey = nil
	if err := InitMasterKey(); err != nil {
		t.Fatalf("InitMasterKey failed: %v", err)
	}

	nonce1 := []byte("nonce-1-aaaaaaaaaa")
	nonce2 := []byte("nonce-2-bbbbbbbbbb")

	key1, err := DeriveSessionKey(nonce1)
	if err != nil {
		t.Fatalf("DeriveSessionKey failed: %v", err)
	}
	if len(key1) != 32 {
		t.Fatalf("expected 32-byte session key, got %d", len(key1))
	}

	key2, err := DeriveSessionKey(nonce2)
	if err != nil {
		t.Fatalf("DeriveSessionKey failed: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Fatal("different nonces produced same session key")
	}

	// Same nonce should produce same key
	key1Again, _ := DeriveSessionKey(nonce1)
	if !bytes.Equal(key1, key1Again) {
		t.Fatal("same nonce produced different session keys")
	}
}

func TestEncryptDecryptData(t *testing.T) {
	MasterKey = nil
	if err := InitMasterKey(); err != nil {
		t.Fatalf("InitMasterKey failed: %v", err)
	}

	key, _ := DeriveSessionKey([]byte("test-nonce-12345"))
	plaintext := []byte("Hello, SOVA Protocol! This is a test message for AES-256-GCM encryption.")

	ciphertext, err := EncryptData(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptData failed: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	decrypted, err := DecryptData(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptData failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted != plaintext: got %q", decrypted)
	}
}

func TestEncryptDecryptDataWrongKey(t *testing.T) {
	MasterKey = nil
	InitMasterKey()

	key1, _ := DeriveSessionKey([]byte("key-1-aaaaaaaaaa"))
	key2, _ := DeriveSessionKey([]byte("key-2-bbbbbbbbbb"))

	plaintext := []byte("secret data")
	ciphertext, _ := EncryptData(key1, plaintext)

	_, err := DecryptData(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

func TestEncryptDecryptChaCha20(t *testing.T) {
	key := make([]byte, 32)
	copy(key, []byte("test-chacha20-key-32-bytes-long!"))

	plaintext := []byte("ChaCha20-Poly1305 test message for SOVA")

	ciphertext, err := EncryptChaCha20(key, plaintext)
	if err != nil {
		t.Fatalf("EncryptChaCha20 failed: %v", err)
	}

	decrypted, err := DecryptChaCha20(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptChaCha20 failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted != plaintext")
	}
}

func TestInitPQKeys(t *testing.T) {
	PQMasterPublicKey = nil
	PQMasterPrivateKey = nil
	PQSignPublicKey = nil
	PQSignPrivateKey = nil

	if err := InitPQKeys(); err != nil {
		t.Fatalf("InitPQKeys failed: %v", err)
	}

	if PQMasterPublicKey == nil {
		t.Fatal("PQMasterPublicKey is nil")
	}
	if PQMasterPrivateKey == nil {
		t.Fatal("PQMasterPrivateKey is nil")
	}
	if PQSignPublicKey == nil {
		t.Fatal("PQSignPublicKey is nil")
	}
	if PQSignPrivateKey == nil {
		t.Fatal("PQSignPrivateKey is nil")
	}
}

func TestPQEncapsulateDecapsulate(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("InitPQKeys failed: %v", err)
	}

	ct, ss1, err := EncapsulatePQ()
	if err != nil {
		t.Fatalf("EncapsulatePQ failed: %v", err)
	}

	if len(ct) == 0 || len(ss1) == 0 {
		t.Fatal("empty ciphertext or shared secret")
	}

	ss2, err := DecapsulatePQ(ct)
	if err != nil {
		t.Fatalf("DecapsulatePQ failed: %v", err)
	}

	if !bytes.Equal(ss1, ss2) {
		t.Fatal("shared secrets don't match after encapsulate/decapsulate")
	}
}

func TestPQSignVerify(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("InitPQKeys failed: %v", err)
	}

	message := []byte("SOVA Protocol signed message")

	sig, err := SignPQ(message)
	if err != nil {
		t.Fatalf("SignPQ failed: %v", err)
	}

	if len(sig) == 0 {
		t.Fatal("empty signature")
	}

	if !VerifyPQ(PQSignPublicKey, message, sig) {
		t.Fatal("valid signature rejected")
	}

	// Tampered message should fail
	tampered := []byte("SOVA Protocol tampered message!")
	if VerifyPQ(PQSignPublicKey, tampered, sig) {
		t.Fatal("tampered message signature accepted")
	}
}

func TestGetPQPublicKeysBytes(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("InitPQKeys failed: %v", err)
	}

	kyberPK, dilPK, err := GetPQPublicKeysBytes()
	if err != nil {
		t.Fatalf("GetPQPublicKeysBytes failed: %v", err)
	}

	if len(kyberPK) == 0 {
		t.Fatal("empty Kyber public key bytes")
	}
	if len(dilPK) == 0 {
		t.Fatal("empty Dilithium public key bytes")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	b1, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("GenerateRandomBytes failed: %v", err)
	}
	if len(b1) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(b1))
	}

	b2, _ := GenerateRandomBytes(32)
	if bytes.Equal(b1, b2) {
		t.Fatal("two random byte slices are equal")
	}
}

func TestDeriveKey(t *testing.T) {
	secret := []byte("my-secret")
	salt := []byte("my-salt-value-16")
	info := []byte("context-info")

	key1, err := DeriveKey(secret, salt, info)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	if len(key1) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key1))
	}

	// Deterministic
	key2, _ := DeriveKey(secret, salt, info)
	if !bytes.Equal(key1, key2) {
		t.Fatal("DeriveKey not deterministic")
	}

	// Different input -> different key
	key3, _ := DeriveKey([]byte("other-secret"), salt, info)
	if bytes.Equal(key1, key3) {
		t.Fatal("different secrets produced same key")
	}
}

func BenchmarkEncryptData(b *testing.B) {
	MasterKey = nil
	InitMasterKey()
	key, _ := DeriveSessionKey([]byte("bench-nonce-1234"))
	data := make([]byte, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncryptData(key, data)
	}
}

func BenchmarkDecryptData(b *testing.B) {
	MasterKey = nil
	InitMasterKey()
	key, _ := DeriveSessionKey([]byte("bench-nonce-1234"))
	data := make([]byte, 4096)
	ct, _ := EncryptData(key, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecryptData(key, ct)
	}
}

func BenchmarkPQEncapsulate(b *testing.B) {
	InitPQKeys()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncapsulatePQ()
	}
}

func BenchmarkPQSign(b *testing.B) {
	InitPQKeys()
	msg := []byte("benchmark message")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SignPQ(msg)
	}
}
