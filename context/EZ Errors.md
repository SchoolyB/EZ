# EZ Language Error and Warning Reference

This document is the **authoritative source** for all error codes, types, and messages in the EZ programming language. Every error and warning MUST have a unique code, type, and message.

---

## Error Code Categories

| Category | Code Range | Description |
|----------|------------|-------------|
| Lexer Errors | E1xxx | Tokenization and lexical analysis errors |
| Parse Errors | E2xxx | Syntax and parsing errors |
| Type Errors | E3xxx | Type system errors |
| Reference Errors | E4xxx | Undefined variables, functions, modules |
| Runtime Errors | E5xxx | General runtime errors |
| Import Errors | E6xxx | Module import errors |
| Builtin Errors | E7xxx | Standard library builtin function errors |
| Math Errors | E8xxx | Mathematical operation errors |
| Array Errors | E9xxx | Array-specific operation errors |
| String Errors | E10xxx | String-specific operation errors |
| Time Errors | E11xxx | Time-specific operation errors |
| Map Errors | E12xxx | Map-specific operation errors |
| Warnings | W1xxx-W3xxx | Warnings (non-fatal) |

---

## Lexer Errors (E1xxx)

### E1001 - illegal-character
**Type:** `illegal-character`
**Message:** `illegal character in source`
**Used for:** Single `&` or `|` without being doubled, other invalid characters
**Example:** `"unexpected character '&', did you mean '&&'?"`

---

### E1002 - illegal-or-character
**Type:** `illegal-or-character`
**Message:** `illegal OR character in source`
**Used for:** Single `|` without being doubled
**Example:** `"unexpected character '|', did you mean '||'?"`

---

### E1003 - unclosed-comment
**Type:** `unclosed-comment`
**Message:** `multi-line comment not closed`
**Used for:** `/*` without matching `*/`
**Example:** `"unclosed multi-line comment"`

---

### E1004 - unclosed-string
**Type:** `unclosed-string`
**Message:** `string literal not closed`
**Used for:** String without closing `"`
**Example:** `"unclosed string literal"`

---

### E1005 - unclosed-char
**Type:** `unclosed-char`
**Message:** `character literal not closed`
**Used for:** Char without closing `'`
**Example:** `"unclosed character literal"`

---

### E1006 - invalid-escape-string
**Type:** `invalid-escape-string`
**Message:** `invalid escape sequence in string`
**Used for:** Unknown escape like `\x` in string
**Example:** `"invalid escape sequence '\\x' in string"`

---

### E1007 - invalid-escape-char
**Type:** `invalid-escape-char`
**Message:** `invalid escape sequence in character`
**Used for:** Unknown escape like `\x` in char literal
**Example:** `"invalid escape sequence '\\x' in character literal"`

---

### E1008 - empty-char-literal
**Type:** `empty-char-literal`
**Message:** `character literal is empty`
**Used for:** Empty char `''`
**Example:** `"empty character literal"`

---

### E1009 - multi-char-literal
**Type:** `multi-char-literal`
**Message:** `character literal contains multiple characters`
**Used for:** Char like `'ab'`
**Example:** `"character literal must contain exactly one character"`

---

### E1010 - invalid-number-format
**Type:** `invalid-number-format`
**Message:** `invalid numeric literal format`
**Used for:** General number format errors
**Example:** `"invalid number format: multiple decimal points"`

---

### E1011 - number-consecutive-underscores
**Type:** `number-consecutive-underscores`
**Message:** `consecutive underscores in number`
**Used for:** Numbers like `1__000`
**Example:** `"consecutive underscores not allowed in numeric literals"`

---

### E1012 - number-leading-underscore
**Type:** `number-leading-underscore`
**Message:** `number starts with underscore`
**Used for:** Numbers like `_123`
**Example:** `"numeric literal cannot start with underscore"`

---

### E1013 - number-trailing-underscore
**Type:** `number-trailing-underscore`
**Message:** `number ends with underscore`
**Used for:** Numbers like `123_`
**Example:** `"numeric literal cannot end with underscore"`

---

### E1014 - number-underscore-before-decimal
**Type:** `number-underscore-before-decimal`
**Message:** `underscore before decimal point`
**Used for:** Numbers like `123_.45`
**Example:** `"underscore cannot appear immediately before decimal point"`

---

### E1015 - number-underscore-after-decimal
**Type:** `number-underscore-after-decimal`
**Message:** `underscore after decimal point`
**Used for:** Numbers like `123._45`
**Example:** `"underscore cannot appear immediately after decimal point"`

---

### E1016 - number-trailing-decimal
**Type:** `number-trailing-decimal`
**Message:** `decimal point without digits`
**Used for:** Numbers like `123.`
**Example:** `"invalid number format: decimal point must be followed by digits"`

---

## Parse Errors (E2xxx)

### E2001 - unexpected-token
**Type:** `unexpected-token`
**Message:** `unexpected token encountered`
**Used for:** Token found where another was expected
**Example:** `"unexpected token 'foo'"`

---

### E2002 - missing-token
**Type:** `missing-token`
**Message:** `expected token not found`
**Used for:** Missing expected token
**Example:** `"expected RPAREN, got IDENT instead"`

---

