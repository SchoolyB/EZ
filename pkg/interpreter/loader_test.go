package interpreter

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// ModuleLoader Tests
// ============================================================================

func TestNewModuleLoader(t *testing.T) {
	loader := NewModuleLoader("/test/root")

	if loader.rootPath != "/test/root" {
		t.Errorf("rootPath = %q, want %q", loader.rootPath, "/test/root")
	}
	if loader.cache == nil {
		t.Error("cache should not be nil")
	}
	if len(loader.warnings) != 0 {
		t.Errorf("warnings should be empty, got %d", len(loader.warnings))
	}
}

func TestModuleLoaderWarnings(t *testing.T) {
	loader := NewModuleLoader("/test")

	// Test AddWarning
	loader.AddWarning("warning 1")
	loader.AddWarning("warning 2")

	// Test GetWarnings
	warnings := loader.GetWarnings()
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}
	if warnings[0] != "warning 1" {
		t.Errorf("warnings[0] = %q, want %q", warnings[0], "warning 1")
	}
	if warnings[1] != "warning 2" {
		t.Errorf("warnings[1] = %q, want %q", warnings[1], "warning 2")
	}

	// Test ClearWarnings
	loader.ClearWarnings()
	if len(loader.GetWarnings()) != 0 {
		t.Error("warnings should be empty after ClearWarnings")
	}
}

func TestModuleLoaderSetCurrentFile(t *testing.T) {
	loader := NewModuleLoader("/test")

	loader.SetCurrentFile("/test/src/main.ez")
	if loader.currentFile != "/test/src/main.ez" {
		t.Errorf("currentFile = %q, want %q", loader.currentFile, "/test/src/main.ez")
	}
}

func TestResolveImport(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	loader := NewModuleLoader(tmpDir)

	tests := []struct {
		name        string
		importPath  string
		currentFile string
		wantSuffix  string
	}{
		{
			name:        "relative import with ./",
			importPath:  "./utils",
			currentFile: filepath.Join(tmpDir, "src", "main.ez"),
			wantSuffix:  filepath.Join("src", "utils"),
		},
		{
			name:        "relative import with ../",
			importPath:  "../lib",
			currentFile: filepath.Join(tmpDir, "src", "sub", "main.ez"),
			wantSuffix:  filepath.Join("src", "lib"),
		},
		{
			name:        "relative import no current file",
			importPath:  "./utils",
			currentFile: "",
			wantSuffix:  "utils",
		},
		{
			name:        "absolute path",
			importPath:  "/absolute/path",
			currentFile: "",
			wantSuffix:  "/absolute/path",
		},
		{
			name:        "relative to root",
			importPath:  "lib/utils",
			currentFile: "",
			wantSuffix:  filepath.Join("lib", "utils"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader.SetCurrentFile(tt.currentFile)
			result, err := loader.ResolveImport(tt.importPath)
			if err != nil {
				t.Fatalf("ResolveImport() error = %v", err)
			}
			if !strings.HasSuffix(result, tt.wantSuffix) {
				t.Errorf("ResolveImport() = %q, want suffix %q", result, tt.wantSuffix)
			}
		})
	}
}

