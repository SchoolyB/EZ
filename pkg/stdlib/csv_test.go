package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to get return values from a tuple
func csvGetReturnValues(t *testing.T, obj object.Object) []object.Object {
	t.Helper()
	rv, ok := obj.(*object.ReturnValue)
	if !ok {
		t.Fatalf("expected ReturnValue, got %T", obj)
	}
	return rv.Values
}

// Helper to create a string object
func csvMakeStr(s string) *object.String {
	return &object.String{Value: s}
}

// Helper to create a 2D array from strings
func csvMake2DArray(rows [][]string) *object.Array {
	elements := make([]object.Object, len(rows))
	for i, row := range rows {
		rowElements := make([]object.Object, len(row))
		for j, field := range row {
			rowElements[j] = &object.String{Value: field}
		}
		elements[i] = &object.Array{Elements: rowElements, ElementType: "string"}
	}
	return &object.Array{Elements: elements, ElementType: "[string]"}
}

// ============================================================================
// csv.parse tests
// ============================================================================

func TestCsvParse(t *testing.T) {
	fn := CsvBuiltins["csv.parse"].Fn

	tests := []struct {
		name         string
		input        string
		expectedRows int
		expectedCols int
		isError      bool
	}{
		{"simple csv", "a,b,c\n1,2,3", 2, 3, false},
		{"single row", "hello,world", 1, 2, false},
		{"quoted fields", "\"hello, world\",test", 1, 2, false},
		{"empty string", "", 0, 0, false},
		{"newlines in quotes", "\"line1\nline2\",b", 1, 2, false},
		{"windows newlines", "a,b\r\nc,d", 2, 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(csvMakeStr(tt.input))
			values := csvGetReturnValues(t, result)

			if tt.isError {
				if values[1] == object.NIL {
					t.Error("expected error, got nil")
				}
				return
			}

			if values[1] != object.NIL {
				t.Errorf("expected nil error, got %v", values[1])
				return
			}

			arr, ok := values[0].(*object.Array)
			if !ok {
				if tt.expectedRows == 0 {
					return // Empty result is okay
				}
				t.Fatalf("expected Array, got %T", values[0])
			}

			if len(arr.Elements) != tt.expectedRows {
				t.Errorf("parse(%q) returned %d rows, want %d", tt.input, len(arr.Elements), tt.expectedRows)
			}

			if tt.expectedRows > 0 {
				firstRow, ok := arr.Elements[0].(*object.Array)
				if !ok {
					t.Fatalf("expected first row to be Array, got %T", arr.Elements[0])
				}
				if len(firstRow.Elements) != tt.expectedCols {
					t.Errorf("parse(%q) first row has %d cols, want %d", tt.input, len(firstRow.Elements), tt.expectedCols)
				}
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong type
	t.Run("wrong type", func(t *testing.T) {
		result := fn(makeInt(42))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

// ============================================================================
// csv.stringify tests
// ============================================================================

func TestCsvStringify(t *testing.T) {
	fn := CsvBuiltins["csv.stringify"].Fn

	tests := []struct {
		name     string
		input    [][]string
		expected string
	}{
		{"simple", [][]string{{"a", "b"}, {"1", "2"}}, "a,b\n1,2\n"},
		{"single row", [][]string{{"hello", "world"}}, "hello,world\n"},
		{"needs quoting", [][]string{{"hello, world", "test"}}, "\"hello, world\",test\n"},
		{"empty array", [][]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(csvMake2DArray(tt.input))
			values := csvGetReturnValues(t, result)

			if values[1] != object.NIL {
				t.Errorf("expected nil error, got %v", values[1])
				return
			}

			str, ok := values[0].(*object.String)
			if !ok {
				t.Fatalf("expected String, got %T", values[0])
			}

			if str.Value != tt.expected {
				t.Errorf("stringify() = %q, want %q", str.Value, tt.expected)
			}
		})
	}

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong type
	t.Run("wrong type", func(t *testing.T) {
		result := fn(csvMakeStr("not an array"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong type")
		}
	})
}

// ============================================================================
// csv.read tests
// ============================================================================

func TestCsvRead(t *testing.T) {
	fn := CsvBuiltins["csv.read"].Fn

	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := "name,age\nAlice,30\nBob,25"
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("read file", func(t *testing.T) {
		result := fn(csvMakeStr(csvPath))
		values := csvGetReturnValues(t, result)

		if values[1] != object.NIL {
			t.Errorf("expected nil error, got %v", values[1])
			return
		}

		arr, ok := values[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", values[0])
		}

		if len(arr.Elements) != 3 {
			t.Errorf("read() returned %d rows, want 3", len(arr.Elements))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result := fn(csvMakeStr("/nonexistent/file.csv"))
		values := csvGetReturnValues(t, result)

		if values[1] == object.NIL {
			t.Error("expected error for nonexistent file")
		}
	})

	// Test with options
	t.Run("with delimiter option", func(t *testing.T) {
		tsvPath := filepath.Join(tmpDir, "test.tsv")
		tsvContent := "name\tage\nAlice\t30"
		err := os.WriteFile(tsvPath, []byte(tsvContent), 0644)
		if err != nil {
			t.Fatalf("failed to create tsv file: %v", err)
		}

		opts := &object.Map{
			Pairs: []*object.MapPair{
				{
					Key:   &object.String{Value: "delimiter"},
					Value: &object.String{Value: "\t"},
				},
			},
		}

		result := fn(csvMakeStr(tsvPath), opts)
		values := csvGetReturnValues(t, result)

		if values[1] != object.NIL {
			t.Errorf("expected nil error, got %v", values[1])
			return
		}

		arr, ok := values[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", values[0])
		}

		if len(arr.Elements) != 2 {
			t.Errorf("read() returned %d rows, want 2", len(arr.Elements))
		}
	})

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})
}

// ============================================================================
// csv.headers tests
// ============================================================================

func TestCsvHeaders(t *testing.T) {
	fn := CsvBuiltins["csv.headers"].Fn

	// Create a temporary CSV file for testing
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csvContent := "name,age,city\nAlice,30,NYC"
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("get headers", func(t *testing.T) {
		result := fn(csvMakeStr(csvPath))
		values := csvGetReturnValues(t, result)

		if values[1] != object.NIL {
			t.Errorf("expected nil error, got %v", values[1])
			return
		}

		arr, ok := values[0].(*object.Array)
		if !ok {
			t.Fatalf("expected Array, got %T", values[0])
		}

		if len(arr.Elements) != 3 {
			t.Errorf("headers() returned %d elements, want 3", len(arr.Elements))
		}

		expected := []string{"name", "age", "city"}
		for i, elem := range arr.Elements {
			str, ok := elem.(*object.String)
			if !ok {
				t.Fatalf("header %d: expected String, got %T", i, elem)
			}
			if str.Value != expected[i] {
				t.Errorf("header %d = %q, want %q", i, str.Value, expected[i])
			}
		}
	})

	t.Run("file not found", func(t *testing.T) {
		result := fn(csvMakeStr("/nonexistent/file.csv"))
		values := csvGetReturnValues(t, result)

		if values[1] == object.NIL {
			t.Error("expected error for nonexistent file")
		}
	})

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})
}

// ============================================================================
// csv.write tests
// ============================================================================

func TestCsvWrite(t *testing.T) {
	fn := CsvBuiltins["csv.write"].Fn

	tmpDir := t.TempDir()

	t.Run("write file", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "output.csv")
		data := csvMake2DArray([][]string{{"name", "age"}, {"Alice", "30"}})

		result := fn(csvMakeStr(outPath), data)
		values := csvGetReturnValues(t, result)

		if values[1] != object.NIL {
			t.Errorf("expected nil error, got %v", values[1])
			return
		}

		boolVal, ok := values[0].(*object.Boolean)
		if !ok {
			t.Fatalf("expected Boolean, got %T", values[0])
		}
		if !boolVal.Value {
			t.Error("expected true, got false")
		}

		// Verify file contents
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		expected := "name,age\nAlice,30\n"
		if string(content) != expected {
			t.Errorf("file content = %q, want %q", string(content), expected)
		}
	})

	t.Run("with delimiter option", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "output.tsv")
		data := csvMake2DArray([][]string{{"name", "age"}, {"Alice", "30"}})

		opts := &object.Map{
			Pairs: []*object.MapPair{
				{
					Key:   &object.String{Value: "delimiter"},
					Value: &object.String{Value: "\t"},
				},
			},
		}

		result := fn(csvMakeStr(outPath), data, opts)
		values := csvGetReturnValues(t, result)

		if values[1] != object.NIL {
			t.Errorf("expected nil error, got %v", values[1])
			return
		}

		// Verify file contents
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}
		expected := "name\tage\nAlice\t30\n"
		if string(content) != expected {
			t.Errorf("file content = %q, want %q", string(content), expected)
		}
	})

	// Test wrong argument count
	t.Run("wrong argument count", func(t *testing.T) {
		result := fn()
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong argument count")
		}
	})

	// Test wrong type
	t.Run("wrong path type", func(t *testing.T) {
		result := fn(makeInt(42), csvMake2DArray([][]string{}))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong path type")
		}
	})

	t.Run("wrong data type", func(t *testing.T) {
		outPath := filepath.Join(tmpDir, "bad.csv")
		result := fn(csvMakeStr(outPath), csvMakeStr("not an array"))
		if _, ok := result.(*object.Error); !ok {
			t.Error("expected error for wrong data type")
		}
	})
}
