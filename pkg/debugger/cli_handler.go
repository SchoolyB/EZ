package debugger

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/object"
)

// CLIHandler provides command-line interactive debugging
type CLIHandler struct {
	reader  *bufio.Reader
	writer  io.Writer
	debugger *Debugger
	lastCommand string // For repeat on empty input
}

// NewCLIHandler creates a new CLI debug event handler
func NewCLIHandler(d *Debugger, reader io.Reader, writer io.Writer) *CLIHandler {
	if reader == nil {
		reader = os.Stdin
	}
	if writer == nil {
		writer = os.Stdout
	}

	return &CLIHandler{
		reader:  bufio.NewReader(reader),
		writer:  writer,
		debugger: d,
		lastCommand: "step",
	}
}

// OnBreakpoint is called when a breakpoint is hit
func (h *CLIHandler) OnBreakpoint(d *Debugger, bp *Breakpoint) {
	h.printf("\n⚡ Breakpoint hit at %s:%d\n", bp.File, bp.Line)
	h.showCurrentLocation(d)
	h.commandLoop(d)
}

// OnStep is called when stepping through code
func (h *CLIHandler) OnStep(d *Debugger, node ast.Node, env *object.Environment) {
	h.showCurrentLocation(d)
	h.commandLoop(d)
}

// OnFunctionCall is called when entering a function
func (h *CLIHandler) OnFunctionCall(d *Debugger, frame *CallFrame) {
	// Silent unless in verbose mode
}

// OnFunctionReturn is called when returning from a function
func (h *CLIHandler) OnFunctionReturn(d *Debugger, frame *CallFrame, result object.Object) {
	// Silent unless in verbose mode
}

// OnError is called when an error occurs
func (h *CLIHandler) OnError(d *Debugger, err error) {
	h.printf("❌ Error: %v\n", err)
}

// showCurrentLocation displays the current execution location
func (h *CLIHandler) showCurrentLocation(d *Debugger) {
	loc := d.GetCurrentLocation()
	if loc == nil {
		h.printf("→ <unknown location>\n")
		return
	}

	// Show file:line
	h.printf("\n→ %s:%d\n", loc.File, loc.Line)

	// Try to show the source line if we can read the file
	h.showSourceLine(loc.File, loc.Line)
}

// showSourceLine displays a line from the source file
func (h *CLIHandler) showSourceLine(file string, line int) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	currentLine := 0

	// Show context: 2 lines before, current line, 2 lines after
	lines := make([]string, 0, 5)
	lineNumbers := make([]int, 0, 5)

	for scanner.Scan() {
		currentLine++
		if currentLine >= line-2 && currentLine <= line+2 {
			lines = append(lines, scanner.Text())
			lineNumbers = append(lineNumbers, currentLine)
		}
		if currentLine > line+2 {
			break
		}
	}

	// Display the lines
	for i, text := range lines {
		lineNum := lineNumbers[i]
		if lineNum == line {
			h.printf("  %4d ▶ %s\n", lineNum, text)
		} else {
			h.printf("  %4d   %s\n", lineNum, text)
		}
	}
}

// commandLoop processes debug commands until continue/step is issued
func (h *CLIHandler) commandLoop(d *Debugger) {
	for {
		h.printf("(debug) ")

		input, err := h.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				h.printf("\nExiting debugger\n")
				os.Exit(0)
			}
			continue
		}

		input = strings.TrimSpace(input)

		// Empty input repeats last command
		if input == "" {
			input = h.lastCommand
		} else {
			h.lastCommand = input
		}

		// Parse command
		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		args := parts[1:]

		if h.handleCommand(d, cmd, args) {
			return // Command requested continue/step
		}
	}
}

// handleCommand processes a debug command and returns true if execution should continue
func (h *CLIHandler) handleCommand(d *Debugger, cmd string, args []string) bool {
	switch cmd {
	case "c", "continue":
		d.Continue()
		return true

	case "s", "step":
		d.StepInto()
		return true

	case "n", "next":
		d.StepOver()
		return true

	case "o", "out":
		d.StepOut()
		return true

	case "b", "break", "breakpoint":
		h.handleBreakpoint(d, args)

	case "bt", "backtrace", "stack":
		h.showCallStack(d)

	case "l", "list":
		h.showSourceContext(d)

	case "p", "print":
		h.handlePrint(d, args)

	case "v", "vars", "variables":
		h.showVariables(d, args)

	case "h", "help", "?":
		h.showHelp()

	case "q", "quit", "exit":
		h.printf("Exiting debugger\n")
		os.Exit(0)

	default:
		h.printf("Unknown command: %s (type 'help' for available commands)\n", cmd)
	}

	return false
}

