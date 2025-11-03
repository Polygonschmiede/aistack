package lipgloss

// Color represents a terminal color. It is kept as a string in the stub
// implementation because no actual styling is performed.
type Color string

// Style provides a fluent API similar to the original lipgloss package. The
// fields are retained for API compatibility only.
type Style struct {
	bold          bool
	foreground    Color
	paddingTop    int
	paddingBottom int
}

// NewStyle creates a new Style value.
func NewStyle() Style {
	return Style{}
}

// Bold sets whether the rendered text should be bold.
func (s Style) Bold(enabled bool) Style {
	s.bold = enabled
	return s
}

// Foreground sets the foreground color.
func (s Style) Foreground(color Color) Style {
	s.foreground = color
	return s
}

// PaddingTop sets the top padding.
func (s Style) PaddingTop(padding int) Style {
	s.paddingTop = padding
	return s
}

// PaddingBottom sets the bottom padding.
func (s Style) PaddingBottom(padding int) Style {
	s.paddingBottom = padding
	return s
}

// Render returns the provided content without any styling. The stub keeps the
// method for API compatibility while avoiding external dependencies.
func (s Style) Render(content string) string {
	return content
}
