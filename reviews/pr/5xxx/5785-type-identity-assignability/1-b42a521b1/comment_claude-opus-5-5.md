# Review: [#5785](https://github.com/gnolang/gno/pull/5785)
Posted: https://github.com/gnolang/gno/pull/5785#pullrequestreview-5547651848
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The runtime interface check in `VerifyImplementedBy` walks the whole method signature for a flat gas charge, a pricing gap this branch introduces and the one blocker; three identity gaps of the class the branch sweeps (variadic spread, `copy`, `:=` redeclaration) predate it and do not block on their own.
Model: claude-opus-5-5, standard review, solo shape: 2 finders at xhigh, judge at high
Commit: b42a521b16655374e507bdb949ab5b8289bc8404
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-5785 b42a521b1`
Round: 1. Two finders, one judge acting as reflector and writer, 7 candidates; every Warning re-run from scratch at the head and at the merge base by an agent that was not its finder, the Missing test and the Suggestion read and run once.

## Body

- Related: [`StaticBlock.Define2`](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/nodes.go#L2389) compares `TypeID()`, so `y, x := 1, struct{A int "b"}{2}` changes the static type of an `x` declared as `struct{A int "a"}`, which Go rejects.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5785 -R gnolang/gno
  cat > gnovm/tests/files/zz_redeclare.gno <<'EOF'
  package main

  func main() {
  	x := struct {
  		A int "a"
  	}{}
  	y, x := 1, struct {
  		A int "b"
  	}{2}
  	println(x.A, y)
  }

  // Output:
  // PLACEHOLDER
  EOF
  (cd gnovm/pkg/gnolang && go test -count=1 -run 'TestFiles/zz_redeclare.gno$' -update-golden-tests . >/dev/null)
  sed -n '/^\/\/ Output:/,$p' gnovm/tests/files/zz_redeclare.gno
  rm gnovm/tests/files/zz_redeclare.gno
  ```

  The VM accepts the redeclaration, which Go rejects:

  ```
  // Output:
  // 2 1
  
  // TypeCheckError:
  // main/zz_redeclare.gno:7:13: cannot use struct{A int}{…} (value of type struct{A int "b"}) as struct{A int "a"} value in multiple assignment
  ```
  </details>

