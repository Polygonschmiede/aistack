package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"aistack/internal/logging"
)

func TestGetStateDir(t *testing.T) {
	tests := []struct {
		name       string
		envValue   string
		defaultDir string
		wantEnv    bool
	}{
		{
			name:       "uses environment variable",
			envValue:   "/custom/state",
			defaultDir: "/default/state",
			wantEnv:    true,
		},
		{
			name:       "uses default when env not set",
			envValue:   "",
			defaultDir: "/default/state",
			wantEnv:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			origEnv := os.Getenv("AISTACK_STATE_DIR")
			defer func() {
				if origEnv != "" {
					_ = os.Setenv("AISTACK_STATE_DIR", origEnv)
				} else {
					_ = os.Unsetenv("AISTACK_STATE_DIR")
				}
			}()

			// Set test env
			if tt.envValue != "" {
				_ = os.Setenv("AISTACK_STATE_DIR", tt.envValue)
			} else {
				_ = os.Unsetenv("AISTACK_STATE_DIR")
			}

			got := GetStateDir(tt.defaultDir)

			if tt.wantEnv && got == tt.defaultDir {
				t.Errorf("GetStateDir() should use env value, got default %v", got)
			}

			if !tt.wantEnv && got != tt.defaultDir {
				t.Errorf("GetStateDir() = %v, want %v", got, tt.defaultDir)
			}
		})
	}
}

func TestEnsureStateDirectory(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "creates new directory",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "newdir")
			},
			wantErr: false,
		},
		{
			name: "succeeds if directory exists",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := filepath.Join(t.TempDir(), "existingdir")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
				return dir
			},
			wantErr: false,
		},
		{
			name: "creates nested directories",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "a", "b", "c")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)

			err := EnsureStateDirectory(path)

			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureStateDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify directory was created
				info, err := os.Stat(path)
				if err != nil {
					t.Errorf("directory not created: %v", err)
					return
				}
				if !info.IsDir() {
					t.Errorf("path is not a directory")
				}
			}
		})
	}
}

func TestAtomicWriteFile(t *testing.T) {
	t.Helper()

	logger := logging.New("test", logging.LevelWarn)

	tests := []struct {
		name    string
		setup   func(t *testing.T) (string, []byte)
		wantErr bool
	}{
		{
			name: "writes new file atomically",
			setup: func(t *testing.T) (string, []byte) {
				t.Helper()
				path := filepath.Join(t.TempDir(), "test.txt")
				return path, []byte("test content")
			},
			wantErr: false,
		},
		{
			name: "overwrites existing file",
			setup: func(t *testing.T) (string, []byte) {
				t.Helper()
				path := filepath.Join(t.TempDir(), "existing.txt")
				_ = os.WriteFile(path, []byte("old content"), 0o600)
				return path, []byte("new content")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, data := tt.setup(t)

			err := AtomicWriteFile(path, data, DefaultFilePermissions, logger)

			if (err != nil) != tt.wantErr {
				t.Errorf("AtomicWriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file contents
				got, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("failed to read file: %v", err)
					return
				}
				if string(got) != string(data) {
					t.Errorf("file content = %q, want %q", got, data)
				}

				// Verify temp file was cleaned up
				tmpPath := path + ".tmp"
				if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
					t.Errorf("temp file still exists: %s", tmpPath)
				}
			}
		})
	}
}

func TestCloseWithError(t *testing.T) {
	t.Helper()

	logger := logging.New("test", logging.LevelWarn)

	tests := []struct {
		name     string
		closer   func() error
		hasError bool
	}{
		{
			name:     "successful close",
			closer:   func() error { return nil },
			hasError: false,
		},
		{
			name:     "close with error",
			closer:   func() error { return os.ErrClosed },
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			CloseWithError(tt.closer, logger, "test_resource")
			CloseWithError(tt.closer, nil, "test_resource")
		})
	}
}
