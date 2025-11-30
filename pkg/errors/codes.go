package errors

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

// ErrorCode represents a specific error type
type ErrorCode struct {
	Code        string
	Name        string
	Description string
}

// =============================================================================
// LEXER ERRORS (E1xxx) - Tokenization and lexical analysis errors
// =============================================================================
var (
	E1001 = ErrorCode{"E1001", "illegal-character", "illegal character in source"}
	E1002 = ErrorCode{"E1002", "illegal-or-character", "illegal OR character in source"}
	E1003 = ErrorCode{"E1003", "unclosed-comment", "multi-line comment not closed"}
	E1004 = ErrorCode{"E1004", "unclosed-string", "string literal not closed"}
	E1005 = ErrorCode{"E1005", "unclosed-char", "character literal not closed"}
	E1006 = ErrorCode{"E1006", "invalid-escape-string", "invalid escape sequence in string"}
	E1007 = ErrorCode{"E1007", "invalid-escape-char", "invalid escape sequence in character"}
	E1008 = ErrorCode{"E1008", "empty-char-literal", "character literal is empty"}
	E1009 = ErrorCode{"E1009", "multi-char-literal", "character literal contains multiple characters"}
	E1010 = ErrorCode{"E1010", "invalid-number-format", "invalid numeric literal format"}
	E1011 = ErrorCode{"E1011", "number-consecutive-underscores", "consecutive underscores in number"}
	E1012 = ErrorCode{"E1012", "number-leading-underscore", "number starts with underscore"}
	E1013 = ErrorCode{"E1013", "number-trailing-underscore", "number ends with underscore"}
	E1014 = ErrorCode{"E1014", "number-underscore-before-decimal", "underscore before decimal point"}
	E1015 = ErrorCode{"E1015", "number-underscore-after-decimal", "underscore after decimal point"}
	E1016 = ErrorCode{"E1016", "number-trailing-decimal", "decimal point without digits"}
)

// =============================================================================
// PARSE ERRORS (E2xxx) - Syntax and parsing errors
// =============================================================================
var (
	E2001 = ErrorCode{"E2001", "unexpected-token", "unexpected token encountered"}
	E2002 = ErrorCode{"E2002", "missing-token", "expected token not found"}
	E2003 = ErrorCode{"E2003", "missing-expression", "expected expression"}
	E2004 = ErrorCode{"E2004", "unclosed-brace", "missing closing brace"}
	E2005 = ErrorCode{"E2005", "unclosed-paren", "missing closing parenthesis"}
	E2006 = ErrorCode{"E2006", "unclosed-bracket", "missing closing bracket"}
	E2007 = ErrorCode{"E2007", "unclosed-interpolation", "string interpolation not closed"}
	E2008 = ErrorCode{"E2008", "invalid-assignment-target", "cannot assign to this expression"}
	E2009 = ErrorCode{"E2009", "using-after-declarations", "using statement must come first"}
	E2010 = ErrorCode{"E2010", "using-before-import", "cannot use module before importing"}
	E2011 = ErrorCode{"E2011", "const-requires-value", "const must be initialized"}
	E2012 = ErrorCode{"E2012", "duplicate-parameter", "parameter name already used"}
	E2013 = ErrorCode{"E2013", "duplicate-field", "field name already used"}
	E2014 = ErrorCode{"E2014", "missing-parameter-type", "parameter missing type annotation"}
	E2015 = ErrorCode{"E2015", "missing-return-type", "expected return type after arrow"}
	E2016 = ErrorCode{"E2016", "empty-enum", "enum must have at least one value"}
	E2017 = ErrorCode{"E2017", "trailing-comma-array", "trailing comma in array literal"}
	E2018 = ErrorCode{"E2018", "trailing-comma-call", "trailing comma in function call"}
	E2019 = ErrorCode{"E2019", "nested-function", "function inside function not allowed"}
	E2020 = ErrorCode{"E2020", "reserved-variable-name", "variable name is reserved"}
	E2021 = ErrorCode{"E2021", "reserved-function-name", "function name is reserved"}
	E2022 = ErrorCode{"E2022", "reserved-type-name", "type name is reserved"}
	E2023 = ErrorCode{"E2023", "duplicate-declaration", "name already declared in scope"}
	E2024 = ErrorCode{"E2024", "invalid-type-name", "expected valid type name"}
	E2025 = ErrorCode{"E2025", "invalid-array-size", "array size must be integer"}
	E2026 = ErrorCode{"E2026", "invalid-enum-type", "enum type must be primitive"}
	E2027 = ErrorCode{"E2027", "integer-parse-error", "cannot parse integer literal"}
	E2028 = ErrorCode{"E2028", "float-parse-error", "cannot parse float literal"}
	E2029 = ErrorCode{"E2029", "expected-identifier", "expected identifier"}
	E2030 = ErrorCode{"E2030", "expected-block", "expected block statement"}
	E2031 = ErrorCode{"E2031", "string-enum-requires-values", "string enum needs explicit values"}
)

