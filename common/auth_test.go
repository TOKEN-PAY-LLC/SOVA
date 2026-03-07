package common

import (
	"testing"
)

func TestGenerateServerKeys(t *testing.T) {
	keys, err := GenerateServerKeys()
	if err != nil {
		t.Fatalf("GenerateServerKeys failed: %v", err)
	}
	if len(keys.PublicKey) == 0 {
		t.Fatal("empty public key")
	}
	if len(keys.PrivateKey) == 0 {
		t.Fatal("empty private key")
	}
}

func TestGenerateChallenge(t *testing.T) {
	c1, err := GenerateChallenge()
	if err != nil {
		t.Fatalf("GenerateChallenge failed: %v", err)
	}
	if len(c1.Nonce) != 32 {
		t.Fatalf("expected 32-byte nonce, got %d", len(c1.Nonce))
	}

	c2, _ := GenerateChallenge()
	if string(c1.Nonce) == string(c2.Nonce) {
		t.Fatal("two challenges have identical nonces")
	}
}

func TestProveAndVerifyPassword(t *testing.T) {
	keys, _ := GenerateServerKeys()
	cred := &UserCredentials{UserID: "testuser", Password: "testpassword123"}

	challenge, err := GenerateChallenge()
	if err != nil {
		t.Fatalf("GenerateChallenge failed: %v", err)
	}

	proof, err := cred.ProvePassword(challenge, keys.PublicKey)
	if err != nil {
		t.Fatalf("ProvePassword failed: %v", err)
	}
	if len(proof.Response) == 0 {
		t.Fatal("empty proof response")
	}

	// Verify with correct password
	err = VerifyProof(proof, challenge, cred.UserID, cred.Password)
	if err != nil {
		t.Fatalf("VerifyProof failed for valid proof: %v", err)
	}
}

func TestVerifyProofWrongPassword(t *testing.T) {
	keys, _ := GenerateServerKeys()
	cred := &UserCredentials{UserID: "testuser", Password: "correctpassword"}

	challenge, _ := GenerateChallenge()
	proof, _ := cred.ProvePassword(challenge, keys.PublicKey)

	// Verify with wrong password should fail
	err := VerifyProof(proof, challenge, cred.UserID, "wrongpassword")
	if err == nil {
		t.Fatal("VerifyProof should fail with wrong password")
	}
}

func TestVerifyProofWrongUserID(t *testing.T) {
	keys, _ := GenerateServerKeys()
	cred := &UserCredentials{UserID: "user1", Password: "password123"}

	challenge, _ := GenerateChallenge()
	proof, _ := cred.ProvePassword(challenge, keys.PublicKey)

	// Verify with wrong userID should fail
	err := VerifyProof(proof, challenge, "user2", cred.Password)
	if err == nil {
		t.Fatal("VerifyProof should fail with wrong userID")
	}
}

func TestEncodeDecodeConfig(t *testing.T) {
	original := &JSONConfig{
		ServerPubKey: "dGVzdC1rZXk=",
		Transports:   []string{"web_mirror", "cloud_carrier", "shadow_websocket"},
		SNIList:      []string{"sova.example.com", "cdn.cloudflare.com"},
	}

	encoded, err := EncodeConfig(original)
	if err != nil {
		t.Fatalf("EncodeConfig failed: %v", err)
	}
	if encoded == "" {
		t.Fatal("encoded config is empty")
	}

	decoded, err := DecodeConfig(encoded)
	if err != nil {
		t.Fatalf("DecodeConfig failed: %v", err)
	}

	if decoded.ServerPubKey != original.ServerPubKey {
		t.Fatalf("ServerPubKey mismatch: %s != %s", decoded.ServerPubKey, original.ServerPubKey)
	}
	if len(decoded.Transports) != len(original.Transports) {
		t.Fatalf("Transports length mismatch")
	}
	for i, tr := range decoded.Transports {
		if tr != original.Transports[i] {
			t.Fatalf("Transport[%d] mismatch: %s != %s", i, tr, original.Transports[i])
		}
	}
	if len(decoded.SNIList) != len(original.SNIList) {
		t.Fatalf("SNIList length mismatch")
	}
}

func TestDecodeConfigInvalidBase64(t *testing.T) {
	_, err := DecodeConfig("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeConfigInvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	_, err := DecodeConfig("bm90LWpzb24=") // "not-json" in base64
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
