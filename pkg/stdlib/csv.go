package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// CsvBuiltins contains the csv module functions for reading and writing CSV data
var CsvBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// String Parsing and Stringifying
	// ============================================================================

	// parse parses a CSV string into a 2D array of strings.
	// Takes CSV string. Returns ([[string]], Error) tuple.
	"csv.parse": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (csv_string)", errors.Ident("csv.parse()"))}
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("csv.parse()"), errors.TypeExpected("string"))}
			}

			reader := csv.NewReader(strings.NewReader(str.Value))
			records, err := reader.ReadAll()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E14001", fmt.Sprintf("%s failed: %s", errors.Ident("csv.parse()"), err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				csvRowsToArray(records),
				object.NIL,
			}}
		},
	},

	// stringify converts a 2D array to a CSV string.
	// Takes [[string]] data. Returns (string, Error) tuple.
	"csv.stringify": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (data)", errors.Ident("csv.stringify()"))}
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires an %s argument", errors.Ident("csv.stringify()"), errors.TypeExpected("array"))}
			}

			rows, convErr := arrayToCsvRows(arr)
			if convErr != nil {
				return convErr
			}

			var sb strings.Builder
			writer := csv.NewWriter(&sb)
			err := writer.WriteAll(rows)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E14002", fmt.Sprintf("%s failed: %s", errors.Ident("csv.stringify()"), err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: sb.String()},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// File Reading
	// ============================================================================

	// read reads a CSV file and returns a 2D array of strings.
	// Takes file path and optional options map. Returns ([[string]], Error) tuple.
	// Options: delimiter (string), skip_empty (bool)
	"csv.read": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 1-2 arguments (path, [options])", errors.Ident("csv.read()"))}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path as first argument", errors.Ident("csv.read()"), errors.TypeExpected("string"))}
			}

			// Validate path
			if err := validatePath(path.Value, "csv.read()"); err != nil {
				return err
			}

			// Parse options
			delimiter := ','
			skipEmpty := false
			if len(args) == 2 {
				opts, ok := args[1].(*object.Map)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s options must be a %s", errors.Ident("csv.read()"), errors.TypeExpected("map"))}
				}
				delimiter, skipEmpty = getCsvReadOptions(opts)
			}

			// Read file
			content, err := os.ReadFile(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read csv")
			}

			reader := csv.NewReader(strings.NewReader(string(content)))
			reader.Comma = delimiter
			reader.FieldsPerRecord = -1 // Allow variable field counts

			records, err := reader.ReadAll()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E14001", fmt.Sprintf("%s failed to parse: %s", errors.Ident("csv.read()"), err.Error())),
				}}
			}

			// Filter empty rows if requested
			if skipEmpty {
				records = filterEmptyRows(records)
			}

			return &object.ReturnValue{Values: []object.Object{
				csvRowsToArray(records),
				object.NIL,
			}}
		},
	},

	// headers reads just the first row (headers) of a CSV file.
	// Takes file path. Returns ([string], Error) tuple.
	"csv.headers": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument (path)", errors.Ident("csv.headers()"))}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path", errors.Ident("csv.headers()"), errors.TypeExpected("string"))}
			}

			// Validate path
			if err := validatePath(path.Value, "csv.headers()"); err != nil {
				return err
			}

			// Open file
			file, err := os.Open(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "read csv headers")
			}
			defer file.Close()

			reader := csv.NewReader(file)
			record, err := reader.Read()
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E14001", fmt.Sprintf("%s failed: %s", errors.Ident("csv.headers()"), err.Error())),
				}}
			}

			// Convert to string array
			elements := make([]object.Object, len(record))
			for i, field := range record {
				elements[i] = &object.String{Value: field}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Array{Elements: elements, ElementType: "string"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// File Writing
	// ============================================================================

	// write writes a 2D array to a CSV file.
	// Takes path, data, and optional options map. Returns (bool, Error) tuple.
	// Options: delimiter (string), quote_all (bool)
	"csv.write": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes 2-3 arguments (path, data, [options])", errors.Ident("csv.write()"))}
			}
			path, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s path as first argument", errors.Ident("csv.write()"), errors.TypeExpected("string"))}
			}
			arr, ok := args[1].(*object.Array)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires an %s as second argument", errors.Ident("csv.write()"), errors.TypeExpected("array"))}
			}

			// Validate path
			if err := validatePath(path.Value, "csv.write()"); err != nil {
				return err
			}

			// Parse options
			delimiter := ','
			quoteAll := false
			if len(args) == 3 {
				opts, ok := args[2].(*object.Map)
				if !ok {
					return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s options must be a %s", errors.Ident("csv.write()"), errors.TypeExpected("map"))}
				}
				delimiter, quoteAll = getCsvWriteOptions(opts)
			}

			rows, convErr := arrayToCsvRows(arr)
			if convErr != nil {
				return convErr
			}

			// Create file
			file, err := os.Create(path.Value)
			if err != nil {
				return CreateStdlibErrorResult(err, "create csv file")
			}
			defer file.Close()

			writer := csv.NewWriter(file)
			writer.Comma = delimiter
			if quoteAll {
				writer.UseCRLF = false
			}

			err = writer.WriteAll(rows)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.FALSE,
					CreateStdlibError("E14002", fmt.Sprintf("%s failed: %s", errors.Ident("csv.write()"), err.Error())),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				object.TRUE,
				object.NIL,
			}}
		},
	},
}

