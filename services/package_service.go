// Package services provides the core business logic for package management operations.
//
// This package implements:
//   - Package version checking (latest and installed versions)
//   - Package status determination (installed, update available, not installed)
//   - Package installation with real-time log streaming
//   - Tool availability verification (npm/pnpm)
//
// The PackageService interface abstracts package management operations and can be
// mocked for testing. The default implementation uses CommandExecutor for running
// npm/pnpm commands and VersionComparator for semantic version comparison.
//
// Architecture:
//
//	PackageService
//	    ├── CommandExecutor (utils package) - executes npm/pnpm commands
//	    └── VersionComparator (utils package) - compares semantic versions
//
// Example usage:
//
//	executor := utils.NewCommandExecutor()
//	comparator := utils.NewVersionComparator()
//	service := services.NewPackageService(executor, comparator)
//
//	// Get package information
//	info := service.GetPackageInfo(pkg)
//	fmt.Printf("Status: %s, Latest: %s\n", info.Status, info.LatestVersion)
//
//	// Install package with real-time logs
//	logChan, errChan := service.InstallPackage(pkg)
//	for log := range logChan {
//	    fmt.Println(log)
//	}
//	if err := <-errChan; err != nil {
//	    fmt.Printf("Installation failed: %v\n", err)
//	}
package services

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/utils"
)

// PackageStatus represents the current state of a package.
type PackageStatus string

const (
	// StatusNotInstalled indicates the package is not installed locally
	StatusNotInstalled PackageStatus = "Not Installed"
	
	// StatusInstalled indicates the package is installed and up-to-date
	StatusInstalled PackageStatus = "Installed"
	
	// StatusUpdateAvailable indicates a newer version is available
	StatusUpdateAvailable PackageStatus = "Update Available"
	
	// StatusChecking indicates version information is being fetched
	StatusChecking PackageStatus = "Checking..."
	
	// StatusError indicates an error occurred while checking the package
	StatusError PackageStatus = "Error"
)

// PackageInfo contains complete information about a package including
// its versions and current status.
type PackageInfo struct {
	// Package is the package definition from the config
	Package config.Package
	
	// LatestVersion is the latest version available from the registry
	LatestVersion string
	
	// InstalledVersion is the currently installed version (empty if not installed)
	InstalledVersion string
	
	// Status is the current state of the package
	Status PackageStatus
	
	// Error contains any error that occurred during version checking
	Error error
}

// PackageService defines the interface for package management operations.
// It handles version checking, installation, and tool availability verification.
// All methods accept a context.Context for timeout and cancellation support.
type PackageService interface {
	// GetPackageInfo retrieves complete information about a package including
	// its latest version, installed version, and current status.
	// The context controls the lifetime of the underlying commands.
	GetPackageInfo(ctx context.Context, pkg config.Package) PackageInfo

	// InstallPackage executes the installation command for a package asynchronously.
	// The context controls the lifetime of the installation process.
	// It returns two channels:
	// - logChan: receives real-time log output from the installation process
	// - errChan: receives a single error if installation fails, or nil on success
	// Both channels are closed when the installation completes.
	InstallPackage(ctx context.Context, pkg config.Package) (<-chan string, <-chan error)

	// CheckToolAvailability checks if a tool (npm or pnpm) is available in the system PATH.
	// The context controls the lifetime of the check command.
	// It returns true if the tool can be executed, false otherwise.
	CheckToolAvailability(ctx context.Context, tool string) bool
}

// packageService is the default implementation of PackageService
type packageService struct {
	executor          utils.CommandExecutor
	versionComparator utils.VersionComparator
}

// NewPackageService creates a new PackageService instance with the provided
// command executor and version comparator.
func NewPackageService(executor utils.CommandExecutor, versionComparator utils.VersionComparator) PackageService {
	return &packageService{
		executor:          executor,
		versionComparator: versionComparator,
	}
}

