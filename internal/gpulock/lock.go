package gpulock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/logging"
)

const (
	// LockFileName is the name of the GPU lock file
	LockFileName = "gpu_lock.json"

	// DefaultLeaseTimeout is the default lease timeout duration
	// After this duration, a stale lock can be considered expired
	DefaultLeaseTimeout = 5 * time.Minute
)

// Manager manages GPU lock acquisition and release
type Manager struct {
	stateDir     string
	logger       *logging.Logger
	leaseTimeout time.Duration
}

// NewManager creates a new GPU lock manager
func NewManager(stateDir string, logger *logging.Logger) *Manager {
	return &Manager{
		stateDir:     stateDir,
		logger:       logger,
		leaseTimeout: DefaultLeaseTimeout,
	}
}

// getLockPath returns the full path to the lock file
func (m *Manager) getLockPath() string {
	return filepath.Join(m.stateDir, LockFileName)
}

// Acquire attempts to acquire the GPU lock for the given holder
// Returns error if lock is held by another service
func (m *Manager) Acquire(holder Holder) error {
	if !holder.IsValid() {
		return fmt.Errorf("invalid holder: %s", holder)
	}

	if holder == HolderNone {
		return fmt.Errorf("cannot acquire lock for HolderNone")
	}

	// Check if lock already exists
	existingLock, err := m.loadLock()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing lock: %w", err)
	}

	// If lock exists, check if it's held by someone else
	if existingLock != nil {
		// If same holder, lock already acquired
		if existingLock.Holder == holder {
			m.logger.Info("gpu.lock.already_held", "GPU lock already held by this service", map[string]interface{}{
				"holder": holder.String(),
			})
			return nil
		}

		// Check if lock is stale
		if time.Since(existingLock.SinceTS) > m.leaseTimeout {
			m.logger.Warn("gpu.lock.stale_detected", "Stale GPU lock detected", map[string]interface{}{
				"current_holder": existingLock.Holder.String(),
				"age_seconds":    time.Since(existingLock.SinceTS).Seconds(),
			})

			// Automatically clear stale lock
			if err := m.forceUnlock(); err != nil {
				return fmt.Errorf("failed to clear stale lock: %w", err)
			}
		} else {
			// Lock is held by another service and not stale
			return fmt.Errorf("GPU lock is held by %s (acquired %s ago)",
				existingLock.Holder.String(),
				time.Since(existingLock.SinceTS).Round(time.Second))
		}
	}

	// Acquire lock
	newLock := &LockInfo{
		Holder:  holder,
		SinceTS: time.Now().UTC(),
	}

	if err := m.saveLock(newLock); err != nil {
		return fmt.Errorf("failed to save lock: %w", err)
	}

	m.logger.Info("gpu.lock.acquired", "GPU lock acquired", map[string]interface{}{
		"holder": holder.String(),
	})

	return nil
}

// Release releases the GPU lock for the given holder
// Only the current holder can release the lock
func (m *Manager) Release(holder Holder) error {
	if !holder.IsValid() {
		return fmt.Errorf("invalid holder: %s", holder)
	}

	existingLock, err := m.loadLock()
	if err != nil {
		if os.IsNotExist(err) {
			// No lock exists, nothing to release
			m.logger.Info("gpu.lock.release.no_lock", "No GPU lock to release", map[string]interface{}{
				"holder": holder.String(),
			})
			return nil
		}
		return fmt.Errorf("failed to read existing lock: %w", err)
	}

	// Verify holder
	if existingLock.Holder != holder {
		return fmt.Errorf("cannot release lock: held by %s, not %s",
			existingLock.Holder.String(), holder.String())
	}

	// Remove lock file
	lockPath := m.getLockPath()
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	m.logger.Info("gpu.lock.released", "GPU lock released", map[string]interface{}{
		"holder": holder.String(),
	})

	return nil
}

// ForceUnlock forcibly removes the GPU lock regardless of holder
// This should only be used for recovery scenarios
func (m *Manager) ForceUnlock() error {
	existingLock, err := m.loadLock()
	if err != nil {
		if os.IsNotExist(err) {
			m.logger.Info("gpu.lock.force_unlock.no_lock", "No GPU lock to force unlock", nil)
			return nil
		}
		return fmt.Errorf("failed to read existing lock: %w", err)
	}

	m.logger.Warn("gpu.lock.stolen", "GPU lock forcibly removed", map[string]interface{}{
		"previous_holder": existingLock.Holder.String(),
		"age_seconds":     time.Since(existingLock.SinceTS).Seconds(),
	})

	return m.forceUnlock()
}

// forceUnlock is the internal implementation of force unlock
func (m *Manager) forceUnlock() error {
	lockPath := m.getLockPath()
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	return nil
}

// GetStatus returns the current lock status
func (m *Manager) GetStatus() (*LockInfo, error) {
	lock, err := m.loadLock()
	if err != nil {
		if os.IsNotExist(err) {
			// No lock exists
			return &LockInfo{
				Holder:  HolderNone,
				SinceTS: time.Time{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read lock: %w", err)
	}

	return lock, nil
}

// IsLocked checks if the GPU is currently locked
func (m *Manager) IsLocked() (bool, error) {
	status, err := m.GetStatus()
	if err != nil {
		return false, err
	}

	if status.Holder == HolderNone {
		return false, nil
	}

	// Check if lock is stale
	if time.Since(status.SinceTS) > m.leaseTimeout {
		m.logger.Warn("gpu.lock.stale_on_check", "Stale lock detected during check", map[string]interface{}{
			"holder":      status.Holder.String(),
			"age_seconds": time.Since(status.SinceTS).Seconds(),
		})
		return false, nil // Stale lock is considered unlocked
	}

	return true, nil
}

// loadLock loads the lock information from disk
func (m *Manager) loadLock() (*LockInfo, error) {
	lockPath := m.getLockPath()

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	var lock LockInfo
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock: %w", err)
	}

	return &lock, nil
}

// saveLock saves the lock information to disk
func (m *Manager) saveLock(lock *LockInfo) error {
	// Ensure state directory exists
	if err := os.MkdirAll(m.stateDir, 0o750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	lockPath := m.getLockPath()

	// Atomic write: write to temp file, then rename
	tmpPath := lockPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp lock file: %w", err)
	}

	if err := os.Rename(tmpPath, lockPath); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			m.logger.Warn("gpu.lock.cleanup_failed", "Failed to remove temp lock file", map[string]interface{}{
				"error": removeErr.Error(),
				"path":  tmpPath,
			})
		}
		return fmt.Errorf("failed to rename lock file: %w", err)
	}

	return nil
}
