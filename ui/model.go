// Package ui provides the terminal user interface (TUI) for my-pnpm-installer.
//
// This package implements the Bubble Tea Model-View-Update pattern to create
// an interactive terminal interface for package management. It provides:
//   - Package list navigation with keyboard controls
//   - Real-time package information display (versions, status)
//   - Installation progress with live log streaming
//   - Responsive layout with graceful degradation for small terminals
//
// Architecture:
//
//	Model (model.go)
//	    ├── State management (list view, installing view)
//	    ├── Event handling (keyboard, window resize, async messages)
//	    └── Integration with PackageService
//
//	View (view.go)
//	    ├── List view rendering (two-panel layout)
//	    ├── Installation view rendering (log display)
//	    └── Lipgloss styling and layout
//
// Keyboard Controls:
//   - ↑/↓ or j/k: Navigate package list
//   - Enter: Install/update selected package
//   - r: Refresh package information
//   - q or Ctrl+C: Quit application
//
// The UI automatically adapts to terminal size and provides helpful error messages
// when the terminal is too small (minimum 80x24).
package ui

import (
	"fmt"
	"strings"
	"time"

	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/services"
)

// ViewState represents the current state of the UI
type ViewState int

const (
	// StateList shows the package list and details panel
	StateList ViewState = iota
	
	// StateInstalling shows the installation progress and logs
	StateInstalling
)

// filterState groups all filter-related fields.
type filterState struct {
	// filteredPackages contains the currently filtered packages (nil if no filter applied)
	filteredPackages []config.Package

	// input tracks the current filter input string
	input string

	// active indicates if the user is currently typing a filter
	active bool
}

// installState groups all installation-related fields.
type installState struct {
	// logs contains the real-time installation log lines
	logs []string

	// logChan is the channel for receiving installation logs
	logChan <-chan string

	// errChan is the channel for receiving installation errors
	errChan <-chan error

	// lastResult stores the result message of the last installation
	// (success or error) so it can be displayed in the list view
	lastResult string

	// lastSuccess indicates whether the last installation succeeded
	lastSuccess bool

	// forceInstall indicates if the next installation should be forced (bypass version check)
	forceInstall bool

	// scrollOffset tracks the current scroll position in the log view
	scrollOffset int

	// autoScroll indicates whether the log view should automatically scroll to bottom
	autoScroll bool
}

// packageInfoState groups package info loading and caching fields.
type packageInfoState struct {
	// current contains version and status information for the selected package
	current *services.PackageInfo

	// cache stores package information by package name for fast switching
	cache map[string]*services.PackageInfo

	// isLoading indicates if package info is currently being fetched
	isLoading bool

	// loadingName tracks which package info is currently being loaded
	// This prevents race conditions when quickly switching between packages
	loadingName string

	// scrollOffset tracks the scroll position in the package details panel
	scrollOffset int

	// prefetchIndex tracks the next package index to prefetch
	prefetchIndex int
}

// Model represents the Bubble Tea model for the TUI application.
// It manages the application state, package list, and installation logs.
type Model struct {
	// list is the Bubbles list component for displaying packages
	list list.Model

	// packages contains all packages from the configuration
	packages []config.Package

	// selectedPackage is the currently selected package (nil if none selected)
	selectedPackage *config.Package

	// width is the current terminal width
	width int

	// height is the current terminal height
	height int

	// service is the package service for version checking and installation
	service services.PackageService

	// state is the current view state (list or installing)
	state ViewState

	// filter groups all filter-related state
	filter filterState

	// install groups all installation-related state
	install installState

	// pkgInfo groups package info loading and caching state
	pkgInfo packageInfoState
}

// Custom message types for async operations

// initMsg is sent on initialization to trigger loading of first package
type initMsg struct{}

// packageInfoMsg is sent when package information has been fetched
type packageInfoMsg services.PackageInfo

// prefetchNextMsg is sent to trigger prefetching of the next package
type prefetchNextMsg struct{}

// installLogMsg is sent when a new log line is received during installation
type installLogMsg string

// installCompleteMsg is sent when installation completes (with or without error)
type installCompleteMsg struct {
	err error
}

