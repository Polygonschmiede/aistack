package models

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"aistack/internal/logging"
)

const (
	// OllamaAPIBase is the base URL for Ollama API
	OllamaAPIBase = "http://localhost:11434"
)

// OllamaManager manages Ollama models
// Story T-022: Ollama model download with progress
type OllamaManager struct {
	stateManager *StateManager
	logger       *logging.Logger
	apiBase      string
	httpClient   *http.Client
}

// NewOllamaManager creates a new Ollama model manager
func NewOllamaManager(stateDir string, logger *logging.Logger) *OllamaManager {
	return &OllamaManager{
		stateManager: NewStateManager(stateDir, ProviderOllama, logger),
		logger:       logger,
		apiBase:      OllamaAPIBase,
		httpClient:   &http.Client{Timeout: 5 * time.Minute},
	}
}

// ollamaListResponse represents the response from Ollama API /api/tags
type ollamaListResponse struct {
	Models []struct {
		Name       string    `json:"name"`
		ModifiedAt time.Time `json:"modified_at"`
		Size       int64     `json:"size"`
	} `json:"models"`
}

// ollamaPullProgress represents a progress event from Ollama pull stream
type ollamaPullProgress struct {
	Status    string `json:"status"`
	Completed int64  `json:"completed,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Error     string `json:"error,omitempty"`
}

// List returns all available Ollama models
func (m *OllamaManager) List() ([]ModelInfo, error) {
	resp, err := m.httpClient.Get(m.apiBase + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	var listResp ollamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, len(listResp.Models))
	for i, model := range listResp.Models {
		models[i] = ModelInfo{
			Name:     model.Name,
			Size:     model.Size,
			Path:     "", // Ollama manages paths internally
			LastUsed: model.ModifiedAt,
		}
	}

	return models, nil
}

// Download downloads a model with progress tracking
// Story T-022: Model download with progress, retry on network errors
func (m *OllamaManager) Download(modelName string, progressChan chan<- DownloadProgress) error {
	m.logger.Info("model.download.started", "Starting model download", map[string]interface{}{
		"provider": ProviderOllama,
		"model":    modelName,
	})

	// Send started event
	if progressChan != nil {
		progressChan <- DownloadProgress{
			ModelName: modelName,
			Status:    "started",
		}
	}

	// Prepare pull request
	pullReq := map[string]string{
		"name": modelName,
	}
	reqBody, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	resp, err := m.httpClient.Post(
		m.apiBase+"/api/pull",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		m.logger.Error("model.download.failed", "Failed to start download", map[string]interface{}{
			"model": modelName,
			"error": err.Error(),
		})
		if progressChan != nil {
			progressChan <- DownloadProgress{
				ModelName: modelName,
				Status:    "failed",
				Error:     err.Error(),
			}
		}
		return fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	// Read progress stream
	scanner := bufio.NewScanner(resp.Body)
	var lastProgress ollamaPullProgress

	for scanner.Scan() {
		line := scanner.Bytes()

		var progress ollamaPullProgress
		if err := json.Unmarshal(line, &progress); err != nil {
			m.logger.Warn("model.download.parse_error", "Failed to parse progress", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		lastProgress = progress

		// Send progress update
		if progressChan != nil {
			downloadProgress := DownloadProgress{
				ModelName:       modelName,
				BytesDownloaded: progress.Completed,
				TotalBytes:      progress.Total,
				Status:          "progress",
			}

			if progress.Total > 0 {
				downloadProgress.Percentage = float64(progress.Completed) / float64(progress.Total) * 100
			}

			progressChan <- downloadProgress
		}

		// Check for errors
		if progress.Error != "" {
			m.logger.Error("model.download.failed", "Download failed", map[string]interface{}{
				"model": modelName,
				"error": progress.Error,
			})
			if progressChan != nil {
				progressChan <- DownloadProgress{
					ModelName: modelName,
					Status:    "failed",
					Error:     progress.Error,
				}
			}
			return fmt.Errorf("download failed: %s", progress.Error)
		}
	}

	if err := scanner.Err(); err != nil {
		m.logger.Error("model.download.stream_error", "Stream error", map[string]interface{}{
			"model": modelName,
			"error": err.Error(),
		})
		return fmt.Errorf("stream error: %w", err)
	}

	// Get final model info
	models, err := m.List()
	if err != nil {
		m.logger.Warn("model.download.list_failed", "Failed to list models after download", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		// Update state with downloaded model
		for _, model := range models {
			if model.Name == modelName {
				model.LastUsed = time.Now().UTC()
				if err := m.stateManager.AddModel(model); err != nil {
					m.logger.Warn("model.download.state_update_failed", "Failed to update state", map[string]interface{}{
						"error": err.Error(),
					})
				}
				break
			}
		}
	}

	m.logger.Info("model.download.completed", "Model download completed", map[string]interface{}{
		"model":  modelName,
		"status": lastProgress.Status,
	})

	// Send completed event
	if progressChan != nil {
		progressChan <- DownloadProgress{
			ModelName:  modelName,
			Status:     "completed",
			Percentage: 100,
		}
	}

	return nil
}

// Delete removes a model
func (m *OllamaManager) Delete(modelName string) error {
	m.logger.Info("model.delete.started", "Deleting model", map[string]interface{}{
		"provider": ProviderOllama,
		"model":    modelName,
	})

	deleteReq := map[string]string{
		"name": modelName,
	}
	reqBody, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequest(
		"DELETE",
		m.apiBase+"/api/delete",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Remove from state
	if err := m.stateManager.RemoveModel(modelName); err != nil {
		m.logger.Warn("model.delete.state_update_failed", "Failed to update state", map[string]interface{}{
			"error": err.Error(),
		})
	}

	m.logger.Info("model.delete.completed", "Model deleted", map[string]interface{}{
		"model": modelName,
	})

	return nil
}

// GetStats returns cache statistics
// Story T-023: Cache overview
func (m *OllamaManager) GetStats() (*CacheStats, error) {
	// Sync state with actual models
	if err := m.SyncState(); err != nil {
		m.logger.Warn("model.stats.sync_failed", "Failed to sync state", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return m.stateManager.GetStats()
}

// SyncState synchronizes the state with actual Ollama models
func (m *OllamaManager) SyncState() error {
	models, err := m.List()
	if err != nil {
		return err
	}

	// Load current state
	state, err := m.stateManager.Load()
	if err != nil {
		return err
	}

	// Create a map of current state for quick lookup
	stateMap := make(map[string]ModelInfo)
	for _, model := range state.Items {
		stateMap[model.Name] = model
	}

	// Update state with current models, preserving last_used if available
	newItems := make([]ModelInfo, 0, len(models))
	for _, model := range models {
		if existing, ok := stateMap[model.Name]; ok {
			// Preserve last_used from existing state
			model.LastUsed = existing.LastUsed
		}
		newItems = append(newItems, model)
	}

	state.Items = newItems
	return m.stateManager.Save(state)
}

// EvictOldest removes the oldest model to free up space
// Story T-023: Evict oldest functionality
func (m *OllamaManager) EvictOldest() (*ModelInfo, error) {
	// Sync state first
	if err := m.SyncState(); err != nil {
		return nil, err
	}

	oldestModels, err := m.stateManager.GetOldestModels()
	if err != nil {
		return nil, err
	}

	if len(oldestModels) == 0 {
		return nil, fmt.Errorf("no models to evict")
	}

	oldest := oldestModels[0]

	m.logger.Info("model.evict.started", "Evicting oldest model", map[string]interface{}{
		"model":     oldest.Name,
		"last_used": oldest.LastUsed,
		"size":      oldest.Size,
	})

	// Delete the model
	if err := m.Delete(oldest.Name); err != nil {
		return nil, fmt.Errorf("failed to delete oldest model: %w", err)
	}

	m.logger.Info("model.evict.completed", "Oldest model evicted", map[string]interface{}{
		"model": oldest.Name,
		"size":  oldest.Size,
	})

	return &oldest, nil
}
