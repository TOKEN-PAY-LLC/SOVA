package common

import (
	"net"
	"testing"
	"time"
)

// TestSOVAProtocolHandshakeAndFrames проверяет полный цикл SOVA протокола:
// хендшейк → шифрование → фреймы → расшифровка
func TestSOVAProtocolHandshakeAndFrames(t *testing.T) {
	psk := "test-psk-for-sova-protocol"

	// Создаём TCP пару через pipe
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	var serverConn *SOVAConn
	var serverErr error
	done := make(chan struct{})

	// Серверная сторона хендшейка
	go func() {
		serverConn, serverErr = ServerHandshake(server, psk)
		done <- struct{}{}
	}()

	// Клиентская сторона хендшейка
	clientConn, err := ClientHandshake(client, psk)
	if err != nil {
		t.Fatalf("ClientHandshake failed: %v", err)
	}

	<-done
	if serverErr != nil {
		t.Fatalf("ServerHandshake failed: %v", serverErr)
	}

	// Тест 1: client → server (net.Pipe synchronous: write in goroutine)
	testPayload := []byte("Hello from SOVA client!")
	errCh := make(chan error, 1)
	go func() {
		errCh <- clientConn.WriteFrame(&Frame{Type: FrameData, Payload: testPayload})
	}()

	frame, err := serverConn.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}
	if e := <-errCh; e != nil {
		t.Fatalf("WriteFrame failed: %v", e)
	}
	if frame.Type != FrameData {
		t.Fatalf("expected FrameData, got %d", frame.Type)
	}
	if string(frame.Payload) != string(testPayload) {
		t.Fatalf("payload mismatch: %q != %q", frame.Payload, testPayload)
	}

	// Тест 2: server → client
	serverPayload := []byte("Response from SOVA server!")
	go func() {
		errCh <- serverConn.WriteFrame(&Frame{Type: FrameData, Payload: serverPayload})
	}()

	frame2, err := clientConn.ReadFrame()
	if err != nil {
		t.Fatalf("Client ReadFrame failed: %v", err)
	}
	if e := <-errCh; e != nil {
		t.Fatalf("Server WriteFrame failed: %v", e)
	}
	if string(frame2.Payload) != string(serverPayload) {
		t.Fatalf("server payload mismatch")
	}

	// Тест 3: CONNECT + ACK
	go func() {
		errCh <- clientConn.WriteFrame(&Frame{Type: FrameConnect, Payload: []byte("google.com:443")})
	}()

	connectFrame, err := serverConn.ReadFrame()
	if err != nil {
		t.Fatalf("CONNECT ReadFrame failed: %v", err)
	}
	<-errCh
	if connectFrame.Type != FrameConnect || string(connectFrame.Payload) != "google.com:443" {
		t.Fatalf("CONNECT frame mismatch")
	}

	// Send ACK back
	go func() {
		errCh <- serverConn.WriteFrame(&Frame{Type: FrameAck, Payload: []byte{0x00}})
	}()

	ackFrame, err := clientConn.ReadFrame()
	if err != nil {
		t.Fatalf("ACK ReadFrame failed: %v", err)
	}
	<-errCh
	if ackFrame.Type != FrameAck || ackFrame.Payload[0] != 0x00 {
		t.Fatal("ACK payload wrong")
	}

	t.Log("✓ SOVA protocol handshake + encrypted framing works")
}

// TestSOVAStream проверяет SOVAStream (net.Conn-совместимый byte-stream)
func TestSOVAStream(t *testing.T) {
	psk := "stream-test-psk"

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	var serverConn *SOVAConn
	done := make(chan struct{})

	go func() {
		serverConn, _ = ServerHandshake(server, psk)
		done <- struct{}{}
	}()

	clientConn, err := ClientHandshake(client, psk)
	if err != nil {
		t.Fatalf("ClientHandshake: %v", err)
	}
	<-done

	clientStream := NewSOVAStream(clientConn)
	serverStream := NewSOVAStream(serverConn)

	// Write through stream interface
	msg := []byte("SOVA autonomous protocol test data — this goes through encrypted frames")
	go func() {
		clientStream.Write(msg)
	}()

	buf := make([]byte, 4096)
	n, err := serverStream.Read(buf)
	if err != nil {
		t.Fatalf("Stream Read failed: %v", err)
	}
	if string(buf[:n]) != string(msg) {
		t.Fatalf("Stream data mismatch: got %q", buf[:n])
	}

	t.Log("✓ SOVAStream (net.Conn interface over encrypted frames) works")
}

// TestDeriveSOVASessionKey проверяет что ключи детерминистичны
func TestDeriveSOVASessionKey(t *testing.T) {
	psk := "test-key"
	salt1 := []byte("client-salt-1234")
	salt2 := []byte("server-salt-5678")

	key1 := DeriveSOVASessionKey(psk, salt1, salt2)
	key2 := DeriveSOVASessionKey(psk, salt1, salt2)

	if len(key1) != 32 {
		t.Fatalf("key length %d, expected 32", len(key1))
	}
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("same inputs produce different keys")
		}
	}

	// Different salt = different key
	key3 := DeriveSOVASessionKey(psk, salt1, []byte("different-salt!!"))
	same := true
	for i := range key1 {
		if key1[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different salts produce same key — broken!")
	}

	t.Log("✓ DeriveSOVASessionKey deterministic and salt-dependent")
}

// TestFragConn проверяет фрагментацию TCP пакетов
func TestFragConn(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	frag := NewFragConn(client, 3, 0) // 3 bytes per fragment, no jitter

	msg := []byte("Hello World!")
	done := make(chan struct{})

	go func() {
		frag.Write(msg)
		done <- struct{}{}
	}()

	// Read all data on server side
	buf := make([]byte, 1024)
	total := 0
	server.SetReadDeadline(time.Now().Add(2 * time.Second))
	for total < len(msg) {
		n, err := server.Read(buf[total:])
		if err != nil {
			break
		}
		total += n
	}

	<-done

	if string(buf[:total]) != string(msg) {
		t.Fatalf("fragmented data mismatch: got %q", buf[:total])
	}

	// Second write should NOT be fragmented
	msg2 := []byte("Second write")
	go func() {
		frag.Write(msg2)
	}()

	n, _ := server.Read(buf)
	if string(buf[:n]) != string(msg2) {
		t.Fatalf("second write was fragmented when it shouldn't be")
	}

	t.Log("✓ FragConn fragments first write, passes subsequent writes through")
}

// TestSelfSignedTLS проверяет генерацию самоподписанного TLS сертификата
func TestSelfSignedTLS(t *testing.T) {
	cfg, err := GenerateSelfSignedTLSConfig()
	if err != nil {
		t.Fatalf("GenerateSelfSignedTLSConfig failed: %v", err)
	}
	if len(cfg.Certificates) == 0 {
		t.Fatal("no certificates generated")
	}
	t.Log("✓ Self-signed TLS certificate generated successfully")
}
