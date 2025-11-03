package wol

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"

	"aistack/internal/logging"
)

// Detector handles Wake-on-LAN detection and configuration
type Detector struct {
	logger *logging.Logger
}

// NewDetector creates a new WoL detector
func NewDetector(logger *logging.Logger) *Detector {
	return &Detector{
		logger: logger,
	}
}

// DetectWoL checks the Wake-on-LAN status of a network interface
func (d *Detector) DetectWoL(iface string) WoLStatus {
	status := WoLStatus{
		Interface:    iface,
		Supported:    false,
		Enabled:      false,
		WoLModes:     []string{},
		CurrentMode:  "d", // disabled by default
		ErrorMessage: "",
	}

	// Check if interface exists
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("interface not found: %v", err)
		d.logger.Warn("wol.detect.interface_not_found", "Interface not found", map[string]interface{}{
			"interface": iface,
			"error":     err.Error(),
		})
		return status
	}

	status.MAC = netIface.HardwareAddr.String()

	// Check if ethtool is available
	if _, err := exec.LookPath("ethtool"); err != nil {
		status.ErrorMessage = "ethtool not found (required for WoL detection)"
		d.logger.Warn("wol.detect.ethtool_not_found", "ethtool not available", map[string]interface{}{
			"interface": iface,
		})
		return status
	}

	// Run ethtool to get WoL status
	cmd := exec.Command("ethtool", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("ethtool failed: %v", err)
		d.logger.Warn("wol.detect.ethtool_failed", "ethtool command failed", map[string]interface{}{
			"interface": iface,
			"error":     err.Error(),
		})
		return status
	}

	// Parse ethtool output
	outputStr := string(output)
	d.parseEthtoolOutput(outputStr, &status)

	d.logger.Info("wol.detect.success", "WoL detection completed", map[string]interface{}{
		"interface":    iface,
		"supported":    status.Supported,
		"enabled":      status.Enabled,
		"current_mode": status.CurrentMode,
	})

	return status
}

// parseEthtoolOutput parses ethtool output to extract WoL information
func (d *Detector) parseEthtoolOutput(output string, status *WoLStatus) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Supports Wake-on: ..." line
		if strings.Contains(line, "Supports Wake-on:") {
			re := regexp.MustCompile(`Supports Wake-on:\s*(.+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				modes := strings.TrimSpace(matches[1])
				status.WoLModes = d.parseWoLModes(modes)
				status.Supported = len(status.WoLModes) > 0
			}
		}

		// Look for "Wake-on: ..." line (current setting)
		if strings.Contains(line, "Wake-on:") && !strings.Contains(line, "Supports") {
			re := regexp.MustCompile(`Wake-on:\s*(\w+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				status.CurrentMode = strings.TrimSpace(matches[1])
				// Mode "g" means WoL is enabled (magic packet)
				status.Enabled = status.CurrentMode == "g"
			}
		}
	}
}

// parseWoLModes parses the WoL modes string from ethtool
// Modes: p=PHY, u=unicast, m=multicast, b=broadcast, a=ARP, g=magic packet, d=disabled
func (d *Detector) parseWoLModes(modes string) []string {
	result := []string{}
	for _, char := range modes {
		mode := string(char)
		if mode != " " {
			result = append(result, mode)
		}
	}
	return result
}

// EnableWoL enables Wake-on-LAN for the specified interface
func (d *Detector) EnableWoL(iface string) error {
	return d.SetWoLMode(iface, "g")
}

// DisableWoL disables Wake-on-LAN for the specified interface
func (d *Detector) DisableWoL(iface string) error {
	return d.SetWoLMode(iface, "d")
}

// SetWoLMode sets the WoL mode (e.g., g, d) for an interface
func (d *Detector) SetWoLMode(iface string, mode string) error {
	if _, err := exec.LookPath("ethtool"); err != nil {
		return fmt.Errorf("ethtool not found: %w", err)
	}

	d.logger.Info("wol.mode.set", "Applying WoL mode", map[string]interface{}{
		"interface": iface,
		"mode":      mode,
	})

	cmd := exec.Command("ethtool", "-s", iface, "wol", mode)
	output, err := cmd.CombinedOutput()
	if err != nil {
		d.logger.Error("wol.mode.failed", "Failed to set WoL mode", map[string]interface{}{
			"interface": iface,
			"mode":      mode,
			"error":     err.Error(),
			"output":    string(output),
		})
		return fmt.Errorf("failed to set WoL mode %s on %s: %w (output: %s)", mode, iface, err, string(output))
	}

	d.logger.Info("wol.mode.applied", "WoL mode applied", map[string]interface{}{
		"interface": iface,
		"mode":      mode,
	})

	return nil
}

// ApplyConfig applies a persisted WoL configuration
func (d *Detector) ApplyConfig(cfg WoLConfig) error {
	if cfg.Interface == "" {
		return fmt.Errorf("WoL config missing interface")
	}

	mode := cfg.WoLState
	if mode == "" {
		mode = "g"
	}

	return d.SetWoLMode(cfg.Interface, mode)
}

// GetDefaultInterface attempts to find the default network interface
func (d *Detector) GetDefaultInterface() (string, error) {
	// Try to get network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to list network interfaces: %w", err)
	}

	// Look for a non-loopback interface with an IPv4 address
	for _, iface := range interfaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Check if it has an IPv4 address
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ipNet.IP.To4() != nil {
					d.logger.Info("wol.default_interface.found", "Found default interface", map[string]interface{}{
						"interface": iface.Name,
						"mac":       iface.HardwareAddr.String(),
					})
					return iface.Name, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}
