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

func closeBody(body io.ReadCloser, logger *logging.Logger, context string) {
	if err := body.Close(); err != nil {
		logger.Warn("ollama.response.close_failed", "Failed to close HTTP response body", map[string]interface{}{
			"context": context,
			"error":   err.Error(),
		})
	}
}

// List returns all available Ollama models
func (m *OllamaManager) List() ([]ModelInfo, error) {
	resp, err := m.httpClient.Get(m.apiBase + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer closeBody(resp.Body, m.logger, "list")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	var listResp ollamaListResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&listResp); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode response: %w", decodeErr)
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
	defer closeBody(resp.Body, m.logger, "download")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama API returned status %d", resp.StatusCode)
	}

	lastProgress, err := m.consumeDownloadProgress(resp.Body, modelName, progressChan)
	if err != nil {
		return err
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
				if addErr := m.stateManager.AddModel(model); addErr != nil {
					m.logger.Warn("model.download.state_update_failed", "Failed to update state", map[string]interface{}{
						"error": addErr.Error(),
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

func (m *OllamaManager) consumeDownloadProgress(body io.Reader, modelName string, progressChan chan<- DownloadProgress) (ollamaPullProgress, error) {
	scanner := bufio.NewScanner(body)
	var lastProgress ollamaPullProgress

	for scanner.Scan() {
		progress, ok := m.parseProgressLine(scanner.Bytes())
		if !ok {
			continue
		}

		lastProgress = progress
		m.emitProgressEvent(progress, modelName, progressChan)

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
			return lastProgress, fmt.Errorf("download failed: %s", progress.Error)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		m.logger.Error("model.download.stream_error", "Stream error", map[string]interface{}{
			"model": modelName,
			"error": scanErr.Error(),
		})
		return lastProgress, fmt.Errorf("stream error: %w", scanErr)
	}

	return lastProgress, nil
}

func (m *OllamaManager) parseProgressLine(line []byte) (ollamaPullProgress, bool) {
	var progress ollamaPullProgress
	if unmarshalErr := json.Unmarshal(line, &progress); unmarshalErr != nil {
		m.logger.Warn("model.download.parse_error", "Failed to parse progress", map[string]interface{}{
			"error": unmarshalErr.Error(),
		})
		return ollamaPullProgress{}, false
	}
	return progress, true
}

func (m *OllamaManager) emitProgressEvent(progress ollamaPullProgress, modelName string, progressChan chan<- DownloadProgress) {
	if progressChan == nil {
		return
	}

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
	defer closeBody(resp.Body, m.logger, "delete")

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("ollama API returned status %d and failed to read body: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Remove from state
	if removeErr := m.stateManager.RemoveModel(modelName); removeErr != nil {
		m.logger.Warn("model.delete.state_update_failed", "Failed to update state", map[string]interface{}{
			"error": removeErr.Error(),
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

	return SyncStateWithModels(m.stateManager, models)
}

// EvictOldest removes the oldest model to free up space
// Story T-023: Evict oldest functionality
func (m *OllamaManager) EvictOldest() (*ModelInfo, error) {
	return evictOldestModel(m.SyncState, m.stateManager, m.Delete, m.logger)
}
