package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/services"
)

// mockPackageService is a simple mock for backward compatibility
type mockPackageService struct{}

func (m *mockPackageService) GetPackageInfo(ctx context.Context, pkg config.Package) services.PackageInfo {
	return services.PackageInfo{
		Package:          pkg,
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
}

func (m *mockPackageService) InstallPackage(ctx context.Context, pkg config.Package) (<-chan string, <-chan error) {
	logChan := make(chan string, 1)
	errChan := make(chan error, 1)
	close(logChan)
	close(errChan)
	return logChan, errChan
}

func (m *mockPackageService) CheckToolAvailability(ctx context.Context, tool string) bool {
	return true
}

// mockPackageServiceWithFuncs is an extended mock with customizable functions
type mockPackageServiceWithFuncs struct {
	getPackageInfoFunc      func(pkg config.Package) services.PackageInfo
	installPackageFunc      func(pkg config.Package) (<-chan string, <-chan error)
	checkToolAvailabilityFunc func(tool string) bool
}

func (m *mockPackageServiceWithFuncs) GetPackageInfo(ctx context.Context, pkg config.Package) services.PackageInfo {
	if m.getPackageInfoFunc != nil {
		return m.getPackageInfoFunc(pkg)
	}
	return services.PackageInfo{
		Package:          pkg,
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
}

func (m *mockPackageServiceWithFuncs) InstallPackage(ctx context.Context, pkg config.Package) (<-chan string, <-chan error) {
	if m.installPackageFunc != nil {
		return m.installPackageFunc(pkg)
	}
	logChan := make(chan string, 1)
	errChan := make(chan error, 1)
	close(logChan)
	close(errChan)
	return logChan, errChan
}

func (m *mockPackageServiceWithFuncs) CheckToolAvailability(ctx context.Context, tool string) bool {
	if m.checkToolAvailabilityFunc != nil {
		return m.checkToolAvailabilityFunc(tool)
	}
	return true
}

// createTestModel creates a model with test packages and mock service
func createTestModel() Model {
	packages := []config.Package{
		{
			Name:                "test-pkg-1",
			InstallCommand:      "npm install -g test-pkg-1",
			VersionCheckCommand: "npm view test-pkg-1 version",
			LocalVersionCommand: "test-pkg-1 --version",
		},
		{
			Name:                "test-pkg-2",
			InstallCommand:      "npm install -g test-pkg-2",
			VersionCheckCommand: "npm view test-pkg-2 version",
			LocalVersionCommand: "test-pkg-2 --version",
		},
	}
	
	service := &mockPackageService{}
	return NewModel(packages, service)
}

// TestNewModel tests the NewModel constructor
func TestNewModel(t *testing.T) {
	packages := []config.Package{
		{
			Name:        "test-pkg",
		},
	}
	
	service := &mockPackageService{}
	model := NewModel(packages, service)
	
	// Verify initial state
	if model.state != StateList {
		t.Errorf("Expected initial state to be StateList, got %v", model.state)
	}
	
	if len(model.packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(model.packages))
	}
	
	if model.selectedPackage == nil {
		t.Fatal("Expected selectedPackage to be set")
	}
	
	if model.selectedPackage.Name != "test-pkg" {
		t.Errorf("Expected selectedPackage name to be 'test-pkg', got '%s'", model.selectedPackage.Name)
	}
	
	if len(model.install.logs) != 0 {
		t.Errorf("Expected empty install.logs, got %d items", len(model.install.logs))
	}
}

// TestNewModel_EmptyPackages tests NewModel with empty packages list
func TestNewModel_EmptyPackages(t *testing.T) {
	packages := []config.Package{}
	service := &mockPackageService{}
	model := NewModel(packages, service)
	
	if model.selectedPackage != nil {
		t.Error("Expected selectedPackage to be nil for empty packages list")
	}
}

// TestInit tests the Init method
func TestInit(t *testing.T) {
	model := createTestModel()
	
	cmd := model.Init()
	
	if cmd == nil {
		t.Fatal("Expected Init to return a command")
	}
	
	// Execute the command to verify it returns initMsg
	msg := cmd()
	if _, ok := msg.(initMsg); !ok {
		t.Errorf("Expected initMsg, got %T", msg)
	}
}

// TestInit_EmptyPackages tests Init with no packages
func TestInit_EmptyPackages(t *testing.T) {
	model := NewModel([]config.Package{}, &mockPackageService{})
	
	cmd := model.Init()
	
	// Init always returns a command now (initMsg)
	if cmd == nil {
		t.Error("Expected Init to return a command")
	}
}

// TestUpdate_QuitKey tests quitting with 'q' key
func TestUpdate_QuitKey(t *testing.T) {
	model := createTestModel()
	
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)
	
	if cmd == nil {
		t.Fatal("Expected quit command")
	}
	
	// Verify it's a quit command by checking if it returns tea.Quit
	if cmd() != tea.Quit() {
		t.Error("Expected tea.Quit command")
	}
}

