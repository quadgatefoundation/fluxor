package fvm

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
)

// VM represents the Fluxor Virtual Machine.
type VM struct {
	modules    map[string]*Module
	callStack  *CallStack
	globals    map[string]*Value
	eventBus   core.EventBus
	vertx      core.Vertx
	ctx        context.Context
	logger     core.Logger
	halted     bool
	maxStack   int
	maxCallDepth int
}

// NewVM creates a new virtual machine.
func NewVM() *VM {
	return &VM{
		modules:      make(map[string]*Module),
		callStack:    NewCallStack(1000),
		globals:      make(map[string]*Value),
		logger:       core.NewDefaultLogger(),
		maxStack:     1000,
		maxCallDepth: 1000,
	}
}

// WithEventBus sets the EventBus for the VM.
func (vm *VM) WithEventBus(eb core.EventBus) *VM {
	vm.eventBus = eb
	return vm
}

// WithVertx sets the Vertx runtime for the VM.
func (vm *VM) WithVertx(v core.Vertx) *VM {
	vm.vertx = v
	return vm
}

// WithContext sets the context for the VM.
func (vm *VM) WithContext(ctx context.Context) *VM {
	vm.ctx = ctx
	return vm
}

// LoadModule loads a module into the VM.
func (vm *VM) LoadModule(module *Module) error {
	if _, exists := vm.modules[module.Name]; exists {
		return fmt.Errorf("module already loaded: %s", module.Name)
	}
	vm.modules[module.Name] = module
	vm.logger.Infof("Loaded module: %s", module.Name)
	return nil
}

// GetModule gets a loaded module by name.
func (vm *VM) GetModule(name string) (*Module, error) {
	module, ok := vm.modules[name]
	if !ok {
		return nil, fmt.Errorf("module not found: %s", name)
	}
	return module, nil
}

// Execute executes a method in a module.
func (vm *VM) Execute(moduleName, methodName string, args ...*Value) (*Value, error) {
	module, err := vm.GetModule(moduleName)
	if err != nil {
		return nil, err
	}

	method, err := module.GetMethod(methodName)
	if err != nil {
		return nil, err
	}

	return vm.InvokeMethod(method, args...)
}

// InvokeMethod invokes a method directly.
func (vm *VM) InvokeMethod(method *Method, args ...*Value) (*Value, error) {
	// Create new frame
	frame := NewFrame(method, vm.maxStack, method.MaxLocals)

	// Initialize arguments as local variables
	for i, arg := range args {
		if i >= method.MaxLocals {
			return nil, fmt.Errorf("too many arguments")
		}
		frame.Locals[i] = arg
	}

	// Push frame onto call stack
	if err := vm.callStack.Push(frame); err != nil {
		return nil, err
	}

	// Execute
	result, err := vm.executeFrame()
	
	// Pop frame
	vm.callStack.Pop()

	return result, err
}

// executeFrame executes the current frame.
func (vm *VM) executeFrame() (*Value, error) {
	frame, err := vm.callStack.Current()
	if err != nil {
		return nil, err
	}

	vm.halted = false

	for !vm.halted && frame.PC < len(frame.Method.Code) {
		// Check context cancellation
		if vm.ctx != nil {
			select {
			case <-vm.ctx.Done():
				return nil, vm.ctx.Err()
			default:
			}
		}

		// Fetch instruction
		instr := frame.Method.Code[frame.PC]
		frame.PC++

		// Execute instruction
		if err := vm.executeInstruction(frame, instr); err != nil {
			return nil, fmt.Errorf("execution error at PC=%d: %w", frame.PC-1, err)
		}
	}

	// If frame still has a value on stack, return it
	if frame.Stack.Size() > 0 {
		return frame.Stack.Pop()
	}

	return NewVoidValue(), nil
}

