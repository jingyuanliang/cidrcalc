package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jingyuanliang/cidrcalc/pkg/cidrcalc"
	"github.com/jingyuanliang/cidrcalc/pkg/version"
)


func main() {
	fmt.Printf("version: %s\n", version.Version)
	operands := []*cidrcalc.IPRanges{}
	cidrs := []string{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch line := scanner.Text(); line {
		case "commit":
			operand, err := cidrcalc.FromCIDRs(cidrs)
			if err != nil {
				fmt.Println(err)
				return
			}
			operands = append(operands, operand)
			cidrs = []string{}
		case "add":
			op1 := operands[len(operands)-2]
			op2 := operands[len(operands)-1]
			operands = operands[:len(operands)-1]
			operands[len(operands)-1] = op1.Add(op2)
		case "subtract":
			op1 := operands[len(operands)-2]
			op2 := operands[len(operands)-1]
			operands = operands[:len(operands)-1]
			operands[len(operands)-1] = op1.Subtract(op2)
		case "simplify":
			op := operands[len(operands)-1]
			operands[len(operands)-1] = op.Simplify()
		default:
			cidrs = append(cidrs, line)
		}
	}

	if len(cidrs) != 0 {
		fmt.Printf("%d stray CIDRs.\n", len(cidrs))
		for _, cidr := range cidrs {
			fmt.Println(cidr)
		}
	}

	fmt.Printf("%d ranges in stack.\n", len(operands))
	for _, operand := range operands {
		fmt.Println()
		for _, cidr := range operand.CIDRs() {
			fmt.Println(cidr)
		}
	}
}
