package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aistack/internal/gpu"
	"aistack/internal/idle"
	"aistack/internal/logging"
	"aistack/internal/services"
)

// Model represents the TUI application state
type Model struct {
	startTime time.Time
	quitting  bool

	logger     *logging.Logger
	composeDir string
	stateDir   string

	gpuReport    gpu.GPUReport
	hasGPUReport bool
	gpuError     string

	idleState    idle.IdleState
	hasIdleState bool
	idleError    string

	backend      services.BackendType
	backendURL   string
	backendError string

	statusMessage string
}

// NewModel creates a new TUI model with preloaded system insights
func NewModel(logger *logging.Logger, composeDir string) Model {
	m := Model{
		startTime:  time.Now(),
		logger:     logger,
		composeDir: composeDir,
	}

	m.loadIdleState()
	m.loadBackend()
	m.loadGPU()

	return m
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
		case "b":
			m = m.toggleBackend()
		case "r":
			m = m.refresh()
		}
	}
	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d7ff")).PaddingTop(1)
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd700")).MarginTop(1)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#87d7af"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f5f"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d7d7d7")).MarginTop(1)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("aistack"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("AI Stack Manager"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("Uptime: "))
	b.WriteString(valueStyle.Render(m.prettyDuration(time.Since(m.startTime))))
	b.WriteString("\n\n")

	b.WriteString(sectionStyle.Render("GPU Readiness"))
	b.WriteString("\n")
	b.WriteString(m.renderGPUSection(labelStyle, valueStyle, errorStyle))

	b.WriteString(sectionStyle.Render("Idle Timer"))
	b.WriteString("\n")
	b.WriteString(m.renderIdleSection(labelStyle, valueStyle, errorStyle))

	b.WriteString(sectionStyle.Render("Backend Binding"))
	b.WriteString("\n")
	b.WriteString(m.renderBackendSection(labelStyle, valueStyle, errorStyle))

	if m.statusMessage != "" {
		b.WriteString(statusStyle.Render(m.statusMessage))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("Press 'b' to toggle backend, 'r' to refresh, 'q' to quit"))
	b.WriteString("\n")

	return b.String()
}

func (m *Model) loadGPU() {
	if os.Getenv("AISTACK_DISABLE_GPU_SCAN") == "1" {
		m.hasGPUReport = false
		m.gpuError = "GPU scan disabled"
		return
	}

	detector := gpu.NewDetector(m.logger)
	report := detector.DetectGPUs()
	m.gpuReport = report
	m.hasGPUReport = true

	if report.ErrorMessage != "" {
		m.gpuError = report.ErrorMessage
		return
	}

	if !report.NVMLOk {
		m.gpuError = "NVML unavailable or failed to initialize"
		return
	}

	m.gpuError = ""
}

func (m *Model) loadIdleState() {
	idleConfig := idle.DefaultIdleConfig()
	m.stateDir = filepath.Dir(idleConfig.StateFilePath)
	manager := idle.NewStateManager(idleConfig.StateFilePath, m.logger)

	state, err := manager.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.idleError = "Idle state not recorded yet"
		} else {
			m.idleError = err.Error()
		}
		m.hasIdleState = false
		return
	}

	m.idleState = state
	m.hasIdleState = true
	m.idleError = ""
}

func (m *Model) loadBackend() {
	stateDir := m.stateDir
	if stateDir == "" {
		stateDir = os.Getenv("AISTACK_STATE_DIR")
		if stateDir == "" {
			stateDir = "/var/lib/aistack"
		}
	}

	manager := services.NewBackendBindingManager(stateDir, m.logger)
	binding, err := manager.GetBinding()
	if err != nil {
		m.backendError = err.Error()
		m.backend = ""
		m.backendURL = ""
		return
	}

	m.backend = binding.ActiveBackend
	m.backendURL = binding.URL
	m.backendError = ""
}

func (m Model) toggleBackend() Model {
	if m.composeDir == "" {
		m.statusMessage = "Compose directory not resolved"
		return m
	}

	manager, err := services.NewManager(m.composeDir, m.logger)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	service, err := manager.GetService("openwebui")
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	openwebui, ok := service.(*services.OpenWebUIService)
	if !ok {
		m.statusMessage = "Backend toggle failed: unexpected service type"
		return m
	}

	current, err := openwebui.GetCurrentBackend()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	var target services.BackendType
	if current == services.BackendLocalAI {
		target = services.BackendOllama
	} else {
		target = services.BackendLocalAI
	}

	if err = openwebui.SwitchBackend(target); err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	m.backend = target
	url, err := services.GetBackendURL(target)
	if err == nil {
		m.backendURL = url
	}
	m.backendError = ""
	m.statusMessage = fmt.Sprintf("Switched backend to %s", string(target))

	return m
}

func (m Model) refresh() Model {
	m.loadIdleState()
	m.loadBackend()
	m.loadGPU()
	m.statusMessage = "Refreshed system state"
	return m
}

func (m Model) renderGPUSection(labelStyle, valueStyle, errorStyle lipgloss.Style) string {
	if m.gpuError != "" {
		return errorStyle.Render(m.gpuError) + "\n"
	}

	if !m.hasGPUReport || len(m.gpuReport.GPUs) == 0 {
		return valueStyle.Render("No GPUs detected") + "\n"
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render("Driver: "))
	b.WriteString(valueStyle.Render(m.gpuReport.DriverVersion))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("CUDA: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.gpuReport.CUDAVersion)))
	b.WriteString("\n")

	for _, gpuInfo := range m.gpuReport.GPUs {
		b.WriteString(fmt.Sprintf("  • %s (%d MB)\n", valueStyle.Render(gpuInfo.Name), gpuInfo.MemoryMB))
	}

	return b.String()
}

func (m Model) renderIdleSection(labelStyle, valueStyle, errorStyle lipgloss.Style) string {
	if m.idleError != "" {
		return errorStyle.Render(m.idleError) + "\n"
	}

	if !m.hasIdleState {
		return valueStyle.Render("Idle engine warming up") + "\n"
	}

	state := m.idleState
	var b strings.Builder
	b.WriteString(labelStyle.Render("Status: "))
	b.WriteString(valueStyle.Render(capitalize(state.Status)))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("Idle for: "))
	b.WriteString(valueStyle.Render(m.prettyDuration(time.Duration(state.IdleForSeconds) * time.Second)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("CPU idle: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%.1f%%", state.CPUIdlePct)))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("GPU idle: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%.1f%%", state.GPUIdlePct)))
	b.WriteString("\n")

	if len(state.GatingReasons) > 0 {
		b.WriteString(labelStyle.Render("Gating: "))
		b.WriteString(valueStyle.Render(strings.Join(state.GatingReasons, ", ")))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderBackendSection(labelStyle, valueStyle, errorStyle lipgloss.Style) string {
	if m.backendError != "" {
		return errorStyle.Render(m.backendError) + "\n"
	}

	backend := m.backend
	if backend == "" {
		backend = services.BackendOllama
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render("Active backend: "))
	b.WriteString(valueStyle.Render(string(backend)))
	b.WriteString("  ")
	b.WriteString(labelStyle.Render("URL: "))
	b.WriteString(valueStyle.Render(m.backendURL))
	b.WriteString("\n")

	return b.String()
}

func (m Model) prettyDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	return d.Truncate(time.Second).String()
}

func capitalize(input string) string {
	if input == "" {
		return ""
	}
	runes := []rune(input)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
