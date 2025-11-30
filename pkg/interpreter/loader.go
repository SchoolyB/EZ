package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/lexer"
	"github.com/marshallburns/ez/pkg/parser"
)

// ModuleState represents the loading state of a module
type ModuleState int

const (
	ModuleUnloaded ModuleState = iota // Not yet loaded
	ModuleLoading                     // Currently being loaded (cycle detection)
	ModuleLoaded                      // Fully loaded and evaluated
)

// Module represents a loaded EZ module
type Module struct {
	Name       string                 // Module name (from declaration or inferred from path)
	FilePath   string                 // Absolute path to the module file/directory
	AST        *ast.Program           // Parsed AST
	Exports    map[string]Object      // Public symbols
	Private    map[string]Object      // Private symbols (file-scoped)
	ModPrivate map[string]Object      // Module-private symbols (directory-scoped)
	State      ModuleState            // Loading state
	Env        *Environment           // Module's environment
	Files      []string               // List of files (for directory modules)
	IsDir      bool                   // True if this is a directory module
}

// ModuleLoader handles loading and caching of modules
type ModuleLoader struct {
	cache       map[string]*Module // Cache of loaded modules (keyed by absolute path)
	rootPath    string             // Project root directory
	currentFile string             // Currently processing file (for relative imports)
}

// NewModuleLoader creates a new module loader
func NewModuleLoader(rootPath string) *ModuleLoader {
	return &ModuleLoader{
		cache:    make(map[string]*Module),
		rootPath: rootPath,
	}
}

// SetCurrentFile sets the current file being processed (for relative import resolution)
func (l *ModuleLoader) SetCurrentFile(path string) {
	l.currentFile = path
}

// Load loads a module from the given path
// The path can be:
//   - A relative path like "./utils" or "../server"
//   - An absolute path
func (l *ModuleLoader) Load(importPath string) (*Module, error) {
	// Resolve the import path to an absolute path
	absPath, err := l.ResolveImport(importPath)
	if err != nil {
		return nil, err
	}

	// Check cache first
	if mod, ok := l.cache[absPath]; ok {
		// Check for circular import during loading
		if mod.State == ModuleLoading {
			// This is a circular import - we'll handle it with forward references
			// For now, return the partially loaded module
			return mod, nil
		}
		return mod, nil
	}

	// Create a new module entry and mark as loading (for cycle detection)
	mod := &Module{
		FilePath:   absPath,
		State:      ModuleLoading,
		Exports:    make(map[string]Object),
		Private:    make(map[string]Object),
		ModPrivate: make(map[string]Object),
	}
	l.cache[absPath] = mod

	// Check if path is a file or directory
	info, err := os.Stat(absPath)
	if err != nil {
		// Try adding .ez extension
		absPathWithExt := absPath + ".ez"
		info, err = os.Stat(absPathWithExt)
		if err != nil {
			delete(l.cache, absPath)
			return nil, &ModuleError{
				Code:    "E6001",
				Message: "module not found: " + importPath,
				Path:    importPath,
			}
		}
		absPath = absPathWithExt
		mod.FilePath = absPath
	}

	if info.IsDir() {
		// Directory module (Odin-style)
		mod.IsDir = true
		if err := l.loadDirectoryModule(mod); err != nil {
			delete(l.cache, absPath)
			return nil, err
		}
	} else {
		// Single-file module
		mod.IsDir = false
		mod.Files = []string{absPath}
		if err := l.loadFileModule(mod, absPath); err != nil {
			delete(l.cache, absPath)
			return nil, err
		}
	}

	mod.State = ModuleLoaded
	return mod, nil
}

