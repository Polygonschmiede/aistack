package tui

import (
	"errors"
	"fmt"
	"os"
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
// Story T-024: Enhanced with menu system and navigation
type Model struct {
	startTime time.Time
	quitting  bool

	logger     *logging.Logger
	composeDir string
	stateDir   string

	// UI State
	currentScreen Screen
	selection     int
	lastError     string
	stateManager  *UIStateManager

	// System State
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
// Story T-024: Initializes with main menu screen
func NewModel(logger *logging.Logger, composeDir string) Model {
	// Determine state directory
	stateDir := os.Getenv("AISTACK_STATE_DIR")
	if stateDir == "" {
		stateDir = "/var/lib/aistack"
	}

	m := Model{
		startTime:     time.Now(),
		logger:        logger,
		composeDir:    composeDir,
		stateDir:      stateDir,
		currentScreen: ScreenMenu,
		selection:     0,
		stateManager:  NewUIStateManager(stateDir, logger),
	}

	// Load persisted UI state
	if state, err := m.stateManager.Load(); err == nil {
		m.currentScreen = state.CurrentScreen
		m.selection = state.Selection
		m.lastError = state.LastError
	}

	// Load system state
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
// Story T-024: Enhanced with menu navigation
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			m.saveState()
			return m, tea.Quit

		case "esc":
			if m.currentScreen != ScreenMenu {
				m = m.returnToMenu()
				m.saveState()
			}

		// Menu navigation (only on menu screen)
		case "up", "k":
			if m.currentScreen == ScreenMenu {
				m = m.navigateUp()
			}

		case "down", "j":
			if m.currentScreen == ScreenMenu {
				m = m.navigateDown()
			}

		case "enter", " ":
			if m.currentScreen == ScreenMenu {
				m = m.selectMenuItem()
				m.saveState()
			}

		// Number key shortcuts (work from any screen)
		case "1", "2", "3", "4", "5", "6", "7", "?":
			m = m.selectMenuByKey(msg.String())
			m.saveState()

		// Screen-specific actions
		case "b":
			if m.currentScreen == ScreenStatus {
				m = m.toggleBackend()
			}

		case "r":
			if m.currentScreen == ScreenStatus {
				m = m.refresh()
			}
		}
	}
	return m, nil
}

// View renders the TUI
// Story T-024: Routes to appropriate screen renderer
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.currentScreen {
	case ScreenMenu:
		return m.renderMenu()
	case ScreenStatus:
		return m.renderStatusScreen()
	case ScreenInstall:
		return m.renderPlaceholderScreen("Install/Uninstall Services", "Manage service installation and removal.")
	case ScreenModels:
		return m.renderPlaceholderScreen("Model Management", "Download, list, and manage AI models.")
	case ScreenPower:
		return m.renderPlaceholderScreen("Power Management", "Configure idle detection and auto-suspend.")
	case ScreenLogs:
		return m.renderPlaceholderScreen("Service Logs", "View and tail service logs.")
	case ScreenDiagnostics:
		return m.renderPlaceholderScreen("Diagnostics", "Run system health checks and diagnostics.")
	case ScreenSettings:
		return m.renderPlaceholderScreen("Settings", "Configure aistack settings.")
	case ScreenHelp:
		return m.renderHelpScreen()
	default:
		return m.renderMenu()
	}
}

// saveState persists the current UI state
func (m *Model) saveState() {
	state := &UIState{
		CurrentScreen: m.currentScreen,
		Selection:     m.selection,
		LastError:     m.lastError,
		Updated:       time.Now().UTC(),
	}

	if err := m.stateManager.Save(state); err != nil {
		m.logger.Warn("tui.state.save_failed", "Failed to save UI state", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// loadGPU loads GPU information
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

// loadIdleState loads idle engine state
func (m *Model) loadIdleState() {
	idleConfig := idle.DefaultIdleConfig()
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

// loadBackend loads backend binding information
func (m *Model) loadBackend() {
	manager := services.NewBackendBindingManager(m.stateDir, m.logger)
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

// toggleBackend toggles the backend between Ollama and LocalAI
func (m Model) toggleBackend() Model {
	if m.composeDir == "" {
		m.statusMessage = "Compose directory not resolved"
		m.lastError = "Compose directory not resolved"
		return m
	}

	manager, err := services.NewManager(m.composeDir, m.logger)
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		m.lastError = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	service, err := manager.GetService("openwebui")
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		m.lastError = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	openwebui, ok := service.(*services.OpenWebUIService)
	if !ok {
		m.statusMessage = "Backend toggle failed: unexpected service type"
		m.lastError = "Backend toggle failed: unexpected service type"
		return m
	}

	current, err := openwebui.GetCurrentBackend()
	if err != nil {
		m.statusMessage = fmt.Sprintf("Backend toggle failed: %v", err)
		m.lastError = fmt.Sprintf("Backend toggle failed: %v", err)
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
		m.lastError = fmt.Sprintf("Backend toggle failed: %v", err)
		return m
	}

	m.backend = target
	url, err := services.GetBackendURL(target)
	if err == nil {
		m.backendURL = url
	}
	m.backendError = ""
	m.statusMessage = fmt.Sprintf("Switched backend to %s", string(target))
	m.lastError = "" // Clear error on success

	return m
}

// refresh refreshes all system state
func (m Model) refresh() Model {
	m.loadIdleState()
	m.loadBackend()
	m.loadGPU()
	m.statusMessage = "Refreshed system state"
	m.lastError = "" // Clear error on refresh
	return m
}

// renderGPUSection renders the GPU section
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

// renderIdleSection renders the idle section
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

// renderBackendSection renders the backend section
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

// prettyDuration formats a duration for display
func (m Model) prettyDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	return d.Truncate(time.Second).String()
}

// capitalize capitalizes the first letter of a string
func capitalize(input string) string {
	if input == "" {
		return ""
	}
	runes := []rune(input)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