// handleBreakpoint sets or clears breakpoints
func (h *CLIHandler) handleBreakpoint(d *Debugger, args []string) {
	if len(args) == 0 {
		h.printf("Current breakpoints:\n")
		bps := d.GetBreakpoints()
		if len(bps) == 0 {
			h.printf("  (none)\n")
		} else {
			for _, bp := range bps {
				status := "enabled"
				if !bp.Enabled {
					status = "disabled"
				}
				h.printf("  %s:%d (%s)\n", bp.File, bp.Line, status)
			}
		}
		return
	}

	// Parse file:line format
	spec := args[0]
	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		h.printf("Usage: break <file>:<line>\n")
		return
	}

	file := parts[0]
	line, err := strconv.Atoi(parts[1])
	if err != nil {
		h.printf("Invalid line number: %s\n", parts[1])
		return
	}

	d.SetBreakpoint(file, line)
	h.printf("Breakpoint set at %s:%d\n", file, line)
}

// showCallStack displays the call stack
func (h *CLIHandler) showCallStack(d *Debugger) {
	stack := d.GetCallStack()
	if len(stack) == 0 {
		h.printf("Call stack is empty\n")
		return
	}

	h.printf("\nCall stack:\n")
	for i := len(stack) - 1; i >= 0; i-- {
		frame := stack[i]
		marker := "  "
		if i == len(stack)-1 {
			marker = "▶ "
		}
		h.printf("%s#%d %s at %s\n", marker, len(stack)-1-i, frame.FunctionName, FormatLocation(frame.Location))
	}
}

// showSourceContext displays source code around current location
func (h *CLIHandler) showSourceContext(d *Debugger) {
	loc := d.GetCurrentLocation()
	if loc == nil {
		h.printf("No current location\n")
		return
	}

	h.showSourceLine(loc.File, loc.Line)
}

// handlePrint evaluates and prints an expression (TODO: needs expression evaluator)
func (h *CLIHandler) handlePrint(d *Debugger, args []string) {
	if len(args) == 0 {
		h.printf("Usage: print <expression>\n")
		return
	}

	// For now, just try to print a variable
	varName := args[0]
	vars := d.GetVariables(0) // Current frame
	if vars == nil {
		h.printf("No variables available\n")
		return
	}

	if val, ok := vars[varName]; ok {
		h.printf("%s = %s\n", varName, val.Inspect())
	} else {
		h.printf("Variable '%s' not found\n", varName)
	}
}

// showVariables displays all variables in the current or specified frame
func (h *CLIHandler) showVariables(d *Debugger, args []string) {
	frameIndex := 0
	if len(args) > 0 {
		var err error
		frameIndex, err = strconv.Atoi(args[0])
		if err != nil {
			h.printf("Invalid frame index: %s\n", args[0])
			return
		}
	}

	vars := d.GetVariables(frameIndex)
	if vars == nil || len(vars) == 0 {
		h.printf("No variables in frame %d\n", frameIndex)
		return
	}

	h.printf("\nVariables (frame %d):\n", frameIndex)
	for name, value := range vars {
		h.printf("  %s = %s\n", name, value.Inspect())
	}
}

// showHelp displays available commands
func (h *CLIHandler) showHelp() {
	h.printf(`
Debug Commands:
  s, step         Step into (execute next statement)
  n, next         Step over (execute next statement, skip function calls)
  o, out          Step out (continue until return from current function)
  c, continue     Continue execution until next breakpoint

  b, break <file>:<line>  Set breakpoint at location
  b, break                List all breakpoints

  bt, backtrace, stack    Show call stack
  l, list                 Show source code around current location
  p, print <var>          Print variable value
  v, vars [frame]         Show all variables in frame (default: current)

  h, help, ?              Show this help
  q, quit, exit           Exit debugger

Press Enter to repeat last command (useful for stepping)
`)
}

// printf writes formatted output
func (h *CLIHandler) printf(format string, args ...interface{}) {
	fmt.Fprintf(h.writer, format, args...)
}
