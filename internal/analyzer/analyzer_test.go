package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzer(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "analyzer-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Go package for testing
	testPkg := filepath.Join(tmpDir, "testpkg")
	if err := os.Mkdir(testPkg, 0755); err != nil {
		t.Fatalf("Failed to create test package dir: %v", err)
	}

	// Create a test file with a simple struct and interface
	testFile := filepath.Join(testPkg, "test.go")
	testContent := `package testpkg

// TestStruct is a test struct
type TestStruct struct {
	Field1 string
	Field2 int
}

// TestInterface is a test interface
type TestInterface interface {
	Method1() string
	Method2() int
}

// Method1 implements TestInterface
func (t *TestStruct) Method1() string {
	return t.Field1
}

// Method2 implements TestInterface
func (t *TestStruct) Method2() int {
	return t.Field2
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create analyzer
	analyzer, err := NewAnalyzer(testPkg)
	if err != nil {
		t.Fatalf("Failed to create analyzer: %v", err)
	}

	// Test LookupType
	t.Run("LookupType", func(t *testing.T) {
		info, err := analyzer.LookupType("TestStruct")
		if err != nil {
			t.Fatalf("LookupType failed: %v", err)
		}
		if info.Name != "TestStruct" {
			t.Errorf("Expected name TestStruct, got %s", info.Name)
		}
		if info.Kind != "struct" {
			t.Errorf("Expected kind struct, got %s", info.Kind)
		}
		if info.Package != "testpkg" {
			t.Errorf("Expected package testpkg, got %s", info.Package)
		}
	})

	// Test ListMethods
	t.Run("ListMethods", func(t *testing.T) {
		methods, err := analyzer.ListMethods("TestStruct")
		if err != nil {
			t.Fatalf("ListMethods failed: %v", err)
		}
		if len(methods) != 2 {
			t.Errorf("Expected 2 methods, got %d", len(methods))
		}
		expectedMethods := map[string]bool{
			"Method1": true,
			"Method2": true,
		}
		for _, method := range methods {
			if !expectedMethods[method.Name] {
				t.Errorf("Unexpected method: %s", method.Name)
			}
		}
	})

	// Test GetExample
	t.Run("GetExample", func(t *testing.T) {
		example, err := analyzer.GetExample("TestStruct")
		if err != nil {
			t.Fatalf("GetExample failed: %v", err)
		}
		if example == "" {
			t.Error("Expected non-empty example")
		}
	})
}
