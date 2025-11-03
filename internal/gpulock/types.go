package gpulock

import "time"

// Holder represents a service that holds the GPU lock
type Holder string

const (
	// HolderNone indicates no service holds the lock
	HolderNone Holder = "none"
	// HolderOpenWebUI indicates Open WebUI holds the lock
	HolderOpenWebUI Holder = "openwebui"
	// HolderLocalAI indicates LocalAI holds the lock
	HolderLocalAI Holder = "localai"
)

// LockInfo represents the GPU lock state
// Story T-021: GPU-Mutex (Dateisperre + Lease)
type LockInfo struct {
	Holder  Holder    `json:"holder"`
	SinceTS time.Time `json:"since_ts"`
}

// String returns the string representation of a Holder
func (h Holder) String() string {
	return string(h)
}

// IsValid checks if a holder value is valid
func (h Holder) IsValid() bool {
	switch h {
	case HolderNone, HolderOpenWebUI, HolderLocalAI:
		return true
	default:
		return false
	}
}
