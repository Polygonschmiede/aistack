package fsutil

import (
	"fmt"
	"os"
	"path/filepath"

	"aistack/internal/logging"
)

const (
	// DefaultStateDir is the default location for aistack state files
	DefaultStateDir = "/var/lib/aistack"
	// DefaultStatePermissions is the default permission for state directories
	DefaultStatePermissions = 0o750
	// DefaultFilePermissions is the default permission for state files
	DefaultFilePermissions = 0o600
)

// GetStateDir returns the state directory from environment or uses the provided default.
// It returns an absolute path when possible.
func GetStateDir(defaultDir string) string {
	if env := os.Getenv("AISTACK_STATE_DIR"); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs
		}
		return env
	}
	return defaultDir
}

// EnsureStateDirectory creates the state directory if it doesn't exist.
// It uses DefaultStatePermissions (0o750) for the directory.
func EnsureStateDirectory(path string) error {
	if err := os.MkdirAll(path, DefaultStatePermissions); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	return nil
}

// AtomicWriteFile writes data to a file atomically by first writing to a temp file
// and then renaming it to the target path. This ensures the file is never partially written.
func AtomicWriteFile(path string, data []byte, perm os.FileMode, logger *logging.Logger) error {
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Try to clean up temp file on failure
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			if logger != nil {
				logger.Warn("cleanup_failed", "Failed to remove temp file",
					"path", tmpPath,
					"error", removeErr.Error())
			}
		}
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// CloseWithError closes a resource and logs any error if a logger is provided.
// This is useful for defer statements where close errors should be handled.
func CloseWithError(closer func() error, logger *logging.Logger, resource string) {
	if err := closer(); err != nil {
		if logger != nil {
			logger.Warn("close_failed", fmt.Sprintf("Failed to close %s", resource), "error", err.Error())
		}
	}
}
