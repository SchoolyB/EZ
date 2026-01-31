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

// JoinObjects joins object Inspect() values with a separator efficiently.
func JoinObjects(elements []object.Object, sep string) string {
	if len(elements) == 0 {
		return ""
	}
	if len(elements) == 1 {
		return elements[0].Inspect()
	}

	var sb strings.Builder
	sb.WriteString(elements[0].Inspect())
	for _, el := range elements[1:] {
		sb.WriteString(sep)
		sb.WriteString(el.Inspect())
	}
	return sb.String()
}
