package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/services"
)

// TestProperty_FilteredNavigationAndSelection verifies Property 1: Fault Condition - Filtered Navigation and Selection
//
// **Validates: Requirements 2.1, 2.2, 2.3**
//
// CRITICAL: This test MUST FAIL on unfixed code - failure confirms the bug exists
// DO NOT attempt to fix the test or the code when it fails
//
// Universal Quantification:
// ∀ filterString ∈ String, navigationKey ∈ {up, down, j, k, enter}:
//   (FilterState = FilterApplied) ∧ (len(VisibleItems) < len(packages)) ⟹
//     selectedPackage = list.SelectedItem() (from filtered view)
//     ∧ selectedPackage ∈ VisibleItems
//     ∧ selectedPackage ≠ packages[list.Index()] (when indices differ)
//
// This property test verifies that:
// 1. When a filter is applied and navigation keys are pressed, the selected package comes from the filtered view
// 2. The selected package is always one of the visible filtered items
// 3. The system does NOT use the unfiltered array index to select packages
func TestProperty_FilteredNavigationAndSelection(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	// Test packages with distinct names for filtering
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	properties.Property("Filtered navigation selects from filtered view, not unfiltered array", prop.ForAll(
		func(filterStr string, navKey string) bool {
			// Create model with test packages
			service := &mockPackageService{}
			model := NewModel(packages, service)
			model.width = 100
			model.height = 30
			model.list.SetSize(50, 26)

			// Apply filter by simulating filter mode
			// Start filter mode with '/'
			filterStartMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
			updatedModel, _ := model.Update(filterStartMsg)
			model = updatedModel.(Model)

			// Type the filter string
			for _, char := range filterStr {
				charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
				updatedModel, _ = model.Update(charMsg)
				model = updatedModel.(Model)
			}

			// Press Enter to apply the filter
			enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
			updatedModel, _ = model.Update(enterMsg)
			model = updatedModel.(Model)

			// Check if filter is applied and has reduced the visible items
			visibleItems := model.list.Items()

			// Skip if filter didn't reduce items (no bug condition)
			if model.filter.filteredPackages == nil || len(visibleItems) >= len(packages) {
				return true // Skip this test case
			}

			// Now press the navigation key
			var keyMsg tea.KeyMsg
			switch navKey {
			case "up":
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			case "down":
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			case "j":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			case "k":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
			case "enter":
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			default:
				return true // Skip invalid key
			}

			updatedModel, _ = model.Update(keyMsg)
			model = updatedModel.(Model)

			// CRITICAL CHECK: Verify selected package is from filtered view
			if model.selectedPackage == nil {
				return true // No selection, skip
			}

			// Get the currently selected item from the list (filtered view)
			selectedItem := model.list.SelectedItem()
			if selectedItem == nil {
				return true // No item selected in list, skip
			}

			pkgItem, ok := selectedItem.(packageItem)
			if !ok {
				t.Logf("Failed to cast selected item to packageItem")
				return false
			}

			expectedPackage := pkgItem.pkg

			// BUG CHECK: The selected package should match the filtered view item
			// If the bug exists, model.selectedPackage will be from packages[list.Index()]
			// which is the WRONG package from the unfiltered array
			if model.selectedPackage.Name != expectedPackage.Name {
				t.Logf("BUG DETECTED: After filtering '%s' and pressing '%s', selected package is '%s' but should be '%s' (from filtered view)",
					filterStr, navKey, model.selectedPackage.Name, expectedPackage.Name)
				t.Logf("  Visible items: %d, Total packages: %d", len(visibleItems), len(packages))
				t.Logf("  List index: %d, Expected package from filtered view: %s", model.list.Index(), expectedPackage.Name)
				return false
			}

			// Verify selected package is in the visible items
			isVisible := false
			for _, item := range visibleItems {
				if pkgItem, ok := item.(packageItem); ok {
					if pkgItem.pkg.Name == model.selectedPackage.Name {
						isVisible = true
						break
					}
				}
			}

			if !isVisible {
				t.Logf("BUG DETECTED: Selected package '%s' is not in visible filtered items", model.selectedPackage.Name)
				return false
			}

			return true
		},
		genFilterString(),
		genNavigationKey(),
	))

	properties.TestingRun(t)
}

