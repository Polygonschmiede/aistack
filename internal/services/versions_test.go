package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionLock_Resolve_NoLock(t *testing.T) {
	var lock *VersionLock
	ref, err := lock.Resolve("ollama", "ollama/ollama:latest")
	if err != nil {
		t.Fatalf("Resolve() with nil lock should not error: %v", err)
	}

	if ref.PullRef != "ollama/ollama:latest" {
		t.Errorf("PullRef = %s, want ollama/ollama:latest", ref.PullRef)
	}
	if ref.TagRef != "ollama/ollama:latest" {
		t.Errorf("TagRef = %s, want ollama/ollama:latest", ref.TagRef)
	}
}

func TestVersionLock_Resolve_WithTag(t *testing.T) {
	lock := &VersionLock{
		entries: map[string]string{
			"ollama": "ollama/ollama:v0.1.0",
		},
		path: "/test/versions.lock",
	}

	ref, err := lock.Resolve("ollama", "ollama/ollama:latest")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if ref.PullRef != "ollama/ollama:v0.1.0" {
		t.Errorf("PullRef = %s, want ollama/ollama:v0.1.0", ref.PullRef)
	}
	if ref.TagRef != "ollama/ollama:latest" {
		t.Errorf("TagRef = %s, want ollama/ollama:latest", ref.TagRef)
	}
}

func TestVersionLock_Resolve_WithDigest(t *testing.T) {
	lock := &VersionLock{
		entries: map[string]string{
			"ollama": "ollama/ollama@sha256:abc123def456",
		},
		path: "/test/versions.lock",
	}

	ref, err := lock.Resolve("ollama", "ollama/ollama:latest")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if ref.PullRef != "ollama/ollama@sha256:abc123def456" {
		t.Errorf("PullRef = %s, want ollama/ollama@sha256:abc123def456", ref.PullRef)
	}
	if ref.TagRef != "ollama/ollama:latest" {
		t.Errorf("TagRef = %s, want ollama/ollama:latest", ref.TagRef)
	}
}

func TestVersionLock_Resolve_ServiceNotInLock(t *testing.T) {
	lock := &VersionLock{
		entries: map[string]string{
			"ollama": "ollama/ollama:v0.1.0",
		},
		path: "/test/versions.lock",
	}

	// Service not in lock should fall back to default
	ref, err := lock.Resolve("localai", "quay.io/go-skynet/local-ai:latest")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if ref.PullRef != "quay.io/go-skynet/local-ai:latest" {
		t.Errorf("PullRef = %s, want quay.io/go-skynet/local-ai:latest", ref.PullRef)
	}
	if ref.TagRef != "quay.io/go-skynet/local-ai:latest" {
		t.Errorf("TagRef = %s, want quay.io/go-skynet/local-ai:latest", ref.TagRef)
	}
}

func TestVersionLock_Resolve_EmptyEntry(t *testing.T) {
	lock := &VersionLock{
		entries: map[string]string{
			"ollama": "   ", // Empty after trim
		},
		path: "/test/versions.lock",
	}

	_, err := lock.Resolve("ollama", "ollama/ollama:latest")
	if err == nil {
		t.Error("Resolve() with empty entry should return error")
	}

	if err != nil && err.Error() != "versions.lock entry for ollama is empty (file: /test/versions.lock)" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLoadVersionLock_NoFile(t *testing.T) {
	// Create temp dir without versions.lock
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	lock, err := loadVersionLock()
	if err != nil {
		t.Fatalf("loadVersionLock() with no file should not error: %v", err)
	}
	if lock != nil {
		t.Error("loadVersionLock() with no file should return nil lock")
	}
}

func TestLoadVersionLock_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "versions.lock")

	content := `# Version lock file
ollama:ollama/ollama:v0.1.0
openwebui:ghcr.io/open-webui/open-webui@sha256:abc123
localai:quay.io/go-skynet/local-ai:latest

# Comment line
`
	if err := os.WriteFile(lockPath, []byte(content), 0o640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set environment variable to use our test file
	os.Setenv("AISTACK_VERSIONS_LOCK", lockPath)
	defer os.Unsetenv("AISTACK_VERSIONS_LOCK")

	lock, err := loadVersionLock()
	if err != nil {
		t.Fatalf("loadVersionLock() error = %v", err)
	}
	if lock == nil {
		t.Fatal("loadVersionLock() returned nil lock")
	}

	if len(lock.entries) != 3 {
		t.Errorf("lock.entries length = %d, want 3", len(lock.entries))
	}

	// Test entries
	if lock.entries["ollama"] != "ollama/ollama:v0.1.0" {
		t.Errorf("ollama entry = %s, want ollama/ollama:v0.1.0", lock.entries["ollama"])
	}
	if lock.entries["openwebui"] != "ghcr.io/open-webui/open-webui@sha256:abc123" {
		t.Errorf("openwebui entry = %s, want ghcr.io/open-webui/open-webui@sha256:abc123", lock.entries["openwebui"])
	}
	if lock.entries["localai"] != "quay.io/go-skynet/local-ai:latest" {
		t.Errorf("localai entry = %s, want quay.io/go-skynet/local-ai:latest", lock.entries["localai"])
	}
}

