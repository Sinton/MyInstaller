package ui

import (
	"testing"
	
	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/Sinton/my-pnpm-installer/config"
	"github.com/Sinton/my-pnpm-installer/services"
)

// TestPackageInfoCache tests the caching mechanism
func TestPackageInfoCache(t *testing.T) {
	model := createTestModel()
	
	// First fetch - should not be cached
	if _, exists := model.pkgInfo.cache[model.packages[0].Name]; exists {
		t.Error("Expected cache to be empty initially")
	}
	
	// Simulate receiving package info
	info := services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusUpdateAvailable,
	}
	
	msg := packageInfoMsg(info)
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	
	// Verify info is cached
	if cachedInfo, exists := m.pkgInfo.cache[model.packages[0].Name]; !exists {
		t.Error("Expected package info to be cached")
	} else {
		if cachedInfo.LatestVersion != "2.0.0" {
			t.Errorf("Expected cached version '2.0.0', got '%s'", cachedInfo.LatestVersion)
		}
	}
	
	// Verify loading state is cleared
	if m.pkgInfo.isLoading {
		t.Error("Expected loading state to be cleared after receiving package info")
	}
}

// TestPackageInfoCache_FastSwitching tests that cached info is returned immediately
func TestPackageInfoCache_FastSwitching(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Pre-populate cache for first package
	info1 := services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	model.pkgInfo.cache[model.packages[0].Name] = &info1
	
	// Pre-populate cache for second package
	info2 := services.PackageInfo{
		Package:          model.packages[1],
		LatestVersion:    "2.0.0",
		InstalledVersion: "1.5.0",
		Status:           services.StatusUpdateAvailable,
	}
	model.pkgInfo.cache[model.packages[1].Name] = &info2
	
	// Switch to second package
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(Model)
	
	// Verify NO command was returned (cached info used immediately)
	if cmd != nil {
		t.Error("Expected nil command for cached info, but got a command")
	}
	
	// Verify cached info was applied immediately
	if m.pkgInfo.current == nil {
		t.Fatal("Expected pkgInfo.current to be set from cache")
	}
	if m.pkgInfo.current.LatestVersion != "2.0.0" {
		t.Errorf("Expected cached version '2.0.0', got '%s'", m.pkgInfo.current.LatestVersion)
	}

	// Verify loading state was not set (because it's cached)
	if m.pkgInfo.isLoading {
		t.Error("Expected loading state to remain false for cached info")
	}
}

