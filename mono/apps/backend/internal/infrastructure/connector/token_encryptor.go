package connector

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// TokenEncryptor provides AES-256-GCM encryption for OAuth tokens.
type TokenEncryptor struct {
	currentKey   []byte
	currentKeyID string
	previousKey  []byte
	previousKeyID string
}

// NewTokenEncryptor creates a new token encryptor.
// The key must be 32 bytes (64 hex characters) for AES-256.
func NewTokenEncryptor(keyHex, keyID string, previousKeyHex, previousKeyID string) (*TokenEncryptor, error) {
	key, err := decodeHexKey(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid current encryption key: %w", err)
	}

	encryptor := &TokenEncryptor{
		currentKey:   key,
		currentKeyID: keyID,
	}

	// Previous key is optional (for key rotation)
	if previousKeyHex != "" {
		prevKey, err := decodeHexKey(previousKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid previous encryption key: %w", err)
		}
		encryptor.previousKey = prevKey
		encryptor.previousKeyID = previousKeyID
	}

	return encryptor, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext and the key ID used.
func (e *TokenEncryptor) Encrypt(plaintext string) (string, string, error) {
	if plaintext == "" {
		return "", "", errors.New("plaintext cannot be empty")
	}

	block, err := aes.NewCipher(e.currentKey)
	if err != nil {
		return "", "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, e.currentKeyID, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// The keyID parameter indicates which key was used for encryption.
func (e *TokenEncryptor) Decrypt(ciphertext string, keyID string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("ciphertext cannot be empty")
	}

	// Select the appropriate key
	var key []byte
	switch keyID {
	case e.currentKeyID:
		key = e.currentKey
	case e.previousKeyID:
		if e.previousKey == nil {
			return "", errors.New("previous key not configured")
		}
		key = e.previousKey
	default:
		return "", fmt.Errorf("unknown key ID: %s", keyID)
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// decodeHexKey decodes a hex string to bytes.
func decodeHexKey(hexKey string) ([]byte, error) {
	hexKey = strings.TrimSpace(hexKey)
	if len(hexKey) != 64 {
		return nil, fmt.Errorf("key must be 64 hex characters (32 bytes), got %d", len(hexKey))
	}

	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		var b byte
		_, err := fmt.Sscanf(hexKey[i*2:i*2+2], "%02x", &b)
		if err != nil {
			return nil, fmt.Errorf("invalid hex at position %d: %w", i*2, err)
		}
		key[i] = b
	}

	return key, nil
}
