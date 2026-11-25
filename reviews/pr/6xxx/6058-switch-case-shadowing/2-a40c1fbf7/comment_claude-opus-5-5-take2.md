# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: REPLY on https://github.com/gnolang/gno/pull/6058#pullrequestreview-6039428906, the review already posted at this head
Verdict: NEEDS DISCUSSION. The fix runs the same as Go on every case tried, and an offset for the case's own names would replace the duplicate slot.
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

  The `// Output:` block is what `go run` prints, and `gno run` at this head prints the same lines. At the base, five of the seven print differently or panic; the `else if` and `defer` shapes already match there.

  </details>

- Missing test: a case declaring `type v int` over the init `v` panics here through #5662, and prints `before 1`, `x 7`, `outer 1` as Go does once #6060 is merged in, so whichever of the two lands second can carry it.
- The fix you made looks good to me, but I think we should stop copying the `switch`'s names into the case's static block and give the case only its own names, at an offset equal to the `switch`'s `NumNames`, so slot = offset + position in the case. The runtime block stays the same, and `b` appears once per static block instead of twice:

  ```go
  switch a, b := 1, 2; a {
  case 1:
  	c := 3
  	b := "x"
  }
  ```

  ```
  runtime block:        [a][b][c][b']     slots 0..3, the same both ways
  switch static block:  Names [a, b]      NumNames 2
  case, this PR:        Names [a, b, c, b]   position = slot, b twice, last wins
  case, with an offset: Names [c, b]         offset 2, c is slot 2, b' is slot 3
  ```

  A lookup of `b` in the case then finds one `b` there, else goes to the `switch`'s block, so no last-wins rule is needed and no later lookup by name can land on the copy. `GetBlockNodeForPath` already splits slots below the offset to the parent, and `ExpandWith` would read the case's values at `slot - offset`. #ISSUE takes the same idea to every scope of a function.

<details><summary>all 21 programs against Go</summary>

Each program ran under `go run` on Go 1.25.9 and under `gno run` built from this head, from its base, and from this head merged with #6060. A value of a locally declared type is printed through `int(x)`, since gno's `println` prints such a value with its type.

| Program | What it checks | This head | Base | With #6060 |
|---|---|---|---|---|
| `c01_pr_example` | use before the shadow, shadow, `fallthrough` | same | differs | same |
| `c02_rhs_outer` | `v := v + 10`, the right-hand `v` is the outer one | same | differs | same |
| `c03_goto_back` | `goto` jumping back over the shadow | same | differs | same |
| `c04_else_if` | `else if` under an `if` init, closure made before the shadow | same | same | same |
| `c05_type_shadow` | `type v int` shadowing the init `v` | panics | panics | same |
| `c06_shadow_twice` | shadow, `fallthrough`, shadow again | same | differs | same |
| `c07_const_len` | `const v` used as an array length | same | panics | same |
| `c08_loop_closures` | closures made in a loop capture the shadow | same | differs | same |
| `c09_pointer_outer` | pointer to the outer `v` taken before the shadow | same | differs | same |
| `c10_typeswitch` | type switch: body shadows an init name, the variable shadows one | same | panics | same |
| `c11_nested` | nested switch shadowing again | same | differs | same |
| `c12_multi_define` | `v, w := v+1, 3` | same | differs | same |
| `c13_if_else` | `if`/`else` body shadow, `var v = v * 3` | same | same | same |
| `c14_two_init_names` | two init names, the second one shadowed | same | panics | same |
| `c15_reuse_loop` | the same switch run three times in a loop | same | panics | same |
| `c16_defer_capture` | `defer` reading the outer `v` | same | same | same |
| `c17_shadow_used_in_init_closure` | closure assigned to a variable outside the switch | same | same | same |
| `n1_block_noshadow` | `type T` in a `{ }` block, nothing shadowed | panics | panics | same |
| `n2_if_noshadow` | `type T` in an `if` body, nothing shadowed | panics | panics | same |
| `n3_for_noshadow` | `type T` in a `for` body, nothing shadowed | panics | panics | same |
| `n4_func_shadow` | function-level `type v` shadowing a package-level `v` | same | same | same |

`c01_pr_example`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		println("before", v)
		v := 99
		println("inner", v)
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// before 1
// inner 99
// outer 1
```

`c02_rhs_outer`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		v := v + 10
		println("inner", v)
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// inner 11
// outer 1
```

`c03_goto_back`

