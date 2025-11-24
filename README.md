# EZ Language

A brutally simple interpreted programming language written in Go. The core philosophy is radical simplicity - no magic, no surprises, just straightforward code that does exactly what it says.

## Development Roadmap

### Phase 1: Project Setup
- [x] Set up Go project structure (cmd/ez, pkg/lexer, pkg/parser, pkg/interpreter)

### Phase 2: Lexer
- [x] Define token types and implement lexer

### Phase 3: AST & Parser
- [x] Define AST node types
- [x] Implement parser (expressions, statements, declarations)

### Phase 4: Interpreter
- [x] Implement interpreter/evaluator
- [x] Add built-in functions
- [x] Add extensive libraries(strings, math, time, etc)
- [ ] Implement type checking

### Phase 5: CLI
- [ ] Create REPL and file execution modes

## First Milestone

Get "Hello World" working:

```
import @std

do main() {
    using std
    println("Hello, World!")
}
```

## Quick Start

```bash
# Build
go build -o ez.bin ./cmd/...

# Run a program
./ez.bin run examples/example.ez
```

---

## Core Philosophy
- **Simplicity above all else** - simpler than Python, C, or anything else
- **Explicit over implicit** - you see exactly what's happening
- **No object-oriented programming** - procedural/functional style only
- **Small feature set that composes well**
- **General-purpose language** - like C, Go, or Odin, suitable for building real programs

## Type System
- **Static typing** - types are checked before runtime
- **Strong typing** - no implicit type coercion
- **No type inference (for now)** - explicit type declarations for clarity

## Mutability & Variable Declaration

### Syntax
Variables use explicit keywords to declare mutability:

```
temp x int = 5      // mutable variable
const y int = 10    // immutable constant
```

### Rules
- **Mutable variables:** Use `temp` keyword
  - Can be reassigned after declaration
  - Can be declared without initialization (uses default value)
  - Example: `temp age int;` (defaults to 0)

- **Immutable constants:** Use `const` keyword
  - Cannot be reassigned after declaration
  - **Must be initialized with a value at declaration**
  - Example: `const PI float = 3.14159`

### Declaration Format
```
keyword name type = value
```

Examples:
```
temp counter int = 0
temp name string = "Marshall"
temp active bool = true
const MAX_USERS int = 100
const PI float = 3.14159
```

## Core Data Types
- Numbers (integers, floats)
- Strings
- Characters (char)
- Booleans
- Arrays
- Structs
- Enums
- Void/null

**Explicitly excluded:**
- No objects or classes
- No inheritance
- No methods on types (just functions)

## Strings and Characters

### Character Type
- `char` - single character type
- Single quotes for char literals: `'A'`
- Double quotes for string literals: `"Hello"`

### String Operations

**Built-in language features:**
```
// Concatenation with + operator
temp greeting string = "Hello, " + "World"

// Indexing (returns char)
temp name string = "Marshall"
temp firstChar char = name[0]  // 'M'

// Mutation (if temp)
temp word string = "Bob"
word[0] = 'R'  // now "Rob"

const title string = "Mr."
title[0] = 'D'  // ERROR - const is immutable

// Formatted output (printf-style)
printf("Name: %s, Age: %d\n", name, age)

// Length (built-in function)
temp length int = len(name)

// Escape sequences
temp message string = "Line 1\nLine 2\tTabbed\n\"Quoted\""
```

**Standard library (strings module):**
- String utilities in `strings` library (similar to Odin)
- Functions include: `upper()`, `lower()`, `contains()`, `split()`, `trim()`, `replace()`, etc.

### Examples
```
import @std
import @strings

do main() {
    temp name string = "Marshall"
    temp initial char = 'M'

    // Built-in operations
    temp length int = len(name)
    temp greeting string = "Hello, " + name
    std.println(greeting)

    // Character access
    temp first char = name[0]
    if first == 'M' {
        std.println("Starts with M!")
    }

    // Library functions
    temp upper string = strings.upper(name)
}
```