// =============================================================================
// TYPE ERRORS (E3xxx) - Type system errors
// =============================================================================
var (
	E3001 = ErrorCode{"E3001", "type-mismatch", "types do not match"}
	E3002 = ErrorCode{"E3002", "invalid-operator-for-type", "operator not valid for type"}
	E3003 = ErrorCode{"E3003", "invalid-index-type", "index must be integer"}
	E3004 = ErrorCode{"E3004", "invalid-index-assignment-type", "invalid type for index assignment"}
	E3005 = ErrorCode{"E3005", "cannot-convert-to-int", "cannot convert value to integer"}
	E3006 = ErrorCode{"E3006", "cannot-convert-to-float", "cannot convert value to float"}
	E3007 = ErrorCode{"E3007", "cannot-convert-array", "cannot convert array to scalar"}
	E3008 = ErrorCode{"E3008", "undefined-type", "type is not defined"}
	E3009 = ErrorCode{"E3009", "undefined-type-in-struct", "field type not defined"}
	E3010 = ErrorCode{"E3010", "undefined-param-type", "parameter type not defined"}
	E3011 = ErrorCode{"E3011", "undefined-return-type", "return type not defined"}
	E3012 = ErrorCode{"E3012", "return-type-mismatch", "return type does not match declaration"}
	E3013 = ErrorCode{"E3013", "return-count-mismatch", "wrong number of return values"}
	E3014 = ErrorCode{"E3014", "incompatible-binary-types", "incompatible types for binary operator"}
	E3015 = ErrorCode{"E3015", "not-callable", "value is not callable"}
	E3016 = ErrorCode{"E3016", "not-indexable", "value is not indexable"}
	E3017 = ErrorCode{"E3017", "not-iterable", "value is not iterable"}
	E3018 = ErrorCode{"E3018", "array-literal-required", "array type requires array literal"}
)

// =============================================================================
// REFERENCE ERRORS (E4xxx) - Undefined variables, functions, modules
// =============================================================================
var (
	E4001 = ErrorCode{"E4001", "undefined-variable", "variable not found in scope"}
	E4002 = ErrorCode{"E4002", "undefined-function", "function not defined"}
	E4003 = ErrorCode{"E4003", "undefined-field", "field does not exist on type"}
	E4004 = ErrorCode{"E4004", "undefined-enum-value", "enum value does not exist"}
	E4005 = ErrorCode{"E4005", "undefined-module-member", "member not found in module"}
	E4006 = ErrorCode{"E4006", "undefined-type-new", "type not found for new expression"}
	E4007 = ErrorCode{"E4007", "module-not-imported", "module has not been imported"}
	E4008 = ErrorCode{"E4008", "ambiguous-function", "function exists in multiple modules"}
	E4009 = ErrorCode{"E4009", "no-main-function", "program has no entry point"}
	E4010 = ErrorCode{"E4010", "nil-member-access", "cannot access member of nil"}
	E4011 = ErrorCode{"E4011", "member-access-invalid-type", "type does not support member access"}
)

// =============================================================================
// RUNTIME ERRORS (E5xxx) - General runtime errors
// =============================================================================
var (
	E5001 = ErrorCode{"E5001", "division-by-zero", "cannot divide by zero"}
	E5002 = ErrorCode{"E5002", "modulo-by-zero", "cannot modulo by zero"}
	E5003 = ErrorCode{"E5003", "index-out-of-bounds", "index outside valid range"}
	E5004 = ErrorCode{"E5004", "index-empty-collection", "cannot index empty collection"}
	E5005 = ErrorCode{"E5005", "nil-operation", "cannot perform operation on nil"}
	E5006 = ErrorCode{"E5006", "immutable-variable", "cannot modify const variable"}
	E5007 = ErrorCode{"E5007", "immutable-array", "cannot modify const array"}
	E5008 = ErrorCode{"E5008", "wrong-argument-count", "incorrect number of arguments"}
	E5009 = ErrorCode{"E5009", "break-outside-loop", "break not inside loop"}
	E5010 = ErrorCode{"E5010", "continue-outside-loop", "continue not inside loop"}
	E5011 = ErrorCode{"E5011", "return-value-unused", "function return value not used"}
	E5012 = ErrorCode{"E5012", "multi-assign-count-mismatch", "assignment value count mismatch"}
	E5013 = ErrorCode{"E5013", "range-start-not-integer", "range start must be integer"}
	E5014 = ErrorCode{"E5014", "range-end-not-integer", "range end must be integer"}
	E5015 = ErrorCode{"E5015", "postfix-requires-identifier", "postfix operator needs variable"}
)

