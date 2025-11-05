package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

const (
	// KeySize is the size of the encryption key (32 bytes for NaCl secretbox)
	KeySize = 32
	// NonceSize is the size of the nonce (24 bytes for NaCl secretbox)
	NonceSize = 24
)

// DeriveKey derives a 32-byte key from a passphrase using SHA-256
// Story T-030: Simple key derivation for secretbox encryption
func DeriveKey(passphrase string) [KeySize]byte {
	hash := sha256.Sum256([]byte(passphrase))
	return hash
}

// Encrypt encrypts data using NaCl secretbox (authenticated encryption)
// Returns: nonce + ciphertext (nonce is prepended to ciphertext)
func Encrypt(plaintext []byte, key *[KeySize]byte) ([]byte, error) {
	// Generate random nonce
	var nonce [NonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with secretbox (authenticated encryption)
	encrypted := secretbox.Seal(nonce[:], plaintext, &nonce, key)

	return encrypted, nil
}

// Decrypt decrypts data encrypted with Encrypt
// Input: nonce + ciphertext (nonce must be prepended)
func Decrypt(encrypted []byte, key *[KeySize]byte) ([]byte, error) {
	if len(encrypted) < NonceSize {
		return nil, fmt.Errorf("encrypted data too short (minimum %d bytes)", NonceSize)
	}

	// Extract nonce from beginning
	var nonce [NonceSize]byte
	copy(nonce[:], encrypted[:NonceSize])

	// Decrypt
	decrypted, ok := secretbox.Open(nil, encrypted[NonceSize:], &nonce, key)
	if !ok {
		return nil, fmt.Errorf("decryption failed (wrong key or corrupted data)")
	}

	return decrypted, nil
}
