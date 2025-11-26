# EZ Language

A brutally simple interpreted programming language written in Go. The core philosophy is radical simplicity - no magic, no surprises, just straightforward code that does exactly what it says.

---

## Table of Contents
- [Introduction](#introduction)
- [Quick Start](#quick-start)
- [Current Features](#current-features)
- [Language Syntax](#language-syntax)
- [Planned Features](#planned-features)
- [Contributing](#contributing)

---

## Introduction

### Core Philosophy
- **Simplicity above all else** - simpler than Python, C, or anything else
- **Explicit over implicit** - you see exactly what's happening
- **No object-oriented programming** - procedural/functional style only
- **Small feature set that composes well**
- **General-purpose language** - like C, Go, or Odin, suitable for building real programs

### Type System
- **Static typing** - types are checked at runtime
- **Strong typing** - no implicit type coercion
- **No type inference** - explicit type declarations for clarity

### Design Goals
- **Radically simple** - anyone can understand any program at a glance
- **No magic** - every feature must justify its existence
- **Comprehensive standard library** - programmers shouldn't need custom helper libraries
- **Clear error messages** - Rust-inspired error formatting with helpful context

---

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Build the interpreter
go build -o ez.bin ./cmd/ez

# Run your first program
./ez.bin run examples/hello.ez
```

### Hello World

Create a file called `hello.ez`:

```ez
import @std

do main() {
    using std
    println("Hello, World!")
}
```

Run it:
```bash
./ez.bin run hello.ez
```

### Basic Example

```ez
import @std

do main() {
    using std

    // Variables
    temp name string = "EZ"
    const version float = 0.1

    println("Welcome to", name, "version", version)

    // Arrays
    temp numbers [int] = {1, 2, 3, 4, 5}
    println("Numbers:", numbers)

    // Functions
    temp result int = add(10, 20)
    println("10 + 20 =", result)
}

do add(x, y int) -> int {
    return x + y
}
```

---

## Current Features

### ✅ Implemented & Working

#### Core Language Features
- **Variables & Constants**
  - `temp` for mutable variables
  - `const` for immutable constants
  - Default values for uninitialized variables
  - Variable shadowing in nested scopes

- **Data Types**
  - Primitives: `int`, `float`, `string`, `char`, `bool`
  - Numeric separators: underscores for readability (`1_000_000`)
  - Arrays: dynamic `[type]` and fixed-size `[type, size]`
  - Structs: user-defined types with fields
  - Enums: integer, float, and string enums with attributes

- **Operators**
  - Arithmetic: `+`, `-`, `*`, `/`, `%`
  - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
  - Logical: `&&`, `||`, `!`
  - Assignment: `=`, `+=`, `-=`, `*=`, `/=`, `%=`
  - Increment/Decrement: `++`, `--`
  - Membership: `in`, `!in`, `not_in`

- **Control Flow**
  - `if` / `or` / `otherwise` conditionals
  - `for` loops with `range(start, end)`
  - `for_each` loops over arrays and strings
  - `break` and `continue` statements

- **Functions**
  - `do` keyword for function declarations
  - Type sharing in parameters: `do add(x, y int)`
  - Return types with `->` syntax
  - Multiple parameters and return values

- **Modules & Imports**
  - `import @module` syntax
  - Module aliasing: `import alias@module`
  - `using` keyword for namespace convenience
  - Standard library: `@std`, `@arrays`

- **Enums**
  - Basic integer enums with auto-increment
  - Attributes: `@(type, skip, increment_value)`
  - Support for int, float, and string types
  - Manual value overrides
  - Dot notation access: `STATUS.ACTIVE`

- **Structs**
  - Struct type definitions with `const Name struct {}`
  - Literal initialization: `Person{name: "Alice", age: 30}`
  - Field access with dot notation
  - Nested structs

- **Strings & Characters**
  - String concatenation with `+`
  - String interpolation with dollar-brace syntax
  - String indexing and character access
  - String mutation for `temp` variables
  - Character type with single quotes: `'A'`
  - String iteration with `for_each`

- **Arrays**
  - Dynamic arrays: `temp arr [int] = {1, 2, 3}`
  - Fixed-size arrays: `const arr [int, 3] = {1, 2, 3}`
  - Array indexing and assignment
  - `arrays.push()` and `arrays.pop()` functions
  - Array iteration with `for_each`

- **Built-in Functions**
  - `len(collection)` - get length of arrays/strings
  - `typeof(value)` - get type as string
  - `println(...)` - print with newline (std module)
  - `print(...)` - print without newline (std module)

- **Error System**
  - Color-coded error messages
  - Error codes (E1xxx, E2xxx, E3xxx)
  - Line and column information
  - Helpful error context
  - `@suppress(code)` attribute for warnings

#### Partially Implemented
- **Arrays** - Multi-dimensional arrays partially supported (issue #5)
- **Standard Library** - Only `@std` and `@arrays` currently available

---

## Language Syntax

### Variables & Constants

```ez
// Mutable variables (can be reassigned)
temp x int = 10
temp name string = "Alice"
temp active bool = true

// Immutable constants (cannot be reassigned)
const PI float = 3.14159
const MAX_SIZE int = 100

// Uninitialized variables get default values
temp count int        // defaults to 0
temp message string   // defaults to ""
temp flag bool        // defaults to false
```

### Data Types

```ez
// Primitives
temp number int = 42
temp price float = 19.99
temp text string = "hello"
temp letter char = 'A'
temp isActive bool = true

// Numeric separators for readability
temp million int = 1_000_000
temp billion int = 7_800_000_000
temp pi float = 3.141_592_653
temp money float = 1_234.56

// Arrays
temp numbers [int] = {1, 2, 3, 4, 5}
const days [string, 7] = {"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Structs
const Person struct {
    name string
    age int
}

temp alice Person = Person{name: "Alice", age: 30}

// Enums
const STATUS enum {
    PENDING   // 0
    ACTIVE    // 1
    INACTIVE  // 2
}

temp status int = STATUS.ACTIVE
```

### Operators

```ez
// Arithmetic
temp sum int = 10 + 5
temp diff int = 20 - 3
temp product int = 4 * 2
temp quotient int = 100 / 10
temp remainder int = 10 % 3

// Compound assignment
temp count int = 0
count += 5    // count = 5
count -= 2    // count = 3
count *= 2    // count = 6
count /= 3    // count = 2

// Increment/Decrement
temp i int = 0
i++  // i = 1
i--  // i = 0

// Comparison
if x == 10 { }
if y != 5 { }
if x < 100 { }

// Logical
if x > 5 && y < 20 { }
if isValid || isAdmin { }
if !isEmpty { }

// Membership
temp nums [int] = {1, 2, 3}
if 2 in nums { }
if 5 !in nums { }
```

### Control Flow

```ez
// If/or/otherwise
if x > 10 {
    println("big")
} or x > 5 {
    println("medium")
} otherwise {
    println("small")
}

// For loop with range
for i in range(0, 10) {
    println(i)
}

// For each loop
temp numbers [int] = {1, 2, 3, 4, 5}
for_each num in numbers {
    println(num)
}

// Break and continue
for i in range(0, 100) {
    if i % 2 == 0 {
        continue  // skip even numbers
    }
    if i > 50 {
        break  // stop at 50
    }
    println(i)
}
```

### Functions

```ez
// Function declaration
do greet(name string) {
    println("Hello,", name)
}

// Function with return type
do add(x, y int) -> int {
    return x + y
}

// Type sharing for parameters
do calculate(a, b, c float, divisor int) -> float {
    return (a + b + c) / divisor
}

// Calling functions
greet("Alice")
temp result int = add(5, 10)
```

### Modules & Imports

```ez
// Import standard library
import @std

// Import with alias
import arr@arrays

do main() {
    // Use with prefix
    std.println("Hello")
    arr.push(numbers, 10)

    // Or use 'using' for convenience
    using std
    println("No prefix needed")
}
```

### Enums

```ez
// Basic enum (int, starts at 0)
const STATUS enum {
    PENDING   // 0
    ACTIVE    // 1
    INACTIVE  // 2
}

// Enum with skip attribute
@(int, skip, 5)
const PRIORITY enum {
    LOW      // 0
    MEDIUM   // 5
    HIGH     // 10
}

// Enum with manual override
@(int, skip, 10)
const ERROR_CODES enum {
    SUCCESS   // 0
    WARNING   // 10
    ERROR = 100
    CRITICAL  // 110
}

// String enum (all values required)
@(string)
const COLORS enum {
    RED = "red"
    GREEN = "green"
    BLUE = "blue"
}

// Float enum
@(float, skip, 0.5)
const PRECISION enum {
    LOW = 0.0
    MEDIUM    // 0.5
    HIGH      // 1.0
}

// Using enums
temp status int = STATUS.ACTIVE
temp color string = COLORS.RED
```

### Structs

```ez
// Define a struct
const Person struct {
    name string
    age int
    email string
}

// Create instance with literal
temp p1 Person = Person{name: "Alice", age: 30}

// Access and modify fields
println(p1.name)
p1.age = 31

// Nested structs
const Address struct {
    street string
    city string
}

const Employee struct {
    name string
    address Address
}

temp emp Employee = Employee{
    name: "Bob",
    address: Address{city: "Austin"}
}
```

### Arrays

```ez
// Dynamic arrays
temp numbers [int] = {1, 2, 3, 4, 5}
numbers[0] = 10          // modify element
temp first int = numbers[0]

// Array operations
import @arrays

arrays.push(numbers, 6)  // append element
temp last int = arrays.pop(numbers)

// Iterate over arrays
for_each num in numbers {
    println(num)
}

// Fixed-size arrays (immutable)
const days [string, 7] = {"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
```

### Strings & Characters

```ez
// String operations
temp greeting string = "Hello, " + "World"
temp name string = "Alice"
temp firstChar char = name[0]  // 'A'

// String mutation (temp only)
temp word string = "Bob"
word[0] = 'R'  // now "Rob"

// String iteration
temp text string = "EZ"
for_each ch in text {
    println(ch)
}
```

### Comments

```ez
// Single-line comment

/*
Multi-line comment
spanning multiple lines
*/
```

### Error Suppression

```ez
// Suppress warnings
@suppress(W2001)
do example() {
    return 5
    temp x int = 10  // No unreachable code warning
}

// Suppress multiple warnings
@suppress(W2001, unused_variable)
do another_example() {
    // warnings suppressed
}
```

---

## Planned Features

The following features are planned but not yet implemented. See the [Issues](https://github.com/SchoolyB/EZ/issues) page for details.

### High Priority
- **Multiple Return Values & Error Handling** ([#32](https://github.com/SchoolyB/EZ/issues/32))
  - Go-style multiple returns: `do func() -> (int, Error)`
  - Explicit error handling with `nil` checks
  - `@ignore` for unused return values

- **new() Function** ([#33](https://github.com/SchoolyB/EZ/issues/33))
  - Allocate structs with default values
  - `temp p Person = new(Person)`

- **Standard Library Expansion** ([#37](https://github.com/SchoolyB/EZ/issues/37))
  - `@strings` - string manipulation functions
  - `@math` - mathematical operations and constants
  - `@time` - time operations and utilities
  - `@io` - file I/O operations

### Medium Priority
- **as_long_as Loop** ([#38](https://github.com/SchoolyB/EZ/issues/38))
  - While-style loops: `as_long_as condition { }`

- **loop Keyword** ([#39](https://github.com/SchoolyB/EZ/issues/39))
  - Infinite loops: `loop { }`

- **Keyword Aliasing** ([#40](https://github.com/SchoolyB/EZ/issues/40))
  - Custom aliases: `@alias(otherwise, else)`
  - Makes language more accessible

- **Circular Import Support** ([#41](https://github.com/SchoolyB/EZ/issues/41))
  - Web-style imports without cyclical errors

- **Type Conversion Functions** ([#42](https://github.com/SchoolyB/EZ/issues/42))
  - `int()`, `float()`, `string()` conversions

- **User Input Functions** ([#43](https://github.com/SchoolyB/EZ/issues/43))
  - `input()` for string input
  - `std.read_int()` for integer input

### Lower Priority
- **REPL Mode** ([#45](https://github.com/SchoolyB/EZ/issues/45))
  - Interactive interpreter
  - Testing and learning tool

- **Privacy System** ([#46](https://github.com/SchoolyB/EZ/issues/46))
  - `private` keyword for module-internal items
  - Public by default

- **Multi-Dimensional Arrays** ([#47](https://github.com/SchoolyB/EZ/issues/47))
  - Enhanced support for `[[type]]` syntax

- **Comprehensive Documentation** ([#48](https://github.com/SchoolyB/EZ/issues/48))
  - Full docs for all built-ins and libraries

### Future Considerations
- Compilation (currently interpreted)
- Pointers and manual memory management
- Concurrency primitives
- First-class functions

---

## Contributing

EZ is under active development. If you'd like to contribute:

1. Check the [Issues](https://github.com/SchoolyB/EZ/issues) page for open tasks
2. Fork the repository
3. Create a feature branch: `feat/issue-number-description`
4. Make your changes
5. Submit a pull request to the `development` branch

### Bug Reports
Found a bug? Please [open an issue](https://github.com/SchoolyB/EZ/issues/new) with:
- EZ code that reproduces the bug
- Expected vs actual behavior
- Error messages (if any)

---

## License

[License information to be added]

---

## Acknowledgments

EZ is inspired by:
- **Go** - simplicity and explicit error handling
- **Odin** - procedural clarity and module system
- **C** - straightforward semantics
- **Rust** - excellent error messages

