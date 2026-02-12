package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// EncodingBuiltins contains the encoding module functions
var EncodingBuiltins = map[string]*object.Builtin{
	// base64_encode encodes a string to base64 format.
	// Takes a string. Returns base64-encoded string.
	"encoding.base64_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.base64_encode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.base64_encode()"), errors.TypeExpected("string"))}
			}

			encoded := base64.StdEncoding.EncodeToString([]byte(str.Value))
			return &object.String{Value: encoded}
		},
	},

	// base64_decode decodes a base64-encoded string.
	// Takes base64 string. Returns (string, Error) tuple.
	"encoding.base64_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.base64_decode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.base64_decode()"), errors.TypeExpected("string"))}
			}

			decoded, err := base64.StdEncoding.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E16001", "invalid base64 input: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(decoded)},
				object.NIL,
			}}
		},
	},

	// hex_encode encodes a string to hexadecimal format.
	// Takes a string. Returns hex-encoded string.
	"encoding.hex_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.hex_encode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.hex_encode()"), errors.TypeExpected("string"))}
			}

			encoded := hex.EncodeToString([]byte(str.Value))
			return &object.String{Value: encoded}
		},
	},

	// hex_decode decodes a hexadecimal-encoded string.
	// Takes hex string. Returns (string, Error) tuple.
	"encoding.hex_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.hex_decode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.hex_decode()"), errors.TypeExpected("string"))}
			}

			decoded, err := hex.DecodeString(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E16002", "invalid hex input: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: string(decoded)},
				object.NIL,
			}}
		},
	},

	// url_encode percent-encodes a string for use in URLs.
	// Takes a string. Returns URL-encoded string.
	"encoding.url_encode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.url_encode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.url_encode()"), errors.TypeExpected("string"))}
			}

			encoded := url.QueryEscape(str.Value)
			return &object.String{Value: encoded}
		},
	},

	// url_decode decodes a URL percent-encoded string.
	// Takes URL-encoded string. Returns (string, Error) tuple.
	"encoding.url_decode": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("encoding.url_decode()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("encoding.url_decode()"), errors.TypeExpected("string"))}
			}

			decoded, err := url.QueryUnescape(str.Value)
			if err != nil {
				return &object.ReturnValue{Values: []object.Object{
					&object.String{Value: ""},
					CreateStdlibError("E16003", "invalid URL encoding: "+err.Error()),
				}}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.String{Value: decoded},
				object.NIL,
			}}
		},
	},
}
