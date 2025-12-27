package debugger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/marshallburns/ez/pkg/ast"
	"github.com/marshallburns/ez/pkg/object"
)

// JSONProtocol handles JSON-RPC communication for IDE integration
type JSONProtocol struct {
	reader   *bufio.Reader
	writer   io.Writer
	debugger *Debugger
	mu       sync.Mutex
	running  bool
}

// Message types
const (
	MessageTypeCommand = "command"
	MessageTypeEvent   = "event"
)

// Command names
const (
	CmdInitialize      = "initialize"
	CmdStart           = "start"
	CmdStepInto        = "stepInto"
	CmdStepOver        = "stepOver"
	CmdStepOut         = "stepOut"
	CmdContinue        = "continue"
	CmdSetBreakpoint   = "setBreakpoint"
	CmdClearBreakpoint = "clearBreakpoint"
	CmdGetVariables    = "getVariables"
	CmdGetStackTrace   = "getStackTrace"
	CmdEvaluate        = "evaluate"
	CmdTerminate       = "terminate"
)

// Event names
const (
	EvtInitialized      = "initialized"
	EvtStarted          = "started"
	EvtStopped          = "stopped"
	EvtOutput           = "output"
	EvtError            = "error"
	EvtVariableUpdate   = "variableUpdate"
	EvtVariables        = "variables"
	EvtStackTrace       = "stackTrace"
	EvtBreakpointSet    = "breakpointSet"
	EvtBreakpointCleared = "breakpointCleared"
	EvtExited           = "exited"
	EvtTerminated       = "terminated"
)

// Message represents a protocol message
type Message struct {
	Type string `json:"type"` // "command" or "event"
}

// CommandMessage represents a command from the IDE
type CommandMessage struct {
	Type    string                 `json:"type"`
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

// EventMessage represents an event to the IDE
type EventMessage struct {
	Type  string                 `json:"type"`
	Event string                 `json:"event"`
	Data  map[string]interface{} `json:"data"`
}

// Location represents a source location
type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Variable represents a variable with its value and type
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// StackFrame represents a call stack frame
type StackFrame struct {
	Index    int    `json:"index"`
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// NewJSONProtocol creates a new JSON protocol handler
func NewJSONProtocol(d *Debugger, reader io.Reader, writer io.Writer) *JSONProtocol {
	if reader == nil {
		reader = os.Stdin
	}
	if writer == nil {
		writer = os.Stdout
	}

	return &JSONProtocol{
		reader:   bufio.NewReader(reader),
		writer:   writer,
		debugger: d,
		running:  false,
	}
}

// Start begins processing messages
func (p *JSONProtocol) Start() error {
	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	for p.running {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := p.handleMessage(line); err != nil {
			p.sendError(err.Error(), "")
		}
	}

	return nil
}

// Stop stops message processing
func (p *JSONProtocol) Stop() {
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
}

// handleMessage processes a single message
func (p *JSONProtocol) handleMessage(line string) error {
	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if msg.Type == MessageTypeCommand {
		var cmd CommandMessage
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			return fmt.Errorf("invalid command: %w", err)
		}
		return p.handleCommand(&cmd)
	}

	return fmt.Errorf("unknown message type: %s", msg.Type)
}

// handleCommand processes a command message
func (p *JSONProtocol) handleCommand(cmd *CommandMessage) error {
	switch cmd.Command {
	case CmdInitialize:
		return p.handleInitialize(cmd.Params)
	case CmdStart:
		return p.handleStart(cmd.Params)
	case CmdStepInto:
		return p.handleStepInto(cmd.Params)
	case CmdStepOver:
		return p.handleStepOver(cmd.Params)
	case CmdStepOut:
		return p.handleStepOut(cmd.Params)
	case CmdContinue:
		return p.handleContinue(cmd.Params)
	case CmdSetBreakpoint:
		return p.handleSetBreakpoint(cmd.Params)
	case CmdClearBreakpoint:
		return p.handleClearBreakpoint(cmd.Params)
	case CmdGetVariables:
		return p.handleGetVariables(cmd.Params)
	case CmdGetStackTrace:
		return p.handleGetStackTrace(cmd.Params)
	case CmdEvaluate:
		return p.handleEvaluate(cmd.Params)
	case CmdTerminate:
		return p.handleTerminate(cmd.Params)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Command)
	}
}

// Command handlers

func (p *JSONProtocol) handleInitialize(params map[string]interface{}) error {
	file, _ := params["file"].(string)
	// workingDir, _ := params["workingDir"].(string)

	// Initialization logic would go here
	// For now, just acknowledge

	return p.sendEvent(EvtInitialized, map[string]interface{}{
		"file": file,
	})
}

func (p *JSONProtocol) handleStart(params map[string]interface{}) error {
	p.debugger.StepInto() // Start in step mode

	if err := p.sendEvent(EvtStarted, map[string]interface{}{}); err != nil {
		return err
	}

	// Note: The actual "stopped" event will be sent by the event handler
	return nil
}

func (p *JSONProtocol) handleStepInto(params map[string]interface{}) error {
	p.debugger.StepInto()
	return nil
}

func (p *JSONProtocol) handleStepOver(params map[string]interface{}) error {
	p.debugger.StepOver()
	return nil
}

func (p *JSONProtocol) handleStepOut(params map[string]interface{}) error {
	p.debugger.StepOut()
	return nil
}

