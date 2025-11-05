package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"aistack/internal/logging"
)

func TestNewSecretStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)

	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Verify secrets directory was created
	if _, err := os.Stat(config.SecretsDir); os.IsNotExist(err) {
		t.Error("Secrets directory was not created")
	}

	// Verify passphrase file was created
	if _, err := os.Stat(config.PassphraseFile); os.IsNotExist(err) {
		t.Error("Passphrase file was not created")
	}

	// Verify passphrase file has correct permissions
	info, err := os.Stat(config.PassphraseFile)
	if err != nil {
		t.Fatalf("Failed to stat passphrase file: %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("Passphrase file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestSecretStore_StoreAndRetrieve(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	tests := []struct {
		name   string
		secret []byte
	}{
		{"simple secret", []byte("my-secret-value")},
		{"empty secret", []byte("")},
		{"binary secret", []byte{0x00, 0x01, 0x02, 0xff}},
		{"long secret", []byte("this is a very long secret with many characters to test storage and retrieval")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretName := "test-secret-" + tt.name

			// Store secret
			err := store.StoreSecret(secretName, tt.secret)
			if err != nil {
				t.Fatalf("StoreSecret() error = %v", err)
			}

			// Verify file exists
			secretPath := filepath.Join(config.SecretsDir, secretName+".enc")
			if _, err := os.Stat(secretPath); os.IsNotExist(err) {
				t.Error("Secret file was not created")
			}

			// Verify file permissions
			info, err := os.Stat(secretPath)
			if err != nil {
				t.Fatalf("Failed to stat secret file: %v", err)
			}

			if info.Mode().Perm() != 0o600 {
				t.Errorf("Secret file permissions = %o, want 0600", info.Mode().Perm())
			}

			// Retrieve secret
			retrieved, err := store.RetrieveSecret(secretName)
			if err != nil {
				t.Fatalf("RetrieveSecret() error = %v", err)
			}

			// Verify retrieved matches original
			if string(retrieved) != string(tt.secret) {
				t.Errorf("Retrieved secret = %q, want %q", retrieved, tt.secret)
			}
		})
	}
}

func TestSecretStore_RetrieveNonexistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	_, err = store.RetrieveSecret("nonexistent")
	if err == nil {
		t.Error("Expected error when retrieving nonexistent secret")
	}

	expectedError := "secret not found: nonexistent"
	if err.Error() != expectedError {
		t.Errorf("Error message = %q, want %q", err.Error(), expectedError)
	}
}

func TestSecretStore_DeleteSecret(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Store a secret
	secretName := "test-secret"
	if err := store.StoreSecret(secretName, []byte("value")); err != nil {
		t.Fatalf("StoreSecret() error = %v", err)
	}

	// Delete it
	if err := store.DeleteSecret(secretName); err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	// Verify it's gone
	secretPath := filepath.Join(config.SecretsDir, secretName+".enc")
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Error("Secret file should not exist after deletion")
	}

	// Try to retrieve deleted secret
	_, err = store.RetrieveSecret(secretName)
	if err == nil {
		t.Error("Expected error when retrieving deleted secret")
	}
}

func TestSecretStore_DeleteNonexistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	err = store.DeleteSecret("nonexistent")
	if err == nil {
		t.Error("Expected error when deleting nonexistent secret")
	}
}

func TestSecretStore_ListSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Initially empty
	secrets, err := store.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(secrets) != 0 {
		t.Errorf("Expected 0 secrets, got %d", len(secrets))
	}

	// Store some secrets
	secretNames := []string{"secret1", "secret2", "secret3"}
	for _, name := range secretNames {
		if err := store.StoreSecret(name, []byte("value")); err != nil {
			t.Fatalf("StoreSecret() error = %v", err)
		}
	}

	// List should show all secrets
	secrets, err = store.ListSecrets()
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(secrets) != len(secretNames) {
		t.Errorf("Expected %d secrets, got %d", len(secretNames), len(secrets))
	}

	// Verify all names present
	secretMap := make(map[string]bool)
	for _, name := range secrets {
		secretMap[name] = true
	}

	for _, expected := range secretNames {
		if !secretMap[expected] {
			t.Errorf("Expected secret %q in list", expected)
		}
	}
}

func TestSecretStore_Index(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Store a secret
	secretName := "test-secret"
	if err := store.StoreSecret(secretName, []byte("value1")); err != nil {
		t.Fatalf("StoreSecret() error = %v", err)
	}

	// Load index
	index, err := store.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex() error = %v", err)
	}

	// Verify entry exists
	if len(index.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(index.Entries))
	}

	if index.Entries[0].Name != secretName {
		t.Errorf("Entry name = %q, want %q", index.Entries[0].Name, secretName)
	}

	// Update the secret (should update last_rotated)
	oldRotated := index.Entries[0].LastRotated

	if err := store.StoreSecret(secretName, []byte("value2")); err != nil {
		t.Fatalf("StoreSecret() error = %v", err)
	}

	// Load index again
	index, err = store.loadIndex()
	if err != nil {
		t.Fatalf("loadIndex() error = %v", err)
	}

	// Still one entry
	if len(index.Entries) != 1 {
		t.Errorf("Expected 1 entry after update, got %d", len(index.Entries))
	}

	// LastRotated should be updated
	newRotated := index.Entries[0].LastRotated
	if !newRotated.After(oldRotated) {
		t.Error("LastRotated should be updated")
	}
}

func TestSecretStore_PermissionsVerification(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)
	store, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Store a secret
	secretName := "test-secret"
	if err := store.StoreSecret(secretName, []byte("value")); err != nil {
		t.Fatalf("StoreSecret() error = %v", err)
	}

	// Verify permissions are correct
	secretPath := filepath.Join(config.SecretsDir, secretName+".enc")
	if err := store.verifyPermissions(secretPath); err != nil {
		t.Errorf("verifyPermissions() error = %v", err)
	}

	// Change permissions to something wrong
	if err := os.Chmod(secretPath, 0o644); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}

	// Verification should fail
	if err := store.verifyPermissions(secretPath); err == nil {
		t.Error("Expected error for wrong permissions")
	}
}

func TestSecretStore_PersistentPassphrase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aistack-secrets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := SecretStoreConfig{
		SecretsDir:     filepath.Join(tmpDir, "secrets"),
		PassphraseFile: filepath.Join(tmpDir, ".passphrase"),
	}

	logger := logging.NewLogger(logging.LevelInfo)

	// Create first store (generates passphrase)
	store1, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Store a secret
	secretName := "test-secret"
	secretValue := []byte("my-secret")
	if err := store1.StoreSecret(secretName, secretValue); err != nil {
		t.Fatalf("StoreSecret() error = %v", err)
	}

	// Create second store (should reuse existing passphrase)
	store2, err := NewSecretStore(config, logger)
	if err != nil {
		t.Fatalf("NewSecretStore() error = %v", err)
	}

	// Should be able to retrieve the secret
	retrieved, err := store2.RetrieveSecret(secretName)
	if err != nil {
		t.Fatalf("RetrieveSecret() error = %v", err)
	}

	if string(retrieved) != string(secretValue) {
		t.Errorf("Retrieved = %q, want %q", retrieved, secretValue)
	}
}