// TestUpdate_CtrlC tests quitting with Ctrl+C
func TestUpdate_CtrlC(t *testing.T) {
	model := createTestModel()
	
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := model.Update(msg)
	
	if cmd == nil {
		t.Fatal("Expected quit command")
	}
	
	if cmd() != tea.Quit() {
		t.Error("Expected tea.Quit command")
	}
}

// TestUpdate_UpKey tests moving selection up
func TestUpdate_UpKey(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Move to second item first
	model.list.CursorDown()
	
	// Now move up
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	if m.list.Index() != 0 {
		t.Errorf("Expected cursor at index 0, got %d", m.list.Index())
	}
	
	if cmd == nil {
		t.Fatal("Expected command to fetch package info")
	}
	
	// Verify command returns packageInfoMsg
	result := cmd()
	if _, ok := result.(packageInfoMsg); !ok {
		t.Errorf("Expected packageInfoMsg, got %T", result)
	}
}

// TestUpdate_DownKey tests moving selection down
func TestUpdate_DownKey(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	if m.list.Index() != 1 {
		t.Errorf("Expected cursor at index 1, got %d", m.list.Index())
	}
	
	if cmd == nil {
		t.Fatal("Expected command to fetch package info")
	}
}

// TestUpdate_VimKeys tests vim-style navigation (j/k)
func TestUpdate_VimKeys(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Test 'j' (down)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	
	if m.list.Index() != 1 {
		t.Errorf("Expected cursor at index 1 after 'j', got %d", m.list.Index())
	}
	
	// Test 'k' (up)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)
	
	if m.list.Index() != 0 {
		t.Errorf("Expected cursor at index 0 after 'k', got %d", m.list.Index())
	}
}

// TestUpdate_EnterKey tests installing a package
func TestUpdate_EnterKey(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	
	// Set package info with update available status to trigger installation
	model.pkgInfo.current = &services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}
	
	// Mock the install function to return controlled channels
	logChan := make(chan string, 1)
	errChan := make(chan error, 1)
	
	model.service = &mockPackageServiceWithFuncs{
		installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
			return logChan, errChan
		},
	}
	
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// Verify state transition
	if m.state != StateInstalling {
		t.Errorf("Expected state to be StateInstalling, got %v", m.state)
	}
	
	// Verify logs were cleared
	if len(m.install.logs) != 0 {
		t.Errorf("Expected empty install.logs, got %d items", len(m.install.logs))
	}
	
	// Verify command was returned
	if cmd == nil {
		t.Fatal("Expected command to wait for install activity")
	}
	
	// Clean up channels
	close(logChan)
	close(errChan)
}

// TestUpdate_EnterKey_NoPackageSelected tests Enter with no package selected
func TestUpdate_EnterKey_NoPackageSelected(t *testing.T) {
	model := createTestModel()
	model.selectedPackage = nil
	
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// State should not change
	if m.state != StateList {
		t.Errorf("Expected state to remain StateList, got %v", m.state)
	}
	
	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when no package is selected")
	}
}

// TestUpdate_EnterKey_WhileInstalling tests Enter key is ignored during installation
func TestUpdate_EnterKey_WhileInstalling(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// State should remain StateInstalling
	if m.state != StateInstalling {
		t.Errorf("Expected state to remain StateInstalling, got %v", m.state)
	}
	
	// No command should be returned
	if cmd != nil {
		t.Error("Expected no command when already installing")
	}
}

