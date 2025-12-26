package fvm

import (
	"fmt"
	"strconv"
	"strings"
)

// Assembler assembles textual assembly code into bytecode.
type Assembler struct {
	module  *Module
	current *Method
	labels  map[string]int
}

// NewAssembler creates a new assembler.
func NewAssembler(moduleName string) *Assembler {
	return &Assembler{
		module: NewModule(moduleName),
		labels: make(map[string]int),
	}
}

// BeginMethod starts defining a new method.
func (a *Assembler) BeginMethod(name string, maxStack, maxLocals int) *Assembler {
	a.current = &Method{
		Name:      name,
		MaxStack:  maxStack,
		MaxLocals: maxLocals,
		Code:      make([]Instruction, 0),
	}
	a.labels = make(map[string]int)
	return a
}

// EndMethod finishes the current method and adds it to the module.
func (a *Assembler) EndMethod() *Assembler {
	if a.current != nil {
		a.module.AddMethod(a.current)
		a.current = nil
	}
	return a
}

// Label defines a label at the current position.
func (a *Assembler) Label(name string) *Assembler {
	if a.current == nil {
		return a
	}
	a.labels[name] = len(a.current.Code)
	return a
}

// Emit emits an instruction.
func (a *Assembler) Emit(op Opcode, operand ...int64) *Assembler {
	if a.current == nil {
		return a
	}

	instr := Instruction{Op: op}
	if len(operand) > 0 {
		instr.Operand = operand[0]
	}

	a.current.Code = append(a.current.Code, instr)
	return a
}

// Convenience methods for common instructions

func (a *Assembler) Add() *Assembler      { return a.Emit(OpAdd) }
func (a *Assembler) Sub() *Assembler      { return a.Emit(OpSub) }
func (a *Assembler) Mul() *Assembler      { return a.Emit(OpMul) }
func (a *Assembler) Div() *Assembler      { return a.Emit(OpDiv) }
func (a *Assembler) Mod() *Assembler      { return a.Emit(OpMod) }
func (a *Assembler) Neg() *Assembler      { return a.Emit(OpNeg) }

func (a *Assembler) Eq() *Assembler       { return a.Emit(OpEq) }
func (a *Assembler) Ne() *Assembler       { return a.Emit(OpNe) }
func (a *Assembler) Lt() *Assembler       { return a.Emit(OpLt) }
func (a *Assembler) Le() *Assembler       { return a.Emit(OpLe) }
func (a *Assembler) Gt() *Assembler       { return a.Emit(OpGt) }
func (a *Assembler) Ge() *Assembler       { return a.Emit(OpGe) }

func (a *Assembler) And() *Assembler      { return a.Emit(OpAnd) }
func (a *Assembler) Or() *Assembler       { return a.Emit(OpOr) }
func (a *Assembler) Not() *Assembler      { return a.Emit(OpNot) }

func (a *Assembler) Push(value int64) *Assembler { return a.Emit(OpPush, value) }
func (a *Assembler) Pop() *Assembler             { return a.Emit(OpPop) }
func (a *Assembler) Dup() *Assembler             { return a.Emit(OpDup) }
func (a *Assembler) Swap() *Assembler            { return a.Emit(OpSwap) }

func (a *Assembler) Load(index int) *Assembler   { return a.Emit(OpLoad, int64(index)) }
func (a *Assembler) Store(index int) *Assembler  { return a.Emit(OpStore, int64(index)) }

func (a *Assembler) LoadInt(value int64) *Assembler    { return a.Emit(OpLoadInt, value) }
func (a *Assembler) LoadBool(value bool) *Assembler {
	if value {
		return a.Emit(OpLoadBool, 1)
	}
	return a.Emit(OpLoadBool, 0)
}
func (a *Assembler) LoadNull() *Assembler              { return a.Emit(OpLoadNull) }

