package tools

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Tool represents a single tool that can be executed
type Tool struct {
	config ToolConfig
	mu     sync.Mutex
}

// NewTool creates a new tool instance
func NewTool(config ToolConfig) *Tool {
	return &Tool{
		config: config,
	}
}

// Execute runs the tool with the given input
func (t *Tool) Execute(ctx context.Context, input string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Set timeout if specified
	if t.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(t.config.Timeout)*time.Second)
		defer cancel()
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, t.config.Command, t.config.Args...)

	// Set environment variables
	for k, v := range t.config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %v", err)
	}

	return string(output), nil
}

// GetName returns the tool's name
func (t *Tool) GetName() string {
	return t.config.Name
}

// GetDescription returns the tool's description
func (t *Tool) GetDescription() string {
	return t.config.Description
}

// ToolManager manages all available tools
type ToolManager struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewToolManager creates a new tool manager
func NewToolManager() *ToolManager {
	return &ToolManager{
		tools: make(map[string]*Tool),
	}
}

// RegisterTool registers a new tool
func (tm *ToolManager) RegisterTool(config ToolConfig) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tools[config.Name] = NewTool(config)
}

// GetTool returns a tool by name
func (tm *ToolManager) GetTool(name string) (*Tool, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tool, ok := tm.tools[name]
	return tool, ok
}

// ListTools returns a list of all registered tools
func (tm *ToolManager) ListTools() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tools := make([]string, 0, len(tm.tools))
	for name := range tm.tools {
		tools = append(tools, name)
	}
	return tools
}
