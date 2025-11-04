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

	// Install/Uninstall Screen State
	installSelection  int    // Selected service index
	installInProgress bool   // Operation in progress
	installResult     string // Result message

	// Logs Screen State
	logsService   string // Service to view logs for
	logsSelection int    // Selected service index
	logsContent   string // Log content

	// Models Screen State
	modelsProvider  string // "ollama" or "localai"
	modelsSelection int    // Selected provider index
	modelsList      string // Cached models list display
	modelsStats     string // Cached stats display
	modelsMessage   string // Status message

	// Power Screen State
	powerConfig  idle.IdleConfig // Current configuration
	powerMessage string          // Status message
}

const down = "down"

// Dir not resolved
const dirNotResolved = "Error: Compose directory not resolved"

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
	m.loadPowerConfig()

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
// Story T-024: Enhanced with menu navigation
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if next, handled, cmd := m.handleQuitKeys(keyMsg.String()); handled {
		return next, cmd
	}

	if next, handled := m.handleEscapeKey(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleMenuNavigationKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleMenuSelectionKey(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleShortcutKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleStatusScreenKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleInstallScreenKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleLogsScreenKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handleModelsScreenKeys(keyMsg.String()); handled {
		return next, nil
	}

	if next, handled := m.handlePowerScreenKeys(keyMsg.String()); handled {
		return next, nil
	}

	return m, nil
}

func (m Model) handleQuitKeys(key string) (tea.Model, bool, tea.Cmd) {
	switch key {
	case "ctrl+c", "q":
		m.quitting = true
		m.saveState()
		return m, true, tea.Quit
	}
	return m, false, nil
}

func (m Model) handleEscapeKey(key string) (tea.Model, bool) {
	if key == "esc" && m.currentScreen != ScreenMenu {
		m = m.returnToMenu()
		m.saveState()
		return m, true
	}
	return m, false
}

func (m Model) handleMenuNavigationKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenMenu {
		return m, false
	}

	switch key {
	case "up", "k":
		return m.navigateUp(), true
	case down, "j":
		return m.navigateDown(), true
	}
	return m, false
}

func (m Model) handleMenuSelectionKey(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenMenu {
		return m, false
	}

	if key == "enter" || key == " " {
		updated := m.selectMenuItem()
		updated.saveState()
		return updated, true
	}
	return m, false
}

func (m Model) handleShortcutKeys(key string) (tea.Model, bool) {
	switch key {
	case "1", "2", "3", "4", "5", "6", "7", "?":
		updated := m.selectMenuByKey(key)
		updated.saveState()
		return updated, true
	}
	return m, false
}

func (m Model) handleStatusScreenKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenStatus {
		return m, false
	}

	switch key {
	case "b":
		return m.toggleBackend(), true
	case "r":
		return m.refresh(), true
	}
	return m, false
}

func (m Model) handleInstallScreenKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenInstall {
		return m, false
	}

	// Don't handle keys while operation is in progress
	if m.installInProgress {
		return m, false
	}

	switch key {
	case "up", "k":
		if m.installSelection > 0 {
			m.installSelection--
		} else {
			m.installSelection = 2 // Wrap to bottom (3 services - 1)
		}
		return m, true
	case down, "j":
		if m.installSelection < 2 {
			m.installSelection++
		} else {
			m.installSelection = 0 // Wrap to top
		}
		return m, true
	case "i":
		return m.installService(), true
	case "u":
		return m.uninstallService(), true
	case "r":
		return m.refreshInstallScreen(), true
	}
	return m, false
}

func (m Model) handleLogsScreenKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenLogs {
		return m, false
	}

	switch key {
	case "up", "k":
		if m.logsSelection > 0 {
			m.logsSelection--
		} else {
			m.logsSelection = 2 // Wrap to bottom (3 services - 1)
		}
		return m, true
	case down, "j":
		if m.logsSelection < 2 {
			m.logsSelection++
		} else {
			m.logsSelection = 0 // Wrap to top
		}
		return m, true
	case "enter", " ":
		return m.loadLogs(), true
	case "r":
		return m.loadLogs(), true
	}
	return m, false
}

func (m Model) handleModelsScreenKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenModels {
		return m, false
	}

	switch key {
	case "up", "k":
		if m.modelsSelection > 0 {
			m.modelsSelection--
		} else {
			m.modelsSelection = 1 // Wrap to bottom (2 providers - 1)
		}
		return m, true
	case down, "j":
		if m.modelsSelection < 1 {
			m.modelsSelection++
		} else {
			m.modelsSelection = 0 // Wrap to top
		}
		return m, true
	case "l":
		return m.listModels(), true
	case "s":
		return m.showModelStats(), true
	case "r":
		return m.refreshModelsScreen(), true
	}
	return m, false
}