// TestConcreteCase_FilterAge_NavigateDown tests the specific case from the bug report
//
// **Validates: Requirements 2.1, 2.2**
//
// CRITICAL: This test MUST FAIL on unfixed code
//
// Concrete test case:
// 1. Apply filter "/age" - should show only "ccusage"
// 2. Press down arrow
// 3. Verify selected package is "ccusage" (the visible filtered item)
// 4. NOT "Claude Code CLI" or another package from unfiltered array
func TestConcreteCase_FilterAge_NavigateDown(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Start filter mode
	filterStartMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterStartMsg)
	model = updatedModel.(Model)

	// Type "age"
	for _, char := range "age" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(Model)
	}

	// Apply filter with Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify filter is applied (custom filter, not bubbles/list built-in)
	if model.filter.filteredPackages == nil {
		t.Fatal("Filter should be applied")
	}

	visibleItems := model.list.Items()
	t.Logf("Visible items after filtering 'age': %d", len(visibleItems))

	// BUG CHECK 1: Visual filtering should reduce visible items
	// Should have only 1 visible item: "ccusage"
	if len(visibleItems) != 1 {
		t.Errorf("BUG CONFIRMED (Visual Filter): Expected 1 visible item after filtering 'age', got %d - filter is not visually working", len(visibleItems))
		// Continue to check navigation bug as well
	}

	// Exit filter mode to test navigation (since filter mode intercepts keys)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(Model)

	// Verify filter state after Esc - custom filter clears on Esc
	if model.filter.filteredPackages == nil {
		t.Log("Filter was cleared by Esc - this is expected behavior")
		// Re-apply filter for navigation test
		filterStartMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		updatedModel, _ = model.Update(filterStartMsg)
		model = updatedModel.(Model)
		
		for _, char := range "age" {
			charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
			updatedModel, _ = model.Update(charMsg)
			model = updatedModel.(Model)
		}
		
		enterMsg = tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ = model.Update(enterMsg)
		model = updatedModel.(Model)
	}

	// Press down arrow to navigate
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(Model)

	// Get the selected item from filtered view
	selectedItem := model.list.SelectedItem()
	if selectedItem == nil {
		t.Fatal("No item selected after navigation")
	}

	pkgItem, ok := selectedItem.(packageItem)
	if !ok {
		t.Fatal("Failed to cast selected item to packageItem")
	}

	expectedPackage := pkgItem.pkg.Name

	// CRITICAL CHECK: model.selectedPackage should match list.SelectedItem()
	if model.selectedPackage == nil {
		t.Fatal("selectedPackage is nil")
	}

	t.Logf("After filtering 'age' and pressing down:")
	t.Logf("  Expected (from list.SelectedItem): %s", expectedPackage)
	t.Logf("  Actual (model.selectedPackage): %s", model.selectedPackage.Name)
	t.Logf("  List index: %d", model.list.Index())
	t.Logf("  Unfiltered package at index %d: %s", model.list.Index(), packages[model.list.Index()].Name)

	// BUG CHECK 2: Navigation should use list.SelectedItem(), not packages[list.Index()]
	if model.selectedPackage.Name != expectedPackage {
		t.Errorf("BUG CONFIRMED (Navigation): After filtering and navigating, model.selectedPackage is '%s' but list.SelectedItem() is '%s'",
			model.selectedPackage.Name, expectedPackage)
		t.Errorf("  This indicates navigation is using packages[%d] instead of list.SelectedItem()", model.list.Index())
	}

	// Additional check: if filter worked correctly, selected package should be "ccusage"
	if len(visibleItems) == 1 && model.selectedPackage.Name != "ccusage" {
		t.Errorf("BUG CONFIRMED: With filter working correctly, expected 'ccusage' to be selected, got '%s'", model.selectedPackage.Name)
	}
}

