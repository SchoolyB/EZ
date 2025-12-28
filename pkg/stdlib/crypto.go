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

	"github.com/marshallburns/ez/pkg/object"
)

// CryptoBuiltins contains the crypto module functions
var CryptoBuiltins = map[string]*object.Builtin{
	// crypto.sha256(data) - returns SHA-256 hash as lowercase hex string
	"crypto.sha256": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "crypto.sha256() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "crypto.sha256() requires a string argument"}
			}

			hash := sha256.Sum256([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// crypto.sha512(data) - returns SHA-512 hash as lowercase hex string
	"crypto.sha512": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "crypto.sha512() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "crypto.sha512() requires a string argument"}
			}

			hash := sha512.Sum512([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// crypto.md5(data) - returns MD5 hash as lowercase hex string
	// Note: MD5 is cryptographically broken, use only for checksums/legacy
	"crypto.md5": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "crypto.md5() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "crypto.md5() requires a string argument"}
			}

			hash := md5.Sum([]byte(str.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	},

	// crypto.random_bytes(length) - returns cryptographically secure random bytes
	"crypto.random_bytes": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "crypto.random_bytes() takes exactly 1 argument"}
			}

			length, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "crypto.random_bytes() requires an integer argument"}
			}

			n := length.Value.Int64()
			if n < 0 {
				return &object.Error{Code: "E7011", Message: "crypto.random_bytes() length cannot be negative"}
			}
			if n == 0 {
				return &object.Array{Elements: []object.Object{}, Mutable: true}
			}

			bytes := make([]byte, n)
			_, err := rand.Read(bytes)
			if err != nil {
				return &object.Error{Code: "E15001", Message: fmt.Sprintf("crypto.random_bytes() failed: %s", err.Error())}
			}

			elements := make([]object.Object, n)
			for i, b := range bytes {
				elements[i] = &object.Byte{Value: b}
			}

			return &object.Array{Elements: elements, Mutable: true, ElementType: "byte"}
		},
	},

	// crypto.random_hex(length) - returns cryptographically secure random hex string
	// length is the number of bytes, output will be 2*length characters
	"crypto.random_hex": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "crypto.random_hex() takes exactly 1 argument"}
			}

			length, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "crypto.random_hex() requires an integer argument"}
			}

			n := length.Value.Int64()
			if n < 0 {
				return &object.Error{Code: "E7011", Message: "crypto.random_hex() length cannot be negative"}
			}
			if n == 0 {
				return &object.String{Value: ""}
			}

			bytes := make([]byte, n)
			_, err := rand.Read(bytes)
			if err != nil {
				return &object.Error{Code: "E15001", Message: fmt.Sprintf("crypto.random_hex() failed: %s", err.Error())}
			}

			return &object.String{Value: hex.EncodeToString(bytes)}
		},
	},
}
