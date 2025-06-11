package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/TFMV/scope/internal/analyzer"
	"github.com/TFMV/scope/internal/cache"
	"github.com/TFMV/scope/internal/tools"
	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

var (
	analyzerInstance *analyzer.Analyzer
	cacheInstance    *cache.Cache
	toolManager      *tools.ToolManager
)

// TypeInfo represents the extracted type information
type TypeInfo struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Package string   `json:"package"`
	Doc     string   `json:"doc"`
	Methods []string `json:"methods,omitempty"`
}

func main() {
	// Initialize logging to write to stderr
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags)

	// Initialize the cache
	cacheDir := filepath.Join(os.TempDir(), "scope")
	var err error
	cacheInstance, err = cache.New(cacheDir)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Initialize the analyzer
	repoPath := os.Getenv("GO_REPO_PATH")
	if repoPath == "" {
		log.Fatal("GO_REPO_PATH environment variable not set")
	}

	analyzerInstance, err = analyzer.NewAnalyzer(repoPath)
	if err != nil {
		log.Fatalf("Failed to initialize analyzer: %v", err)
	}

	// Initialize tool manager
	toolManager = tools.NewToolManager()
	log.Printf("Tool manager initialized")

	// Get the directory of the executable
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	execDir := filepath.Dir(execPath)
	log.Printf("Looking for config files in: %s", execDir)

	// Load tool configurations
	toolsConfig, err := tools.LoadToolsConfig(execDir)
	if err != nil {
		log.Fatalf("Failed to load tools configuration: %v", err)
	}
	log.Printf("Loaded tools configuration with %d tools", len(toolsConfig.Tools))

	// Register all tools from config
	for _, toolConfig := range toolsConfig.Tools {
		log.Printf("Attempting to register tool: %s", toolConfig.Name)
		toolManager.RegisterTool(toolConfig)
		log.Printf("Registered tool: %s", toolConfig.Name)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the MCP server with HTTP transport
	server := mcp.NewServer(stdio.NewStdioServerTransport())

	log.Println("Scope server initialized...")

	log.Println("Registering tools...")

	if err := registerTools(server); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	log.Println("Starting server...")

	// Start server in a goroutine
	go func() {
		if err := server.Serve(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down Scope server...")
}

func registerTools(server *mcp.Server) error {
	// Register lookup_type tool
	if err := server.RegisterTool("lookup_type", "Get documentation and definition of a Go type", lookupTypeHandler); err != nil {
		return fmt.Errorf("failed to register lookup_type tool: %w", err)
	}
	log.Printf("Registered lookup_type tool")

	// Register list_methods tool
	if err := server.RegisterTool("list_methods", "List public methods for a Go type", listMethodsHandler); err != nil {
		return fmt.Errorf("failed to register list_methods tool: %w", err)
	}
	log.Printf("Registered list_methods tool")

	// Register show_example tool
	if err := server.RegisterTool("show_example", "Return a code example for a Go type or topic", showExampleHandler); err != nil {
		return fmt.Errorf("failed to register show_example tool: %w", err)
	}
	log.Printf("Registered show_example tool")

	// Register code_search tool
	if err := server.RegisterTool("code_search", "Search through codebase using semantic search", codeSearchHandler); err != nil {
		return fmt.Errorf("failed to register code_search tool: %w", err)
	}
	log.Printf("Registered code_search tool")

	// Register code_edit tool
	if err := server.RegisterTool("code_edit", "Edit code files with AI assistance", codeEditHandler); err != nil {
		return fmt.Errorf("failed to register code_edit tool: %w", err)
	}
	log.Printf("Registered code_edit tool")

	// Register code_review tool
	if err := server.RegisterTool("code_review", "Review code changes and provide feedback", codeReviewHandler); err != nil {
		return fmt.Errorf("failed to register code_review tool: %w", err)
	}
	log.Printf("Registered code_review tool")

	log.Printf("Successfully registered %d tools", 6)
	return nil
}

type LookupTypeArgs struct {
	TypeName string `json:"type_name" jsonschema:"required,description=The name of the Go type"`
}

func lookupTypeHandler(args LookupTypeArgs) (*mcp.ToolResponse, error) {
	log.Printf("Looking up type: %s", args.TypeName)
	// Check cache first
	if cached, found := cacheInstance.Get(fmt.Sprintf("type:%s", args.TypeName)); found {
		if typeInfo, ok := cached.(*analyzer.TypeInfo); ok {
			jsonData, err := json.Marshal(typeInfo)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal type info: %w", err)
			}
			return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
		}
	}

	// Not in cache, look it up
	typeInfo, err := analyzerInstance.LookupType(args.TypeName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := cacheInstance.Set(fmt.Sprintf("type:%s", args.TypeName), typeInfo, 24*time.Hour); err != nil {
		log.Printf("Warning: failed to cache type info: %v", err)
	}

	jsonData, err := json.Marshal(typeInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal type info: %w", err)
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
}

type ListMethodsArgs struct {
	TypeName string `json:"type_name" jsonschema:"required,description=Name of the type"`
}

func listMethodsHandler(args ListMethodsArgs) (*mcp.ToolResponse, error) {
	log.Printf("Listing methods for type: %s", args.TypeName)
	// Check cache first
	if cached, found := cacheInstance.Get(fmt.Sprintf("methods:%s", args.TypeName)); found {
		if methods, ok := cached.([]string); ok {
			jsonData, err := json.Marshal(methods)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal methods: %w", err)
			}
			return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
		}
	}

	// Not in cache, look it up
	methods, err := analyzerInstance.ListMethods(args.TypeName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := cacheInstance.Set(fmt.Sprintf("methods:%s", args.TypeName), methods, 24*time.Hour); err != nil {
		log.Printf("Warning: failed to cache methods: %v", err)
	}

	jsonData, err := json.Marshal(methods)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal methods: %w", err)
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonData))), nil
}

