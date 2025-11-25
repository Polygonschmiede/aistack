package models

// SyncStateWithModels synchronizes the state with a list of current models,
// preserving LastUsed timestamps from existing state.
//
// This helper function extracts the common pattern used by both OllamaManager
// and LocalAIManager to avoid code duplication.
func SyncStateWithModels(stateManager *StateManager, currentModels []ModelInfo) error {
	// Load current state to preserve last_used timestamps
	state, err := stateManager.Load()
	if err != nil {
		return err
	}

	// Create a map of current state for quick lookup
	stateMap := make(map[string]ModelInfo)
	for _, model := range state.Items {
		stateMap[model.Name] = model
	}

	// Update state with current models, preserving last_used if available
	newItems := make([]ModelInfo, 0, len(currentModels))
	for _, model := range currentModels {
		if existing, ok := stateMap[model.Name]; ok {
			// Preserve last_used from existing state if it's newer
			if existing.LastUsed.After(model.LastUsed) {
				model.LastUsed = existing.LastUsed
			}
		}
		newItems = append(newItems, model)
	}

	state.Items = newItems
	return stateManager.Save(state)
}
