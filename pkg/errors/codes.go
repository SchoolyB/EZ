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
	E1017 = ErrorCode{"E1017", "unclosed-raw-string", "raw string literal not closed"}
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
	E2032 = ErrorCode{"E2032", "const-array-requires-size", "const array must have fixed size"}
	E2033 = ErrorCode{"E2033", "reserved-param-name", "parameter name is reserved"}
	E2034 = ErrorCode{"E2034", "invalid-struct-field", "invalid struct field name"}
	E2035 = ErrorCode{"E2035", "invalid-enum-value", "invalid enum value name"}
	E2036 = ErrorCode{"E2036", "import-inside-block", "import must be at file level"}
	E2037 = ErrorCode{"E2037", "reserved-struct-name", "struct name is reserved"}
	E2038 = ErrorCode{"E2038", "reserved-enum-name", "enum name is reserved"}
	E2039 = ErrorCode{"E2039", "required-after-default", "required parameter after parameter with default"}
	E2040 = ErrorCode{"E2040", "mutable-with-default", "mutable parameter cannot have default value"}
	E2041 = ErrorCode{"E2041", "when-missing-default", "when statement requires a default case"}
	E2042 = ErrorCode{"E2042", "when-strict-has-default", "strict when statement cannot have a default case"}
	E2043 = ErrorCode{"E2043", "when-duplicate-case", "duplicate case value in when statement"}
	E2044 = ErrorCode{"E2044", "when-float-not-allowed", "float type not allowed in when statement"}
	E2045 = ErrorCode{"E2045", "when-strict-non-enum", "strict attribute only allowed on enum when statements"}
	E2046 = ErrorCode{"E2046", "when-strict-missing-case", "strict when statement missing enum case"}
	E2047 = ErrorCode{"E2047", "when-type-as-condition", "when condition must be a value, not a type name"}
	E2048 = ErrorCode{"E2048", "when-bool-condition", "when condition cannot be a boolean; use if/or/otherwise instead"}
	E2049 = ErrorCode{"E2049", "when-nil-condition", "when condition cannot be nil; use if/otherwise to check for nil"}
	E2050 = ErrorCode{"E2050", "when-collection-condition", "when condition cannot be an array or map"}
	E2051 = ErrorCode{"E2051", "suppress-invalid-target", "#suppress can only be applied at file scope or to function declarations"}
	E2052 = ErrorCode{"E2052", "suppress-invalid-code", "warning code cannot be suppressed"}
	E2053 = ErrorCode{"E2053", "type-definition-in-function", "type definitions must be at file level"}
	E2054 = ErrorCode{"E2054", "when-strict-non-enum-case", "#strict when requires explicit enum member values in cases"}
	E2055 = ErrorCode{"E2055", "strict-invalid-target", "#strict can only be applied to when statements"}
	E2056 = ErrorCode{"E2056", "executable-at-file-scope", "executable statement not allowed at file scope"}
	E2057 = ErrorCode{"E2057", "invalid-interpolation-syntax", "invalid string interpolation syntax"}
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
	E3019 = ErrorCode{"E3019", "signed-to-unsigned", "cannot assign signed type to unsigned"}
	E3020 = ErrorCode{"E3020", "negative-to-unsigned", "cannot assign negative value to unsigned type"}
	E3021 = ErrorCode{"E3021", "type-change-not-allowed", "cannot change type of variable after declaration"}
	E3022 = ErrorCode{"E3022", "undefined-struct-field", "struct field not found"}
	E3023 = ErrorCode{"E3023", "enum-value-not-found", "enum value not found"}

	E3024 = ErrorCode{"E3024", "missing-return-statement", "function must return a value"}
	E3025 = ErrorCode{"E3025", "byte-value-out-of-range", "byte value must be between 0 and 255"}
	E3026 = ErrorCode{"E3026", "byte-array-element-out-of-range", "byte array element must be between 0 and 255"}
	E3027 = ErrorCode{"E3027", "const-to-mutable-param", "cannot pass immutable variable to mutable parameter"}
	E3028 = ErrorCode{"E3028", "enum-mixed-types", "enum members must all have the same type"}
	E3029 = ErrorCode{"E3029", "float-enum-map-key", "float-based enum cannot be used as map key"}
	E3030 = ErrorCode{"E3030", "type-as-value", "type definition cannot be used as a runtime value"}
	E3031 = ErrorCode{"E3031", "function-as-value", "function cannot be used as a value without calling it"}
	E3032 = ErrorCode{"E3032", "enum-type-mismatch", "cannot compare values from different enum types"}
	E3033 = ErrorCode{"E3033", "duplicate-enum-value", "enum contains duplicate values"}
	E3034 = ErrorCode{"E3034", "any-type-not-allowed", "'any' type is reserved for internal use"}
	E3035 = ErrorCode{"E3035", "not-all-paths-return", "not all code paths return a value"}
	E3036 = ErrorCode{"E3036", "integer-out-of-range", "integer literal exceeds type range"}
	E3037 = ErrorCode{"E3037", "invalid-private-usage", "private modifier cannot be used here"}
	E3038 = ErrorCode{"E3038", "void-type-not-allowed", "'void' is not a valid type"}
	E3039 = ErrorCode{"E3039", "ensure-expects-call", "ensure expects a function call"}
	E3040 = ErrorCode{"E3040", "multi-return-to-single-var", "cannot assign multiple return values to single variable"}
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
	E4012 = ErrorCode{"E4012", "shadows-type", "variable shadows a type definition"}
	E4013 = ErrorCode{"E4013", "shadows-function", "variable shadows a function"}
	E4014 = ErrorCode{"E4014", "shadows-module", "variable shadows an imported module"}
	E4015 = ErrorCode{"E4015", "shadows-used-module-function", "variable shadows a function from a used module"}
	E4016 = ErrorCode{"E4016", "loop-variable-shadows-loop-variable", "loop variable shadows outer loop variable"}
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
	E5016 = ErrorCode{"E5016", "immutable-parameter", "cannot modify read-only parameter"}
	E5017 = ErrorCode{"E5017", "immutable-struct", "cannot modify field of const struct"}
	E5018 = ErrorCode{"E5018", "max-recursion-depth", "maximum recursion depth exceeded"}
	E5019 = ErrorCode{"E5019", "range-step-not-integer", "range step must be integer"}
	E5020 = ErrorCode{"E5020", "range-in-operand-not-integer", "value checked against range must be integer"}
	E5021 = ErrorCode{"E5021", "panic", "explicit panic called"}
	E5022 = ErrorCode{"E5022", "assertion-failed", "assertion condition was false"}
	E5023 = ErrorCode{"E5023", "postfix-requires-integer", "postfix operator needs integer operand"}
)

