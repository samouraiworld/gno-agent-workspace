# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: REPLY on https://github.com/gnolang/gno/pull/6058#pullrequestreview-6039428906, the review already posted at this head
Verdict: APPROVE, unchanged. Seven case shapes run the same as Go here and no filetest pins them; the design note is for later work.
Model: claude-opus-5-5, high, no review round: a differential run of 21 programs under Go 1.25.9, gno at a40c1fbf7, gno at its base and gno at a40c1fbf7 merged with #6060
Commit: a40c1fbf7 (latest)
Overview: [overview](../overview.md)

## Body

- Missing test: these shapes run the same as Go at this head, and no filetest pins them:
  - a shadow that `goto` jumps back over
  - an `else if` under an `if` init, with a closure made before the shadow
  - a pointer to the outer name taken before the shadow
  - a `defer` reading the outer name
  - `v, w := v+1, 3`, where the right-hand `v` is the outer one
  - a shadowed second init name
  - a switch reused across loop iterations

  <details><summary>test cases</summary>

  ```go
  package main

  func gotoBack() {
  	switch v := 1; v {
  	case 1:
  		n := 0
  	L:
  		println("L", v)
  		n++
  		v := 5 + n
  		println("after", v)
  		if n < 2 {
  			goto L
  		}
  	}
  }

  func elseIf() {
  	if v := 1; v > 5 {
  	} else if w := v + 1; v < 5 {
  		f := func() (int, int) { return v, w }
  		v := w * 10
  		w := v + 1
  		a, b := f()
  		println(v, w, a, b)
  	}
  }

  func pointerOuter() {
  	switch v := 1; v {
  	case 1:
  		v = 5
  		p := &v
  		v := 9
  		*p = 6
  		println("inner", v)
  		fallthrough
  	case 2:
  		println("outer", v)
  	}
  }

  func deferCapture() (r int) {
  	switch v := 1; v {
  	case 1:
  		defer func() { r = v }()
  		v := 50
  		_ = v
  	}
  	return 0
  }

  func multiDefine() {
  	switch v := 1; v {
  	case 1:
  		v, w := v+1, 3
  		println("inner", v, w)
  		fallthrough
  	case 2:
  		println("outer", v)
  	}
  }

  func twoInitNames() {
  	switch a, b := 1, 2; {
  	case a < b:
  		b := "x"
  		println(a, b)
  		fallthrough
  	default:
  		println(a, b)
  	}
  }

  func reuseInLoop() {
  	for i := 0; i < 3; i++ {
  		switch v := i; v {
  		case 0:
  			v := 10
  			println("c0", v)
  			fallthrough
  		case 1:
  			println("c1", v)
  		default:
  			v := "s"
  			println("d", v)
  		}
  	}
  }

  func main() {
  	gotoBack()
  	elseIf()
  	pointerOuter()
  	println("defer", deferCapture())
  	multiDefine()
  	twoInitNames()
  	reuseInLoop()
  }

  // Output:
  // L 1
  // after 6
  // L 1
  // after 7
  // 20 21 1 2
  // inner 9
  // outer 6
  // defer 1
  // inner 2 3
  // outer 1
  // 1 x
  // 1 2
  // c0 10
  // c1 0
  // c1 1
  // d s
  ```

  The `// Output:` block is what `go run` prints, and `gno run` at this head prints the same lines. At the base, five of the seven print differently; the `else if` and `defer` shapes already match there.

  </details>

- Missing test: a case declaring `type v int` over the init `v` panics here through #5662, and prints `before 1`, `x 7`, `outer 1` as Go does once #6060 is merged in, so whichever of the two lands second can carry it.
- I think a case should later hold only the names it declares itself, in slots that start where the `switch`'s names end, rather than copies of the `switch`'s names. No name would then appear twice in a case, so no future lookup by name could land on the copy instead of the shadow. That refactor is for later, not for this PR, and #ISSUE proposes it.