// TestUpdate_RefreshKey tests refreshing package info
func TestUpdate_RefreshKey(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := model.Update(msg)
	
	if cmd == nil {
		t.Fatal("Expected command to fetch package info")
	}
	
	// Verify command returns packageInfoMsg
	result := cmd()
	if _, ok := result.(packageInfoMsg); !ok {
		t.Errorf("Expected packageInfoMsg, got %T", result)
	}
}

// TestUpdate_RefreshKey_NoPackageSelected tests refresh with no package selected
func TestUpdate_RefreshKey_NoPackageSelected(t *testing.T) {
	model := createTestModel()
	model.selectedPackage = nil
	
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := model.Update(msg)
	
	if cmd != nil {
		t.Error("Expected no command when no package is selected")
	}
}

// TestUpdate_WindowSizeMsg tests terminal resize handling
func TestUpdate_WindowSizeMsg(t *testing.T) {
	model := createTestModel()
	
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := model.Update(msg)
	
	m := updatedModel.(Model)
	
	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
	
	// Verify list was resized (left panel takes half the width)
	expectedWidth := 120 / 2
	expectedHeight := 40 - 4
	
	if m.list.Width() != expectedWidth {
		t.Errorf("Expected list width %d, got %d", expectedWidth, m.list.Width())
	}
	
	if m.list.Height() != expectedHeight {
		t.Errorf("Expected list height %d, got %d", expectedHeight, m.list.Height())
	}
}

// TestUpdate_PackageInfoMsg tests receiving package info
func TestUpdate_PackageInfoMsg(t *testing.T) {
	model := createTestModel()
	
	info := services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}
	
	msg := packageInfoMsg(info)
	updatedModel, _ := model.Update(msg)
	
	m := updatedModel.(Model)
	
	if m.pkgInfo.current == nil {
		t.Fatal("Expected pkgInfo.current to be set")
	}

	if m.pkgInfo.current.LatestVersion != "2.0.0" {
		t.Errorf("Expected latest version '2.0.0', got '%s'", m.pkgInfo.current.LatestVersion)
	}

	if m.pkgInfo.current.Status != services.StatusUpdateAvailable {
		t.Errorf("Expected status StatusUpdateAvailable, got %v", m.pkgInfo.current.Status)
	}
}

// TestUpdate_InstallLogMsg tests receiving installation log messages
func TestUpdate_InstallLogMsg(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	
	// Set up channels for the waitForInstallActivity command
	logChan := make(chan string, 2)
	errChan := make(chan error, 1)
	model.install.logChan = logChan
	model.install.errChan = errChan
	
	// Send first log message
	msg := installLogMsg("Installing package...")
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	if len(m.install.logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(m.install.logs))
	}

	if m.install.logs[0] != "Installing package..." {
		t.Errorf("Expected log 'Installing package...', got '%s'", m.install.logs[0])
	}
	
	// Verify command was returned to continue waiting
	if cmd == nil {
		t.Fatal("Expected command to continue waiting for install activity")
	}
	
	// Send second log message
	msg = installLogMsg("Download complete")
	updatedModel, _ = m.Update(msg)
	
	m = updatedModel.(Model)
	
	if len(m.install.logs) != 2 {
		t.Fatalf("Expected 2 log entries, got %d", len(m.install.logs))
	}

	if m.install.logs[1] != "Download complete" {
		t.Errorf("Expected log 'Download complete', got '%s'", m.install.logs[1])
	}
	
	// Clean up
	close(logChan)
	close(errChan)
}

// TestUpdate_InstallLogMsg_BufferLimit tests log buffer limiting
func TestUpdate_InstallLogMsg_BufferLimit(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	
	logChan := make(chan string, 1)
	errChan := make(chan error, 1)
	model.install.logChan = logChan
	model.install.errChan = errChan

	// Add 1005 log entries (exceeds 1000 limit)
	for i := 0; i < 1005; i++ {
		msg := installLogMsg("Log entry")
		updatedModel, _ := model.Update(msg)
		model = updatedModel.(Model)
	}

	// Verify buffer was limited to 1000
	if len(model.install.logs) != 1000 {
		t.Errorf("Expected 1000 log entries (buffer limit), got %d", len(model.install.logs))
	}
	
	// Clean up
	close(logChan)
	close(errChan)
}

