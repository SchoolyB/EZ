package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/marshallburns/ez/pkg/object"
)

// JsonBuiltins contains the json module functions
var JsonBuiltins = map[string]*object.Builtin{
	// json.encode(value) -> (string, error)
	// Serializes an EZ value to a JSON string
	"json.encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "json.encode() takes exactly 1 argument"}
			}

			result, err := encodeToJSON(args[0], make(map[uintptr]bool))
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError(err.code, err.message),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: result},
				object.NIL,
			}}
		},
	},

	// json.decode(text string) -> (any, error)
	// json.decode(text string, Type) -> (Type, error)
	// Parses a JSON string into EZ types.
	// With 1 arg: returns dynamic types (maps, arrays, primitives)
	// With 2 args: returns a typed struct instance
	"json.decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Code: "E7001", Message: "json.decode() takes 1 or 2 arguments (text) or (text, Type)"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "json.decode() requires a string as first argument"}
			}

			// 1-arg form: return dynamic types
			if len(args) == 1 {
				result, err := decodeFromJSON(str.Value)
				if err != nil {
					return &object.ReturnValue{Values: []object.Object{
						object.NIL,
						CreateStdlibError("E13001", fmt.Sprintf("invalid JSON syntax: %s", err.Error())),
					}}
				}

				return &object.ReturnValue{Values: []object.Object{
					result,
					object.NIL,
				}}
			}

			// 2-arg form: return typed struct
			typeVal, ok := args[1].(*object.TypeValue)
			if !ok {
				return &object.Error{Code: "E7003", Message: "json.decode() second argument must be a type"}
			}

			if typeVal.Def == nil {
				return &object.Error{Code: "E13002", Message: fmt.Sprintf("cannot decode JSON to primitive type '%s'", typeVal.TypeName)}
			}

			result, jsonErr := decodeToStruct(str.Value, typeVal.Def)
			if jsonErr != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError(jsonErr.code, jsonErr.message),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				result,
				object.NIL,
			}}
		},
	},

	// json.pretty(value, indent string) -> (string, error)
	// Serializes an EZ value to a formatted JSON string with indentation
	"json.pretty": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "json.pretty() takes exactly 2 arguments (value, indent)"}
			}

			indent, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "json.pretty() requires a string indent argument"}
			}

			result, err := encodeToPrettyJSON(args[0], indent.Value, make(map[uintptr]bool))
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError(err.code, err.message),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: result},
				object.NIL,
			}}
		},
	},

	// json.is_valid(text string) -> bool
	// Checks if a string is valid JSON (pure function, no error tuple)
	"json.is_valid": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "json.is_valid() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "json.is_valid() requires a string argument"}
			}

			if json.Valid([]byte(str.Value)) {
				return object.TRUE
			}
			return object.FALSE
		},
	},
}

// jsonError represents a JSON encoding/decoding error
type jsonError struct {
	code    string
	message string
}


// encodeToJSON converts an EZ object to a JSON string
func encodeToJSON(obj object.Object, seen map[uintptr]bool) (string, *jsonError) {
	value, err := objectToGoValue(obj, seen)
	if err != nil {
		return "", err
	}

	bytes, jsonErr := json.Marshal(value)
	if jsonErr != nil {
		return "", &jsonError{code: "E13002", message: fmt.Sprintf("failed to encode: %s", jsonErr.Error())}
	}

	return string(bytes), nil
}

// encodeToPrettyJSON converts an EZ object to a formatted JSON string
func encodeToPrettyJSON(obj object.Object, indent string, seen map[uintptr]bool) (string, *jsonError) {
	value, err := objectToGoValue(obj, seen)
	if err != nil {
		return "", err
	}

	bytes, jsonErr := json.MarshalIndent(value, "", indent)
	if jsonErr != nil {
		return "", &jsonError{code: "E13002", message: fmt.Sprintf("failed to encode: %s", jsonErr.Error())}
	}

	return string(bytes), nil
}

// objectToGoValue converts an EZ object to a Go value suitable for json.Marshal
func objectToGoValue(obj object.Object, seen map[uintptr]bool) (interface{}, *jsonError) {
	switch v := obj.(type) {
	case *object.Integer:
		// Check if value fits in int64, otherwise encode as string to preserve precision
		if v.Value.IsInt64() {
			return v.Value.Int64(), nil
		}
		// Large integers are encoded as strings to preserve precision
		return v.Value.String(), nil

	case *object.Float:
		return v.Value, nil

	case *object.String:
		return v.Value, nil

	case *object.Boolean:
		return v.Value, nil

	case *object.Nil:
		return nil, nil

	case *object.Char:
		return string(v.Value), nil

	case *object.Byte:
		return int(v.Value), nil

	case *object.Array:
		arr := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			val, err := objectToGoValue(elem, seen)
			if err != nil {
				return nil, err
			}
			arr[i] = val
		}
		return arr, nil

	case *object.Map:
		// Check for non-string keys
		m := make(map[string]interface{})
		for _, pair := range v.Pairs {
			keyStr, ok := pair.Key.(*object.String)
			if !ok {
				return nil, &jsonError{
					code:    "E13003",
					message: fmt.Sprintf("JSON object keys must be strings, got %s", getEZTypeName(pair.Key)),
				}
			}
			val, err := objectToGoValue(pair.Value, seen)
			if err != nil {
				return nil, err
			}
			m[keyStr.Value] = val
		}
		return m, nil

	case *object.Struct:
		m := make(map[string]interface{})
		for key, val := range v.Fields {
			goVal, err := objectToGoValue(val, seen)
			if err != nil {
				return nil, err
			}
			switch tag := v.FieldTags[key].(type) {
			case *object.JSONTag:
				if tag.Ignore {
					break
				}
				if tag.OmitEmpty && val.Type() == object.NIL_OBJ {
					break
				}
				if tag.EncodeAsString {
					switch v := goVal.(type) {
					case *object.Integer:
						m[tag.Name] = v.Inspect()
					case *object.Float:
						m[tag.Name] = v.Inspect()
					default:
						m[tag.Name] = goVal
					}
				} else {
					m[tag.Name] = goVal
				}
			default:
				m[key] = goVal
			}
		}
		return m, nil

	case *object.Function, *object.Builtin:
		return nil, &jsonError{
			code:    "E13002",
			message: "functions cannot be converted to JSON",
		}

	case *object.Error:
		return nil, &jsonError{
			code:    "E13002",
			message: "error objects cannot be converted to JSON",
		}

	default:
		return nil, &jsonError{
			code:    "E13002",
			message: fmt.Sprintf("type %s cannot be converted to JSON", getEZTypeName(obj)),
		}
	}
}