- Related: the variadic branch of [`checkAssignableTo`](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/type_check.go#L584-L585) checks element assignability rather than identity, so a spread `[]int` passes as `...any` and the callee writes any value into the caller's slice.
  - `g(s...)` with `s []int` and `g(xs ...any)`: `xs[0] = "hello"` leaves `s[0]` printing `hello`.
  - `f(ts...)` with `ts []T` and `f(xs ...I)`: storing a `U` leaves `ts[0].x` printing `hello` and `ts[0].M()` running `U.M`.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5785 -R gnolang/gno
  cat > gnovm/tests/files/zz_vararg_spread.gno <<'EOF'
  package main

  type I interface{ M() int }

  type T struct{ x int }

  func (t T) M() int { return t.x }

  type U struct{ s string }

  func (u U) M() int { return len(u.s) }

  func f(xs ...I) { xs[0] = U{"hello"} }

  func g(xs ...any) { xs[0] = "hello" }

  func main() {
  	ts := []T{{1}}
  	f(ts...)
  	println(ts[0].x, ts[0].M())
  	s := []int{1, 2}
  	g(s...)
  	println(s[0])
  }

  // Output:
  // PLACEHOLDER
  EOF
  (cd gnovm/pkg/gnolang && go test -count=1 -run 'TestFiles/zz_vararg_spread.gno$' -update-golden-tests . >/dev/null)
  sed -n '/^\/\/ Output:/,$p' gnovm/tests/files/zz_vararg_spread.gno
  rm gnovm/tests/files/zz_vararg_spread.gno
  ```

  The VM runs both calls, which Go rejects; only the go/types pass reports them:

  ```
  // Output:
  // hello 5
  // hello
  
  // TypeCheckError:
  // main/zz_vararg_spread.gno:19:4: cannot use ts (variable of type []T) as []I value in argument to f; main/zz_vararg_spread.gno:22:4: cannot use s (variable of type []int) as []any value in argument to g
  ```
  </details>

- Related: [`copy`](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/uverse.go#L857-L859) types `dst` and `src` as the independent generics `X` and `Y`, so a `copy` from `[]string` into `[]int` runs and stores `"x"` in an `int` slot.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5785 -R gnolang/gno
  cat > gnovm/tests/files/zz_copy_elem.gno <<'EOF'
  package main

  func main() {
  	dst := make([]int, 1)
  	src := []string{"x"}
  	n := copy(dst, src)
  	println(n, dst[0])
  }

  // Output:
  // PLACEHOLDER
  EOF
  (cd gnovm/pkg/gnolang && go test -count=1 -run 'TestFiles/zz_copy_elem.gno$' -update-golden-tests . >/dev/null)
  sed -n '/^\/\/ Output:/,$p' gnovm/tests/files/zz_copy_elem.gno
  rm gnovm/tests/files/zz_copy_elem.gno
  ```

  The VM copies a string into an `int` slot, which Go rejects; [`copy_struct_tags.gno`](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/tests/files/copy_struct_tags.gno#L13-L14) pins the same acceptance with a `TypeCheckError` and no `Error`:

  ```
  // Output:
  // 1 x
  
  // TypeCheckError:
  // main/zz_copy_elem.gno:6:12: invalid copy: arguments dst (variable of type []int) and src (variable of type []string) have different element types int and string
  ```
  </details>

## gnovm/pkg/gnolang/types.go:1253 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/types.go#L1253) · [↗](../../../../../.worktrees/gno-review-5785/gnovm/pkg/gnolang/types.go#L1253) · Warning [posted](https://github.com/gnolang/gno/pull/5785#discussion_r4294560575)
`identicalTypes(mt, im.Type)` walks the method signature on each `x.(I)` at flat gas: 20000 assertions cost 42.9M gas in 0.01s with 1 field and 1.39s with 5000.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5785 -R gnolang/gno
d=$(mktemp -d)
go build -o $d/gno ./gnovm/cmd/gno
for size in 1 20 50; do
  inner=" G0 int;"
  [ $size = 1 ] || { inner=""; for j in $(seq 0 99); do inner="$inner G$j int;"; done; }
  outer=""; for i in $(seq 1 $size); do outer="$outer F$i struct{$inner};"; done
  p=$d/s$size; mkdir -p $p
  printf 'module = "gno.land/p/demo/sfcost"\ngno = "0.9"\n' > $p/gnomod.toml
  cat > $p/a.gno <<EOF
package sfcost

type I interface{ M(s struct{$outer}) }

type T struct{}

func (T) M(s struct{$outer}) {}

func Loop(n int) int {
	var x any = T{}
	c := 0
	for i := 0; i < n; i++ {
		if _, ok := x.(I); ok {
			c++
		}
	}
	return c
}
EOF
  cat > $p/a_test.gno <<'EOF'
package sfcost

import "testing"

func TestLoop(t *testing.T) {
	if Loop(20000) != 20000 {
		t.Fatal("bad")
	}
}
EOF
  printf 'fields=%s source=%s bytes: ' $([ $size = 1 ] && echo 1 || echo $((size * 100))) $(wc -c < $p/a.gno)
  (cd $p && GNOROOT=$OLDPWD $d/gno test -v -print-runtime-metrics . 2>&1) | grep -oE 'TestLoop \([0-9.]+s\)|GAS: *[0-9]+' | tr '\n' ' '
  echo
done
rm -rf $d
```

The gas stays flat while the time grows with the signature:

```
fields=1 source=263 bytes: TestLoop (0.01s) GAS:  42886054 
fields=2000 source=36363 bytes: TestLoop (0.56s) GAS:  42886532 
fields=5000 source=90603 bytes: TestLoop (1.39s) GAS:  42900410
```

The merge base compared two `TypeID()` strings here, which `FuncType` caches, so its cost per check was a string compare.
</details>

## SKIP gnovm/pkg/gnolang/types.go:1253 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/pkg/gnolang/types.go#L1253) · [↗](../../../../../.worktrees/gno-review-5785/gnovm/pkg/gnolang/types.go#L1253) · Suggestion
Suggestion: this flips deployed `x.(interface{ M(...int) })` assertions on `M([]int)` from `true` to `false`; ship it as a chain upgrade or keep runtime callers on `TypeID`.

Not posted: the ADR's Decision and Consequences record the runtime interface-assertion change as intended, and its rationale saying static-only strictness avoids consensus-visible changes is ADR text.

## gnovm/tests/files/interface51.gno:12 [gh](https://github.com/gnolang/gno/blob/b42a521b16655374e507bdb949ab5b8289bc8404/gnovm/tests/files/interface51.gno#L12) · [↗](../../../../../.worktrees/gno-review-5785/gnovm/tests/files/interface51.gno#L12) · Missing test [posted](https://github.com/gnolang/gno/pull/5785#discussion_r4294560587)
Missing test: no filetest runs `x.(I)` or `case I` against a method differing only in variadicity or a tag, so reverting the runtime half keeps every test green.

<details><summary>test cases</summary>

```go
package main

type T struct{}

func (T) M(xs []int) {}

type S struct{}

func (S) N(s struct{ A int "a" }) {}

func main() {
	var x any = T{}
	_, ok := x.(interface{ M(...int) })
	println(ok)
	switch x.(type) {
	case interface{ M(...int) }:
		println("case I")
	default:
		println("default")
	}
	var y any = S{}
	_, ok2 := y.(interface{ N(struct{ A int "b" }) })
	println(ok2)
}

// Output:
// false
// default
// false
```
</details>
