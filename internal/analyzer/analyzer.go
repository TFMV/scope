package analyzer

import (
	"context"
	"fmt"
	"go/ast"
	"go/doc"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Analyzer handles the analysis of Go types and methods with enterprise-grade features
type Analyzer struct {
	repoPath    string
	fset        *token.FileSet
	pkgs        map[string]*types.Package
	docPkgs     map[string]*doc.Package
	info        *types.Info
	mu          sync.RWMutex
	logger      *log.Logger
	initialized bool
	config      *Config
	files       map[string][]string // Maps package name to list of files
}

// Config holds configuration options for the analyzer
type Config struct {
	MaxConcurrency  int           // Maximum number of concurrent operations
	CacheTimeout    time.Duration // How long to cache results
	IncludeTests    bool          // Whether to include test files
	IncludeVendor   bool          // Whether to include vendor directory
	ExcludePatterns []string      // Patterns to exclude from analysis
	MaxFileSize     int64         // Maximum file size to analyze (bytes)
	AnalysisTimeout time.Duration // Timeout for analysis operations
	EnableProfiling bool          // Enable performance profiling
	LogLevel        LogLevel      // Logging level
}

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// TypeInfo represents comprehensive information about a Go type
type TypeInfo struct {
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	Package      string            `json:"package"`
	ImportPath   string            `json:"import_path"`
	Doc          string            `json:"doc"`
	Methods      []MethodInfo      `json:"methods,omitempty"`
	Fields       []FieldInfo       `json:"fields,omitempty"`
	Interfaces   []string          `json:"interfaces,omitempty"`
	Examples     []ExampleInfo     `json:"examples,omitempty"`
	Position     Position          `json:"position"`
	Exported     bool              `json:"exported"`
	Size         int64             `json:"size,omitempty"`
	Alignment    int64             `json:"alignment,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	UsedBy       []string          `json:"used_by,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// MethodInfo represents information about a method
type MethodInfo struct {
	Name       string      `json:"name"`
	Signature  string      `json:"signature"`
	Doc        string      `json:"doc"`
	Receiver   string      `json:"receiver,omitempty"`
	Parameters []ParamInfo `json:"parameters,omitempty"`
	Results    []ParamInfo `json:"results,omitempty"`
	Position   Position    `json:"position"`
	Exported   bool        `json:"exported"`
	IsPointer  bool        `json:"is_pointer"`
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Tag      string   `json:"tag,omitempty"`
	Doc      string   `json:"doc"`
	Position Position `json:"position"`
	Exported bool     `json:"exported"`
	Embedded bool     `json:"embedded"`
}

// ParamInfo represents parameter or result information
type ParamInfo struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// ExampleInfo represents code example information
type ExampleInfo struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Output string `json:"output,omitempty"`
	Doc    string `json:"doc"`
}

// Position represents source code position
type Position struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// AnalysisResult represents the result of a comprehensive analysis
type AnalysisResult struct {
	Types     []TypeInfo        `json:"types"`
	Functions []FunctionInfo    `json:"functions"`
	Variables []VariableInfo    `json:"variables"`
	Constants []ConstantInfo    `json:"constants"`
	Imports   []ImportInfo      `json:"imports"`
	Packages  []PackageInfo     `json:"packages"`
	Metrics   AnalysisMetrics   `json:"metrics"`
	Errors    []AnalysisError   `json:"errors,omitempty"`
	Warnings  []AnalysisWarning `json:"warnings,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string      `json:"name"`
	Signature  string      `json:"signature"`
	Doc        string      `json:"doc"`
	Package    string      `json:"package"`
	Parameters []ParamInfo `json:"parameters,omitempty"`
	Results    []ParamInfo `json:"results,omitempty"`
	Position   Position    `json:"position"`
	Exported   bool        `json:"exported"`
	IsMethod   bool        `json:"is_method"`
	Complexity int         `json:"complexity,omitempty"`
}

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Doc      string   `json:"doc"`
	Package  string   `json:"package"`
	Position Position `json:"position"`
	Exported bool     `json:"exported"`
	Value    string   `json:"value,omitempty"`
}

// ConstantInfo represents information about a constant
type ConstantInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Value    string   `json:"value"`
	Doc      string   `json:"doc"`
	Package  string   `json:"package"`
	Position Position `json:"position"`
	Exported bool     `json:"exported"`
}

// ImportInfo represents information about an import
type ImportInfo struct {
	Path     string   `json:"path"`
	Name     string   `json:"name,omitempty"`
	Doc      string   `json:"doc"`
	Position Position `json:"position"`
	Used     bool     `json:"used"`
}

// PackageInfo represents information about a package
type PackageInfo struct {
	Name       string   `json:"name"`
	ImportPath string   `json:"import_path"`
	Doc        string   `json:"doc"`
	Files      []string `json:"files"`
	Position   Position `json:"position"`
	IsMain     bool     `json:"is_main"`
	Size       int64    `json:"size"`
}

// AnalysisMetrics represents metrics about the analysis
type AnalysisMetrics struct {
	TotalFiles     int           `json:"total_files"`
	TotalLines     int           `json:"total_lines"`
	TotalTypes     int           `json:"total_types"`
	TotalFunctions int           `json:"total_functions"`
	TotalPackages  int           `json:"total_packages"`
	AnalysisTime   time.Duration `json:"analysis_time"`
	MemoryUsage    int64         `json:"memory_usage"`
}

// AnalysisError represents an error during analysis
type AnalysisError struct {
	Message  string   `json:"message"`
	Position Position `json:"position,omitempty"`
	Type     string   `json:"type"`
	Severity string   `json:"severity"`
}

// AnalysisWarning represents a warning during analysis
type AnalysisWarning struct {
	Message  string   `json:"message"`
	Position Position `json:"position,omitempty"`
	Type     string   `json:"type"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConcurrency:  4,
		CacheTimeout:    30 * time.Minute,
		IncludeTests:    false,
		IncludeVendor:   false,
		ExcludePatterns: []string{".git", "node_modules", "vendor"},
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		AnalysisTimeout: 5 * time.Minute,
		EnableProfiling: false,
		LogLevel:        LogLevelInfo,
	}
}

