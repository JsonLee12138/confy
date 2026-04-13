package confy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedPrefix = "enc:AES_GCM:"

// encryptAESGCM encrypts plaintext using AES-256-GCM.
func encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("confy: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("confy: failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("confy: failed to generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptAESGCM decrypts ciphertext using AES-256-GCM.
func decryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("confy: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("confy: failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("confy: ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("confy: decryption failed: %w", err)
	}

	return plaintext, nil
}

// encryptToBase64 encrypts a string and returns base64-encoded ciphertext.
func encryptToBase64(plaintext string, key []byte) (string, error) {
	ciphertext, err := encryptAESGCM([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptFromBase64 decodes base64 and decrypts to plaintext string.
func decryptFromBase64(encoded string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("confy: failed to decode base64: %w", err)
	}

	plaintext, err := decryptAESGCM(ciphertext, key)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// isEncryptedValue checks if a string value is an encrypted config value.
func isEncryptedValue(val string) bool {
	return strings.HasPrefix(val, encryptedPrefix) && len(val) > len(encryptedPrefix)
}

// decryptConfigValue decrypts a value with the "enc:AES_GCM:" prefix.
func decryptConfigValue(val string, key []byte) (string, error) {
	if !isEncryptedValue(val) {
		return val, nil
	}
	encoded := val[len(encryptedPrefix):]
	return decryptFromBase64(encoded, key)
}

// encryptConfigValue encrypts a value and adds the "enc:AES_GCM:" prefix.
func encryptConfigValue(val string, key []byte) (string, error) {
	encoded, err := encryptToBase64(val, key)
	if err != nil {
		return "", err
	}
	return encryptedPrefix + encoded, nil
}
