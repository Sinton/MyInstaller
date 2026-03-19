package config

import (
	"testing"
)

func TestLoadConfig_RealConfigFile(t *testing.T) {
	// Try to load the actual config.yaml from the project root
	cfg, err := LoadConfig("../config.yaml")
	if err != nil {
		t.Fatalf("Failed to load real config.yaml: %v", err)
	}
	
	if cfg == nil {
		t.Fatal("Config is nil")
	}
	
	if len(cfg.Packages) == 0 {
		t.Fatal("Expected at least one package in config")
	}
	
	// Verify the first package has all required fields
	pkg := cfg.Packages[0]
	if pkg.Name == "" {
		t.Error("Package name is empty")
	}
	if pkg.InstallCommand == "" {
		t.Error("Package install_command is empty")
	}
	if pkg.InstallCommand == "" {
		t.Error("Package install_command is empty")
	}
	if pkg.VersionCheckCommand == "" {
		t.Error("Package version_check_command is empty")
	}
	if pkg.LocalVersionCommand == "" {
		t.Error("Package local_version_command is empty")
	}
	
	t.Logf("Successfully loaded config with %d package(s)", len(cfg.Packages))
	t.Logf("First package: %s", pkg.Name)
}