// TestUpdate_InstallCompleteMsg_Success tests successful installation completion
func TestUpdate_InstallCompleteMsg_Success(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	model.install.logs = []string{"Installing...", "Complete"}
	
	msg := installCompleteMsg{err: nil}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// Verify state transition back to list
	if m.state != StateList {
		t.Errorf("Expected state to be StateList, got %v", m.state)
	}
	
	// Verify channels were cleared
	if m.install.logChan != nil {
		t.Error("Expected install.logChan to be nil")
	}

	if m.install.errChan != nil {
		t.Error("Expected install.errChan to be nil")
	}
	
	// Verify command to refresh package info
	if cmd == nil {
		t.Fatal("Expected command to refresh package info")
	}
	
	// Verify no error message was added to logs
	lastLog := m.install.logs[len(m.install.logs)-1]
	if lastLog == "Installation failed!" {
		t.Error("Expected no error message for successful installation")
	}
}

// TestUpdate_InstallCompleteMsg_Error tests failed installation
func TestUpdate_InstallCompleteMsg_Error(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	model.install.logs = []string{"Installing..."}
	
	msg := installCompleteMsg{err: &mockError{msg: "installation failed"}}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// Verify state transition back to list
	if m.state != StateList {
		t.Errorf("Expected state to be StateList, got %v", m.state)
	}
	
	// Verify error result is stored for display in list view
	if m.install.lastResult == "" {
		t.Error("Expected install.lastResult to be set")
	}
	if m.install.lastSuccess {
		t.Error("Expected install.lastSuccess to be false")
	}
	if !strings.Contains(m.install.lastResult, "installation failed") {
		t.Errorf("Expected install.lastResult to contain error, got '%s'", m.install.lastResult)
	}
	
	// Verify command to refresh package info
	if cmd != nil {
		// Command should still be returned to refresh
		result := cmd()
		if _, ok := result.(packageInfoMsg); !ok {
			t.Errorf("Expected packageInfoMsg, got %T", result)
		}
	}
}

// TestUpdate_InstallCompleteMsg_NoPackageSelected tests completion with no package
func TestUpdate_InstallCompleteMsg_NoPackageSelected(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	model.selectedPackage = nil
	
	msg := installCompleteMsg{err: nil}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// State should transition back to list
	if m.state != StateList {
		t.Errorf("Expected state to be StateList, got %v", m.state)
	}
	
	// No command should be returned (no package to refresh)
	if cmd != nil {
		t.Error("Expected no command when no package is selected")
	}
}

// TestStateTransitions tests all valid state transitions
func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name          string
		initialState  ViewState
		msgType       string // "enter", "complete", "down", "log"
		expectedState ViewState
	}{
		{
			name:          "List to Installing on Enter",
			initialState:  StateList,
			msgType:       "enter",
			expectedState: StateInstalling,
		},
		{
			name:          "Installing to List on completion",
			initialState:  StateInstalling,
			msgType:       "complete",
			expectedState: StateList,
		},
		{
			name:          "List remains List on navigation",
			initialState:  StateList,
			msgType:       "down",
			expectedState: StateList,
		},
		{
			name:          "Installing remains Installing on log",
			initialState:  StateInstalling,
			msgType:       "log",
			expectedState: StateInstalling,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := createTestModel()
			model.state = tt.initialState
			
			// Set up channels for installing state
			if tt.initialState == StateInstalling {
				logChan := make(chan string, 1)
				errChan := make(chan error, 1)
				model.install.logChan = logChan
				model.install.errChan = errChan
				defer close(logChan)
				defer close(errChan)
			}
			
			// Create appropriate message based on type
			var msg tea.Msg
			switch tt.msgType {
			case "enter":
				msg = tea.KeyMsg{Type: tea.KeyEnter}
				// Set package info with update available to trigger installation
				model.pkgInfo.current = &services.PackageInfo{
					Package:          model.packages[0],
					LatestVersion:    "2.0.0",
					InstalledVersion: "1.0.0",
					Status:           services.StatusUpdateAvailable,
				}
				// Mock install function for Enter key
				logChan := make(chan string, 1)
				errChan := make(chan error, 1)
				model.service = &mockPackageServiceWithFuncs{
					installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
						return logChan, errChan
					},
				}
				defer close(logChan)
				defer close(errChan)
			case "complete":
				msg = installCompleteMsg{err: nil}
			case "down":
				msg = tea.KeyMsg{Type: tea.KeyDown}
			case "log":
				msg = installLogMsg("test log")
			}
			
			updatedModel, _ := model.Update(msg)
			m := updatedModel.(Model)
			
			if m.state != tt.expectedState {
				t.Errorf("Expected state %v, got %v", tt.expectedState, m.state)
			}
		})
	}
}