## Enums

### Syntax
Enums must be declared as constants and can have optional attributes to control behavior:

```
const ENUM_NAME enum {
    VALUE1
    VALUE2
    VALUE3
}
```

### Attributes
Enums can use attributes to specify type and increment behavior:

```
@(type, skip, increment_value)
```

### Rules
- **Must be declared as const** - `const ENUM_NAME enum { }`
- **Default type:** `int` with increment of 1
- **Default first value:** 0 (unless manually specified)
- **Manual override:** Any value can be manually assigned
- **After override:** Subsequent values continue from override + increment
- **String enums:** Must have all values manually assigned by programmer
- **Access values:** Use dot notation: `ENUM_NAME.VALUE`

### Examples

**Default integer enum:**
```
const STATUS enum {
    PENDING     // 0
    ACTIVE      // 1
    INACTIVE    // 2
}

temp currentStatus int = STATUS.ACTIVE  // 1
```

**Enum with skip attribute:**
```
@(int, skip, 5)
const PRIORITY enum {
    LOW         // 0
    MEDIUM      // 5
    HIGH        // 10
}
```

**Manual value override:**
```
@(int, skip, 10)
const ERROR_CODES enum {
    SUCCESS     // 0
    WARNING     // 10
    ERROR = 100 // 100 (manual override)
    CRITICAL    // 110 (continues from 100 + 10)
}
```

**String enum (all values required):**
```
@(string)
const COLORS enum {
    RED = "red"
    GREEN = "green"
    BLUE = "blue"
}

temp favoriteColor string = COLORS.RED  // "red"
```

**Float enum:**
```
@(float, skip, 0.5)
const PRECISION enum {
    LOW = 0.0
    MEDIUM      // 0.5
    HIGH        // 1.0
}
```

### Valid Enum Types
- `int` (default)
- `float`
- `string`
- Any basic data type

## Structs

### Definition Syntax
```
TypeName struct {
    fieldName type
    anotherField type
}
```

### Creating Instances

**Option 1: Literal initialization**
```
temp instance TypeName = TypeName{field1: value1, field2: value2}
```

**Option 2: new() with field assignment**
```
temp instance TypeName = new(TypeName)
instance.field1 = value1
instance.field2 = value2
```

### Rules
- Struct fields are accessed with dot notation: `instance.fieldName`
- **new() initializes all fields to default values:**
  - Numbers: `0`
  - Strings: `""` (empty string)
  - Booleans: `false`
  - Arrays: `{}` (empty)
  - Pointers: `nil`
- **Literal initialization does not require all fields** - unspecified fields get default values
- **Mutability follows variable declaration:**
  - `temp` structs: fields can be modified
  - `const` structs: fields cannot be modified

### Examples
```
// Define a struct
Person struct {
    name string
    age int
    email string
}

// Literal initialization (partial fields OK)
temp p1 Person = Person{name: "Marshall", age: 30}
// p1.email is "" by default

// Using new()
temp p2 Person = new(Person)
p2.name = "Bob"
p2.age = 25
// p2.email is ""

// Accessing fields
print(p1.name)     // "Marshall"
p1.age = 31        // OK - p1 is temp

// Const struct
const p3 Person = Person{name: "Alice", age: 30}
p3.age = 31        // ERROR - p3 is const

// Nested structs
Address struct {
    street string
    city string
}

Person struct {
    name string
    address Address
}

temp person Person = Person{
    name: "Marshall",
    address: Address{city: "Austin"}
}
print(person.address.city)  // "Austin"
```

### Future Considerations
- Pointers (`*Type`) - deferred for later versions

## Error Handling

### Philosophy
Following Go/Odin's explicit error handling pattern - errors are values that must be explicitly handled.

### Multiple Return Values
Functions can return multiple values using parentheses in the return type:

```
do function_name() -> (returnType1, returnType2) {
    return value1, value2
}
```

### Error Type
- Built-in `std.Error` struct in standard library
- Error type is **constant** and cannot be modified
- Contains at minimum: `message` and `code` fields
- Returns `nil` when no error occurs