// =============================================================================
// IMPORT ERRORS (E6xxx) - Module import errors
// =============================================================================
var (
	E6001 = ErrorCode{"E6001", "circular-import", "circular import detected"}
	E6002 = ErrorCode{"E6002", "module-not-found", "module file not found"}
	E6003 = ErrorCode{"E6003", "invalid-module-format", "module has invalid format"}
	E6004 = ErrorCode{"E6004", "module-load-error", "failed to load module"}
	E6005 = ErrorCode{"E6005", "module-name-mismatch", "module name does not match directory"}
	E6006 = ErrorCode{"E6006", "module-name-conflict", "files in directory declare different module names"}
	E6007 = ErrorCode{"E6007", "internal-import-denied", "cannot import from internal/ directory outside package"}
	E6008 = ErrorCode{"E6008", "module-member-readonly", "cannot assign to module member"}
	E6009 = ErrorCode{"E6009", "private-access-denied", "cannot access private member from outside module"}
)

// =============================================================================
// STDLIB VALIDATION ERRORS (E7xxx) - Generic argument validation for all stdlib
// These are shared across all stdlib modules for common validation patterns
// =============================================================================
var (
	// Argument count errors
	E7001 = ErrorCode{"E7001", "stdlib-argument-mismatch", "wrong number of arguments"}

	// Type requirement errors - used when function requires specific type
	E7002 = ErrorCode{"E7002", "requires-array", "argument must be an array"}
	E7003 = ErrorCode{"E7003", "requires-string", "argument must be a string"}
	E7004 = ErrorCode{"E7004", "requires-integer", "argument must be an integer"}
	E7005 = ErrorCode{"E7005", "requires-number", "argument must be a number"}
	E7006 = ErrorCode{"E7006", "requires-function", "argument must be a function"}
	E7007 = ErrorCode{"E7007", "requires-map", "argument must be a map"}
	E7008 = ErrorCode{"E7008", "requires-boolean", "argument must be a boolean"}
	E7009 = ErrorCode{"E7009", "requires-char", "argument must be a char"}

	// Value validation errors
	E7010 = ErrorCode{"E7010", "invalid-argument-value", "argument value is invalid"}
	E7011 = ErrorCode{"E7011", "negative-not-allowed", "argument cannot be negative"}
	E7012 = ErrorCode{"E7012", "zero-not-allowed", "argument cannot be zero"}
	E7013 = ErrorCode{"E7013", "empty-not-allowed", "argument cannot be empty"}

	// Conversion errors
	E7014 = ErrorCode{"E7014", "type-conversion-failed", "type conversion failed"}

	// Legacy codes for backward compatibility with builtins
	E7015 = ErrorCode{"E7015", "len-unsupported-type", "len not supported for type"}

	// I/O errors
	E7016 = ErrorCode{"E7016", "file-not-found", "file or directory not found"}
	E7017 = ErrorCode{"E7017", "permission-denied", "permission denied"}
	E7018 = ErrorCode{"E7018", "cannot-remove-directory", "io.remove() cannot remove directories"}
	E7019 = ErrorCode{"E7019", "cannot-remove-file", "io.remove_dir() can only remove directories"}
	E7020 = ErrorCode{"E7020", "safety-check-failed", "cannot remove root or home directory"}
	E7021 = ErrorCode{"E7021", "cannot-copy-directory", "io.copy() cannot copy directories"}
	E7022 = ErrorCode{"E7022", "file-already-exists", "file or directory already exists"}
	E7023 = ErrorCode{"E7023", "directory-not-empty", "directory not empty"}

	// OS module errors
	E7024 = ErrorCode{"E7024", "env-var-operation-failed", "environment variable operation failed"}
	E7035 = ErrorCode{"E7035", "env-var-not-set", "environment variable is not set"}

	E7025 = ErrorCode{"E7025", "get-cwd-failed", "failed to get current directory"}
	E7026 = ErrorCode{"E7026", "chdir-failed", "failed to change directory"}
	E7027 = ErrorCode{"E7027", "get-hostname-failed", "failed to get hostname"}
	E7028 = ErrorCode{"E7028", "get-username-failed", "failed to get username"}
	E7029 = ErrorCode{"E7029", "get-homedir-failed", "failed to get home directory"}
	E7030 = ErrorCode{"E7030", "command-not-found", "command or executable not found"}
	E7031 = ErrorCode{"E7031", "command-failed", "command execution failed"}
	E7032 = ErrorCode{"E7032", "sleep-negative", "sleep duration cannot be negative"}
	E7033 = ErrorCode{"E7033", "conversion-overflow", "value exceeds target type range"}

	// Path validation errors
	E7040 = ErrorCode{"E7040", "empty-path", "path cannot be empty"}
	E7041 = ErrorCode{"E7041", "path-null-byte", "path contains null byte"}
	E7042 = ErrorCode{"E7042", "read-directory-as-file", "cannot read directory as file"}

	// Glob pattern errors
	E7043 = ErrorCode{"E7043", "invalid-glob-pattern", "invalid glob pattern syntax"}

	// File handle errors
	E7050 = ErrorCode{"E7050", "file-handle-closed", "file handle is closed"}

	// General I/O error (catch-all)
	E7099 = ErrorCode{"E7099", "io-error", "general I/O error"}
)

