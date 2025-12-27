package debugger

import (
	"fmt"
	"sync"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/object"
)

// StepMode defines how the debugger should step through code
type StepMode int

const (
	// Continue running until next breakpoint
	ModeContinue StepMode = iota
	// Step to next statement, stepping into function calls
	ModeStepInto
	// Step to next statement, stepping over function calls
	ModeStepOver
	// Run until return from current function
	ModeStepOut
)

// CallFrame represents a single function call in the call stack
type CallFrame struct {
	FunctionName string
	Node         ast.Node
	Env          *object.Environment
	Location     *ast.Location
	CallDepth    int // Depth in call stack (for step over/out)
}

// Breakpoint represents a location where execution should pause
type Breakpoint struct {
	File      string
	Line      int
	Condition string // Optional condition expression
	Enabled   bool
}

// Debugger manages debug state and controls execution
type Debugger struct {
	enabled      bool
	stepMode     StepMode
	callStack    []*CallFrame
	breakpoints  map[string]map[int]*Breakpoint // file -> line -> breakpoint
	stepDepth    int                             // Call depth for step over/out
	paused       bool
	pauseChan    chan bool // Channel for pausing execution
	resumeChan   chan bool // Channel for resuming execution
	currentNode  ast.Node
	currentEnv   *object.Environment
	mu           sync.RWMutex
	eventHandler EventHandler
}

// EventHandler is an interface for handling debug events
type EventHandler interface {
	OnBreakpoint(d *Debugger, bp *Breakpoint)
	OnStep(d *Debugger, node ast.Node, env *object.Environment)
	OnFunctionCall(d *Debugger, frame *CallFrame)
	OnFunctionReturn(d *Debugger, frame *CallFrame, result object.Object)
	OnError(d *Debugger, err error)
}

// Global debugger instance
var globalDebugger *Debugger
var debuggerMu sync.RWMutex

// New creates a new debugger instance
func New() *Debugger {
	return &Debugger{
		enabled:     false,
		stepMode:    ModeContinue,
		callStack:   make([]*CallFrame, 0),
		breakpoints: make(map[string]map[int]*Breakpoint),
		pauseChan:   make(chan bool, 1),
		resumeChan:  make(chan bool, 1),
	}
}

// SetGlobalDebugger sets the global debugger instance
func SetGlobalDebugger(d *Debugger) {
	debuggerMu.Lock()
	defer debuggerMu.Unlock()
	globalDebugger = d
}

// GetGlobalDebugger returns the global debugger instance
func GetGlobalDebugger() *Debugger {
	debuggerMu.RLock()
	defer debuggerMu.RUnlock()
	return globalDebugger
}

// Enable enables the debugger
func (d *Debugger) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

// Disable disables the debugger
func (d *Debugger) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
}

// IsEnabled returns whether the debugger is enabled
func (d *Debugger) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetEventHandler sets the event handler
func (d *Debugger) SetEventHandler(handler EventHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.eventHandler = handler
}

// SetBreakpoint sets a breakpoint at the specified file and line
func (d *Debugger) SetBreakpoint(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.breakpoints[file] == nil {
		d.breakpoints[file] = make(map[int]*Breakpoint)
	}

	d.breakpoints[file][line] = &Breakpoint{
		File:    file,
		Line:    line,
		Enabled: true,
	}
}

// ClearBreakpoint removes a breakpoint at the specified file and line
func (d *Debugger) ClearBreakpoint(file string, line int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.breakpoints[file] != nil {
		delete(d.breakpoints[file], line)
	}
}

// ClearAllBreakpoints removes all breakpoints
func (d *Debugger) ClearAllBreakpoints() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.breakpoints = make(map[string]map[int]*Breakpoint)
}

// GetBreakpoints returns all breakpoints
func (d *Debugger) GetBreakpoints() []*Breakpoint {
	d.mu.RLock()
	defer d.mu.RUnlock()

	bps := make([]*Breakpoint, 0)
	for _, fileBreakpoints := range d.breakpoints {
		for _, bp := range fileBreakpoints {
			bps = append(bps, bp)
		}
	}
	return bps
}

