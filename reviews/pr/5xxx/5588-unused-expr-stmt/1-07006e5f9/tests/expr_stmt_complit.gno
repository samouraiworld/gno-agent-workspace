// Adversarial fixture: composite literal as standalone statement.
// PR catches this correctly (CompositeLit, not CallExpr).
package main

type T struct {
	x int
}

func main() {
	T{x: 1}
}

// Error:
// main/expr_stmt_complit.gno:10:2-9: invalid operation: T<VPBlock(2,0)>{x: (const (1 int))} is not used

// TypeCheckError:
// main/expr_stmt_complit.gno:10:2: T{…} (value of struct type T) is not used
