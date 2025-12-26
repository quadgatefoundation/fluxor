# Fluxor Virtual Machine (FVM) Architecture

## Overview

The Fluxor Virtual Machine (FVM) is a managed execution environment inspired by JVM and .NET CLR, designed specifically for the Fluxor framework. It provides bytecode execution, runtime services, type safety, and integration with Fluxor's reactive patterns.

## Design Goals

1. **Managed Execution**: Safe, sandboxed execution of Fluxor programs
2. **Bytecode Based**: Platform-independent intermediate representation
3. **Stack-Based VM**: Simple, efficient instruction set architecture
4. **Type Safety**: Runtime type checking and verification
5. **Memory Management**: Automatic memory management with GC
6. **Integration**: Seamless integration with Fluxor's EventBus and Verticles
7. **Performance**: JIT compilation and optimization support
8. **Introspection**: Reflection and debugging capabilities

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│              (Fluxor Programs, Verticles)                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                    FVM Runtime API                           │
│  • Module Loading      • Type System                         │
│  • Method Invocation   • Reflection                          │
│  • Exception Handling  • Debugging                           │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                  Execution Engine                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Interpreter │  │ JIT Compiler│  │  Optimizer  │          │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
│         │                │                │                  │
│  ┌──────▼────────────────▼────────────────▼──────┐           │
│  │         Bytecode Execution Engine             │           │
│  │  • Stack Management                           │           │
│  │  • Instruction Dispatch                       │           │
│  │  • Method Calls                               │           │
│  └───────────────────────────────────────────────┘           │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                   Runtime Services                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Memory    │  │   Type      │  │   Module    │          │
│  │  Manager    │  │  System     │  │   Loader    │          │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          │
│         │                │                │                  │
│  ┌──────▼────────────────▼────────────────▼──────┐           │
│  │           Garbage Collector                   │           │
│  └───────────────────────────────────────────────┘           │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                Fluxor Core Services                          │
│  • EventBus   • Vertx   • Context   • Concurrency          │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Bytecode Format

**Module Structure**:
```
Module {
    Header {
        Magic: 0xFVM1        // FVM magic number
        Version: uint32       // Format version
        Flags: uint32         // Module flags
    }
    
    ConstantPool {          // String and constant literals
        Count: uint32
        Entries: [Constant]
    }
    
    TypeTable {             // Type definitions
        Count: uint32
        Types: [TypeDef]
    }
    
    MethodTable {           // Method definitions
        Count: uint32
        Methods: [MethodDef]
    }
    
    Code {                  // Bytecode instructions
        Methods: [MethodCode]
    }
    
    Metadata {              // Reflection data
        Attributes: [Attribute]
    }
}
```

**Instruction Set** (Stack-Based):
```
Arithmetic:
  ADD, SUB, MUL, DIV, MOD, NEG

Comparison:
  EQ, NE, LT, LE, GT, GE

Stack:
  PUSH <value>, POP, DUP, SWAP

Local Variables:
  LOAD <index>, STORE <index>

Control Flow:
  JMP <offset>, JZ <offset>, JNZ <offset>
  CALL <method>, RET
  
Object Operations:
  NEW <type>, GETFIELD <field>, SETFIELD <field>
  INVOKE <method>, INVOKEVIRTUAL <method>

Array Operations:
  NEWARRAY <type>, ARRAYLENGTH
  ALOAD <index>, ASTORE <index>

Type Operations:
  CAST <type>, INSTANCEOF <type>

Special:
  NOP, HALT, THROW, TRY, CATCH
```

### 2. Type System

**Base Types**:
- `void`, `bool`, `int8`, `int16`, `int32`, `int64`
- `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`, `string`
- `any` (interface{})

**Composite Types**:
- `array[T]` - Fixed-size arrays
- `slice[T]` - Dynamic arrays
- `map[K,V]` - Hash maps
- `struct` - Composite types
- `interface` - Abstract types
- `func` - Function types

**Type Safety**:
- Compile-time type checking in assembler
- Runtime type verification
- No implicit conversions
- Type casting with runtime checks

### 3. Memory Model

```
┌─────────────────────────────────────────────────────────────┐
│                        Heap                                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Object Space                            │   │
│  │  • Managed objects                                   │   │
│  │  • Reference-counted or mark-sweep GC                │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              String Intern Pool                      │   │
│  │  • Interned strings                                  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Per-Thread Stack                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Call Frames                             │   │
│  │  Frame N: [Locals] [Operand Stack] [Return Addr]    │   │
│  │  Frame N-1: ...                                      │   │
│  │  Frame 0: [Main]                                     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Garbage Collection**:
- Mark-and-sweep for initial implementation
- Reference counting for deterministic cleanup
- Generational GC for optimization
- Integration with Go's GC where possible

### 4. Execution Model

**Call Frame**:
```go
type Frame struct {
    Method      *Method
    PC          int              // Program counter
    Stack       *OperandStack    // Operand stack
    Locals      []Value          // Local variables
    ReturnAddr  int              // Return address
    Exception   *Value           // Current exception
}
```

**Execution Flow**:
```
1. Load Module → Parse → Verify → Link
2. Create Runtime Instance
3. Initialize Static Fields
4. Find Entry Point (main method)
5. Create Initial Frame
6. Execute Bytecode Loop:
   - Fetch instruction at PC
   - Decode instruction
   - Execute instruction
   - Update PC
   - Check for exceptions
