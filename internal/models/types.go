package models

import "time"

// Provider represents the model provider type
type Provider string

const (
	// ProviderOllama represents Ollama models
	ProviderOllama Provider = "ollama"
	// ProviderLocalAI represents LocalAI models
	ProviderLocalAI Provider = "localai"
)

// IsValid checks if the provider is valid
func (p Provider) IsValid() bool {
	return p == ProviderOllama || p == ProviderLocalAI
}

// String returns the string representation of the provider
func (p Provider) String() string {
	return string(p)
}

// ModelInfo represents a cached model entry
// Story T-022, T-023: Model Management & Caching
type ModelInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`      // Size in bytes
	Path     string    `json:"path"`      // Path to model files
	LastUsed time.Time `json:"last_used"` // Last access timestamp
}

// State represents the overall model cache state
// Data Contract from EP-012: models_state.json
type State struct {
	Provider Provider    `json:"provider"`
	Items    []ModelInfo `json:"items"`
	Updated  time.Time   `json:"updated"`
}

// DownloadProgress represents model download progress
// Story T-022: Download progress tracking
type DownloadProgress struct {
	ModelName       string  `json:"model_name"`
	BytesDownloaded int64   `json:"bytes_downloaded"`
	TotalBytes      int64   `json:"total_bytes,omitempty"`
	Percentage      float64 `json:"percentage"`
	Status          string  `json:"status"` // "started", "progress", "completed", "failed"
	Error           string  `json:"error,omitempty"`
}

// DownloadOptions represents options for model download
type DownloadOptions struct {
	ModelName string
	Resume    bool // Whether to resume partial downloads
}

// CacheStats represents cache statistics
// Story T-023: Cache overview
type CacheStats struct {
	Provider    Provider   `json:"provider"`
	TotalSize   int64      `json:"total_size"`             // Total size in bytes
	ModelCount  int        `json:"model_count"`            // Number of models
	OldestModel *ModelInfo `json:"oldest_model,omitempty"` // Oldest model by last_used
}
