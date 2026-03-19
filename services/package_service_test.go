package services

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Sinton/my-pnpm-installer/config"
)

// mockCommandExecutor is a mock implementation of CommandExecutor for testing
type mockCommandExecutor struct {
	executeFunc           func(command string) (string, error)
	executeWithStreamFunc func(command string) (io.ReadCloser, error)
	parseCommandFunc      func(command string) (string, []string)
}

func (m *mockCommandExecutor) Execute(ctx context.Context, command string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(command)
	}
	return "", nil
}

func (m *mockCommandExecutor) ExecuteWithStream(ctx context.Context, command string) (io.ReadCloser, error) {
	if m.executeWithStreamFunc != nil {
		return m.executeWithStreamFunc(command)
	}
	// Default: return a reader with empty content that closes successfully
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockCommandExecutor) ParseCommand(command string) (string, []string) {
	if m.parseCommandFunc != nil {
		return m.parseCommandFunc(command)
	}
	return "sh", []string{"-c", command}
}

// ValidateCommand implements utils.CommandExecutor
func (m *mockCommandExecutor) ValidateCommand(command string) error {
	// Default: allow all commands for testing
	return nil
}

// mockVersionComparator is a mock implementation of VersionComparator for testing
type mockVersionComparator struct {
	parseVersionFunc  func(output string) (string, error)
	isNewerVersionFunc func(v1, v2 string) bool
}

func (m *mockVersionComparator) ParseVersion(output string) (string, error) {
	if m.parseVersionFunc != nil {
		return m.parseVersionFunc(output)
	}
	return output, nil
}

func (m *mockVersionComparator) CompareVersions(v1, v2 string) (int, error) {
	return 0, nil
}

func (m *mockVersionComparator) IsNewerVersion(v1, v2 string) bool {
	if m.isNewerVersionFunc != nil {
		return m.isNewerVersionFunc(v1, v2)
	}
	return false
}

func TestGetPackageInfo_UpdateAvailable(t *testing.T) {
	// Setup mock executor that returns different versions
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "5.3.0", nil
			}
			if command == "npm list -g typescript --depth=0" {
				return "5.2.0", nil
			}
			return "", errors.New("unknown command")
		},
	}

	// Setup mock version comparator
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			return output, nil
		},
		isNewerVersionFunc: func(v1, v2 string) bool {
			return v1 == "5.3.0" && v2 == "5.2.0"
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	if info.Status != StatusUpdateAvailable {
		t.Errorf("Expected status %s, got %s", StatusUpdateAvailable, info.Status)
	}
	if info.LatestVersion != "5.3.0" {
		t.Errorf("Expected latest version 5.3.0, got %s", info.LatestVersion)
	}
	if info.InstalledVersion != "5.2.0" {
		t.Errorf("Expected installed version 5.2.0, got %s", info.InstalledVersion)
	}
	if info.Error != nil {
		t.Errorf("Expected no error, got %v", info.Error)
	}
}

func TestGetPackageInfo_Installed(t *testing.T) {
	// Setup mock executor that returns same version
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "5.3.0", nil
			}
			if command == "npm list -g typescript --depth=0" {
				return "5.3.0", nil
			}
			return "", errors.New("unknown command")
		},
	}

	// Setup mock version comparator
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			return output, nil
		},
		isNewerVersionFunc: func(v1, v2 string) bool {
			return false // Same version
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	if info.Status != StatusInstalled {
		t.Errorf("Expected status %s, got %s", StatusInstalled, info.Status)
	}
	if info.LatestVersion != "5.3.0" {
		t.Errorf("Expected latest version 5.3.0, got %s", info.LatestVersion)
	}
	if info.InstalledVersion != "5.3.0" {
		t.Errorf("Expected installed version 5.3.0, got %s", info.InstalledVersion)
	}
}

func TestGetPackageInfo_NotInstalled(t *testing.T) {
	// Setup mock executor where local version command fails
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "5.3.0", nil
			}
			if command == "npm list -g typescript --depth=0" {
				return "", errors.New("package not found")
			}
			return "", errors.New("unknown command")
		},
	}

	// Setup mock version comparator
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			return output, nil
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	if info.Status != StatusNotInstalled {
		t.Errorf("Expected status %s, got %s", StatusNotInstalled, info.Status)
	}
	if info.LatestVersion != "5.3.0" {
		t.Errorf("Expected latest version 5.3.0, got %s", info.LatestVersion)
	}
	if info.InstalledVersion != "" {
		t.Errorf("Expected empty installed version, got %s", info.InstalledVersion)
	}
}