// shouldBreak checks if execution should break at the current node
func (d *Debugger) shouldBreak(node ast.Node) bool {
	if node == nil {
		return false
	}

	loc := getNodeLocation(node)
	if loc == nil {
		return false
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	fileBreakpoints := d.breakpoints[loc.File]
	if fileBreakpoints == nil {
		return false
	}

	bp := fileBreakpoints[loc.Line]
	if bp == nil || !bp.Enabled {
		return false
	}

	// TODO: Evaluate condition if present
	return true
}

// shouldPause determines if execution should pause based on step mode
func (d *Debugger) shouldPause(node ast.Node) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	switch d.stepMode {
	case ModeContinue:
		return d.shouldBreak(node)

	case ModeStepInto:
		// Pause at every statement
		return isSteppableNode(node)

	case ModeStepOver:
		// Pause at statements at current depth or shallower
		currentDepth := len(d.callStack)
		return isSteppableNode(node) && currentDepth <= d.stepDepth

	case ModeStepOut:
		// Pause when we return to a shallower depth
		currentDepth := len(d.callStack)
		return currentDepth < d.stepDepth

	default:
		return false
	}
}

// BeforeEval is called before evaluating a node
func (d *Debugger) BeforeEval(node ast.Node, env *object.Environment) {
	if !d.IsEnabled() {
		return
	}

	d.mu.Lock()
	d.currentNode = node
	d.currentEnv = env
	d.mu.Unlock()

	if d.shouldPause(node) {
		d.pause(node, env)
	}
}

// AfterEval is called after evaluating a node
func (d *Debugger) AfterEval(node ast.Node, result object.Object, env *object.Environment) {
	// Currently no-op, but could be used for watch expressions, etc.
}

