# Review: [#6209](https://github.com/gnolang/gno/pull/6209)
Posted: https://github.com/gnolang/gno/pull/6209#pullrequestreview-5548097705
Event: COMMENT
Verdict: NEEDS DISCUSSION. The four scope fixes hold, and two points stay open: a func literal's result names are masked before its signature is resolved, which predates the branch and does not block it, and loop shapes the base already runs give a different result once a node re-preprocesses a deployed package.
Model: claude-opus-5-5, standard review, solo shape: 2 finders at xhigh, judge at high
Commit: 1de7f587d
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6209 1de7f587d`
Round: 1. Solo round: 2 finders, one reflector pass, 7 candidates (4 from the finders, 3 from the reflector), each run from scratch by a judge that was not its finder, the one Nit included.

## Body

- Related: [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/1de7f587d/gnovm/pkg/gnolang/preprocess.go#L344) masks a func literal's result names before its signature, so `func() (x [len(x)]int)` in a loop over `x` reads an outer `x`. Pushing the `FuncLitExpr` frame at `TRANS_BLOCK` returns Go's length and keeps the six new `loopvar_*` fixtures green.

  <details><summary>repro</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6209 -R gnolang/gno
  cat > gnovm/tests/files/zz_funclit_type_mask.gno <<'EOF'
  package main

  func main() {
  	x := [3]int{}
  	_ = x
  	for x := [2]int{}; x[0] < 1; x[0]++ {
  		f := func() (x [len(x)]int) { return }
  		println(len(f()))
  	}
  }

  // Output:
  // 2
  EOF
  (cd gnovm/pkg/gnolang && go test -short -count=1 -run 'TestFiles/zz_funclit_type_mask.gno$' .)
  rm gnovm/tests/files/zz_funclit_type_mask.gno
  ```

  The fixture fails because the result type resolves against the outer array:

  ```
  files_test.go:135: Output diff:
      @@ -1 +1 @@
      -2
      +3
  --- FAIL: TestFiles/zz_funclit_type_mask.gno
  ```

  Inside `for x := [2]int{}`, the literal sizes its result from the outer `x := [3]int{}` and returns 3 where Go returns 2, with no error. The merge base gives the same `-2 +3`. The param form, `func(x [len(x)]int) int`, is refused at head and at the merge base with `name x not declared`.
  </details>