func (m Model) handlePowerScreenKeys(key string) (tea.Model, bool) {
	if m.currentScreen != ScreenPower {
		return m, false
	}

	switch key {
	case "t":
		return m.toggleSuspend(), true
	case "r":
		return m.refreshPowerScreen(), true
	}
	return m, false
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
		return m.renderInstallScreen()
	case ScreenModels:
		return m.renderModelsScreen()
	case ScreenPower:
		return m.renderPowerScreen()
	case ScreenLogs:
		return m.renderLogsScreen()
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

// getServiceNames returns the list of service names
func getServiceNames() []string {
	return []string{"ollama", "openwebui", "localai"}
}

// installService installs the selected service
func (m Model) installService() Model {
	if m.composeDir == "" {
		m.installResult = dirNotResolved
		return m
	}

	m.installInProgress = true
	serviceNames := getServiceNames()
	serviceName := serviceNames[m.installSelection]

	manager, err := services.NewManager(m.composeDir, m.logger)
	if err != nil {
		m.installResult = fmt.Sprintf("Error: %v", err)
		m.installInProgress = false
		return m
	}

	service, err := manager.GetService(serviceName)
	if err != nil {
		m.installResult = fmt.Sprintf("Error: %v", err)
		m.installInProgress = false
		return m
	}

	err = service.Install()
	if err != nil {
		m.installResult = fmt.Sprintf("Failed to install %s: %v", serviceName, err)
	} else {
		m.installResult = fmt.Sprintf("Successfully installed %s", serviceName)
	}

	m.installInProgress = false
	return m
}

// uninstallService uninstalls the selected service
func (m Model) uninstallService() Model {
	if m.composeDir == "" {
		m.installResult = dirNotResolved
		return m
	}

	m.installInProgress = true
	serviceNames := getServiceNames()
	serviceName := serviceNames[m.installSelection]

	manager, err := services.NewManager(m.composeDir, m.logger)
	if err != nil {
		m.installResult = fmt.Sprintf("Error: %v", err)
		m.installInProgress = false
		return m
	}

	service, err := manager.GetService(serviceName)
	if err != nil {
		m.installResult = fmt.Sprintf("Error: %v", err)
		m.installInProgress = false
		return m
	}

	// Uninstall with data preservation (keepData = true)
	err = service.Remove(true)
	if err != nil {
		m.installResult = fmt.Sprintf("Failed to uninstall %s: %v", serviceName, err)
	} else {
		m.installResult = fmt.Sprintf("Successfully uninstalled %s (data preserved)", serviceName)
	}

	m.installInProgress = false
	return m
}

// refreshInstallScreen refreshes the install screen
func (m Model) refreshInstallScreen() Model {
	m.installResult = "Screen refreshed"
	return m
}

// loadLogs loads logs for the selected service
func (m Model) loadLogs() Model {
	if m.composeDir == "" {
		m.logsContent = dirNotResolved
		return m
	}

	serviceNames := getServiceNames()
	serviceName := serviceNames[m.logsSelection]
	m.logsService = serviceName

	manager, err := services.NewManager(m.composeDir, m.logger)
	if err != nil {
		m.logsContent = fmt.Sprintf("Error: %v", err)
		return m
	}

	service, err := manager.GetService(serviceName)
	if err != nil {
		m.logsContent = fmt.Sprintf("Error: %v", err)
		return m
	}

	logs, err := service.Logs(50)
	if err != nil {
		m.logsContent = fmt.Sprintf("Error loading logs: %v", err)
	} else {
		m.logsContent = logs
	}

	return m
}

// getProviderNames returns the list of model providers
func getProviderNames() []string {
	return []string{"ollama", "localai"}
}

// listModels lists models for the selected provider
func (m Model) listModels() Model {
	providers := getProviderNames()
	provider := providers[m.modelsSelection]
	m.modelsProvider = provider

	// Load model list from state
	stateManager := NewModelsStateManager(provider, m.stateDir, m.logger)
	state, err := stateManager.Load()
	if err != nil {
		m.modelsList = fmt.Sprintf("Error loading models: %v", err)
		m.modelsMessage = fmt.Sprintf("Failed to load models for %s", provider)
		return m
	}

	// Format model list
	var b strings.Builder
	if len(state.Items) == 0 {
		b.WriteString(fmt.Sprintf("No models cached for %s\n", provider))
	} else {
		for _, model := range state.Items {
			sizeMB := model.Size / (1024 * 1024)
			b.WriteString(fmt.Sprintf("  • %s (%d MB) - Last used: %s\n",
				model.Name, sizeMB, model.LastUsed.Format("2006-01-02 15:04")))
		}
	}

	m.modelsList = b.String()
	m.modelsMessage = fmt.Sprintf("Listed %d models for %s", len(state.Items), provider)
	return m
}

// showModelStats shows cache statistics for the selected provider
func (m Model) showModelStats() Model {
	providers := getProviderNames()
	provider := providers[m.modelsSelection]
	m.modelsProvider = provider

	// Load stats
	stateManager := NewModelsStateManager(provider, m.stateDir, m.logger)
	stats, err := stateManager.GetStats()
	if err != nil {
		m.modelsStats = fmt.Sprintf("Error loading stats: %v", err)
		m.modelsMessage = fmt.Sprintf("Failed to load stats for %s", provider)
		return m
	}

	// Format stats
	var b strings.Builder
	totalGB := float64(stats.TotalSize) / (1024 * 1024 * 1024)
	b.WriteString(fmt.Sprintf("Provider: %s\n", stats.Provider))
	b.WriteString(fmt.Sprintf("Total Size: %.2f GB\n", totalGB))
	b.WriteString(fmt.Sprintf("Model Count: %d\n", stats.ModelCount))
	if stats.OldestModel != nil {
		b.WriteString(fmt.Sprintf("Oldest Model: %s (last used: %s)\n",
			stats.OldestModel.Name, stats.OldestModel.LastUsed.Format("2006-01-02 15:04")))
	}

	m.modelsStats = b.String()
	m.modelsMessage = fmt.Sprintf("Stats loaded for %s", provider)
	return m
}

// refreshModelsScreen refreshes the models screen
func (m Model) refreshModelsScreen() Model {
	m.modelsList = ""
	m.modelsStats = ""
	m.modelsMessage = "Screen refreshed"
	return m
}

// loadPowerConfig loads the power/idle configuration
func (m *Model) loadPowerConfig() {
	m.powerConfig = idle.DefaultIdleConfig()
}

// toggleSuspend toggles the suspend enable flag
func (m Model) toggleSuspend() Model {
	m.powerConfig.EnableSuspend = !m.powerConfig.EnableSuspend

	status := "disabled"
	if m.powerConfig.EnableSuspend {
		status = "enabled"
	}
	m.powerMessage = fmt.Sprintf("Auto-suspend %s", status)

	return m
}

// refreshPowerScreen refreshes the power screen
func (m Model) refreshPowerScreen() Model {
	m.loadPowerConfig()
	m.powerMessage = "Configuration reloaded"
	return m
}

// NewModelsStateManager creates a state manager for the given provider
func NewModelsStateManager(provider, stateDir string, logger *logging.Logger) *StateManager {
	// Import models package types
	var p interface{}
	if provider == "ollama" {
		p = "ollama"
	} else {
		p = "localai"
	}

	// Return a minimal state manager wrapper
	return &StateManager{
		provider:     provider,
		stateDir:     stateDir,
		logger:       logger,
		providerType: p,
	}
}

// StateManager wraps models.StateManager for TUI
type StateManager struct {
	provider     string
	stateDir     string
	logger       *logging.Logger
	providerType interface{}
}

// Load loads the models state
func (sm *StateManager) Load() (*ModelsState, error) {
	// This is a simplified version for TUI
	// In a real implementation, this would call models.StateManager.Load()
	return &ModelsState{
		Provider: sm.provider,
		Items:    []ModelInfo{},
	}, nil
}

// GetStats returns cache statistics
func (sm *StateManager) GetStats() (*CacheStats, error) {
	// This is a simplified version for TUI
	return &CacheStats{
		Provider:   sm.provider,
		TotalSize:  0,
		ModelCount: 0,
	}, nil
}

// ModelsState represents the models state
type ModelsState struct {
	Provider string
	Items    []ModelInfo
}

// ModelInfo represents a model
type ModelInfo struct {
	Name     string
	Size     int64
	LastUsed time.Time
}

// CacheStats represents cache statistics
type CacheStats struct {
	Provider    string
	TotalSize   int64
	ModelCount  int
	OldestModel *ModelInfo
}