// TestConcreteCase_FilterAge_PressEnter tests installation with filter applied
//
// **Validates: Requirements 2.3**
//
// CRITICAL: This test MUST FAIL on unfixed code
//
// Concrete test case:
// 1. Apply filter "/age" - should show only "ccusage"
// 2. Press Enter to install
// 3. Verify the correct package "ccusage" would be installed
// 4. NOT a wrong package from the unfiltered array
func TestConcreteCase_FilterAge_PressEnter(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	// Track which package was installed
	var installedPackage *config.Package
	service := &mockPackageServiceWithFuncs{
		getPackageInfoFunc: func(pkg config.Package) services.PackageInfo {
			return services.PackageInfo{
				Package:          pkg,
				LatestVersion:    "2.0.0",
				InstalledVersion: "1.0.0",
				Status:           services.StatusUpdateAvailable,
			}
		},
		installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
			installedPackage = &pkg
			logChan := make(chan string, 1)
			errChan := make(chan error, 1)
			close(logChan)
			errChan <- nil
			return logChan, errChan
		},
	}

	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Set package info to allow installation
	model.pkgInfo.current = &services.PackageInfo{
		Package:          packages[0],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}

	// Start filter mode
	filterStartMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterStartMsg)
	model = updatedModel.(Model)

	// Type "age"
	for _, char := range "age" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(Model)
	}

	// Apply filter with Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Update package info for the filtered selection
	if model.selectedPackage != nil {
		model.pkgInfo.current = &services.PackageInfo{
			Package:          *model.selectedPackage,
			LatestVersion:    "2.0.0",
			InstalledVersion: "1.0.0",
			Status:           services.StatusUpdateAvailable,
		}
	}

	// Get the expected package from filtered view
	selectedItem := model.list.SelectedItem()
	if selectedItem == nil {
		t.Fatal("No item selected after filtering")
	}

	pkgItem, ok := selectedItem.(packageItem)
	if !ok {
		t.Fatal("Failed to cast selected item to packageItem")
	}

	expectedPackage := pkgItem.pkg.Name

	// Press Enter to install (but filter is still applied, so this should be handled by list)
	// We need to exit filter mode first by pressing Escape
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(Model)

	// Now press Enter to install
	enterInstallMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterInstallMsg)
	model = updatedModel.(Model)

	// Check which package was selected for installation
	if installedPackage == nil {
		t.Log("No package was installed (might be skipped if already at latest)")
		// Check model.selectedPackage instead
		if model.selectedPackage == nil {
			t.Fatal("selectedPackage is nil")
		}

		t.Logf("After filtering 'age' and pressing Enter:")
		t.Logf("  Expected (from filtered view): %s", expectedPackage)
		t.Logf("  Actual (model.selectedPackage): %s", model.selectedPackage.Name)

		if model.selectedPackage.Name != expectedPackage {
			t.Errorf("BUG CONFIRMED: After filtering 'age', selected package for installation is '%s' but should be '%s'",
				model.selectedPackage.Name, expectedPackage)
		}

		if model.selectedPackage.Name != "ccusage" {
			t.Errorf("BUG CONFIRMED: Expected 'ccusage' to be selected for installation, got '%s'", model.selectedPackage.Name)
		}
	} else {
		t.Logf("Package installed: %s", installedPackage.Name)
		t.Logf("Expected package: %s", expectedPackage)

		if installedPackage.Name != expectedPackage {
			t.Errorf("BUG CONFIRMED: Installed package is '%s' but should be '%s' (from filtered view)",
				installedPackage.Name, expectedPackage)
		}

		if installedPackage.Name != "ccusage" {
			t.Errorf("BUG CONFIRMED: Expected 'ccusage' to be installed, got '%s'", installedPackage.Name)
		}
	}
}

