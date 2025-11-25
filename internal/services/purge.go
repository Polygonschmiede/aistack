package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aistack/internal/configdir"
	"aistack/internal/fsutil"
	"aistack/internal/logging"
)

// UninstallLog represents a log of an uninstall/purge operation
type UninstallLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Target       string    `json:"target"`
	KeepCache    bool      `json:"keep_cache"`
	RemovedItems []string  `json:"removed_items"`
	Errors       []string  `json:"errors,omitempty"`
}

// PurgeManager handles complete system purge operations
type PurgeManager struct {
	manager  *Manager
	logger   *logging.Logger
	stateDir string
}

// NewPurgeManager creates a new purge manager
func NewPurgeManager(manager *Manager, logger *logging.Logger) *PurgeManager {
	return &PurgeManager{
		manager:  manager,
		logger:   logger,
		stateDir: fsutil.GetStateDir(fsutil.DefaultStateDir),
	}
}

// PurgeAll removes all services, volumes, and optionally configs
func (pm *PurgeManager) PurgeAll(removeConfigs bool) (*UninstallLog, error) {
	pm.logger.Info("purge.started", "Starting full purge operation", map[string]interface{}{
		"remove_configs": removeConfigs,
	})

	log := &UninstallLog{
		Timestamp:    time.Now(),
		Target:       "all",
		KeepCache:    false,
		RemovedItems: []string{},
		Errors:       []string{},
	}

	// Get all services
	services := []string{"ollama", "openwebui", "localai"}

	// Remove all services with purge
	for _, serviceName := range services {
		service, err := pm.manager.GetService(serviceName)
		if err != nil {
			log.Errors = append(log.Errors, fmt.Sprintf("failed to get service %s: %v", serviceName, err))
			continue
		}

		pm.logger.Info("purge.service", fmt.Sprintf("Removing service %s", serviceName), map[string]interface{}{
			"service": serviceName,
		})

		if err := service.Remove(false); err != nil {
			errMsg := fmt.Sprintf("failed to remove service %s: %v", serviceName, err)
			log.Errors = append(log.Errors, errMsg)
			pm.logger.Warn("purge.service.error", errMsg, nil)
		} else {
			log.RemovedItems = append(log.RemovedItems, fmt.Sprintf("service:%s", serviceName))
		}
	}

	// Remove common network
	pm.logger.Info("purge.network", "Removing aistack network", nil)
	if err := pm.manager.runtime.RemoveNetwork("aistack-net"); err != nil {
		errMsg := fmt.Sprintf("failed to remove network: %v", err)
		log.Errors = append(log.Errors, errMsg)
		pm.logger.Warn("purge.network.error", errMsg, nil)
	} else {
		log.RemovedItems = append(log.RemovedItems, "network:aistack-net")
	}

	// Remove state directory contents (except configs unless requested)
	if err := pm.cleanStateDirectory(log, removeConfigs); err != nil {
		errMsg := fmt.Sprintf("failed to clean state directory: %v", err)
		log.Errors = append(log.Errors, errMsg)
	}

	// Remove configs if requested
	if removeConfigs {
		if err := pm.removeConfigs(log); err != nil {
			errMsg := fmt.Sprintf("failed to remove configs: %v", err)
			log.Errors = append(log.Errors, errMsg)
		}
	}

	pm.logger.Info("purge.completed", "Purge operation completed", map[string]interface{}{
		"removed_items": len(log.RemovedItems),
		"errors":        len(log.Errors),
	})

	return log, nil
}

