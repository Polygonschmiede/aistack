module aistack

go 1.22

toolchain go1.22.0

require (
        github.com/charmbracelet/bubbletea v0.25.0
        github.com/charmbracelet/lipgloss v0.9.1
)

replace github.com/charmbracelet/bubbletea => ./third_party/bubbletea

replace github.com/charmbracelet/lipgloss => ./third_party/lipgloss