// NewModel creates a new Model instance with the provided packages and service.
// It initializes the list component and sets up the initial state.
//
// Preconditions:
//   - packages is a non-empty slice of valid Package structs
//   - service is a properly initialized PackageService
//
// Postconditions:
//   - Returns a Model ready to be used with Bubble Tea
//   - The first package is selected by default
//   - State is set to StateList
func NewModel(packages []config.Package, service services.PackageService) Model {
	// Create list items from packages
	items := make([]list.Item, len(packages))
	for i, pkg := range packages {
		items[i] = packageItem{pkg: pkg}
	}
	
	// Initialize the list component
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Packages"
	
	// Enable filtering
	l.SetFilteringEnabled(true)
	
	// Set the first package as selected if available
	var selectedPkg *config.Package
	if len(packages) > 0 {
		selectedPkg = &packages[0]
	}
	
	return Model{
		list:            l,
		packages:        packages,
		selectedPackage: selectedPkg,
		width:           0,
		height:          0,
		service:         service,
		state:           StateList,
		filter: filterState{
			filteredPackages: nil,
			input:            "",
			active:           false,
		},
		install: installState{
			logs:         []string{},
			logChan:      nil,
			errChan:      nil,
			lastResult:   "",
			lastSuccess:  false,
			forceInstall: false,
		},
		pkgInfo: packageInfoState{
			current:       nil,
			cache:         make(map[string]*services.PackageInfo),
			isLoading:     false,
			loadingName:   "",
			prefetchIndex: 1, // Start from index 1 (index 0 is loaded in Init)
		},
	}
}

// Init is the Bubble Tea initialization method.
// It returns a command to trigger initial package info loading.
//
// Preconditions:
//   - Model is properly initialized with NewModel
//
// Postconditions:
//   - Returns a command to trigger initialization
func (m Model) Init() tea.Cmd {
	// Send init message to trigger loading in Update method
	return func() tea.Msg {
		return initMsg{}
	}
}

// fetchPackageInfo returns a command that fetches package information asynchronously.
// The result is sent back as a packageInfoMsg.
func (m Model) fetchPackageInfo(pkg config.Package) tea.Cmd {
	return func() tea.Msg {
		info := m.service.GetPackageInfo(context.Background(), pkg)
		return packageInfoMsg(info)
	}
}

// fetchPackageInfoWithCache fetches package info with caching support.
// If forceRefresh is true, it bypasses the cache and fetches fresh data.
// Returns a batch command that sets loading state and fetches the info.
func (m *Model) fetchPackageInfoWithCache(pkg config.Package, forceRefresh bool) tea.Cmd {
	// Set loading state
	m.pkgInfo.isLoading = true
	m.pkgInfo.loadingName = pkg.Name

	// Check cache first (unless force refresh)
	if !forceRefresh {
		if cachedInfo, exists := m.pkgInfo.cache[pkg.Name]; exists {
			// Return cached info immediately
			return func() tea.Msg {
				return packageInfoMsg(*cachedInfo)
			}
		}
	}

	// Fetch fresh data
	return func() tea.Msg {
		info := m.service.GetPackageInfo(context.Background(), pkg)
		return packageInfoMsg(info)
	}
}

// fetchPackageInfoInBackground fetches package info without setting loading state.
// Used for background prefetching to avoid UI flickering.
func (m *Model) fetchPackageInfoInBackground(pkg config.Package) tea.Cmd {
	return func() tea.Msg {
		info := m.service.GetPackageInfo(context.Background(), pkg)
		return packageInfoMsg(info)
	}
}

// triggerPrefetchNext returns a command to trigger the next prefetch after a delay.
// The delay prevents overwhelming the system with too many concurrent requests.
func (m *Model) triggerPrefetchNext() tea.Cmd {
	return tea.Tick(config.PrefetchDelay, func(t time.Time) tea.Msg {
		return prefetchNextMsg{}
	})
}

// packageItem is a wrapper for config.Package to implement list.Item interface
type packageItem struct {
	pkg config.Package
}

// FilterValue implements list.Item interface
// Returns a searchable string that includes both the display name and the actual package name
func (i packageItem) FilterValue() string {
	// Include both the display name and the actual package name for better filtering
	// Example: "ccusage ccusage" or "Qwen Code CLI @qwen-code/qwen-code"
	packageName := extractPackageName(i.pkg.InstallCommand)
	if packageName != i.pkg.Name {
		return i.pkg.Name + " " + packageName
	}
	return i.pkg.Name
}

