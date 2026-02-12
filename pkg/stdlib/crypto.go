package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// CryptoBuiltins contains the crypto module functions
var CryptoBuiltins = map[string]*object.Builtin{
	// sha256 computes the SHA-256 hash of a string.
	// Takes a string. Returns lowercase hex-encoded hash.
	"crypto.sha256": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("crypto.sha256()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("crypto.sha256()"), errors.TypeExpected("string"))}
			}

			hash := sha256.Sum256([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// sha512 computes the SHA-512 hash of a string.
	// Takes a string. Returns lowercase hex-encoded hash.
	"crypto.sha512": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("crypto.sha512()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("crypto.sha512()"), errors.TypeExpected("string"))}
			}

			hash := sha512.Sum512([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// md5 computes the MD5 hash of a string (legacy, not secure).
	// Takes a string. Returns lowercase hex-encoded hash.
	"crypto.md5": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("crypto.md5()"))}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: fmt.Sprintf("%s requires a %s argument", errors.Ident("crypto.md5()"), errors.TypeExpected("string"))}
			}

			hash := md5.Sum([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// random_bytes generates cryptographically secure random bytes.
	// Takes length as integer. Returns byte array.
	"crypto.random_bytes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("crypto.random_bytes()"))}
			}

			length, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s argument", errors.Ident("crypto.random_bytes()"), errors.TypeExpected("integer"))}
			}

			n := length.Value.Int64()
			if n < 0 {
				return &object.Error{Code: "E7011", Message: fmt.Sprintf("%s length cannot be negative", errors.Ident("crypto.random_bytes()"))}
			}
			if n == 0 {
				return &object.Array{Elements: []object.Object{}, Mutable: true, ElementType: "byte"}
			}

			bytes := make([]byte, n)
			_, err := rand.Read(bytes)
			if err != nil {
				return &object.Error{Code: "E15001", Message: fmt.Sprintf("%s failed: %s", errors.Ident("crypto.random_bytes()"), err.Error())}
			}

			elements := make([]object.Object, n)
			for i, b := range bytes {
				elements[i] = &object.Byte{Value: b}
			}

			return &object.Array{Elements: elements, Mutable: true, ElementType: "byte"}
		},
	},

	// random_hex generates cryptographically secure random hex string.
	// Takes byte count. Returns hex string (2*length characters).
	"crypto.random_hex": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("crypto.random_hex()"))}
			}

			length, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: fmt.Sprintf("%s requires an %s argument", errors.Ident("crypto.random_hex()"), errors.TypeExpected("integer"))}
			}

			n := length.Value.Int64()
			if n < 0 {
				return &object.Error{Code: "E7011", Message: fmt.Sprintf("%s length cannot be negative", errors.Ident("crypto.random_hex()"))}
			}
			if n == 0 {
				return &object.String{Value: ""}
			}

			bytes := make([]byte, n)
			_, err := rand.Read(bytes)
			if err != nil {
				return &object.Error{Code: "E15001", Message: fmt.Sprintf("%s failed: %s", errors.Ident("crypto.random_hex()"), err.Error())}
			}

			return &object.String{Value: hex.EncodeToString(bytes)}
		},
	},
}