func (a *Assembler) Jmp(label string) *Assembler       { return a.emitJump(OpJmp, label) }
func (a *Assembler) Jz(label string) *Assembler        { return a.emitJump(OpJz, label) }
func (a *Assembler) Jnz(label string) *Assembler       { return a.emitJump(OpJnz, label) }

func (a *Assembler) Ret() *Assembler                   { return a.Emit(OpRet) }
func (a *Assembler) RetVal() *Assembler                { return a.Emit(OpRetVal) }
func (a *Assembler) Halt() *Assembler                  { return a.Emit(OpHalt) }
func (a *Assembler) Print() *Assembler                 { return a.Emit(OpPrint) }

func (a *Assembler) emitJump(op Opcode, label string) *Assembler {
	if a.current == nil {
		return a
	}

	// Check if label is already defined (backward jump)
	if target, ok := a.labels[label]; ok {
		return a.Emit(op, int64(target))
	}

	// Forward reference - emit with placeholder
	// Note: This simple implementation doesn't handle forward refs perfectly
	// For production, we'd need a two-pass assembler
	return a.Emit(op, 0)
}

// Build builds the module.
func (a *Assembler) Build() *Module {
	return a.module
}

// ParseAssembly parses assembly code and returns a module.
func ParseAssembly(source string) (*Module, error) {
	lines := strings.Split(source, "\n")
	asm := NewAssembler("parsed")
	
	var inMethod bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse method directive
		if strings.HasPrefix(line, ".method") {
			parts := strings.Fields(line)
			if len(parts) < 4 {
				return nil, fmt.Errorf("invalid .method directive: %s", line)
			}
			name := parts[1]
			maxStack, _ := strconv.Atoi(parts[2])
			maxLocals, _ := strconv.Atoi(parts[3])
			asm.BeginMethod(name, maxStack, maxLocals)
			inMethod = true
			continue
		}
		
		// Parse end method
		if strings.HasPrefix(line, ".end") {
			asm.EndMethod()
			inMethod = false
			continue
		}
		
		if !inMethod {
			continue
		}
		
		// Parse label
		if strings.HasSuffix(line, ":") {
			label := strings.TrimSuffix(line, ":")
			asm.Label(label)
			continue
		}
		
		// Parse instruction
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		
		opname := strings.ToUpper(parts[0])
		
		// Match opcode
		var foundOp Opcode
		var found bool
		for op, info := range OpcodeTable {
			if info.Name == opname {
				foundOp = op
				found = true
				break
			}
		}
		
		if !found {
			return nil, fmt.Errorf("unknown instruction: %s", opname)
		}
		
		// Parse operand if needed
		if foundOp.HasOperand() && len(parts) > 1 {
			operand, err := strconv.ParseInt(parts[1], 0, 64)
			if err != nil {
				// Try as label for jumps
				if foundOp == OpJmp || foundOp == OpJz || foundOp == OpJnz {
					asm.emitJump(foundOp, parts[1])
				} else {
					return nil, fmt.Errorf("invalid operand: %s", parts[1])
				}
			} else {
				asm.Emit(foundOp, operand)
			}
		} else {
			asm.Emit(foundOp)
		}
	}
	
	return asm.Build(), nil
}

// Disassemble disassembles a method into assembly code.
func Disassemble(method *Method) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf(".method %s %d %d\n", method.Name, method.MaxStack, method.MaxLocals))
	
	for i, instr := range method.Code {
		sb.WriteString(fmt.Sprintf("  %3d: %-12s", i, instr.Op.String()))
		if instr.Op.HasOperand() {
			sb.WriteString(fmt.Sprintf(" %d", instr.Operand))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString(".end\n")
	
	return sb.String()
}

// DisassembleModule disassembles an entire module.
func DisassembleModule(module *Module) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("; Module: %s\n", module.Name))
	sb.WriteString(fmt.Sprintf("; Version: %d\n\n", module.Version))
	
	for _, method := range module.Methods {
		sb.WriteString(Disassemble(method))
		sb.WriteString("\n")
	}
	
	return sb.String()
}