### E2003 - missing-expression
**Type:** `missing-expression`
**Message:** `expected expression`
**Used for:** Missing expression after operator or `=`
**Example:** `"expected expression after '='"`

---

### E2004 - unclosed-brace
**Type:** `unclosed-brace`
**Message:** `missing closing brace`
**Used for:** Missing `}`
**Example:** `"unclosed brace - expected '}'"`

---

### E2005 - unclosed-paren
**Type:** `unclosed-paren`
**Message:** `missing closing parenthesis`
**Used for:** Missing `)`
**Example:** `"unclosed parenthesis - expected ')'"`

---

### E2006 - unclosed-bracket
**Type:** `unclosed-bracket`
**Message:** `missing closing bracket`
**Used for:** Missing `]`
**Example:** `"unclosed bracket - expected ']'"`

---

### E2007 - unclosed-interpolation
**Type:** `unclosed-interpolation`
**Message:** `string interpolation not closed`
**Used for:** `"Hello ${name"` without closing `}`
**Example:** `"unclosed interpolation in string"`

---

### E2008 - invalid-assignment-target
**Type:** `invalid-assignment-target`
**Message:** `cannot assign to this expression`
**Used for:** Assignment to non-assignable (not identifier, index, or member)
**Example:** `"invalid assignment target: cannot assign to *ast.IntegerValue"`

---

### E2009 - using-after-declarations
**Type:** `using-after-declarations`
**Message:** `using statement must come first`
**Used for:** `using` after other declarations
**Example:** `"file-scoped 'using' must come before all declarations"`

---

### E2010 - using-before-import
**Type:** `using-before-import`
**Message:** `cannot use module before importing`
**Used for:** `using math` before `import math`
**Example:** `"cannot use module 'math' before importing it"`

---

### E2011 - const-requires-value
**Type:** `const-requires-value`
**Message:** `const must be initialized`
**Used for:** `const x int` without value
**Example:** `"const 'x' must be initialized with a value"`

---

### E2012 - duplicate-parameter
**Type:** `duplicate-parameter`
**Message:** `parameter name already used`
**Used for:** `do foo(x int, x string)`
**Example:** `"duplicate parameter name 'x'"`

---

### E2013 - duplicate-field
**Type:** `duplicate-field`
**Message:** `field name already used`
**Used for:** Struct with duplicate field names
**Example:** `"duplicate field name 'x' in struct"`

---

### E2014 - missing-parameter-type
**Type:** `missing-parameter-type`
**Message:** `parameter missing type annotation`
**Used for:** `do foo(x)` without type
**Example:** `"parameter 'x' is missing a type"`

---

### E2015 - missing-return-type
**Type:** `missing-return-type`
**Message:** `expected return type after arrow`
**Used for:** `do foo() ->` without type
**Example:** `"expected return type after '->'"`

---

### E2016 - empty-enum
**Type:** `empty-enum`
**Message:** `enum must have at least one value`
**Used for:** `enum Status {}`
**Example:** `"enum 'Status' must have at least one value"`

---

### E2017 - trailing-comma-array
**Type:** `trailing-comma-array`
**Message:** `trailing comma in array literal`
**Used for:** `{1, 2, 3,}`
**Example:** `"unexpected trailing comma in array literal"`

---

### E2018 - trailing-comma-call
**Type:** `trailing-comma-call`
**Message:** `trailing comma in function call`
**Used for:** `foo(1, 2,)`
**Example:** `"unexpected trailing comma in function call"`

---

### E2019 - nested-function
**Type:** `nested-function`
**Message:** `function inside function not allowed`
**Used for:** Function declared inside another function
**Example:** `"function declarations are not allowed inside functions"`

---

### E2020 - reserved-variable-name
**Type:** `reserved-variable-name`
**Message:** `variable name is reserved`
**Used for:** Using keyword as variable name
**Example:** `"'temp' is a reserved keyword and cannot be used as a variable name"`

---

### E2021 - reserved-function-name
**Type:** `reserved-function-name`
**Message:** `function name is reserved`
**Used for:** Using builtin as function name
**Example:** `"'len' is a reserved keyword and cannot be used as a function name"`

---

### E2022 - reserved-type-name
**Type:** `reserved-type-name`
**Message:** `type name is reserved`
**Used for:** Using keyword as struct/enum name
**Example:** `"'int' is a reserved keyword and cannot be used as a type name"`

---

### E2023 - duplicate-declaration
**Type:** `duplicate-declaration`
**Message:** `name already declared in scope`
**Used for:** Re-declaring same name in scope
**Example:** `"'x' is already declared in this scope"`

---

### E2024 - invalid-type-name
**Type:** `invalid-type-name`
**Message:** `expected valid type name`
**Used for:** Invalid type in declaration
**Example:** `"expected type name, got RBRACE"`

---

### E2025 - invalid-array-size
**Type:** `invalid-array-size`
**Message:** `array size must be integer`
**Used for:** `[string, "5"]` instead of `[string, 5]`
**Example:** `"expected integer for array size, got STRING"`

---

### E2026 - invalid-enum-type
**Type:** `invalid-enum-type`
**Message:** `enum type must be primitive`
**Used for:** `enum Status : CustomType`
**Example:** `"enum type attributes must be primitive types (int, float, or string), got 'Custom'"`

---