func TestGetPackageInfo_LatestVersionCheckError(t *testing.T) {
	// Setup mock executor where version check fails
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "", errors.New("network error")
			}
			return "", errors.New("unknown command")
		},
	}

	comparator := &mockVersionComparator{}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	if info.Status != StatusError {
		t.Errorf("Expected status %s, got %s", StatusError, info.Status)
	}
	if info.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestGetPackageInfo_LatestVersionParseError(t *testing.T) {
	// Setup mock executor that returns unparseable version
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "invalid-version", nil
			}
			return "", errors.New("unknown command")
		},
	}

	// Setup mock version comparator that fails to parse
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			return "", errors.New("invalid version format")
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	if info.Status != StatusError {
		t.Errorf("Expected status %s, got %s", StatusError, info.Status)
	}
	if info.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestGetPackageInfo_InstalledVersionParseError(t *testing.T) {
	// Setup mock executor that returns unparseable installed version
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "5.3.0", nil
			}
			if command == "npm list -g typescript --depth=0" {
				return "invalid-version", nil
			}
			return "", errors.New("unknown command")
		},
	}

	// Setup mock version comparator
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			if output == "5.3.0" {
				return "5.3.0", nil
			}
			return "", errors.New("invalid version format")
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	// When installed version can't be parsed, treat as not installed
	if info.Status != StatusNotInstalled {
		t.Errorf("Expected status %s, got %s", StatusNotInstalled, info.Status)
	}
	if info.InstalledVersion != "" {
		t.Errorf("Expected empty installed version, got %s", info.InstalledVersion)
	}
}

func TestInstallPackage_Success(t *testing.T) {
	// Setup mock executor with stream support
	executor := &mockCommandExecutor{
		executeWithStreamFunc: func(command string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("Downloading package...\nInstalling...\n")), nil
		},
	}
	comparator := &mockVersionComparator{}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:           "typescript",
		InstallCommand: "echo 'Installing typescript'",
	}

	logChan, errChan := service.InstallPackage(context.Background(), pkg)

	// Collect all logs
	var logs []string
	for log := range logChan {
		logs = append(logs, log)
	}

	// Check for errors
	err := <-errChan

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(logs) == 0 {
		t.Error("Expected at least one log message")
	}

	// Check that completion message is present
	foundCompletion := false
	for _, log := range logs {
		if contains(log, "Installation") && contains(log, "completed successfully") {
			foundCompletion = true
			break
		}
	}

	if !foundCompletion {
		t.Error("Expected completion message in logs")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInstallPackage_CommandFailure(t *testing.T) {
	// Setup mock executor that fails to start
	executor := &mockCommandExecutor{
		executeWithStreamFunc: func(command string) (io.ReadCloser, error) {
			return nil, errors.New("command not found")
		},
	}
	comparator := &mockVersionComparator{}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:           "typescript",
		InstallCommand: "exit 1", // Command that fails
	}

	logChan, errChan := service.InstallPackage(context.Background(), pkg)

	// Drain log channel
	for range logChan {
		// Just consume logs
	}

	// Check for error
	err := <-errChan

	if err == nil {
		t.Error("Expected error for failed command, got nil")
	}
}

func TestCheckToolAvailability_Available(t *testing.T) {
	// Setup mock executor that succeeds for version check
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm --version" {
				return "10.2.3", nil
			}
			if command == "pnpm --version" {
				return "8.10.0", nil
			}
			return "", errors.New("unknown command")
		},
	}

	comparator := &mockVersionComparator{}
	service := NewPackageService(executor, comparator)

	// Test npm availability
	if !service.CheckToolAvailability(context.Background(), "npm") {
		t.Error("Expected npm to be available")
	}

	// Test pnpm availability
	if !service.CheckToolAvailability(context.Background(), "pnpm") {
		t.Error("Expected pnpm to be available")
	}
}

func TestCheckToolAvailability_NotAvailable(t *testing.T) {
	// Setup mock executor that fails for version check
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			return "", errors.New("command not found")
		},
	}

	comparator := &mockVersionComparator{}
	service := NewPackageService(executor, comparator)

	// Test tool not available
	if service.CheckToolAvailability(context.Background(), "nonexistent-tool") {
		t.Error("Expected nonexistent-tool to be unavailable")
	}
}

func TestCheckToolAvailability_EmptyTool(t *testing.T) {
	// Setup mock executor
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == " --version" {
				return "", errors.New("invalid command")
			}
			return "", errors.New("unknown command")
		},
	}

	comparator := &mockVersionComparator{}
	service := NewPackageService(executor, comparator)

	// Test empty tool name
	if service.CheckToolAvailability(context.Background(), "") {
		t.Error("Expected empty tool name to be unavailable")
	}
}

