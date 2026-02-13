package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"strings"

	"github.com/marshallburns/ez/pkg/object"
)

// =============================================================================
// String Building Utilities
// =============================================================================

// objectDisplayValue returns the display string for an object.
// For strings, this returns the raw value without quotes.
// For all other types, this returns the Inspect() representation.
func objectDisplayValue(obj object.Object) string {
	if s, ok := obj.(*object.String); ok {
		return s.Value
	}
	return obj.Inspect()
}

// JoinObjects joins object display values with a separator efficiently.
func JoinObjects(elements []object.Object, sep string) string {
	if len(elements) == 0 {
		return ""
	}
	if len(elements) == 1 {
		return objectDisplayValue(elements[0])
	}

	var sb strings.Builder
	sb.WriteString(objectDisplayValue(elements[0]))
	for _, el := range elements[1:] {
		sb.WriteString(sep)
		sb.WriteString(objectDisplayValue(el))
	}
	return sb.String()
}
