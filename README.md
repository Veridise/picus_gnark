# gnark support for Picus

[Picus](https://github.com/Veridise/Picus), 
which is a verification tool to determine if a circuit is deterministic, 
supports gnark (for R1CS constraints).
However, it requires users to manually annotate 
some metadata to extract constraints into a format that we call `sr1cs`. 
This documentation details the constraint extraction along with the `sr1cs` format.

## Requirements

1. Go and gnark [installation](https://docs.gnark.consensys.io/HowTo/get_started) (tested with Go 1.21.5 and gnark v0.9.1) 
2. Picus [installation](https://www.github.com/veridise/picus)

## `picus_gnark` workflow

The overall workflow to run Picus on a gnark circuit consists of two parts: we first generate `sr1cs` constraints that correspond to the circuit, and then run Picus on the generated sr1cs.
There are multiple steps to generate `sr1cs` constraints.
We will use the `MyCircuit` example circuit from the [gnark tutorial](https://docs.gnark.consensys.io/HowTo/write/circuit_api) as a running example, 
to demonstrate how one could verify that `MyCircuit` is deterministic.

### Annotate input and output signals

Make a modification to the circuit's corresponding `Define` function to annotate which signals are inputs and which are output. 
This is needed because Picus checks if a circuit is deterministic and so it must know which signals are inputs/outputs.
To annotate, we will use the following functions that `picus_gnark` provides.

- The functions `picus_gnark.CircuitVarIn` and `picus_gnark.CircuitVarOut` are used to mark the signals as inputs or outputs and should be called on every input and output `frontend.Variable` signal in the circuit. 
- Optionally, the `picus_gnark.Label` function can be used to assign certain signals concrete names to help interpret the counterexamples produced by Picus as you will see later.

For example, the `MyCircuit` example circuit originally has the following definition:

``` go
type MyCircuit struct {
	X, Y frontend.Variable
}

func (circuit *MyCircuit) Define(api frontend.API) error {
	x3 := api.Mul(circuit.X, circuit.X, circuit.X)
	api.AssertIsEqual(circuit.Y, api.Add(x3, circuit.X, 5))
	return nil
}
```

After we add annotations, it would become:

``` go
type MyCircuit struct {
	X, Y frontend.Variable
}

func (circuit *MyCircuit) Define(api frontend.API) error {
	picus_gnark.CircuitVarIn(circuit.X)
	picus_gnark.CircuitVarOut(circuit.Y)
	picus_gnark.Label(circuit.X, "X")
	picus_gnark.Label(circuit.Y, "Y")
	x3 := api.Mul(circuit.X, circuit.X, circuit.X)
	api.AssertIsEqual(circuit.Y, api.Add(x3, circuit.X, 5))
	return nil
}
```

### Compiling the circuit

Call `picus_gnark.CompilePicus` on the circuit to compile it to `sr1cs` constraints. 
The first argument is the file name to generate.
The second argument is the circuit.
And the last argument is the field size.
For example, your main file `main.go` in your project could look like this:

```go
package main

import (
	"github.com/Veridise/picus_gnark"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
)

func main() {
	var circuit MyCircuit
	picus_gnark.CompilePicus("circuit", &circuit, ecc.BN254.ScalarField())
}

type MyCircuit struct {
	X, Y frontend.Variable
}

func (circuit *MyCircuit) Define(api frontend.API) error {
	picus_gnark.CircuitVarIn(circuit.X)
	picus_gnark.CircuitVarOut(circuit.Y)
	picus_gnark.Label(circuit.X, "X")
	picus_gnark.Label(circuit.Y, "Y")
	x3 := api.Mul(circuit.X, circuit.X, circuit.X)
	api.AssertIsEqual(circuit.Y, api.Add(x3, circuit.X, 5))
	return nil
}
```

Note that the you do not need to write the circuit the main file.
You can arbitrarily import it (or its components) from other files, 
like you can normally do in gnark.

### Generating `sr1cs`

Running `go run main.go` should produce the following result:

```
12:27:59 INF compiling circuit
12:27:59 INF parsed circuit inputs nbPublic=0 nbSecret=2
12:27:59 INF building constraint builder nbConstraints=3
```

along with the `circuit.sr1cs` file:

```
(prime-number 21888242871839275222246405745257275088548364400416034343698204186575808495617)
(in 1)
(out 2)
(label 1 X)
(label 2 Y)
(constraint [(1 1) ] [(1 1) ] [(1 3) ])
(constraint [(1 3) ] [(1 1) ] [(1 4) ])
(constraint [(1 0) ] [(1 2) ] [(5 0) (1 1) (1 4) ])
```

### Running Picus

This `circuit.sr1cs` can be used with Picus directly. Running `/path/to/run-picus circuit.sr1cs` produces the following result:

```
The circuit is properly constrained
Exiting Picus with the code 8
```

As another example, if we verify the following `BadCircuit`,
which asserts that the square of the output signal is equal to the input signal.

```go
package main

import (
	"github.com/Veridise/picus_gnark"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
)

func main() {
	var circuit BadCircuit
	picus_gnark.CompilePicus("circuit", &circuit, ecc.BN254.ScalarField())
}

type BadCircuit struct {
	X, Y frontend.Variable
}

func (circuit *BadCircuit) Define(api frontend.API) error {
	picus_gnark.CircuitVarIn(circuit.X)
	picus_gnark.CircuitVarOut(circuit.Y)
	picus_gnark.Label(circuit.X, "X")
	picus_gnark.Label(circuit.Y, "Y")
	api.AssertIsEqual(api.Mul(circuit.Y, circuit.Y), circuit.X)
	return nil
}
```

We would obtain an under-constrained `sr1cs` file.

```bash
$ go run main.go
<elided>
$ /path/to/run-picus circuit.sr1cs 
working directory: <elided>
The circuit is underconstrained
Counterexample:
  inputs:
    X: 1
  first possible outputs:
    Y: 1
  second possible outputs:
    Y: 21888242871839275222246405745257275088548364400416034343698204186575808495616
  first internal variables:
    3: 1
  second internal variables:
    3: 1
Exiting Picus with the code 9
```

## `sr1cs` format 

As of now, the S-expression R1CS format has the following grammar:

```
(prime-number <prime-number>)
(in <signal-number>) ...
(out <signal-number>) ...
(label <signal-number> <identifier>) ...
(constraint [(<coeff> <signal-number>) ...]
            [(<coeff> <signal-number>) ...]
            [(<coeff> <signal-number>) ...]) ...
```

- The `prime-number` clause indicates the field size.
- The `in` clauses indicate the input signals of the circuit.
- The `out` clauses indicate the output signals of the circuit, and Picus will prove or find a counterexample if the output variables are properly constrained (not under-constrained).
- The `label` clauses are optional. They provide readable names for the signals.
- The `constraint` clauses are the R1CS constraints. Each constraint has three blocks: L, R, and O. Each block consists of a list of pairs of a coefficient and a signal.

The sr1cs format is not yet finalized, and could be arbitrarily changed without preserving backward compatibility.
