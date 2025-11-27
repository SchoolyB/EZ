# EZ Built-in Functions

This document describes all built-in functions available in the EZ programming language. Built-in functions are organized into three categories: global built-ins (available everywhere), the `std` module (for standard I/O operations), and the `arrays` module (for array manipulation).

## Table of Contents

- [Global Built-ins](#global-built-ins)
- [Standard Library (std module)](#standard-library-std-module)
- [Arrays Module](#arrays-module)
  - [Basic Operations](#basic-operations)
  - [Adding Elements](#adding-elements)
  - [Removing Elements](#removing-elements)
  - [Accessing Elements](#accessing-elements)
  - [Slicing](#slicing)
  - [Searching](#searching)
  - [Reordering](#reordering)
  - [Combining Arrays](#combining-arrays)
  - [Unique and Duplicates](#unique-and-duplicates)
  - [Mathematical Operations](#mathematical-operations)
  - [Array Creation](#array-creation)
  - [Copying](#copying)
  - [String Conversion](#string-conversion)
  - [Boolean Checks](#boolean-checks)

## Global Built-ins

Global built-in functions are available everywhere without needing to import any module.

### `len(value)`

Returns the length of a string or array.

**Parameters:**
- `value` - A string or array

**Returns:** Integer representing the number of characters in a string or elements in an array

**Example:**
```ez
const name string = "EZ"
const length int = len(name)  // 2

const numbers [int] = [1, 2, 3, 4, 5]
const count int = len(numbers)  // 5
```

### `typeof(value)`

Returns the type of a value as a string.

**Parameters:**
- `value` - Any value

**Returns:** String representing the type (e.g., "int", "float", "string", "bool", "array", etc.)

**Example:**
```ez
const x int = 42
const t string = typeof(x)  // "int"

const y float = 3.14
const t2 string = typeof(y)  // "float"
```

### `int(value)`

Converts a value to an integer.

**Parameters:**
- `value` - An integer, float, string, or char

**Returns:** Integer value

**Conversion rules:**
- Integer: returns as-is
- Float: truncates to integer (no rounding)
- String: parses the string as a base-10 integer
- Char: returns the ASCII/Unicode value

**Example:**
```ez
const a int = int(3.14)      // 3
const b int = int("42")      // 42
const c int = int('A')       // 65
```

### `float(value)`

Converts a value to a floating-point number.

**Parameters:**
- `value` - An integer, float, or string

**Returns:** Float value

**Conversion rules:**
- Float: returns as-is
- Integer: converts to equivalent float
- String: parses the string as a floating-point number

**Example:**
```ez
const a float = float(42)      // 42.0
const b float = float("3.14")  // 3.14
```

### `string(value)`

Converts any value to its string representation.

**Parameters:**
- `value` - Any value

**Returns:** String representation of the value

**Example:**
```ez
const a string = string(42)      // "42"
const b string = string(3.14)    // "3.14"
const c string = string(true)    // "true"
```

### `input()`

Reads a line of text from standard input (stdin).

**Parameters:** None

**Returns:** String containing the input line (newline character is removed)

**Example:**
```ez
using std

std.println("Enter your name:")
const name string = input()
std.println("Hello, ${name}!")
```

## Standard Library (std module)

The `std` module provides standard input/output functions. To use these functions, you must first import the module:

```ez
import @std
using std
```

### `std.println(...)`

Prints values to standard output followed by a newline.

**Parameters:**
- `...` - Any number of values of any type

**Returns:** nil

**Behavior:**
- Multiple arguments are separated by spaces
- Always adds a newline at the end

**Example:**
```ez
using std

std.println("Hello, World!")
std.println("x =", 42, "y =", 3.14)  // x = 42 y = 3.14
```

### `std.print(...)`

Prints values to standard output without a newline.

**Parameters:**
- `...` - Any number of values of any type

**Returns:** nil

**Behavior:**
- Multiple arguments are separated by spaces
- Does NOT add a newline at the end

**Example:**
```ez
using std

std.print("Loading")
std.print(".")
std.print(".")
std.print(".")
std.println(" Done!")  // Output: Loading... Done!
```

### `std.read_int()`

Reads an integer from standard input. This function returns two values: the integer value and an error object.

**Parameters:** None

**Returns:** Two values:
1. Integer - The parsed value (0 if error occurred)
2. Error or nil - Error struct if parsing failed, nil on success

**Error codes:**
- `code: 1` - Failed to read input
- `code: 2` - Invalid integer input (parsing failed)

**Example:**
```ez
using std

std.println("Enter a number:")
const value, err = std.read_int()

if err != nil {
    std.println("Error:", err.message)
} else {
    std.println("You entered:", value)
}
```

## Arrays Module

The `arrays` module provides comprehensive array manipulation functions. To use these functions, import the module:

```ez
import @arrays
using arrays
```

### Basic Operations

#### `arrays.len(array)`

Returns the number of elements in an array.

**Parameters:**
- `array` - An array

**Returns:** Integer representing the number of elements

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const count int = arrays.len(nums)  // 5
```

#### `arrays.is_empty(array)`

Checks if an array is empty.

**Parameters:**
- `array` - An array

**Returns:** Boolean - true if array has no elements, false otherwise

**Example:**
```ez
using arrays

const empty [int] = []
const result bool = arrays.is_empty(empty)  // true

const nums [int] = [1, 2, 3]
const result2 bool = arrays.is_empty(nums)  // false
```

### Adding Elements

#### `arrays.push(array, value, ...)`

Adds one or more elements to the end of an array. Modifies the array in-place.

**Parameters:**
- `array` - An array
- `value` - One or more values to add

**Returns:** nil

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3]
arrays.push(nums, 4)       // nums is now [1, 2, 3, 4]
arrays.push(nums, 5, 6)    // nums is now [1, 2, 3, 4, 5, 6]
```

#### `arrays.unshift(array, value, ...)`

Adds one or more elements to the beginning of an array. Returns a new array.

**Parameters:**
- `array` - An array
- `value` - One or more values to add at the beginning

**Returns:** New array with values prepended

**Example:**
```ez
using arrays

const nums [int] = [3, 4, 5]
const result [int] = arrays.unshift(nums, 1, 2)  // [1, 2, 3, 4, 5]
```

#### `arrays.insert(array, index, value)`

Inserts a value at a specific index in the array. Returns a new array.

**Parameters:**
- `array` - An array
- `index` - Integer index where to insert (0-based)
- `value` - Value to insert

**Returns:** New array with value inserted

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 4, 5]
const result [int] = arrays.insert(nums, 2, 3)  // [1, 2, 3, 4, 5]
```

### Removing Elements

#### `arrays.pop(array)`

Removes and returns the last element from an array. Modifies the array in-place.

**Parameters:**
- `array` - An array

**Returns:** The removed element

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const last int = arrays.pop(nums)  // 5, nums is now [1, 2, 3, 4]
```

#### `arrays.shift(array)`

Returns the first element from an array.

**Parameters:**
- `array` - An array

**Returns:** The first element

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const first int = arrays.shift(nums)  // 1
```

#### `arrays.remove_at(array, index)`

Removes the element at a specific index. Returns a new array.

**Parameters:**
- `array` - An array
- `index` - Integer index to remove (0-based)

**Returns:** New array with element removed

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const result [int] = arrays.remove_at(nums, 2)  // [1, 2, 4, 5]
```

#### `arrays.remove(array, value)`

Removes the first occurrence of a value from an array. Returns a new array.

**Parameters:**
- `array` - An array
- `value` - Value to remove

**Returns:** New array with first occurrence removed (or original if not found)

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4]
const result [int] = arrays.remove(nums, 2)  // [1, 3, 2, 4]
```

#### `arrays.remove_all(array, value)`

Removes all occurrences of a value from an array. Returns a new array.

**Parameters:**
- `array` - An array
- `value` - Value to remove

**Returns:** New array with all occurrences removed

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4, 2]
const result [int] = arrays.remove_all(nums, 2)  // [1, 3, 4]
```

#### `arrays.clear(array)`

Returns an empty array of the same type.

**Parameters:**
- `array` - An array

**Returns:** New empty array

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const result [int] = arrays.clear(nums)  // []
```

### Accessing Elements

#### `arrays.get(array, index)`

Gets the element at a specific index.

**Parameters:**
- `array` - An array
- `index` - Integer index (0-based)

**Returns:** The element at the index

**Example:**
```ez
using arrays

const nums [int] = [10, 20, 30, 40]
const value int = arrays.get(nums, 2)  // 30
```

#### `arrays.first(array)`

Gets the first element of an array.

**Parameters:**
- `array` - An array

**Returns:** The first element, or nil if array is empty

**Example:**
```ez
using arrays

const nums [int] = [10, 20, 30]
const value int = arrays.first(nums)  // 10
```

#### `arrays.last(array)`

Gets the last element of an array.

**Parameters:**
- `array` - An array

**Returns:** The last element, or nil if array is empty

**Example:**
```ez
using arrays

const nums [int] = [10, 20, 30]
const value int = arrays.last(nums)  // 30
```

#### `arrays.set(array, index, value)`

Sets the element at a specific index. Returns a new array.

**Parameters:**
- `array` - An array
- `index` - Integer index (0-based)
- `value` - New value to set

**Returns:** New array with value updated

**Example:**
```ez
using arrays

const nums [int] = [10, 20, 30, 40]
const result [int] = arrays.set(nums, 2, 99)  // [10, 20, 99, 40]
```

### Slicing

#### `arrays.slice(array, start, end?)`

Extracts a portion of an array from start index up to (but not including) end index. Returns a new array.

**Parameters:**
- `array` - An array
- `start` - Integer starting index (can be negative to count from end)
- `end` - Optional integer ending index (can be negative to count from end). If omitted, slices to end of array

**Returns:** New array containing the slice

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const slice1 [int] = arrays.slice(nums, 1, 4)  // [2, 3, 4]
const slice2 [int] = arrays.slice(nums, 2)     // [3, 4, 5]
const slice3 [int] = arrays.slice(nums, -3)    // [3, 4, 5]
```

#### `arrays.take(array, n)`

Takes the first n elements from an array. Returns a new array.

**Parameters:**
- `array` - An array
- `n` - Integer number of elements to take

**Returns:** New array with first n elements

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const result [int] = arrays.take(nums, 3)  // [1, 2, 3]
```

#### `arrays.drop(array, n)`

Drops the first n elements from an array. Returns a new array.

**Parameters:**
- `array` - An array
- `n` - Integer number of elements to drop

**Returns:** New array without first n elements

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const result [int] = arrays.drop(nums, 2)  // [3, 4, 5]
```

### Searching

#### `arrays.contains(array, value)`

Checks if an array contains a specific value.

**Parameters:**
- `array` - An array
- `value` - Value to search for

**Returns:** Boolean - true if found, false otherwise

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const found bool = arrays.contains(nums, 3)  // true
const missing bool = arrays.contains(nums, 10)  // false
```

#### `arrays.index_of(array, value)`

Finds the index of the first occurrence of a value.

**Parameters:**
- `array` - An array
- `value` - Value to search for

**Returns:** Integer index if found, -1 if not found

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4]
const index int = arrays.index_of(nums, 2)  // 1
const missing int = arrays.index_of(nums, 10)  // -1
```

#### `arrays.last_index_of(array, value)`

Finds the index of the last occurrence of a value.

**Parameters:**
- `array` - An array
- `value` - Value to search for

**Returns:** Integer index if found, -1 if not found

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4]
const index int = arrays.last_index_of(nums, 2)  // 3
```

#### `arrays.count(array, value)`

Counts how many times a value appears in an array.

**Parameters:**
- `array` - An array
- `value` - Value to count

**Returns:** Integer count of occurrences

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4, 2]
const count int = arrays.count(nums, 2)  // 3
```

### Reordering

#### `arrays.reverse(array)`

Reverses the order of elements in an array. Returns a new array.

**Parameters:**
- `array` - An array

**Returns:** New array with elements in reverse order

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const reversed [int] = arrays.reverse(nums)  // [5, 4, 3, 2, 1]
```

#### `arrays.sort(array)`

Sorts an array in ascending order. Returns a new array.

**Parameters:**
- `array` - An array (must contain comparable elements)

**Returns:** New array sorted in ascending order

**Example:**
```ez
using arrays

const nums [int] = [3, 1, 4, 1, 5, 9, 2, 6]
const sorted [int] = arrays.sort(nums)  // [1, 1, 2, 3, 4, 5, 6, 9]

const words [string] = ["dog", "cat", "apple", "banana"]
const sorted_words [string] = arrays.sort(words)  // ["apple", "banana", "cat", "dog"]
```

#### `arrays.sort_desc(array)`

Sorts an array in descending order. Returns a new array.

**Parameters:**
- `array` - An array (must contain comparable elements)

**Returns:** New array sorted in descending order

**Example:**
```ez
using arrays

const nums [int] = [3, 1, 4, 1, 5, 9, 2, 6]
const sorted [int] = arrays.sort_desc(nums)  // [9, 6, 5, 4, 3, 2, 1, 1]
```

#### `arrays.shuffle(array)`

Randomly shuffles the elements of an array. Returns a new array.

**Parameters:**
- `array` - An array

**Returns:** New array with elements in random order

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const shuffled [int] = arrays.shuffle(nums)  // Random order, e.g., [3, 1, 5, 2, 4]
```

### Combining Arrays

#### `arrays.concat(array1, array2, ...)`

Concatenates two or more arrays. Returns a new array.

**Parameters:**
- `array1`, `array2`, ... - Arrays to concatenate

**Returns:** New array containing all elements from all arrays

**Example:**
```ez
using arrays

const a [int] = [1, 2, 3]
const b [int] = [4, 5]
const c [int] = [6, 7, 8]
const result [int] = arrays.concat(a, b, c)  // [1, 2, 3, 4, 5, 6, 7, 8]
```

#### `arrays.zip(array1, array2)`

Combines two arrays element-wise into pairs. Returns a new array of arrays.

**Parameters:**
- `array1` - First array
- `array2` - Second array

**Returns:** New array where each element is a 2-element array containing corresponding elements from both input arrays. Length is determined by the shorter array.

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3]
const letters [string] = ["a", "b", "c"]
const zipped = arrays.zip(nums, letters)  // [[1, "a"], [2, "b"], [3, "c"]]
```

#### `arrays.flatten(array)`

Flattens a nested array by one level. Returns a new array.

**Parameters:**
- `array` - An array containing nested arrays

**Returns:** New array with nested arrays flattened by one level

**Example:**
```ez
using arrays

const nested = [[1, 2], [3, 4], [5, 6]]
const flat = arrays.flatten(nested)  // [1, 2, 3, 4, 5, 6]
```

### Unique and Duplicates

#### `arrays.unique(array)`

Removes duplicate elements from an array, keeping only the first occurrence of each value. Returns a new array.

**Parameters:**
- `array` - An array

**Returns:** New array with duplicates removed

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4, 1, 5]
const unique [int] = arrays.unique(nums)  // [1, 2, 3, 4, 5]
```

#### `arrays.duplicates(array)`

Returns all duplicate values from an array (one occurrence of each duplicate). Returns a new array.

**Parameters:**
- `array` - An array

**Returns:** New array containing values that appear more than once (one occurrence each)

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 2, 4, 1, 5, 3]
const dups [int] = arrays.duplicates(nums)  // [1, 2, 3]
```

### Mathematical Operations

These functions work with arrays of numeric values (integers and floats).

#### `arrays.sum(array)`

Calculates the sum of all elements in a numeric array.

**Parameters:**
- `array` - An array of numbers (int or float)

**Returns:** Sum as integer if all elements are integers, float if any element is a float

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const total int = arrays.sum(nums)  // 15

const values [float] = [1.5, 2.5, 3.0]
const total_float float = arrays.sum(values)  // 7.0
```

#### `arrays.product(array)`

Calculates the product of all elements in a numeric array.

**Parameters:**
- `array` - An array of numbers (int or float)

**Returns:** Product as integer if all elements are integers, float if any element is a float. Returns 1 for empty array.

**Example:**
```ez
using arrays

const nums [int] = [2, 3, 4]
const result int = arrays.product(nums)  // 24
```

#### `arrays.min(array)`

Finds the minimum value in an array.

**Parameters:**
- `array` - An array of comparable elements

**Returns:** The minimum value

**Example:**
```ez
using arrays

const nums [int] = [5, 2, 8, 1, 9]
const minimum int = arrays.min(nums)  // 1
```

#### `arrays.max(array)`

Finds the maximum value in an array.

**Parameters:**
- `array` - An array of comparable elements

**Returns:** The maximum value

**Example:**
```ez
using arrays

const nums [int] = [5, 2, 8, 1, 9]
const maximum int = arrays.max(nums)  // 9
```

#### `arrays.avg(array)`

Calculates the average (mean) of all elements in a numeric array.

**Parameters:**
- `array` - An array of numbers (int or float)

**Returns:** Average as a float

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const average float = arrays.avg(nums)  // 3.0
```

### Array Creation

#### `arrays.range(end)` or `arrays.range(start, end)` or `arrays.range(start, end, step)`

Creates an array of integers in a range.

**Parameters:**
- `end` - End value (exclusive). If only this parameter is provided, start defaults to 0 and step to 1.
- `start` - Optional starting value (inclusive)
- `end` - Ending value (exclusive)
- `step` - Optional step value (default 1)

**Returns:** New array of integers

**Example:**
```ez
using arrays

const a [int] = arrays.range(5)           // [0, 1, 2, 3, 4]
const b [int] = arrays.range(2, 7)        // [2, 3, 4, 5, 6]
const c [int] = arrays.range(0, 10, 2)    // [0, 2, 4, 6, 8]
const d [int] = arrays.range(10, 0, -2)   // [10, 8, 6, 4, 2]
```

#### `arrays.repeat(value, n)`

Creates an array by repeating a value n times.

**Parameters:**
- `value` - Value to repeat
- `n` - Integer number of times to repeat

**Returns:** New array with value repeated n times

**Example:**
```ez
using arrays

const zeros [int] = arrays.repeat(0, 5)        // [0, 0, 0, 0, 0]
const words [string] = arrays.repeat("hi", 3)  // ["hi", "hi", "hi"]
```

#### `arrays.fill(array, value)`

Creates a new array of the same length filled with a specific value.

**Parameters:**
- `array` - An array (used for determining length)
- `value` - Value to fill with

**Returns:** New array of same length filled with value

**Example:**
```ez
using arrays

const nums [int] = [1, 2, 3, 4, 5]
const filled [int] = arrays.fill(nums, 0)  // [0, 0, 0, 0, 0]
```

### Copying

#### `arrays.copy(array)`

Creates a shallow copy of an array.

**Parameters:**
- `array` - An array

**Returns:** New array with same elements

**Example:**
```ez
using arrays

const original [int] = [1, 2, 3]
const copy [int] = arrays.copy(original)
```

### String Conversion

#### `arrays.join(array, separator)`

Joins all elements of an array into a string, separated by a separator.

**Parameters:**
- `array` - An array
- `separator` - String to use as separator

**Returns:** String with all elements joined

**Example:**
```ez
using arrays

const words [string] = ["Hello", "World", "!"]
const sentence string = arrays.join(words, " ")  // "Hello World !"

const nums [int] = [1, 2, 3, 4, 5]
const csv string = arrays.join(nums, ",")  // "1,2,3,4,5"
```

### Boolean Checks

#### `arrays.all_equal(array)`

Checks if all elements in an array are equal.

**Parameters:**
- `array` - An array

**Returns:** Boolean - true if all elements are equal (or array has 0-1 elements), false otherwise

**Example:**
```ez
using arrays

const same [int] = [5, 5, 5, 5]
const result1 bool = arrays.all_equal(same)  // true

const different [int] = [1, 2, 3]
const result2 bool = arrays.all_equal(different)  // false
```
