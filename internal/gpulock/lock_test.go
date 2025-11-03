package gpulock

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"aistack/internal/logging"
)

func TestNewManager(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager("/tmp", logger)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.stateDir != "/tmp" {
		t.Errorf("Expected stateDir '/tmp', got: %s", manager.stateDir)
	}

	if manager.leaseTimeout != DefaultLeaseTimeout {
		t.Errorf("Expected default lease timeout %v, got: %v", DefaultLeaseTimeout, manager.leaseTimeout)
	}
}

func TestAcquire_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock
	err = manager.Acquire(HolderOpenWebUI)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify lock file exists
	lockPath := filepath.Join(tmpDir, LockFileName)
	if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
		t.Error("Lock file was not created")
	}

	// Verify lock status
	status, err := manager.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Holder != HolderOpenWebUI {
		t.Errorf("Expected holder OpenWebUI, got: %s", status.Holder)
	}
}

func TestAcquire_AlreadyHeld(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock first time
	if err = manager.Acquire(HolderLocalAI); err != nil {
		t.Fatal(err)
	}

	// Acquire again with same holder - should succeed
	if err = manager.Acquire(HolderLocalAI); err != nil {
		t.Errorf("Expected no error when same holder acquires again, got: %v", err)
	}
}

func TestAcquire_ConflictingHolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock for OpenWebUI
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Try to acquire for LocalAI - should fail
	err = manager.Acquire(HolderLocalAI)
	if err == nil {
		t.Error("Expected error when different holder tries to acquire, got nil")
	}
}

func TestAcquire_StaleLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Set very short lease timeout for testing
	manager.leaseTimeout = 100 * time.Millisecond

	// Acquire lock
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Wait for lock to become stale
	time.Sleep(150 * time.Millisecond)

	// Try to acquire with different holder - should succeed (stale lock auto-cleared)
	err = manager.Acquire(HolderLocalAI)
	if err != nil {
		t.Errorf("Expected stale lock to be cleared, got error: %v", err)
	}

	// Verify new holder
	status, err := manager.GetStatus()
	if err != nil {
		t.Fatal(err)
	}

	if status.Holder != HolderLocalAI {
		t.Errorf("Expected holder LocalAI after stale lock, got: %s", status.Holder)
	}
}

func TestRelease_Success(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Release lock
	err = manager.Release(HolderOpenWebUI)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify lock is released
	status, err := manager.GetStatus()
	if err != nil {
		t.Fatal(err)
	}

	if status.Holder != HolderNone {
		t.Errorf("Expected holder None after release, got: %s", status.Holder)
	}
}

func TestRelease_WrongHolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock for OpenWebUI
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Try to release with wrong holder
	err = manager.Release(HolderLocalAI)
	if err == nil {
		t.Error("Expected error when wrong holder tries to release, got nil")
	}
}

func TestRelease_NoLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Release when no lock exists - should not error
	err = manager.Release(HolderOpenWebUI)
	if err != nil {
		t.Errorf("Expected no error when releasing non-existent lock, got: %v", err)
	}
}

func TestForceUnlock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Acquire lock
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Force unlock
	err = manager.ForceUnlock()
	if err != nil {
		t.Fatalf("ForceUnlock failed: %v", err)
	}

	// Verify lock is gone
	status, err := manager.GetStatus()
	if err != nil {
		t.Fatal(err)
	}

	if status.Holder != HolderNone {
		t.Errorf("Expected holder None after force unlock, got: %s", status.Holder)
	}
}

func TestIsLocked(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Initially not locked
	locked, err := manager.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("Expected not locked initially")
	}

	// Acquire lock
	if err = manager.Acquire(HolderLocalAI); err != nil {
		t.Fatal(err)
	}

	// Should be locked now
	locked, err = manager.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Error("Expected to be locked after acquire")
	}

	// Release lock
	if err = manager.Release(HolderLocalAI); err != nil {
		t.Fatal(err)
	}

	// Should not be locked
	locked, err = manager.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("Expected not locked after release")
	}
}

func TestIsLocked_StaleLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gpulock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := logging.NewLogger(logging.LevelInfo)
	manager := NewManager(tmpDir, logger)

	// Set very short lease timeout
	manager.leaseTimeout = 100 * time.Millisecond

	// Acquire lock
	if err = manager.Acquire(HolderOpenWebUI); err != nil {
		t.Fatal(err)
	}

	// Wait for lock to become stale
	time.Sleep(150 * time.Millisecond)

	// Should not be considered locked (stale)
	locked, err := manager.IsLocked()
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		t.Error("Expected stale lock to be considered unlocked")
	}
}

func TestHolderIsValid(t *testing.T) {
	tests := []struct {
		holder Holder
		valid  bool
	}{
		{HolderNone, true},
		{HolderOpenWebUI, true},
		{HolderLocalAI, true},
		{Holder("invalid"), false},
		{Holder("ollama"), false},
	}

	for _, test := range tests {
		if test.holder.IsValid() != test.valid {
			t.Errorf("Expected IsValid() for %s to be %v, got %v",
				test.holder, test.valid, !test.valid)
		}
	}
}