### Rules
- **Multiple returns must be captured or explicitly ignored**
- Use comma-separated variables to unpack (NO parentheses on left side)
- Use `@ignore` to explicitly discard a return value
- Check errors with `!= nil`
- `nil` keyword represents null/nothing (may change to `null`, `void`, or `nothing`)

### Examples

**Basic error handling:**
```
import @std

do divide(a, b int) -> (int, std.Error) {
    if b == 0 {
        return 0, std.Error{message: "division by zero", code: 1}
    }
    return a / b, nil
}

do main() {
    temp result, err = divide(10, 0)

    if err != nil {
        std.println("Error occurred")
        return
    }

    std.println(result)
}
```

**Ignoring return values:**
```
import @std

do some_function() -> (std.Error, string) {
    return std.Error{message: "failed", code: 1}, "Error message"
}

do caller() {
    // Ignore the string, keep error
    temp err, @ignore = some_function()

    if err != nil {
        std.println("ERROR")
    }

    // Ignore error, keep string (not recommended!)
    temp @ignore, msg = some_function()

    // ERROR: Must capture or ignore all returns
    some_function()  // INVALID
}
```

**Custom error structs:**
```
// Users can create their own error types
CustomError struct {
    message string
    timestamp int
    context string
}

do custom_operation() -> (bool, CustomError) {
    return false, CustomError{
        message: "operation failed",
        timestamp: 1234567890,
        context: "user input"
    }
}
```

## Import and Module System

### Import Syntax
```
import @module_name
import alias@module_name
```

### Using Keyword
Bring a module's namespace into scope to avoid dot notation (similar to Odin):

```
using module_name
```

### Rules
- **Import with @ prefix:** `import @std` for standard library modules
- **Aliasing:** `import net@networking` - use `net.function()` instead of `networking.function()`
- **Using scope:** Can be file-scoped or function-scoped
  - File-scoped: `using` at top of file applies to entire file
  - Function-scoped: `using` inside function applies only to that function
- **Multiple using:** Can use multiple namespaces in same scope
- **No name conflicts:** Users cannot create functions/variables that share names with built-in functions or keywords
- **Module creation:** Create `.ez` files with functions - they become importable modules

### Examples

**Basic imports:**
```
import @std
import @strings
import @math

do main() {
    std.println("Hello, World!")
    temp upper string = strings.upper("hello")
    temp pi float = math.pi
}
```

**Import aliasing:**
```
import @std
import net@networking

do main() {
    std.println("Starting server...")
    net.connect("localhost:8080")
}
```

**Using keyword - function scope:**
```
import @std
import @strings

do process_text(text string) {
    using std
    using strings

    // No prefixes needed within this function
    println("Processing...")
    temp upper string = upper(text)
    println(upper)
}

do other_function() {
    // Must use prefixes here
    std.println("Different function")
}
```

**Using keyword - file scope:**
```
import @std
using std

// Available for entire file
do main() {
    println("Hello")  // No std. needed
}

do other() {
    println("World")  // Still no std. needed
}
```

### Module System Philosophy
- **Simple file-based modules** (Go-style)
- Filename becomes the module name
- All functions in a file are accessible when imported
- No package declarations or header files needed
- Import path is relative to project or standard library

## Built-in Functions

### Global Built-ins (no import required)
```
len(collection) -> int           // Length of arrays/strings
typeof(variable) -> string       // Get type of variable
input() -> string                // Read user input
```

### Type Conversion Functions
```
int(value) -> int               // Convert to int
float(value) -> float           // Convert to float
string(value) -> string         // Convert to string
```

### Memory Functions
```
new(Type) -> Type               // Allocate new instance (uses Go's GC)
```

### Standard Library I/O (requires `import @std`)
```
std.println(args...)             // Print with newline
std.print(args...)               // Print without newline
std.read_int() -> int            // Read integer from input
```