// NewAnalyzer creates a new enterprise-grade Analyzer instance
func NewAnalyzer(repoPath string) (*Analyzer, error) {
	return NewAnalyzerWithConfig(repoPath, DefaultConfig())
}

// NewAnalyzerWithConfig creates a new Analyzer with custom configuration
func NewAnalyzerWithConfig(repoPath string, config *Config) (*Analyzer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate repository path
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository path does not exist: %s", repoPath)
	}

	// Initialize logger
	logger := log.New(os.Stderr, "[ANALYZER] ", log.LstdFlags|log.Lshortfile)

	analyzer := &Analyzer{
		repoPath: repoPath,
		fset:     token.NewFileSet(),
		pkgs:     make(map[string]*types.Package),
		docPkgs:  make(map[string]*doc.Package),
		info:     &types.Info{},
		logger:   logger,
		config:   config,
		files:    make(map[string][]string),
	}

	// Initialize the analyzer
	if err := analyzer.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize analyzer: %w", err)
	}

	return analyzer, nil
}

// initialize performs the initial analysis of the repository
func (a *Analyzer) initialize() error {
	start := time.Now()
	a.logInfo("Starting repository analysis: %s", a.repoPath)

	// Parse all Go files in the repository
	if err := a.parseRepository(); err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Type check all packages
	if err := a.typeCheckPackages(); err != nil {
		return fmt.Errorf("failed to type check packages: %w", err)
	}

	// Generate documentation
	if err := a.generateDocumentation(); err != nil {
		a.logWarn("Failed to generate documentation: %v", err)
	}

	a.initialized = true
	duration := time.Since(start)
	a.logInfo("Repository analysis completed in %v", duration)

	return nil
}

// parseRepository recursively parses all Go files in the repository
func (a *Analyzer) parseRepository() error {
	return filepath.Walk(a.repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip excluded patterns
		for _, pattern := range a.config.ExcludePatterns {
			if strings.Contains(path, pattern) {
				return nil
			}
		}

		// Skip test files if not included
		if !a.config.IncludeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip large files
		if info.Size() > a.config.MaxFileSize {
			a.logWarn("Skipping large file: %s (%d bytes)", path, info.Size())
			return nil
		}

		// Parse the file
		if err := a.parseFile(path); err != nil {
			a.logWarn("Failed to parse file %s: %v", path, err)
		}

		return nil
	})
}

