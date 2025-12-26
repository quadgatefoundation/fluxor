package fvm

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// Magic number for FVM bytecode files
	FVMMagic = 0x46564D31 // "FVM1" in ASCII

	// Version
	FVMVersion = 1
)

// Module represents a loaded FVM module.
type Module struct {
	Name         string
	Version      uint32
	ConstantPool []interface{}
	Methods      []*Method
	Metadata     map[string]string
}

// Method represents a method in the module.
type Method struct {
	Name       string
	Signature  string
	MaxStack   int
	MaxLocals  int
	Code       []Instruction
	Parameters []string
	ReturnType ValueType
}

// NewModule creates a new module.
func NewModule(name string) *Module {
	return &Module{
		Name:         name,
		Version:      FVMVersion,
		ConstantPool: make([]interface{}, 0),
		Methods:      make([]*Method, 0),
		Metadata:     make(map[string]string),
	}
}

// AddMethod adds a method to the module.
func (m *Module) AddMethod(method *Method) {
	m.Methods = append(m.Methods, method)
}

// GetMethod gets a method by name.
func (m *Module) GetMethod(name string) (*Method, error) {
	for _, method := range m.Methods {
		if method.Name == name {
			return method, nil
		}
	}
	return nil, fmt.Errorf("method not found: %s", name)
}

// AddConstant adds a constant to the constant pool.
func (m *Module) AddConstant(value interface{}) int {
	index := len(m.ConstantPool)
	m.ConstantPool = append(m.ConstantPool, value)
	return index
}

// GetConstant gets a constant from the pool.
func (m *Module) GetConstant(index int) (interface{}, error) {
	if index < 0 || index >= len(m.ConstantPool) {
		return nil, fmt.Errorf("constant pool index out of bounds: %d", index)
	}
	return m.ConstantPool[index], nil
}

// WriteTo writes the module in binary format.
func (m *Module) WriteTo(w io.Writer) error {
	// Write header
	if err := binary.Write(w, binary.LittleEndian, uint32(FVMMagic)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, m.Version); err != nil {
		return err
	}

	// Write module name
	nameBytes := []byte(m.Name)
	if err := binary.Write(w, binary.LittleEndian, uint32(len(nameBytes))); err != nil {
		return err
	}
	if _, err := w.Write(nameBytes); err != nil {
		return err
	}

	// Write constant pool
	if err := binary.Write(w, binary.LittleEndian, uint32(len(m.ConstantPool))); err != nil {
		return err
	}
	for _, constant := range m.ConstantPool {
		if err := writeConstant(w, constant); err != nil {
			return err
		}
	}

	// Write methods
	if err := binary.Write(w, binary.LittleEndian, uint32(len(m.Methods))); err != nil {
		return err
	}
	for _, method := range m.Methods {
		if err := writeMethod(w, method); err != nil {
			return err
		}
	}

	return nil
}

// ReadFrom reads a module from binary format.
func ReadFrom(r io.Reader) (*Module, error) {
	// Read header
	var magic uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != FVMMagic {
		return nil, fmt.Errorf("invalid magic number: 0x%X", magic)
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, err
	}

	// Read module name
	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBytes); err != nil {
		return nil, err
	}
	moduleName := string(nameBytes)

	module := NewModule(moduleName)
	module.Version = version

	// Read constant pool
	var poolSize uint32
	if err := binary.Read(r, binary.LittleEndian, &poolSize); err != nil {
		return nil, err
	}
	for i := 0; i < int(poolSize); i++ {
		constant, err := readConstant(r)
		if err != nil {
			return nil, err
		}
		module.ConstantPool = append(module.ConstantPool, constant)
	}

	// Read methods
	var methodCount uint32
	if err := binary.Read(r, binary.LittleEndian, &methodCount); err != nil {
		return nil, err
	}
	for i := 0; i < int(methodCount); i++ {
		method, err := readMethod(r)
		if err != nil {
			return nil, err
		}
		module.Methods = append(module.Methods, method)
	}

	return module, nil
}

func writeConstant(w io.Writer, constant interface{}) error {
	switch v := constant.(type) {
	case int64:
		if err := binary.Write(w, binary.LittleEndian, byte(1)); err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, v)
	case float64:
		if err := binary.Write(w, binary.LittleEndian, byte(2)); err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, v)
	case string:
		if err := binary.Write(w, binary.LittleEndian, byte(3)); err != nil {
			return err
		}
		strBytes := []byte(v)
		if err := binary.Write(w, binary.LittleEndian, uint32(len(strBytes))); err != nil {
			return err
		}
		_, err := w.Write(strBytes)
		return err
	default:
		return fmt.Errorf("unsupported constant type: %T", constant)
	}
}

func readConstant(r io.Reader) (interface{}, error) {
	var constType byte
	if err := binary.Read(r, binary.LittleEndian, &constType); err != nil {
		return nil, err
	}

	switch constType {
	case 1: // int64
		var val int64
		if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
			return nil, err
		}
		return val, nil
	case 2: // float64
		var val float64
		if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
			return nil, err
		}
		return val, nil
	case 3: // string
		var strLen uint32
		if err := binary.Read(r, binary.LittleEndian, &strLen); err != nil {
			return nil, err
		}
		strBytes := make([]byte, strLen)
		if _, err := io.ReadFull(r, strBytes); err != nil {
			return nil, err
		}
		return string(strBytes), nil
	default:
		return nil, fmt.Errorf("unknown constant type: %d", constType)
	}
}

func writeMethod(w io.Writer, method *Method) error {
	// Write method name
	nameBytes := []byte(method.Name)
	if err := binary.Write(w, binary.LittleEndian, uint32(len(nameBytes))); err != nil {
		return err
	}
	if _, err := w.Write(nameBytes); err != nil {
		return err
	}

	// Write max stack and locals
	if err := binary.Write(w, binary.LittleEndian, uint32(method.MaxStack)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(method.MaxLocals)); err != nil {
		return err
	}

	// Write code
	if err := binary.Write(w, binary.LittleEndian, uint32(len(method.Code))); err != nil {
		return err
	}
	for _, instr := range method.Code {
		if err := binary.Write(w, binary.LittleEndian, instr.Op); err != nil {
			return err
		}
		if instr.Op.HasOperand() {
			if err := binary.Write(w, binary.LittleEndian, instr.Operand); err != nil {
				return err
			}
		}
	}

	return nil
}

func readMethod(r io.Reader) (*Method, error) {
	// Read method name
	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return nil, err
	}
	nameBytes := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBytes); err != nil {
		return nil, err
	}
	methodName := string(nameBytes)

	// Read max stack and locals
	var maxStack, maxLocals uint32
	if err := binary.Read(r, binary.LittleEndian, &maxStack); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &maxLocals); err != nil {
		return nil, err
	}

	// Read code
	var codeLen uint32
	if err := binary.Read(r, binary.LittleEndian, &codeLen); err != nil {
		return nil, err
	}

	code := make([]Instruction, codeLen)
	for i := 0; i < int(codeLen); i++ {
		var op Opcode
		if err := binary.Read(r, binary.LittleEndian, &op); err != nil {
			return nil, err
		}
		code[i].Op = op

		if op.HasOperand() {
			if err := binary.Read(r, binary.LittleEndian, &code[i].Operand); err != nil {
				return nil, err
			}
		}
	}

	return &Method{
		Name:      methodName,
		MaxStack:  int(maxStack),
		MaxLocals: int(maxLocals),
		Code:      code,
	}, nil
}