// TestConcreteCase_FilterCli_Navigate tests navigation through multiple filtered items
//
// **Validates: Requirements 2.1, 2.2**
//
// CRITICAL: This test MUST FAIL on unfixed code
//
// Concrete test case:
// 1. Apply filter "/cli" - should show "Qwen Code CLI" and "Claude Code CLI"
// 2. Navigate through visible items
// 3. Verify selection stays within filtered items
func TestConcreteCase_FilterCli_Navigate(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Start filter mode
	filterStartMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterStartMsg)
	model = updatedModel.(Model)

	// Type "cli"
	for _, char := range "cli" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(Model)
	}

	// Apply filter with Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify filter is applied (custom filter)
	if model.filter.filteredPackages == nil {
		t.Fatal("Filter should be applied")
	}

	visibleItems := model.list.Items()
	t.Logf("Visible items after filtering 'cli': %d", len(visibleItems))

	// Should have 2 visible items: "Qwen Code CLI" and "Claude Code CLI"
	if len(visibleItems) != 2 {
		t.Logf("Expected 2 visible items, got %d", len(visibleItems))
	}

	// Navigate down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(Model)

	// Check selected package
	selectedItem := model.list.SelectedItem()
	if selectedItem == nil {
		t.Fatal("No item selected after navigation")
	}

	pkgItem, ok := selectedItem.(packageItem)
	if !ok {
		t.Fatal("Failed to cast selected item to packageItem")
	}

	expectedPackage := pkgItem.pkg.Name

	if model.selectedPackage == nil {
		t.Fatal("selectedPackage is nil")
	}

	t.Logf("After filtering 'cli' and pressing down:")
	t.Logf("  Expected (from filtered view): %s", expectedPackage)
	t.Logf("  Actual (model.selectedPackage): %s", model.selectedPackage.Name)

	if model.selectedPackage.Name != expectedPackage {
		t.Errorf("BUG CONFIRMED: After filtering 'cli' and navigating, selected package is '%s' but should be '%s'",
			model.selectedPackage.Name, expectedPackage)
	}

	// Verify selected package is one of the CLI packages
	if model.selectedPackage.Name != "Qwen Code CLI" && model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("BUG CONFIRMED: Selected package '%s' is not one of the filtered CLI packages", model.selectedPackage.Name)
	}

	// Verify selected package is NOT "ccusage" (which should be filtered out)
	if model.selectedPackage.Name == "ccusage" {
		t.Errorf("BUG CONFIRMED: Selected package is 'ccusage' which should be filtered out", )
	}
}

// TestConcreteCase_FilterNoMatch_Navigate tests navigation with no matching items
//
// **Validates: Requirements 2.1, 2.2**
//
// Edge case: Filter that matches nothing
// 1. Apply filter "/xyz" - should show "No items found"
// 2. Press down arrow
// 3. Verify no crash and no selection of invisible items
func TestConcreteCase_FilterNoMatch_Navigate(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Start filter mode
	filterStartMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(filterStartMsg)
	model = updatedModel.(Model)

	// Type "xyz" (no matches)
	for _, char := range "xyz" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(Model)
	}

	// Apply filter with Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify filter is applied (custom filter)
	if model.filter.filteredPackages == nil && len(model.filter.filteredPackages) != 0 {
		// For no-match case, filteredPackages will be an empty slice (not nil)
	}

	visibleItems := model.list.Items()
	t.Logf("Visible items after filtering 'xyz': %d", len(visibleItems))

	// Should have 0 visible items
	if len(visibleItems) != 0 {
		t.Logf("Expected 0 visible items, got %d", len(visibleItems))
	}

	// Press down arrow (should not crash)
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(Model)

	// Verify no crash occurred (test passes if we reach here)
	t.Log("No crash occurred when navigating with empty filtered list")

	// Verify selected item is nil or unchanged
	selectedItem := model.list.SelectedItem()
	if selectedItem != nil {
		pkgItem, ok := selectedItem.(packageItem)
		if ok {
			t.Logf("Selected item after navigation: %s", pkgItem.pkg.Name)
			
			// If there's a selection, it should not be from the unfiltered list
			// when no items are visible
			if len(visibleItems) == 0 {
				t.Errorf("BUG: Selected item '%s' exists when no items are visible", pkgItem.pkg.Name)
			}
		}
	}
}

// genFilterString generates filter strings for property-based testing
func genFilterString() gopter.Gen {
	return gen.OneGenOf(
		gen.Const("age"),    // Matches "ccusage"
		gen.Const("cli"),    // Matches "Qwen Code CLI" and "Claude Code CLI"
		gen.Const("qwen"),   // Matches "Qwen Code CLI"
		gen.Const("claude"), // Matches "Claude Code CLI"
		gen.Const("code"),   // Matches "Qwen Code CLI" and "Claude Code CLI"
		gen.Const("usage"),  // Matches "ccusage"
		gen.Const("xyz"),    // Matches nothing
		gen.Const(""),       // Empty filter (matches all)
	)
}

// genNavigationKey generates navigation keys for property-based testing
func genNavigationKey() gopter.Gen {
	return gen.OneGenOf(
		gen.Const("up"),
		gen.Const("down"),
		gen.Const("j"),
		gen.Const("k"),
		gen.Const("enter"),
	)
}