// TestView tests the View method switches between views
func TestView(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	
	// Test list view
	model.state = StateList
	view := model.View()
	
	if view == "" {
		t.Error("Expected non-empty view for StateList")
	}
	
	// Test installing view
	model.state = StateInstalling
	model.install.logs = []string{"Installing..."}
	view = model.View()
	
	if view == "" {
		t.Error("Expected non-empty view for StateInstalling")
	}
}

// TestPackageItem tests the packageItem list.Item implementation
func TestPackageItem(t *testing.T) {
	pkg := config.Package{
		Name:           "Test Package",
		InstallCommand: "npm install -g test-pkg",
	}
	
	item := packageItem{pkg: pkg}
	
	// FilterValue should include both display name and package name for better filtering
	expectedFilter := "Test Package test-pkg"
	if item.FilterValue() != expectedFilter {
		t.Errorf("Expected FilterValue '%s', got '%s'", expectedFilter, item.FilterValue())
	}
	
	if item.Title() != "Test Package" {
		t.Errorf("Expected Title 'Test Package', got '%s'", item.Title())
	}
	
	if item.Description() != "test-pkg" {
		t.Errorf("Expected Description 'test-pkg', got '%s'", item.Description())
	}
}


// TestWaitForInstallActivity tests the waitForInstallActivity command
func TestWaitForInstallActivity(t *testing.T) {
	// Test receiving log message
	t.Run("Receive log message", func(t *testing.T) {
		logChan := make(chan string, 1)
		errChan := make(chan error, 1)
		
		logChan <- "Test log"
		
		cmd := waitForInstallActivity(logChan, errChan)
		msg := cmd()
		
		if logMsg, ok := msg.(installLogMsg); !ok {
			t.Errorf("Expected installLogMsg, got %T", msg)
		} else if string(logMsg) != "Test log" {
			t.Errorf("Expected log 'Test log', got '%s'", string(logMsg))
		}
		
		close(logChan)
		close(errChan)
	})
	
	// Test log channel closed (completion)
	t.Run("Log channel closed", func(t *testing.T) {
		logChan := make(chan string, 1)
		errChan := make(chan error, 1)
		
		close(logChan)
		errChan <- nil
		
		cmd := waitForInstallActivity(logChan, errChan)
		msg := cmd()
		
		if completeMsg, ok := msg.(installCompleteMsg); !ok {
			t.Errorf("Expected installCompleteMsg, got %T", msg)
		} else if completeMsg.err != nil {
			t.Errorf("Expected nil error, got %v", completeMsg.err)
		}
		
		close(errChan)
	})
	
	// Test receiving error (logChan must close first, then errChan is read)
	t.Run("Receive error", func(t *testing.T) {
		logChan := make(chan string, 1)
		errChan := make(chan error, 1)

		testErr := &mockError{msg: "test error"}
		close(logChan)
		errChan <- testErr

		cmd := waitForInstallActivity(logChan, errChan)
		msg := cmd()
		
		if completeMsg, ok := msg.(installCompleteMsg); !ok {
			t.Errorf("Expected installCompleteMsg, got %T", msg)
		} else if completeMsg.err == nil {
			t.Error("Expected error, got nil")
		} else if completeMsg.err.Error() != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", completeMsg.err.Error())
		}

		close(errChan)
	})
}

