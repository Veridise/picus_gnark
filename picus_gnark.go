package picus_gnark

import (
	"fmt"
	"os"
	"regexp"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

var fInfo *os.File
var extraCnsts []string

func AddExtraConstraint(x string) {
	extraCnsts = append(extraCnsts, x)
}

func Extract(x frontend.Variable) string {
	r, _ := regexp.Compile("^\\[{([0-9]+)")
	return r.FindStringSubmatch(fmt.Sprint(x))[1]
}

func CircuitVarIn(v frontend.Variable) {
	fmt.Fprintf(fInfo, "(in %v)\n", Extract(v))
}

func CircuitVarOut(v frontend.Variable) {
	fmt.Fprintf(fInfo, "(out %v)\n", Extract(v))
}

func Label(v frontend.Variable, name string) {
	fmt.Fprintf(fInfo, "(label %v %v)\n", Extract(v), name)
}

func CompilePicus(name string, circuit frontend.Circuit) {
	fTmp, _ := os.Create(name + ".sr1cs")
	fInfo = fTmp
	defer fInfo.Close()

	r1cs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	fmt.Fprintf(fInfo, "(num-wires %v)\n", r1cs.GetNbSecretVariables()+r1cs.GetNbPublicVariables()+r1cs.GetNbInternalVariables())
	fmt.Fprintf(fInfo, "(prime-number %v)\n", r1cs.Field())

	for _, x := range extraCnsts {
		fmt.Fprintf(fInfo, "%v", x)
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
