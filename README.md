<p align="center">
  <img src="images/EZ_LOGO.jpeg" alt="EZ Logo" width="400">
</p>

# EZ Language

A simple interpreted programming language written in Go. The core philosophy is simplicity and lower barrier of entry.

**Note: Currently under heavy active development so things may change**
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

  EZ is a simple, statically-typed programming language designed to be easy to learn and
  understand. No object-oriented programming or no implicit behaviors, just clear & explicit code
  that does what it says.

  ### Design Philosophy
  - **Simplicity above all else** - easier than Python, clearer than Go
  - **Explicit over implicit** - no magic, every feature justifies its existence
  - **Procedural/functional only** - no OOP, classes, or inheritance
  - **Comprehensive standard library** - batteries included, no need for helper libraries
  - **Clear error messages** - Rust-inspired formatting with helpful context
  - **Easy to learn, ready to use** - gentle learning curve with real-world utility

  ### Type System
  - **Static typing** - types are checked before execution
  - **Strong typing** - no implicit type coercion
  - **Explicit declarations** - no type inference, types are always visible

---

## Quick Start

### Installation

**Option 1: Install Globally (Recommended)**
```bash
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Install EZ to your system (requires sudo password)
make install

# Run from anywhere
ez examples/hello.ez
ez help
```

**Option 2: Build Locally**
```bash
# Clone the repository
git clone https://github.com/SchoolyB/EZ.git
cd EZ

# Build the binary
make build

# Run with local binary
./ez run examples/hello.ez
```

**Uninstall**
```bash
sudo make uninstall
```

**Requirements**
- Go 1.23.1 or higher

  # Hello World
  ## Here's what's in examples/hello.ez:
  ```ez
  import @std

  do main() {
    std.println("Hello, World!")
  }
  ```

### Usage

Once installed, you can use EZ from anywhere:

```bash
ez your_program.ez       # Direct execution
ez run your_program.ez   # Explicit run command
ez build                 # Build project in current directory
ez build ./myproject     # Build project in specified directory
ez build file.ez         # Check syntax/types for a single file
ez repl                  # Start interactive REPL
ez help                  # Show help
ez version               # Show version
```

### Building Projects

The `ez build` command type-checks your entire project without running it:

```bash
# Build project in current directory (requires main.ez)
ez build

# Build project in a specific directory
ez build ./myproject

# Check a single file only
ez build utils.ez
```

Project builds start from `main.ez` and automatically discover and check all imported modules. This is useful for catching type errors across your entire codebase before running.

### REPL Mode

EZ includes an interactive REPL (Read-Eval-Print-Loop) for experimenting and learning:

```bash
$ ez repl
EZ Language REPL v0.1.0
Type 'help' for commands, 'exit' or 'quit' to exit

>> temp x int = 5
>> x + 10
15
>> import @std
>> using std
>> println("Hello, REPL!")
Hello, REPL!
>> exit
Goodbye!
```

**REPL Commands:**
- `exit`, `quit` - Exit the REPL
- `clear` - Reset environment (clear all variables/functions)
- `help` - Show help message

**Features:**
- Variables and functions persist across lines
- Expressions are automatically printed
- Multi-line input for functions and blocks
- Full error reporting with helpful messages

### Basic Example

```ez
import @std
import @math

do main() {
    using std

    // Variables
    temp name string = "EZ"
    const version float = 0.1

    println("Welcome to ${name} version: ${version}!")

    // Arrays
    temp numbers [int] = {1, 2, 3, 4, 5}
    println("Numbers:", numbers)

    
    //Functions
    std.println("Hello, ${get_user_name()}")
    
}
//Function that gets user input and returns its value
do get_user_name() -> string{
  temp username = input()
  return username
}


```

---

## Current Features

### ✅ Implemented & Working

#### Core Language Features
- **Variables & Constants**
  - `temp` for mutable variables
  - `const` for immutable constants
  - Default values for uninitialized `temp` variables
  - Variable shadowing in nested scopes

