# gnark support for Picus

[Picus](https://github.com/Veridise/Picus) supports gnark, but it requires users to manually annotate 
some metadata to extract constraints into a format that we call `sr1cs`. 
This documentation details the constraint extraction along with the `sr1cs` format.

## Step-by-step instructions

There are two steps to extract constraints into the `sr1cs` format:

### Step 1: entrypoint file

Create an entry point, say, `picus.go` with the following content:

```go
package main

import (
	"github.com/Veridise/picus_gnark"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
)

func main() {
	var circuit <PUT-THE-CIRCUIT-TYPE-HERE>

	picus_gnark.CompilePicus("circuit", &circuit, ecc.BN254.ScalarField())
}

// circuit-specific details start here

type <PUT-THE-CIRCUIT-TYPE-HERE> struct {
	...
}

func ... Define(api frontend.API) error {
	...
}
```

where `<PUT-THE-CIRCUIT-TYPE-HERE>` should be replaced with the circuit type that we wish to verify, and the section after 
`// circuit-specific details start here` should be replaced with the circuit implementation.

### Step 2: annotate inputs and outputs

Use the function `picus_gnark.CircuitVarIn` or `picus_gnark.CircuitVarOut` with 
a `frontend.Variable` to annotate that the variable should be treated as an input/output of the circuit.
These functions can be called multiple times.
Optionally, use the function `picus_gnark.Label` to annotate a variable name.

## Examples

Let's say that we want to verify that the following `MyCircuit` circuit from the [gnark tutorial](https://docs.gnark.consensys.io/HowTo/write/circuit_api)
is properly constrained.

```go
type MyCircuit struct {
	X, Y frontend.Variable
}

func (circuit *MyCircuit) Define(api frontend.API) error {
	x3 := api.Mul(circuit.X, circuit.X, circuit.X)
	api.AssertIsEqual(circuit.Y, api.Add(x3, circuit.X, 5))
	return nil
}
```

We create the entry point file, replace `<PUT-THE-CIRCUIT-TYPE-HERE>` with `MyCircuit`, 
and put the above circuit implementation at the end.

Next, we annotate the input and output variables by making the following modification:


```go
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

The full content of `picus.go` is now as follows:

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

Running `go run picus.go` should produce the following result:

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

This `circuit.sr1cs` can be used with Picus directly. Running `/path/to/run-picus circuit.sr1cs` produces the following result:

```
The circuit is properly constrained
Exiting Picus with the code 8
```

On the other hand, if we verify the following circuit instead:

```go
func (circuit *MyCircuit) Define(api frontend.API) error {
	picus_gnark.CircuitVarIn(circuit.X)
	picus_gnark.CircuitVarOut(circuit.Y)
	picus_gnark.Label(circuit.X, "X")
	picus_gnark.Label(circuit.Y, "Y")
	api.AssertIsEqual(api.Mul(circuit.Y, circuit.Y), circuit.X)
	return nil
}
```

We would obtain an `sr1cs` file that is under-constrained.

```bash
$ go run picus.go
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

## sr1cs format 

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