// Title implements list.DefaultItem interface
func (i packageItem) Title() string {
	return i.pkg.Name
}

// Description implements list.DefaultItem interface
func (i packageItem) Description() string {
	// Extract package name from install command
	// e.g., "npm install -g @scope/package@latest" -> "@scope/package"
	return extractPackageName(i.pkg.InstallCommand)
}

// extractPackageName extracts the package name from an install command
func extractPackageName(installCmd string) string {
	// Common patterns:
	// npm install -g package@latest
	// pnpm install -g @scope/package@latest
	// npm install -g package
	
	parts := strings.Fields(installCmd)
	for i, part := range parts {
		if part == "-g" && i+1 < len(parts) {
			// Next part is the package name
			pkgWithVersion := parts[i+1]
			// Remove @latest or version suffix
			if idx := strings.LastIndex(pkgWithVersion, "@"); idx > 0 {
				return pkgWithVersion[:idx]
			}
			return pkgWithVersion
		}
	}
	
	// Fallback: return the install command
	return installCmd
}

// applyFilter applies the current filter string to the package list
// and updates the list component with filtered items
func (m *Model) applyFilter(filterStr string) {
	if filterStr == "" {
		// No filter, show all packages
		m.filter.filteredPackages = nil
		items := make([]list.Item, len(m.packages))
		for i, pkg := range m.packages {
			items[i] = packageItem{pkg: pkg}
		}
		m.list.SetItems(items)
		return
	}

	// Filter packages based on filter string (case-insensitive substring match)
	filterLower := strings.ToLower(filterStr)
	filtered := []config.Package{}

	for _, pkg := range m.packages {
		pkgName := strings.ToLower(pkg.Name)
		installCmd := strings.ToLower(extractPackageName(pkg.InstallCommand))
		if strings.Contains(pkgName, filterLower) || strings.Contains(installCmd, filterLower) {
			filtered = append(filtered, pkg)
		}
	}

	// Update filtered packages and list items
	m.filter.filteredPackages = filtered
	items := make([]list.Item, len(filtered))
	for i, pkg := range filtered {
		items[i] = packageItem{pkg: pkg}
	}
	m.list.SetItems(items)

	// Select first item if available
	if len(filtered) > 0 {
		m.list.Select(0)
		m.selectedPackage = &filtered[0]
	} else {
		m.selectedPackage = nil
	}
}

// Update is the Bubble Tea update method that handles all events and messages.
// It processes keyboard input, window resize events, and custom async messages.
//
// Message handling flow:
//  1. KeyMsg: User keyboard input (navigation, actions)
//  2. WindowSizeMsg: Terminal resize events
//  3. packageInfoMsg: Async package info fetched from service
//  4. installLogMsg: Real-time installation log lines
//  5. installCompleteMsg: Installation completion (success or error)
//
// State transitions:
//  - StateList → StateInstalling: When user presses Enter to install
//  - StateInstalling → StateList: When installation completes
//
// Preconditions:
//   - msg is a valid Bubble Tea message
//   - Model state is consistent
//
// Postconditions:
//   - Returns updated model reflecting state changes
//   - Returns command for async operations (or nil)
//   - State transitions are valid
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initMsg:
		return m.handleInitMsg()
	case prefetchNextMsg:
		return m.handlePrefetchNextMsg()
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case packageInfoMsg:
		return m.handlePackageInfoMsg(msg)
	case installLogMsg:
		return m.handleInstallLogMsg(msg)
	case installCompleteMsg:
		return m.handleInstallCompleteMsg(msg)
	}
	return m, nil
}

// handleInitMsg handles the initialization message to load first package info.
func (m Model) handleInitMsg() (tea.Model, tea.Cmd) {
	if len(m.packages) > 0 && m.selectedPackage != nil {
		return m, tea.Batch(
			m.fetchPackageInfoWithCache(*m.selectedPackage, false),
			m.triggerPrefetchNext(),
		)
	}
	return m, nil
}

// handlePrefetchNextMsg handles background prefetching of package info.
func (m Model) handlePrefetchNextMsg() (tea.Model, tea.Cmd) {
	// Check if prefetch is complete
	if m.pkgInfo.prefetchIndex >= len(m.packages) {
		return m, nil
	}

	pkg := m.packages[m.pkgInfo.prefetchIndex]

	// Skip if already cached
	if _, exists := m.pkgInfo.cache[pkg.Name]; exists {
		m.pkgInfo.prefetchIndex++
		return m, m.triggerPrefetchNext()
	}

	// Fetch this package info in background
	m.pkgInfo.prefetchIndex++
	return m, tea.Batch(
		m.fetchPackageInfoInBackground(pkg),
		m.triggerPrefetchNext(),
	)
}