### E2027 - integer-parse-error
**Type:** `integer-parse-error`
**Message:** `cannot parse integer literal`
**Used for:** Invalid integer format
**Example:** `"could not parse \"abc\" as integer"`

---

### E2028 - float-parse-error
**Type:** `float-parse-error`
**Message:** `cannot parse float literal`
**Used for:** Invalid float format
**Example:** `"could not parse \"1.2.3\" as float"`

---

### E2029 - expected-identifier
**Type:** `expected-identifier`
**Message:** `expected identifier`
**Used for:** Missing identifier where required
**Example:** `"expected identifier or @ignore, got RBRACE"`

---

### E2030 - expected-block
**Type:** `expected-block`
**Message:** `expected block statement`
**Used for:** Missing `{` after if/for/while
**Example:** `"expected '{' after condition"`

---

### E2031 - string-enum-requires-values
**Type:** `string-enum-requires-values`
**Message:** `string enum needs explicit values`
**Used for:** String enum without values
**Example:** `"string enums require explicit values for all members"`

---

## Type Errors (E3xxx)

### E3001 - type-mismatch
**Type:** `type-mismatch`
**Message:** `types do not match`
**Used for:** General type mismatch
**Example:** `"type mismatch: expected int, got string"`

---

### E3002 - invalid-operator-for-type
**Type:** `invalid-operator-for-type`
**Message:** `operator not valid for type`
**Used for:** `"hello" * "world"`
**Example:** `"unknown operator: STRING * STRING"`

---

### E3003 - invalid-index-type
**Type:** `invalid-index-type`
**Message:** `index must be integer`
**Used for:** `arr["foo"]` instead of `arr[0]`
**Example:** `"index must be an integer, got STRING"`

---

### E3004 - invalid-index-assignment-type
**Type:** `invalid-index-assignment-type`
**Message:** `invalid type for index assignment`
**Used for:** Assigning wrong type to string index
**Example:** `"can only assign character to string index, got INTEGER"`

---

### E3005 - cannot-convert-to-int
**Type:** `cannot-convert-to-int`
**Message:** `cannot convert value to integer`
**Used for:** `int("abc")`
**Example:** `"cannot convert \"abc\" to int: invalid integer format"`

---

### E3006 - cannot-convert-to-float
**Type:** `cannot-convert-to-float`
**Message:** `cannot convert value to float`
**Used for:** `float("abc")`
**Example:** `"cannot convert \"abc\" to float: invalid float format"`

---

### E3007 - cannot-convert-array
**Type:** `cannot-convert-array`
**Message:** `cannot convert array to scalar`
**Used for:** `int(myArray)`
**Example:** `"cannot convert ARRAY to int"`

---

### E3008 - undefined-type
**Type:** `undefined-type`
**Message:** `type is not defined`
**Used for:** Using unknown type name
**Example:** `"undefined type 'Foo'"`

---

### E3009 - undefined-type-in-struct
**Type:** `undefined-type-in-struct`
**Message:** `field type not defined`
**Used for:** Struct field with unknown type
**Example:** `"undefined type 'Custom' in struct 'Person'"`

---

### E3010 - undefined-param-type
**Type:** `undefined-param-type`
**Message:** `parameter type not defined`
**Used for:** Function param with unknown type
**Example:** `"undefined type 'Foo' for parameter 'x'"`

---

### E3011 - undefined-return-type
**Type:** `undefined-return-type`
**Message:** `return type not defined`
**Used for:** Function with unknown return type
**Example:** `"undefined return type 'Bar' in function 'foo'"`

---

### E3012 - return-type-mismatch
**Type:** `return-type-mismatch`
**Message:** `return type does not match declaration`
**Used for:** Returning wrong type
**Example:** `"return type mismatch: expected int, got string"`

---

### E3013 - return-count-mismatch
**Type:** `return-count-mismatch`
**Message:** `wrong number of return values`
**Used for:** Returning wrong count
**Example:** `"wrong number of return values: expected 2, got 1"`

---

### E3014 - incompatible-binary-types
**Type:** `incompatible-binary-types`
**Message:** `incompatible types for binary operator`
**Used for:** `1 + "hello"`
**Example:** `"incompatible types for operator '+': INTEGER and STRING"`

---

### E3015 - not-callable
**Type:** `not-callable`
**Message:** `value is not callable`
**Used for:** Calling non-function
**Example:** `"not a function: INTEGER"`

---

### E3016 - not-indexable
**Type:** `not-indexable`
**Message:** `value is not indexable`
**Used for:** Indexing non-array/string
**Example:** `"index operator not supported for STRUCT"`

---

### E3017 - not-iterable
**Type:** `not-iterable`
**Message:** `value is not iterable`
**Used for:** for_each on non-iterable
**Example:** `"for_each requires array or string, got INTEGER"`

---

### E3018 - array-literal-required
**Type:** `array-literal-required`
**Message:** `array type requires array literal`
**Used for:** `const arr [int] = 10`
**Example:** `"array type requires value in {} format, got scalar"`

---

## Reference Errors (E4xxx)

### E4001 - undefined-variable
**Type:** `undefined-variable`
**Message:** `variable not found in scope`
**Used for:** Using undeclared variable
**Example:** `"identifier not found: 'foo'"`

---