// PushFrame adds a call frame to the call stack
func (d *Debugger) PushFrame(functionName string, node ast.Node, env *object.Environment, loc *ast.Location) {
	if !d.IsEnabled() {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	frame := &CallFrame{
		FunctionName: functionName,
		Node:         node,
		Env:          env,
		Location:     loc,
		CallDepth:    len(d.callStack),
	}

	d.callStack = append(d.callStack, frame)

	if d.eventHandler != nil {
		d.mu.Unlock()
		d.eventHandler.OnFunctionCall(d, frame)
		d.mu.Lock()
	}
}

// PopFrame removes the top call frame from the call stack
func (d *Debugger) PopFrame(result object.Object) {
	if !d.IsEnabled() {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.callStack) == 0 {
		return
	}

	frame := d.callStack[len(d.callStack)-1]
	d.callStack = d.callStack[:len(d.callStack)-1]

	if d.eventHandler != nil {
		d.mu.Unlock()
		d.eventHandler.OnFunctionReturn(d, frame, result)
		d.mu.Lock()
	}
}

// pause pauses execution and waits for resume
func (d *Debugger) pause(node ast.Node, env *object.Environment) {
	d.mu.Lock()
	d.paused = true

	if d.eventHandler != nil {
		bp := d.getBreakpointAt(node)
		handler := d.eventHandler
		d.mu.Unlock()

		if bp != nil {
			handler.OnBreakpoint(d, bp)
		} else {
			handler.OnStep(d, node, env)
		}

		d.mu.Lock()
	}
	d.mu.Unlock()

	// Wait for resume signal
	<-d.resumeChan

	d.mu.Lock()
	d.paused = false
	d.mu.Unlock()
}

// getBreakpointAt returns the breakpoint at the given node's location, if any
func (d *Debugger) getBreakpointAt(node ast.Node) *Breakpoint {
	loc := getNodeLocation(node)
	if loc == nil {
		return nil
	}

	fileBreakpoints := d.breakpoints[loc.File]
	if fileBreakpoints == nil {
		return nil
	}

	return fileBreakpoints[loc.Line]
}

// Continue resumes execution until next breakpoint
func (d *Debugger) Continue() {
	d.mu.Lock()
	d.stepMode = ModeContinue
	d.mu.Unlock()
	d.resume()
}

// StepInto steps to the next statement, entering function calls
func (d *Debugger) StepInto() {
	d.mu.Lock()
	d.stepMode = ModeStepInto
	d.stepDepth = len(d.callStack)
	d.mu.Unlock()
	d.resume()
}

// StepOver steps to the next statement, skipping over function calls
func (d *Debugger) StepOver() {
	d.mu.Lock()
	d.stepMode = ModeStepOver
	d.stepDepth = len(d.callStack)
	d.mu.Unlock()
	d.resume()
}

// StepOut runs until return from current function
func (d *Debugger) StepOut() {
	d.mu.Lock()
	d.stepMode = ModeStepOut
	d.stepDepth = len(d.callStack)
	d.mu.Unlock()
	d.resume()
}

// resume sends the resume signal
func (d *Debugger) resume() {
	select {
	case d.resumeChan <- true:
	default:
		// Channel already has a value or not blocking
	}
}

// GetCallStack returns a copy of the current call stack
func (d *Debugger) GetCallStack() []*CallFrame {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stack := make([]*CallFrame, len(d.callStack))
	copy(stack, d.callStack)
	return stack
}

// GetVariables returns variables from the specified call frame (0 = current)
func (d *Debugger) GetVariables(frameIndex int) map[string]object.Object {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if frameIndex < 0 || frameIndex >= len(d.callStack) {
		return nil
	}

	frame := d.callStack[len(d.callStack)-1-frameIndex]
	return getAllVariables(frame.Env)
}

// GetCurrentLocation returns the current execution location
func (d *Debugger) GetCurrentLocation() *ast.Location {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return getNodeLocation(d.currentNode)
}

// IsPaused returns whether execution is currently paused
func (d *Debugger) IsPaused() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.paused
}

// Helper functions

// getNodeLocation extracts location from an AST node's Token
func getNodeLocation(node ast.Node) *ast.Location {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		if len(n.Statements) > 0 {
			return getNodeLocation(n.Statements[0])
		}
	case *ast.VariableDeclaration:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.FunctionDeclaration:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.ReturnStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.IfStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.WhileStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.ForStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.ForEachStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.LoopStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.ExpressionStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.AssignmentStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.BreakStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.ContinueStatement:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	case *ast.BlockStatement:
		if len(n.Statements) > 0 {
			return getNodeLocation(n.Statements[0])
		}
	case *ast.CallExpression:
		return &ast.Location{File: n.Token.File, Line: n.Token.Line, Column: n.Token.Column}
	}

	return nil
}

// isSteppableNode returns true if the node represents a steppable statement
func isSteppableNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.VariableDeclaration,
		*ast.AssignmentStatement,
		*ast.ExpressionStatement,
		*ast.ReturnStatement,
		*ast.IfStatement,
		*ast.WhileStatement,
		*ast.ForStatement,
		*ast.ForEachStatement,
		*ast.LoopStatement,
		*ast.BreakStatement,
		*ast.ContinueStatement:
		return true
	}
	return false
}

// getAllVariables extracts all variables from an environment and its parents
func getAllVariables(env *object.Environment) map[string]object.Object {
	vars := make(map[string]object.Object)

	// Walk up the environment chain
	for env != nil {
		envVars := env.GetAll()
		for name, value := range envVars {
			// Only add if not already present (inner scope shadows outer)
			if _, exists := vars[name]; !exists {
				vars[name] = value
			}
		}
		env = env.Outer()
	}

	return vars
}

// FormatLocation returns a human-readable location string
func FormatLocation(loc *ast.Location) string {
	if loc == nil {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Column)
}
