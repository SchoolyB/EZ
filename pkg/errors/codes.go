package errors

// ErrorCode represents a specific error type
type ErrorCode struct {
	Code        string
	Name        string
	Description string
}

// Error code definitions
var (
	// Parse Errors (E1xxx)
	E1001 = ErrorCode{"E1001", "unexpected-token", "Unexpected token"}
	E1002 = ErrorCode{"E1002", "missing-token", "Expected token not found"}
	E1003 = ErrorCode{"E1003", "invalid-syntax", "Invalid syntax"}
	E1004 = ErrorCode{"E1004", "unclosed-brace", "Unclosed brace"}
	E1005 = ErrorCode{"E1005", "unclosed-string", "Unclosed string literal"}
	E1006 = ErrorCode{"E1006", "invalid-number", "Invalid number format"}
	E1007 = ErrorCode{"E1007", "missing-type", "Missing type annotation"}
	E1008 = ErrorCode{"E1008", "invalid-assignment", "Invalid assignment target"}

	// Type Errors (E2xxx)
	E2001 = ErrorCode{"E2001", "type-mismatch", "Type mismatch"}
	E2002 = ErrorCode{"E2002", "invalid-operation", "Invalid operation for type"}
	E2003 = ErrorCode{"E2003", "cannot-convert", "Cannot convert between types"}
	E2004 = ErrorCode{"E2004", "wrong-arg-type", "Wrong argument type"}
	E2005 = ErrorCode{"E2005", "wrong-return-type", "Wrong return type"}

	// Reference Errors (E3xxx)
	E3001 = ErrorCode{"E3001", "undefined-variable", "Undefined variable"}
	E3002 = ErrorCode{"E3002", "undefined-function", "Undefined function"}
	E3003 = ErrorCode{"E3003", "undefined-field", "Undefined struct field"}
	E3004 = ErrorCode{"E3004", "undefined-module", "Module not found"}
	E3005 = ErrorCode{"E3005", "not-imported", "Module not imported"}

	// Runtime Errors (E4xxx)
	E4001 = ErrorCode{"E4001", "division-by-zero", "Division by zero"}
	E4002 = ErrorCode{"E4002", "index-out-of-bounds", "Index out of bounds"}
	E4003 = ErrorCode{"E4003", "nil-reference", "Nil reference"}
	E4004 = ErrorCode{"E4004", "wrong-arg-count", "Wrong number of arguments"}
	E4005 = ErrorCode{"E4005", "immutable-assign", "Cannot assign to immutable variable"}
	E4006 = ErrorCode{"E4006", "return-mismatch", "Return value count mismatch"}

	// Import Errors (E5xxx)
	E5001 = ErrorCode{"E5001", "circular-import", "Circular import detected"}
	E5002 = ErrorCode{"E5002", "file-not-found", "File not found"}
	E5003 = ErrorCode{"E5003", "invalid-module", "Invalid module"}
)

// Warning code definitions
var (
	// Code Style Warnings (W1xxx)
	W1001 = ErrorCode{"W1001", "unused-variable", "Unused variable"}
	W1002 = ErrorCode{"W1002", "unused-import", "Unused import"}
	W1003 = ErrorCode{"W1003", "unused-function", "Unused function"}

	// Potential Bug Warnings (W2xxx)
	W2001 = ErrorCode{"W2001", "unreachable-code", "Unreachable code"}
	W2002 = ErrorCode{"W2002", "shadowed-variable", "Variable shadows outer scope"}
)
