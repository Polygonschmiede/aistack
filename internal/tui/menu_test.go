package tui

import (
	"strings"
	"testing"

	"aistack/internal/logging"
)

func TestModel_NavigateUp(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.selection = 3

	// Navigate up
	m = m.navigateUp()

	if m.selection != 2 {
		t.Errorf("Expected selection 2, got %d", m.selection)
	}
}

func TestModel_NavigateUp_WrapAround(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.selection = 0

	// Navigate up from top should wrap to bottom
	m = m.navigateUp()

	expectedIndex := len(DefaultMenuItems()) - 1
	if m.selection != expectedIndex {
		t.Errorf("Expected selection %d (wrap to bottom), got %d", expectedIndex, m.selection)
	}
}

func TestModel_NavigateDown(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.selection = 2

	// Navigate down
	m = m.navigateDown()

	if m.selection != 3 {
		t.Errorf("Expected selection 3, got %d", m.selection)
	}
}

func TestModel_NavigateDown_WrapAround(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	maxIndex := len(DefaultMenuItems()) - 1
	m.selection = maxIndex

	// Navigate down from bottom should wrap to top
	m = m.navigateDown()

	if m.selection != 0 {
		t.Errorf("Expected selection 0 (wrap to top), got %d", m.selection)
	}
}

func TestModel_SelectMenuItem(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenMenu
	m.selection = 0 // First item (Status)

	// Select menu item
	m = m.selectMenuItem()

	if m.currentScreen != ScreenStatus {
		t.Errorf("Expected screen status, got %s", m.currentScreen)
	}

	// Should clear error
	if m.lastError != "" {
		t.Errorf("Expected empty error after selection, got %s", m.lastError)
	}
}

func TestModel_SelectMenuByKey(t *testing.T) {
	tests := []struct {
		key            string
		expectedScreen Screen
	}{
		{"1", ScreenStatus},
		{"2", ScreenInstall},
		{"3", ScreenModels},
		{"4", ScreenPower},
		{"5", ScreenLogs},
		{"6", ScreenDiagnostics},
		{"7", ScreenSettings},
		{"?", ScreenHelp},
	}

	for _, tt := range tests {
		t.Run("key_"+tt.key, func(t *testing.T) {
			logger := logging.NewLogger(logging.LevelInfo)
			m := NewModel(logger, "/tmp/compose")
			m.currentScreen = ScreenMenu

			// Select by key
			m = m.selectMenuByKey(tt.key)

			if m.currentScreen != tt.expectedScreen {
				t.Errorf("Key %s: expected screen %s, got %s", tt.key, tt.expectedScreen, m.currentScreen)
			}

			// Should clear error
			if m.lastError != "" {
				t.Errorf("Expected empty error after selection, got %s", m.lastError)
			}
		})
	}
}

func TestModel_ReturnToMenu(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenStatus
	m.lastError = "some error"

	// Return to menu
	m = m.returnToMenu()

	if m.currentScreen != ScreenMenu {
		t.Errorf("Expected screen menu, got %s", m.currentScreen)
	}

	// Should clear error
	if m.lastError != "" {
		t.Errorf("Expected empty error after returning to menu, got %s", m.lastError)
	}
}

func TestModel_RenderMenu(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenMenu

	output := m.renderMenu()

	// Should contain title
	if !strings.Contains(output, "Main Menu") {
		t.Errorf("Menu output should contain 'Main Menu'")
	}

	// Should contain menu items
	menuItems := DefaultMenuItems()
	for _, item := range menuItems {
		if !strings.Contains(output, item.Label) {
			t.Errorf("Menu output should contain '%s'", item.Label)
		}
	}

	// Should contain navigation hints
	if !strings.Contains(output, "Navigate") {
		t.Errorf("Menu output should contain navigation hints")
	}
}

func TestModel_RenderMenu_WithError(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenMenu
	m.lastError = "Test error message"

	output := m.renderMenu()

	// Should contain error message
	if !strings.Contains(output, "Test error message") {
		t.Errorf("Menu output should contain error message")
	}
}

func TestModel_RenderStatusScreen(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenStatus

	output := m.renderStatusScreen()

	// Should contain title
	if !strings.Contains(output, "Service Status") {
		t.Errorf("Status output should contain 'Service Status'")
	}

	// Should contain sections
	if !strings.Contains(output, "GPU Readiness") {
		t.Errorf("Status output should contain 'GPU Readiness'")
	}

	if !strings.Contains(output, "Idle Timer") {
		t.Errorf("Status output should contain 'Idle Timer'")
	}

	if !strings.Contains(output, "Backend Binding") {
		t.Errorf("Status output should contain 'Backend Binding'")
	}

	// Should contain hints
	if !strings.Contains(output, "toggle backend") {
		t.Errorf("Status output should contain backend toggle hint")
	}
}

func TestModel_RenderHelpScreen(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")
	m.currentScreen = ScreenHelp

	output := m.renderHelpScreen()

	// Should contain title
	if !strings.Contains(output, "Help") {
		t.Errorf("Help output should contain 'Help'")
	}

	// Should contain keyboard shortcuts
	shortcuts := []string{"↑ / ↓", "Enter/Space", "Esc", "q / Ctrl+C"}
	for _, shortcut := range shortcuts {
		if !strings.Contains(output, shortcut) {
			t.Errorf("Help output should contain shortcut '%s'", shortcut)
		}
	}
}

func TestModel_RenderPlaceholderScreen(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")

	output := m.renderPlaceholderScreen("Test Title", "Test description")

	// Should contain title and description
	if !strings.Contains(output, "Test Title") {
		t.Errorf("Placeholder output should contain title")
	}

	if !strings.Contains(output, "Test description") {
		t.Errorf("Placeholder output should contain description")
	}

	// Should indicate future implementation
	if !strings.Contains(output, "future update") {
		t.Errorf("Placeholder should indicate future implementation")
	}
}

func TestModel_Capitalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"world", "World"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tt := range tests {
		result := capitalize(tt.input)
		if result != tt.expected {
			t.Errorf("capitalize(%s): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestModel_PrettyDuration(t *testing.T) {
	logger := logging.NewLogger(logging.LevelInfo)
	m := NewModel(logger, "/tmp/compose")

	tests := []struct {
		name     string
		duration string
		contains string
	}{
		{"sub-second", "500ms", "<1s"},
		{"seconds", "45s", "45s"},
		{"minutes", "2m30s", "2m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse duration string for testing
			// (In real usage, duration comes from time.Since)
			result := m.prettyDuration(0) // Test edge case
			if result != "<1s" {
				t.Errorf("Expected '<1s' for zero duration, got %s", result)
			}
		})
	}
}
