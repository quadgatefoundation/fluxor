package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fluxorio/fluxor/pkg/fvm"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("Fluxor Virtual Machine Demo")
	fmt.Println("========================================\n")

	// Demo 1: Basic arithmetic
	fmt.Println("Demo 1: Basic Arithmetic (5 + 3 * 2)")
	demo1()

	// Demo 2: Fibonacci
	fmt.Println("\nDemo 2: Fibonacci Sequence")
	demo2()

	// Demo 3: Factorial
	fmt.Println("\nDemo 3: Factorial Calculation")
	demo3()

	// Demo 4: Array operations
	fmt.Println("\nDemo 4: Array Operations")
	demo4()

	// Demo 5: Object operations
	fmt.Println("\nDemo 5: Object Operations")
	demo5()

	fmt.Println("\n========================================")
	fmt.Println("All demos completed successfully!")
	fmt.Println("========================================")
}

func demo1() {
	// Build a program: (5 + 3) * 2
	asm := fvm.NewAssembler("arithmetic")
	asm.BeginMethod("main", 20, 0).
		LoadInt(5).
		LoadInt(3).
		Add().
		LoadInt(2).
		Mul().
		Print().
		Halt().
		EndMethod()

	module := asm.Build()

	// Execute
	vm := fvm.NewVM().WithContext(context.Background())
	if err := vm.LoadModule(module); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Bytecode:")
	fmt.Println(fvm.DisassembleModule(module))

	if _, err := vm.Execute("arithmetic", "main"); err != nil {
		log.Fatal(err)
	}
}

func demo2() {
	// Fibonacci calculator
	asm := fvm.NewAssembler("fibonacci")
	asm.BeginMethod("fib", 20, 5).
		// Calculate Fibonacci(n) where n is in local[0]
		// local[1] = a, local[2] = b, local[3] = temp, local[4] = i
		
		LoadInt(0).Store(1).  // a = 0
		LoadInt(1).Store(2).  // b = 1
		LoadInt(0).Store(4).  // i = 0
		
		Label("loop").
		Load(4).Load(0).Lt().  // i < n?
		Jz("end").
		
		Load(1).Load(2).Add().Store(3).  // temp = a + b
		Load(2).Store(1).                // a = b
		Load(3).Store(2).                // b = temp
		Load(4).LoadInt(1).Add().Store(4).  // i++
		
		Jmp("loop").
		Label("end").
		Load(2).   // return b
		RetVal().
		EndMethod()

	module := asm.Build()

	vm := fvm.NewVM()
	vm.LoadModule(module)

	// Calculate Fibonacci numbers
	for n := int64(0); n <= 15; n++ {
		result, err := vm.InvokeMethod(module.Methods[0], fvm.NewIntValue(n))
		if err != nil {
			log.Fatal(err)
		}
		fib, _ := result.AsInt()
		fmt.Printf("  fib(%2d) = %d\n", n, fib)
	}
}

func demo3() {
	// Factorial calculator: n! = n * (n-1) * ... * 1
	asm := fvm.NewAssembler("factorial")
	asm.BeginMethod("fact", 20, 3).
		// local[0] = n (input)
		// local[1] = result = 1
		// local[2] = i
		
		LoadInt(1).Store(1).  // result = 1
		Load(0).Store(2).     // i = n
		
		Label("loop").
		Load(2).
		LoadInt(1).
		Lt().                 // i < 1?
		Jnz("end").           // if true, exit
		
		Load(1).Load(2).Mul().Store(1).  // result *= i
		Load(2).LoadInt(1).Sub().Store(2).  // i--
		
		Jmp("loop").
		Label("end").
		Load(1).              // return result
		RetVal().
		EndMethod()

	module := asm.Build()

	vm := fvm.NewVM()
	vm.LoadModule(module)

	// Calculate factorials
	for n := int64(0); n <= 10; n++ {
		result, err := vm.InvokeMethod(module.Methods[0], fvm.NewIntValue(n))
		if err != nil {
			log.Fatal(err)
		}
		fact, _ := result.AsInt()
		fmt.Printf("  %2d! = %d\n", n, fact)
	}
}

func demo4() {
	// Array operations
	asm := fvm.NewAssembler("arrays")
	asm.BeginMethod("main", 20, 2).
		// Create array of size 5
		Emit(fvm.OpNewArray, 5).
		Store(0).
		
		// Fill array with values
		LoadInt(0).Store(1).  // i = 0
		
		Label("fillLoop").
		Load(1).LoadInt(5).Ge().
		Jnz("fillEnd").
		
		Load(0).              // array
		Load(1).              // index
		Load(1).              // value = index
		LoadInt(10).
		Mul().                // value = index * 10
		Emit(fvm.OpArrayStore).
		
		Load(1).LoadInt(1).Add().Store(1).  // i++
		Jmp("fillLoop").
		
		Label("fillEnd").
		
		// Print array elements
		LoadInt(0).Store(1).  // i = 0
		
		Label("printLoop").
		Load(1).LoadInt(5).Ge().
		Jnz("printEnd").
		
		Load(0).              // array
		Load(1).              // index
		Emit(fvm.OpArrayLoad).
		Print().
		
		Load(1).LoadInt(1).Add().Store(1).  // i++
		Jmp("printLoop").
		
		Label("printEnd").
		Halt().
		EndMethod()

	module := asm.Build()

	vm := fvm.NewVM()
	vm.LoadModule(module)

	fmt.Println("  Creating and filling array:")
	if _, err := vm.Execute("arrays", "main"); err != nil {
		log.Fatal(err)
	}
}

func demo5() {
	// Object operations
	asm := fvm.NewAssembler("objects")
	asm.BeginMethod("main", 20, 1).
		// Create object
		Emit(fvm.OpNew, 1).
		Store(0).
		
		// Set field 0 to 42
		Load(0).
		LoadInt(42).
		Emit(fvm.OpSetField, 0).
		
		// Set field 1 to 100
		Load(0).
		LoadInt(100).
		Emit(fvm.OpSetField, 1).
		
		// Get and print field 0
		Load(0).
		Emit(fvm.OpGetField, 0).
		Print().
		
		// Get and print field 1
		Load(0).
		Emit(fvm.OpGetField, 1).
		Print().
		
		// Compute field0 + field1
		Load(0).Emit(fvm.OpGetField, 0).
		Load(0).Emit(fvm.OpGetField, 1).
		Add().
		Print().
		
		Halt().
		EndMethod()

	module := asm.Build()

	vm := fvm.NewVM()
	vm.LoadModule(module)

	fmt.Println("  Creating object and manipulating fields:")
	if _, err := vm.Execute("objects", "main"); err != nil {
		log.Fatal(err)
	}
}
