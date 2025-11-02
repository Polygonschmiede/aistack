package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the TUI application state
type Model struct {
	startTime time.Time
	quitting  bool
}

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		startTime: time.Now(),
		quitting:  false,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Define styles using Lip Gloss
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		PaddingTop(1).
		PaddingBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		PaddingTop(1)

	// Build the view
	title := titleStyle.Render("aistack")
	subtitle := subtitleStyle.Render("AI Stack Manager - TUI Interface")
	help := helpStyle.Render("Press 'q' or Ctrl+C to quit")

	return fmt.Sprintf("\n%s\n%s\n\n%s\n", title, subtitle, help)
}