// ============================================================================
// PRESERVATION PROPERTY TESTS (Task 2)
// ============================================================================
// These tests verify that existing functionality WITHOUT filters continues
// to work correctly. They establish a baseline that must be preserved after
// the fix is implemented.
//
// **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6**
//
// IMPORTANT: These tests should PASS on UNFIXED code
// ============================================================================

// TestProperty_UnfilteredNavigationPreservation verifies Property 2: Preservation - Unfiltered Navigation and Selection
//
// **Validates: Requirements 3.1, 3.2**
//
// Universal Quantification:
// ∀ navigationKey ∈ {up, down, j, k}:
//   (FilterState = Unfiltered) ⟹
//     selectedPackage = packages[list.Index()]
//     ∧ navigation works correctly
//     ∧ behavior is unchanged from original
//
// This property test verifies that:
// 1. Navigation without filters works correctly
// 2. Selected package is correctly updated based on cursor position
// 3. The system uses packages[list.Index()] when no filter is applied (current behavior)
func TestProperty_UnfilteredNavigationPreservation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	properties.Property("Unfiltered navigation preserves existing behavior", prop.ForAll(
		func(navKey string, startIndex int) bool {
			// Create model with test packages
			service := &mockPackageService{}
			model := NewModel(packages, service)
			model.width = 100
			model.height = 30
			model.list.SetSize(50, 26)

			// Set starting position
			validStartIndex := startIndex % len(packages)
			model.list.Select(validStartIndex)
			model.selectedPackage = &packages[validStartIndex]

			// Verify no filter is applied
			if model.filter.active || model.filter.filteredPackages != nil {
				return true // Skip if filter is somehow active
			}

			// Press the navigation key
			var keyMsg tea.KeyMsg
			switch navKey {
			case "up":
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			case "down":
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			case "j":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
			case "k":
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
			default:
				return true // Skip invalid key
			}

			updatedModel, _ := model.Update(keyMsg)
			model = updatedModel.(Model)

			// PRESERVATION CHECK: Verify selected package matches packages[list.Index()]
			// This is the CURRENT behavior that must be preserved for unfiltered navigation
			if model.selectedPackage == nil {
				t.Logf("selectedPackage is nil after navigation")
				return false
			}

			expectedPackage := packages[model.list.Index()]
			if model.selectedPackage.Name != expectedPackage.Name {
				t.Logf("PRESERVATION VIOLATION: Without filter, selected package is '%s' but packages[%d] is '%s'",
					model.selectedPackage.Name, model.list.Index(), expectedPackage.Name)
				return false
			}

			return true
		},
		genNavigationKey(),
		gen.IntRange(0, 100), // Start index
	))

	properties.TestingRun(t)
}

// TestConcrete_UnfilteredNavigation_UpDown tests concrete navigation without filter
//
// **Validates: Requirements 3.1**
//
// Concrete test case:
// 1. Start at first package
// 2. Press down arrow - should select second package
// 3. Press down arrow again - should select third package
// 4. Press up arrow - should select second package
// 5. Verify all selections are correct
func TestConcrete_UnfilteredNavigation_UpDown(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Initial state: first package selected
	if model.selectedPackage == nil || model.selectedPackage.Name != "Qwen Code CLI" {
		t.Errorf("Expected 'Qwen Code CLI' to be initially selected, got '%v'", model.selectedPackage)
	}

	// Press down arrow
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' after down, got '%v'", model.selectedPackage)
	}

	// Press down arrow again
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "ccusage" {
		t.Errorf("Expected 'ccusage' after second down, got '%v'", model.selectedPackage)
	}

	// Press up arrow
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, _ = model.Update(upMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' after up, got '%v'", model.selectedPackage)
	}

	t.Log("PRESERVATION CONFIRMED: Unfiltered navigation works correctly")
}

