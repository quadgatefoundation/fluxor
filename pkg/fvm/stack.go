package fvm

import (
	"fmt"
)

// OperandStack is a stack for VM values during execution.
type OperandStack struct {
	values []*Value
	sp     int // Stack pointer (points to next free slot)
	maxSize int
}

// NewOperandStack creates a new operand stack.
func NewOperandStack(maxSize int) *OperandStack {
	return &OperandStack{
		values:  make([]*Value, maxSize),
		sp:      0,
		maxSize: maxSize,
	}
}

// Push pushes a value onto the stack.
func (s *OperandStack) Push(value *Value) error {
	if s.sp >= s.maxSize {
		return fmt.Errorf("stack overflow")
	}
	s.values[s.sp] = value
	s.sp++
	return nil
}

// Pop pops a value from the stack.
func (s *OperandStack) Pop() (*Value, error) {
	if s.sp <= 0 {
		return nil, fmt.Errorf("stack underflow")
	}
	s.sp--
	return s.values[s.sp], nil
}

// Peek returns the top value without popping.
func (s *OperandStack) Peek() (*Value, error) {
	if s.sp <= 0 {
		return nil, fmt.Errorf("stack is empty")
	}
	return s.values[s.sp-1], nil
}

// Dup duplicates the top value.
func (s *OperandStack) Dup() error {
	if s.sp <= 0 {
		return fmt.Errorf("stack is empty")
	}
	if s.sp >= s.maxSize {
		return fmt.Errorf("stack overflow")
	}
	s.values[s.sp] = s.values[s.sp-1]
	s.sp++
	return nil
}

// Swap swaps the top two values.
func (s *OperandStack) Swap() error {
	if s.sp < 2 {
		return fmt.Errorf("need at least 2 values on stack")
	}
	s.values[s.sp-1], s.values[s.sp-2] = s.values[s.sp-2], s.values[s.sp-1]
	return nil
}

// Size returns the current stack size.
func (s *OperandStack) Size() int {
	return s.sp
}

// Clear clears the stack.
func (s *OperandStack) Clear() {
	s.sp = 0
}

// String returns a string representation of the stack.
func (s *OperandStack) String() string {
	str := "["
	for i := 0; i < s.sp; i++ {
		if i > 0 {
			str += ", "
		}
		str += s.values[i].String()
	}
	str += "]"
	return str
}

// Frame represents a call frame in the VM.
type Frame struct {
	Method     *Method        // Method being executed
	PC         int            // Program counter
	Stack      *OperandStack  // Operand stack for this frame
	Locals     []*Value       // Local variables
	ReturnAddr int            // Return address for this frame
	Exception  *Value         // Current exception (if any)
}

// NewFrame creates a new call frame.
func NewFrame(method *Method, maxStack int, maxLocals int) *Frame {
	return &Frame{
		Method: method,
		PC:     0,
		Stack:  NewOperandStack(maxStack),
		Locals: make([]*Value, maxLocals),
	}
}

// GetLocal gets a local variable.
func (f *Frame) GetLocal(index int) (*Value, error) {
	if index < 0 || index >= len(f.Locals) {
		return nil, fmt.Errorf("local variable index out of bounds: %d", index)
	}
	val := f.Locals[index]
	if val == nil {
		return nil, fmt.Errorf("local variable %d not initialized", index)
	}
	return val, nil
}

// SetLocal sets a local variable.
func (f *Frame) SetLocal(index int, value *Value) error {
	if index < 0 || index >= len(f.Locals) {
		return fmt.Errorf("local variable index out of bounds: %d", index)
	}
	f.Locals[index] = value
	return nil
}

// String returns a string representation of the frame.
func (f *Frame) String() string {
	return fmt.Sprintf("Frame{method=%s, pc=%d, stack=%s}",
		f.Method.Name, f.PC, f.Stack.String())
}

// CallStack manages the call frames.
type CallStack struct {
	frames   []*Frame
	sp       int // Stack pointer (index of current frame)
	maxDepth int
}

// NewCallStack creates a new call stack.
func NewCallStack(maxDepth int) *CallStack {
	return &CallStack{
		frames:   make([]*Frame, maxDepth),
		sp:       -1,
		maxDepth: maxDepth,
	}
}

// Push pushes a new frame onto the call stack.
func (cs *CallStack) Push(frame *Frame) error {
	if cs.sp >= cs.maxDepth-1 {
		return fmt.Errorf("call stack overflow (max depth: %d)", cs.maxDepth)
	}
	cs.sp++
	cs.frames[cs.sp] = frame
	return nil
}

// Pop pops the current frame from the call stack.
func (cs *CallStack) Pop() (*Frame, error) {
	if cs.sp < 0 {
		return nil, fmt.Errorf("call stack underflow")
	}
	frame := cs.frames[cs.sp]
	cs.sp--
	return frame, nil
}

// Current returns the current frame.
func (cs *CallStack) Current() (*Frame, error) {
	if cs.sp < 0 {
		return nil, fmt.Errorf("no current frame")
	}
	return cs.frames[cs.sp], nil
}

// Depth returns the current call stack depth.
func (cs *CallStack) Depth() int {
	return cs.sp + 1
}

// IsEmpty returns true if the call stack is empty.
func (cs *CallStack) IsEmpty() bool {
	return cs.sp < 0
}

// GetTrace returns a stack trace.
func (cs *CallStack) GetTrace() []string {
	trace := make([]string, 0, cs.Depth())
	for i := cs.sp; i >= 0; i-- {
		frame := cs.frames[i]
		trace = append(trace, fmt.Sprintf("  at %s (PC=%d)", frame.Method.Name, frame.PC))
	}
	return trace
}