// GetPackageInfo retrieves complete information about a package including
// its latest version, installed version, and current status.
//
// Algorithm:
// 1. Execute version_check_command to get latest version from registry
// 2. Parse the latest version using VersionComparator
// 3. Execute local_version_command to get installed version
// 4. Parse the installed version using VersionComparator
// 5. Compare versions and determine PackageStatus
// 6. Handle errors appropriately (set StatusError when commands fail)
//
// Preconditions:
//   - pkg is valid and non-nil
//   - pkg.VersionCheckCommand and pkg.LocalVersionCommand are executable
//   - executor and versionComparator are initialized
//
// Postconditions:
//   - Returns PackageInfo with Status set correctly
//   - If any command fails, Status is StatusError and Error field is set
//   - Version strings are normalized to semver format
func (s *packageService) GetPackageInfo(ctx context.Context, pkg config.Package) PackageInfo {
	info := PackageInfo{
		Package: pkg,
		Status:  StatusChecking,
	}

	// Step 1: Get latest version from registry
	latestOutput, err := s.executor.Execute(ctx, pkg.VersionCheckCommand)
	if err != nil {
		info.Status = StatusError
		info.Error = fmt.Errorf("failed to check latest version for '%s': %w\n\n"+
			"Possible causes:\n"+
			"  1. Network connection issues - check your internet connection\n"+
			"  2. npm registry is unavailable - try again later\n"+
			"  3. Package name is incorrect - verify the package exists on npm\n"+
			"  4. Command syntax error - check version-check-command in config.yaml", pkg.Name, err)
		return info
	}

	// Step 2: Parse latest version
	latestVersion, err := s.versionComparator.ParseVersion(latestOutput)
	if err != nil {
		info.Status = StatusError
		info.Error = fmt.Errorf("failed to parse latest version for '%s': %w\n\n"+
			"Command output: %s\n\n"+
			"Suggestion: The version-check-command may not be returning a valid version number. "+
			"Verify the command output format", pkg.Name, err, latestOutput)
		return info
	}
	info.LatestVersion = latestVersion

	// Step 3: Get installed version
	installedOutput, err := s.executor.Execute(ctx, pkg.LocalVersionCommand)
	if err != nil {
		errStr := strings.ToLower(err.Error())
		isNotFound := strings.Contains(errStr, "command not found") ||
			strings.Contains(errStr, "not recognized") ||
			strings.Contains(errStr, "executable file not found") ||
			strings.Contains(errStr, "no such file or directory")

		if isNotFound {
			// Not installed is not an error - the command may fail if package is not installed
			info.InstalledVersion = ""
			info.Status = StatusNotInstalled
			return info
		}

		// The command exists but execution failed (e.g., node version too low, syntax error)
		info.InstalledVersion = ""
		info.Status = StatusError
		info.Error = fmt.Errorf("local execution failed (e.g., incompatible Node.js version): %w", err)
		return info
	}

	// Step 4: Parse installed version
	installedVersion, err := s.versionComparator.ParseVersion(installedOutput)
	if err != nil {
		// If we can't parse the version, treat as not installed
		info.InstalledVersion = ""
		info.Status = StatusNotInstalled
		return info
	}
	info.InstalledVersion = installedVersion

	// Step 5: Compare versions and determine status
	if info.InstalledVersion == "" {
		info.Status = StatusNotInstalled
	} else if s.versionComparator.IsNewerVersion(info.LatestVersion, info.InstalledVersion) {
		info.Status = StatusUpdateAvailable
	} else {
		info.Status = StatusInstalled
	}

	return info
}