// handleKeyMsg dispatches keyboard input based on current state.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == StateList {
		if m.filter.active {
			return m.handleFilterKeyMsg(msg)
		}
		return m.handleListKeyMsg(msg)
	}
	// In installing view, only handle quit
	if msg.String() == "q" || msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	return m, nil
}

// handleFilterKeyMsg handles keyboard input while filter mode is active.
func (m Model) handleFilterKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filter.active = false
		m.filter.input = ""
		m.applyFilter("")
		return m, nil
	case "enter":
		m.filter.active = false
		m.applyFilter(m.filter.input)
		if m.selectedPackage != nil {
			return m, m.fetchPackageInfoWithCache(*m.selectedPackage, false)
		}
		return m, nil
	case "backspace":
		if len(m.filter.input) > 0 {
			m.filter.input = m.filter.input[:len(m.filter.input)-1]
		}
		return m, nil
	default:
		if len(msg.Runes) > 0 {
			m.filter.input += string(msg.Runes)
		}
		return m, nil
	}
}

// handleNavigation handles up/down cursor movement and updates selected package.
func (m *Model) handleNavigation(direction string) tea.Cmd {
	if direction == "up" {
		m.list.CursorUp()
	} else {
		m.list.CursorDown()
	}

	if selectedItem := m.list.SelectedItem(); selectedItem != nil {
		if pkgItem, ok := selectedItem.(packageItem); ok {
			m.selectedPackage = &pkgItem.pkg
			m.pkgInfo.scrollOffset = 0

			// Check if info is cached
			if cachedInfo, exists := m.pkgInfo.cache[m.selectedPackage.Name]; exists {
				m.pkgInfo.current = cachedInfo
				m.pkgInfo.isLoading = false
				m.pkgInfo.loadingName = ""
				return nil
			}

			// Not cached, fetch with loading state
			return m.fetchPackageInfoWithCache(*m.selectedPackage, false)
		}
	}
	return nil
}

// handleListKeyMsg handles keyboard input in the list view (non-filter mode).
func (m Model) handleListKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.filter.active = true
		m.filter.input = ""
		return m, nil

	case "esc":
		if m.filter.filteredPackages != nil {
			m.applyFilter("")
			if m.selectedPackage != nil {
				return m, m.fetchPackageInfoWithCache(*m.selectedPackage, false)
			}
		}
		return m, nil

	case "up", "k":
		if m.state == StateInstalling {
			m.install.autoScroll = false
			if m.install.scrollOffset > 0 {
				m.install.scrollOffset--
			}
			return m, nil
		}
		cmd := m.handleNavigation("up")
		return m, cmd

	case "down", "j":
		if m.state == StateInstalling {
			if m.install.scrollOffset < len(m.install.logs)-1 {
				m.install.scrollOffset++
			}
			// If we reached the bottom, re-enable autoscroll
			if m.install.scrollOffset >= len(m.install.logs)-1 {
				m.install.autoScroll = true
			}
			return m, nil
		}
		cmd := m.handleNavigation("down")
		return m, cmd

	case "enter":
		if m.selectedPackage != nil && m.pkgInfo.current != nil {
			if m.pkgInfo.current.Status == services.StatusInstalled && !m.install.forceInstall {
				return m, nil
			}
			m.install.forceInstall = false
			m.state = StateInstalling
			m.install.logs = []string{}
			m.install.lastResult = ""
			m.install.autoScroll = true
			m.install.scrollOffset = 0
			logChan, errChan := m.service.InstallPackage(context.Background(), *m.selectedPackage)
			m.install.logChan = logChan
			m.install.errChan = errChan
			return m, waitForInstallActivity(m.install.logChan, m.install.errChan)
		}
		return m, nil

	case "f":
		m.install.forceInstall = true
		return m, nil

	case "r":
		if m.selectedPackage != nil {
			m.pkgInfo.isLoading = true
			m.pkgInfo.loadingName = m.selectedPackage.Name
			return m, m.fetchPackageInfoWithCache(*m.selectedPackage, true)
		}
		return m, nil

	case "K": // Shift+K to scroll details panel up
		if m.state == StateList && m.pkgInfo.scrollOffset > 0 {
			m.pkgInfo.scrollOffset--
		}
		return m, nil

	case "J": // Shift+J to scroll details panel down
		if m.state == StateList {
			m.pkgInfo.scrollOffset++
		}
		return m, nil

	case "pgup", "pageup":
		if m.state == StateList && m.pkgInfo.scrollOffset > 0 {
			m.pkgInfo.scrollOffset -= 5
			if m.pkgInfo.scrollOffset < 0 {
				m.pkgInfo.scrollOffset = 0
			}
			return m, nil
		}
	case "pgdown", "pagedown":
		if m.state == StateList {
			m.pkgInfo.scrollOffset += 5
			return m, nil
		}

	default:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleWindowSizeMsg handles terminal resize events.
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.list.SetSize(msg.Width/2, msg.Height-4)
	return m, nil
}

