package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/i18n"
	"github.com/Sinton/my-pnpm-installer/services"
)

// Lipgloss styles for UI components
var (
	// Panel styles
	panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)
	
	// Title styles
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	// Label styles
	labelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99"))
	
	// Status styles
	statusInstalledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)
	
	statusUpdateAvailableStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	statusNotInstalledStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	statusErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	statusCheckingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("99"))
	
	// Help text style
	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	
	// Error style
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
	
	// Log style
	logStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
)

// renderListView renders the main list view with side-by-side layout.
// Layout: [Package List | Package Details]
//
// Preconditions:
//   - Model state is StateList
//   - width and height are set
//
// Postconditions:
//   - Returns formatted string with two-panel layout
//   - Left panel shows package list
//   - Right panel shows selected package details
func (m Model) renderListView() string {
	// Minimum terminal size check
	const minWidth = config.MinTerminalWidth
	const minHeight = config.MinTerminalHeight

	// Handle terminal too small scenario
	if m.width < minWidth || m.height < minHeight {
		return m.renderTerminalTooSmall(minWidth, minHeight)
	}
	
	// Calculate panel dimensions
	leftWidth := m.width / 2
	rightWidth := m.width - leftWidth
	panelHeight := m.height - 4 // Reserve space for help text
	
	// Render left panel (package list)
	leftPanel := m.list.View()
	
	// Render right panel (package details)
	var rightPanel string
	if m.pkgInfo.isLoading {
		// Show loading indicator
		rightPanel = lipgloss.NewStyle().
			Width(rightWidth - 4).
			Height(panelHeight - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(statusCheckingStyle.Render(i18n.T().LoadingPackageInfo))
	} else if m.pkgInfo.current != nil {
		rightPanel = m.renderPackageDetails(*m.pkgInfo.current)
	} else {
		rightPanel = lipgloss.NewStyle().
			Width(rightWidth - 4).
			Height(panelHeight - 4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(i18n.T().NoPackageSelected)
	}
	
	// Style the panels
	leftPanelStyled := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Render(leftPanel)
	
	rightPanelStyled := panelStyle.
		Width(rightWidth - 4).
		Height(panelHeight - 4).
		Render(rightPanel)
	
	// Join panels horizontally
	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanelStyled,
		rightPanelStyled,
	)
	
	// Add install result notification (if any)
	var resultNotification string
	if m.install.lastResult != "" {
		if m.install.lastSuccess {
			resultNotification = statusInstalledStyle.Render(m.install.lastResult)
		} else {
			resultNotification = statusErrorStyle.Render(m.install.lastResult)
		}
	}

	// Add help text footer
	var helpText string
	if m.filter.active {
		// Show filter input
		helpText = helpStyle.Render(fmt.Sprintf("过滤: %s_", m.filter.input))
	} else if m.filter.filteredPackages != nil {
		// Show filter status
		helpText = helpStyle.Render(fmt.Sprintf("%s | 按 Esc 清除过滤", i18n.T().ListHelp))
	} else {
		helpText = helpStyle.Render(i18n.T().ListHelp)
	}

	// Join main view, notification, and help text vertically
	parts := []string{mainView}
	if resultNotification != "" {
		parts = append(parts, resultNotification)
	}
	parts = append(parts, helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		parts...,
	)
}

// renderPackageDetails renders the details panel for the selected package.
// Shows package name, versions, status, and any errors.
//
// Preconditions:
//   - info is a valid PackageInfo struct
//
// Postconditions:
//   - Returns formatted string with package details
func (m Model) renderPackageDetails(info services.PackageInfo) string {
	var b strings.Builder
	
	// Package title
	b.WriteString(titleStyle.Render(info.Package.Name))
	b.WriteString("\n\n")

	// Package name (extracted from install command)
	b.WriteString(labelStyle.Render(i18n.T().PackageLabel + ": "))
	b.WriteString(extractPackageName(info.Package.InstallCommand))
	b.WriteString("\n\n")

	// Latest version
	b.WriteString(labelStyle.Render(i18n.T().LatestVersionLabel + ": "))
	if info.LatestVersion != "" {
		b.WriteString(info.LatestVersion)
	} else {
		b.WriteString(i18n.T().StatusUnknown)
	}
	b.WriteString("\n\n")

	// Installed version
	b.WriteString(labelStyle.Render(i18n.T().InstalledLabel + ": "))
	if info.InstalledVersion != "" {
		b.WriteString(info.InstalledVersion)
	} else {
		b.WriteString(i18n.T().StatusNotInstalled)
	}
	b.WriteString("\n\n")

	// Status with color coding
	b.WriteString(labelStyle.Render(i18n.T().StatusLabel + ": "))
	var statusText string
	switch info.Status {
	case services.StatusInstalled:
		statusText = i18n.T().StatusInstalled
		b.WriteString(statusInstalledStyle.Render(statusText))
	case services.StatusUpdateAvailable:
		statusText = i18n.T().StatusUpdateAvail
		b.WriteString(statusUpdateAvailableStyle.Render(statusText))
	case services.StatusNotInstalled:
		statusText = i18n.T().StatusNotInstalled
		b.WriteString(statusNotInstalledStyle.Render(statusText))
	case services.StatusError:
		statusText = string(info.Status)
		b.WriteString(statusErrorStyle.Render(statusText))
	case services.StatusChecking:
		statusText = string(info.Status)
		b.WriteString(statusCheckingStyle.Render(statusText))
	default:
		statusText = string(info.Status)
		b.WriteString(statusText)
	}
	b.WriteString("\n")

	// Error message if present
	if info.Error != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("%v", info.Error)))
	}

	// Calculate panel dimensions
	rightWidth := m.width - (m.width / 2) - 6 // Account for padding and borders
	displayHeight := m.height - 8
	if displayHeight < 1 {
		displayHeight = 1
	}

	// Wrap the entire content to the available width first
	// This ensures that "logical lines" match "visual lines" for accurate scrolling
	wrappedContent := lipgloss.NewStyle().Width(rightWidth).Render(b.String())
	lines := strings.Split(wrappedContent, "\n")

	// Bounds check for scroll offset
	if m.pkgInfo.scrollOffset < 0 {
		m.pkgInfo.scrollOffset = 0
	}
	if len(lines) > 0 && m.pkgInfo.scrollOffset >= len(lines) {
		m.pkgInfo.scrollOffset = len(lines) - 1
	}

	end := m.pkgInfo.scrollOffset + displayHeight
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[m.pkgInfo.scrollOffset:end], "\n")
}