```go
package main

func main() {
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

// Output:
// L 1
// after 6
// L 1
// after 7
```

`c04_else_if`

```go
package main

func main() {
	if v := 1; v > 5 {
		println("no")
	} else if w := v + 1; v < 5 {
		f := func() (int, int) { return v, w }
		println("pre", v, w)
		v := w * 10
		w := v + 1
		println("post", v, w)
		a, b := f()
		println("closure", a, b)
	}
}

// Output:
// pre 1 2
// post 20 21
// closure 1 2
```

`c05_type_shadow`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		println("before", v)
		type v int
		var x v = 7
		println("x", int(x))
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// before 1
// x 7
// outer 1
```

`c06_shadow_twice`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		v := 2
		println("c1", v)
		fallthrough
	case 2:
		println("c2 pre", v)
		v := 3
		println("c2", v)
		fallthrough
	case 3:
		println("c3", v)
	}
}

// Output:
// c1 2
// c2 pre 1
// c2 3
// c3 1
```

`c07_const_len`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		println("pre", v)
		v = 4
		const v = 3
		var a [v]int
		println(len(a), v)
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// pre 1
// 3 3
// outer 4
```

`c08_loop_closures`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		get := func() int { return v }
		v := 100
		var fs []func() int
		for i := 0; i < 3; i++ {
			v++
			fs = append(fs, func() int { return v + i })
		}
		for _, f := range fs {
			println(f())
		}
		println("outer via closure", get())
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// 103
// 104
// 105
// outer via closure 1
// outer 1
```

`c09_pointer_outer`

```go
package main

func main() {
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

// Output:
// inner 9
// outer 6
```

`c10_typeswitch`

```go
package main

func main() {
	switch v := 42; x := any(v).(type) {
	case int:
		v := "s"
		println(v, x)
	}
	switch v := 42; v := any(v).(type) {
	case int:
		println(v + 1)
	case string:
		println(v)
	}
}

// Output:
// s 42
// 43
```

`c11_nested`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		switch v := v + 1; v {
		case 2:
			v := v * 10
			println("deep", v)
			fallthrough
		default:
			println("mid", v)
		}
		println("top", v)
	}
}

// Output:
// deep 20
// mid 2
// top 1
```

`c12_multi_define`

```go
package main

func main() {
	switch v := 1; v {
	case 1:
		v, w := v+1, 3
		println("inner", v, w)
		fallthrough
	case 2:
		println("outer", v)
	}
}

// Output:
// inner 2 3
// outer 1
```

`c13_if_else`

```go
package main

func main() {
	if v := 1; v > 5 {
		println("no")
	} else {
		println("else pre", v)
		v := 2
		println("else", v)
	}
	if v := 1; v < 5 {
		var v = v * 3
		println("then", v)
	}
}

// Output:
// else pre 1
// else 2
// then 3
```

`c14_two_init_names`

```go
package main

func main() {
	switch a, b := 1, 2; {
	case a < b:
		b := "x"
		println(a, b)
		fallthrough
	default:
		println(a, b)
	}
}

// Output:
// 1 x
// 1 2
```

`c15_reuse_loop`

```go
package main

func main() {
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

// Output:
// c0 10
// c1 0
// c1 1
// d s
```

`c16_defer_capture`

```go
package main

func f() (r int) {
	switch v := 1; v {
	case 1:
		defer func() { r = v }()
		v := 50
		_ = v
	}
	return 0
}

func main() {
	println(f())
}

// Output:
// 1
```

`c17_shadow_used_in_init_closure`

```go
package main

func main() {
	g := func() int { return 0 }
	switch v := 1; v {
	case 1:
		g = func() int { return v }
		v := 7
		v++
		println("inner", v)
	}
	println("g", g())
}

// Output:
// inner 8
// g 1
```

`n1_block_noshadow`

```go
package main

func main() {
	{
		type T int
		var x T = 7
		println(int(x))
	}
}

// Output:
// 7
```

`n2_if_noshadow`

```go
package main

func main() {
	if true {
		type T int
		var x T = 7
		println(int(x))
	}
}

// Output:
// 7
```

`n3_for_noshadow`

```go
package main

func main() {
	for i := 0; i < 1; i++ {
		type T struct{ A int }
		println(T{A: i}.A)
	}
}

// Output:
// 0
```

`n4_func_shadow`

```go
package main

var v = 1

func main() {
	println(v)
	type v int
	var x v = 7
	println(int(x))
}

// Output:
// 1
// 7
```

</details>