### E4002 - undefined-function
**Type:** `undefined-function`
**Message:** `function not defined`
**Used for:** Calling undeclared function
**Example:** `"undefined function: 'foo'"`

---

### E4003 - undefined-field
**Type:** `undefined-field`
**Message:** `field does not exist on type`
**Used for:** Accessing non-existent struct field
**Example:** `"field 'foo' not found"`

---

### E4004 - undefined-enum-value
**Type:** `undefined-enum-value`
**Message:** `enum value does not exist`
**Used for:** Accessing non-existent enum value
**Example:** `"enum value 'BAR' not found in enum 'Status'"`

---

### E4005 - undefined-module-member
**Type:** `undefined-module-member`
**Message:** `member not found in module`
**Used for:** `math.nonexistent()`
**Example:** `"'bar' not found in module 'math'"`

---

### E4006 - undefined-type-new
**Type:** `undefined-type-new`
**Message:** `type not found for new expression`
**Used for:** `new UnknownType{}`
**Example:** `"undefined type: 'Foo'"`

---

### E4007 - module-not-imported
**Type:** `module-not-imported`
**Message:** `module has not been imported`
**Used for:** Using module without import
**Example:** `"cannot use 'math': module not imported"`

---

### E4008 - ambiguous-function
**Type:** `ambiguous-function`
**Message:** `function exists in multiple modules`
**Used for:** With `using`, function in multiple modules
**Example:** `"function 'len' found in multiple modules: arrays, strings"`

---

### E4009 - no-main-function
**Type:** `no-main-function`
**Message:** `program has no entry point`
**Used for:** Missing `main()` function
**Example:** `"Program must define a main() function"`

---

### E4010 - nil-member-access
**Type:** `nil-member-access`
**Message:** `cannot access member of nil`
**Used for:** `nilValue.field`
**Example:** `"nil reference: cannot access member 'x' of nil"`

---

### E4011 - member-access-invalid-type
**Type:** `member-access-invalid-type`
**Message:** `type does not support member access`
**Used for:** `123.field`
**Example:** `"member access not supported on type INTEGER"`

---

## Runtime Errors (E5xxx)

### E5001 - division-by-zero
**Type:** `division-by-zero`
**Message:** `cannot divide by zero`
**Used for:** `x / 0`
**Example:** `"division by zero"`

---

### E5002 - modulo-by-zero
**Type:** `modulo-by-zero`
**Message:** `cannot modulo by zero`
**Used for:** `x % 0`
**Example:** `"modulo by zero"`

---

### E5003 - index-out-of-bounds
**Type:** `index-out-of-bounds`
**Message:** `index outside valid range`
**Used for:** `arr[100]` when array has 5 elements
**Example:** `"index out of bounds: attempted to access index 100, but valid range is 0-4"`

---

### E5004 - index-empty-collection
**Type:** `index-empty-collection`
**Message:** `cannot index empty collection`
**Used for:** `arr[0]` when array is empty
**Example:** `"index out of bounds: array is empty (length 0)"`

---

### E5005 - nil-operation
**Type:** `nil-operation`
**Message:** `cannot perform operation on nil`
**Used for:** `nil + 1`
**Example:** `"nil reference: cannot use nil with operator '+'"`

---

### E5006 - immutable-variable
**Type:** `immutable-variable`
**Message:** `cannot modify const variable`
**Used for:** Reassigning const
**Example:** `"cannot assign to immutable variable 'x' (declared as const)"`

---

### E5007 - immutable-array
**Type:** `immutable-array`
**Message:** `cannot modify const array`
**Used for:** Modifying const array
**Example:** `"cannot modify immutable array (declared as const)"`

---

### E5008 - wrong-argument-count
**Type:** `wrong-argument-count`
**Message:** `incorrect number of arguments`
**Used for:** `foo(1, 2)` when foo takes 3
**Example:** `"wrong number of arguments: expected 3, got 2"`

---

### E5009 - break-outside-loop
**Type:** `break-outside-loop`
**Message:** `break not inside loop`
**Used for:** `break` outside for/while
**Example:** `"break statement outside loop"`

---

### E5010 - continue-outside-loop
**Type:** `continue-outside-loop`
**Message:** `continue not inside loop`
**Used for:** `continue` outside for/while
**Example:** `"continue statement outside loop"`

---

### E5011 - return-value-unused
**Type:** `return-value-unused`
**Message:** `function return value not used`
**Used for:** Ignoring return without `@ignore`
**Example:** `"return value from function not used (use @ignore to discard)"`

---

### E5012 - multi-assign-count-mismatch
**Type:** `multi-assign-count-mismatch`
**Message:** `assignment value count mismatch`
**Used for:** `a, b = getOne()`
**Example:** `"expected 2 values, got 1"`

---

### E5013 - range-start-not-integer
**Type:** `range-start-not-integer`
**Message:** `range start must be integer`
**Used for:** `for i in "a"..10`
**Example:** `"range start must be integer"`

---

### E5014 - range-end-not-integer
**Type:** `range-end-not-integer`
**Message:** `range end must be integer`
**Used for:** `for i in 1.."b"`
**Example:** `"range end must be integer"`

---

### E5015 - postfix-requires-identifier
**Type:** `postfix-requires-identifier`
**Message:** `postfix operator needs variable`
**Used for:** `5++`
**Example:** `"postfix operator requires identifier"`

