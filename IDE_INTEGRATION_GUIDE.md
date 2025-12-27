# IDE Integration Guide - Native Go Debugger

This guide explains how to integrate the native Go debugger into the EZ IDE, replacing or complementing the existing REPL-based debugger.

## Overview

The native Go debugger offers significant advantages over the REPL-based approach:

- ✅ True breakpoint support (not just stepping)
- ✅ Works with multi-file programs and imports
- ✅ Better performance for complex programs
- ✅ Can step into functions and loops properly
- ✅ Full call stack inspection
- ✅ No file structure requirements

## Architecture

```
┌─────────────────────────────────────┐
│         EZ IDE (PyQt6)              │
│                                      │
│  ┌────────────────────────────────┐ │
│  │   GoDebugSession               │ │
│  │   (go_debug_session.py)        │ │
│  │                                │ │
│  │   Signals:                     │ │
│  │   - line_changed               │ │
│  │   - variable_updated           │ │
│  │   - output_received            │ │
│  │   - session_started/ended      │ │
│  └────────────────────────────────┘ │
│              ↓ QProcess             │
│     (JSON-RPC over stdin/stdout)    │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│    ez debugserver <file.ez>         │
│    (Go binary with JSON protocol)   │
│                                      │
│  ┌────────────────────────────────┐ │
│  │   JSONProtocol Handler         │ │
│  │   (json_protocol.go)           │ │
│  └────────────────────────────────┘ │
│              ↓                       │
│  ┌────────────────────────────────┐ │
│  │   Core Debugger                │ │
│  │   (debugger.go)                │ │
│  └────────────────────────────────┘ │
│              ↓                       │
│  ┌────────────────────────────────┐ │
│  │   EZ Interpreter               │ │
│  │   (with debug hooks)           │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
```

## Files Provided

### Go Debugger

1. **Debugger_Version/EZ/pkg/debugger/debugger.go**
   - Core debugger implementation
   - Breakpoint management
   - Call stack tracking
   - Event-driven architecture

2. **Debugger_Version/EZ/pkg/debugger/json_protocol.go**
   - JSON-RPC protocol handler
   - Command processing
   - Event emission
   - Line-delimited JSON communication

3. **Debugger_Version/EZ/cmd/ez/main.go**
   - Added `debugserver` command
   - Entry point: `ez debugserver <file.ez>`

### Python IDE Integration

4. **IDE/app/go_debug_session.py**
   - GoDebugSession class
   - Same signal interface as DebugSession
   - Drop-in replacement for REPL debugger

## Integration Steps

### Step 1: Copy the Go Debugger Session to IDE

The file is already created at:
```
IDE/app/go_debug_session.py
```

This provides the `GoDebugSession` class with the same interface as the existing `DebugSession`.

### Step 2: Update main_window.py to Support Both Debuggers

Modify `IDE/app/main_window.py` to allow switching between debugger backends.

#### Option A: Replace REPL Debugger Completely

Find the import in `main_window.py`:

```python
from app.debug_session import DebugSession
```

Change to:

```python
from app.go_debug_session import GoDebugSession as DebugSession
```

That's it! The IDE will now use the native debugger.

#### Option B: Make It Configurable (Recommended)

Add both imports:

```python
from app.debug_session import DebugSession as REPLDebugSession
from app.go_debug_session import GoDebugSession
```

In the `__init__` method where DebugSession is created, add selection logic:

```python
# In EZIDEMainWindow.__init__()

# Create debug session based on settings
debugger_type = self.settings_manager.settings.ez.debugger_type
if debugger_type == "native":
    self.debug_session = GoDebugSession(self.settings_manager, self)
else:
    self.debug_session = REPLDebugSession(self.settings_manager, self)
```

### Step 3: Update Settings (Optional - for configurable approach)

Modify `IDE/app/settings.py` to add debugger selection:

Find the `EZSettings` dataclass:

```python
@dataclass
class EZSettings:
    interpreter_path: str = ""
```

Add:

```python
@dataclass
class EZSettings:
    interpreter_path: str = ""
    debugger_type: str = "native"  # "native" or "repl"
```

### Step 4: Add UI for Debugger Selection (Optional)

In `main_window.py`, find the menu setup for Run menu and add:

```python
# In _setup_menus() method, in the Run menu section

# Add debugger type selector
debugger_menu = run_menu.addMenu("Debugger Type")

native_action = QAction("Native Go Debugger", self)
native_action.setCheckable(True)
native_action.setChecked(self.settings_manager.settings.ez.debugger_type == "native")
native_action.triggered.connect(lambda: self._set_debugger_type("native"))
debugger_menu.addAction(native_action)

repl_action = QAction("REPL Debugger", self)
repl_action.setCheckable(True)
repl_action.setChecked(self.settings_manager.settings.ez.debugger_type == "repl")
repl_action.triggered.connect(lambda: self._set_debugger_type("repl"))
debugger_menu.addAction(repl_action)

# Add the actions to an action group for mutual exclusivity
debugger_group = QActionGroup(self)
debugger_group.addAction(native_action)
debugger_group.addAction(repl_action)
```