func (p *JSONProtocol) handleContinue(params map[string]interface{}) error {
	p.debugger.Continue()
	return nil
}

func (p *JSONProtocol) handleSetBreakpoint(params map[string]interface{}) error {
	file, _ := params["file"].(string)
	line, _ := params["line"].(float64) // JSON numbers are float64

	p.debugger.SetBreakpoint(file, int(line))

	return p.sendEvent(EvtBreakpointSet, map[string]interface{}{
		"file": file,
		"line": int(line),
		"id":   1, // TODO: actual breakpoint ID
	})
}

func (p *JSONProtocol) handleClearBreakpoint(params map[string]interface{}) error {
	file, _ := params["file"].(string)
	line, _ := params["line"].(float64)

	p.debugger.ClearBreakpoint(file, int(line))

	return p.sendEvent(EvtBreakpointCleared, map[string]interface{}{
		"file": file,
		"line": int(line),
	})
}

func (p *JSONProtocol) handleGetVariables(params map[string]interface{}) error {
	frameIndex, _ := params["frameIndex"].(float64)

	vars := p.debugger.GetVariables(int(frameIndex))
	variables := make([]Variable, 0, len(vars))

	for name, value := range vars {
		variables = append(variables, Variable{
			Name:  name,
			Value: value.Inspect(),
			Type:  string(value.Type()),
		})
	}

	return p.sendEvent(EvtVariables, map[string]interface{}{
		"frameIndex": int(frameIndex),
		"variables":  variables,
	})
}

func (p *JSONProtocol) handleGetStackTrace(params map[string]interface{}) error {
	stack := p.debugger.GetCallStack()
	frames := make([]StackFrame, len(stack))

	for i, frame := range stack {
		frames[len(stack)-1-i] = StackFrame{
			Index:    len(stack) - 1 - i,
			Function: frame.FunctionName,
			File:     frame.Location.File,
			Line:     frame.Location.Line,
			Column:   frame.Location.Column,
		}
	}

	return p.sendEvent(EvtStackTrace, map[string]interface{}{
		"frames": frames,
	})
}

func (p *JSONProtocol) handleEvaluate(params map[string]interface{}) error {
	// expr, _ := params["expression"].(string)
	// frameIndex, _ := params["frameIndex"].(float64)

	// TODO: Implement expression evaluation
	return p.sendError("Expression evaluation not yet implemented", "")
}

func (p *JSONProtocol) handleTerminate(params map[string]interface{}) error {
	if err := p.sendEvent(EvtTerminated, map[string]interface{}{}); err != nil {
		return err
	}
	p.Stop()
	return nil
}

// Event senders

func (p *JSONProtocol) sendEvent(event string, data map[string]interface{}) error {
	msg := EventMessage{
		Type:  MessageTypeEvent,
		Event: event,
		Data:  data,
	}
	return p.sendMessage(&msg)
}

func (p *JSONProtocol) sendError(message string, command string) error {
	data := map[string]interface{}{
		"message": message,
	}
	if command != "" {
		data["command"] = command
	}
	return p.sendEvent(EvtError, data)
}

func (p *JSONProtocol) sendMessage(msg interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(p.writer, "%s\n", data)
	return err
}

// JSONEventHandler implements EventHandler for JSON protocol
type JSONEventHandler struct {
	protocol *JSONProtocol
}

// NewJSONEventHandler creates a new JSON event handler
func NewJSONEventHandler(protocol *JSONProtocol) *JSONEventHandler {
	return &JSONEventHandler{
		protocol: protocol,
	}
}

// OnBreakpoint is called when a breakpoint is hit
func (h *JSONEventHandler) OnBreakpoint(d *Debugger, bp *Breakpoint) {
	loc := d.GetCurrentLocation()
	data := map[string]interface{}{
		"reason": "breakpoint",
		"location": map[string]interface{}{
			"file":   loc.File,
			"line":   loc.Line,
			"column": loc.Column,
		},
		"breakpointId": 1, // TODO: actual breakpoint ID
	}
	h.protocol.sendEvent(EvtStopped, data)
}

// OnStep is called when stepping through code
func (h *JSONEventHandler) OnStep(d *Debugger, node ast.Node, env *object.Environment) {
	loc := getNodeLocation(node)
	if loc == nil {
		return
	}

	data := map[string]interface{}{
		"reason": "step",
		"location": map[string]interface{}{
			"file":   loc.File,
			"line":   loc.Line,
			"column": loc.Column,
		},
	}
	h.protocol.sendEvent(EvtStopped, data)

	// Send variable updates for current frame
	vars := d.GetVariables(0)
	for name, value := range vars {
		h.protocol.sendEvent(EvtVariableUpdate, map[string]interface{}{
			"name":       name,
			"value":      value.Inspect(),
			"type":       string(value.Type()),
			"frameIndex": 0,
		})
	}
}

// OnFunctionCall is called when entering a function
func (h *JSONEventHandler) OnFunctionCall(d *Debugger, frame *CallFrame) {
	// Silent for JSON protocol
}

// OnFunctionReturn is called when returning from a function
func (h *JSONEventHandler) OnFunctionReturn(d *Debugger, frame *CallFrame, result object.Object) {
	// Silent for JSON protocol
}

// OnError is called when an error occurs
func (h *JSONEventHandler) OnError(d *Debugger, err error) {
	h.protocol.sendError(err.Error(), "")
}
