package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelInitializesFields(t *testing.T) {
	model := NewModel()
	if model.startTime.IsZero() {
		t.Fatal("expected start time to be initialized")
	}
	if model.quitting {
		t.Fatal("expected model to start in non-quitting state")
	}
}

func TestUpdateHandlesQuitKeys(t *testing.T) {
	model := NewModel()
	updated, cmd := model.Update(tea.KeyMsg{Value: "q"})
	if !updated.(Model).quitting {
		t.Fatal("expected quitting flag to be set")
	}
	if cmd == nil {
		t.Fatal("expected quit command to be returned")
	}
}

func TestViewContainsCopy(t *testing.T) {
	model := NewModel()
	view := model.View()
	if view == "" {
		t.Fatal("view output should not be empty")
	}
	expected := []string{"aistack", "AI Stack Manager", "Press 'q'"}
	for _, phrase := range expected {
		if !strings.Contains(view, phrase) {
			t.Fatalf("expected view to contain %q", phrase)
		}
	}
}