// executeInstruction executes a single instruction.
func (vm *VM) executeInstruction(frame *Frame, instr Instruction) error {
	switch instr.Op {
	// Arithmetic operations
	case OpAdd:
		return vm.execAdd(frame)
	case OpSub:
		return vm.execSub(frame)
	case OpMul:
		return vm.execMul(frame)
	case OpDiv:
		return vm.execDiv(frame)
	case OpMod:
		return vm.execMod(frame)
	case OpNeg:
		return vm.execNeg(frame)

	// Comparison operations
	case OpEq:
		return vm.execEq(frame)
	case OpNe:
		return vm.execNe(frame)
	case OpLt:
		return vm.execLt(frame)
	case OpLe:
		return vm.execLe(frame)
	case OpGt:
		return vm.execGt(frame)
	case OpGe:
		return vm.execGe(frame)

	// Logical operations
	case OpAnd:
		return vm.execAnd(frame)
	case OpOr:
		return vm.execOr(frame)
	case OpNot:
		return vm.execNot(frame)

	// Stack operations
	case OpPush:
		return frame.Stack.Push(NewIntValue(instr.Operand))
	case OpPop:
		_, err := frame.Stack.Pop()
		return err
	case OpDup:
		return frame.Stack.Dup()
	case OpSwap:
		return frame.Stack.Swap()

	// Local variable operations
	case OpLoad:
		val, err := frame.GetLocal(int(instr.Operand))
		if err != nil {
			return err
		}
		return frame.Stack.Push(val)
	case OpStore:
		val, err := frame.Stack.Pop()
		if err != nil {
			return err
		}
		return frame.SetLocal(int(instr.Operand), val)

	// Control flow
	case OpJmp:
		frame.PC = int(instr.Operand)
		return nil
	case OpJz:
		return vm.execJz(frame, int(instr.Operand))
	case OpJnz:
		return vm.execJnz(frame, int(instr.Operand))
	case OpCall:
		return vm.execCall(frame, int(instr.Operand))
	case OpRet:
		vm.halted = true
		return nil
	case OpRetVal:
		vm.halted = true
		return nil

	// Constant loading
	case OpLoadInt:
		return frame.Stack.Push(NewIntValue(instr.Operand))
	case OpLoadBool:
		return frame.Stack.Push(NewBoolValue(instr.Operand != 0))
	case OpLoadNull:
		return frame.Stack.Push(NewNullValue())
	case OpLoadString:
		return vm.execLoadString(frame, int(instr.Operand))

	// Object operations
	case OpNew:
		return vm.execNew(frame, int(instr.Operand))
	case OpGetField:
		return vm.execGetField(frame, int(instr.Operand))
	case OpSetField:
		return vm.execSetField(frame, int(instr.Operand))

	// Array operations
	case OpNewArray:
		return vm.execNewArray(frame, int(instr.Operand))
	case OpArrayLen:
		return vm.execArrayLen(frame)
	case OpArrayLoad:
		return vm.execArrayLoad(frame)
	case OpArrayStore:
		return vm.execArrayStore(frame)

	// Special operations
	case OpNop:
		return nil
	case OpHalt:
		vm.halted = true
		return nil
	case OpPrint:
		return vm.execPrint(frame)

	// EventBus operations
	case OpEventBusSend:
		return vm.execEventBusSend(frame)
	case OpEventBusPublish:
		return vm.execEventBusPublish(frame)

	// Context operations
	case OpGetContext:
		return vm.execGetContext(frame)
	case OpGetVertx:
		return vm.execGetVertx(frame)

	default:
		return fmt.Errorf("unknown opcode: %s", instr.Op)
	}
}

// Arithmetic operations
func (vm *VM) execAdd(frame *Frame) error {
	b, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewIntValue(aVal + bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewFloatValue(aVal + bVal))
	}
	if a.Type == TypeString && b.Type == TypeString {
		aVal, _ := a.AsString()
		bVal, _ := b.AsString()
		return frame.Stack.Push(NewStringValue(aVal + bVal))
	}

	return fmt.Errorf("incompatible types for ADD: %s + %s", a.Type, b.Type)
}

func (vm *VM) execSub(frame *Frame) error {
	b, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewIntValue(aVal - bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewFloatValue(aVal - bVal))
	}

	return fmt.Errorf("incompatible types for SUB: %s - %s", a.Type, b.Type)
}

