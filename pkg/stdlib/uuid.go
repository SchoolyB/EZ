package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// UUID regex pattern for validation (RFC 4122)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUIDBuiltins contains the uuid module functions
var UUIDBuiltins = map[string]*object.Builtin{
	// uuid.create() - generates a new UUID v4 (random)
	"uuid.create": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Code: "E7001", Message: "uuid.create() takes no arguments"}
			}

			uuid, err := generateUUIDv4()
			if err != nil {
				return &object.Error{Code: "E8007", Message: fmt.Sprintf("failed to generate UUID: %s", err.Error())}
			}

			return &object.String{Value: uuid}
		},
	},

	// uuid.create_compact() - generates a new UUID v4 without hyphens
	"uuid.create_compact": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Code: "E7001", Message: "uuid.create_compact() takes no arguments"}
			}

			uuid, err := generateUUIDv4()
			if err != nil {
				return &object.Error{Code: "E8007", Message: fmt.Sprintf("failed to generate UUID: %s", err.Error())}
			}

			// Remove hyphens
			compact := strings.ReplaceAll(uuid, "-", "")
			return &object.String{Value: compact}
		},
	},

	// uuid.is_valid(str) - checks if a string is a valid UUID
	"uuid.is_valid": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "uuid.is_valid() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7002", Message: "uuid.is_valid() requires a string argument"}
			}

			if uuidPattern.MatchString(str.Value) {
				return object.TRUE
			}
			return object.FALSE
		},
	},

	// uuid.NIL - the nil UUID constant
	"uuid.NIL": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{Code: "E7001", Message: "uuid.NIL takes no arguments"}
			}
			return &object.String{Value: "00000000-0000-0000-0000-000000000000"}
		},
	},
}

// generateUUIDv4 generates a random UUID v4 according to RFC 4122
func generateUUIDv4() (string, error) {
	var uuid [16]byte

	// Fill with random bytes
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", err
	}

	// Set version to 4 (random)
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// Set variant to RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	// Format as string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}