func TestInstallPackage_ChannelClosure(t *testing.T) {
	// Setup mock executor with stream support
	executor := &mockCommandExecutor{
		executeWithStreamFunc: func(command string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("test output\n")), nil
		},
	}
	comparator := &mockVersionComparator{}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:           "typescript",
		InstallCommand: "echo 'test'",
	}

	logChan, errChan := service.InstallPackage(context.Background(), pkg)

	// Verify logChan is closed after completion
	logCount := 0
	for range logChan {
		logCount++
	}

	// Verify errChan is closed after completion
	err, ok := <-errChan
	if ok {
		// Channel should be closed, but if it's not, check the error
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}

	// Verify we received at least one log message
	if logCount == 0 {
		t.Error("Expected at least one log message before channel closure")
	}
}

func TestInstallPackage_StderrOutput(t *testing.T) {
	// Setup mock executor that returns merged stdout/stderr content
	executor := &mockCommandExecutor{
		executeWithStreamFunc: func(command string) (io.ReadCloser, error) {
			// Simulate merged output (stdout and stderr are combined by ExecuteWithStream)
			return io.NopCloser(strings.NewReader("progress info\nerror message\n")), nil
		},
	}
	comparator := &mockVersionComparator{}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:           "test-package",
		InstallCommand: "echo 'error message' >&2",
	}

	logChan, errChan := service.InstallPackage(context.Background(), pkg)

	// Collect all logs
	var logs []string
	for log := range logChan {
		logs = append(logs, log)
	}

	// Check for errors
	err := <-errChan
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify stderr output is captured (now merged without [ERROR] prefix)
	foundOutput := false
	for _, log := range logs {
		if contains(log, "error message") {
			foundOutput = true
			break
		}
	}

	if !foundOutput {
		t.Error("Expected stderr output to be captured in logs")
	}
}

func TestInstallPackage_ExecuteWithStreamCalled(t *testing.T) {
	// Track if ExecuteWithStream was called with the correct command
	executeWithStreamCalled := false
	capturedCommand := ""

	executor := &mockCommandExecutor{
		executeWithStreamFunc: func(command string) (io.ReadCloser, error) {
			executeWithStreamCalled = true
			capturedCommand = command
			return io.NopCloser(strings.NewReader("done\n")), nil
		},
	}

	comparator := &mockVersionComparator{}
	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:           "typescript",
		InstallCommand: "npm install -g typescript",
	}

	logChan, errChan := service.InstallPackage(context.Background(), pkg)

	// Drain channels
	for range logChan {
	}
	<-errChan

	if !executeWithStreamCalled {
		t.Error("Expected ExecuteWithStream to be called during installation")
	}
	if capturedCommand != "npm install -g typescript" {
		t.Errorf("Expected command 'npm install -g typescript', got '%s'", capturedCommand)
	}
}

func TestGetPackageInfo_EmptyVersionStrings(t *testing.T) {
	// Setup mock executor that returns empty strings
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			return "", nil
		},
	}

	// Setup mock version comparator that returns empty version
	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			if output == "" {
				return "", errors.New("empty output")
			}
			return output, nil
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	info := service.GetPackageInfo(context.Background(), pkg)

	// Should get error status when latest version can't be parsed
	if info.Status != StatusError {
		t.Errorf("Expected status %s, got %s", StatusError, info.Status)
	}
}

func TestGetPackageInfo_MultipleCallsConsistency(t *testing.T) {
	// Setup mock executor with consistent responses
	executor := &mockCommandExecutor{
		executeFunc: func(command string) (string, error) {
			if command == "npm view typescript version" {
				return "5.3.0", nil
			}
			if command == "npm list -g typescript --depth=0" {
				return "5.2.0", nil
			}
			return "", errors.New("unknown command")
		},
	}

	comparator := &mockVersionComparator{
		parseVersionFunc: func(output string) (string, error) {
			return output, nil
		},
		isNewerVersionFunc: func(v1, v2 string) bool {
			return v1 == "5.3.0" && v2 == "5.2.0"
		},
	}

	service := NewPackageService(executor, comparator)

	pkg := config.Package{
		Name:                 "typescript",
		VersionCheckCommand:  "npm view typescript version",
		LocalVersionCommand:  "npm list -g typescript --depth=0",
	}

	// Call GetPackageInfo multiple times
	info1 := service.GetPackageInfo(context.Background(), pkg)
	info2 := service.GetPackageInfo(context.Background(), pkg)

	// Verify consistency
	if info1.Status != info2.Status {
		t.Errorf("Expected consistent status, got %s and %s", info1.Status, info2.Status)
	}
	if info1.LatestVersion != info2.LatestVersion {
		t.Errorf("Expected consistent latest version, got %s and %s", info1.LatestVersion, info2.LatestVersion)
	}
	if info1.InstalledVersion != info2.InstalledVersion {
		t.Errorf("Expected consistent installed version, got %s and %s", info1.InstalledVersion, info2.InstalledVersion)
	}
}
