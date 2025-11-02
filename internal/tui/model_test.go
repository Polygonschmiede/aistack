package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	if m.startTime.IsZero() {
		t.Error("Expected startTime to be set, got zero time")
	}

	if m.quitting {
		t.Error("Expected quitting to be false initially")
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	if cmd != nil {
		t.Error("Expected Init to return nil command")
	}
}

func TestModelUpdate_QuitOnQ(t *testing.T) {
	m := NewModel()

	// Test 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := m.Update(msg)

	updatedM, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("Expected Model type from Update")
	}

	if !updatedM.quitting {
		t.Error("Expected quitting to be true after 'q' key")
	}

	if cmd == nil {
		t.Error("Expected quit command to be returned")
	}
}

func TestModelUpdate_QuitOnCtrlC(t *testing.T) {
	m := NewModel()

	// Test Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedModel, cmd := m.Update(msg)

	updatedM, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("Expected Model type from Update")
	}

	if !updatedM.quitting {
		t.Error("Expected quitting to be true after Ctrl+C")
	}

	if cmd == nil {
		t.Error("Expected quit command to be returned")
	}
}

func TestModelUpdate_OtherKey(t *testing.T) {
	m := NewModel()

	// Test other key (should not quit)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	updatedModel, cmd := m.Update(msg)

	updatedM, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("Expected Model type from Update")
	}

	if updatedM.quitting {
		t.Error("Expected quitting to remain false for non-quit key")
	}

	if cmd != nil {
		t.Error("Expected no command for non-quit key")
	}
}

func TestModelView_NotQuitting(t *testing.T) {
	m := NewModel()
	view := m.View()

	// Check that view contains expected elements
	expectedStrings := []string{"aistack", "Press 'q'", "quit"}

	for _, expected := range expectedStrings {
		if !strings.Contains(view, expected) {
			t.Errorf("Expected view to contain %q, but it didn't.\nView: %s", expected, view)
		}
	}

	if view == "" {
		t.Error("Expected non-empty view when not quitting")
	}
}

func TestModelView_Quitting(t *testing.T) {
	m := NewModel()
	m.quitting = true
	view := m.View()

	if view != "" {
		t.Errorf("Expected empty view when quitting, got: %s", view)
	}
}
