package picus_gnark

import (
	"fmt"
	"math/big"
	"os"
	"regexp"

	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

var extraCnsts []string
var varIns []string
var varOuts []string
var labels [][2]string

func AddExtraConstraint(x string) {
	extraCnsts = append(extraCnsts, x)
}

func Extract(x frontend.Variable) string {
	r, _ := regexp.Compile(`^\[{([0-9]+)`)
	return r.FindStringSubmatch(fmt.Sprint(x))[1]
}

func CircuitVarIn(v frontend.Variable) {
	varIns = append(varIns, Extract(v))
}

func CircuitVarOut(v frontend.Variable) {
	varOuts = append(varOuts, Extract(v))
}

func Label(v frontend.Variable, name string) {
	labels = append(labels, [2]string{Extract(v), name})
}

func CompilePicus(name string, circuit frontend.Circuit, field *big.Int) {
	extraCnsts = []string{}
	varIns = []string{}
	varOuts = []string{}
	labels = [][2]string{}

	fInfo, _ := os.Create(name + ".sr1cs")
	defer fInfo.Close()

	r1cs, _ := frontend.Compile(field, r1cs.NewBuilder, circuit)
	fmt.Fprintf(fInfo, "(prime-number %v)\n", r1cs.Field())

	for _, x := range varIns {
		fmt.Fprintf(fInfo, "(in %v)\n", x)
	}

	for _, x := range varOuts {
		fmt.Fprintf(fInfo, "(out %v)\n", x)
	}

	for _, x := range labels {
		fmt.Fprintf(fInfo, "(label %v %v)\n", x[0], x[1])
	}

	for _, x := range extraCnsts {
		fmt.Fprintf(fInfo, "(extra-constraint %v)\n", x)
	}

	nR1CS, ok := r1cs.(constraint.R1CS)
	if ok {
		constraints := nR1CS.GetR1Cs()
		for _, r1c := range constraints {
			fmt.Fprintf(fInfo, "(constraint ")
			fmt.Fprintf(fInfo, "[")

			for i := 0; i < len(r1c.L); i++ {
				fmt.Fprintf(fInfo, "(%v %v) ", r1cs.CoeffToString(int(r1c.L[i].CID)), r1c.L[i].VID)
			}
			fmt.Fprintf(fInfo, "] [")
			for i := 0; i < len(r1c.R); i++ {
				fmt.Fprintf(fInfo, "(%v %v) ", r1cs.CoeffToString(int(r1c.R[i].CID)), r1c.R[i].VID)
			}
			fmt.Fprintf(fInfo, "] [")
			for i := 0; i < len(r1c.O); i++ {
				fmt.Fprintf(fInfo, "(%v %v) ", r1cs.CoeffToString(int(r1c.O[i].CID)), r1c.O[i].VID)
			}
			fmt.Fprintf(fInfo, "])\n")
		}
	}
}
