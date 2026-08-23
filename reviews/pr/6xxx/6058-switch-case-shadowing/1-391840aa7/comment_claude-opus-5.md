# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: REQUEST_CHANGES

## Body
- The `nameIndex` contract comment at [`nodes.go:1671-1678`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L1671-L1678) still names `Define2` as the append path and as what maintains the map; after this branch `Define2` never appends and `Reserve` is a second entry into `defineNew`.

## gnovm/pkg/gnolang/nodes.go:2341 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2341) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2341)
A `const` shadow reaches `Consts` through this append, and [`getLocalIsConst`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L1916-L1918) matches by name, so a use of the outer name textually before the declaration const-folds against the copy's value-less slot and yields the zero value where Go yields the outer one. [#6060](https://github.com/gnolang/gno/pull/6060) is what settles it, so the order the two land in decides whether master carries the divergence.

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

The name-keyed `Consts` predates this branch, and an ordinary nested block prints `0` too. What is new is that a case body reaches it: the same file answers `StaticBlock.Define2(v) cannot change const status` at the merge base.

Merging 6830e2549 into this head reports no conflict and turns the file green, with `switch52.gno` still passing:

```
--- PASS: TestFiles/switch52.gno (0.00s)
--- PASS: TestFiles/switch58.gno (0.00s)
```

Refusing the append here when `isConst` is set restores the merge base's rejection and reddens `switch52.gno` instead, so position-sensitive const resolution is the only shape that satisfies both.
</details>

## gnovm/pkg/gnolang/op_exec.go:736 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L736) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/op_exec.go#L736)
Lowering `len(b.Values)` here makes [`ExpandWith`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L3149-L3151) size its `AllocateBlockItems` call against the switch's own name count rather than the previous clause's, so every `fallthrough` charges allocation gas for the target clause's whole set on a slice [`growBlockValues`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L2879-L2881) only re-slices within capacity.

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

The VM cycle count is identical to the pre-truncation figures on every row, so the whole delta is allocation accounting. [`Allocate`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/alloc.go#L327-L349) both charges gas and counts toward `maxBytes`, which drives the GC callback. `cap(b.Values)` does not move, so `GetShallowSize` and the storage deposit are unaffected.
</details>

## gnovm/pkg/gnolang/preprocess.go:1007 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1007) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/preprocess.go#L1007)
Missing test: no filetest constructs a type switch, so the move from three `last.Define` calls to a write at `numFauxCopiedNames()` is uncovered, and so is the call site the `NameSource` comparison in `Reserve` exists for.

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
Missing test: nothing covers the move to last-wins, and a filetest cannot cover it, because reverting this map to first-wins makes `Reserve` append a third slot for the same name and program output never moves.

<details><summary>test cases</summary>

`GetLocalIndex` does diverge on the revert, so the assertion belongs on the two branches directly. Add to `gnovm/pkg/gnolang/nodes_test.go`, with `"fmt"` in the import block:

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

## gnovm/pkg/gnolang/nodes.go:2334-2339 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2334-L2339) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2334)
Suggestion: the type switch variable is reserved at [`preprocess.go:567`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L567) right after the copy loop has made exactly `Parent.GetNumNames()` slots, so its index equals `numFauxCopiedNames()` and the return four lines above always fires first, which leaves this branch unreachable and its `Define2` rejection never performed.

```suggestion
```

<details><summary>what the deletion was run against</summary>

Replacing the branch body with a panic and running the `switch`, `typeswitch`, `if`, `type` and `select` filetest families never reached it. With the branch deleted, `TestFiles` is green in full, and `TestStaticBlock`, `TestRunMemPackage`, `TestDebug` and `TestPreprocess` pass.

`switch t := x.(type) { case int: t := 5 }` is rejected by `go/types`, carrying Go's own wording, at the head and at the merge base alike:

```
main/zzts1.gno:7:5: no new variables on left side of :=
```

So [`pr6058_faux_block_shadowing.md:81-88`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L81-L88) argues the opposite of what holds: the slot index is the only thing stopping it, and that is enough.
</details>

## gnovm/pkg/gnolang/nodes.go:2365-2370 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/nodes.go#L2365-L2370) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/nodes.go#L2365)
Suggestion: `debugAssert` is built by [`gnovm/Makefile:118`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/Makefile#L118) and by no workflow, so in every shipped build this function is two writes with nothing tying `idx` to `n`, and a boundary that drifts mis-types a name instead of panicking.

```suggestion
	if sb.Names[idx] != n {
		panic(fmt.Sprintf(
			"faux copy slot %d holds %s, not %s", idx, sb.Names[idx], n))
	}
```

<details><summary>what it costs</summary>

The two call sites are [`preprocess.go:1009`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L1009) and [`preprocess.go:4022`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/preprocess.go#L4022), both at preprocess time and neither in the VM loop, so the check costs one `Name` comparison per copied name per clause and nothing for an `if` or `switch` with no init statement. With the wrapper dropped and a second copy of the check made unconditional, `TestFiles` is green in full and never trips it.

Same paragraph: `pr6058_faux_block_shadowing.md:142-147` reads "enforced, not just documented" for the `defineNew` assertion, which holds for a local run and for no check and no shipped binary.
</details>

## gnovm/tests/files/if9.gno:1 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/tests/files/if9.gno#L1) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/tests/files/if9.gno#L1)
Missing test: four shapes reach the new append and none of the eight new files covers them, an `else if`, a shadow in a `default` clause, two init names with one shadowed, and leaving a shadowing clause by labelled `break` or `goto`, which takes the `GOTO` arm of [`op_exec.go:708-717`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L708-L717) and restores frames without touching `b.Values`.

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
Nit: the `Reserve` call sites number sixteen, at `preprocess.go` lines 409, 439, 450, 456, 481, 517, 525, 532, 540, 549, 563, 567, 577, 585, 594 and 777, and line 143's sole-append claim has one more exception, the amino decoder appending to `Names` at [`pb3_gen.go:12685`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/pb3_gen.go#L12685) and 12707.

```suggestion
   Every one of the sixteen `Reserve` call sites passes a stable triple, but
```
