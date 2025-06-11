package tools

import (
	"context"
	"testing"
)

func TestNewTool(t *testing.T) {
	config := ToolConfig{
		Name:        "test_tool",
		Description: "A test tool",
		Command:     "echo",
		Args:        []string{"test"},
		Timeout:     5,
	}

	tool := NewTool(config)
	if tool == nil {
		t.Fatal("NewTool returned nil")
	}

	if tool.GetName() != config.Name {
		t.Errorf("Expected name %s, got %s", config.Name, tool.GetName())
	}

	if tool.GetDescription() != config.Description {
		t.Errorf("Expected description %s, got %s", config.Description, tool.GetDescription())
	}
}

func TestToolExecute(t *testing.T) {
	// Test successful execution
	config := ToolConfig{
		Name:    "echo_test",
		Command: "echo",
		Args:    []string{"hello"},
		Timeout: 5,
	}

	tool := NewTool(config)
	output, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if output != "hello\n" {
		t.Errorf("Expected output 'hello\n', got '%s'", output)
	}

	// Test timeout
	config = ToolConfig{
		Name:    "sleep_test",
		Command: "sleep",
		Args:    []string{"10"},
		Timeout: 1,
	}

	tool = NewTool(config)
	_, err = tool.Execute(context.Background(), "")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Test environment variables
	config = ToolConfig{
		Name:    "env_test",
		Command: "sh",
		Args:    []string{"-c", "echo $TEST_VAR"},
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
		Timeout: 5,
	}

	tool = NewTool(config)
	output, err = tool.Execute(context.Background(), "")
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if output != "test_value\n" {
		t.Errorf("Expected output 'test_value\n', got '%s'", output)
	}
}

func TestToolManager(t *testing.T) {
	manager := NewToolManager()

	// Test registering tools
	config := ToolConfig{
		Name:        "test_tool",
		Description: "A test tool",
		Command:     "echo",
		Args:        []string{"test"},
	}

	manager.RegisterTool(config)

	// Test getting tool
	tool, ok := manager.GetTool("test_tool")
	if !ok {
		t.Error("Tool not found after registration")
	}
	if tool.GetName() != config.Name {
		t.Errorf("Expected name %s, got %s", config.Name, tool.GetName())
	}

	// Test getting non-existent tool
	_, ok = manager.GetTool("non_existent")
	if ok {
		t.Error("Expected tool not to be found")
	}

	// Test listing tools
	tools := manager.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
	if tools[0] != config.Name {
		t.Errorf("Expected tool name %s, got %s", config.Name, tools[0])
	}
}

func TestToolConcurrentExecution(t *testing.T) {
	config := ToolConfig{
		Name:    "echo_test",
		Command: "echo",
		Args:    []string{"test"},
		Timeout: 5,
	}

	tool := NewTool(config)
	ctx := context.Background()

	// Run multiple executions concurrently
	results := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := tool.Execute(ctx, "")
			results <- err
		}()
	}

	// Check all executions completed successfully
	for i := 0; i < 10; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent execution failed: %v", err)
		}
	}
}

func TestToolWithInvalidCommand(t *testing.T) {
	config := ToolConfig{
		Name:    "invalid_test",
		Command: "non_existent_command",
		Args:    []string{"test"},
		Timeout: 5,
	}

	tool := NewTool(config)
	_, err := tool.Execute(context.Background(), "")
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}