// parseFile parses a single Go file
func (a *Analyzer) parseFile(filename string) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Parse the file
	file, err := parser.ParseFile(a.fset, filename, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// Add to package
	pkgName := file.Name.Name
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			a.logWarn("Type checking error: %v", err)
		},
	}

	// Create type info
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}

	// Type check the file
	pkg, err := conf.Check(pkgName, a.fset, []*ast.File{file}, info)
	if err != nil {
		a.logWarn("Type checking failed for file %s: %v", filename, err)
		return err
	}

	a.pkgs[pkgName] = pkg
	a.files[pkgName] = append(a.files[pkgName], filename)

	// Merge info if this is the first package or extend as needed
	if len(a.info.Types) == 0 {
		a.info = info
	}

	return nil
}

// typeCheckPackages performs type checking on all parsed packages
func (a *Analyzer) typeCheckPackages() error {
	conf := types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			a.logWarn("Type checking error: %v", err)
		},
	}

	for pkgName, files := range a.files {
		// Convert ast.Files to slice
		var astFiles []*ast.File
		for _, file := range files {
			astFile, err := parser.ParseFile(a.fset, file, nil, parser.ParseComments)
			if err != nil {
				a.logWarn("Failed to parse file %s: %v", file, err)
				continue
			}
			astFiles = append(astFiles, astFile)
		}

		// Create type info
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
		}

		// Type check the package
		pkg, err := conf.Check(pkgName, a.fset, astFiles, info)
		if err != nil {
			a.logWarn("Type checking failed for package %s: %v", pkgName, err)
			continue
		}

		a.pkgs[pkgName] = pkg
		// Merge info if this is the first package or extend as needed
		if len(a.info.Types) == 0 {
			a.info = info
		}
	}

	return nil
}

// generateDocumentation generates documentation for all packages
func (a *Analyzer) generateDocumentation() error {
	for pkgName, pkg := range a.pkgs {
		// Create documentation using the type information
		docPkg := &doc.Package{
			Name:   pkgName,
			Types:  make([]*doc.Type, 0),
			Funcs:  make([]*doc.Func, 0),
			Vars:   make([]*doc.Value, 0),
			Consts: make([]*doc.Value, 0),
		}

		// Add types and functions from the package
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil {
				continue
			}

			switch obj := obj.(type) {
			case *types.TypeName:
				docPkg.Types = append(docPkg.Types, &doc.Type{
					Name: obj.Name(),
				})
			case *types.Func:
				docPkg.Funcs = append(docPkg.Funcs, &doc.Func{
					Name: obj.Name(),
				})
			}
		}

		a.docPkgs[pkgName] = docPkg
	}
	return nil
}

// LookupType finds and returns comprehensive information about a specific type
func (a *Analyzer) LookupType(typeName string) (*TypeInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.initialized {
		return nil, fmt.Errorf("analyzer not initialized")
	}

	for pkgName, pkg := range a.pkgs {
		obj := pkg.Scope().Lookup(typeName)
		if obj == nil {
			continue
		}

		typeInfo := &TypeInfo{
			Name:       typeName,
			Package:    pkgName,
			ImportPath: pkg.Path(),
			Exported:   obj.Exported(),
		}

		// Get position information
		if pos := a.fset.Position(obj.Pos()); pos.IsValid() {
			typeInfo.Position = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}

		// Get documentation
		if docPkg := a.docPkgs[pkgName]; docPkg != nil {
			for _, docType := range docPkg.Types {
				if docType.Name == typeName {
					typeInfo.Doc = docType.Doc
					break
				}
			}
		}

		// Analyze the type
		switch t := obj.Type().Underlying().(type) {
		case *types.Struct:
			typeInfo.Kind = "struct"
			typeInfo.Fields = a.analyzeStructFields(t, obj.Type())
		case *types.Interface:
			typeInfo.Kind = "interface"
			typeInfo.Methods = a.analyzeInterfaceMethods(t)
		case *types.Slice:
			typeInfo.Kind = "slice"
		case *types.Array:
			typeInfo.Kind = "array"
		case *types.Map:
			typeInfo.Kind = "map"
		case *types.Chan:
			typeInfo.Kind = "channel"
		case *types.Pointer:
			typeInfo.Kind = "pointer"
		case *types.Signature:
			typeInfo.Kind = "function"
		default:
			typeInfo.Kind = "other"
		}

		// Get methods
		typeInfo.Methods = a.getTypeMethods(obj.Type())

		// Get size and alignment information
		if sizes := types.SizesFor("gc", "amd64"); sizes != nil {
			typeInfo.Size = sizes.Sizeof(obj.Type())
			typeInfo.Alignment = sizes.Alignof(obj.Type())
		}

		return typeInfo, nil
	}

	return nil, fmt.Errorf("type %s not found", typeName)
}