- Nit: [`pushClone`](https://github.com/gnolang/gno/blob/1de7f587d/gnovm/pkg/gnolang/preprocess.go#L295) skips only when `modify == nil`, so the second `initStaticBlocks` pass allocates a frame per `FuncDecl` and `FuncLitExpr`. Both callers passing a `modify` only delete, so `if len(parent) == 0` suffices: 100 functions drop from 206 allocations to 4.

  <details><summary>allocation count</summary>

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 6209 -R gnolang/gno
  cat > gnovm/pkg/gnolang/zz_allocs_test.go <<'EOF'
  package gnolang

  import (
  	"fmt"
  	"strings"
  	"testing"
  )

  func TestZZSecondPassAllocs(t *testing.T) {
  	for _, n := range []int{1, 10, 100} {
  		var sb strings.Builder
  		sb.WriteString("package main\n")
  		for i := 0; i < n; i++ {
  			fmt.Fprintf(&sb, "func f%d(a int) int { g := func(b int) int { return b }; return g(a) }\n", i)
  		}
  		m := NewMachine("main", nil)
  		fn := m.MustParseFile("a.gno", sb.String())
  		pn := NewPackageNode("main", "main", nil)
  		initStaticBlocks1(nil, pn, fn)
  		a := testing.AllocsPerRun(10, func() { initStaticBlocks1(nil, pn, fn) })
  		t.Logf("funcs=%d second-pass allocs=%.0f", n, a)
  	}
  }
  EOF
  (cd gnovm/pkg/gnolang && go test -short -count=1 -v -run 'TestZZSecondPassAllocs$' . | grep allocs=)
  rm gnovm/pkg/gnolang/zz_allocs_test.go
  ```

  ```
  funcs=1 second-pass allocs=7
  funcs=10 second-pass allocs=25
  funcs=100 second-pass allocs=206
  ```

  With `if len(parent) == 0 {` in `pushClone`: 3, 3 and 4, and `TestFiles/(loopvar|closure|heap|fun|range|typeswitch|for|var|block|shadow).*` passes.
  </details>

## gnovm/pkg/gnolang/preprocess.go:377 [gh](https://github.com/gnolang/gno/blob/1de7f587d/gnovm/pkg/gnolang/preprocess.go#L377) · [↗](../../../../../.worktrees/gno-review-6209/gnovm/pkg/gnolang/preprocess.go#L377) · Suggestion [posted](https://github.com/gnolang/gno/pull/6209#discussion_r4294908148)

Suggestion: `TRANS_BLOCK2` changes the result of four loop shapes the merge base already runs, so scan deployed packages for them before the upgrade.

<details><summary>cases and repro</summary>

Where a plain variable of the loop variable's name is in scope, the merge base binds the clause to it; head matches Go in all four:

- `for x := range make([]int, x)` inside `for x := 1; x < 3; x++`, after `x := 100`: 200 iterations on the merge base, 3 at head.
- `switch x := x.(type)` inside `for x := any(1); ...`, after `var x any = "outer"`: `other outer` on the merge base, `int 1` at head.
- `for k = range []int{7, 8}` inside `for k := 0; ...`, after `k := -1`: the range writes the outer `k` on the merge base and the loop's `k` at head.
- `for x := x` inside `for x := 0; ...` after an inner `x := 5`: prints 0 to 6 on the merge base, 5 and 6 at head.


The keeper re-preprocesses every stored package at boot ([`keeper.go:203`](https://github.com/gnolang/gno/blob/1de7f587d/gno.land/pkg/sdk/vm/keeper.go#L203)), so a deployed realm with one of these shapes computes the head result from the first block on the new binary.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6209 -R gnolang/gno
d=gnovm/tests/files
cat > $d/zz_range_outer_plain.gno <<'EOF'
package main

func main() {
	x := 100
	_ = x
	for x := 1; x < 3; x++ {
		for x := range make([]int, x) {
			println(x)
		}
	}
}

// Output:
// 0
// 0
// 1
EOF
cat > $d/zz_typeswitch_outer_plain.gno <<'EOF'
package main

func main() {
	var x interface{} = "outer"
	_ = x
	for x := interface{}(1); x != nil; x = nil {
		switch x := x.(type) {
		case int:
			println("int", x)
		default:
			println("other", x)
		}
	}
}

// Output:
// int 1
EOF
cat > $d/zz_range_assign_outer_plain.gno <<'EOF'
package main

func main() {
	k := -1
	for k := 0; k < 1; k++ {
		for k = range []int{7, 8} {
		}
		println(k)
	}
	println(k)
}

// Output:
// 1
// -1
EOF
cat > $d/zz_shadow_between.gno <<'EOF'
package main

func main() {
	for x := 0; x < 1; x++ {
		x := 5
		for x := x; x < 7; x++ {
			println(x)
		}
	}
}

// Output:
// 5
// 6
EOF
run() { (cd gnovm/pkg/gnolang && go test -short -count=1 -v -run 'TestFiles/zz_' . 2>&1 | grep -E -- '--- (PASS|FAIL)|^\s+[-+@]'); }
run                                   # head: four PASS
git checkout origin/master -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/transcribe.go
run                                   # merge base: four FAIL
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/transcribe.go
rm $d/zz_*.gno
```

On the merge base the four programs run to completion and print the outer variable's result, which is the change of behaviour (`# …` marks omitted lines):

```
--- PASS: TestFiles/zz_range_assign_outer_plain.gno
--- PASS: TestFiles/zz_shadow_between.gno
--- PASS: TestFiles/zz_range_outer_plain.gno
--- PASS: TestFiles/zz_typeswitch_outer_plain.gno
    @@ -1 +1,6 @@
    +0
    +1
    # …
    @@ -1,2 +1,2 @@
    +0
     1
    --1
    @@ -1 +1 @@
    -int 1
    +other outer
    @@ -1,3 +1,200 @@
    # …
--- FAIL: TestFiles/zz_shadow_between.gno
--- FAIL: TestFiles/zz_range_assign_outer_plain.gno
--- FAIL: TestFiles/zz_typeswitch_outer_plain.gno
--- FAIL: TestFiles/zz_range_outer_plain.gno
```

[`store.go:162-163`](https://github.com/gnolang/gno/blob/1de7f587d/gnovm/pkg/gnolang/store.go#L162-L163) keeps BlockNodes out of the database, so every boot rebuilds them from source with the running binary's preprocessor.
</details>

## SKIP gnovm/pkg/gnolang/preprocess.go:398 [gh](https://github.com/gnolang/gno/blob/1de7f587d/gnovm/pkg/gnolang/preprocess.go#L398) · [↗](../../../../../.worktrees/gno-review-6209/gnovm/pkg/gnolang/preprocess.go#L398) · Suggestion

Suggestion: the init names enter the frame after `.Init`, so `for x := x` below an inner `x := 5` prints 5 6, not the merge base's 0 to 6.

Not posted: the same program is the fourth case of the section at `preprocess.go:377`.
