package errors

// ErrorCode represents a specific error type
type ErrorCode struct {
	Code        string
	Name        string
	Description string
}

// Error code definitions
// Description field is used as caret text - keep it short and descriptive
var (
	// Parse Errors (E1xxx)
	E1001 = ErrorCode{"E1001", "unexpected-token", "unexpected token here"}
	E1002 = ErrorCode{"E1002", "missing-token", "expected token not found"}
	E1003 = ErrorCode{"E1003", "invalid-syntax", "invalid syntax"}
	E1004 = ErrorCode{"E1004", "unclosed-brace", "unclosed brace"}
	E1005 = ErrorCode{"E1005", "unclosed-string", "string literal not closed"}
	E1006 = ErrorCode{"E1006", "invalid-number", "invalid number format"}
	E1007 = ErrorCode{"E1007", "missing-type", "type annotation required"}
	E1008 = ErrorCode{"E1008", "invalid-assignment", "cannot assign to this"}
	E1009 = ErrorCode{"E1009", "reserved-name", "this name is reserved"}
	E1010 = ErrorCode{"E1010", "duplicate-name", "name already declared"}
	E1011 = ErrorCode{"E1011", "nested-function", "functions cannot be declared inside functions"}

	// Type Errors (E2xxx)
	E2001 = ErrorCode{"E2001", "type-mismatch", "types do not match"}
	E2002 = ErrorCode{"E2002", "invalid-operation", "invalid operation for this type"}
	E2003 = ErrorCode{"E2003", "cannot-convert", "cannot convert types"}
	E2004 = ErrorCode{"E2004", "wrong-arg-type", "wrong argument type"}
	E2005 = ErrorCode{"E2005", "wrong-return-type", "wrong return type"}
	E2006 = ErrorCode{"E2006", "undefined-type", "type not defined"}
	E2007 = ErrorCode{"E2007", "type-mismatch-binary", "incompatible types for operator"}

	// Reference Errors (E3xxx)
	E3001 = ErrorCode{"E3001", "undefined-variable", "not found in this scope"}
	E3002 = ErrorCode{"E3002", "undefined-function", "function not defined"}
	E3003 = ErrorCode{"E3003", "undefined-field", "field does not exist"}
	E3004 = ErrorCode{"E3004", "undefined-module", "module not found"}
	E3005 = ErrorCode{"E3005", "not-imported", "module not imported"}

	// Runtime Errors (E4xxx)
	E4001 = ErrorCode{"E4001", "division-by-zero", "cannot divide by zero"}
	E4002 = ErrorCode{"E4002", "index-out-of-bounds", "index out of bounds"}
	E4003 = ErrorCode{"E4003", "nil-reference", "nil reference"}
	E4004 = ErrorCode{"E4004", "wrong-arg-count", "wrong number of arguments"}
	E4005 = ErrorCode{"E4005", "immutable-assign", "cannot assign to const"}
	E4006 = ErrorCode{"E4006", "return-mismatch", "wrong number of return values"}
	E4007 = ErrorCode{"E4007", "uncaptured-return", "return value must be used"}

	// Import Errors (E5xxx)
	E5001 = ErrorCode{"E5001", "circular-import", "circular import detected"}
	E5002 = ErrorCode{"E5002", "file-not-found", "file not found"}
	E5003 = ErrorCode{"E5003", "invalid-module", "invalid module"}
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
