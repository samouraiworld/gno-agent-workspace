// Adversarial fixture for PR #5588.
// Drop into gnovm/tests/files/ to verify gap.
package main

func main() {
	x := 5
	int64(x) // Go: rejected ("int64(x) (value of type int64) is not used")
	         // PR: PASSES the n.X.(*CallExpr) check because type conversion
	         //     stays as *CallExpr in the AST. Real bug.
	println("ok")
}

// Output:
// ok

// TypeCheckError:
// main/expr_stmt_typeconv.gno:5:2: int64(x) (value of type int64) is not used