func TestLoadFileModule(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test .ez file
	testFile := filepath.Join(tmpDir, "test.ez")
	content := `module test

do add(a int, b int) -> int {
    return a + b
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	mod, err := loader.Load("./test")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if mod.Name != "test" {
		t.Errorf("module name = %q, want %q", mod.Name, "test")
	}
	if mod.IsDir {
		t.Error("module should not be a directory module")
	}
	if mod.AST == nil {
		t.Error("module AST should not be nil")
	}
}

func TestLoadDirectoryModule(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a module directory
	modDir := filepath.Join(tmpDir, "mymod")
	if err := os.Mkdir(modDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create two .ez files in the module
	file1 := `module mymod

do foo() -> int {
    return 1
}
`
	file2 := `module mymod

do bar() -> int {
    return 2
}
`
	if err := os.WriteFile(filepath.Join(modDir, "foo.ez"), []byte(file1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "bar.ez"), []byte(file2), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	mod, err := loader.Load("./mymod")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if mod.Name != "mymod" {
		t.Errorf("module name = %q, want %q", mod.Name, "mymod")
	}
	if !mod.IsDir {
		t.Error("module should be a directory module")
	}
	if len(mod.Files) != 2 {
		t.Errorf("module files count = %d, want 2", len(mod.Files))
	}
}

func TestLoadModuleNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	loader := NewModuleLoader(tmpDir)
	_, err = loader.Load("./nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent module")
	}

	modErr, ok := err.(*ModuleError)
	if !ok {
		t.Fatalf("error should be *ModuleError, got %T", err)
	}
	if modErr.Code != "E6001" {
		t.Errorf("error code = %q, want %q", modErr.Code, "E6001")
	}
}

func TestLoadEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	_, err = loader.Load("./empty")
	if err == nil {
		t.Fatal("expected error for empty directory")
	}

	modErr, ok := err.(*ModuleError)
	if !ok {
		t.Fatalf("error should be *ModuleError, got %T", err)
	}
	if modErr.Code != "E6001" {
		t.Errorf("error code = %q, want %q", modErr.Code, "E6001")
	}
}

func TestLoadModuleNameMismatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a module directory with mismatched names
	modDir := filepath.Join(tmpDir, "mymod")
	if err := os.Mkdir(modDir, 0755); err != nil {
		t.Fatal(err)
	}

	file1 := `module mymod
do foo() -> int { return 1 }
`
	file2 := `module different_name
do bar() -> int { return 2 }
`
	if err := os.WriteFile(filepath.Join(modDir, "foo.ez"), []byte(file1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "bar.ez"), []byte(file2), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	_, err = loader.Load("./mymod")
	if err == nil {
		t.Fatal("expected error for module name mismatch")
	}

	modErr, ok := err.(*ModuleError)
	if !ok {
		t.Fatalf("error should be *ModuleError, got %T", err)
	}
	if modErr.Code != "E6006" {
		t.Errorf("error code = %q, want %q", modErr.Code, "E6006")
	}
}

func TestCheckInternalAccess(t *testing.T) {
	loader := NewModuleLoader("/project")

	tests := []struct {
		name        string
		targetPath  string
		currentFile string
		wantErr     bool
	}{
		{
			name:        "no internal in path",
			targetPath:  "/project/lib/utils",
			currentFile: "/project/src/main.ez",
			wantErr:     false,
		},
		{
			name:        "sibling access to internal",
			targetPath:  "/project/internal/utils",
			currentFile: "/project/main.ez",
			wantErr:     false,
		},
		{
			name:        "child access to internal",
			targetPath:  "/project/internal/utils",
			currentFile: "/project/src/main.ez",
			wantErr:     false,
		},
		{
			name:        "external access to internal blocked",
			targetPath:  "/project/pkg/internal/secret",
			currentFile: "/other/project/main.ez",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader.SetCurrentFile(tt.currentFile)
			err := loader.checkInternalAccess(tt.targetPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkInternalAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				modErr, ok := err.(*ModuleError)
				if !ok {
					t.Errorf("error should be *ModuleError, got %T", err)
				} else if modErr.Code != "E6007" {
					t.Errorf("error code = %q, want %q", modErr.Code, "E6007")
				}
			}
		})
	}
}

func TestGetCachedAndClearCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "cached.ez")
	content := `do main() { }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)

	// Load the module
	mod, err := loader.Load("./cached")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// The cache key is the resolved path before .ez extension is added
	// So we need to resolve the import path to get the cache key
	cacheKey, _ := loader.ResolveImport("./cached")

	// Test GetCached - should find it
	cached, ok := loader.GetCached(cacheKey)
	if !ok {
		t.Error("GetCached() should find cached module")
	}
	if cached != mod {
		t.Error("GetCached() should return same module")
	}

	// Test GetCached with wrong path - should not find
	_, ok = loader.GetCached("/nonexistent/path")
	if ok {
		t.Error("GetCached() should not find nonexistent path")
	}

	// Test ClearCache
	loader.ClearCache()
	_, ok = loader.GetCached(cacheKey)
	if ok {
		t.Error("GetCached() should not find module after ClearCache")
	}
}

func TestModuleErrorFormat(t *testing.T) {
	// Test simple error
	err := &ModuleError{
		Code:    "E6001",
		Message: "module not found",
		Path:    "/test/path",
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "E6001") {
		t.Errorf("error string should contain code, got %q", errStr)
	}
	if !strings.Contains(errStr, "module not found") {
		t.Errorf("error string should contain message, got %q", errStr)
	}
}

func TestLoadCachedModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "cached.ez")
	content := `do main() { }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)

	// Load the module twice
	mod1, err := loader.Load("./cached")
	if err != nil {
		t.Fatalf("first Load() error = %v", err)
	}

	mod2, err := loader.Load("./cached")
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}

	// Should return same cached module
	if mod1 != mod2 {
		t.Error("second Load() should return cached module")
	}
}

func TestLoadModuleWithParseError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with invalid syntax
	testFile := filepath.Join(tmpDir, "invalid.ez")
	content := `do invalid( { }`  // Missing parameter list close
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	_, err = loader.Load("./invalid")
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}

	modErr, ok := err.(*ModuleError)
	if !ok {
		t.Fatalf("error should be *ModuleError, got %T", err)
	}
	if modErr.Code != "E6002" {
		t.Errorf("error code = %q, want %q", modErr.Code, "E6002")
	}
}

func TestLoadModuleNameWarning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ez-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file where module name doesn't match filename
	testFile := filepath.Join(tmpDir, "myfile.ez")
	content := `module different_name
do foo() -> int { return 1 }
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewModuleLoader(tmpDir)
	_, err = loader.Load("./myfile")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	warnings := loader.GetWarnings()
	if len(warnings) == 0 {
		t.Error("expected warning for module name mismatch")
	}
	if len(warnings) > 0 && !strings.Contains(warnings[0], "W4001") {
		t.Errorf("warning should contain W4001, got %q", warnings[0])
	}
}
