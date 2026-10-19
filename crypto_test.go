package confy

import (
	"crypto/rand"
	"testing"
	"time"
)

func TestEncryptDecrypt_AESGCM(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := []byte("super_secret_password")

	ciphertext, err := encryptAESGCM(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := decryptAESGCM(ciphertext, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestDecryptAESGCM_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	ciphertext, _ := encryptAESGCM([]byte("secret"), key1)

	_, err := decryptAESGCM(ciphertext, key2)
	if err == nil {
		t.Error("expected error for wrong key")
	}
}

func TestEncryptDecrypt_Base64(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := "my_database_password"

	encoded, err := encryptToBase64(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := decryptFromBase64(encoded, key)
	if err != nil {
		t.Fatal(err)
	}

	if decoded != plaintext {
		t.Errorf("expected '%s', got '%s'", plaintext, decoded)
	}
}

func TestIsEncryptedValue(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"enc:AES_GCM:abc123", true},
		{"plain_value", false},
		{"enc:", false},
		{"ENC:AES_GCM:abc", false},
	}

	for _, tt := range tests {
		if got := isEncryptedValue(tt.input); got != tt.expected {
			t.Errorf("isEncryptedValue(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestBindEncryptedViewTime(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encrypted, err := encryptConfigValue("plain", key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "started_at: '2026-10-19T06:00:00Z'\npassword: '"+encrypted+"'\n")
	cfg, err := New(WithPath(dir), WithEncryption("aes-gcm", key))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var target struct {
		StartedAt time.Time `mapstructure:"started_at"`
		Password  string    `mapstructure:"password"`
	}
	if err := cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target.StartedAt.IsZero() || target.Password != "plain" {
		t.Fatalf("binding = %+v", target)
	}
	if cfg.Get("password") != encrypted {
		t.Fatal("decryption changed source")
	}
}