- **Data Types**
  - Primitives: `int`, `float`, `string`, `char`, `bool`
  - Sized integers: `i8`, `i16`, `i32`, `i64`, `i128`, `i256` (signed)
  - Sized integers: `u8`, `u16`, `u32`, `u64`, `u128`, `u256`, `uint` (unsigned)
  - Numeric separators: underscores for readability (`1_000_000`)
  - Arrays: dynamic `[type]` and fixed-size `[type, size]`
  - Maps: key-value pairs with `map[keyType:valueType]` syntax
  - Structs: user-defined types with fields
  - Enums: integer, float, and string enums with attributes

- **Operators**
  - Arithmetic: `+`, `-`, `*`, `/`, `%`
  - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
  - Logical: `&&`, `||`, `!`
  - Assignment: `=`, `+=`, `-=`, `*=`, `/=`, `%=`
  - Increment/Decrement: `++`, `--`
  - Membership: `in`, `!in` or `not_in`

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
  - Multiple inline imports: `import @std, @arrays`
  - Module aliasing: `import alias@module`
  - Mixed syntax support: `import @std, arr@arrays`
  - `import & use @module` for combined import and using
  - `using` keyword for namespace convenience
  - Standard library: `@std`, `@arrays`, `@maps`, `@math`, `@time`

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
  - `arrays.append()` and `arrays.pop()` functions
  - Array iteration with `for_each`

- **Built-in Functions**
  - `len(collection)` - get length of arrays/strings
  - `typeof(value)` - get type as string
  - `println(...)` - print with newline (std module)
  - `print(...)` - print without newline (std module)

- **REPL (Interactive Mode)**
  - Interactive Read-Eval-Print-Loop with `ez repl`
  - Persistent variables and functions across lines
  - Auto-print expressions
  - Multi-line input for functions and blocks
  - Commands: `help`, `exit`/`quit`, `clear`

- **Error System**
  - Color-coded error messages
  - Error codes (E1xxx, E2xxx, E3xxx)
  - Line and column information
  - Helpful error context
  - `@suppress(error_code)` attribute for warnings

#### Partially Implemented
- **Arrays** - Multi-dimensional arrays partially supported (issue #5)
- **Standard Library** - Currently Implemented: `@std`, `@arrays`, `@maps`, `@math`, `@time`

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

// Sized integers (signed)
temp small_signed i8 = -128
temp medium_signed i32 = -100000
temp large_signed i64 = -9223372036854775808
temp huge_signed i128 = 1000000000000

// Sized integers (unsigned) - cannot hold negative values
temp small_unsigned u8 = 255
temp medium_unsigned u32 = 4294967295
temp large_unsigned u64 = 18446744073709551615
temp huge_unsigned u128 = 1000000000000

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

// Import multiple modules on one line
import @std, @arrays, @math

// Import with alias
import arr@arrays

// Import multiple modules with aliases
import s@std, arr@arrays

// Mix both syntaxes
import @std, arr@arrays

// Import and use in one statement (no prefix needed)
import & use @std

do main() {
    // Use with prefix
    std.println("Hello")
    arr.append(numbers, 10)

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

arrays.append(numbers, 6)  // append element
temp last int = arrays.pop(numbers)

// Iterate over arrays
for_each num in numbers {
    println(num)
}

// Fixed-size arrays (immutable)
const days [string, 7] = {"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
```

### Maps

```ez
import @maps

// Map literal creation
temp users map[string:int] = {"alice": 25, "bob": 30, "charlie": 35}

// Map access
temp age int = users["alice"]

// Add or update entries
users["dave"] = 40
users["alice"] = 26

// Integer keys
temp scores map[int:string] = {1: "first", 2: "second", 3: "third"}

// Empty map
temp emptyMap map[string:string]

// Map operations via @maps stdlib
temp hasAlice bool = maps.has(users, "alice")
temp keys = maps.keys(users)
temp values = maps.values(users)
temp userCount int = len(users)
maps.delete(users, "dave")
temp usersCopy = maps.copy(users)
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

MIT License - Copyright (c) 2025-Present Marshall A Burns

See [LICENSE](LICENSE) for details.

---

## Acknowledgments

EZ is inspired by:
- **Go** - simplicity and explicit error handling
- **Odin** - procedural clarity and module system
- **C** - straightforward semantics
- **Rust** - excellent error messages