// analyzeStructFields analyzes struct fields
func (a *Analyzer) analyzeStructFields(structType *types.Struct, namedType types.Type) []FieldInfo {
	var fields []FieldInfo

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		tag := structType.Tag(i)

		fieldInfo := FieldInfo{
			Name:     field.Name(),
			Type:     field.Type().String(),
			Tag:      tag,
			Exported: field.Exported(),
			Embedded: field.Embedded(),
		}

		// Get position if available
		if pos := a.fset.Position(field.Pos()); pos.IsValid() {
			fieldInfo.Position = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}

		fields = append(fields, fieldInfo)
	}

	return fields
}

// analyzeInterfaceMethods analyzes interface methods
func (a *Analyzer) analyzeInterfaceMethods(interfaceType *types.Interface) []MethodInfo {
	var methods []MethodInfo

	for i := 0; i < interfaceType.NumMethods(); i++ {
		method := interfaceType.Method(i)
		sig := method.Type().(*types.Signature)

		methodInfo := MethodInfo{
			Name:      method.Name(),
			Signature: sig.String(),
			Exported:  method.Exported(),
		}

		// Get parameters and results
		methodInfo.Parameters = a.analyzeSignatureParams(sig.Params())
		methodInfo.Results = a.analyzeSignatureParams(sig.Results())

		// Get position if available
		if pos := a.fset.Position(method.Pos()); pos.IsValid() {
			methodInfo.Position = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}

		methods = append(methods, methodInfo)
	}

	return methods
}

// getTypeMethods gets all methods for a type
func (a *Analyzer) getTypeMethods(t types.Type) []MethodInfo {
	var methods []MethodInfo

	// Get methods for the type
	mset := types.NewMethodSet(t)
	for i := 0; i < mset.Len(); i++ {
		selection := mset.At(i)
		if selection.Kind() != types.MethodVal {
			continue
		}

		method := selection.Obj().(*types.Func)
		sig := method.Type().(*types.Signature)

		methodInfo := MethodInfo{
			Name:      method.Name(),
			Signature: sig.String(),
			Exported:  method.Exported(),
			IsPointer: selection.Indirect(),
		}

		// Get receiver information
		if recv := sig.Recv(); recv != nil {
			methodInfo.Receiver = recv.Type().String()
		}

		// Get parameters and results
		methodInfo.Parameters = a.analyzeSignatureParams(sig.Params())
		methodInfo.Results = a.analyzeSignatureParams(sig.Results())

		// Get position if available
		if pos := a.fset.Position(method.Pos()); pos.IsValid() {
			methodInfo.Position = Position{
				Filename: pos.Filename,
				Line:     pos.Line,
				Column:   pos.Column,
			}
		}

		methods = append(methods, methodInfo)
	}

	// Also check pointer type methods
	if _, ok := t.(*types.Pointer); !ok {
		ptrType := types.NewPointer(t)
		ptrMset := types.NewMethodSet(ptrType)
		for i := 0; i < ptrMset.Len(); i++ {
			selection := ptrMset.At(i)
			if selection.Kind() != types.MethodVal {
				continue
			}

			method := selection.Obj().(*types.Func)
			sig := method.Type().(*types.Signature)

			// Skip if we already have this method
			found := false
			for _, existing := range methods {
				if existing.Name == method.Name() {
					found = true
					break
				}
			}
			if found {
				continue
			}

			methodInfo := MethodInfo{
				Name:      method.Name(),
				Signature: sig.String(),
				Exported:  method.Exported(),
				IsPointer: true,
			}

			// Get receiver information
			if recv := sig.Recv(); recv != nil {
				methodInfo.Receiver = recv.Type().String()
			}

			// Get parameters and results
			methodInfo.Parameters = a.analyzeSignatureParams(sig.Params())
			methodInfo.Results = a.analyzeSignatureParams(sig.Results())

			// Get position if available
			if pos := a.fset.Position(method.Pos()); pos.IsValid() {
				methodInfo.Position = Position{
					Filename: pos.Filename,
					Line:     pos.Line,
					Column:   pos.Column,
				}
			}

			methods = append(methods, methodInfo)
		}
	}

	// Sort methods by name
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	return methods
}