// ResolveImport resolves an import path to an absolute path
func (l *ModuleLoader) ResolveImport(importPath string) (string, error) {
	// Handle relative imports
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		var basePath string
		if l.currentFile != "" {
			basePath = filepath.Dir(l.currentFile)
		} else {
			basePath = l.rootPath
		}
		return filepath.Abs(filepath.Join(basePath, importPath))
	}

	// Absolute path
	if filepath.IsAbs(importPath) {
		return importPath, nil
	}

	// Relative to root
	return filepath.Abs(filepath.Join(l.rootPath, importPath))
}

// loadFileModule loads a single .ez file as a module
func (l *ModuleLoader) loadFileModule(mod *Module, filePath string) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &ModuleError{
			Code:    "E6002",
			Message: "failed to read module file: " + err.Error(),
			Path:    filePath,
		}
	}

	// Parse the file
	lex := lexer.NewLexer(string(content))
	p := parser.NewWithSource(lex, string(content), filePath)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return &ModuleError{
			Code:    "E6002",
			Message: "parse error in module: " + strings.Join(p.Errors(), "; "),
			Path:    filePath,
		}
	}

	mod.AST = program

	// Extract module name from declaration or infer from filename
	if program.Module != nil {
		mod.Name = program.Module.Name.Value
	} else {
		// Infer from filename
		mod.Name = strings.TrimSuffix(filepath.Base(filePath), ".ez")
	}

	return nil
}

// loadDirectoryModule loads all .ez files in a directory as a single module
func (l *ModuleLoader) loadDirectoryModule(mod *Module) error {
	// Find all .ez files in the directory
	entries, err := os.ReadDir(mod.FilePath)
	if err != nil {
		return &ModuleError{
			Code:    "E6002",
			Message: "failed to read module directory: " + err.Error(),
			Path:    mod.FilePath,
		}
	}

	var ezFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ez") {
			ezFiles = append(ezFiles, filepath.Join(mod.FilePath, entry.Name()))
		}
	}

	if len(ezFiles) == 0 {
		return &ModuleError{
			Code:    "E6001",
			Message: "no .ez files found in module directory",
			Path:    mod.FilePath,
		}
	}

	mod.Files = ezFiles

	// Parse all files and verify they declare the same module name
	var declaredModuleName string
	var combinedStatements []ast.Statement

	for _, filePath := range ezFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return &ModuleError{
				Code:    "E6002",
				Message: "failed to read module file: " + err.Error(),
				Path:    filePath,
			}
		}

		lex := lexer.NewLexer(string(content))
		p := parser.NewWithSource(lex, string(content), filePath)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			return &ModuleError{
				Code:    "E6002",
				Message: "parse error in module file: " + strings.Join(p.Errors(), "; "),
				Path:    filePath,
			}
		}

		// Check module declaration consistency
		if program.Module != nil {
			if declaredModuleName == "" {
				declaredModuleName = program.Module.Name.Value
			} else if program.Module.Name.Value != declaredModuleName {
				return &ModuleError{
					Code: "E6006",
					Message: "module name mismatch in directory: " + filePath +
						" declares '" + program.Module.Name.Value +
						"' but expected '" + declaredModuleName + "'",
					Path: filePath,
				}
			}
		}

		// Combine statements
		combinedStatements = append(combinedStatements, program.Statements...)
	}

	// Set module name
	if declaredModuleName != "" {
		mod.Name = declaredModuleName
	} else {
		// Infer from directory name
		mod.Name = filepath.Base(mod.FilePath)
	}

	// Create combined AST
	mod.AST = &ast.Program{
		Statements: combinedStatements,
	}

	return nil
}

// GetCached returns a cached module if it exists
func (l *ModuleLoader) GetCached(absPath string) (*Module, bool) {
	mod, ok := l.cache[absPath]
	return mod, ok
}

// ClearCache clears the module cache
func (l *ModuleLoader) ClearCache() {
	l.cache = make(map[string]*Module)
}

// ModuleError represents a module loading error
type ModuleError struct {
	Code    string
	Message string
	Path    string
}

func (e *ModuleError) Error() string {
	return e.Code + ": " + e.Message
}