// =============================================================================
// MATH ERRORS (E8xxx) - Math-specific domain errors
// These are errors unique to mathematical operations
// =============================================================================
var (
	E8001 = ErrorCode{"E8001", "sqrt-negative", "cannot take square root of negative number"}
	E8002 = ErrorCode{"E8002", "log-non-positive", "logarithm requires positive number"}
	E8003 = ErrorCode{"E8003", "trig-out-of-range", "trigonometric function input out of valid range"}
	E8004 = ErrorCode{"E8004", "factorial-negative", "factorial requires non-negative integer"}
	E8005 = ErrorCode{"E8005", "factorial-overflow", "factorial result exceeds maximum value"}
	E8006 = ErrorCode{"E8006", "random-invalid-range", "random range is invalid"}
	E8007 = ErrorCode{"E8007", "map-range-div-zero", "map_range requires in_min != in_max"}
)

// =============================================================================
// ARRAY ERRORS (E9xxx) - Array-specific domain errors
// These are errors unique to array operations
// =============================================================================
var (
	E9001 = ErrorCode{"E9001", "array-is-empty", "operation requires non-empty array"}
	E9002 = ErrorCode{"E9002", "array-non-numeric", "operation requires numeric array"}
	E9003 = ErrorCode{"E9003", "range-step-zero", "range step cannot be zero"}
	E9004 = ErrorCode{"E9004", "chunk-size-invalid", "chunk size must be greater than zero"}
	E9005 = ErrorCode{"E9005", "range-invalid-bounds", "range start must be less than or equal to end"}
	E9006 = ErrorCode{"E9006", "array-modified-during-iteration", "cannot modify array during for_each iteration"}
	E9007 = ErrorCode{"E9007", "empty-array-selection", "cannot select from empty array"}
	E9008 = ErrorCode{"E9008", "sample-count-exceeds-length", "sample count exceeds array length"}
)

// =============================================================================
// STRING ERRORS (E10xxx) - String-specific domain errors
// These are errors unique to string operations
// =============================================================================
var (
	E10001 = ErrorCode{"E10001", "repeat-count-negative", "repeat count cannot be negative"}

	E10003 = ErrorCode{"E10003", "string-index-out-of-bounds", "string index out of bounds"}
	E10004 = ErrorCode{"E10004", "string-empty-index", "cannot index empty string"}
)

