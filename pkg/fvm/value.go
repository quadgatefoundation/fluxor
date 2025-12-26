package fvm

import (
	"fmt"
	"reflect"
)

// ValueType represents the type of a value.
type ValueType int

const (
	TypeVoid ValueType = iota
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeObject
	TypeArray
	TypeFunction
	TypeNull
)

func (t ValueType) String() string {
	switch t {
	case TypeVoid:
		return "void"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeObject:
		return "object"
	case TypeArray:
		return "array"
	case TypeFunction:
		return "function"
	case TypeNull:
		return "null"
	default:
		return "unknown"
	}
}

// Value represents a runtime value in the VM.
type Value struct {
	Type ValueType
	Data interface{}
}

// NewVoidValue creates a void value.
func NewVoidValue() *Value {
	return &Value{Type: TypeVoid, Data: nil}
}

// NewBoolValue creates a boolean value.
func NewBoolValue(b bool) *Value {
	return &Value{Type: TypeBool, Data: b}
}

// NewIntValue creates an integer value.
func NewIntValue(i int64) *Value {
	return &Value{Type: TypeInt, Data: i}
}

// NewFloatValue creates a float value.
func NewFloatValue(f float64) *Value {
	return &Value{Type: TypeFloat, Data: f}
}

// NewStringValue creates a string value.
func NewStringValue(s string) *Value {
	return &Value{Type: TypeString, Data: s}
}

// NewObjectValue creates an object value.
func NewObjectValue(obj *Object) *Value {
	return &Value{Type: TypeObject, Data: obj}
}

// NewArrayValue creates an array value.
func NewArrayValue(arr *Array) *Value {
	return &Value{Type: TypeArray, Data: arr}
}

// NewNullValue creates a null value.
func NewNullValue() *Value {
	return &Value{Type: TypeNull, Data: nil}
}

// AsBool returns the value as a boolean.
func (v *Value) AsBool() (bool, error) {
	if v.Type != TypeBool {
		return false, fmt.Errorf("value is not boolean (got %s)", v.Type)
	}
	return v.Data.(bool), nil
}

// AsInt returns the value as an integer.
func (v *Value) AsInt() (int64, error) {
	if v.Type != TypeInt {
		return 0, fmt.Errorf("value is not integer (got %s)", v.Type)
	}
	return v.Data.(int64), nil
}

// AsFloat returns the value as a float.
func (v *Value) AsFloat() (float64, error) {
	if v.Type != TypeFloat {
		return 0, fmt.Errorf("value is not float (got %s)", v.Type)
	}
	return v.Data.(float64), nil
}

// AsString returns the value as a string.
func (v *Value) AsString() (string, error) {
	if v.Type != TypeString {
		return "", fmt.Errorf("value is not string (got %s)", v.Type)
	}
	return v.Data.(string), nil
}

// AsObject returns the value as an object.
func (v *Value) AsObject() (*Object, error) {
	if v.Type != TypeObject {
		return nil, fmt.Errorf("value is not object (got %s)", v.Type)
	}
	return v.Data.(*Object), nil
}

// AsArray returns the value as an array.
func (v *Value) AsArray() (*Array, error) {
	if v.Type != TypeArray {
		return nil, fmt.Errorf("value is not array (got %s)", v.Type)
	}
	return v.Data.(*Array), nil
}

// IsTruthy returns true if the value is considered true.
func (v *Value) IsTruthy() bool {
	switch v.Type {
	case TypeBool:
		return v.Data.(bool)
	case TypeInt:
		return v.Data.(int64) != 0
	case TypeFloat:
		return v.Data.(float64) != 0.0
	case TypeString:
		return v.Data.(string) != ""
	case TypeNull:
		return false
	default:
		return v.Data != nil
	}
}

// IsNull returns true if the value is null.
func (v *Value) IsNull() bool {
	return v.Type == TypeNull
}

// Equals checks if two values are equal.
func (v *Value) Equals(other *Value) bool {
	if v.Type != other.Type {
		return false
	}
	return reflect.DeepEqual(v.Data, other.Data)
}

// String returns a string representation of the value.
func (v *Value) String() string {
	switch v.Type {
	case TypeVoid:
		return "void"
	case TypeNull:
		return "null"
	case TypeBool:
		return fmt.Sprintf("%t", v.Data)
	case TypeInt:
		return fmt.Sprintf("%d", v.Data)
	case TypeFloat:
		return fmt.Sprintf("%f", v.Data)
	case TypeString:
		return fmt.Sprintf("%q", v.Data)
	case TypeObject:
		return fmt.Sprintf("object<%p>", v.Data)
	case TypeArray:
		arr := v.Data.(*Array)
		return fmt.Sprintf("array[%d]", len(arr.Elements))
	default:
		return fmt.Sprintf("<%s>", v.Type)
	}
}

// Clone creates a deep copy of the value.
func (v *Value) Clone() *Value {
	switch v.Type {
	case TypeObject:
		obj := v.Data.(*Object)
		return NewObjectValue(obj.Clone())
	case TypeArray:
		arr := v.Data.(*Array)
		return NewArrayValue(arr.Clone())
	default:
		// Primitives can be copied directly
		return &Value{
			Type: v.Type,
			Data: v.Data,
		}
	}
}

// Object represents a runtime object.
type Object struct {
	TypeName string
	Fields   map[string]*Value
}

// NewObject creates a new object.
func NewObject(typeName string) *Object {
	return &Object{
		TypeName: typeName,
		Fields:   make(map[string]*Value),
	}
}

// GetField gets a field value.
func (o *Object) GetField(name string) (*Value, error) {
	val, ok := o.Fields[name]
	if !ok {
		return nil, fmt.Errorf("field '%s' not found", name)
	}
	return val, nil
}

// SetField sets a field value.
func (o *Object) SetField(name string, value *Value) {
	o.Fields[name] = value
}

// Clone creates a deep copy of the object.
func (o *Object) Clone() *Object {
	clone := NewObject(o.TypeName)
	for k, v := range o.Fields {
		clone.Fields[k] = v.Clone()
	}
	return clone
}

// Array represents a runtime array.
type Array struct {
	ElementType ValueType
	Elements    []*Value
}

// NewArray creates a new array.
func NewArray(elementType ValueType, size int) *Array {
	return &Array{
		ElementType: elementType,
		Elements:    make([]*Value, size),
	}
}

// Get returns an element at index.
func (a *Array) Get(index int) (*Value, error) {
	if index < 0 || index >= len(a.Elements) {
		return nil, fmt.Errorf("array index out of bounds: %d", index)
	}
	return a.Elements[index], nil
}

// Set sets an element at index.
func (a *Array) Set(index int, value *Value) error {
	if index < 0 || index >= len(a.Elements) {
		return fmt.Errorf("array index out of bounds: %d", index)
	}
	a.Elements[index] = value
	return nil
}

// Len returns the array length.
func (a *Array) Len() int {
	return len(a.Elements)
}

// Clone creates a deep copy of the array.
func (a *Array) Clone() *Array {
	clone := NewArray(a.ElementType, len(a.Elements))
	for i, v := range a.Elements {
		if v != nil {
			clone.Elements[i] = v.Clone()
		}
	}
	return clone
}
