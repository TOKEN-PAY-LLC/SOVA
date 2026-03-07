package common

import (
	"testing"
)

// TestEncryptDecrypt tests basic AES-GCM encryption and decryption
func TestEncryptDecrypt(t *testing.T) {
	// Initialize master key
	if err := InitMasterKey(); err != nil {
		t.Fatalf("Failed to init master key: %v", err)
	}

	// Create a test nonce
	testNonce := make([]byte, 16)
	for i := 0; i < len(testNonce); i++ {
		testNonce[i] = byte(i)
	}

	// Derive session key
	sessionKey, err := DeriveSessionKey(testNonce)
	if err != nil {
		t.Fatalf("Failed to derive session key: %v", err)
	}

	plaintext := []byte("Hello, SOVA Protocol!")
	
	// Encrypt
	ciphertext, err := EncryptData(sessionKey, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Decrypt
	decrypted, err := DecryptData(sessionKey, ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// Verify
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch. Got: %s, Expected: %s", decrypted, plaintext)
	}
}

// TestDeriveSessionKey tests session key derivation
func TestDeriveSessionKey(t *testing.T) {
	if err := InitMasterKey(); err != nil {
		t.Fatalf("Failed to init master key: %v", err)
	}

	nonce1 := make([]byte, 16)
	nonce2 := make([]byte, 16)
	
	for i := 0; i < len(nonce1); i++ {
		nonce1[i] = byte(i)
		nonce2[i] = byte(i + 1)
	}

	key1, err := DeriveSessionKey(nonce1)
	if err != nil {
		t.Fatalf("Failed to derive key1: %v", err)
	}

	key2, err := DeriveSessionKey(nonce2)
	if err != nil {
		t.Fatalf("Failed to derive key2: %v", err)
	}

	// Keys should be different for different nonces
	if string(key1) == string(key2) {
		t.Error("Different nonces should produce different keys")
	}

	// Keys should be consistent
	key1_again, _ := DeriveSessionKey(nonce1)
	if string(key1) != string(key1_again) {
		t.Error("Same nonce should produce same key")
	}
}

// TestInitPQKeys tests post-quantum key initialization
func TestInitPQKeys(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("Failed to init PQ keys: %v", err)
	}

	if PQMasterPublicKey == nil {
		t.Error("PQ master public key is nil")
	}

	if PQMasterPrivateKey == nil {
		t.Error("PQ master private key is nil")
	}

	if PQSignPublicKey == nil {
		t.Error("PQ signing public key is nil")
	}

	if PQSignPrivateKey == nil {
		t.Error("PQ signing private key is nil")
	}
}

// TestPQEncapsulation tests PQ key encapsulation
func TestPQEncapsulation(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("Failed to init PQ keys: %v", err)
	}

	// Encapsulate
	ciphertext, sharedSecret, err := EncapsulatePQ()
	if err != nil {
		t.Fatalf("Encapsulation failed: %v", err)
	}

	if ciphertext == nil || len(ciphertext) == 0 {
		t.Error("Ciphertext is empty")
	}

	if sharedSecret == nil || len(sharedSecret) == 0 {
		t.Error("Shared secret is empty")
	}

	// Decapsulate
	decapsSecret, err := DecapsulatePQ(ciphertext)
	if err != nil {
		t.Fatalf("Decapsulation failed: %v", err)
	}

	// Shared secrets should match
	if string(sharedSecret) != string(decapsSecret) {
		t.Error("Shared secrets do not match after encapsulation/decapsulation")
	}
}

// TestPQSignature tests PQ signing
func TestPQSignature(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("Failed to init PQ keys: %v", err)
	}

	message := []byte("Test message for SOVA Protocol")

	// Sign
	signature, err := SignPQ(message)
	if err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	if signature == nil || len(signature) == 0 {
		t.Error("Signature is empty")
	}

	// Verify
	isValid := VerifyPQ(PQSignPublicKey, message, signature)
	if !isValid {
		t.Error("Signature verification failed")
	}

	// Verify with wrong message should fail
	wrongMessage := []byte("Wrong message")
	isValid = VerifyPQ(PQSignPublicKey, wrongMessage, signature)
	if isValid {
		t.Error("Signature verification should fail for wrong message")
	}
}

// TestGetPQPublicKeysBytes tests serialization of PQ public keys
func TestGetPQPublicKeysBytes(t *testing.T) {
	if err := InitPQKeys(); err != nil {
		t.Fatalf("Failed to init PQ keys: %v", err)
	}

	kyberPK, dilPK, err := GetPQPublicKeysBytes()
	if err != nil {
		t.Fatalf("Failed to get PQ public keys: %v", err)
	}

	if kyberPK == nil || len(kyberPK) == 0 {
		t.Error("Kyber public key bytes are empty")
	}

	if dilPK == nil || len(dilPK) == 0 {
		t.Error("Dilithium public key bytes are empty")
	}
}

// Benchmark tests
func BenchmarkEncryptData(b *testing.B) {
	if err := InitMasterKey(); err != nil {
		b.Fatalf("Failed to init master key: %v", err)
	}

	nonce := make([]byte, 16)
	sessionKey, _ := DeriveSessionKey(nonce)
	plaintext := []byte("This is a test message for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncryptData(sessionKey, plaintext)
	}
}

func BenchmarkDecryptData(b *testing.B) {
	if err := InitMasterKey(); err != nil {
		b.Fatalf("Failed to init master key: %v", err)
	}

	nonce := make([]byte, 16)
	sessionKey, _ := DeriveSessionKey(nonce)
	plaintext := []byte("This is a test message for benchmarking")
	ciphertext, _ := EncryptData(sessionKey, plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecryptData(sessionKey, ciphertext)
	}
}

func BenchmarkPQEncapsulation(b *testing.B) {
	if err := InitPQKeys(); err != nil {
		b.Fatalf("Failed to init PQ keys: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncapsulatePQ()
	}
}

func BenchmarkPQSignature(b *testing.B) {
	if err := InitPQKeys(); err != nil {
		b.Fatalf("Failed to init PQ keys: %v", err)
	}

	message := []byte("Test message for benchmarking")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SignPQ(message)
	}
}
