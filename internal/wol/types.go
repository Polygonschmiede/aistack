package wol

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// WoLConfig holds Wake-on-LAN configuration
//
//nolint:revive // exported name intentionally mirrors package (wol.WoLConfig)
type WoLConfig struct {
	// Interface is the network interface name (e.g., "eth0", "eno1")
	Interface string `json:"interface"`

	// MAC is the MAC address of the interface
	MAC string `json:"mac"`

	// WoLState is the current WoL state ("g" = enabled, "d" = disabled)
	WoLState string `json:"wol_state"`

	// BroadcastIP is the broadcast address for sending magic packets
	BroadcastIP string `json:"broadcast_ip"`
}

// WoLStatus represents the Wake-on-LAN status of a system
//
//nolint:revive // exported name intentionally mirrors package (wol.WoLStatus)
type WoLStatus struct {
	// Supported indicates if WoL is supported by the hardware/driver
	Supported bool `json:"supported"`

	// Enabled indicates if WoL is currently enabled
	Enabled bool `json:"enabled"`

	// Interface is the network interface being checked
	Interface string `json:"interface"`

	// MAC is the MAC address of the interface
	MAC string `json:"mac"`

	// WoLModes lists the available WoL modes (e.g., "g", "p", "u", "m", "b", "a", "d")
	WoLModes []string `json:"wol_modes"`

	// CurrentMode is the currently active WoL mode
	CurrentMode string `json:"current_mode"`

	// ErrorMessage contains any error encountered during detection
	ErrorMessage string `json:"error_message,omitempty"`
}

// MagicPacket represents a Wake-on-LAN magic packet
type MagicPacket struct {
	// TargetMAC is the MAC address to wake
	TargetMAC string

	// BroadcastAddr is the broadcast address to send to
	BroadcastAddr string

	// Port is the UDP port (typically 7 or 9)
	Port int
}

// ValidateMAC validates a MAC address format
func ValidateMAC(mac string) error {
	// Remove common separators
	cleaned := strings.ReplaceAll(mac, ":", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) != 12 {
		return fmt.Errorf("invalid MAC address length: expected 12 hex digits, got %d", len(cleaned))
	}

	// Check if all characters are valid hex
	matched, err := regexp.MatchString("^[0-9A-Fa-f]{12}$", cleaned)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}

	if !matched {
		return fmt.Errorf("invalid MAC address format: must contain only hex digits")
	}

	return nil
}

// NormalizeMAC normalizes a MAC address to colon-separated format (AA:BB:CC:DD:EE:FF)
func NormalizeMAC(mac string) (string, error) {
	// Remove separators
	cleaned := strings.ReplaceAll(mac, ":", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ToUpper(cleaned)

	if err := ValidateMAC(mac); err != nil {
		return "", err
	}

	// Format as AA:BB:CC:DD:EE:FF
	result := ""
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			result += ":"
		}
		result += cleaned[i : i+2]
	}

	return result, nil
}

// ParseMAC parses a MAC address string into a net.HardwareAddr
func ParseMAC(mac string) (net.HardwareAddr, error) {
	normalized, err := NormalizeMAC(mac)
	if err != nil {
		return nil, err
	}

	return net.ParseMAC(normalized)
}

// GetBroadcastAddr calculates the broadcast address for a given interface
func GetBroadcastAddr(iface string) (string, error) {
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return "", fmt.Errorf("failed to get interface %s: %w", iface, err)
	}

	addrs, err := netIface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", iface, err)
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipNet.IP.To4() != nil {
				// Calculate broadcast address
				broadcast := make(net.IP, 4)
				for i := 0; i < 4; i++ {
					broadcast[i] = ipNet.IP.To4()[i] | ^ipNet.Mask[i]
				}
				return broadcast.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found on interface %s", iface)
}
