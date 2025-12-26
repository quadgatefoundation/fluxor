package fvm

// Opcode represents a bytecode instruction opcode.
type Opcode byte

const (
	// Arithmetic operations
	OpAdd Opcode = 0x01 // ADD: pop b, pop a, push (a + b)
	OpSub Opcode = 0x02 // SUB: pop b, pop a, push (a - b)
	OpMul Opcode = 0x03 // MUL: pop b, pop a, push (a * b)
	OpDiv Opcode = 0x04 // DIV: pop b, pop a, push (a / b)
	OpMod Opcode = 0x05 // MOD: pop b, pop a, push (a % b)
	OpNeg Opcode = 0x06 // NEG: pop a, push (-a)

	// Comparison operations
	OpEq Opcode = 0x10 // EQ: pop b, pop a, push (a == b)
	OpNe Opcode = 0x11 // NE: pop b, pop a, push (a != b)
	OpLt Opcode = 0x12 // LT: pop b, pop a, push (a < b)
	OpLe Opcode = 0x13 // LE: pop b, pop a, push (a <= b)
	OpGt Opcode = 0x14 // GT: pop b, pop a, push (a > b)
	OpGe Opcode = 0x15 // GE: pop b, pop a, push (a >= b)

	// Logical operations
	OpAnd Opcode = 0x20 // AND: pop b, pop a, push (a && b)
	OpOr  Opcode = 0x21 // OR: pop b, pop a, push (a || b)
	OpNot Opcode = 0x22 // NOT: pop a, push (!a)

	// Stack operations
	OpPush Opcode = 0x30 // PUSH <value>: push constant
	OpPop  Opcode = 0x31 // POP: discard top of stack
	OpDup  Opcode = 0x32 // DUP: duplicate top of stack
	OpSwap Opcode = 0x33 // SWAP: swap top two values

	// Local variable operations
	OpLoad  Opcode = 0x40 // LOAD <index>: push local[index]
	OpStore Opcode = 0x41 // STORE <index>: pop to local[index]

	// Control flow
	OpJmp    Opcode = 0x50 // JMP <offset>: unconditional jump
	OpJz     Opcode = 0x51 // JZ <offset>: jump if zero/false
	OpJnz    Opcode = 0x52 // JNZ <offset>: jump if not zero/true
	OpCall   Opcode = 0x53 // CALL <method>: call method
	OpRet    Opcode = 0x54 // RET: return from method
	OpRetVal Opcode = 0x55 // RETVAL: return with value

	// Object operations
	OpNew      Opcode = 0x60 // NEW <type>: create new object
	OpGetField Opcode = 0x61 // GETFIELD <field>: get object field
	OpSetField Opcode = 0x62 // SETFIELD <field>: set object field
	OpInvoke   Opcode = 0x63 // INVOKE <method>: invoke method on object

	// Array operations
	OpNewArray   Opcode = 0x70 // NEWARRAY <type> <size>: create array
	OpArrayLen   Opcode = 0x71 // ARRAYLEN: get array length
	OpArrayLoad  Opcode = 0x72 // ALOAD: load from array
	OpArrayStore Opcode = 0x73 // ASTORE: store to array

	// Type operations
	OpCast       Opcode = 0x80 // CAST <type>: cast value
	OpInstanceOf Opcode = 0x81 // INSTANCEOF <type>: check type

	// Special operations
	OpNop   Opcode = 0x90 // NOP: no operation
	OpHalt  Opcode = 0x91 // HALT: stop execution
	OpThrow Opcode = 0x92 // THROW: throw exception
	OpPrint Opcode = 0x93 // PRINT: debug print (temp)

	// Constant loading
	OpLoadConst   Opcode = 0xA0 // LOADCONST <index>: load from constant pool
	OpLoadInt     Opcode = 0xA1 // LOADINT <value>: load integer constant
	OpLoadFloat   Opcode = 0xA2 // LOADFLOAT <value>: load float constant
	OpLoadString  Opcode = 0xA3 // LOADSTRING <index>: load string constant
	OpLoadBool    Opcode = 0xA4 // LOADBOOL <value>: load boolean constant
	OpLoadNull    Opcode = 0xA5 // LOADNULL: load null value

	// EventBus integration
	OpEventBusSend    Opcode = 0xB0 // EBSEND: send to EventBus
	OpEventBusPublish Opcode = 0xB1 // EBPUBLISH: publish to EventBus
	OpEventBusRequest Opcode = 0xB2 // EBREQUEST: request-reply on EventBus

	// Context operations
	OpGetContext Opcode = 0xC0 // GETCONTEXT: get FluxorContext
	OpGetVertx   Opcode = 0xC1 // GETVERTX: get Vertx
)

// Instruction represents a single bytecode instruction.
type Instruction struct {
	Op      Opcode // Instruction opcode
	Operand int64  // Operand (for instructions that need one)
}

