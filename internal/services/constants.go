package services

import "aistack/internal/fsutil"

const (
	defaultStateDir      = fsutil.DefaultStateDir
	planStatusPending    = "pending"
	planStatusCompleted  = "completed"
	planStatusFailed     = "failed"
	planStatusRolledBack = "rolled_back"
)

const (
	backendStateFilename = "ui_binding.json"
)
