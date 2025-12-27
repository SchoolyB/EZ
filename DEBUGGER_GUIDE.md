# EZ Interactive Debugger - User Guide

## Overview

The EZ Interactive Debugger is a powerful command-line debugging tool that allows you to step through your EZ programs, inspect variables, set breakpoints, and examine the call stack in real-time.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Basic Concepts](#basic-concepts)
3. [Debug Commands Reference](#debug-commands-reference)
4. [Step-by-Step Tutorial](#step-by-step-tutorial)
5. [Advanced Features](#advanced-features)
6. [Tips and Best Practices](#tips-and-best-practices)
7. [Troubleshooting](#troubleshooting)

---

## Getting Started

### Building the Debugger

```bash
cd Debugger_Version/EZ
go build -o ez ./cmd/ez
```

### Starting a Debug Session

To debug an EZ program, use the `debug` command:

```bash
./ez debug your_program.ez
```

You will see the debug banner and the debugger will automatically pause at the first executable statement:

```
╔══════════════════════════════════════════════════════════╗
║        EZ Interactive Debugger - Debug Mode             ║
╚══════════════════════════════════════════════════════════╝

Debugging: your_program.ez
Type 'help' at the debug prompt for available commands
Execution will start automatically. Use 'step' to begin stepping.

→ your_program.ez:5
     3
     4   do main() {
     5 ▶     println("Starting program")
     6
     7       temp x int = 10

(debug)
```

### Your First Debug Session

At the `(debug)` prompt, you can enter commands:

- Type `step` (or just `s`) to execute the current line
- Type `help` (or `?`) to see all available commands
- Type `quit` (or `q`) to exit the debugger

---

## Basic Concepts

### Execution Control

The debugger gives you fine-grained control over program execution:

- **Stepping**: Execute code one statement at a time
- **Breakpoints**: Pause execution at specific locations
- **Continue**: Run until the next breakpoint or program end

### Call Stack

The debugger tracks the sequence of function calls (call stack), allowing you to:

- See which functions are currently executing
- Inspect variables at different stack levels
- Navigate up and down the call stack

### Variable Inspection

At any point during debugging, you can:

- View all variables in the current scope
- Print individual variable values
- Examine variables in different stack frames

---

## Debug Commands Reference

### Execution Control Commands

#### `s`, `step` - Step Into
Execute the next statement. If the statement is a function call, step into that function.

```
(debug) step
→ your_program.ez:8
     6
     7       temp x int = 10
     8 ▶     temp y int = calculateValue(x)
     9
    10   }

(debug) s         # Steps into calculateValue()
→ your_program.ez:15
    13
    14   do calculateValue(n int) -> int {
    15 ▶     temp result int = n * 2
    16       return result
    17   }
```

**Use when**: You want to examine what happens inside function calls.

---

#### `n`, `next` - Step Over
Execute the next statement. If the statement is a function call, execute the entire function without stepping through it.

```
(debug) next
→ your_program.ez:9
     7       temp x int = 10
     8       temp y int = calculateValue(x)  # Function executed completely
     9 ▶     println("Result: ${y}")
    10
    11   }
```

**Use when**: You want to skip over function calls and stay at the current level.

---

#### `o`, `out` - Step Out
Continue execution until the current function returns.

```
(debug) out
→ your_program.ez:9    # Back in main() after function returns
     7       temp x int = 10
     8       temp y int = calculateValue(x)
     9 ▶     println("Result: ${y}")
    10
    11   }
```

**Use when**: You've stepped into a function but want to quickly return to the caller.

---

#### `c`, `continue` - Continue Execution
Resume normal execution until the next breakpoint is hit or the program completes.

```
(debug) continue
Starting program
Result: 20
Debug test complete

✓ Program completed successfully
```

**Use when**: You want to run to the next breakpoint or program end.

---

### Breakpoint Commands

#### `b`, `break`, `breakpoint` - Manage Breakpoints

**Set a breakpoint:**
```
(debug) break my_program.ez:15
Breakpoint set at my_program.ez:15
```

**List all breakpoints:**
```
(debug) break
Current breakpoints:
  my_program.ez:15 (enabled)
  my_program.ez:23 (enabled)
```

**Breakpoint format**: `<filename>:<line_number>`

**Note**: Use the full or relative filename as it appears in the source code location display.

---

### Inspection Commands

#### `bt`, `backtrace`, `stack` - Show Call Stack
Display the current call stack with function names and locations.

```
(debug) bt

Call stack:
▶ #0 calculateValue at my_program.ez:15
  #1 main at my_program.ez:8
```

The `▶` marker indicates the current frame (most recent call).

---

#### `l`, `list` - Show Source Code
Display source code around the current execution point (2 lines before and after).

```
(debug) list

→ my_program.ez:8
     6
     7       temp x int = 10
     8 ▶     temp y int = calculateValue(x)
     9       println("Result: ${y}")
    10
```

---

#### `p`, `print <variable>` - Print Variable Value
Display the value of a specific variable.

```
(debug) print x
x = 10

(debug) print sum
sum = 30
```

**Note**: Currently prints variables from the current stack frame only.

---

#### `v`, `vars`, `variables [frame]` - Show All Variables
Display all variables in the specified stack frame (default: current frame).

```
(debug) vars

Variables (frame 0):
  x = 10
  y = 20
  sum = 30

(debug) vars 1    # Variables from caller's frame

Variables (frame 1):
  result = 100
  count = 5
```

**Frame numbering**:
- Frame 0 = current function (deepest in stack)
- Frame 1 = caller of current function
- Frame 2 = caller's caller, etc.

---

### Help and Control Commands

#### `h`, `help`, `?` - Show Help
Display a quick reference of all available commands.

```
(debug) help

Debug Commands:
  s, step         Step into (execute next statement)
  n, next         Step over (execute next statement, skip function calls)
  o, out          Step out (continue until return from current function)
  c, continue     Continue execution until next breakpoint
  ...
```

---

#### `q`, `quit`, `exit` - Exit Debugger
Terminate the debug session and exit the program immediately.

```
(debug) quit
Exiting debugger
```

---

### Special Features

#### Empty Input - Repeat Last Command
Pressing Enter without typing a command repeats your last command. This is especially useful for stepping:

```
(debug) step
→ program.ez:8
(debug)          # Press Enter
→ program.ez:9
(debug)          # Press Enter again
→ program.ez:10
```

---

## Step-by-Step Tutorial

Let's walk through debugging a simple Fibonacci program.

### Example Program: `fibonacci.ez`

```ez
import @std
using std

// Recursive Fibonacci
do fibonacci(n int) -> int {
    if n <= 0 {
        return 0
    }
    if n == 1 {
        return 1
    }
    return fibonacci(n - 1) + fibonacci(n - 2)
}

do main() {
    println("=== Fibonacci Sequence ===")

    // Calculate fib(5)
    temp result int = fibonacci(5)
    println("fib(5) = ${result}")
}
```

### Debug Session Walkthrough

#### 1. Start the debugger

```bash
./ez debug fibonacci.ez
```

Output:
```
╔══════════════════════════════════════════════════════════╗
║        EZ Interactive Debugger - Debug Mode             ║
╚══════════════════════════════════════════════════════════╝

Debugging: fibonacci.ez
Type 'help' at the debug prompt for available commands

→ fibonacci.ez:16
    14   do main() {
    15       println("=== Fibonacci Sequence ===")
    16 ▶
    17       // Calculate fib(5)
    18       temp result int = fibonacci(5)

(debug)
```

#### 2. Step through main()

```
(debug) step
=== Fibonacci Sequence ===

→ fibonacci.ez:19
    17       // Calculate fib(5)
    18       temp result int = fibonacci(5)
    19 ▶     println("fib(5) = ${result}")

(debug)
```

Wait - we skipped the fibonacci call! Let's restart and step INTO it.

#### 3. Restart and step into fibonacci()

Exit with `q` and restart. This time, when we reach the fibonacci call:

```
→ fibonacci.ez:19
    17
    18       temp result int = fibonacci(5)
    19 ▶     println("fib(5) = ${result}")

(debug) step      # Step into the println (oops, built-in)
```

Let's set a breakpoint inside fibonacci instead:

```
(debug) break fibonacci.ez:6
Breakpoint set at fibonacci.ez:6

(debug) continue
→ fibonacci.ez:6
     4
     5   do fibonacci(n int) -> int {
     6 ▶     if n <= 0 {
     7           return 0
     8       }

(debug) vars
Variables (frame 0):
  n = 5

(debug) step
→ fibonacci.ez:9
     7           return 0
     8       }
     9 ▶     if n == 1 {
    10           return 1
    11       }
```

#### 4. Examine the call stack

After stepping through more:

```
(debug) bt

Call stack:
▶ #0 fibonacci at fibonacci.ez:12
  #1 fibonacci at fibonacci.ez:12
  #2 fibonacci at fibonacci.ez:12
  #3 main at fibonacci.ez:19
```

This shows the recursive calls! We're 3 levels deep in fibonacci.

#### 5. Inspect variables at different levels

```
(debug) vars 0    # Current fibonacci call
Variables (frame 0):
  n = 3

(debug) vars 1    # Previous fibonacci call
Variables (frame 1):
  n = 4

(debug) vars 3    # main function
Variables (frame 3):
  result = <not yet assigned>
```

#### 6. Continue to completion

```
(debug) continue
fib(5) = 5

✓ Program completed successfully
```

---

## Advanced Features

### Setting Multiple Breakpoints

You can set breakpoints at different locations and use `continue` to jump between them:

```
(debug) break program.ez:10
Breakpoint set at program.ez:10

(debug) break program.ez:25
Breakpoint set at program.ez:25

(debug) break program.ez:42
Breakpoint set at program.ez:42

(debug) break    # List all
Current breakpoints:
  program.ez:10 (enabled)
  program.ez:25 (enabled)
  program.ez:42 (enabled)

(debug) continue    # Runs to line 10
(debug) continue    # Runs to line 25
(debug) continue    # Runs to line 42
```

### Debugging Loops

When debugging loops, use `next` to avoid stepping through every iteration:

```
→ program.ez:10
     8
     9       for i in range(0, 100) {
    10 ▶         sum = sum + i
    11       }
    12

(debug) next    # Execute this iteration, move to next
(debug) next    # Execute next iteration
(debug) next    # ...or set a breakpoint and continue
```

Or set a conditional breakpoint (manually check the variable):

```
(debug) vars
Variables (frame 0):
  i = 45
  sum = 1035

(debug) continue    # Skip ahead
```

### Examining Complex Data Structures

Use `vars` to see all variables, including arrays and maps:

```
(debug) vars

Variables (frame 0):
  numbers = [1, 2, 3, 4, 5]
  config = {name: "test", version: 1}
  result = 42
```

---

## Tips and Best Practices

### 1. Use Breakpoints for Long Programs

Instead of stepping through hundreds of lines, set breakpoints at key locations:

```
(debug) break program.ez:50    # Right before the interesting part
(debug) continue
```

### 2. Press Enter to Repeat

When stepping through code, just press Enter repeatedly instead of typing `step` each time.

### 3. Use `next` for Library Calls

When calling standard library functions you don't need to debug:

```
→ program.ez:15
    13
    14       temp data string = readFile("input.txt")    # Don't want to step into this
    15 ▶     temp lines array = split(data, "\n")

(debug) next    # Executes the entire readFile() call
```

### 4. Check the Call Stack When Lost

If you're deep in nested calls and forgot how you got there:

```
(debug) bt

Call stack:
▶ #0 processItem at utils.ez:45
  #1 processAll at utils.ez:30
  #2 handleRequest at server.ez:120
  #3 main at main.ez:15
```

### 5. Use `out` to Escape Deep Calls

Stepped into a function by mistake?

```
(debug) out    # Quickly return to the caller
```

### 6. Inspect Variables Before Continuing

Before using `continue`, check your variables to verify state:

```
(debug) vars
Variables (frame 0):
  counter = 50
  isValid = true

(debug) continue    # Now continue with confidence
```

---

## Troubleshooting

### Issue: "Variable 'x' not found"

**Cause**: The variable might not be in scope or hasn't been created yet.

**Solution**:
- Check if you're at the right location in the code
- Use `vars` to see all available variables
- Step forward if the variable is declared later

### Issue: Can't step into a function

**Cause**: You might be using `next` instead of `step`, or it's a built-in function.

**Solution**:
- Use `step` (not `next`) to enter functions
- Built-in functions (like `println`) can't be stepped into
- Set a breakpoint inside the function and use `continue`

### Issue: Lost track of where I am

**Solution**:
```
(debug) list     # Show source code
(debug) bt       # Show call stack
(debug) vars     # Show current variables
```

### Issue: Program runs without stopping

**Cause**: No breakpoints set and using `continue` instead of `step`.

**Solution**:
- Press Ctrl+C to interrupt (if supported)
- Restart the debug session
- Use `step` or set breakpoints before `continue`

### Issue: Can't see the source file

**Cause**: The debugger can't find the source file at the displayed path.

**Solution**:
- Make sure you're running the debugger from the correct directory
- Use absolute paths when starting the debugger
- Check that the file hasn't been moved

---

## Quick Reference Card

```
EXECUTION CONTROL          INSPECTION               BREAKPOINTS
------------------         ----------               -----------
s, step      Step into     v, vars     Variables    b <file>:<line>  Set
n, next      Step over     p <var>     Print var    b                List
o, out       Step out      bt, stack   Call stack
c, continue  Continue      l, list     Source code

HELP & EXIT
-----------
h, help, ?   Show help
q, quit      Exit debugger
<Enter>      Repeat last command
```

---

## Examples

### Example 1: Simple Variable Inspection

```
→ program.ez:10
     8       temp x int = 10
     9       temp y int = 20
    10 ▶     temp sum int = x + y

(debug) vars
Variables (frame 0):
  x = 10
  y = 20

(debug) step
→ program.ez:11

(debug) vars
Variables (frame 0):
  x = 10
  y = 20
  sum = 30
```

### Example 2: Debugging with Breakpoints

```
(debug) break program.ez:25
Breakpoint set at program.ez:25

(debug) break program.ez:50
Breakpoint set at program.ez:50

(debug) continue
⚡ Breakpoint hit at program.ez:25

→ program.ez:25
    23
    24       if isValid {
    25 ▶         processData(data)

(debug) vars
Variables (frame 0):
  isValid = true
  data = "test data"

(debug) continue
⚡ Breakpoint hit at program.ez:50
```

### Example 3: Navigating the Call Stack

```
(debug) bt

Call stack:
▶ #0 helper at utils.ez:15
  #1 process at main.ez:30
  #2 main at main.ez:10

(debug) vars 0
Variables (frame 0):
  value = 42

(debug) vars 1
Variables (frame 1):
  count = 5
  total = 210

(debug) vars 2
Variables (frame 2):
  filename = "input.txt"
```

---

## Conclusion

The EZ Interactive Debugger provides powerful debugging capabilities through an intuitive command-line interface. By mastering the commands in this guide, you'll be able to efficiently debug even complex EZ programs.

**Key Takeaways**:
- Use `step` to go line-by-line, `next` to skip over functions
- Set breakpoints to jump to important locations
- Use `vars` and `print` to inspect program state
- Check the call stack with `bt` when navigating nested calls
- Press Enter to repeat your last command

Happy debugging! 🐛🔍
