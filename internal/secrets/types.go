package secrets

import "time"

// SecretIndex tracks stored secrets metadata
// Story T-030: Lokale Secret-Verschlüsselung (libsodium)
type SecretIndex struct {
	Entries []SecretEntry `json:"entries"`
}

// SecretEntry represents metadata for a stored secret
type SecretEntry struct {
	Name        string    `json:"name"`
	LastRotated time.Time `json:"last_rotated"`
}

// SecretStoreConfig holds configuration for the secret store
type SecretStoreConfig struct {
	SecretsDir     string
	PassphraseFile string
}

// DefaultSecretStoreConfig returns default configuration
func DefaultSecretStoreConfig() SecretStoreConfig {
	return SecretStoreConfig{
		SecretsDir:     "/var/lib/aistack/secrets",
		PassphraseFile: "/var/lib/aistack/.passphrase",
	}
}
