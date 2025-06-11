package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TFMV/scope/internal/analyzer"
	"github.com/TFMV/scope/internal/cache"
)

func TestMain(m *testing.M) {
	// Set up test environment
	tempDir, err := os.MkdirTemp("", "featherhead-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test package
	pkgDir := filepath.Join(tempDir, "testpkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		panic(err)
	}

	// Create a test Go file
	testFile := filepath.Join(pkgDir, "test.go")
	testContent := `package testpkg

// TestStruct is a test struct
type TestStruct struct {
	Field string
}

// TestMethod is a test method
func (t *TestStruct) TestMethod() string {
	return t.Field
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		panic(err)
	}

	// Set environment variable for tests
	os.Setenv("ARROW_GO_PATH", pkgDir)

	// Initialize analyzer and cache
	var err2 error
	analyzerInstance, err2 = analyzer.NewAnalyzer(pkgDir)
	if err2 != nil {
		panic(err2)
	}

	cacheInstance, err2 = cache.New(tempDir)
	if err2 != nil {
		panic(err2)
	}

	// Run tests
	code := m.Run()

	// Clean up
	os.Unsetenv("ARROW_GO_PATH")

	os.Exit(code)
}

func TestLookupTypeHandler(t *testing.T) {
	args := LookupTypeArgs{
		TypeName: "TestStruct",
	}

	response, err := lookupTypeHandler(args)
	if err != nil {
		t.Errorf("lookupTypeHandler failed: %v", err)
	}

	if response == nil {
		t.Error("response should not be nil")
	}
}

func TestListMethodsHandler(t *testing.T) {
	args := ListMethodsArgs{
		TypeName: "TestStruct",
	}

	response, err := listMethodsHandler(args)
	if err != nil {
		t.Errorf("listMethodsHandler failed: %v", err)
	}

	if response == nil {
		t.Error("response should not be nil")
	}
}

func TestShowExampleHandler(t *testing.T) {
	args := ShowExampleArgs{
		Topic: "TestStruct",
	}

	response, err := showExampleHandler(args)
	if err != nil {
		t.Errorf("showExampleHandler failed: %v", err)
	}

	if response == nil {
		t.Error("response should not be nil")
	}
}
