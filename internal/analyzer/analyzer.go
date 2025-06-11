package analyzer

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
)

// Analyzer handles the analysis of Go types and methods
type Analyzer struct {
	repoPath string
	fset     *token.FileSet
	pkgs     map[string]*types.Package
	info     *types.Info
}

// TypeInfo represents information about a Go type
type TypeInfo struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Package string   `json:"package"`
	Doc     string   `json:"doc"`
	Methods []string `json:"methods,omitempty"`
}

// NewAnalyzer creates a new Analyzer instance
func NewAnalyzer(repoPath string) (*Analyzer, error) {
	fset := token.NewFileSet()

	// Parse the directory
	pkgs, err := parser.ParseDir(fset, repoPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory: %w", err)
	}

	// Create type checker
	conf := types.Config{
		Importer: importer.Default(),
	}

	// Convert ast.Packages to types.Packages
	typePkgs := make(map[string]*types.Package)
	for pkgName, pkg := range pkgs {
		// Create a slice of files for this package
		var files []*ast.File
		for _, file := range pkg.Files {
			files = append(files, file)
		}

		// Type check the package
		info := &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
		}

		typePkg, err := conf.Check(pkgName, fset, files, info)
		if err != nil {
			return nil, fmt.Errorf("failed to type check package %s: %w", pkgName, err)
		}

		typePkgs[pkgName] = typePkg
	}

	return &Analyzer{
		repoPath: repoPath,
		fset:     fset,
		pkgs:     typePkgs,
		info:     &types.Info{},
	}, nil
}

// LookupType finds and returns information about a specific type
func (a *Analyzer) LookupType(typeName string) (*TypeInfo, error) {
	for pkgName, pkg := range a.pkgs {
		obj := pkg.Scope().Lookup(typeName)
		if obj != nil {
			info := &TypeInfo{
				Name:    typeName,
				Package: pkgName,
			}

			// Get documentation
			if doc := obj.String(); doc != "" {
				info.Doc = doc
			}

			// Determine kind
			switch obj.Type().Underlying().(type) {
			case *types.Struct:
				info.Kind = "struct"
			case *types.Interface:
				info.Kind = "interface"
			default:
				info.Kind = "other"
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("type %s not found", typeName)
}

// ListMethods returns all methods for a given type
func (a *Analyzer) ListMethods(typeName string) ([]string, error) {
	var methods []string

	for _, pkg := range a.pkgs {
		obj := pkg.Scope().Lookup(typeName)
		if obj != nil {
			// Get all methods for this type
			mset := types.NewMethodSet(obj.Type())
			for i := 0; i < mset.Len(); i++ {
				selection := mset.At(i)
				if selection.Kind() == types.MethodVal {
					methods = append(methods, selection.Obj().Name())
				}
			}

			// Also check for methods on pointer type
			ptrType := types.NewPointer(obj.Type())
			mset = types.NewMethodSet(ptrType)
			for i := 0; i < mset.Len(); i++ {
				selection := mset.At(i)
				if selection.Kind() == types.MethodVal {
					methods = append(methods, selection.Obj().Name())
				}
			}
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no methods found for type %s", typeName)
	}

	return methods, nil
}

// GetExample returns an example for a given type or topic
func (a *Analyzer) GetExample(topic string) (string, error) {
	// For now, return a placeholder. In a real implementation, this would:
	// 1. Look up examples in a curated examples directory
	// 2. Parse test files for examples
	// 3. Check documentation for embedded examples
	return fmt.Sprintf("Example not found for: %s", topic), nil
}