// ============================================================================
// Helper Functions
// ============================================================================

// csvRowsToArray converts Go [][]string to EZ [[string]]
func csvRowsToArray(rows [][]string) *object.Array {
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

// arrayToCsvRows converts EZ [[string]] to Go [][]string
func arrayToCsvRows(arr *object.Array) ([][]string, *object.Error) {
	rows := make([][]string, len(arr.Elements))
	for i, elem := range arr.Elements {
		rowArr, ok := elem.(*object.Array)
		if !ok {
			return nil, &object.Error{
				Code:    "E7003",
				Message: fmt.Sprintf("csv data row %d must be an %s", i, errors.TypeExpected("array")),
			}
		}
		row := make([]string, len(rowArr.Elements))
		for j, field := range rowArr.Elements {
			str, ok := field.(*object.String)
			if !ok {
				return nil, &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("csv data field at row %d, column %d must be a %s", i, j, errors.TypeExpected("string")),
				}
			}
			row[j] = str.Value
		}
		rows[i] = row
	}
	return rows, nil
}

// getCsvReadOptions extracts read options from a map
func getCsvReadOptions(opts *object.Map) (delimiter rune, skipEmpty bool) {
	delimiter = ','
	skipEmpty = false

	for _, pair := range opts.Pairs {
		keyStr, ok := pair.Key.(*object.String)
		if !ok {
			continue
		}

		switch keyStr.Value {
		case "delimiter":
			if valStr, ok := pair.Value.(*object.String); ok && len(valStr.Value) > 0 {
				delimiter = rune(valStr.Value[0])
			}
		case "skip_empty":
			if valBool, ok := pair.Value.(*object.Boolean); ok {
				skipEmpty = valBool.Value
			}
		}
	}

	return delimiter, skipEmpty
}

// getCsvWriteOptions extracts write options from a map
func getCsvWriteOptions(opts *object.Map) (delimiter rune, quoteAll bool) {
	delimiter = ','
	quoteAll = false

	for _, pair := range opts.Pairs {
		keyStr, ok := pair.Key.(*object.String)
		if !ok {
			continue
		}

		switch keyStr.Value {
		case "delimiter":
			if valStr, ok := pair.Value.(*object.String); ok && len(valStr.Value) > 0 {
				delimiter = rune(valStr.Value[0])
			}
		case "quote_all":
			if valBool, ok := pair.Value.(*object.Boolean); ok {
				quoteAll = valBool.Value
			}
		}
	}

	return delimiter, quoteAll
}

// filterEmptyRows removes rows where all fields are empty strings
func filterEmptyRows(rows [][]string) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		isEmpty := true
		for _, field := range row {
			if field != "" {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			result = append(result, row)
		}
	}
	return result
}