---

## Import Errors (E6xxx)

### E6001 - circular-import
**Type:** `circular-import`
**Message:** `circular import detected`
**Used for:** A imports B, B imports A
**Example:** `"circular import detected: a.ez -> b.ez -> a.ez"`

---

### E6002 - module-not-found
**Type:** `module-not-found`
**Message:** `module file not found`
**Used for:** `import nonexistent`
**Example:** `"module 'foo' not found"`

---

### E6003 - invalid-module-format
**Type:** `invalid-module-format`
**Message:** `module has invalid format`
**Used for:** Module file with syntax errors
**Example:** `"invalid module 'foo': parse error"`

---

### E6004 - module-load-error
**Type:** `module-load-error`
**Message:** `failed to load module`
**Used for:** Module file read error
**Example:** `"failed to load module 'foo': permission denied"`

---

## Builtin Argument Errors (E7xxx)

### E7001 - builtin-arg-count
**Type:** `builtin-arg-count`
**Message:** `wrong argument count for builtin`
**Used for:** Wrong number of args to builtin
**Example:** `"len() takes exactly 1 argument, got 2"`

---

### E7002 - builtin-arg-type
**Type:** `builtin-arg-type`
**Message:** `wrong argument type for builtin`
**Used for:** Wrong type to builtin
**Example:** `"typeof() takes exactly 1 argument"`

---

### E7003 - builtin-requires-array
**Type:** `builtin-requires-array`
**Message:** `builtin requires array argument`
**Used for:** Non-array to array builtin
**Example:** `"arrays.len() requires an array"`

---

### E7004 - builtin-requires-string
**Type:** `builtin-requires-string`
**Message:** `builtin requires string argument`
**Used for:** Non-string to string builtin
**Example:** `"strings.len() requires a string argument"`

---

### E7005 - builtin-requires-number
**Type:** `builtin-requires-number`
**Message:** `builtin requires numeric argument`
**Used for:** Non-number to math builtin
**Example:** `"expected number, got STRING"`

---

### E7006 - builtin-requires-integer
**Type:** `builtin-requires-integer`
**Message:** `builtin requires integer argument`
**Used for:** Non-integer where integer needed
**Example:** `"arrays.get() requires an integer index"`

---

### E7007 - builtin-requires-function
**Type:** `builtin-requires-function`
**Message:** `builtin requires function argument`
**Used for:** Non-function callback
**Example:** `"arrays.map() requires a function"`

---

### E7008 - len-unsupported-type
**Type:** `len-unsupported-type`
**Message:** `len not supported for type`
**Used for:** `len(123)`
**Example:** `"len() not supported for INTEGER"`

---

### E7009 - type-conversion-failed
**Type:** `type-conversion-failed`
**Message:** `type conversion failed`
**Used for:** Failed int/float/string conversion
**Example:** `"cannot convert STRING to int"`

---

## Math Errors (E8xxx)

### E8001 - sqrt-negative
**Type:** `sqrt-negative`
**Message:** `cannot take square root of negative`
**Used for:** `math.sqrt(-1)`
**Example:** `"math.sqrt() cannot take negative number"`

---

### E8002 - log-non-positive
**Type:** `log-non-positive`
**Message:** `logarithm requires positive number`
**Used for:** `math.log(0)` or `math.log(-1)`
**Example:** `"math.log() requires positive number"`

---

### E8003 - log2-non-positive
**Type:** `log2-non-positive`
**Message:** `log2 requires positive number`
**Used for:** `math.log2(0)`
**Example:** `"math.log2() requires positive number"`

---

### E8004 - log10-non-positive
**Type:** `log10-non-positive`
**Message:** `log10 requires positive number`
**Used for:** `math.log10(0)`
**Example:** `"math.log10() requires positive number"`

---

### E8005 - asin-out-of-range
**Type:** `asin-out-of-range`
**Message:** `asin requires value in [-1, 1]`
**Used for:** `math.asin(2)`
**Example:** `"math.asin() requires value between -1 and 1"`

---

### E8006 - acos-out-of-range
**Type:** `acos-out-of-range`
**Message:** `acos requires value in [-1, 1]`
**Used for:** `math.acos(2)`
**Example:** `"math.acos() requires value between -1 and 1"`

---

### E8007 - factorial-negative
**Type:** `factorial-negative`
**Message:** `factorial requires non-negative`
**Used for:** `math.factorial(-1)`
**Example:** `"math.factorial() requires non-negative integer"`

---

### E8008 - factorial-overflow
**Type:** `factorial-overflow`
**Message:** `factorial result too large`
**Used for:** `math.factorial(21)`
**Example:** `"math.factorial() overflow for n > 20"`

---

### E8009 - random-max-not-positive
**Type:** `random-max-not-positive`
**Message:** `random max must be positive`
**Used for:** `math.random(0)` or `math.random(-5)`
**Example:** `"math.random() max must be positive"`

---

### E8010 - random-max-less-than-min
**Type:** `random-max-less-than-min`
**Message:** `random max must exceed min`
**Used for:** `math.random(10, 5)`
**Example:** `"math.random() max must be greater than min"`

---

