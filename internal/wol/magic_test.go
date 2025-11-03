package wol

import (
	"bytes"
	"testing"

	"aistack/internal/logging"
)

func TestBuildMagicPacket(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	sender := NewSender(logger)

	// Test MAC address
	mac, err := ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatalf("Failed to parse test MAC: %v", err)
	}

	packet, err := buildMagicPacket(mac)
	if err != nil {
		t.Fatalf("buildMagicPacket failed: %v", err)
	}

	// Verify packet length (6 + 16*6 = 102 bytes)
	if len(packet) != 102 {
		t.Errorf("Expected packet length 102, got %d", len(packet))
	}

	// Verify first 6 bytes are 0xFF
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			t.Errorf("Byte %d should be 0xFF, got 0x%02X", i, packet[i])
		}
	}

	// Verify MAC is repeated 16 times
	for i := 0; i < 16; i++ {
		start := 6 + (i * 6)
		end := start + 6
		segment := packet[start:end]

		if !bytes.Equal(segment, mac) {
			t.Errorf("MAC repetition %d doesn't match: expected %v, got %v", i, mac, segment)
		}
	}

	// Verify with ValidateMagicPacket
	if err := ValidateMagicPacket(packet); err != nil {
		t.Errorf("ValidateMagicPacket failed: %v", err)
	}

	_ = sender // Avoid unused variable
}

func TestValidateMagicPacket_Valid(t *testing.T) {
	// Build a valid packet
	mac, _ := ParseMAC("AA:BB:CC:DD:EE:FF")
	packet, _ := buildMagicPacket(mac)

	if err := ValidateMagicPacket(packet); err != nil {
		t.Errorf("Expected valid packet, got error: %v", err)
	}
}

func TestValidateMagicPacket_InvalidLength(t *testing.T) {
	packet := make([]byte, 50) // Wrong length

	if err := ValidateMagicPacket(packet); err == nil {
		t.Error("Expected error for invalid packet length")
	}
}

func TestValidateMagicPacket_InvalidHeader(t *testing.T) {
	packet := make([]byte, 102)

	// Set first byte to something other than 0xFF
	packet[0] = 0x00

	if err := ValidateMagicPacket(packet); err == nil {
		t.Error("Expected error for invalid packet header")
	}
}

func TestValidateMagicPacket_InvalidRepetition(t *testing.T) {
	packet := make([]byte, 102)

	// Set header correctly
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Set first MAC repetition
	mac := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	copy(packet[6:12], mac)

	// Set second repetition to different MAC (invalid)
	differentMAC := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}
	copy(packet[12:18], differentMAC)

	if err := ValidateMagicPacket(packet); err == nil {
		t.Error("Expected error for invalid MAC repetition")
	}
}

func TestSender_Creation(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	sender := NewSender(logger)

	if sender == nil {
		t.Error("Expected sender to be created")
	}

	if sender.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestBuildMagicPacket_InvalidMACLength(t *testing.T) {
	// Test with invalid MAC length
	invalidMAC := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE} // Only 5 bytes

	_, err := buildMagicPacket(invalidMAC)
	if err == nil {
		t.Error("Expected error for invalid MAC length")
	}
}
