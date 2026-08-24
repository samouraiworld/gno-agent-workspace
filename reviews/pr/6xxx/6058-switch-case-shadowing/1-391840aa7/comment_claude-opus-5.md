# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: COMMENT

## Body
[AI review]

- Nit: the `nameIndex` comment at [`nodes.go:1671-1678`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L1671-L1678) names `Define2` as the append path, and after this branch `Define2` never appends; `defineNew` does.

## gnovm/pkg/gnolang/nodes.go:2341 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2341) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2341)
A use of the outer name before a `const` shadow prints 0 where Go prints the outer value, because this append records the name in `Consts` and [`getLocalIsConst`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L1916-L1918) matches it by name alone. [#6060](https://github.com/gnolang/gno/pull/6060) fixes it, so this branch ships the wrong value unless that one lands first.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
cat > gnovm/tests/files/switch58.gno <<'GNO'
package main

func main() {
	switch v := 1; v {
	case 1:
		println(v)
		const v = "c"
		println(v)
	}
}

// Output:
// 1
// c
GNO
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/switch58.gno$' -count=1
rm gnovm/tests/files/switch58.gno
```

The `// Output:` block is what `go run` prints for the same source. The run fails:

```
--- FAIL: TestFiles/switch58.gno (0.00s)
    files_test.go:135: Output diff:
        --- Expected
        +++ Actual
        @@ -1,2 +1,2 @@
        -1
        +0
         c
```

The same divergence appears in an `if` branch, an `else` branch, a clause reached by `fallthrough`, and through a closure called before the shadow. It is not confined to `println`: `x := v * 2` before a `const v = 3` in a clause whose init is `v := 10` returns 0 rather than 20.

The name-keyed `Consts` predates this branch, and an ordinary nested block prints `0` too. What is new is that a case body reaches it at all: the same file answers `StaticBlock.Define2(v) cannot change const status` at the merge base.

Merging 6830e2549 into this head reports no conflict and turns the file green, with `switch52.gno` still passing:

```
--- PASS: TestFiles/switch52.gno (0.00s)
--- PASS: TestFiles/switch58.gno (0.00s)
```

Refusing the append here when `isConst` is set brings the old rejection back and reddens `switch52.gno`, which this branch adds. Only position-sensitive const resolution passes both.
</details>

## gnovm/pkg/gnolang/op_exec.go:736 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L736) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/op_exec.go#L736)
Accounted allocation grows with the length of a `fallthrough` chain while the block does not, because this truncation makes [`ExpandWith`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L3149-L3151) size its `AllocateBlockItems` call against the switch's own name count rather than the block's current length.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6058 -R gnolang/gno
curl -fsSL -o gnovm/pkg/gnolang/zz_ftgas_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6058-switch-case-shadowing/1-391840aa7/tests/fallthrough-alloc_test.go
go test ./gnovm/pkg/gnolang/ -run 'TestZZFallthroughAlloc' -count=1 -v
rm gnovm/pkg/gnolang/zz_ftgas_test.go
```

`zzChain(n, 4)` builds a switch of `n` clauses chained by `fallthrough`, each declaring four locals, so no clause is narrower than the one before it. Accounted bytes grow with the chain while the footprint does not:

```
ZZ chain-1x4          allocDelta=2760    gasDelta=9553      cycleDelta=7449
ZZ chain-2x4          allocDelta=2920    gasDelta=12486     cycleDelta=10323
ZZ chain-10x4         allocDelta=4200    gasDelta=35950     cycleDelta=33315
ZZ chain-50x4         allocDelta=10600   gasDelta=153270    cycleDelta=148275
```

On that path [`growBlockValues`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L2879-L2881) re-slices inside the capacity already there, so nothing is allocated for the charge, and [`Allocate`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/alloc.go#L327-L355) both charges gas and counts toward `maxBytes`, which drives the GC callback. `cap(b.Values)` does not move, so `GetShallowSize` and the storage deposit are unaffected.
</details>

## gnovm/pkg/gnolang/preprocess.go:1007 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1007) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L1007)
Missing test: no filetest shadows a name inside a type switch, so this write is never exercised on a clause block that carries one.

<details><summary>test cases</summary>

Passes at the head; panics at the merge base with `cannot change .T; was string, new int`. The `// Output:` block is `go run`'s.

```go
// gnovm/tests/files/typeswitch11.gno
package main

// A type switch clause may shadow a name declared by the type switch's init
// statement, and the type switch variable itself may shadow one.
func main() {
	var i any = 42
	switch v := "init"; y := i.(type) {
	case int:
		println("pre", v, y)
		v := y * 2
		println("post", v)
	default:
		println("def", v)
	}

	switch v := 42; v := any(v).(type) {
	case int:
		println("ts int", v)
	default:
		println("def")
	}
}

// Output:
// pre init 42
// post 84
// ts int 42
```
</details>

## gnovm/pkg/gnolang/nodes.go:2003-2011 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2003-L2011) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2003)
Missing test: nothing covers last-wins here, and no filetest can: put first-wins back and a filetest with a wide shadowing clause still passes.

<details><summary>test cases</summary>

`GetLocalIndex` itself does diverge on the revert, so the assertion belongs on the two branches directly. Add to `gnovm/pkg/gnolang/nodes_test.go`, with `"fmt"` in the import block:

```go
func TestStaticBlock_GetLocalIndex_LastWinsPastThreshold(t *testing.T) {
	// A faux case block that shadows a name copied in from its if/switch
	// holds that name twice, and GetLocalIndex must answer the later slot.
	// Past nameIndexThreshold (32) that answer comes from nameIndex rather
	// than the linear scan, so buildNameIndex has to be last-wins too.
	// Both widths are checked so the two branches cannot drift apart.
	for _, width := range []int{20, 40} {
		sb := new(gnolang.StaticBlock)
		for i := range width {
			n := gnolang.Name(fmt.Sprintf("n%02d", i))
			switch i {
			case 0:
				n = "v" // the faux copy
			case 3:
				n = "v" // the shadow declared in the case body
			}
			sb.Names = append(sb.Names, n)
			sb.Types = append(sb.Types, nil)
			sb.NameSources = append(sb.NameSources, gnolang.NameSource{})
			sb.HeapItems = append(sb.HeapItems, false)
		}
		sb.NumNames = uint16(width)

		idx, ok := sb.GetLocalIndex("v")
		assert.True(t, ok, "width %d", width)
		assert.Equal(t, uint16(3), idx, "width %d: want the shadow's slot", width)
	}
}
```

Green at the head. With the map restored to first-wins:

```
--- FAIL: TestStaticBlock_GetLocalIndex_LastWinsPastThreshold (0.00s)
    nodes_test.go:82: Error: Not equal: expected: 0x3  actual: 0x0
                      Messages: width 40: want the shadow's slot
```
</details>

## gnovm/pkg/gnolang/nodes.go:2333-2339 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2333-L2339) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2333)
Suggestion: this branch never runs, because the type switch variable is reserved at [`preprocess.go:567`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L567) at exactly `numFauxCopiedNames()` and the `idx >= sb.numFauxCopiedNames()` return above it always fires first.

```suggestion
		}
```

<details><summary>what the deletion was run against</summary>

Replacing the branch body with `panic("ZZ NSTypeSwitch branch reached")` and running `TestFiles` in full reaches it zero times. With the branch deleted the same suite is green, along with `TestStaticBlock`, `TestRunMemPackage`, `TestDebug` and `TestPreprocess`.

`switch t := x.(type) { case int: t := 5 }` is still rejected, by `go/types` and in Go's own wording:

```
// TypeCheckError:
// main/zzts2.gno:7:5: no new variables on left side of :=
```
</details>

## gnovm/pkg/gnolang/nodes.go:2365-2370 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2365-L2370) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2365)
Suggestion: no shipped build ties `idx` to `n`, so a boundary that drifts mis-types a name instead of panicking, and `debugAssert` is a [`make` target](https://github.com/gnolang/gno/blob/391840aa7/gnovm/Makefile#L118) that no workflow runs.

```suggestion
	if sb.Names[idx] != n {
		panic(fmt.Sprintf(
			"faux copy slot %d holds %s, not %s", idx, sb.Names[idx], n))
	}
```

<details><summary>what it costs</summary>

The two call sites are [`preprocess.go:1009`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1009) and [`preprocess.go:4022`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L4022), both at preprocess time and neither in the VM loop, so the check costs one `Name` comparison per copied name per clause and nothing for an `if` or `switch` with no init statement. With the wrapper dropped, `TestFiles` is green in full and never trips it.
</details>

## gnovm/tests/files/if9.gno:1 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/tests/files/if9.gno#L1) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/tests/files/if9.gno#L1)
Missing test: four shapes reach the new append and none of the eight new files covers them.

- an `else if`, which opens its own faux block with its own init
- a shadow in a `default` clause
- two init names with only one shadowed
- leaving a shadowing clause by labelled `break` or `goto`, which takes the `GOTO` arm of [`op_exec.go:708-717`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L708-L717) and never touches `b.Values`

<details><summary>test cases</summary>

Each panics at the merge base, passes at the head, and carries `go run`'s output.

```go
// gnovm/tests/files/if10.gno
package main

// An `else if` opens its own faux block, so its init statement's name may be
// shadowed in that branch, and a branch of the outer `if` may shadow the outer
// init name from inside the `else`.
func main() {
	if v := 1; v == 0 {
		println("no")
	} else if v := "two"; v == "two" {
		v := 3
		println("elseif", v)
	} else {
		println("no", v)
	}

	if v := 1; v == 0 {
		println("no")
	} else if v == 1 {
		v := "inner"
		println("chain", v)
	}
}

// Output:
// elseif 3
// chain inner
```

```go
// gnovm/tests/files/switch54.gno
package main

// Shadowing a switch or if init name from a default clause, and with two init
// names of which only one is shadowed.
func main() {
	switch v := 1; v {
	case 9:
		println("no")
	default:
		println("pre", v)
		v := "d"
		println("post", v)
	}

	switch a, b := 1, 2; a {
	case 1:
		println("outer", a, b)
		b := "shadow"
		println("inner", a, b)
	}

	if a, b := 1, 2; a == 1 {
		println("outer", a, b)
		b := "shadow"
		println("inner", a, b)
	}
}

// Output:
// pre 1
// post d
// outer 1 2
// inner 1 shadow
// outer 1 2
// inner 1 shadow
```

```go
// gnovm/tests/files/switch55.gno
package main

// Leaving a clause that shadows its switch's init name, by labelled break and
// by goto: the jump must not leave the shadow's slot live.
func main() {
L:
	switch v := 1; v {
	case 1:
		v := "shadow"
		println("inner", v)
		if len(v) > 0 {
			break L
		}
		println("unreachable")
	}

	switch v := 1; v {
	case 1:
		v := 10
		if v > 5 {
			goto done
		}
		println("no")
	}
done:
	println("done")
}

// Output:
// inner shadow
// done
```

```go
// gnovm/tests/files/switch57.gno
package main

// Both slots of a shadowed name heap-captured in the same clause, then a
// fallthrough out of it; and the same pair inside a loop, where the faux block
// is re-created per iteration.
func main() {
	switch v := 1; v {
	case 1:
		f := func() int { return v }
		v := 99
		g := func() int { return v }
		println("outer-cap", f(), "shadow-cap", g())
		fallthrough
	case 2:
		println("next", v)
	}

	fs := []func() int{}
	for i := 0; i < 2; i++ {
		if v := i; v >= 0 {
			fs = append(fs, func() int { return v })
			v := v + 10
			fs = append(fs, func() int { return v })
		}
	}
	for _, f := range fs {
		println(f())
	}
}

// Output:
// outer-cap 1 shadow-cap 99
// next 1
// 0
// 10
// 1
// 11
```

`switch53.gno` splits the outer capture and the shadow capture across two separate switches, so `HeapItems` is never non-`false` on the copied slot and the shadow slot at once, which `switch57.gno` covers.
</details>

## gnovm/adr/pr6058_faux_block_shadowing.md:58 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L58) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/adr/pr6058_faux_block_shadowing.md#L58)
Nit: `Reserve` has sixteen call sites, at `preprocess.go` lines 409, 439, 450, 456, 481, 517, 525, 532, 540, 549, 563, 567, 577, 585, 594 and 777.

```suggestion
   Every one of the sixteen `Reserve` call sites passes a stable triple, but
```

## gnovm/adr/pr6058_faux_block_shadowing.md:81-87 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L81-L87) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/adr/pr6058_faux_block_shadowing.md#L81)
Nit: this paragraph credits the `NSTypeSwitch` test, which never runs, and dismisses the slot index, which is what holds.

```suggestion
A **type switch's variable is deliberately left unshadowable**. Go declares it
in each clause's own block, so a clause body redeclaring it is an error, not a
shadow, and `go/types` rejects the program with "no new variables on left side
of :=" exactly as before. Nothing in `Reserve` has to enforce that: the variable
is reserved at exactly `numFauxCopiedNames()`, so the
`idx >= numFauxCopiedNames()` return above it always fires first.
```

## gnovm/adr/pr6058_faux_block_shadowing.md:142-147 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L142-L147) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/adr/pr6058_faux_block_shadowing.md#L142)
Nit: no workflow builds `debugAssert`, so the check enforces nothing on a pull request or in a shipped build, and `defineNew` is not the only append path: the amino decoder appends to `Names` at [`pb3_gen.go:12685`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/pb3_gen.go#L12685) and 12707.

```suggestion
- Only a faux case block holds a name twice, and `defineNew` is the one path
  that can append the second slot. A `debugAssert` check there panics on any
  duplicate at or past the boundary, which `make -C gnovm test.debugAssert`
  runs and no workflow does. A full `-tags debugAssert` filetest run does not
  trip it, and its failure set is identical to the base commit's apart from the
  three new tests, which the base fails for lack of the fix.
```