// InstallPackage executes the installation command for a package asynchronously.
// It returns two channels:
// - logChan: receives real-time log output from the installation process
// - errChan: receives a single error if installation fails, or nil on success
// Both channels are closed when the installation completes.
//
// Algorithm:
// 1. Parse and prepare command using CommandExecutor
// 2. Setup stdout and stderr pipes
// 3. Start command execution
// 4. Stream output to logChan line by line
// 5. Wait for completion and send error (if any) to errChan
// 6. Close both channels
//
// Preconditions:
//   - pkg.InstallCommand is valid and executable
//   - npm/pnpm is installed on the system
//
// Postconditions:
//   - Installation command is executed asynchronously
//   - Log lines are sent to logChan as they arrive
//   - Final error (if any) is sent to errChan
//   - Both channels are closed when installation completes
func (s *packageService) InstallPackage(ctx context.Context, pkg config.Package) (<-chan string, <-chan error) {
	// Use buffer sizes from config for optimal performance
	// Log buffer: large enough to prevent blocking during UI updates
	// Error buffer: 1 is sufficient as only one error is sent
	logChan := make(chan string, config.DefaultLogBufferSize)
	errChan := make(chan error, config.DefaultErrorBufferSize)

	go func() {
		defer close(logChan)
		defer close(errChan)

		// Step 1: Start command via CommandExecutor interface (enables mocking in tests)
		reader, err := s.executor.ExecuteWithStream(ctx, pkg.InstallCommand)
		if err != nil {
			errChan <- fmt.Errorf("failed to start installation command for '%s': %w\n\n"+
				"Possible causes:\n"+
				"  1. npm/pnpm is not installed - install Node.js and npm first\n"+
				"  2. Command not found in PATH - verify npm/pnpm is accessible\n"+
				"  3. Permission denied - try running with appropriate permissions\n"+
				"  4. Invalid command syntax - check install-command in config.yaml", pkg.Name, err)
			return
		}

		// Step 2: Stream merged stdout/stderr output using custom split function
		// This handles pnpm's use of '\r' for dynamic progress bars without buffering endlessly
		scanner := bufio.NewScanner(reader)
		scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			// Find the next \r or \n
			for i := 0; i < len(data); i++ {
				if data[i] == '\n' || data[i] == '\r' {
					return i + 1, data[:i], nil
				}
			}
			// If we're at EOF, we have a final, non-terminated line. Return it.
			if atEOF {
				return len(data), data, nil
			}
			// Request more data.
			return 0, nil, nil
		})

		for scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				logChan <- text
			}
		}

		// Step 3: Close reader (waits for command completion) and check result
		if err := reader.Close(); err != nil {
			errChan <- fmt.Errorf("installation failed for '%s': %w\n\n"+
				"Possible causes:\n"+
				"  1. Network error - check your internet connection\n"+
				"  2. Insufficient disk space - free up some disk space\n"+
				"  3. Permission denied - you may need administrator/sudo privileges\n"+
				"  4. Package not found - verify the package name is correct\n"+
				"  5. Registry error - the npm registry may be temporarily unavailable\n\n"+
				"Suggestion: Check the installation logs above for more details", pkg.Name, err)
			return
		}

		logChan <- fmt.Sprintf("✓ Installation of '%s' completed successfully", pkg.Name)
	}()

	return logChan, errChan
}

// CheckToolAvailability checks if a tool (npm or pnpm) is available in the system PATH.
// It executes a simple version check command to verify the tool can be executed.
//
// Algorithm:
// 1. Construct a version check command for the tool
// 2. Execute the command using CommandExecutor
// 3. Return true if command succeeds, false if it fails
//
// Preconditions:
//   - tool is a non-empty string (typically "npm" or "pnpm")
//   - executor is initialized
//
// Postconditions:
//   - Returns true if tool is available and executable
//   - Returns false if tool is not found or command fails
//   - No side effects (read-only operation)
func (s *packageService) CheckToolAvailability(ctx context.Context, tool string) bool {
	// Step 1: Construct version check command
	// Using --version flag which is supported by both npm and pnpm
	command := fmt.Sprintf("%s --version", tool)

	// Step 2: Execute the command
	_, err := s.executor.Execute(ctx, command)

	// Step 3: Return true if command succeeds, false otherwise
	return err == nil
}