### E8011 - random-float-arg-count
**Type:** `random-float-arg-count`
**Message:** `random_float wrong argument count`
**Used for:** `math.random_float(1)`
**Example:** `"math.random_float() takes 0 or 2 arguments"`

---

### E8012 - avg-no-arguments
**Type:** `avg-no-arguments`
**Message:** `avg requires at least one value`
**Used for:** `math.avg()`
**Example:** `"math.avg() requires at least 1 argument"`

---

## Array Errors (E9xxx)

### E9001 - array-pop-empty
**Type:** `array-pop-empty`
**Message:** `cannot pop from empty array`
**Used for:** `arrays.pop(emptyArray)`
**Example:** `"arrays.pop() cannot pop from empty array"`

---

### E9002 - array-shift-empty
**Type:** `array-shift-empty`
**Message:** `cannot shift from empty array`
**Used for:** `arrays.shift(emptyArray)`
**Example:** `"arrays.shift() cannot shift from empty array"`

---

### E9003 - array-insert-out-of-bounds
**Type:** `array-insert-out-of-bounds`
**Message:** `insert index out of bounds`
**Used for:** `arrays.insert(arr, 100, val)`
**Example:** `"arrays.insert() index out of bounds"`

---

### E9004 - array-get-out-of-bounds
**Type:** `array-get-out-of-bounds`
**Message:** `get index out of bounds`
**Used for:** `arrays.get(arr, 100)`
**Example:** `"arrays.get() index out of bounds"`

---

### E9005 - array-set-out-of-bounds
**Type:** `array-set-out-of-bounds`
**Message:** `set index out of bounds`
**Used for:** `arrays.set(arr, 100, val)`
**Example:** `"arrays.set() index out of bounds"`

---

### E9006 - array-remove-at-out-of-bounds
**Type:** `array-remove-at-out-of-bounds`
**Message:** `remove_at index out of bounds`
**Used for:** `arrays.remove_at(arr, 100)`
**Example:** `"arrays.remove_at() index out of bounds"`

---

### E9007 - array-min-empty
**Type:** `array-min-empty`
**Message:** `cannot get min of empty array`
**Used for:** `arrays.min(emptyArray)`
**Example:** `"arrays.min() requires non-empty array"`

---

### E9008 - array-max-empty
**Type:** `array-max-empty`
**Message:** `cannot get max of empty array`
**Used for:** `arrays.max(emptyArray)`
**Example:** `"arrays.max() requires non-empty array"`

---

### E9009 - array-avg-empty
**Type:** `array-avg-empty`
**Message:** `cannot get avg of empty array`
**Used for:** `arrays.avg(emptyArray)`
**Example:** `"arrays.avg() requires non-empty array"`

---

### E9010 - array-sum-non-numeric
**Type:** `array-sum-non-numeric`
**Message:** `sum requires numeric array`
**Used for:** `arrays.sum(["a", "b"])`
**Example:** `"arrays.sum() requires numeric array"`

---

### E9011 - array-product-non-numeric
**Type:** `array-product-non-numeric`
**Message:** `product requires numeric array`
**Used for:** `arrays.product(["a", "b"])`
**Example:** `"arrays.product() requires numeric array"`

---

### E9012 - array-avg-non-numeric
**Type:** `array-avg-non-numeric`
**Message:** `avg requires numeric array`
**Used for:** `arrays.avg(["a", "b"])`
**Example:** `"arrays.avg() requires numeric array"`

---

### E9013 - array-range-step-zero
**Type:** `array-range-step-zero`
**Message:** `range step cannot be zero`
**Used for:** `arrays.range(0, 10, 0)`
**Example:** `"arrays.range() step cannot be zero"`

---

### E9014 - array-concat-requires-arrays
**Type:** `array-concat-requires-arrays`
**Message:** `concat requires array arguments`
**Used for:** `arrays.concat(arr, "string")`
**Example:** `"arrays.concat() requires arrays"`

---

### E9015 - array-zip-requires-arrays
**Type:** `array-zip-requires-arrays`
**Message:** `zip requires array arguments`
**Used for:** `arrays.zip(arr, "string")`
**Example:** `"arrays.zip() requires arrays"`

---

### E9016 - array-slice-invalid-indices
**Type:** `array-slice-invalid-indices`
**Message:** `slice requires integer indices`
**Used for:** `arrays.slice(arr, "a", "b")`
**Example:** `"arrays.slice() requires integer indices"`

---

### E9017 - array-repeat-invalid-count
**Type:** `array-repeat-invalid-count`
**Message:** `repeat requires integer count`
**Used for:** `arrays.repeat(val, "5")`
**Example:** `"arrays.repeat() requires integer count"`

---

### E9018 - array-join-invalid-separator
**Type:** `array-join-invalid-separator`
**Message:** `join requires string separator`
**Used for:** `arrays.join(arr, 123)`
**Example:** `"arrays.join() requires a string separator"`

---

## String Errors (E10xxx)

### E10001 - string-split-invalid-separator
**Type:** `string-split-invalid-separator`
**Message:** `split requires string separator`
**Used for:** `strings.split(str, 123)`
**Example:** `"strings.split() requires string arguments"`

---

### E10002 - string-join-invalid-array
**Type:** `string-join-invalid-array`
**Message:** `join requires array first argument`
**Used for:** `strings.join("str", ",")`
**Example:** `"strings.join() requires an array as first argument"`

