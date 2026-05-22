// Adversarial fixture: IIFE allowed (CallExpr with FuncLit).
package main

func main() {
	func() { println("ok") }()
}

// Output:
// ok