// TestConcrete_UnfilteredNavigation_VimKeys tests vim-style navigation without filter
//
// **Validates: Requirements 3.1**
//
// Concrete test case:
// 1. Start at first package
// 2. Press 'j' (down) - should select second package
// 3. Press 'j' again - should select third package
// 4. Press 'k' (up) - should select second package
// 5. Verify all selections are correct
func TestConcrete_UnfilteredNavigation_VimKeys(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Press 'j' (down)
	jMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updatedModel, _ := model.Update(jMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' after 'j', got '%v'", model.selectedPackage)
	}

	// Press 'j' again
	updatedModel, _ = model.Update(jMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "ccusage" {
		t.Errorf("Expected 'ccusage' after second 'j', got '%v'", model.selectedPackage)
	}

	// Press 'k' (up)
	kMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	updatedModel, _ = model.Update(kMsg)
	model = updatedModel.(Model)

	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' after 'k', got '%v'", model.selectedPackage)
	}

	t.Log("PRESERVATION CONFIRMED: Vim-style navigation works correctly")
}

// TestConcrete_UnfilteredInstallation tests installation without filter
//
// **Validates: Requirements 3.2**
//
// Concrete test case:
// 1. Navigate to second package without filter
// 2. Press Enter to install
// 3. Verify correct package is selected for installation
func TestConcrete_UnfilteredInstallation(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageServiceWithFuncs{
		getPackageInfoFunc: func(pkg config.Package) services.PackageInfo {
			return services.PackageInfo{
				Package:          pkg,
				LatestVersion:    "2.0.0",
				InstalledVersion: "1.0.0",
				Status:           services.StatusUpdateAvailable,
			}
		},
		installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
			logChan := make(chan string, 1)
			errChan := make(chan error, 1)
			close(logChan)
			errChan <- nil
			return logChan, errChan
		},
	}

	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Navigate to second package
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	model = updatedModel.(Model)

	// Set package info to allow installation
	model.pkgInfo.current = &services.PackageInfo{
		Package:          *model.selectedPackage,
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}

	// Verify we're at the second package
	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Fatalf("Expected 'Claude Code CLI' to be selected, got '%v'", model.selectedPackage)
	}

	// Press Enter to install
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify state transitioned to installing
	if model.state != StateInstalling {
		t.Errorf("Expected state to be StateInstalling, got %v", model.state)
	}

	// Verify correct package was selected for installation
	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' to be installed, got '%v'", model.selectedPackage)
	}

	t.Log("PRESERVATION CONFIRMED: Unfiltered installation works correctly")
}

// TestConcrete_UnfilteredRefresh tests refresh functionality without filter
//
// **Validates: Requirements 3.3**
//
// Concrete test case:
// 1. Navigate to a package without filter
// 2. Press 'r' to refresh
// 3. Verify correct package info is refreshed
func TestConcrete_UnfilteredRefresh(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	var refreshedPackage *config.Package
	service := &mockPackageServiceWithFuncs{
		getPackageInfoFunc: func(pkg config.Package) services.PackageInfo {
			refreshedPackage = &pkg
			return services.PackageInfo{
				Package:          pkg,
				LatestVersion:    "2.0.0",
				InstalledVersion: "1.0.0",
				Status:           services.StatusUpdateAvailable,
			}
		},
	}

	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Navigate to third package
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	model = updatedModel.(Model)
	updatedModel, _ = model.Update(downMsg)
	model = updatedModel.(Model)

	// Verify we're at the third package
	if model.selectedPackage == nil || model.selectedPackage.Name != "ccusage" {
		t.Fatalf("Expected 'ccusage' to be selected, got '%v'", model.selectedPackage)
	}

	// Press 'r' to refresh
	rMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, cmd := model.Update(rMsg)
	model = updatedModel.(Model)

	// Verify command was returned
	if cmd == nil {
		t.Fatal("Expected command to fetch package info")
	}

	// Execute the command to trigger the refresh
	msg := cmd()
	if _, ok := msg.(packageInfoMsg); !ok {
		t.Errorf("Expected packageInfoMsg, got %T", msg)
	}

	// Verify correct package was refreshed
	if refreshedPackage == nil || refreshedPackage.Name != "ccusage" {
		t.Errorf("Expected 'ccusage' to be refreshed, got '%v'", refreshedPackage)
	}

	t.Log("PRESERVATION CONFIRMED: Unfiltered refresh works correctly")
}