// =============================================================================
// IMPORT ERRORS (E6xxx) - Module import errors
// =============================================================================
var (
	E6001 = ErrorCode{"E6001", "circular-import", "circular import detected"}
	E6002 = ErrorCode{"E6002", "module-not-found", "module file not found"}
	E6003 = ErrorCode{"E6003", "invalid-module-format", "module has invalid format"}
	E6004 = ErrorCode{"E6004", "module-load-error", "failed to load module"}
)

// =============================================================================
// BUILTIN ARGUMENT ERRORS (E7xxx) - Standard library builtin function errors
// =============================================================================
var (
	E7001 = ErrorCode{"E7001", "builtin-arg-count", "wrong argument count for builtin"}
	E7002 = ErrorCode{"E7002", "builtin-arg-type", "wrong argument type for builtin"}
	E7003 = ErrorCode{"E7003", "builtin-requires-array", "builtin requires array argument"}
	E7004 = ErrorCode{"E7004", "builtin-requires-string", "builtin requires string argument"}
	E7005 = ErrorCode{"E7005", "builtin-requires-number", "builtin requires numeric argument"}
	E7006 = ErrorCode{"E7006", "builtin-requires-integer", "builtin requires integer argument"}
	E7007 = ErrorCode{"E7007", "builtin-requires-function", "builtin requires function argument"}
	E7008 = ErrorCode{"E7008", "len-unsupported-type", "len not supported for type"}
	E7009 = ErrorCode{"E7009", "type-conversion-failed", "type conversion failed"}
)

// =============================================================================
// MATH ERRORS (E8xxx) - Mathematical operation errors
// =============================================================================
var (
	E8001 = ErrorCode{"E8001", "sqrt-negative", "cannot take square root of negative"}
	E8002 = ErrorCode{"E8002", "log-non-positive", "logarithm requires positive number"}
	E8003 = ErrorCode{"E8003", "log2-non-positive", "log2 requires positive number"}
	E8004 = ErrorCode{"E8004", "log10-non-positive", "log10 requires positive number"}
	E8005 = ErrorCode{"E8005", "asin-out-of-range", "asin requires value in [-1, 1]"}
	E8006 = ErrorCode{"E8006", "acos-out-of-range", "acos requires value in [-1, 1]"}
	E8007 = ErrorCode{"E8007", "factorial-negative", "factorial requires non-negative"}
	E8008 = ErrorCode{"E8008", "factorial-overflow", "factorial result too large"}
	E8009 = ErrorCode{"E8009", "random-max-not-positive", "random max must be positive"}
	E8010 = ErrorCode{"E8010", "random-max-less-than-min", "random max must exceed min"}
	E8011 = ErrorCode{"E8011", "random-float-arg-count", "random_float wrong argument count"}
	E8012 = ErrorCode{"E8012", "avg-no-arguments", "avg requires at least one value"}
)

// =============================================================================
// ARRAY ERRORS (E9xxx) - Array-specific operation errors
// =============================================================================
var (
	E9001 = ErrorCode{"E9001", "array-pop-empty", "cannot pop from empty array"}
	E9002 = ErrorCode{"E9002", "array-shift-empty", "cannot shift from empty array"}
	E9003 = ErrorCode{"E9003", "array-insert-out-of-bounds", "insert index out of bounds"}
	E9004 = ErrorCode{"E9004", "array-get-out-of-bounds", "get index out of bounds"}
	E9005 = ErrorCode{"E9005", "array-set-out-of-bounds", "set index out of bounds"}
	E9006 = ErrorCode{"E9006", "array-remove-at-out-of-bounds", "remove_at index out of bounds"}
	E9007 = ErrorCode{"E9007", "array-min-empty", "cannot get min of empty array"}
	E9008 = ErrorCode{"E9008", "array-max-empty", "cannot get max of empty array"}
	E9009 = ErrorCode{"E9009", "array-avg-empty", "cannot get avg of empty array"}
	E9010 = ErrorCode{"E9010", "array-sum-non-numeric", "sum requires numeric array"}
	E9011 = ErrorCode{"E9011", "array-product-non-numeric", "product requires numeric array"}
	E9012 = ErrorCode{"E9012", "array-avg-non-numeric", "avg requires numeric array"}
	E9013 = ErrorCode{"E9013", "array-range-step-zero", "range step cannot be zero"}
	E9014 = ErrorCode{"E9014", "array-concat-requires-arrays", "concat requires array arguments"}
	E9015 = ErrorCode{"E9015", "array-zip-requires-arrays", "zip requires array arguments"}
	E9016 = ErrorCode{"E9016", "array-slice-invalid-indices", "slice requires integer indices"}
	E9017 = ErrorCode{"E9017", "array-repeat-invalid-count", "repeat requires integer count"}
	E9018 = ErrorCode{"E9018", "array-join-invalid-separator", "join requires string separator"}
)