// OpcodeInfo provides metadata about an opcode.
type OpcodeInfo struct {
	Name        string
	Description string
	StackEffect int  // Net effect on stack depth (+1 = push, -1 = pop)
	HasOperand  bool // Whether instruction has an operand
}

// OpcodeTable maps opcodes to their metadata.
var OpcodeTable = map[Opcode]OpcodeInfo{
	OpAdd: {"ADD", "Add two values", -1, false},
	OpSub: {"SUB", "Subtract two values", -1, false},
	OpMul: {"MUL", "Multiply two values", -1, false},
	OpDiv: {"DIV", "Divide two values", -1, false},
	OpMod: {"MOD", "Modulo operation", -1, false},
	OpNeg: {"NEG", "Negate value", 0, false},

	OpEq: {"EQ", "Equal comparison", -1, false},
	OpNe: {"NE", "Not equal comparison", -1, false},
	OpLt: {"LT", "Less than comparison", -1, false},
	OpLe: {"LE", "Less or equal comparison", -1, false},
	OpGt: {"GT", "Greater than comparison", -1, false},
	OpGe: {"GE", "Greater or equal comparison", -1, false},

	OpAnd: {"AND", "Logical AND", -1, false},
	OpOr:  {"OR", "Logical OR", -1, false},
	OpNot: {"NOT", "Logical NOT", 0, false},

	OpPush: {"PUSH", "Push constant", 1, true},
	OpPop:  {"POP", "Pop value", -1, false},
	OpDup:  {"DUP", "Duplicate top", 1, false},
	OpSwap: {"SWAP", "Swap top two", 0, false},

	OpLoad:  {"LOAD", "Load local variable", 1, true},
	OpStore: {"STORE", "Store local variable", -1, true},

	OpJmp:    {"JMP", "Unconditional jump", 0, true},
	OpJz:     {"JZ", "Jump if zero", -1, true},
	OpJnz:    {"JNZ", "Jump if not zero", -1, true},
	OpCall:   {"CALL", "Call method", 0, true},
	OpRet:    {"RET", "Return from method", 0, false},
	OpRetVal: {"RETVAL", "Return with value", -1, false},

	OpNew:      {"NEW", "Create object", 1, true},
	OpGetField: {"GETFIELD", "Get object field", 0, true},
	OpSetField: {"SETFIELD", "Set object field", -2, true},
	OpInvoke:   {"INVOKE", "Invoke method", -1, true},

	OpNewArray:   {"NEWARRAY", "Create array", 0, true},
	OpArrayLen:   {"ARRAYLEN", "Get array length", 0, false},
	OpArrayLoad:  {"ALOAD", "Load from array", -1, false},
	OpArrayStore: {"ASTORE", "Store to array", -3, false},

	OpCast:       {"CAST", "Cast value", 0, true},
	OpInstanceOf: {"INSTANCEOF", "Type check", 0, true},

	OpNop:   {"NOP", "No operation", 0, false},
	OpHalt:  {"HALT", "Halt execution", 0, false},
	OpThrow: {"THROW", "Throw exception", -1, false},
	OpPrint: {"PRINT", "Debug print", -1, false},

	OpLoadConst:  {"LOADCONST", "Load constant", 1, true},
	OpLoadInt:    {"LOADINT", "Load integer", 1, true},
	OpLoadFloat:  {"LOADFLOAT", "Load float", 1, true},
	OpLoadString: {"LOADSTRING", "Load string", 1, true},
	OpLoadBool:   {"LOADBOOL", "Load boolean", 1, true},
	OpLoadNull:   {"LOADNULL", "Load null", 1, false},

	OpEventBusSend:    {"EBSEND", "EventBus send", -2, false},
	OpEventBusPublish: {"EBPUBLISH", "EventBus publish", -2, false},
	OpEventBusRequest: {"EBREQUEST", "EventBus request", -2, false},

	OpGetContext: {"GETCONTEXT", "Get FluxorContext", 1, false},
	OpGetVertx:   {"GETVERTX", "Get Vertx", 1, false},
}

// GetInfo returns metadata for an opcode.
func (op Opcode) GetInfo() OpcodeInfo {
	info, ok := OpcodeTable[op]
	if !ok {
		return OpcodeInfo{Name: "UNKNOWN", Description: "Unknown opcode"}
	}
	return info
}

// String returns the mnemonic for an opcode.
func (op Opcode) String() string {
	return op.GetInfo().Name
}

// HasOperand returns true if the opcode requires an operand.
func (op Opcode) HasOperand() bool {
	return op.GetInfo().HasOperand
}

// StackEffect returns the net effect on stack depth.
func (op Opcode) StackEffect() int {
	return op.GetInfo().StackEffect
}