---

### E10003 - string-replace-invalid-args
**Type:** `string-replace-invalid-args`
**Message:** `replace requires string arguments`
**Used for:** `strings.replace(str, 1, 2)`
**Example:** `"strings.replace() requires string arguments"`

---

### E10004 - string-index-invalid-args
**Type:** `string-index-invalid-args`
**Message:** `index requires string arguments`
**Used for:** `strings.index(str, 123)`
**Example:** `"strings.index() requires string arguments"`

---

### E10005 - string-contains-invalid-args
**Type:** `string-contains-invalid-args`
**Message:** `contains requires string arguments`
**Used for:** `strings.contains(str, 123)`
**Example:** `"strings.contains() requires string arguments"`

---

### E10006 - string-starts-with-invalid-args
**Type:** `string-starts-with-invalid-args`
**Message:** `starts_with requires string arguments`
**Used for:** `strings.starts_with(str, 123)`
**Example:** `"strings.starts_with() requires string arguments"`

---

### E10007 - string-ends-with-invalid-args
**Type:** `string-ends-with-invalid-args`
**Message:** `ends_with requires string arguments`
**Used for:** `strings.ends_with(str, 123)`
**Example:** `"strings.ends_with() requires string arguments"`

---

## Time Errors (E11xxx)

### E11001 - time-sleep-invalid-arg
**Type:** `time-sleep-invalid-arg`
**Message:** `sleep requires numeric argument`
**Used for:** `time.sleep("5")`
**Example:** `"time.sleep() requires a number"`

---

### E11002 - time-sleep-ms-invalid-arg
**Type:** `time-sleep-ms-invalid-arg`
**Message:** `sleep_ms requires integer argument`
**Used for:** `time.sleep_ms("100")`
**Example:** `"time.sleep_ms() requires an integer"`

---

### E11003 - time-format-invalid-format
**Type:** `time-format-invalid-format`
**Message:** `format requires string format`
**Used for:** `time.format(123)`
**Example:** `"time.format() requires a string format"`

---

### E11004 - time-format-invalid-timestamp
**Type:** `time-format-invalid-timestamp`
**Message:** `format requires integer timestamp`
**Used for:** `time.format("abc", "YYYY")`
**Example:** `"time.format() requires an integer timestamp"`

---

### E11005 - time-parse-failed
**Type:** `time-parse-failed`
**Message:** `failed to parse time string`
**Used for:** Invalid time string
**Example:** `"time.parse() failed: parsing time \"abc\""`

---

### E11006 - time-parse-invalid-args
**Type:** `time-parse-invalid-args`
**Message:** `parse requires string arguments`
**Used for:** `time.parse(123, 456)`
**Example:** `"time.parse() requires a string"`

---

### E11007 - time-make-invalid-args
**Type:** `time-make-invalid-args`
**Message:** `make requires integer arguments`
**Used for:** `time.make("2024", 1, 1)`
**Example:** `"time.make() requires integer arguments"`

---

### E11008 - time-add-invalid-timestamp
**Type:** `time-add-invalid-timestamp`
**Message:** `time add requires integer timestamp`
**Used for:** `time.add_days("abc", 5)`
**Example:** `"time.add_days() requires integer timestamp"`

---

### E11009 - time-add-invalid-amount
**Type:** `time-add-invalid-amount`
**Message:** `time add requires integer amount`
**Used for:** `time.add_days(ts, "5")`
**Example:** `"time.add_days() requires integer days"`

---

### E11010 - time-diff-invalid-args
**Type:** `time-diff-invalid-args`
**Message:** `diff requires integer timestamps`
**Used for:** `time.diff("a", "b")`
**Example:** `"time.diff() requires integer timestamps"`

---

### E11011 - time-is-leap-year-invalid-arg
**Type:** `time-is-leap-year-invalid-arg`
**Message:** `is_leap_year requires integer year`
**Used for:** `time.is_leap_year("2024")`
**Example:** `"time.is_leap_year() requires an integer year"`

---

### E11012 - time-days-in-month-invalid-args
**Type:** `time-days-in-month-invalid-args`
**Message:** `days_in_month requires integer arguments`
**Used for:** `time.days_in_month("2024", "1")`
**Example:** `"time.days_in_month() requires integer arguments"`

---

### E11013 - time-elapsed-invalid-arg
**Type:** `time-elapsed-invalid-arg`
**Message:** `elapsed_ms requires integer tick`
**Used for:** `time.elapsed_ms("abc")`
**Example:** `"time.elapsed_ms() requires integer tick"`

---

## Map Errors (E12xxx)

### E12001 - map-requires-map
**Type:** `map-requires-map`
**Message:** `function requires a map argument`
**Used for:** Passing non-map to map function
**Example:** `"maps.len() requires a map"`

---

### E12002 - map-key-not-hashable
**Type:** `map-key-not-hashable`
**Message:** `map key must be a hashable type`
**Used for:** Using non-hashable type as map key
**Example:** `"maps.get() key must be a hashable type (string, int, bool, char)"`

---

### E12003 - map-immutable
**Type:** `map-immutable`
**Message:** `cannot modify immutable map`
**Used for:** Attempting to modify a const map
**Example:** `"cannot modify immutable map (declared as const)"`