And add the handler method:

```python
def _set_debugger_type(self, debugger_type: str):
    """Set the debugger type and recreate debug session"""
    self.settings_manager.settings.ez.debugger_type = debugger_type
    self.settings_manager.save()

    # Recreate debug session
    old_session = self.debug_session

    if debugger_type == "native":
        self.debug_session = GoDebugSession(self.settings_manager, self)
    else:
        self.debug_session = REPLDebugSession(self.settings_manager, self)

    # Reconnect signals
    self._connect_debug_signals()

    # Clean up old session
    if old_session:
        old_session.stop()

    self.statusBar().showMessage(f"Switched to {debugger_type} debugger", 3000)
```

### Step 5: Build the Go Debugger

From the command line:

```bash
cd Debugger_Version/EZ
go build -o ez ./cmd/ez
```

Install it where the IDE can find it (same location as the regular `ez` binary).

### Step 6: Test the Integration

1. Open the IDE
2. Open an EZ file
3. Press F5 to start debugging
4. The native debugger should now be in use
5. Test:
   - Step (F10) should work
   - Variable inspection should update in real-time
   - Source line highlighting should work
   - Breakpoints can be set/cleared

## Testing Checklist

- [ ] IDE starts without errors
- [ ] Can open and edit .ez files
- [ ] F5 starts debug mode
- [ ] Debug panel shows up
- [ ] First line is highlighted
- [ ] F10 steps to next line
- [ ] Variable tree updates with values
- [ ] Can step through loops
- [ ] Can step into functions
- [ ] Can set breakpoints (if implemented in UI)
- [ ] Program output appears in debug panel
- [ ] Can stop debugging (Shift+F5)
- [ ] Program completes successfully

## Troubleshooting

### "EZ interpreter not found"

- Configure the interpreter path in: Run → Select EZ Interpreter
- Make sure you're pointing to the `ez` binary you just built

### No response when stepping

- Check that `ez debugserver` command exists:
  ```bash
  ./ez debugserver --help
  ```
- Check IDE console/terminal for error messages

### Variables not updating

- The debugger sends variable updates automatically
- Check the debug panel output for JSON messages
- Verify the variable tree widget is connected to `variable_updated` signal

### Debug line not highlighting

- Verify the editor has the `set_debug_line()` method
- Check that `line_changed` signal is connected
- Line numbers should be 1-indexed (EZ uses 1-based line numbers)

### Process crashes or hangs

- Check stderr output from the debugserver
- Add logging to `_handle_event()` in GoDebugSession
- Verify JSON protocol messages are well-formed

## Advanced Features

### Adding Breakpoint UI

The debugger supports breakpoints, but the IDE may need UI for setting them.

In the editor (editor.py), you could add:

1. Gutter area for breakpoint markers
2. Click handler to toggle breakpoints
3. Visual indicators (red dots)
4. Call `debug_session.set_breakpoint(file, line)` when clicked

### Call Stack View

The debugger provides stack traces via `getStackTrace` command.

You could add a stack view widget that shows the call stack and allows frame selection.

### Watch Expressions

Future enhancement: Add a watch panel that evaluates expressions.

Use the `evaluate` command (when implemented in the Go debugger).

## Benefits Over REPL Debugger

| Feature | REPL Debugger | Native Debugger |
|---------|---------------|-----------------|
| Step through code | ✅ | ✅ |
| Step into functions | ❌ | ✅ |
| Step through loops | Limited | ✅ |
| Breakpoints | ❌ | ✅ |
| Multi-file support | ❌ | ✅ |
| Module imports | ❌ | ✅ |
| Call stack | ❌ | ✅ |
| Performance | Slow for large files | Fast |
| Variable inspection | ✅ | ✅ |
| Expression evaluation | ❌ | ✅ (planned) |

## Protocol Reference

See `DEBUGGER_PROTOCOL.md` for the complete JSON-RPC protocol specification.

### Key Commands

```json
{"type":"command","command":"stepInto","params":{}}
{"type":"command","command":"stepOver","params":{}}
{"type":"command","command":"continue","params":{}}
{"type":"command","command":"setBreakpoint","params":{"file":"test.ez","line":10}}
```

### Key Events

```json
{"type":"event","event":"stopped","data":{"reason":"step","location":{"file":"test.ez","line":10,"column":5}}}
{"type":"event","event":"variableUpdate","data":{"name":"x","value":"42","type":"int","frameIndex":0}}
{"type":"event","event":"output","data":{"category":"stdout","text":"Hello\n"}}
```

## Summary

The native Go debugger integration:

1. ✅ Provides production-quality debugging
2. ✅ Drop-in replacement for existing debugger
3. ✅ Uses same signal interface - minimal IDE changes
4. ✅ Supports all major debugging features
5. ✅ Extensible via JSON-RPC protocol
6. ✅ Ready for immediate use

The integration is designed to be as simple as possible - in the minimal case, it's just changing one import statement!

---

For questions or issues, see:
- `DEBUGGER_GUIDE.md` - User documentation
- `DEBUGGER_PROTOCOL.md` - Protocol specification
- `README.md` - Project overview
