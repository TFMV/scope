package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ToolConfig represents the configuration for a single tool
type ToolConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Timeout     int               `json:"timeout"` // in seconds
}

// ToolsConfig represents the configuration for all tools
type ToolsConfig struct {
	Tools []ToolConfig `json:"tools"`
}

// LoadToolsConfig loads the tools configuration from a JSON file
func LoadToolsConfig(configPath string) (*ToolsConfig, error) {
	// Default config path if none provided
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(homeDir, ".featherhead", "tools.json")
	} else {
		// If configPath is a directory, append tools.json
		if info, err := os.Stat(configPath); err == nil && info.IsDir() {
			configPath = filepath.Join(configPath, "tools.json")
		}
	}

	// Debug log the final path
	fmt.Printf("Loading tools config from: %s\n", configPath)

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := &ToolsConfig{
			Tools: []ToolConfig{
				{
					Name:        "code_search",
					Description: "Search through codebase using semantic search",
					Command:     "featherhead-search",
					Args:        []string{"--semantic"},
					Timeout:     30,
				},
				{
					Name:        "code_edit",
					Description: "Edit code files with AI assistance",
					Command:     "featherhead-edit",
					Args:        []string{"--ai"},
					Timeout:     60,
				},
				{
					Name:        "code_review",
					Description: "Review code changes and provide feedback",
					Command:     "featherhead-review",
					Args:        []string{"--review"},
					Timeout:     45,
				},
			},
		}

		// Create directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, err
		}

		// Write default config
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return nil, err
		}

		return config, nil
	}

	// Read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config ToolsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