// analyzeSignatureParams analyzes function signature parameters
func (a *Analyzer) analyzeSignatureParams(tuple *types.Tuple) []ParamInfo {
	var params []ParamInfo

	for i := 0; i < tuple.Len(); i++ {
		param := tuple.At(i)
		paramInfo := ParamInfo{
			Name: param.Name(),
			Type: param.Type().String(),
		}
		params = append(params, paramInfo)
	}

	return params
}

// ListMethods returns all methods for a given type with comprehensive information
func (a *Analyzer) ListMethods(typeName string) ([]MethodInfo, error) {
	typeInfo, err := a.LookupType(typeName)
	if err != nil {
		return nil, err
	}

	return typeInfo.Methods, nil
}

// GetExample returns examples for a given type or topic
func (a *Analyzer) GetExample(topic string) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var examples []string

	// Look for examples in documentation
	for _, docPkg := range a.docPkgs {
		// Check type examples
		for _, docType := range docPkg.Types {
			if strings.Contains(strings.ToLower(docType.Name), strings.ToLower(topic)) {
				for _, example := range docType.Examples {
					examples = append(examples, fmt.Sprintf("Example: %s\n%s\n%s",
						example.Name,
						fmt.Sprintf("%s", example.Code),
						example.Doc))
				}
			}
		}

		// Check function examples
		for _, docFunc := range docPkg.Funcs {
			if strings.Contains(strings.ToLower(docFunc.Name), strings.ToLower(topic)) {
				for _, example := range docFunc.Examples {
					examples = append(examples, fmt.Sprintf("Example: %s\n%s\n%s",
						example.Name,
						fmt.Sprintf("%s", example.Code),
						example.Doc))
				}
			}
		}

		// Check package examples
		for _, example := range docPkg.Examples {
			if strings.Contains(strings.ToLower(example.Name), strings.ToLower(topic)) {
				examples = append(examples, fmt.Sprintf("Example: %s\n%s\n%s",
					example.Name,
					fmt.Sprintf("%s", example.Code),
					example.Doc))
			}
		}
	}

	if len(examples) == 0 {
		return "", fmt.Errorf("no examples found for topic: %s", topic)
	}

	return strings.Join(examples, "\n\n"), nil
}

// AnalyzeRepository performs a comprehensive analysis of the entire repository
func (a *Analyzer) AnalyzeRepository(ctx context.Context) (*AnalysisResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.initialized {
		return nil, fmt.Errorf("analyzer not initialized")
	}

	start := time.Now()
	result := &AnalysisResult{
		Timestamp: start,
	}

	// Analyze types
	for pkgName, pkg := range a.pkgs {
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil {
				continue
			}

			switch obj := obj.(type) {
			case *types.TypeName:
				if typeInfo, err := a.LookupType(obj.Name()); err == nil {
					result.Types = append(result.Types, *typeInfo)
				}
			case *types.Func:
				funcInfo := a.analyzeFunctionObject(obj, pkgName)
				result.Functions = append(result.Functions, funcInfo)
			case *types.Var:
				varInfo := a.analyzeVariableObject(obj, pkgName)
				result.Variables = append(result.Variables, varInfo)
			case *types.Const:
				constInfo := a.analyzeConstantObject(obj, pkgName)
				result.Constants = append(result.Constants, constInfo)
			}
		}
	}

	// Analyze packages
	for pkgName, pkg := range a.pkgs {
		pkgInfo := PackageInfo{
			Name:       pkgName,
			ImportPath: pkg.Path(),
			IsMain:     pkgName == "main",
		}

		// Get documentation
		if docPkg := a.docPkgs[pkgName]; docPkg != nil {
			pkgInfo.Doc = docPkg.Doc
		}

		// Get files
		pkgInfo.Files = a.files[pkgName]

		result.Packages = append(result.Packages, pkgInfo)
	}

	// Calculate metrics
	result.Metrics = AnalysisMetrics{
		TotalTypes:     len(result.Types),
		TotalFunctions: len(result.Functions),
		TotalPackages:  len(result.Packages),
		AnalysisTime:   time.Since(start),
	}

	result.Duration = time.Since(start)
	return result, nil
}

