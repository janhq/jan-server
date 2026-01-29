package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptString encrypts plaintext using AES-GCM with the given secret key
func EncryptString(secret, plaintext string) (string, error) {
	if secret == "" {
		return "", errors.New("secret key cannot be empty")
	}

	// Ensure the key is 32 bytes for AES-256
	key := []byte(secret)
	if len(key) < 32 {
		// Pad the key to 32 bytes
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		// Truncate the key to 32 bytes
		key = key[:32]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts ciphertext using AES-GCM with the given secret key
func DecryptString(secret, ciphertext string) (string, error) {
	if secret == "" {
		return "", errors.New("secret key cannot be empty")
	}

	// Ensure the key is 32 bytes for AES-256
	key := []byte(secret)
	if len(key) < 32 {
		// Pad the key to 32 bytes
		paddedKey := make([]byte, 32)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 32 {
		// Truncate the key to 32 bytes
		key = key[:32]
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext2 := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext2, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
