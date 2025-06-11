package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadToolsConfig(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "featherhead-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test loading non-existent config (should create default)
	configPath := filepath.Join(tempDir, "tools.json")
	config, err := LoadToolsConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify default tools are created
	if len(config.Tools) == 0 {
		t.Error("Expected default tools to be created")
	}

	// Verify specific default tools
	foundTools := make(map[string]bool)
	for _, tool := range config.Tools {
		foundTools[tool.Name] = true
	}

	expectedTools := []string{"code_search", "code_edit", "code_review"}
	for _, name := range expectedTools {
		if !foundTools[name] {
			t.Errorf("Expected tool %s not found in default config", name)
		}
	}

	// Test loading existing config
	config, err = LoadToolsConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load existing config: %v", err)
	}

	if len(config.Tools) == 0 {
		t.Error("Expected tools to be loaded from existing config")
	}
}

func TestLoadToolsConfigWithCustomPath(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "featherhead-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom config file
	customConfig := &ToolsConfig{
		Tools: []ToolConfig{
			{
				Name:        "custom_tool",
				Description: "A custom test tool",
				Command:     "echo",
				Args:        []string{"test"},
				Timeout:     30,
			},
		},
	}

	configPath := filepath.Join(tempDir, "custom-tools.json")
	data, err := json.MarshalIndent(customConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal custom config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write custom config: %v", err)
	}

	// Test loading custom config
	config, err := LoadToolsConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load custom config: %v", err)
	}

	if len(config.Tools) != 1 {
		t.Errorf("Expected 1 tool in custom config, got %d", len(config.Tools))
	}

	if config.Tools[0].Name != "custom_tool" {
		t.Errorf("Expected tool name 'custom_tool', got '%s'", config.Tools[0].Name)
	}
}

func TestLoadToolsConfigWithInvalidJSON(t *testing.T) {
	// Create a temporary directory for test config
	tempDir, err := os.MkdirTemp("", "featherhead-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create an invalid config file
	configPath := filepath.Join(tempDir, "invalid-tools.json")
	if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Test loading invalid config
	_, err = LoadToolsConfig(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid config, got nil")
	}
}

func TestLoadToolsConfigWithEmptyPath(t *testing.T) {
	// Test loading config with empty path (should use default path)
	config, err := LoadToolsConfig("")
	if err != nil {
		t.Fatalf("Failed to load config with empty path: %v", err)
	}

	if len(config.Tools) == 0 {
		t.Error("Expected default tools to be created with empty path")
	}
}
