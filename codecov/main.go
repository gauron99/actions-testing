package main

import (
	"fmt"
	"main/math"
)

func main() {
	a, b := 20, 10
	fmt.Printf("> %v", math.Plus(a, b))
	fmt.Printf("> %v", math.Minus(a, b))
	fmt.Printf("> %v", math.Times(a, b))
	fmt.Printf("> %v", math.Divide(a, b))
}
