package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/base64"
	"encoding/hex"
	"net/url"

	"github.com/marshallburns/ez/pkg/object"
)

// createEncodingError creates an error struct for encoding operations
func createEncodingError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code":    &object.String{Value: code},
		},
	}
}

// EncodingBuiltins contains the encoding module functions
var EncodingBuiltins = map[string]*object.Builtin{
	// encoding.base64_encode(data) - encodes string to base64
	"encoding.base64_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.base64_encode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.base64_encode() requires a string argument"}
			}

			encoded := base64.StdEncoding.EncodeToString([]byte(str.Value))
			return &object.String{Value: encoded}
		},
	},

	// encoding.base64_decode(data) - decodes base64 to string
	"encoding.base64_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.base64_decode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.base64_decode() requires a string argument"}
			}

			decoded, err := base64.StdEncoding.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createEncodingError("E16001", "invalid base64 input: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(decoded)},
				object.NIL,
			}}
		},
	},

	// encoding.hex_encode(data) - encodes string to hex
	"encoding.hex_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.hex_encode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.hex_encode() requires a string argument"}
			}

			encoded := hex.EncodeToString([]byte(str.Value))
			return &object.String{Value: encoded}
		},
	},

	// encoding.hex_decode(data) - decodes hex to string
	"encoding.hex_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.hex_decode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.hex_decode() requires a string argument"}
			}

			decoded, err := hex.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createEncodingError("E16002", "invalid hex input: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(decoded)},
				object.NIL,
			}}
		},
	},

	// encoding.url_encode(data) - URL percent-encodes a string
	"encoding.url_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.url_encode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.url_encode() requires a string argument"}
			}

			encoded := url.QueryEscape(str.Value)
			return &object.String{Value: encoded}
		},
	},

	// encoding.url_decode(data) - decodes URL percent-encoding
	"encoding.url_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "encoding.url_decode() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "encoding.url_decode() requires a string argument"}
			}

			decoded, err := url.QueryUnescape(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					createEncodingError("E16003", "invalid URL encoding: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: decoded},
				object.NIL,
			}}
		},
	},
}