// TestPackageInfoCache_ForceRefresh tests that 'r' key bypasses cache
func TestPackageInfoCache_ForceRefresh(t *testing.T) {
	callCount := 0
	
	// Create model with mock service that tracks calls
	packages := []config.Package{
		{
			Name:                "test-pkg",
			VersionCheckCommand: "test",
			LocalVersionCommand: "test",
		},
	}
	
	service := &mockPackageServiceWithFuncs{
		getPackageInfoFunc: func(pkg config.Package) services.PackageInfo {
			callCount++
			return services.PackageInfo{
				Package:          pkg,
				LatestVersion:    "1.0.0",
				InstalledVersion: "1.0.0",
				Status:           services.StatusInstalled,
			}
		},
	}
	
	model := NewModel(packages, service)
	
	// First call - should fetch from service
	cmd := model.fetchPackageInfoWithCache(packages[0], false)
	cmd() // Execute command
	
	if callCount != 1 {
		t.Errorf("Expected 1 service call, got %d", callCount)
	}
	
	// Populate cache
	info := services.PackageInfo{
		Package:          packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	model.pkgInfo.cache[packages[0].Name] = &info
	
	// Second call with cache - should not call service
	cmd = model.fetchPackageInfoWithCache(packages[0], false)
	cmd() // Execute command
	
	if callCount != 1 {
		t.Errorf("Expected still 1 service call (cached), got %d", callCount)
	}
	
	// Force refresh - should call service again
	cmd = model.fetchPackageInfoWithCache(packages[0], true)
	cmd() // Execute command
	
	if callCount != 2 {
		t.Errorf("Expected 2 service calls (force refresh), got %d", callCount)
	}
}

// TestPackageInfoCache_ClearAfterInstall tests that cache is cleared after installation
func TestPackageInfoCache_ClearAfterInstall(t *testing.T) {
	model := createTestModel()
	model.state = StateInstalling
	
	// Pre-populate cache
	info := services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	model.pkgInfo.cache[model.packages[0].Name] = &info

	// Verify cache exists
	if _, exists := model.pkgInfo.cache[model.packages[0].Name]; !exists {
		t.Fatal("Expected cache to be populated")
	}
	
	// Simulate installation completion
	msg := installCompleteMsg{err: nil}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	
	// Verify cache was cleared for the installed package
	if _, exists := m.pkgInfo.cache[model.packages[0].Name]; exists {
		t.Error("Expected cache to be cleared after installation")
	}
}

// TestLoadingState tests the loading state indicator
func TestLoadingState(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Initially not loading
	if model.pkgInfo.isLoading {
		t.Error("Expected loading state to be false initially")
	}
	
	// Navigate to package not in cache - should set loading state
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(Model)
	
	if !m.pkgInfo.isLoading {
		t.Error("Expected loading state to be true when fetching uncached package")
	}
	
	// Receive package info - should clear loading state
	info := services.PackageInfo{
		Package:          m.packages[m.list.Index()],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	
	infoMsg := packageInfoMsg(info)
	updatedModel, _ = m.Update(infoMsg)
	m = updatedModel.(Model)
	
	if m.pkgInfo.isLoading {
		t.Error("Expected loading state to be false after receiving package info")
	}
}

// TestPackageInfoCache_SwitchBackAndForth tests switching between packages
// Scenario: Select pkg1 -> switch to pkg2 -> switch back to pkg1
// Expected: pkg1 info should be cached and displayed immediately
func TestPackageInfoCache_SwitchBackAndForth(t *testing.T) {
	model := createTestModel()
	model.width = 100
	model.height = 30
	model.list.SetSize(50, 26)
	
	// Step 1: Start with first package (index 0)
	if model.list.Index() != 0 {
		t.Fatalf("Expected to start at index 0, got %d", model.list.Index())
	}
	
	// Simulate receiving info for first package
	info1 := services.PackageInfo{
		Package:          model.packages[0],
		LatestVersion:    "1.0.0",
		InstalledVersion: "1.0.0",
		Status:           services.StatusInstalled,
	}
	msg1 := packageInfoMsg(info1)
	updatedModel, _ := model.Update(msg1)
	model = updatedModel.(Model)
	
	// Verify first package is cached
	if _, exists := model.pkgInfo.cache[model.packages[0].Name]; !exists {
		t.Fatal("Expected first package to be cached")
	}
	t.Logf("�?First package '%s' cached", model.packages[0].Name)
	
	// Step 2: Switch to second package (down key)
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(downMsg)
	model = updatedModel.(Model)
	
	if model.list.Index() != 1 {
		t.Fatalf("Expected to be at index 1, got %d", model.list.Index())
	}
	
	// Execute command to fetch second package info
	if cmd != nil {
		result := cmd()
		if _, ok := result.(packageInfoMsg); ok {
			// Simulate receiving info for second package
			info2 := services.PackageInfo{
				Package:          model.packages[1],
				LatestVersion:    "2.0.0",
				InstalledVersion: "1.5.0",
				Status:           services.StatusUpdateAvailable,
			}
			msg2 := packageInfoMsg(info2)
			updatedModel, _ = model.Update(msg2)
			model = updatedModel.(Model)
		}
	}
	
	// Verify second package is cached
	if _, exists := model.pkgInfo.cache[model.packages[1].Name]; !exists {
		t.Fatal("Expected second package to be cached")
	}
	t.Logf("�?Second package '%s' cached", model.packages[1].Name)
	
	// Step 3: Switch back to first package (up key)
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd = model.Update(upMsg)
	model = updatedModel.(Model)
	
	if model.list.Index() != 0 {
		t.Fatalf("Expected to be back at index 0, got %d", model.list.Index())
	}
	
	// Verify NO command was returned (cached info used immediately)
	if cmd != nil {
		t.Error("Expected nil command for cached info, but got a command")
	}
	
	// Verify cached info was applied immediately
	if model.pkgInfo.current == nil {
		t.Fatal("Expected pkgInfo.current to be set from cache")
	}
	if model.pkgInfo.current.Package.Name != model.packages[0].Name {
		t.Errorf("Expected first package '%s', got '%s'", model.packages[0].Name, model.pkgInfo.current.Package.Name)
	}
	if model.pkgInfo.current.LatestVersion != "1.0.0" {
		t.Errorf("Expected cached version '1.0.0', got '%s'", model.pkgInfo.current.LatestVersion)
	}
	t.Logf("�?First package info returned from cache immediately")
	
	// Verify loading state was NOT set (because it's cached)
	if model.pkgInfo.isLoading {
		t.Error("Expected loading state to be false when returning to cached package")
	}
	t.Logf("�?No loading state shown for cached package")
	
	// Verify both packages are still in cache
	if len(model.pkgInfo.cache) != 2 {
		t.Errorf("Expected 2 packages in cache, got %d", len(model.pkgInfo.cache))
	}
	t.Logf("�?Cache contains %d packages", len(model.pkgInfo.cache))
}