// TestConcrete_UnfilteredForceInstall tests force install without filter
//
// **Validates: Requirements 3.4**
//
// Concrete test case:
// 1. Navigate to a package without filter
// 2. Press 'f' to set force install flag
// 3. Press Enter to force install
// 4. Verify correct package is force installed
func TestConcrete_UnfilteredForceInstall(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
		{
			Name:                "ccusage",
			InstallCommand:      "npm install -g ccusage",
			VersionCheckCommand: "npm view ccusage version",
			LocalVersionCommand: "ccusage --version",
		},
	}

	service := &mockPackageServiceWithFuncs{
		getPackageInfoFunc: func(pkg config.Package) services.PackageInfo {
			return services.PackageInfo{
				Package:          pkg,
				LatestVersion:    "1.0.0",
				InstalledVersion: "1.0.0",
				Status:           services.StatusInstalled, // Already at latest
			}
		},
		installPackageFunc: func(pkg config.Package) (<-chan string, <-chan error) {
			logChan := make(chan string, 1)
			errChan := make(chan error, 1)
			close(logChan)
			errChan <- nil
			return logChan, errChan
		},
	}

	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Navigate to second package
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(downMsg)
	model = updatedModel.(Model)

	// Set package info (already at latest version)
	model.pkgInfo.current = &services.PackageInfo{
		Package:          *model.selectedPackage,
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}

	// Verify we're at the second package
	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Fatalf("Expected 'Claude Code CLI' to be selected, got '%v'", model.selectedPackage)
	}

	// Press 'f' to set force install flag
	fMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updatedModel, _ = model.Update(fMsg)
	model = updatedModel.(Model)

	// Verify force install flag is set
	if !model.install.forceInstall {
		t.Fatal("Expected forceInstall flag to be set")
	}

	// Press Enter to force install
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify state transitioned to installing (force install bypassed version check)
	if model.state != StateInstalling {
		t.Errorf("Expected state to be StateInstalling, got %v", model.state)
	}

	// Verify correct package was selected for installation
	if model.selectedPackage == nil || model.selectedPackage.Name != "Claude Code CLI" {
		t.Errorf("Expected 'Claude Code CLI' to be force installed, got '%v'", model.selectedPackage)
	}

	t.Log("PRESERVATION CONFIRMED: Unfiltered force install works correctly")
}

// TestConcrete_UnfilteredQuit tests quit functionality without filter
//
// **Validates: Requirements 3.5**
//
// Concrete test case:
// 1. Press 'q' to quit - should work
// 2. Press Ctrl+C to quit - should work
func TestConcrete_UnfilteredQuit(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied
	if model.filter.active || model.filter.filteredPackages != nil {
		t.Fatal("Expected no filter to be applied")
	}

	// Test 'q' key
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(qMsg)

	if cmd == nil {
		t.Fatal("Expected quit command for 'q' key")
	}

	if cmd() != tea.Quit() {
		t.Error("Expected tea.Quit command for 'q' key")
	}

	// Test Ctrl+C
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(ctrlCMsg)

	if cmd == nil {
		t.Fatal("Expected quit command for Ctrl+C")
	}

	if cmd() != tea.Quit() {
		t.Error("Expected tea.Quit command for Ctrl+C")
	}

	t.Log("PRESERVATION CONFIRMED: Unfiltered quit works correctly")
}

// TestConcrete_FilterModeEntry tests filter mode entry and exit
//
// **Validates: Requirements 3.6**
//
// Concrete test case:
// 1. Press '/' to enter filter mode - should be handled by list component
// 2. Press Esc to exit filter mode - should be handled by list component
// 3. Verify filter state transitions correctly
func TestConcrete_FilterModeEntry(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
		},
		{
			Name:                "Claude Code CLI",
			InstallCommand:      "npm install -g @claude-code/claude-code",
			VersionCheckCommand: "npm view @claude-code/claude-code version",
			LocalVersionCommand: "claude-code --version",
		},
	}

	service := &mockPackageService{}
	model := NewModel(packages, service)
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)

	// Verify no filter is applied initially
	if model.filter.active {
		t.Fatal("Expected no filter to be active initially")
	}

	// Press '/' to enter filter mode
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(slashMsg)
	model = updatedModel.(Model)

	// Verify filter mode is active
	if !model.filter.active {
		t.Errorf("Expected filter.active to be true after pressing '/'")
	}

	// Press Esc to exit filter mode
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(Model)

	// Verify filter mode is exited
	if model.filter.active {
		t.Error("Expected filter.active to be false after Esc")
	}

	t.Log("PRESERVATION CONFIRMED: Filter mode entry and exit work correctly")
}