---

### E12004 - map-invalid-pair
**Type:** `map-invalid-pair`
**Message:** `map element must be a [key, value] pair`
**Used for:** `maps.from_array()` with invalid element format
**Example:** `"maps.from_array() element 0 must be a [key, value] pair"`

---

### E12005 - map-value-not-hashable
**Type:** `map-value-not-hashable`
**Message:** `map value is not hashable and cannot become a key`
**Used for:** `maps.invert()` when value cannot be used as key
**Example:** `"maps.invert() value {1, 2, 3} is not hashable and cannot become a key"`

---

### E12006 - map-key-not-found
**Type:** `map-key-not-found`
**Message:** `key not found in map`
**Used for:** Accessing a key that doesn't exist in a map
**Example:** `"key \"foo\" not found in map"`

---

### E12007 - map-key-not-found-compound
**Type:** `map-key-not-found-compound`
**Message:** `key not found in map for compound assignment`
**Used for:** Using compound assignment (`+=`, etc.) on non-existent key
**Example:** `"key not found in map for compound assignment"`

---

## Warnings (W1xxx - W3xxx)

### W1001 - unused-variable
**Type:** `unused-variable`
**Message:** `variable declared but not used`
**Example:** `"variable 'x' is declared but never used"`

---

### W1002 - unused-import
**Type:** `unused-import`
**Message:** `module imported but not used`
**Example:** `"module 'math' is imported but never used"`

---

### W1003 - unused-function
**Type:** `unused-function`
**Message:** `function declared but not called`
**Example:** `"function 'helper' is declared but never called"`

---

### W1004 - unused-parameter
**Type:** `unused-parameter`
**Message:** `parameter declared but not used`
**Example:** `"parameter 'x' is declared but never used"`

---

### W2001 - unreachable-code
**Type:** `unreachable-code`
**Message:** `code will never execute`
**Used for:** Code after return/break/continue
**Example:** `"unreachable code after return/break/continue"`
**Suppressible:** `@suppress(W2001)` or `@suppress(unreachable_code)`

---

### W2002 - shadowed-variable
**Type:** `shadowed-variable`
**Message:** `variable shadows outer scope`
**Example:** `"variable 'x' shadows variable in outer scope"`

---

### W2003 - missing-return
**Type:** `missing-return`
**Message:** `function may not return value`
**Used for:** Function with return type but no return
**Example:** `"Function 'foo' declares return type(s) but has no return statement"`
**Suppressible:** `@suppress(W2003)` or `@suppress(missing_return)`

---

### W2004 - implicit-type-conversion
**Type:** `implicit-type-conversion`
**Message:** `implicit type conversion occurring`
**Example:** `"implicit conversion from int to float"`

---

### W2005 - deprecated-feature
**Type:** `deprecated-feature`
**Message:** `using deprecated feature`
**Example:** `"feature 'xyz' is deprecated, use 'abc' instead"`

---

### W3001 - empty-block
**Type:** `empty-block`
**Message:** `block statement is empty`
**Example:** `"if statement has empty body"`

---

### W3002 - redundant-condition
**Type:** `redundant-condition`
**Message:** `condition is always true/false`
**Example:** `"condition 'true' is always true"`

---

---

## Summary Statistics

| Category | Count | Range |
|----------|-------|-------|
| Lexer Errors | 16 | E1001-E1016 |
| Parse Errors | 31 | E2001-E2031 |
| Type Errors | 18 | E3001-E3018 |
| Reference Errors | 11 | E4001-E4011 |
| Runtime Errors | 15 | E5001-E5015 |
| Import Errors | 4 | E6001-E6004 |
| Builtin Errors | 9 | E7001-E7009 |
| Math Errors | 12 | E8001-E8012 |
| Array Errors | 18 | E9001-E9018 |
| String Errors | 7 | E10001-E10007 |
| Time Errors | 13 | E11001-E11013 |
| Map Errors | 7 | E12001-E12007 |
| Warnings | 12 | W1001-W3002 |
| **Total** | **173** | |

---

## Implementation Checklist

When implementing these errors in code:

1. [ ] Update `pkg/errors/codes.go` with all new error codes
2. [ ] Update `pkg/errors/suggestions.go` with suggestions for each error
3. [ ] Update `pkg/lexer/lexer.go` to use new lexer error codes (E1xxx)
4. [ ] Update `pkg/parser/parser.go` to use new parse error codes (E2xxx)
5. [ ] Update `pkg/typechecker/typechecker.go` to use type error codes (E3xxx)
6. [ ] Update `pkg/interpreter/evaluator.go` to use reference and runtime error codes (E4xxx, E5xxx)
7. [ ] Update `pkg/interpreter/builtins.go` to use builtin error codes (E7xxx)
8. [ ] Update `pkg/interpreter/lib_math.go` to use math error codes (E8xxx)
9. [ ] Update `pkg/interpreter/lib_arrays.go` to use array error codes (E9xxx)
10. [ ] Update `pkg/interpreter/lib_strings.go` to use string error codes (E10xxx)
11. [ ] Update `pkg/interpreter/lib_time.go` to use time error codes (E11xxx)
12. [x] Update `pkg/stdlib/maps.go` to use map error codes (E12xxx)
