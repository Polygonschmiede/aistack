package tui

import "time"

// Screen represents different TUI screens
// Story T-024: Main menu & navigation
type Screen string

const (
	// ScreenMenu is the main menu screen
	ScreenMenu Screen = "menu"
	// ScreenStatus shows service status
	ScreenStatus Screen = "status"
	// ScreenInstall shows install/uninstall options
	ScreenInstall Screen = "install"
	// ScreenModels shows model management
	ScreenModels Screen = "models"
	// ScreenLogs shows service logs
	ScreenLogs Screen = "logs"
	// ScreenDiagnostics shows diagnostics
	ScreenDiagnostics Screen = "diagnostics"
	// ScreenSettings shows settings
	ScreenSettings Screen = "settings"
	// ScreenHelp shows help overlay
	ScreenHelp Screen = "help"
)

// MenuItem represents a menu item
type MenuItem struct {
	Key         string // Number key (1-8) or letter
	Label       string // Display label
	Description string // Short description
	Screen      Screen // Target screen
}

// UIState represents the persisted UI state
// Data Contract from EP-013: ui_state.json
type UIState struct {
	CurrentScreen Screen    `json:"menu"`       // Current screen
	Selection     int       `json:"selection"`  // Current menu selection index
	LastError     string    `json:"last_error"` // Last error message
	Updated       time.Time `json:"updated"`    // Last update timestamp
}

// DefaultMenuItems returns the default main menu items
func DefaultMenuItems() []MenuItem {
	return []MenuItem{
		{Key: "1", Label: "Status", Description: "View service status", Screen: ScreenStatus},
		{Key: "2", Label: "Install/Uninstall", Description: "Manage service installation", Screen: ScreenInstall},
		{Key: "3", Label: "Models", Description: "Model management", Screen: ScreenModels},
		{Key: "4", Label: "Logs", Description: "View service logs", Screen: ScreenLogs},
		{Key: "5", Label: "Diagnostics", Description: "Run diagnostics", Screen: ScreenDiagnostics},
		{Key: "6", Label: "Settings", Description: "Configure aistack", Screen: ScreenSettings},
		{Key: "?", Label: "Help", Description: "Show help", Screen: ScreenHelp},
	}
}