// analyzeFunctionObject analyzes a function object
func (a *Analyzer) analyzeFunctionObject(fn *types.Func, pkgName string) FunctionInfo {
	sig := fn.Type().(*types.Signature)

	funcInfo := FunctionInfo{
		Name:     fn.Name(),
		Package:  pkgName,
		Exported: fn.Exported(),
		IsMethod: sig.Recv() != nil,
	}

	// Get signature
	funcInfo.Signature = sig.String()

	// Get parameters and results
	funcInfo.Parameters = a.analyzeSignatureParams(sig.Params())
	funcInfo.Results = a.analyzeSignatureParams(sig.Results())

	// Get position
	if pos := a.fset.Position(fn.Pos()); pos.IsValid() {
		funcInfo.Position = Position{
			Filename: pos.Filename,
			Line:     pos.Line,
			Column:   pos.Column,
		}
	}

	return funcInfo
}

// analyzeVariableObject analyzes a variable object
func (a *Analyzer) analyzeVariableObject(v *types.Var, pkgName string) VariableInfo {
	varInfo := VariableInfo{
		Name:     v.Name(),
		Type:     v.Type().String(),
		Package:  pkgName,
		Exported: v.Exported(),
	}

	// Get position
	if pos := a.fset.Position(v.Pos()); pos.IsValid() {
		varInfo.Position = Position{
			Filename: pos.Filename,
			Line:     pos.Line,
			Column:   pos.Column,
		}
	}

	return varInfo
}

// analyzeConstantObject analyzes a constant object
func (a *Analyzer) analyzeConstantObject(c *types.Const, pkgName string) ConstantInfo {
	constInfo := ConstantInfo{
		Name:     c.Name(),
		Type:     c.Type().String(),
		Value:    c.Val().String(),
		Package:  pkgName,
		Exported: c.Exported(),
	}

	// Get position
	if pos := a.fset.Position(c.Pos()); pos.IsValid() {
		constInfo.Position = Position{
			Filename: pos.Filename,
			Line:     pos.Line,
			Column:   pos.Column,
		}
	}

	return constInfo
}

// SearchTypes searches for types matching a query
func (a *Analyzer) SearchTypes(query string) ([]TypeInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []TypeInfo
	query = strings.ToLower(query)

	for _, pkg := range a.pkgs {
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			if obj == nil {
				continue
			}

			if typeName, ok := obj.(*types.TypeName); ok {
				// Check if name matches query
				if strings.Contains(strings.ToLower(typeName.Name()), query) {
					if typeInfo, err := a.LookupType(typeName.Name()); err == nil {
						results = append(results, *typeInfo)
					}
				}
			}
		}
	}

	return results, nil
}

// GetPackageInfo returns information about a specific package
func (a *Analyzer) GetPackageInfo(packageName string) (*PackageInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	pkg, exists := a.pkgs[packageName]
	if !exists {
		return nil, fmt.Errorf("package %s not found", packageName)
	}

	pkgInfo := &PackageInfo{
		Name:       packageName,
		ImportPath: pkg.Path(),
		IsMain:     packageName == "main",
	}

	// Get documentation
	if docPkg := a.docPkgs[packageName]; docPkg != nil {
		pkgInfo.Doc = docPkg.Doc
	}

	// Get files
	pkgInfo.Files = a.files[packageName]

	return pkgInfo, nil
}

// Refresh re-analyzes the repository
func (a *Analyzer) Refresh() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logInfo("Refreshing repository analysis")

	// Clear existing data
	a.pkgs = make(map[string]*types.Package)
	a.docPkgs = make(map[string]*doc.Package)
	a.fset = token.NewFileSet()
	a.initialized = false
	a.files = make(map[string][]string)

	// Re-initialize
	return a.initialize()
}

// Close cleans up resources
func (a *Analyzer) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logInfo("Closing analyzer")
	a.initialized = false
	return nil
}

// Logging methods
func (a *Analyzer) logWarn(format string, args ...interface{}) {
	if a.config.LogLevel >= LogLevelWarn {
		a.logger.Printf("[WARN] "+format, args...)
	}
}

func (a *Analyzer) logInfo(format string, args ...interface{}) {
	if a.config.LogLevel >= LogLevelInfo {
		a.logger.Printf("[INFO] "+format, args...)
	}
}
