// Package main is the entry point for my-pnpm-installer, a cross-platform TUI tool
// for managing and installing pnpm/npm global packages.
//
// my-pnpm-installer provides an interactive terminal interface for:
//   - Viewing package versions (latest and installed)
//   - Installing and updating global npm/pnpm packages
//   - Managing packages through a YAML configuration file
//
// The tool uses the Bubble Tea framework for the TUI and supports Windows, macOS, and Linux.
//
// Usage:
//
//	my-pnpm-installer
//
// The tool will search for config.yaml in:
//  1. Current directory
//  2. ~/.config/pnpm-manager/
//
// Example config.yaml:
//
//	packages:
//	  - name: typescript
//	    display_name: TypeScript
//	    install_command: npm install -g typescript@latest
//	    version_check_command: npm view typescript version
//	    local_version_command: npm list -g typescript --depth=0
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/i18n"
	"github.com/Sinton/my-pnpm-installer/services"
	"github.com/Sinton/my-pnpm-installer/ui"
	"github.com/Sinton/my-pnpm-installer/utils"
)

// main is the entry point for the application.
// It orchestrates the following workflow:
//  1. Load and validate configuration from config.yaml
//  2. Initialize service layer components (command executor, version comparator, package service)
//  3. Verify npm/pnpm availability on the system
//  4. Create and start the Bubble Tea TUI
//  5. Handle errors and exit with appropriate status codes
//
// Exit codes:
//   - 0: Success (user quit normally)
//   - 1: Error (config not found, validation failed, npm/pnpm not available, TUI error)
func main() {
	// Step 0: Initialize internationalization
	i18n.Init()
	
	// Step 1: Load configuration
	// findConfigPath is handled by config.LoadConfig when path is empty
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Step 3: Initialize services
	executor := utils.NewCommandExecutor()
	versionComparator := utils.NewVersionComparator()
	service := services.NewPackageService(executor, versionComparator)

	// Step 3.5: Check if npm/pnpm is available before starting UI
	if !service.CheckToolAvailability(context.Background(), "npm") && !service.CheckToolAvailability(context.Background(), "pnpm") {
		fmt.Fprintf(os.Stderr, "Error: Neither npm nor pnpm is available on your system\n\n"+
			"This tool requires npm or pnpm to be installed to manage packages.\n\n"+
			"To install npm:\n"+
			"  - Install Node.js from https://nodejs.org/ (includes npm)\n"+
			"  - Or use your system's package manager:\n"+
			"    • macOS: brew install node\n"+
			"    • Ubuntu/Debian: sudo apt install nodejs npm\n"+
			"    • Windows: Download from https://nodejs.org/\n\n"+
			"To install pnpm:\n"+
			"  - npm install -g pnpm\n"+
			"  - Or see https://pnpm.io/installation\n")
		os.Exit(1)
	}

	// Step 4: Create UI Model
	model := ui.NewModel(cfg.Packages, service)

	// Step 5: Start Bubble Tea program with alternate screen
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Step 6: Run the program and handle errors
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n\n"+
			"Possible causes:\n"+
			"  1. Terminal is not compatible with TUI applications\n"+
			"  2. Terminal size is too small - try resizing your terminal\n"+
			"  3. Terminal capabilities are limited\n\n"+
			"Suggestion: Try running in a different terminal emulator\n", err)
		os.Exit(1)
	}
}