// =============================================================================
// STRING ERRORS (E10xxx) - String-specific operation errors
// =============================================================================
var (
	E10001 = ErrorCode{"E10001", "string-split-invalid-separator", "split requires string separator"}
	E10002 = ErrorCode{"E10002", "string-join-invalid-array", "join requires array first argument"}
	E10003 = ErrorCode{"E10003", "string-replace-invalid-args", "replace requires string arguments"}
	E10004 = ErrorCode{"E10004", "string-index-invalid-args", "index requires string arguments"}
	E10005 = ErrorCode{"E10005", "string-contains-invalid-args", "contains requires string arguments"}
	E10006 = ErrorCode{"E10006", "string-starts-with-invalid-args", "starts_with requires string arguments"}
	E10007 = ErrorCode{"E10007", "string-ends-with-invalid-args", "ends_with requires string arguments"}
)

// =============================================================================
// TIME ERRORS (E11xxx) - Time-specific operation errors
// =============================================================================
var (
	E11001 = ErrorCode{"E11001", "time-sleep-invalid-arg", "sleep requires numeric argument"}
	E11002 = ErrorCode{"E11002", "time-sleep-ms-invalid-arg", "sleep_ms requires integer argument"}
	E11003 = ErrorCode{"E11003", "time-format-invalid-format", "format requires string format"}
	E11004 = ErrorCode{"E11004", "time-format-invalid-timestamp", "format requires integer timestamp"}
	E11005 = ErrorCode{"E11005", "time-parse-failed", "failed to parse time string"}
	E11006 = ErrorCode{"E11006", "time-parse-invalid-args", "parse requires string arguments"}
	E11007 = ErrorCode{"E11007", "time-make-invalid-args", "make requires integer arguments"}
	E11008 = ErrorCode{"E11008", "time-add-invalid-timestamp", "time add requires integer timestamp"}
	E11009 = ErrorCode{"E11009", "time-add-invalid-amount", "time add requires integer amount"}
	E11010 = ErrorCode{"E11010", "time-diff-invalid-args", "diff requires integer timestamps"}
	E11011 = ErrorCode{"E11011", "time-is-leap-year-invalid-arg", "is_leap_year requires integer year"}
	E11012 = ErrorCode{"E11012", "time-days-in-month-invalid-args", "days_in_month requires integer arguments"}
	E11013 = ErrorCode{"E11013", "time-elapsed-invalid-arg", "elapsed_ms requires integer tick"}
)

// =============================================================================
// WARNINGS (W1xxx - W3xxx)
// =============================================================================
var (
	// Code Style Warnings (W1xxx)
	W1001 = ErrorCode{"W1001", "unused-variable", "variable declared but not used"}
	W1002 = ErrorCode{"W1002", "unused-import", "module imported but not used"}
	W1003 = ErrorCode{"W1003", "unused-function", "function declared but not called"}
	W1004 = ErrorCode{"W1004", "unused-parameter", "parameter declared but not used"}

	// Potential Bug Warnings (W2xxx)
	W2001 = ErrorCode{"W2001", "unreachable-code", "code will never execute"}
	W2002 = ErrorCode{"W2002", "shadowed-variable", "variable shadows outer scope"}
	W2003 = ErrorCode{"W2003", "missing-return", "function may not return value"}
	W2004 = ErrorCode{"W2004", "implicit-type-conversion", "implicit type conversion occurring"}
	W2005 = ErrorCode{"W2005", "deprecated-feature", "using deprecated feature"}

	// Code Quality Warnings (W3xxx)
	W3001 = ErrorCode{"W3001", "empty-block", "block statement is empty"}
	W3002 = ErrorCode{"W3002", "redundant-condition", "condition is always true/false"}
)
