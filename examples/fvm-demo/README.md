# FVM Demo - Fluxor Virtual Machine Examples

This demo showcases the Fluxor Virtual Machine (FVM) with practical examples.

## Overview

The FVM is a JVM/.NET CLR-inspired virtual machine that executes bytecode using a stack-based architecture. This demo includes:

1. **Basic Arithmetic** - Simple calculations
2. **Fibonacci Sequence** - Iterative algorithm with loops
3. **Factorial Calculator** - Demonstrates backward iteration
4. **Array Operations** - Create, fill, and iterate arrays
5. **Object Operations** - Create objects and manipulate fields

## Running the Demo

```bash
cd examples/fvm-demo
go run main.go
```

## Example Output

```
========================================
Fluxor Virtual Machine Demo
========================================

Demo 1: Basic Arithmetic (5 + 3 * 2)
Bytecode:
; Module: arithmetic
.method main 20 0
  0: LOADINT      5
  1: LOADINT      3
  2: ADD
  3: LOADINT      2
  4: MUL
  5: PRINT
  6: HALT
.end

[FVM] 16

Demo 2: Fibonacci Sequence
  fib( 0) = 1
  fib( 1) = 1
  fib( 2) = 2
  fib( 3) = 3
  fib( 4) = 5
  fib( 5) = 8
  fib( 6) = 13
  fib( 7) = 21
  fib( 8) = 34
  fib( 9) = 55
  fib(10) = 89
  ...

Demo 3: Factorial Calculation
   0! = 1
   1! = 1
   2! = 2
   3! = 6
   4! = 24
   5! = 120
   ...

Demo 4: Array Operations
  Creating and filling array:
  [FVM] 0
  [FVM] 10
  [FVM] 20
  [FVM] 30
  [FVM] 40

Demo 5: Object Operations
  Creating object and manipulating fields:
  [FVM] 42
  [FVM] 100
  [FVM] 142

========================================
All demos completed successfully!
========================================
```

## What Each Demo Shows

### Demo 1: Basic Arithmetic

Demonstrates:
- Stack operations
- Arithmetic instructions (ADD, MUL)
- Bytecode structure
- Disassembly output

### Demo 2: Fibonacci

Demonstrates:
- Local variables (LOAD/STORE)
- Loops with labels and jumps
- Conditional branching (JZ)
- Method parameters
- Return values

### Demo 3: Factorial

Demonstrates:
- Backward iteration
- Multiplication accumulation
- Conditional exits (JNZ)
- State management across iterations

### Demo 4: Arrays

Demonstrates:
- Array creation (NEWARRAY)
- Array indexing (ALOAD/ASTORE)
- Array length (ARRAYLEN)
- Loop iteration over arrays

### Demo 5: Objects

Demonstrates:
- Object creation (NEW)
- Field access (GETFIELD/SETFIELD)
- Object state management
- Computed values from fields

## Code Structure

Each demo follows this pattern:

```go
func demoN() {
    // 1. Create assembler
    asm := fvm.NewAssembler("module-name")
    
    // 2. Define method with bytecode
    asm.BeginMethod("method-name", maxStack, maxLocals).
        // Emit instructions
        LoadInt(42).
        Print().
        Halt().
        EndMethod()
    
    // 3. Build module
    module := asm.Build()
    
    // 4. Create and configure VM
    vm := fvm.NewVM()
    vm.LoadModule(module)
    
    // 5. Execute
    vm.Execute("module-name", "method-name")
}
```

## Assembly Language Reference

### Instructions Used

```assembly
LOADINT <value>     ; Push integer constant
STORE <index>       ; Store to local variable
LOAD <index>        ; Load from local variable
ADD, SUB, MUL, DIV  ; Arithmetic operations
LT, LE, GT, GE      ; Comparisons
JMP <label>         ; Unconditional jump
JZ <label>          ; Jump if zero/false
JNZ <label>         ; Jump if not zero/true
PRINT               ; Debug output
HALT                ; Stop execution
RETVAL              ; Return with value
```

### Control Flow

```assembly
; Labels define jump targets
loop:
    LOAD 0
    LOADINT 10
    LT              ; Compare
    JZ end          ; Exit if false
    ; ... loop body ...
    JMP loop        ; Continue
end:
    RET
```

## Comparison with Assembly

FVM bytecode is similar to assembly language:

| Concept          | x86 ASM      | FVM Bytecode |
|------------------|--------------|--------------|
| **Registers**    | eax, ebx     | Stack        |
| **Stack**        | ESP          | Operand Stack|
| **Locals**       | [ebp-4]      | local[0]     |
| **Jump**         | jmp/jz       | JMP/JZ       |
| **Call**         | call         | CALL         |
| **Return**       | ret          | RET          |

## Performance Notes

The FVM interpreter is:
- **~10-50x slower** than native Go (typical for interpreters)
- **Fast startup**: <1ms module loading
- **Low memory**: ~100 bytes per call frame

For performance-critical code, consider:
1. Native Go implementation
2. Calling Go functions from FVM (future feature)
3. JIT compilation (planned)

## Extending the Examples

Try modifying the demos:

1. **Change the algorithms**: Implement different calculations
2. **Add more loops**: Nested loops, different patterns
3. **Use more locals**: Complex state management
4. **Create larger arrays**: Test performance
5. **Build complex objects**: Multiple fields, nested objects

## Integration with Fluxor

The FVM can integrate with Fluxor's EventBus:

```go
asm.BeginMethod("sendMessage", 20, 0).
    LoadString("my.address").
    LoadInt(42).
    Emit(fvm.OpEventBusSend).  // Send to EventBus
    Ret().
    EndMethod()

vm := fvm.NewVM().WithEventBus(eventBus)
vm.LoadModule(asm.Build())
vm.Execute("module", "sendMessage")
```

## Learning Resources

- **FVM README**: `pkg/fvm/README.md`
- **Architecture**: `pkg/fvm/ARCHITECTURE.md`
- **Tests**: `pkg/fvm/vm_test.go` for more examples

## Troubleshooting

### Stack Overflow

If you get "stack overflow":
- Increase `maxStack` in `BeginMethod`
- Check for infinite recursion
- Verify jump targets are correct

### Local Variable Out of Bounds

If you get "local variable index out of bounds":
- Increase `maxLocals` in `BeginMethod`
- Check LOAD/STORE indices

### Unknown Instruction

If you get "unknown opcode":
- Verify the instruction is valid
- Check for typos in assembly

## Next Steps

1. **Read the Architecture**: Understand the VM internals
2. **Write Your Own Programs**: Create custom bytecode
3. **Explore Integration**: Connect with EventBus and Verticles
4. **Build a Compiler**: Create a higher-level language that compiles to FVM bytecode

## Summary

This demo shows that FVM provides:
- ✅ Complete bytecode execution
- ✅ Practical algorithms (Fibonacci, Factorial)
- ✅ Data structures (Arrays, Objects)
- ✅ Control flow (Loops, Conditionals)
- ✅ Easy-to-use assembly language
- ✅ Clear debugging (Disassembly)

The FVM is ready for use in Fluxor applications!
