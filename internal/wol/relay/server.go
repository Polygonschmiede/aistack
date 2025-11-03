package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aistack/internal/logging"
	"aistack/internal/wol"
)

// Server provides a minimal HTTP → WoL relay
type Server struct {
	addr   string
	key    string
	logger *logging.Logger
	sender *wol.Sender
}

// NewServer constructs a relay server instance
func NewServer(addr, key string, logger *logging.Logger) *Server {
	return &Server{
		addr:   addr,
		key:    key,
		logger: logger,
		sender: wol.NewSender(logger),
	}
}

// Serve starts the HTTP server and blocks
func (s *Server) Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/wake", s.handleWake)

	server := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	s.logger.Info("wol.relay.started", "Starting WoL relay", map[string]interface{}{
		"listen": s.addr,
	})

	return server.ListenAndServe()
}

type wakeRequest struct {
	MAC       string `json:"mac"`
	Broadcast string `json:"broadcast"`
	Key       string `json:"key"`
}

type wakeResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req wakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if req.Key == "" || req.Key != s.key {
		s.writeError(w, http.StatusForbidden, "invalid relay key")
		return
	}

	if err := wol.ValidateMAC(req.MAC); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid MAC: %v", err))
		return
	}

	if err := s.sender.SendMagicPacket(req.MAC, req.Broadcast); err != nil {
		s.logger.Warn("wol.relay.send_failed", "Failed to send magic packet", map[string]interface{}{
			"mac":       req.MAC,
			"broadcast": req.Broadcast,
			"error":     err.Error(),
		})
		s.writeError(w, http.StatusInternalServerError, "failed to send magic packet")
		return
	}

	s.logger.Info("wol.relay.sent", "Magic packet relayed", map[string]interface{}{
		"mac":       req.MAC,
		"broadcast": req.Broadcast,
	})

	s.writeJSON(w, http.StatusOK, wakeResponse{Status: "ok"})
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