// cleanStateDirectory removes contents of /var/lib/aistack
func (pm *PurgeManager) cleanStateDirectory(log *UninstallLog, removeAll bool) error {
	pm.logger.Info("purge.state_dir", "Cleaning state directory", map[string]interface{}{
		"path": pm.stateDir,
	})

	// Check if directory exists
	if _, err := os.Stat(pm.stateDir); os.IsNotExist(err) {
		pm.logger.Info("purge.state_dir.not_found", "State directory does not exist", nil)
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(pm.stateDir)
	if err != nil {
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(pm.stateDir, entry.Name())

		// Skip certain files if not removing all
		if !removeAll && (entry.Name() == "config.yaml" || entry.Name() == "wol_config.json") {
			pm.logger.Info("purge.state_dir.skip", "Skipping file", map[string]interface{}{
				"file": entry.Name(),
			})
			continue
		}

		// Remove file or directory
		if err := os.RemoveAll(path); err != nil {
			errMsg := fmt.Sprintf("failed to remove %s: %v", path, err)
			log.Errors = append(log.Errors, errMsg)
			pm.logger.Warn("purge.state_dir.error", errMsg, nil)
		} else {
			log.RemovedItems = append(log.RemovedItems, fmt.Sprintf("state:%s", entry.Name()))
		}
	}

	return nil
}

// removeConfigs removes /etc/aistack configuration directory
func (pm *PurgeManager) removeConfigs(log *UninstallLog) error {
	configDir := configdir.ConfigDir()
	pm.logger.Info("purge.configs", "Removing config directory", map[string]interface{}{
		"path": configDir,
	})

	// Only remove if it's /etc/aistack (safety check)
	if configDir != "/etc/aistack" {
		pm.logger.Warn("purge.configs.skip", "Skipping non-standard config directory", map[string]interface{}{
			"path": configDir,
		})
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		pm.logger.Info("purge.configs.not_found", "Config directory does not exist", nil)
		return nil
	}

	// Remove directory
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to remove config directory: %w", err)
	}

	log.RemovedItems = append(log.RemovedItems, fmt.Sprintf("configs:%s", configDir))
	return nil
}

// VerifyClean checks if the system is clean after purge
func (pm *PurgeManager) VerifyClean() (bool, []string) {
	pm.logger.Info("purge.verify", "Verifying system is clean", nil)

	leftovers := []string{}

	// Check for running containers
	services := []string{"ollama", "openwebui", "localai"}
	for _, serviceName := range services {
		containerName := fmt.Sprintf("aistack-%s", serviceName)
		running, err := pm.manager.runtime.IsContainerRunning(containerName)
		if err == nil && running {
			leftovers = append(leftovers, fmt.Sprintf("container:%s", containerName))
		}
	}

	// Check for volumes
	volumes := []string{"ollama_data", "openwebui_data", "localai_models"}
	for _, volume := range volumes {
		exists, err := pm.manager.runtime.VolumeExists(volume)
		if err == nil && exists {
			leftovers = append(leftovers, fmt.Sprintf("volume:%s", volume))
		}
	}

	// Check state directory
	if _, err := os.Stat(pm.stateDir); err == nil {
		entries, err := os.ReadDir(pm.stateDir)
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				leftovers = append(leftovers, fmt.Sprintf("state:%s", entry.Name()))
			}
		}
	}

	isClean := len(leftovers) == 0
	pm.logger.Info("purge.verify.complete", "Verification complete", map[string]interface{}{
		"clean":     isClean,
		"leftovers": len(leftovers),
	})

	return isClean, leftovers
}

// SaveUninstallLog saves the uninstall log to a file
func (pm *PurgeManager) SaveUninstallLog(log *UninstallLog, path string) error {
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal uninstall log: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := fsutil.EnsureStateDirectory(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("failed to write uninstall log: %w", err)
	}

	pm.logger.Info("purge.log.saved", "Uninstall log saved", map[string]interface{}{
		"path": path,
	})

	return nil
}

// CreateUninstallLogForService creates an uninstall log for a single service
func CreateUninstallLogForService(serviceName string, keepData bool, removed []string, errors []string) *UninstallLog {
	return &UninstallLog{
		Timestamp:    time.Now(),
		Target:       serviceName,
		KeepCache:    keepData,
		RemovedItems: removed,
		Errors:       errors,
	}
}