func TestLoadVersionLock_InvalidFormat(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing colon separator",
			content: "ollama-service ollama_image_latest\n",
			wantErr: "invalid versions.lock entry on line 1",
		},
		{
			name:    "empty service name",
			content: ":ollama/ollama:latest\n",
			wantErr: "invalid versions.lock entry on line 1",
		},
		{
			name:    "empty reference",
			content: "ollama:\n",
			wantErr: "invalid versions.lock entry on line 1",
		},
		{
			name:    "empty reference with spaces",
			content: "ollama:   \n",
			wantErr: "invalid versions.lock entry on line 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			lockPath := filepath.Join(tmpDir, "versions.lock")

			if err := os.WriteFile(lockPath, []byte(tt.content), 0o640); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			os.Setenv("AISTACK_VERSIONS_LOCK", lockPath)
			defer os.Unsetenv("AISTACK_VERSIONS_LOCK")

			_, err := loadVersionLock()
			if err == nil {
				t.Error("loadVersionLock() should return error for invalid format")
			}

			if err != nil && !containsSubstring(err.Error(), tt.wantErr) {
				t.Errorf("Error message = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLoadVersionLock_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "versions.lock")

	// Create empty file
	if err := os.WriteFile(lockPath, []byte(""), 0o640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	os.Setenv("AISTACK_VERSIONS_LOCK", lockPath)
	defer os.Unsetenv("AISTACK_VERSIONS_LOCK")

	lock, err := loadVersionLock()
	if err != nil {
		t.Fatalf("loadVersionLock() with empty file should not error: %v", err)
	}
	if lock == nil {
		t.Fatal("loadVersionLock() returned nil lock")
	}
	if len(lock.entries) != 0 {
		t.Errorf("lock.entries length = %d, want 0", len(lock.entries))
	}
}

func TestLoadVersionLock_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "versions.lock")

	content := `# Comment line 1
# Comment line 2
# Comment line 3
`
	if err := os.WriteFile(lockPath, []byte(content), 0o640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	os.Setenv("AISTACK_VERSIONS_LOCK", lockPath)
	defer os.Unsetenv("AISTACK_VERSIONS_LOCK")

	lock, err := loadVersionLock()
	if err != nil {
		t.Fatalf("loadVersionLock() with only comments should not error: %v", err)
	}
	if lock == nil {
		t.Fatal("loadVersionLock() returned nil lock")
	}
	if len(lock.entries) != 0 {
		t.Errorf("lock.entries length = %d, want 0", len(lock.entries))
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test existing file
	existingFile := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0o640); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if !fileExists(existingFile) {
		t.Error("fileExists() returned false for existing file")
	}

	// Test non-existing file
	nonExisting := filepath.Join(tmpDir, "nonexisting.txt")
	if fileExists(nonExisting) {
		t.Error("fileExists() returned true for non-existing file")
	}

	// Test directory (should return false)
	if fileExists(tmpDir) {
		t.Error("fileExists() returned true for directory")
	}
}

// Helper function
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 && findSubstring(str, substr)))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