// TestNavigationBoundaries tests navigation at list boundaries
func TestNavigationBoundaries(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Try to move up from first item (should stay at 0)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	
	if m.list.Index() != 0 {
		t.Errorf("Expected cursor to stay at index 0, got %d", m.list.Index())
	}
	
	// Move to last item
	m.list.Select(len(m.packages) - 1)
	
	// Try to move down from last item (should stay at last)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)
	
	expectedIndex := len(m.packages) - 1
	if m.list.Index() != expectedIndex {
		t.Errorf("Expected cursor to stay at index %d, got %d", expectedIndex, m.list.Index())
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// TestUpdate_EnterKey_SkipIfAlreadyLatest tests that Enter key skips installation if already at latest version
func TestUpdate_EnterKey_SkipIfAlreadyLatest(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	
	// Set package info with installed status (already at latest)
	model.pkgInfo.current = &services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)

	m := updatedModel.(Model)

	// Verify state did NOT transition to installing
	if m.state != StateList {
		t.Errorf("Expected state to remain StateList, got %v", m.state)
	}

	// Verify no command was returned (no installation triggered)
	if cmd != nil {
		t.Error("Expected no command when package is already at latest version")
	}
}

// TestUpdate_ForceInstall tests that 'f' key sets force install flag
func TestUpdate_ForceInstall(t *testing.T) {
	model := createTestModel()
	model.state = StateList

	// Press 'f' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updatedModel, _ := model.Update(msg)

	m := updatedModel.(Model)

	// Verify force install flag is set
	if !m.install.forceInstall {
		t.Error("Expected install.forceInstall to be true after pressing 'f'")
	}
}

// TestUpdate_ForceInstall_WithEnter tests that force install bypasses version check
func TestUpdate_ForceInstall_WithEnter(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.install.forceInstall = true // Simulate 'f' was pressed

	// Set package info with installed status (already at latest)
	model.pkgInfo.current = &services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	
	// Mock the install function
	logChan := make(chan string, 1)
	errChan := make(chan error, 1)
	model.service = &mockPackageServiceWithFuncs{
		installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
			return logChan, errChan
		},
	}
	
	// Press Enter
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	
	m := updatedModel.(Model)
	
	// Verify state transitioned to installing (force install worked)
	if m.state != StateInstalling {
		t.Errorf("Expected state to be StateInstalling with force install, got %v", m.state)
	}
	
	// Verify command was returned
	if cmd == nil {
		t.Error("Expected command for force install")
	}
	
	// Verify force install flag was reset
	if m.install.forceInstall {
		t.Error("Expected install.forceInstall to be reset after use")
	}
	
	// Clean up
	close(logChan)
	close(errChan)
}

// TestUpdate_EnterKey_FilteringActive tests that Enter applies filter when filtering is active
func TestUpdate_EnterKey_FilteringActive(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	
	// Simulate filtering mode by starting a filter
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)
	
	// Now press Enter while filtering
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel2, _ := m.Update(enterMsg)
	m2 := updatedModel2.(Model)
	
	// Verify state did NOT transition to installing (Enter was handled by list filter)
	if m2.state != StateList {
		t.Errorf("Expected state to remain StateList during filtering, got %v", m2.state)
	}
}

// TestPrefetch tests background prefetching of package info
func TestPrefetch(t *testing.T) {
	model := createTestModel()

	// Verify initial state
	if model.pkgInfo.prefetchIndex != 1 {
		t.Errorf("Expected pkgInfo.prefetchIndex to be 1, got %d", model.pkgInfo.prefetchIndex)
	}

	// Trigger init which should start prefetching
	cmd := model.Init()
	if cmd == nil {
		t.Fatal("Expected Init to return a command")
	}

	// Execute init command
	msg := cmd()
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Verify prefetching was started (index should be set)
	if m.pkgInfo.prefetchIndex < 1 {
		t.Errorf("Expected pkgInfo.prefetchIndex to be >= 1 after init, got %d", m.pkgInfo.prefetchIndex)
	}
}

// TestPrefetch_SkipsCachedPackages tests that prefetch skips already cached packages
func TestPrefetch_SkipsCachedPackages(t *testing.T) {
	model := createTestModel()

	// Pre-populate cache for second package
	model.pkgInfo.cache[model.packages[1].Name] = &services.PackageInfo{
		Package:          model.packages[1],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}

	model.pkgInfo.prefetchIndex = 1

	// Trigger prefetch
	msg := prefetchNextMsg{}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Verify index was incremented (skipped cached package)
	if m.pkgInfo.prefetchIndex != 2 {
		t.Errorf("Expected pkgInfo.prefetchIndex to be 2 after skipping cached package, got %d", m.pkgInfo.prefetchIndex)
	}
}

