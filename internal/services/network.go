package services

import (
	"aistack/internal/logging"
	"fmt"
)

const (
	// AistackNetwork is the name of the common aistack network
	AistackNetwork = "aistack-net"
)

// NetworkManager handles network setup and teardown
type NetworkManager struct {
	runtime Runtime
	logger  *logging.Logger
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(runtime Runtime, logger *logging.Logger) *NetworkManager {
	return &NetworkManager{
		runtime: runtime,
		logger:  logger,
	}
}

// EnsureNetwork creates the aistack network if it doesn't exist (idempotent)
// Story T-005: Compose-Template: Netzwerk & Volumes
func (nm *NetworkManager) EnsureNetwork() error {
	nm.logger.Info("network.ensure", "Ensuring aistack network exists", map[string]interface{}{
		"network": AistackNetwork,
	})

	if err := nm.runtime.CreateNetwork(AistackNetwork); err != nil {
		nm.logger.Error("network.create.error", "Failed to create network", map[string]interface{}{
			"network": AistackNetwork,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to ensure network: %w", err)
	}

	nm.logger.Info("network.ready", "Network ready", map[string]interface{}{
		"network": AistackNetwork,
	})

	return nil
}

// EnsureVolumes creates the required volumes if they don't exist (idempotent)
// Story T-005: Compose-Template: Netzwerk & Volumes
func (nm *NetworkManager) EnsureVolumes(volumes []string) error {
	for _, vol := range volumes {
		nm.logger.Info("volume.ensure", "Ensuring volume exists", map[string]interface{}{
			"volume": vol,
		})

		if err := nm.runtime.CreateVolume(vol); err != nil {
			nm.logger.Error("volume.create.error", "Failed to create volume", map[string]interface{}{
				"volume": vol,
				"error":  err.Error(),
			})
			return fmt.Errorf("failed to ensure volume %s: %w", vol, err)
		}
	}

	nm.logger.Info("volumes.ready", "All volumes ready", map[string]interface{}{
		"count": len(volumes),
	})

	return nil
}