func (vm *VM) execMul(frame *Frame) error {
	b, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewIntValue(aVal * bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewFloatValue(aVal * bVal))
	}

	return fmt.Errorf("incompatible types for MUL: %s * %s", a.Type, b.Type)
}

func (vm *VM) execDiv(frame *Frame) error {
	b, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		if bVal == 0 {
			return fmt.Errorf("division by zero")
		}
		return frame.Stack.Push(NewIntValue(aVal / bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		if bVal == 0.0 {
			return fmt.Errorf("division by zero")
		}
		return frame.Stack.Push(NewFloatValue(aVal / bVal))
	}

	return fmt.Errorf("incompatible types for DIV: %s / %s", a.Type, b.Type)
}

func (vm *VM) execMod(frame *Frame) error {
	b, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		if bVal == 0 {
			return fmt.Errorf("modulo by zero")
		}
		return frame.Stack.Push(NewIntValue(aVal % bVal))
	}

	return fmt.Errorf("incompatible types for MOD: %s %% %s", a.Type, b.Type)
}

func (vm *VM) execNeg(frame *Frame) error {
	a, err := frame.Stack.Pop()
	if err != nil {
		return err
	}

	if a.Type == TypeInt {
		aVal, _ := a.AsInt()
		return frame.Stack.Push(NewIntValue(-aVal))
	}
	if a.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		return frame.Stack.Push(NewFloatValue(-aVal))
	}

	return fmt.Errorf("incompatible type for NEG: %s", a.Type)
}

// Comparison operations
func (vm *VM) execEq(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()
	return frame.Stack.Push(NewBoolValue(a.Equals(b)))
}

func (vm *VM) execNe(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()
	return frame.Stack.Push(NewBoolValue(!a.Equals(b)))
}

func (vm *VM) execLt(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewBoolValue(aVal < bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewBoolValue(aVal < bVal))
	}

	return fmt.Errorf("incompatible types for LT: %s < %s", a.Type, b.Type)
}

func (vm *VM) execLe(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewBoolValue(aVal <= bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewBoolValue(aVal <= bVal))
	}

	return fmt.Errorf("incompatible types for LE: %s <= %s", a.Type, b.Type)
}

func (vm *VM) execGt(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewBoolValue(aVal > bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewBoolValue(aVal > bVal))
	}

	return fmt.Errorf("incompatible types for GT: %s > %s", a.Type, b.Type)
}

func (vm *VM) execGe(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()

	if a.Type == TypeInt && b.Type == TypeInt {
		aVal, _ := a.AsInt()
		bVal, _ := b.AsInt()
		return frame.Stack.Push(NewBoolValue(aVal >= bVal))
	}
	if a.Type == TypeFloat && b.Type == TypeFloat {
		aVal, _ := a.AsFloat()
		bVal, _ := b.AsFloat()
		return frame.Stack.Push(NewBoolValue(aVal >= bVal))
	}

	return fmt.Errorf("incompatible types for GE: %s >= %s", a.Type, b.Type)
}

// Logical operations
func (vm *VM) execAnd(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()
	return frame.Stack.Push(NewBoolValue(a.IsTruthy() && b.IsTruthy()))
}

func (vm *VM) execOr(frame *Frame) error {
	b, _ := frame.Stack.Pop()
	a, _ := frame.Stack.Pop()
	return frame.Stack.Push(NewBoolValue(a.IsTruthy() || b.IsTruthy()))
}

func (vm *VM) execNot(frame *Frame) error {
	a, _ := frame.Stack.Pop()
	return frame.Stack.Push(NewBoolValue(!a.IsTruthy()))
}

// Control flow
func (vm *VM) execJz(frame *Frame, target int) error {
	val, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	if !val.IsTruthy() {
		frame.PC = target
	}
	return nil
}

func (vm *VM) execJnz(frame *Frame, target int) error {
	val, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	if val.IsTruthy() {
		frame.PC = target
	}
	return nil
}

func (vm *VM) execCall(frame *Frame, methodIndex int) error {
	// For now, simple placeholder - full implementation would resolve method
	return fmt.Errorf("CALL not yet fully implemented")
}

