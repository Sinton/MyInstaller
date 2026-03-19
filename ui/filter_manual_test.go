package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sinton/my-pnpm-installer/config"
)

// TestManualFilter_ApplyFilter tests the new manual filtering implementation
func TestManualFilter_ApplyFilter(t *testing.T) {
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

	// Test 1: Enter filter mode with '/'
	slashMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := model.Update(slashMsg)
	model = updatedModel.(Model)

	if !model.filter.active {
		t.Fatal("Expected filter.active to be true after pressing '/'")
	}

	// Test 2: Type "age"
	for _, char := range "age" {
		charMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedModel, _ = model.Update(charMsg)
		model = updatedModel.(Model)
	}

	if model.filter.input != "age" {
		t.Errorf("Expected filter.input to be 'age', got '%s'", model.filter.input)
	}

	// Test 3: Apply filter with Enter
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ = model.Update(enterMsg)
	model = updatedModel.(Model)

	if model.filter.active {
		t.Error("Expected filter.active to be false after pressing Enter")
	}

	// Test 4: Verify filtered packages
	if model.filter.filteredPackages == nil {
		t.Fatal("Expected filter.filteredPackages to be set")
	}

	if len(model.filter.filteredPackages) != 1 {
		t.Errorf("Expected 1 filtered package, got %d", len(model.filter.filteredPackages))
	}

	if len(model.filter.filteredPackages) > 0 && model.filter.filteredPackages[0].Name != "ccusage" {
		t.Errorf("Expected filtered package to be 'ccusage', got '%s'", model.filter.filteredPackages[0].Name)
	}

	// Test 5: Verify list items were updated
	items := model.list.Items()
	if len(items) != 1 {
		t.Errorf("Expected 1 list item, got %d", len(items))
	}

	// Test 6: Verify selected package
	if model.selectedPackage == nil || model.selectedPackage.Name != "ccusage" {
		t.Errorf("Expected selected package to be 'ccusage', got '%v'", model.selectedPackage)
	}

	t.Log("✅ Manual filter implementation works correctly!")
	t.Logf("   Filter input: '%s'", model.filter.input)
	t.Logf("   Filtered packages: %d", len(model.filter.filteredPackages))
	t.Logf("   List items: %d", len(items))
	t.Logf("   Selected package: %s", model.selectedPackage.Name)
}

// TestManualFilter_ClearFilter tests clearing the filter
func TestManualFilter_ClearFilter(t *testing.T) {
	packages := []config.Package{
		{
			Name:                "Qwen Code CLI",
			InstallCommand:      "npm install -g @qwen-code/qwen-code",
			VersionCheckCommand: "npm view @qwen-code/qwen-code version",
			LocalVersionCommand: "qwen-code --version",
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

	// Apply filter
	model.filter.active = true
	model.filter.input = "age"
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify filter is applied
	if len(model.filter.filteredPackages) != 1 {
		t.Fatalf("Expected 1 filtered package, got %d", len(model.filter.filteredPackages))
	}

	// Clear filter with Esc
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = model.Update(escMsg)
	model = updatedModel.(Model)

	// Verify filter is cleared
	if model.filter.filteredPackages != nil {
		t.Error("Expected filter.filteredPackages to be nil after clearing filter")
	}

	items := model.list.Items()
	if len(items) != 2 {
		t.Errorf("Expected 2 list items after clearing filter, got %d", len(items))
	}

	t.Log("✅ Filter clearing works correctly!")
}

// TestManualFilter_NoMatches tests filtering with no matches
func TestManualFilter_NoMatches(t *testing.T) {
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

	// Apply filter with no matches
	model.filter.active = true
	model.filter.input = "xyz"
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	model = updatedModel.(Model)

	// Verify no packages match
	if len(model.filter.filteredPackages) != 0 {
		t.Errorf("Expected 0 filtered packages, got %d", len(model.filter.filteredPackages))
	}

	items := model.list.Items()
	if len(items) != 0 {
		t.Errorf("Expected 0 list items, got %d", len(items))
	}

	if model.selectedPackage != nil {
		t.Error("Expected selectedPackage to be nil when no matches")
	}

	t.Log("✅ No matches handling works correctly!")
}
