# EZ Debugger JSON-RPC Protocol Specification

## Overview

This document specifies the JSON-RPC protocol used for communication between the EZ IDE (Python/PyQt6) and the EZ debugger (Go).

## Communication Model

- **Transport**: stdin/stdout (newline-delimited JSON messages)
- **Direction**: Bidirectional
- **Message Format**: One JSON object per line (line-delimited JSON)

## Message Types

### Commands (IDE → Debugger)

Commands are sent from the IDE to control the debugger.

### Events (Debugger → IDE)

Events are sent from the debugger to notify the IDE of state changes.

---

## Command Messages

All commands follow this format:

```json
{
  "type": "command",
  "command": "command_name",
  "params": {
    // command-specific parameters
  }
}
```

### 1. Initialize

Start a debug session.

```json
{
  "type": "command",
  "command": "initialize",
  "params": {
    "file": "/absolute/path/to/program.ez",
    "workingDir": "/working/directory"
  }
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "initialized",
  "data": {
    "file": "/absolute/path/to/program.ez"
  }
}
```

### 2. Start Execution

Begin or resume program execution.

```json
{
  "type": "command",
  "command": "start",
  "params": {}
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "started",
  "data": {}
}
```

### 3. Step Into

Execute the next statement, stepping into function calls.

```json
{
  "type": "command",
  "command": "stepInto",
  "params": {}
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "stopped",
  "data": {
    "reason": "step",
    "location": {
      "file": "/path/to/file.ez",
      "line": 42,
      "column": 10
    }
  }
}
```

### 4. Step Over

Execute the next statement, stepping over function calls.

```json
{
  "type": "command",
  "command": "stepOver",
  "params": {}
}
```

**Response Event:** Same as Step Into

### 5. Step Out

Continue execution until returning from the current function.

```json
{
  "type": "command",
  "command": "stepOut",
  "params": {}
}
```

**Response Event:** Same as Step Into

### 6. Continue

Resume execution until the next breakpoint or program completion.

```json
{
  "type": "command",
  "command": "continue",
  "params": {}
}
```

**Response Events:**
- `stopped` event when breakpoint hit
- `exited` event when program completes

### 7. Set Breakpoint

Set a breakpoint at a specific location.

```json
{
  "type": "command",
  "command": "setBreakpoint",
  "params": {
    "file": "/path/to/file.ez",
    "line": 25
  }
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "breakpointSet",
  "data": {
    "file": "/path/to/file.ez",
    "line": 25,
    "id": 1
  }
}
```

### 8. Clear Breakpoint

Remove a breakpoint.

```json
{
  "type": "command",
  "command": "clearBreakpoint",
  "params": {
    "file": "/path/to/file.ez",
    "line": 25
  }
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "breakpointCleared",
  "data": {
    "file": "/path/to/file.ez",
    "line": 25
  }
}
```

### 9. Get Variables

Request all variables in the current or specified frame.

```json
{
  "type": "command",
  "command": "getVariables",
  "params": {
    "frameIndex": 0
  }
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "variables",
  "data": {
    "frameIndex": 0,
    "variables": [
      {"name": "x", "value": "10", "type": "int"},
      {"name": "y", "value": "20", "type": "int"},
      {"name": "sum", "value": "30", "type": "int"}
    ]
  }
}
```

### 10. Get Stack Trace

Request the current call stack.

```json
{
  "type": "command",
  "command": "getStackTrace",
  "params": {}
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "stackTrace",
  "data": {
    "frames": [
      {
        "index": 0,
        "function": "fibonacci",
        "file": "/path/to/file.ez",
        "line": 12,
        "column": 5
      },
      {
        "index": 1,
        "function": "main",
        "file": "/path/to/file.ez",
        "line": 20,
        "column": 10
      }
    ]
  }
}
```

### 11. Evaluate Expression

Evaluate an expression in the current frame context.

```json
{
  "type": "command",
  "command": "evaluate",
  "params": {
    "expression": "x + y",
    "frameIndex": 0
  }
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "evaluateResult",
  "data": {
    "expression": "x + y",
    "result": "30",
    "type": "int"
  }
}
```

### 12. Terminate

Stop the debug session and terminate the program.

```json
{
  "type": "command",
  "command": "terminate",
  "params": {}
}
```

**Response Event:**
```json
{
  "type": "event",
  "event": "terminated",
  "data": {}
}
```

---

## Event Messages

All events follow this format:

```json
{
  "type": "event",
  "event": "event_name",
  "data": {
    // event-specific data
  }
}
```

### 1. Initialized

Debug session initialized successfully.

```json
{
  "type": "event",
  "event": "initialized",
  "data": {
    "file": "/path/to/program.ez"
  }
}
```

### 2. Started

Program execution has started or resumed.

```json
{
  "type": "event",
  "event": "started",
  "data": {}
}
```

### 3. Stopped

Execution has paused (breakpoint, step, etc.).

```json
{
  "type": "event",
  "event": "stopped",
  "data": {
    "reason": "step" | "breakpoint" | "entry",
    "location": {
      "file": "/path/to/file.ez",
      "line": 42,
      "column": 10
    },
    "breakpointId": 1  // only if reason is "breakpoint"
  }
}
```

### 4. Output

Program has produced output (stdout).

```json
{
  "type": "event",
  "event": "output",
  "data": {
    "category": "stdout" | "stderr",
    "text": "Hello, World!\n"
  }
}
```

### 5. Error

A runtime or debug error occurred.

```json
{
  "type": "event",
  "event": "error",
  "data": {
    "message": "error description",
    "location": {
      "file": "/path/to/file.ez",
      "line": 42,
      "column": 10
    }
  }
}
```