type ShowExampleArgs struct {
	Topic string `json:"topic" jsonschema:"required,description=What to show an example for"`
}

func showExampleHandler(args ShowExampleArgs) (*mcp.ToolResponse, error) {
	log.Printf("Showing example for topic: %s", args.Topic)
	// Check cache first
	if cached, found := cacheInstance.Get(fmt.Sprintf("example:%s", args.Topic)); found {
		if example, ok := cached.(string); ok {
			return mcp.NewToolResponse(mcp.NewTextContent(example)), nil
		}
	}

	// Not in cache, look it up
	example, err := analyzerInstance.GetExample(args.Topic)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := cacheInstance.Set(fmt.Sprintf("example:%s", args.Topic), example, 24*time.Hour); err != nil {
		log.Printf("Warning: failed to cache example: %v", err)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(example)), nil
}

type CodeSearchArgs struct {
	Query string `json:"query" jsonschema:"required,description=The search query"`
}

func codeSearchHandler(args CodeSearchArgs) (*mcp.ToolResponse, error) {
	log.Printf("Executing code search: %s", args.Query)
	tool, ok := toolManager.GetTool("code_search")
	if !ok {
		return nil, fmt.Errorf("code_search tool not found")
	}

	output, err := tool.Execute(context.Background(), args.Query)
	if err != nil {
		return nil, fmt.Errorf("code search failed: %w", err)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(output)), nil
}

type CodeEditArgs struct {
	File    string `json:"file" jsonschema:"required,description=The file to edit"`
	Changes string `json:"changes" jsonschema:"required,description=The changes to apply"`
}

func codeEditHandler(args CodeEditArgs) (*mcp.ToolResponse, error) {
	log.Printf("Executing code edit for file: %s", args.File)
	tool, ok := toolManager.GetTool("code_edit")
	if !ok {
		return nil, fmt.Errorf("code_edit tool not found")
	}

	input := fmt.Sprintf("%s\n%s", args.File, args.Changes)
	output, err := tool.Execute(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("code edit failed: %w", err)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(output)), nil
}

type CodeReviewArgs struct {
	Changes string `json:"changes" jsonschema:"required,description=The code changes to review"`
}

func codeReviewHandler(args CodeReviewArgs) (*mcp.ToolResponse, error) {
	log.Printf("Executing code review")
	tool, ok := toolManager.GetTool("code_review")
	if !ok {
		return nil, fmt.Errorf("code_review tool not found")
	}

	output, err := tool.Execute(context.Background(), args.Changes)
	if err != nil {
		return nil, fmt.Errorf("code review failed: %w", err)
	}

	return mcp.NewToolResponse(mcp.NewTextContent(output)), nil
}