// renderInstallView renders the installation view showing real-time progress.
// Only the latest log line is displayed, simulating pnpm's dynamic terminal
// output where progress lines overwrite each other in place.
//
// Preconditions:
//   - Model state is StateInstalling
//   - installLogs contains log lines
//
// Postconditions:
//   - Returns formatted string with current installation progress
//   - Shows only the most recent log line (dynamic single-line update)
func (m Model) renderInstallView() string {
	var b strings.Builder

	// Title
	packageName := ""
	if m.selectedPackage != nil {
		packageName = m.selectedPackage.Name
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s: %s", i18n.T().InstallingTitle, packageName)))
	b.WriteString("\n\n")

	// Calculate available height for logs
	// Total height minus padding, title, and help footer
	logHeight := m.height - 8
	if logHeight < 1 {
		logHeight = 1
	}

	// Calculate which logs to show based on scrollOffset and available logHeight
	endIndex := m.install.scrollOffset
	if endIndex > len(m.install.logs) {
		endIndex = len(m.install.logs)
	}
	startIndex := endIndex - logHeight
	if startIndex < 0 {
		startIndex = 0
	}

	if len(m.install.logs) > 0 {
		for i := startIndex; i < endIndex; i++ {
			b.WriteString(logStyle.Render(m.install.logs[i]))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(logStyle.Render("..."))
		b.WriteString("\n")
	}

	// Add help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(i18n.T().PressQToReturn))

	return panelStyle.
		Width(m.width - 4).
		Height(m.height - 2).
		Render(b.String())
}

// renderTerminalTooSmall renders a message when the terminal is too small.
// It provides clear instructions to the user on how to fix the issue.
//
// Preconditions:
//   - Terminal dimensions are below minimum requirements
//
// Postconditions:
//   - Returns formatted error message with current and required dimensions
func (m Model) renderTerminalTooSmall(minWidth, minHeight int) string {
	var b strings.Builder
	
	b.WriteString(errorStyle.Render("⚠ Terminal Too Small"))
	b.WriteString("\n\n")
	
	b.WriteString(fmt.Sprintf("Current size: %d × %d\n", m.width, m.height))
	b.WriteString(fmt.Sprintf("Minimum required: %d × %d\n\n", minWidth, minHeight))
	
	b.WriteString("Please resize your terminal window to at least\n")
	b.WriteString(fmt.Sprintf("%d columns wide and %d rows tall.\n\n", minWidth, minHeight))
	
	b.WriteString(helpStyle.Render("Press q to quit"))
	
	// Center the message in the available space
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(b.String())
}
