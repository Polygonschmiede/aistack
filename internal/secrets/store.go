package secrets

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

// SecretStore handles encrypted secret storage
// Story T-030: Lokale Secret-Verschlüsselung mit libsodium (NaCl secretbox)
type SecretStore struct {
	config SecretStoreConfig
	key    *[KeySize]byte
	logger *logging.Logger
}

// NewSecretStore creates a new secret store
func NewSecretStore(config SecretStoreConfig, logger *logging.Logger) (*SecretStore, error) {
	// Ensure secrets directory exists with proper permissions
	if err := os.MkdirAll(config.SecretsDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Load or generate passphrase
	passphrase, err := loadOrGeneratePassphrase(config.PassphraseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load passphrase: %w", err)
	}

	// Derive encryption key
	key := DeriveKey(passphrase)

	return &SecretStore{
		config: config,
		key:    &key,
		logger: logger,
	}, nil
}

// StoreSecret stores an encrypted secret
func (s *SecretStore) StoreSecret(name string, value []byte) error {
	// Encrypt the secret
	encrypted, err := Encrypt(value, s.key)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Write encrypted secret with permissions 600
	secretPath := filepath.Join(s.config.SecretsDir, name+".enc")
	if err := os.WriteFile(secretPath, encrypted, 0o600); err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}

	// Verify permissions
	if err := s.verifyPermissions(secretPath); err != nil {
		s.logger.Warn("secrets.permissions.invalid", "Secret file has incorrect permissions", map[string]interface{}{
			"path":  secretPath,
			"error": err.Error(),
		})
	}

	// Update index
	if err := s.updateIndex(name); err != nil {
		s.logger.Warn("secrets.index.update_failed", "Failed to update secrets index", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
	}

	s.logger.Info("secrets.stored", "Secret stored successfully", map[string]interface{}{
		"name": name,
	})

	return nil
}

// RetrieveSecret retrieves and decrypts a secret
func (s *SecretStore) RetrieveSecret(name string) ([]byte, error) {
	secretPath := filepath.Join(s.config.SecretsDir, name+".enc")

	// Read encrypted secret
	encrypted, err := os.ReadFile(secretPath) // #nosec G304 -- path is constructed from controlled secrets dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret not found: %s", name)
		}
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}

	// Verify permissions before decrypting
	if err := s.verifyPermissions(secretPath); err != nil {
		s.logger.Warn("secrets.permissions.warning", "Secret file permissions should be 600", map[string]interface{}{
			"path": secretPath,
		})
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, s.key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	s.logger.Debug("secrets.retrieved", "Secret retrieved successfully", map[string]interface{}{
		"name": name,
	})

	return decrypted, nil
}

// DeleteSecret removes a secret
func (s *SecretStore) DeleteSecret(name string) error {
	secretPath := filepath.Join(s.config.SecretsDir, name+".enc")

	if err := os.Remove(secretPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("secret not found: %s", name)
		}
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Update index
	if err := s.removeFromIndex(name); err != nil {
		s.logger.Warn("secrets.index.remove_failed", "Failed to remove from secrets index", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
	}

	s.logger.Info("secrets.deleted", "Secret deleted successfully", map[string]interface{}{
		"name": name,
	})

	return nil
}

// ListSecrets returns a list of all stored secrets
func (s *SecretStore) ListSecrets() ([]string, error) {
	index, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(index.Entries))
	for i, entry := range index.Entries {
		names[i] = entry.Name
	}

	return names, nil
}

// verifyPermissions checks if file has correct permissions (600)
func (s *SecretStore) verifyPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0o600)

	// Check if permissions are exactly 600
	if mode.Perm() != expectedMode {
		return fmt.Errorf("file has permissions %o, expected %o", mode.Perm(), expectedMode)
	}

	return nil
}

// updateIndex updates the secrets index
func (s *SecretStore) updateIndex(name string) error {
	index, err := s.loadIndex()
	if err != nil {
		index = &SecretIndex{Entries: []SecretEntry{}}
	}

	// Check if entry exists
	found := false
	for i, entry := range index.Entries {
		if entry.Name == name {
			index.Entries[i].LastRotated = time.Now().UTC()
			found = true
			break
		}
	}

	// Add new entry if not found
	if !found {
		index.Entries = append(index.Entries, SecretEntry{
			Name:        name,
			LastRotated: time.Now().UTC(),
		})
	}

	return s.saveIndex(index)
}

// removeFromIndex removes an entry from the secrets index
func (s *SecretStore) removeFromIndex(name string) error {
	index, err := s.loadIndex()
	if err != nil {
		return err
	}

	// Filter out the entry
	filtered := make([]SecretEntry, 0, len(index.Entries))
	for _, entry := range index.Entries {
		if entry.Name != name {
			filtered = append(filtered, entry)
		}
	}

	index.Entries = filtered
	return s.saveIndex(index)
}

// loadIndex loads the secrets index
func (s *SecretStore) loadIndex() (*SecretIndex, error) {
	indexPath := filepath.Join(s.config.SecretsDir, "secrets_index.json")

	data, err := os.ReadFile(indexPath) // #nosec G304 -- path is constructed from controlled secrets dir
	if err != nil {
		if os.IsNotExist(err) {
			return &SecretIndex{Entries: []SecretEntry{}}, nil
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	var index SecretIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	return &index, nil
}

// saveIndex saves the secrets index
func (s *SecretStore) saveIndex(index *SecretIndex) error {
	indexPath := filepath.Join(s.config.SecretsDir, "secrets_index.json")

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	return nil
}

// loadOrGeneratePassphrase loads passphrase from file or generates a new one
func loadOrGeneratePassphrase(path string) (string, error) {
	// Try to read existing passphrase
	data, err := os.ReadFile(path) // #nosec G304 -- path is from config
	if err == nil {
		return string(data), nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}

	// Generate new passphrase
	passphrase, err := generatePassphrase()
	if err != nil {
		return "", fmt.Errorf("failed to generate passphrase: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create passphrase directory: %w", err)
	}

	// Write passphrase with strict permissions
	if err := os.WriteFile(path, []byte(passphrase), 0o600); err != nil {
		return "", fmt.Errorf("failed to write passphrase: %w", err)
	}

	return passphrase, nil
}

// generatePassphrase generates a random passphrase
func generatePassphrase() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to hex string
	return fmt.Sprintf("%x", bytes), nil
}
