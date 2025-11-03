package wol

import (
	"os/exec"
	"testing"

	"aistack/internal/logging"
)

func TestDetector_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	if detector == nil {
		t.Error("Expected detector to be created")
	}

	if detector.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestDetector_ParseWoLModes(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	tests := []struct {
		input    string
		expected []string
	}{
		{"pumbg", []string{"p", "u", "m", "b", "g"}},
		{"g", []string{"g"}},
		{"d", []string{"d"}},
		{"", []string{}},
		{"p u m b g", []string{"p", "u", "m", "b", "g"}},
	}

	for _, test := range tests {
		result := detector.parseWoLModes(test.input)

		if len(result) != len(test.expected) {
			t.Errorf("parseWoLModes(%q) returned %d modes, expected %d", test.input, len(result), len(test.expected))
			continue
		}

		for i, mode := range result {
			if mode != test.expected[i] {
				t.Errorf("parseWoLModes(%q)[%d] = %s, expected %s", test.input, i, mode, test.expected[i])
			}
		}
	}
}

func TestDetector_ParseEthtoolOutput(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Sample ethtool output
	output := `Settings for eth0:
	Supported ports: [ TP ]
	Supported link modes:   10baseT/Half 10baseT/Full
	                        100baseT/Half 100baseT/Full
	                        1000baseT/Full
	Supports auto-negotiation: Yes
	Supports Wake-on: pumbg
	Wake-on: g
	Link detected: yes`

	status := WoLStatus{}
	detector.parseEthtoolOutput(output, &status)

	// Verify supported modes
	if !status.Supported {
		t.Error("Expected WoL to be supported")
	}

	expectedModes := []string{"p", "u", "m", "b", "g"}
	if len(status.WoLModes) != len(expectedModes) {
		t.Errorf("Expected %d WoL modes, got %d", len(expectedModes), len(status.WoLModes))
	}

	// Verify current mode
	if status.CurrentMode != "g" {
		t.Errorf("Expected current mode 'g', got '%s'", status.CurrentMode)
	}

	// Verify enabled status
	if !status.Enabled {
		t.Error("Expected WoL to be enabled when mode is 'g'")
	}
}

func TestDetector_ParseEthtoolOutput_Disabled(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Sample ethtool output with WoL disabled
	output := `Settings for eth0:
	Supported ports: [ TP ]
	Supports Wake-on: pumbg
	Wake-on: d
	Link detected: yes`

	status := WoLStatus{}
	detector.parseEthtoolOutput(output, &status)

	// Verify current mode is disabled
	if status.CurrentMode != "d" {
		t.Errorf("Expected current mode 'd', got '%s'", status.CurrentMode)
	}

	// Verify enabled status
	if status.Enabled {
		t.Error("Expected WoL to be disabled when mode is 'd'")
	}
}

func TestDetector_DetectWoL_InvalidInterface(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Try to detect on non-existent interface
	status := detector.DetectWoL("nonexistent999")

	if status.Supported {
		t.Error("Expected WoL to not be supported on invalid interface")
	}

	if status.ErrorMessage == "" {
		t.Error("Expected error message for invalid interface")
	}
}

func TestDetector_GetDefaultInterface(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Try to get default interface
	// This may or may not succeed depending on the system
	iface, err := detector.GetDefaultInterface()

	if err != nil {
		t.Logf("No default interface found (expected on some systems): %v", err)
	} else {
		t.Logf("Default interface: %s", iface)

		if iface == "" {
			t.Error("Expected non-empty interface name")
		}
	}
}

func TestDetector_EnableWoL_NoEthtool(t *testing.T) {
	// This test only runs if ethtool is NOT available
	if _, err := exec.LookPath("ethtool"); err == nil {
		t.Skip("ethtool available; skipping no-ethtool test")
	}

	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Should fail gracefully without ethtool
	err := detector.EnableWoL("eth0")
	if err == nil {
		t.Error("Expected error when ethtool is not available")
	}
}

func TestDetector_DisableWoL_NoEthtool(t *testing.T) {
	// This test only runs if ethtool is NOT available
	if _, err := exec.LookPath("ethtool"); err == nil {
		t.Skip("ethtool available; skipping no-ethtool test")
	}

	logger := logging.NewLogger(logging.LevelInfo)
	detector := NewDetector(logger)

	// Should fail gracefully without ethtool
	err := detector.DisableWoL("eth0")
	if err == nil {
		t.Error("Expected error when ethtool is not available")
	}
}