// handlePackageInfoMsg handles async package info responses.
func (m Model) handlePackageInfoMsg(msg packageInfoMsg) (tea.Model, tea.Cmd) {
	info := services.PackageInfo(msg)

	if m.selectedPackage != nil && info.Package.Name == m.selectedPackage.Name {
		m.pkgInfo.current = &info
		m.pkgInfo.cache[info.Package.Name] = &info
		if m.pkgInfo.loadingName == info.Package.Name {
			m.pkgInfo.isLoading = false
			m.pkgInfo.loadingName = ""
		}
	} else {
		// Stale response - just cache it
		m.pkgInfo.cache[info.Package.Name] = &info
	}
	return m, nil
}

// handleInstallLogMsg handles real-time installation log lines.
func (m Model) handleInstallLogMsg(msg installLogMsg) (tea.Model, tea.Cmd) {
	m.install.logs = append(m.install.logs, string(msg))
	if len(m.install.logs) > config.MaxLogLines {
		m.install.logs = m.install.logs[len(m.install.logs)-config.MaxLogLines:]
	}
	if m.install.autoScroll {
		m.install.scrollOffset = len(m.install.logs)
	}
	return m, waitForInstallActivity(m.install.logChan, m.install.errChan)
}

// handleInstallCompleteMsg handles installation completion (success or error).
func (m Model) handleInstallCompleteMsg(msg installCompleteMsg) (tea.Model, tea.Cmd) {
	m.state = StateList
	m.install.logChan = nil
	m.install.errChan = nil
	if msg.err != nil {
		m.install.lastResult = fmt.Sprintf("✗ %v", msg.err)
		m.install.lastSuccess = false
	} else {
		pkgName := ""
		if m.selectedPackage != nil {
			pkgName = m.selectedPackage.Name
		}
		m.install.lastResult = fmt.Sprintf("✓ %s installed successfully", pkgName)
		m.install.lastSuccess = true
	}
	if m.selectedPackage != nil {
		delete(m.pkgInfo.cache, m.selectedPackage.Name)
		return m, m.fetchPackageInfoWithCache(*m.selectedPackage, true)
	}
	return m, nil
}

// waitForInstallActivity creates a command that waits for the next log message or completion.
// This allows us to stream logs back to the UI in real-time using Bubble Tea's message system.
//
// The function reads only from logChan. When logChan closes, it reads the final
// error from errChan. This sequential approach avoids a race condition where a
// select on both channels could pick errChan before all logs are consumed,
// causing the last few log lines to be lost.
func waitForInstallActivity(logChan <-chan string, errChan <-chan error) tea.Cmd {
	return func() tea.Msg {
		log, ok := <-logChan
		if !ok {
			// Log channel closed, all logs have been sent
			// Now wait for the final error status from errChan
			err := <-errChan
			return installCompleteMsg{err: err}
		}
		// Send log message to UI and continue waiting for more
		return installLogMsg(log)
	}
}

// View is the Bubble Tea view method that renders the UI.
// It switches between list view and installation view based on the current state.
//
// Preconditions:
//   - Model state is consistent
//   - width and height are positive integers
//
// Postconditions:
//   - Returns formatted string for terminal display
//   - No state mutations
func (m Model) View() string {
	if m.state == StateInstalling {
		return m.renderInstallView()
	}
	return m.renderListView()
}
