package wol

import (
	"bytes"
	"fmt"
	"net"

	"aistack/internal/logging"
)

// Sender handles sending Wake-on-LAN magic packets
type Sender struct {
	logger *logging.Logger
}

// NewSender creates a new magic packet sender
func NewSender(logger *logging.Logger) *Sender {
	return &Sender{
		logger: logger,
	}
}

// SendMagicPacket sends a Wake-on-LAN magic packet to the specified MAC address
func (s *Sender) SendMagicPacket(targetMAC, broadcastAddr string) error {
	// Parse and validate MAC address
	hwAddr, err := ParseMAC(targetMAC)
	if err != nil {
		return fmt.Errorf("invalid MAC address: %w", err)
	}

	// Build magic packet
	packet, err := buildMagicPacket(hwAddr)
	if err != nil {
		return fmt.Errorf("failed to build magic packet: %w", err)
	}

	// Default broadcast address if not specified
	if broadcastAddr == "" {
		broadcastAddr = "255.255.255.255"
	}

	// Send packet on both standard WoL ports (7 and 9)
	ports := []int{7, 9}
	var lastErr error

	for _, port := range ports {
		addr := fmt.Sprintf("%s:%d", broadcastAddr, port)
		if err := s.sendUDP(addr, packet); err != nil {
			s.logger.Warn("wol.send.port_failed", "Failed to send on port", map[string]interface{}{
				"port":      port,
				"broadcast": broadcastAddr,
				"error":     err.Error(),
			})
			lastErr = err
		} else {
			s.logger.Info("wol.send.success", "Magic packet sent", map[string]interface{}{
				"mac":       targetMAC,
				"broadcast": broadcastAddr,
				"port":      port,
			})
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to send magic packet: %w", lastErr)
	}

	return nil
}

// sendUDP sends the magic packet via UDP
func (s *Sender) sendUDP(addr string, packet []byte) error {
	// Resolve UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve address %s: %w", addr, err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			s.logger.Warn("wol.send.close_failed", "Failed to close UDP connection", map[string]interface{}{
				"address": addr,
				"error":   closeErr.Error(),
			})
		}
	}()

	// Send packet
	n, err := conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to write packet: %w", err)
	}

	if n != len(packet) {
		return fmt.Errorf("incomplete packet send: sent %d of %d bytes", n, len(packet))
	}

	return nil
}

// buildMagicPacket constructs a Wake-on-LAN magic packet
// Magic packet format:
// - 6 bytes of 0xFF
// - 16 repetitions of the target MAC address (6 bytes each)
// Total: 102 bytes
func buildMagicPacket(mac net.HardwareAddr) ([]byte, error) {
	if len(mac) != 6 {
		return nil, fmt.Errorf("invalid MAC address length: expected 6 bytes, got %d", len(mac))
	}

	var buf bytes.Buffer

	// Write 6 bytes of 0xFF
	for i := 0; i < 6; i++ {
		buf.WriteByte(0xFF)
	}

	// Write MAC address 16 times
	for i := 0; i < 16; i++ {
		buf.Write(mac)
	}

	return buf.Bytes(), nil
}

// ValidateMagicPacket validates that a byte slice is a valid magic packet
func ValidateMagicPacket(packet []byte) error {
	if len(packet) != 102 {
		return fmt.Errorf("invalid packet length: expected 102 bytes, got %d", len(packet))
	}

	// Check first 6 bytes are 0xFF
	for i := 0; i < 6; i++ {
		if packet[i] != 0xFF {
			return fmt.Errorf("invalid packet header: byte %d should be 0xFF, got 0x%02X", i, packet[i])
		}
	}

	// Extract MAC from first repetition
	mac := packet[6:12]

	// Verify all 16 repetitions match
	for i := 0; i < 16; i++ {
		start := 6 + (i * 6)
		end := start + 6
		segment := packet[start:end]

		if !bytes.Equal(segment, mac) {
			return fmt.Errorf("MAC repetition %d does not match: expected %v, got %v", i, mac, segment)
		}
	}

	return nil
}
