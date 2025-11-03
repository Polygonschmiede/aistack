package services

const (
	defaultStateDir      = "/var/lib/aistack"
	planStatusPending    = "pending"
	planStatusCompleted  = "completed"
	planStatusFailed     = "failed"
	planStatusRolledBack = "rolled_back"
)

const (
	backendStateFilename = "ui_binding.json"
)
