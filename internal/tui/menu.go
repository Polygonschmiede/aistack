package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderMenu renders the main menu screen
// Story T-024: Main menu with keyboard navigation
func (m Model) renderMenu() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff")).MarginBottom(1)
	menuItemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	menuItemSelectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#00d7ff")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")).PaddingLeft(2)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff")).MarginTop(1)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f")).Bold(true).MarginTop(1)

	b.WriteString(titleStyle.Render("aistack — Main Menu"))
	b.WriteString("\n\n")

	menuItems := DefaultMenuItems()

	for i, item := range menuItems {
		prefix := fmt.Sprintf("[%s] ", item.Key)

		var itemText string
		if i == m.selection {
			itemText = menuItemSelectedStyle.Render(prefix + item.Label)
		} else {
			itemText = menuItemStyle.Render(prefix + item.Label)
		}

		b.WriteString(itemText)
		b.WriteString("\n")
		b.WriteString(descStyle.Render(item.Description))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Navigate: ↑/↓ or numbers | Select: Enter/Space | Back: Esc | Quit: q"))
	b.WriteString("\n")

	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("⚠ " + m.lastError))
		b.WriteString("\n")
	}

	return b.String()
}

// renderStatusScreen renders the status screen
func (m Model) renderStatusScreen() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff")).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd700")).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7af"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff")).MarginTop(1)

	b.WriteString(titleStyle.Render("Service Status"))
	b.WriteString("\n\n")

	// GPU Section
	b.WriteString(sectionStyle.Render("GPU Readiness"))
	b.WriteString("\n")
	b.WriteString(m.renderGPUSection(labelStyle, valueStyle, errorStyle))

	// Idle Section
	b.WriteString(sectionStyle.Render("Idle Timer"))
	b.WriteString("\n")
	b.WriteString(m.renderIdleSection(labelStyle, valueStyle, errorStyle))

	// Backend Section
	b.WriteString(sectionStyle.Render("Backend Binding"))
	b.WriteString("\n")
	b.WriteString(m.renderBackendSection(labelStyle, valueStyle, errorStyle))

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press 'b' to toggle backend, 'r' to refresh, Esc to return to menu, 'q' to quit"))
	b.WriteString("\n")

	return b.String()
}

// renderPlaceholderScreen renders a placeholder screen for not-yet-implemented features
func (m Model) renderPlaceholderScreen(title, description string) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff")).MarginBottom(1)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).MarginTop(1)
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff")).MarginTop(2)

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")
	b.WriteString(textStyle.Render(description))
	b.WriteString("\n")
	b.WriteString(textStyle.Render("This feature will be implemented in a future update."))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press Esc to return to menu, 'q' to quit"))
	b.WriteString("\n")

	return b.String()
}

// renderHelpScreen renders the help screen
func (m Model) renderHelpScreen() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff")).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd700")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7af")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff")).MarginTop(2)

	b.WriteString(titleStyle.Render("Help — Keyboard Shortcuts"))
	b.WriteString("\n\n")

	b.WriteString(sectionStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("1-8, ?      "))
	b.WriteString(descStyle.Render("Quick menu selection by number/key"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("↑ / ↓       "))
	b.WriteString(descStyle.Render("Navigate menu items"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Enter/Space "))
	b.WriteString(descStyle.Render("Select highlighted item"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("Esc         "))
	b.WriteString(descStyle.Render("Return to main menu"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("q / Ctrl+C  "))
	b.WriteString(descStyle.Render("Quit aistack"))
	b.WriteString("\n")

	b.WriteString(sectionStyle.Render("Status Screen"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("b           "))
	b.WriteString(descStyle.Render("Toggle backend (Ollama ↔ LocalAI)"))
	b.WriteString("\n")
	b.WriteString(keyStyle.Render("r           "))
	b.WriteString(descStyle.Render("Refresh system state"))
	b.WriteString("\n")

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press Esc to return to menu"))
	b.WriteString("\n")

	return b.String()
}

// navigateUp moves selection up in the menu
func (m Model) navigateUp() Model {
	if m.selection > 0 {
		m.selection--
	} else {
		// Wrap to bottom
		m.selection = len(DefaultMenuItems()) - 1
	}
	return m
}

// navigateDown moves selection down in the menu
func (m Model) navigateDown() Model {
	maxIndex := len(DefaultMenuItems()) - 1
	if m.selection < maxIndex {
		m.selection++
	} else {
		// Wrap to top
		m.selection = 0
	}
	return m
}

// selectMenuItem handles menu item selection
func (m Model) selectMenuItem() Model {
	menuItems := DefaultMenuItems()
	if m.selection >= 0 && m.selection < len(menuItems) {
		m.currentScreen = menuItems[m.selection].Screen
		m.lastError = "" // Clear error on screen change
	}
	return m
}

// selectMenuByKey handles direct menu selection by key press (1-8, ?)
func (m Model) selectMenuByKey(key string) Model {
	menuItems := DefaultMenuItems()
	for i, item := range menuItems {
		if item.Key == key {
			m.selection = i
			m.currentScreen = item.Screen
			m.lastError = "" // Clear error on screen change
			break
		}
	}
	return m
}

// returnToMenu returns to the main menu
func (m Model) returnToMenu() Model {
	m.currentScreen = ScreenMenu
	m.lastError = "" // Clear error when returning to menu
	return m
}