// Object operations
func (vm *VM) execNew(frame *Frame, typeIndex int) error {
	obj := NewObject(fmt.Sprintf("Type%d", typeIndex))
	return frame.Stack.Push(NewObjectValue(obj))
}

func (vm *VM) execGetField(frame *Frame, fieldIndex int) error {
	obj, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	objVal, err := obj.AsObject()
	if err != nil {
		return err
	}

	fieldName := fmt.Sprintf("field%d", fieldIndex)
	val, err := objVal.GetField(fieldName)
	if err != nil {
		return frame.Stack.Push(NewNullValue())
	}
	return frame.Stack.Push(val)
}

func (vm *VM) execSetField(frame *Frame, fieldIndex int) error {
	val, _ := frame.Stack.Pop()
	obj, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	objVal, err := obj.AsObject()
	if err != nil {
		return err
	}

	fieldName := fmt.Sprintf("field%d", fieldIndex)
	objVal.SetField(fieldName, val)
	return nil
}

// Array operations
func (vm *VM) execNewArray(frame *Frame, size int) error {
	arr := NewArray(TypeInt, size)
	return frame.Stack.Push(NewArrayValue(arr))
}

func (vm *VM) execArrayLen(frame *Frame) error {
	val, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	arr, err := val.AsArray()
	if err != nil {
		return err
	}
	return frame.Stack.Push(NewIntValue(int64(arr.Len())))
}

func (vm *VM) execArrayLoad(frame *Frame) error {
	index, _ := frame.Stack.Pop()
	arr, _ := frame.Stack.Pop()

	indexVal, _ := index.AsInt()
	arrVal, _ := arr.AsArray()

	elem, err := arrVal.Get(int(indexVal))
	if err != nil {
		return err
	}
	return frame.Stack.Push(elem)
}

func (vm *VM) execArrayStore(frame *Frame) error {
	val, _ := frame.Stack.Pop()
	index, _ := frame.Stack.Pop()
	arr, _ := frame.Stack.Pop()

	indexVal, _ := index.AsInt()
	arrVal, _ := arr.AsArray()

	return arrVal.Set(int(indexVal), val)
}

// Special operations
func (vm *VM) execPrint(frame *Frame) error {
	val, err := frame.Stack.Pop()
	if err != nil {
		return err
	}
	vm.logger.Info(fmt.Sprintf("[FVM] %s", val.String()))
	return nil
}

func (vm *VM) execLoadString(frame *Frame, index int) error {
	// Get string from module constant pool
	if frame.Method == nil {
		return fmt.Errorf("no method context")
	}

	// For now, just create a placeholder string
	return frame.Stack.Push(NewStringValue(fmt.Sprintf("string%d", index)))
}

// EventBus operations
func (vm *VM) execEventBusSend(frame *Frame) error {
	if vm.eventBus == nil {
		return fmt.Errorf("EventBus not available")
	}

	data, _ := frame.Stack.Pop()
	addr, _ := frame.Stack.Pop()

	addrStr, err := addr.AsString()
	if err != nil {
		return err
	}

	return vm.eventBus.Send(addrStr, data.Data)
}

func (vm *VM) execEventBusPublish(frame *Frame) error {
	if vm.eventBus == nil {
		return fmt.Errorf("EventBus not available")
	}

	data, _ := frame.Stack.Pop()
	addr, _ := frame.Stack.Pop()

	addrStr, err := addr.AsString()
	if err != nil {
		return err
	}

	return vm.eventBus.Publish(addrStr, data.Data)
}

// Context operations
func (vm *VM) execGetContext(frame *Frame) error {
	// Return a placeholder - full integration would return actual FluxorContext
	return frame.Stack.Push(NewObjectValue(NewObject("FluxorContext")))
}

func (vm *VM) execGetVertx(frame *Frame) error {
	// Return a placeholder - full integration would return actual Vertx
	return frame.Stack.Push(NewObjectValue(NewObject("Vertx")))
}

// GetStackTrace returns the current stack trace.
func (vm *VM) GetStackTrace() []string {
	return vm.callStack.GetTrace()
}