7. Handle Returns and Exceptions
8. Cleanup and Exit
```

### 5. Module Loading

**Module Loader**:
```go
type ModuleLoader interface {
    Load(name string) (*Module, error)
    LoadBytes(data []byte) (*Module, error)
    Verify(module *Module) error
    Link(module *Module) error
    Initialize(module *Module) error
}
```

**Loading Phases**:
1. **Load**: Read module from disk/network
2. **Parse**: Parse bytecode format
3. **Verify**: Type checking, bytecode verification
4. **Link**: Resolve references, create runtime structures
5. **Initialize**: Run static initializers

### 6. JIT Compilation

**Tiered Compilation**:
```
Tier 0: Interpreter (immediate execution)
   ↓
Tier 1: Template JIT (hot methods, simple codegen)
   ↓
Tier 2: Optimizing JIT (very hot methods, full optimization)
```

**JIT Architecture**:
```go
type JITCompiler interface {
    Compile(method *Method) (NativeCode, error)
    ShouldCompile(method *Method) bool
    GetCompiledCode(method *Method) NativeCode
    Optimize(method *Method, profile ProfileData) (NativeCode, error)
}
```

**Optimizations**:
- Inlining
- Dead code elimination
- Constant folding
- Loop optimization
- Escape analysis

### 7. Introspection and Debugging

**Reflection API**:
```go
type Type interface {
    Name() string
    Kind() TypeKind
    Methods() []Method
    Fields() []Field
    Implements(iface Type) bool
}

type Method interface {
    Name() string
    Signature() Signature
    Invoke(receiver Value, args []Value) (Value, error)
}
```

**Debugging Support**:
- Breakpoints
- Single-step execution
- Stack traces
- Variable inspection
- Hot code reloading

### 8. Exception Handling

**Exception Model**:
```
TRY:
    <protected code>
CATCH <exceptionType>:
    <handler code>
FINALLY:
    <cleanup code>
END
```

**Exception Table**:
```go
type ExceptionHandler struct {
    StartPC   int
    EndPC     int
    HandlerPC int
    CatchType *Type
}
```

## Integration with Fluxor

### EventBus Integration

```go
// Bytecode can invoke EventBus methods
PUSH "my.address"
PUSH <data>
CALL EventBus.Send

// EventBus consumers can be bytecode methods
eventBus.Consumer("address").Handler(func(ctx, msg) {
    vm.Invoke("handler.method", ctx, msg)
})
```

### Verticle Integration

```go
// Verticles can be FVM modules
type FVMVerticle struct {
    vm     *VM
    module *Module
}

func (v *FVMVerticle) Start(ctx FluxorContext) error {
    return v.vm.InvokeMethod("start", ctx)
}
```

### Context Integration

```go
// FVM can access FluxorContext
GETCONTEXT
CALL FluxorContext.EventBus
```

## Performance Characteristics

**Interpreter**:
- Startup: < 1ms
- Execution: 10-50x slower than native
- Memory: Low overhead

**JIT Tier 1**:
- Compilation: 10-50ms per method
- Execution: 2-5x slower than native
- Memory: Medium overhead

**JIT Tier 2**:
- Compilation: 100-500ms per method
- Execution: Near-native performance
- Memory: Higher overhead

## Comparison with JVM/.NET

| Feature | JVM | .NET CLR | FVM |
|---------|-----|----------|-----|
| **Bytecode** | Java bytecode | IL | FVM bytecode |
| **VM Type** | Stack-based | Stack-based | Stack-based |
| **GC** | Generational | Generational | Mark-sweep + RC |
| **JIT** | HotSpot, GraalVM | RyuJIT | Tiered JIT |
| **Language** | Multi-language | Multi-language | Fluxor-specific |
| **Runtime** | JRE | CLR | Fluxor Runtime |
| **Type System** | Static + Dynamic | Static + Dynamic | Static + Dynamic |
| **Reflection** | Full reflection | Full reflection | Basic reflection |
| **Size** | Large (~100MB) | Medium (~50MB) | Small (~5MB) |

## Security Model

**Sandboxing**:
- Memory isolation per VM instance
- Resource limits (CPU, memory, time)
- Permission system for I/O operations
- No access to host system by default

**Verification**:
- Bytecode verification on load
- Type safety verification
- Stack depth verification
- Branch target verification

## Deployment Models

### 1. Embedded VM
```go
vm := fvm.New()
module, _ := vm.LoadModule("app.fvm")
vm.Invoke("main")
```

### 2. Standalone Runtime
```bash
fvm run app.fvm
fvm compile app.fva -o app.fvm
```

### 3. Fluxor Integration
```go
app := fluxor.NewMainVerticle("config.json")
app.DeployVerticle(fvm.NewFVMVerticle("module.fvm"))
app.Start()
```

## File Extensions

- `.fva` - FVM Assembly (human-readable)
- `.fvm` - FVM Bytecode (binary)
- `.fvd` - FVM Debug Info
- `.fvp` - FVM Package (multiple modules)

## Future Enhancements

1. **AOT Compilation**: Ahead-of-time compilation to native code
2. **WASM Backend**: Compile to WebAssembly
3. **LLVM Backend**: Use LLVM for optimization
4. **Multi-language**: Support multiple source languages
5. **Distributed VM**: VM instances across cluster
6. **GPU Support**: Offload compute to GPU
7. **Persistent Compilation**: Cache compiled code

## Summary

The FVM provides:
- ✅ Platform-independent bytecode execution
- ✅ Type-safe managed runtime
- ✅ Integration with Fluxor's reactive patterns
- ✅ JIT compilation for performance
- ✅ Memory management with GC
- ✅ Reflection and debugging support
- ✅ Security and sandboxing
- ✅ Small, focused runtime

This creates a complete execution environment for Fluxor applications, similar to how JVM serves Java and CLR serves .NET.
