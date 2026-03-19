package ui

import (
	"testing"

	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/internal/testutil"
	"github.com/Sinton/my-pnpm-installer/services"
)

func TestRenderPackageDetails(t *testing.T) {
	// Create a model with test data
	model := createTestModelWithPackages()

	// Create test package info
	info := services.PackageInfo{
		Package: config.Package{
			Name:           "Test Package",
			InstallCommand: "npm install -g test-pkg",
		},
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}

	// Render package details
	view := model.renderPackageDetails(info)

	// Verify output contains expected content
	if !testutil.Contains(view, "Test Package") {
		t.Error("Expected view to contain package name")
	}
	if !testutil.Contains(view, "2.0.0") {
		t.Error("Expected view to contain latest version")
	}
	if !testutil.Contains(view, "1.0.0") {
		t.Error("Expected view to contain installed version")
	}
}

func TestRenderPackageDetailsNotInstalled(t *testing.T) {
	model := createTestModelWithPackages()

	info := services.PackageInfo{
		Package: config.Package{
			Name:           "Test Package",
			InstallCommand: "npm install -g test-pkg",
		},
		LatestVersion:    "2.0.0",
		InstalledVersion: "",
		Status:           services.StatusNotInstalled,
	}

	view := model.renderPackageDetails(info)

	if !testutil.Contains(view, "Test Package") {
		t.Error("Expected view to contain package name")
	}
}

func TestRenderPackageDetailsWithError(t *testing.T) {
	model := createTestModelWithPackages()

	info := services.PackageInfo{
		Package: config.Package{
			Name:           "Test Package",
			InstallCommand: "npm install -g test-pkg",
		},
		LatestVersion:    "",
		InstalledVersion: "",
		Status:           services.StatusError,
		Error:            testutil.NewMockError("test error"),
	}

	view := model.renderPackageDetails(info)

	if !testutil.Contains(view, "Test Package") {
		t.Error("Expected view to contain package name")
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Simple package with @latest",
			command:  "npm install -g typescript@latest",
			expected: "typescript",
		},
		{
			name:     "Scoped package with @latest",
			command:  "pnpm install -g @qwen-code/qwen-code@latest",
			expected: "@qwen-code/qwen-code",
		},
		{
			name:     "Package without version",
			command:  "npm install -g typescript",
			expected: "typescript",
		},
		{
			name:     "Scoped package without version",
			command:  "pnpm install -g @angular/cli",
			expected: "@angular/cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackageName(tt.command)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestPackageItemFilterValue(t *testing.T) {
	pkg := config.Package{
		Name:           "TypeScript",
		InstallCommand: "npm install -g typescript@latest",
	}

	item := packageItem{pkg: pkg}
	filterValue := item.FilterValue()

	// Should contain both display name and package name
	if !testutil.Contains(filterValue, "TypeScript") {
		t.Error("Expected filter value to contain display name")
	}
	if !testutil.Contains(filterValue, "typescript") {
		t.Error("Expected filter value to contain package name")
	}
}

func TestPackageItemTitle(t *testing.T) {
	pkg := config.Package{
		Name:           "Test Package",
		InstallCommand: "npm install -g test-pkg",
	}

	item := packageItem{pkg: pkg}
	title := item.Title()

	if title != "Test Package" {
		t.Errorf("Expected title to be 'Test Package', got '%s'", title)
	}
}

func TestPackageItemDescription(t *testing.T) {
	pkg := config.Package{
		Name:           "Test Package",
		InstallCommand: "npm install -g test-pkg@latest",
	}

	item := packageItem{pkg: pkg}
	description := item.Description()

	if description != "test-pkg" {
		t.Errorf("Expected description to be 'test-pkg', got '%s'", description)
	}
}

func TestApplyFilter(t *testing.T) {
	model := createTestModelWithPackages()

	// Apply filter
	model.filter.input = "test"
	model.applyFilter("test")

	// Verify filter was applied
	if model.filter.input != "test" {
		t.Error("Expected filter input to be 'test'")
	}
}

func TestApplyFilterEmpty(t *testing.T) {
	model := createTestModelWithPackages()

	// Apply empty filter
	model.applyFilter("")

	// Verify all packages are shown
	if model.filter.filteredPackages != nil {
		t.Error("Expected filteredPackages to be nil for empty filter")
	}
}

func TestApplyFilterNoMatch(t *testing.T) {
	model := createTestModelWithPackages()

	// Apply filter with no matches
	model.applyFilter("nonexistent123")

	// Verify no packages match
	if len(model.filter.filteredPackages) != 0 {
		t.Error("Expected no packages to match nonexistent filter")
	}
	if model.selectedPackage != nil {
		t.Error("Expected selectedPackage to be nil when no matches")
	}
}

// Helper function to create a test model with custom packages
func createTestModelWithPackages(packages ...config.Package) Model {
	if len(packages) == 0 {
		packages = []config.Package{
			{
				Name:                "Test Package",
				InstallCommand:      "npm install -g test-pkg@latest",
				VersionCheckCommand: "npm view test-pkg version",
				LocalVersionCommand: "test-pkg --version",
			},
			{
				Name:                "TypeScript",
				InstallCommand:      "npm install -g typescript@latest",
				VersionCheckCommand: "npm view typescript version",
				LocalVersionCommand: "tsc --version",
			},
		}
	}

	service := &mockPackageServiceWithFuncs{}
	return NewModel(packages, service)
}