// =============================================================================
// TIME ERRORS (E11xxx) - Time-specific domain errors
// These are errors unique to time operations
// =============================================================================
var (
	E11001 = ErrorCode{"E11001", "time-parse-failed", "failed to parse time string"}
)

// =============================================================================
// MAP ERRORS (E12xxx) - Map-specific domain errors
// These are errors unique to map operations
// =============================================================================
var (
	E12001 = ErrorCode{"E12001", "map-key-not-hashable", "map key must be a hashable type"}
	E12002 = ErrorCode{"E12002", "map-immutable", "cannot modify immutable map"}
	E12003 = ErrorCode{"E12003", "map-key-not-found", "key not found in map"}
	E12004 = ErrorCode{"E12004", "map-invalid-pair", "map entry must be a [key, value] pair"}
	E12005 = ErrorCode{"E12005", "map-value-not-hashable", "map value is not hashable and cannot become a key"}
	E12006 = ErrorCode{"E12006", "map-duplicate-key", "map literal contains duplicate key"}
)

// =============================================================================
// JSON ERRORS (E13xxx) - JSON-specific domain errors
// These are errors unique to JSON operations
// =============================================================================
var (
	E13001 = ErrorCode{"E13001", "json-syntax-error", "invalid JSON syntax"}
	E13002 = ErrorCode{"E13002", "json-unsupported-type", "type cannot be converted to JSON"}
	E13003 = ErrorCode{"E13003", "json-invalid-map-key", "JSON object keys must be strings"}
	E13004 = ErrorCode{"E13004", "json-decode-requires-type", "json.decode() requires a type argument"}
)

// =============================================================================
// HTTP ERRORS (E14xxx) - HTTP-specific domain errors
// These are errors unique to HTTP operations
// =============================================================================
var (
	E14001 = ErrorCode{"E14001", "http-invalid-url", "invalid URL"}
	E14002 = ErrorCode{"E14002", "http-failed-request", "request failed (network error)"}
	E14003 = ErrorCode{"E14003", "http-timeout", "timeout exceeded"}
	E14004 = ErrorCode{"E14004", "http-invalid-method", "invalid HTTP method"}
	E14005 = ErrorCode{"E14005", "http-failed-url-decode", "URL decode failed"}
	E14006 = ErrorCode{"E14006", "http-failed-json-encoding", "JSON encoding failed"}
)

// =============================================================================
// CRYPTO ERRORS (E15xxx) - Cryptography-specific errors
// =============================================================================
var (
	E15001 = ErrorCode{"E15001", "crypto-random-failed", "cryptographic random generation failed"}
)

// =============================================================================
// ENCODING ERRORS (E16xxx) - Encoding/decoding errors
// =============================================================================
var (
	E16001 = ErrorCode{"E16001", "invalid-base64", "invalid base64 encoded input"}
	E16002 = ErrorCode{"E16002", "invalid-hex", "invalid hexadecimal encoded input"}
	E16003 = ErrorCode{"E16003", "invalid-url-encoding", "invalid URL encoded input"}
)

// =============================================================================
// DB ERRORS (E17xxx) - DB-specific domain errors
// These are errors unique to database operations
// =============================================================================
var (
	E17001 = ErrorCode{"E17001", "db-open-failed", "failed to open database"}
	E17002 = ErrorCode{"E17002", "db-read-failed", "failed to read database file"}
	E17003 = ErrorCode{"E17003", "db-write-failed", "failed to write database file"}
	E17004 = ErrorCode{"E17004", "db-corrupted", "database file is corrupted"}
	E17005 = ErrorCode{"E17005", "db-closed", "operation on closed database"}
)

// =============================================================================
// WARNINGS (W1xxx - W6xxx)
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
	W2006 = ErrorCode{"W2006", "byte-overflow-potential", "byte arithmetic may overflow"}
	W2007 = ErrorCode{"W2007", "shadows-global", "variable shadows global variable or constant"}
	W2008 = ErrorCode{"W2008", "integer-overflow-potential", "integer arithmetic may overflow"}
	W2009 = ErrorCode{"W2009", "nil-dereference-potential", "accessing member on potentially nil value"}
	W2010 = ErrorCode{"W2010", "chained-nil-access", "chained member access on nullable struct type"}

	// Code Quality Warnings (W3xxx)
	W3001 = ErrorCode{"W3001", "empty-block", "block statement is empty"}
	W3002 = ErrorCode{"W3002", "redundant-condition", "condition is always true/false"}
	W3003 = ErrorCode{"W3003", "array-size-mismatch", "fixed-size array not fully initialized"}

	// Module Warnings (W4xxx)
	W4001 = ErrorCode{"W4001", "module-name-mismatch", "module name does not match directory name"}
)