### 6. Variable Update

A variable has changed value (sent automatically during stepping).

```json
{
  "type": "event",
  "event": "variableUpdate",
  "data": {
    "name": "counter",
    "value": "42",
    "type": "int",
    "frameIndex": 0
  }
}
```

### 7. Variables

Response to getVariables command.

```json
{
  "type": "event",
  "event": "variables",
  "data": {
    "frameIndex": 0,
    "variables": [
      {"name": "x", "value": "10", "type": "int"}
    ]
  }
}
```

### 8. Stack Trace

Response to getStackTrace command.

```json
{
  "type": "event",
  "event": "stackTrace",
  "data": {
    "frames": [...]
  }
}
```

### 9. Breakpoint Set

Confirmation that a breakpoint was set.

```json
{
  "type": "event",
  "event": "breakpointSet",
  "data": {
    "file": "/path/to/file.ez",
    "line": 25,
    "id": 1
  }
}
```

### 10. Breakpoint Cleared

Confirmation that a breakpoint was cleared.

```json
{
  "type": "event",
  "event": "breakpointCleared",
  "data": {
    "file": "/path/to/file.ez",
    "line": 25
  }
}
```

### 11. Exited

Program execution has completed.

```json
{
  "type": "event",
  "event": "exited",
  "data": {
    "exitCode": 0
  }
}
```

### 12. Terminated

Debug session has been terminated.

```json
{
  "type": "event",
  "event": "terminated",
  "data": {}
}
```

---

## Typical Session Flow

### Simple Stepping Session

```
IDE → {"type":"command","command":"initialize","params":{"file":"/path/to/test.ez","workingDir":"/path"}}
DBG → {"type":"event","event":"initialized","data":{"file":"/path/to/test.ez"}}

IDE → {"type":"command","command":"start","params":{}}
DBG → {"type":"event","event":"started","data":{}}
DBG → {"type":"event","event":"stopped","data":{"reason":"entry","location":{"file":"/path/to/test.ez","line":5,"column":1}}}

IDE → {"type":"command","command":"getVariables","params":{"frameIndex":0}}
DBG → {"type":"event","event":"variables","data":{"frameIndex":0,"variables":[]}}

IDE → {"type":"command","command":"stepInto","params":{}}
DBG → {"type":"event","event":"stopped","data":{"reason":"step","location":{"file":"/path/to/test.ez","line":6,"column":1}}}
DBG → {"type":"event","event":"variableUpdate","data":{"name":"x","value":"10","type":"int","frameIndex":0}}

IDE → {"type":"command","command":"stepInto","params":{}}
DBG → {"type":"event","event":"stopped","data":{"reason":"step","location":{"file":"/path/to/test.ez","line":7,"column":1}}}
DBG → {"type":"event","event":"variableUpdate","data":{"name":"y","value":"20","type":"int","frameIndex":0}}

IDE → {"type":"command","command":"continue","params":{}}
DBG → {"type":"event","event":"output","data":{"category":"stdout","text":"Result: 30\n"}}
DBG → {"type":"event","event":"exited","data":{"exitCode":0}}
```

### Breakpoint Session

```
IDE → {"type":"command","command":"initialize","params":{"file":"/path/to/test.ez"}}
DBG → {"type":"event","event":"initialized","data":{...}}

IDE → {"type":"command","command":"setBreakpoint","params":{"file":"/path/to/test.ez","line":15}}
DBG → {"type":"event","event":"breakpointSet","data":{"file":"/path/to/test.ez","line":15,"id":1}}

IDE → {"type":"command","command":"start","params":{}}
DBG → {"type":"event","event":"started","data":{}}
DBG → {"type":"event","event":"output","data":{"category":"stdout","text":"Starting...\n"}}
DBG → {"type":"event","event":"stopped","data":{"reason":"breakpoint","location":{...},"breakpointId":1}}

IDE → {"type":"command","command":"getStackTrace","params":{}}
DBG → {"type":"event","event":"stackTrace","data":{"frames":[...]}}

IDE → {"type":"command","command":"continue","params":{}}
DBG → {"type":"event","event":"exited","data":{"exitCode":0}}
```

---

## Error Handling

### Command Errors

If a command cannot be executed, send an error event:

```json
{
  "type": "event",
  "event": "error",
  "data": {
    "message": "Cannot step: program not started",
    "command": "stepInto"
  }
}
```

### Runtime Errors

Runtime errors are reported as error events with location:

```json
{
  "type": "event",
  "event": "error",
  "data": {
    "message": "division by zero",
    "location": {
      "file": "/path/to/test.ez",
      "line": 42,
      "column": 15
    }
  }
}
```

---

## Implementation Notes

### Line-Delimited JSON

Each message is a complete JSON object on a single line, terminated by `\n`.

**Example:**
```
{"type":"command","command":"stepInto","params":{}}\n
{"type":"event","event":"stopped","data":{"reason":"step","location":{"file":"test.ez","line":10,"column":1}}}\n
```

### Concurrency

- Commands are processed sequentially
- Events may be sent at any time
- IDE should be prepared to receive events asynchronously

### Variable Values

Variable values are always strings in JSON, with type information separate:

```json
{"name": "x", "value": "42", "type": "int"}
{"name": "name", "value": "\"Alice\"", "type": "string"}
{"name": "items", "value": "[1, 2, 3]", "type": "array"}
```

---

## Future Extensions

Possible future additions:

- Conditional breakpoints
- Data breakpoints (watchpoints)
- Log points
- Hit count breakpoints
- Expression watches
- Variable modification
- Reverse debugging
- Multi-threaded debugging