### Examples
```
import @std

do main() {
    // I/O (requires std prefix or using std)
    std.println("Hello, World!")
    std.print("No newline")
    temp name string = input()       // Global built-in

    // Utility (global built-ins)
    temp numbers [int] = {1, 2, 3, 4, 5}
    temp length int = len(numbers)
    temp nameLen int = len("Marshall")
    temp t string = typeof(length)   // "int"

    // Type conversion (global built-ins)
    temp x int = int(3.14)           // 3
    temp y float = float(10)         // 10.0
    temp s string = string(42)       // "42"

    // Memory
    temp p Person = new(Person)
    p.name = "Marshall"
}
```

## Arrays

### Syntax
Arrays use square brackets for type and curly braces for literal values:

**Dynamic arrays (temp):**
```
temp name [type] = {values}
```

**Fixed-size arrays (const):**
```
const name [type, size] = {values}
```

### Rules
- **temp arrays:** Can grow and shrink, elements can be modified
- **const arrays:** Fixed size, cannot be modified, must specify size and provide all values
- Array elements are accessed with bracket notation: `array[index]`
- Array literals use curly braces: `{1, 2, 3}`

### Examples
```
// Dynamic array
temp numbers [int] = {1, 2, 3, 4, 5}
numbers[0] = 10        // OK - modify element
numbers.append(6)      // OK - grow array

// Fixed-size array
const daysOfWeek [string, 7] = {"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
daysOfWeek[0] = "Monday"  // ERROR - const array is immutable
daysOfWeek.append("Day8") // ERROR - cannot grow const array

// Empty dynamic array
temp empty [int] = {}

// Multi-dimensional arrays
temp matrix [[int]] = {{1, 2}, {3, 4}, {5, 6}}
```

**Note:** Array length is obtained using the `len()` built-in function.

## Memory Management
- **Starting approach:** Leverage Go's garbage collector under the hood
- **Language semantics:** Design as if manual memory management
- **Future consideration:** Could swap in real manual memory management later
- **Rationale:** Manual memory is complex to implement; GC allows focus on language design first

## Functions

### Declaration Syntax
Functions are declared using the `do` keyword:

```
do function_name(parameters) -> return_type {
    // function body
}
```

### Rules
- Use `do` keyword to **define** functions
- Call functions normally without any keyword: `function_name()`
- Return type is specified with `->` arrow syntax
- **If no return type specified, function returns void** (implicit)
- **Function parameters are immutable (const)** - cannot be modified inside function body
- **Type sharing:** Parameters of the same type can share a type declaration (Go-style)

### Examples
```
do main() {
    // implicit void return
    hello_world()
    print("Done")
}

do hello_world() -> string {
    return "Hello, World"
}

do add(x, y int) -> int {
    // type sharing - both x and y are int
    return x + y
}

do calculate(a, b, c float, divisor int) -> float {
    // a, b, c are float; divisor is int
    return (a + b + c) / divisor
}

do process(x int) {
    x = x + 1  // ERROR: parameters are immutable
    temp y int = x + 1  // OK: create new temp variable
}
```

## Control Flow

### Conditional Statements
```
if condition {
    // code
} or other_condition {
    // code
} otherwise {
    // code
}
```

**Rules:**
- No parentheses required around conditions
- `or` keyword for else-if branches
- `otherwise` keyword for else branch
- Both `or` and `otherwise` blocks are optional

**Membership operators:**
- `in` - check if value exists in array
- `not_in` or `!in` - check if value does NOT exist in array (both syntaxes valid)

**Examples:**
```
if x > 10 {
    print("big")
} or x > 5 {
    print("medium")
} otherwise {
    print("small")
}

if is_valid {
    process()
}

// Membership checking
temp numbers [int] = {1, 2, 3, 4, 5}

if 3 in numbers {
    print("Found it!")
}

if 10 not_in numbers {
    print("Not found - using not_in")
}

if 10 !in numbers {
    print("Not found - using !in")
}
```

### Loops

