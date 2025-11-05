package secrets

import (
	"bytes"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
	}{
		{"simple passphrase", "test123"},
		{"complex passphrase", "My$ecretP@ssw0rd!"},
		{"empty passphrase", ""},
		{"long passphrase", "this is a very long passphrase with many characters to test the key derivation function"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := DeriveKey(tt.passphrase)

			// Verify key size
			if len(key) != KeySize {
				t.Errorf("Key size = %d, want %d", len(key), KeySize)
			}

			// Same passphrase should produce same key
			key2 := DeriveKey(tt.passphrase)
			if key != key2 {
				t.Error("Same passphrase produced different keys")
			}

			// Different passphrase should produce different key
			if tt.passphrase != "" {
				key3 := DeriveKey(tt.passphrase + "x")
				if key == key3 {
					t.Error("Different passphrases produced same key")
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := DeriveKey("test-passphrase")

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple text", []byte("hello world")},
		{"empty", []byte("")},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}},
		{"long text", []byte("this is a much longer text that should still be encrypted and decrypted correctly without any issues")},
		{"special chars", []byte("!@#$%^&*()_+-={}[]|\\:\";<>?,./")},
		{"unicode", []byte("Hello 世界 🌍")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(tt.plaintext, &key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Verify encrypted is longer (nonce + ciphertext + overhead)
			if len(encrypted) <= len(tt.plaintext) {
				t.Errorf("Encrypted data should be longer than plaintext")
			}

			// Verify encrypted is different from plaintext
			if len(tt.plaintext) > 0 && bytes.Equal(encrypted[NonceSize:], tt.plaintext) {
				t.Error("Encrypted data should not equal plaintext")
			}

			// Decrypt
			decrypted, err := Decrypt(encrypted, &key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Verify decrypted matches original
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncrypt_RandomNonce(t *testing.T) {
	key := DeriveKey("test-passphrase")
	plaintext := []byte("test data")

	// Encrypt same data twice
	encrypted1, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	encrypted2, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Verify different nonces were used (encrypted data should differ)
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("Same plaintext encrypted twice should produce different ciphertext (different nonces)")
	}

	// Both should decrypt to same plaintext
	decrypted1, _ := Decrypt(encrypted1, &key)
	decrypted2, _ := Decrypt(encrypted2, &key)

	if !bytes.Equal(decrypted1, plaintext) || !bytes.Equal(decrypted2, plaintext) {
		t.Error("Both encrypted versions should decrypt to original plaintext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := DeriveKey("correct-passphrase")
	key2 := DeriveKey("wrong-passphrase")

	plaintext := []byte("secret data")

	encrypted, err := Encrypt(plaintext, &key1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with wrong key
	_, err = Decrypt(encrypted, &key2)
	if err == nil {
		t.Error("Decrypt() with wrong key should fail")
	}

	expectedError := "decryption failed (wrong key or corrupted data)"
	if err.Error() != expectedError {
		t.Errorf("Error message = %q, want %q", err.Error(), expectedError)
	}
}

func TestDecrypt_CorruptedData(t *testing.T) {
	key := DeriveKey("test-passphrase")
	plaintext := []byte("test data")

	encrypted, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Corrupt the ciphertext
	corrupted := make([]byte, len(encrypted))
	copy(corrupted, encrypted)
	corrupted[NonceSize+5] ^= 0xFF // Flip bits in ciphertext

	// Try to decrypt corrupted data
	_, err = Decrypt(corrupted, &key)
	if err == nil {
		t.Error("Decrypt() with corrupted data should fail")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := DeriveKey("test-passphrase")

	// Data shorter than nonce size
	tooShort := []byte("short")

	_, err := Decrypt(tooShort, &key)
	if err == nil {
		t.Error("Decrypt() with data shorter than nonce size should fail")
	}

	if err.Error() != "encrypted data too short (minimum 24 bytes)" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestEncryptDecrypt_LargeData(t *testing.T) {
	key := DeriveKey("test-passphrase")

	// Create 1MB of data
	plaintext := make([]byte, 1024*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(plaintext, &key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	decrypted, err := Decrypt(encrypted, &key)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Large data not decrypted correctly")
	}
}