// TestPrefetch_StopsAtEnd tests that prefetch stops when all packages are processed
func TestPrefetch_StopsAtEnd(t *testing.T) {
	model := createTestModel()

	// Set prefetch to last package
	model.pkgInfo.prefetchIndex = len(model.packages)

	// Trigger prefetch
	msg := prefetchNextMsg{}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)

	// Verify prefetching stopped (index should not increment past end)
	if m.pkgInfo.prefetchIndex != len(model.packages) {
		t.Errorf("Expected pkgInfo.prefetchIndex to remain at %d, got %d", len(model.packages), m.pkgInfo.prefetchIndex)
	}
}

// TestFilter_CanTypeR tests that 'r' can be typed in filter mode
func TestFilter_CanTypeR(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Start filtering with '/'
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)
	
	// Verify filtering is active
	if !m.filter.active {
		t.Error("Expected filter.active to be true")
	}

	// Type 'r' - should be added to filter input, not refresh
	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel2, _ := m.Update(rMsg)
	m2 := updatedModel2.(Model)

	// Verify still in filtering mode (not refreshed)
	if !m2.filter.active {
		t.Error("Expected to remain in filtering state after typing 'r'")
	}

	// Verify 'r' was added to filter input
	if m2.filter.input != "r" {
		t.Errorf("Expected filter.input to be 'r', got '%s'", m2.filter.input)
	}

	// Verify loading state was NOT triggered (no refresh)
	if m2.pkgInfo.isLoading {
		t.Error("Expected pkgInfo.isLoading to be false (no refresh should happen)")
	}
}

// TestFilter_CanTypeF tests that 'f' can be typed in filter mode
func TestFilter_CanTypeF(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Start filtering
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)
	
	// Type 'f' - should be handled by list, not set force install
	fMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updatedModel2, _ := m.Update(fMsg)
	m2 := updatedModel2.(Model)
	
	// Verify 'f' was added to filter input, not setting force install
	if m2.install.forceInstall {
		t.Error("Expected install.forceInstall to be false (should not be set during filtering)")
	}
	if m2.filter.input != "f" {
		t.Errorf("Expected filter.input to be 'f', got '%s'", m2.filter.input)
	}
}

// TestFilter_EnterAppliesFilter tests that Enter applies filter instead of installing
func TestFilter_EnterAppliesFilter(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Set package info to make installation possible
	model.pkgInfo.current = &services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}

	// Start filtering
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)

	// Press Enter while filtering
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel2, _ := m.Update(enterMsg)
	m2 := updatedModel2.(Model)

	// Verify state did NOT transition to installing
	if m2.state != StateList {
		t.Errorf("Expected state to remain StateList, got %v", m2.state)
	}
}

// TestFilter_TypingQDoesNotQuit tests that 'q' is typed into filter instead of quitting
func TestFilter_TypingQDoesNotQuit(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Start filtering
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)

	// Press 'q' - should be typed into filter, not quit
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel2, cmd := m.Update(qMsg)
	m2 := updatedModel2.(Model)

	// Verify no quit command was returned
	if cmd != nil {
		t.Error("Expected no command when typing 'q' in filter mode")
	}

	// Verify 'q' was added to filter input
	if m2.filter.input != "q" {
		t.Errorf("Expected filter.input to be 'q', got '%s'", m2.filter.input)
	}
}

// TestFilter_UpdatesSelectedPackage tests that navigation works during filtering
func TestFilter_UpdatesSelectedPackage(t *testing.T) {
	model := createTestModel()
	model.state = StateList
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Start filtering
	filterMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterMsg)
	m := updatedModel.(Model)
	
	// Verify filtering is active
	if !m.filter.active {
		t.Error("Expected filter.active to be true")
	}

	// Move down - in filter mode, arrow keys are handled by default case
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel2, _ := m.Update(downMsg)
	m2 := updatedModel2.(Model)

	// Verify still in filtering mode
	if !m2.filter.active {
		t.Error("Expected to remain in filtering state after navigation")
	}

	// Verify selected package is still set
	if m2.selectedPackage == nil {
		t.Error("Expected selectedPackage to remain set during filtering")
	}
}