**Range-based for loop:**
```
for i in range(start, end) {
    // code
}
```

**Rules:**
- Range is INCLUSIVE - `range(1, 100)` runs from 1 to 100 (not 99!)
- Loop variable `i` can be implicitly or explicitly typed
- Type determines how range behaves (int vs float increments)

**Examples:**
```
// Implicit int type
for i in range(1, 10) {
    print(i)  // 1, 2, 3, ... 10
}

// Explicit int type
for i int in range(1, 10) {
    print(i)
}

// Float type
for i float in range(0.1, 0.9) {
    print(i)  // 0.1, 0.2, 0.3, ... 0.9
}
```

**While loop:**
```
as_long_as condition {
    // code
}
```

**Rules:**
- No parentheses required around condition
- Loop continues as long as condition is true

**Example:**
```
temp x int = 0
as_long_as x < 10 {
    print(x)
    x = x + 1
}
```

**Infinite loop:**
```
loop {
    // code
    if done { break }
}
```

**Rules:**
- Runs forever until explicitly broken out of
- Use `break` to exit loop

**Example:**
```
temp counter int = 0
loop {
    print(counter)
    counter = counter + 1
    if counter >= 100 {
        break
    }
}
```

### For-each loop
```
for_each item in collection {
    // code
}
```

**Rules:**
- Iterates over each element in a collection (array, list, etc.)
- Loop variable contains the current item

**Example:**
```
temp numbers [int] = {1, 2, 3, 4, 5}
for_each num in numbers {
    print(num)
}

temp names [string] = {"Alice", "Bob", "Charlie"}
for_each name in names {
    print(name)
}
```

## Syntax Style
- Familiar feel similar to Odin/C/Go
- More verbose than these languages, but clearer
- "Makes sense in my brain" - explicit and readable
- No syntactic sugar that obscures what's happening

## Comments
```
// Single-line comment

/*
Multi-line comment
spanning multiple lines
*/
```

## Semicolons
- **Optional** - statements do not require semicolons
- Newlines serve as statement terminators
- Semicolons can be used if desired but are not enforced

## Operators

### Arithmetic Operators
```
+   addition
-   subtraction
*   multiplication
/   division
%   modulo/remainder
```

### Assignment Operators
```
=   assignment
+=  add and assign
-=  subtract and assign
*=  multiply and assign
/=  divide and assign
%=  modulo and assign
```

### Comparison Operators
```
==  equal to
!=  not equal to
<   less than
>   greater than
<=  less than or equal to
>=  greater than or equal to
```

### Logical Operators
```
&&  logical AND
||  logical OR
!   logical NOT
```

### Membership Operators
```
in      check if value exists in array
not_in  check if value does NOT exist in array
!in     check if value does NOT exist in array (alternative syntax)
```

### Increment/Decrement Operators
```
++  increment
--  decrement
```

### Examples
```
// Arithmetic
temp x int = 10 + 5
temp y int = 20 - 3
temp z int = 4 * 2
temp result int = 100 / 10
temp remainder int = 10 % 3

// Assignment
temp count int = 0
count += 5    // count is now 5
count -= 2    // count is now 3
count *= 2    // count is now 6

// Comparison
if x == 10 { }
if y != 5 { }
if x < 100 { }

// Logical
if x > 5 && y < 20 { }
if isValid || isAdmin { }
if !isEmpty { }

// Increment/Decrement
temp i int = 0
i++  // i is now 1
i--  // i is now 0

// Membership
temp numbers [int] = {1, 2, 3}
if 2 in numbers { }
if 5 not_in numbers { }
if 5 !in numbers { }
```

## Killer Feature
**Radical simplicity** - the language should be so simple that anyone can understand any program written in it at a glance. Every feature must justify its existence.

## Implementation Details
- **Implementation language:** Go
- **Execution model:** Interpreted (no separate compilation step)
- **Target use case:** General-purpose programming

## Open Questions
1. Should functions be first-class citizens?
2. Concurrency primitives?
3. Pointers and manual memory management?