// decodeFromJSON parses a JSON string into an EZ object
func decodeFromJSON(jsonStr string) (object.Object, error) {
	var value interface{}
	if err := json.Unmarshal([]byte(jsonStr), &value); err != nil {
		return nil, err
	}
	return goValueToObject(value), nil
}

// decodeToStruct parses a JSON string into a typed EZ struct
func decodeToStruct(jsonStr string, structDef *object.StructDef) (*object.Struct, *jsonError) {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
		return nil, &jsonError{code: "E13001", message: fmt.Sprintf("invalid JSON syntax: %s", err.Error())}
	}

	// Create struct with fields from JSON
	fields := make(map[string]object.Object)
	for fieldName, fieldType := range structDef.Fields {
		switch tag := structDef.FieldTags[fieldName].(type) {
		case *object.JSONTag:
			if tag.Ignore {
				fields[fieldName] = zeroValueForType(fieldType)
				break
			}

			if jsonVal, exists := jsonMap[tag.Name]; exists {
				fields[fieldName] = convertToTypedValue(jsonVal, fieldType)
			} else if !exists {
				fields[fieldName] = zeroValueForType(fieldType)
			}
		default:
			if jsonVal, exists := jsonMap[fieldName]; exists {
				fields[fieldName] = convertToTypedValue(jsonVal, fieldType)
			} else {
				// Set zero value for missing field
				fields[fieldName] = zeroValueForType(fieldType)
			}
		}
	}

	return &object.Struct{
		TypeName: structDef.Name,
		Fields:   fields,
	}, nil
}

// convertToTypedValue converts a JSON value to an EZ object of the specified type
func convertToTypedValue(value interface{}, targetType string) object.Object {
	switch targetType {
	case "int":
		switch v := value.(type) {
		case float64:
			return &object.Integer{Value: big.NewInt(int64(v))}
		case string:
			// Try to parse string as int
			if i, ok := new(big.Int).SetString(v, 10); ok {
				return &object.Integer{Value: i}
			}
		}
		return &object.Integer{Value: big.NewInt(0)}

	case "float":
		switch v := value.(type) {
		case float64:
			return &object.Float{Value: v, DeclaredType: "float"}
		case string:
			// Strings stay as strings, can't auto-convert
		}
		return &object.Float{Value: 0.0, DeclaredType: "float"}

	case "string":
		switch v := value.(type) {
		case string:
			return &object.String{Value: v}
		case float64:
			// Numbers can be converted to strings
			if v == float64(int64(v)) {
				return &object.String{Value: fmt.Sprintf("%d", int64(v))}
			}
			return &object.String{Value: fmt.Sprintf("%v", v)}
		case bool:
			if v {
				return &object.String{Value: "true"}
			}
			return &object.String{Value: "false"}
		}
		return &object.String{Value: ""}

	case "bool":
		switch v := value.(type) {
		case bool:
			if v {
				return object.TRUE
			}
			return object.FALSE
		}
		return object.FALSE

	default:
		// For arrays and other complex types, use dynamic conversion
		return goValueToObject(value)
	}
}

// zeroValueForType returns the zero value for an EZ type
func zeroValueForType(typeName string) object.Object {
	switch typeName {
	case "int":
		return &object.Integer{Value: big.NewInt(0)}
	case "float":
		return &object.Float{Value: 0.0, DeclaredType: "float"}
	case "string":
		return &object.String{Value: ""}
	case "bool":
		return object.FALSE
	case "char":
		return &object.Char{Value: 0}
	case "byte":
		return &object.Byte{Value: 0}
	default:
		// For complex types (arrays, maps, structs), return nil
		return object.NIL
	}
}

// goValueToObject converts a Go value from json.Unmarshal to an EZ object
func goValueToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.NIL

	case bool:
		if v {
			return object.TRUE
		}
		return object.FALSE

	case float64:
		// JSON numbers are always float64, check if it's actually an integer
		if v == float64(int64(v)) {
			return &object.Integer{Value: big.NewInt(int64(v))}
		}
		return &object.Float{Value: v, DeclaredType: "float"}

	case string:
		return &object.String{Value: v}

	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, elem := range v {
			elements[i] = goValueToObject(elem)
		}
		return &object.Array{Elements: elements}

	case map[string]interface{}:
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		pairs := make([]*object.MapPair, 0, len(v))
		index := make(map[string]int)
		for i, k := range keys {
			keyObj := &object.String{Value: k}
			valObj := goValueToObject(v[k])
			pairs = append(pairs, &object.MapPair{Key: keyObj, Value: valObj})
			hash, _ := object.HashKey(keyObj)
			index[hash] = i
		}
		return &object.Map{Pairs: pairs, Index: index}

	default:
		// Shouldn't happen with valid JSON
		return object.NIL
	}
}
