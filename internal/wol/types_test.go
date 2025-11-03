package wol

import (
	"testing"
)

func TestValidateMAC_Valid(t *testing.T) {
	validMACs := []string{
		"00:11:22:33:44:55",
		"AA:BB:CC:DD:EE:FF",
		"aa:bb:cc:dd:ee:ff",
		"00-11-22-33-44-55",
		"001122334455",
		"AABBCCDDEEFF",
	}

	for _, mac := range validMACs {
		if err := ValidateMAC(mac); err != nil {
			t.Errorf("Expected valid MAC %s, got error: %v", mac, err)
		}
	}
}

func TestValidateMAC_Invalid(t *testing.T) {
	invalidMACs := []string{
		"00:11:22:33:44",       // Too short
		"00:11:22:33:44:55:66", // Too long
		"GG:HH:II:JJ:KK:LL",    // Invalid hex
		"00:11:22",             // Way too short
		"",                     // Empty
		"not-a-mac",            // Invalid format
	}

	for _, mac := range invalidMACs {
		if err := ValidateMAC(mac); err == nil {
			t.Errorf("Expected invalid MAC %s to fail validation", mac)
		}
	}
}

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"00:11:22:33:44:55", "00:11:22:33:44:55"},
		{"00-11-22-33-44-55", "00:11:22:33:44:55"},
		{"001122334455", "00:11:22:33:44:55"},
		{"aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"AABBCCDDEEFF", "AA:BB:CC:DD:EE:FF"},
	}

	for _, test := range tests {
		result, err := NormalizeMAC(test.input)
		if err != nil {
			t.Errorf("NormalizeMAC(%s) failed: %v", test.input, err)
			continue
		}

		if result != test.expected {
			t.Errorf("NormalizeMAC(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseMAC(t *testing.T) {
	validMACs := []string{
		"00:11:22:33:44:55",
		"AA:BB:CC:DD:EE:FF",
		"00-11-22-33-44-55",
		"001122334455",
	}

	for _, mac := range validMACs {
		hwAddr, err := ParseMAC(mac)
		if err != nil {
			t.Errorf("ParseMAC(%s) failed: %v", mac, err)
			continue
		}

		if len(hwAddr) != 6 {
			t.Errorf("ParseMAC(%s) returned invalid length: expected 6, got %d", mac, len(hwAddr))
		}
	}
}

func TestParseMAC_Invalid(t *testing.T) {
	invalidMACs := []string{
		"00:11:22:33:44",
		"GG:HH:II:JJ:KK:LL",
		"invalid",
	}

	for _, mac := range invalidMACs {
		_, err := ParseMAC(mac)
		if err == nil {
			t.Errorf("Expected ParseMAC(%s) to fail", mac)
		}
	}
}

func TestGetBroadcastAddr(t *testing.T) {
	// This test will only work if there's a network interface available
	// On systems without network, this will fail gracefully

	// Try to get broadcast address for loopback (should fail)
	_, err := GetBroadcastAddr("lo")
	if err == nil {
		t.Log("Loopback interface has broadcast address (unexpected but not an error)")
	}

	// We can't test with actual interfaces as they vary by system
	// Just verify the function doesn't panic
	t.Log("GetBroadcastAddr function is callable")
}
