# Review: [#6058](https://github.com/gnolang/gno/pull/6058)
Event: COMMENT

## Body
[AI review]

## gnovm/pkg/gnolang/op_exec.go:736 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/op_exec.go#L736) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/pkg/gnolang/op_exec.go#L736)
Accounted allocation grows with the length of a `fallthrough` chain while the block does not, because this truncation makes [`ExpandWith`](https://github.com/gnolang/gno/blob/391840aa7/gnovm/pkg/gnolang/values.go#L3149-L3151) size its `AllocateBlockItems` call against the switch's own name count rather than the block's current length. It arrives with [#6056](https://github.com/gnolang/gno/pull/6056) and master measures the same, so the fix belongs there rather than here.

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
Missing test: no filetest puts a shadow in a type switch clause, so nothing pins this write to the slot before the shadow rather than after it.

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
Missing test: nothing covers last-wins here, and no filetest can, because putting first-wins back leaves a clause wide enough to reach this map still printing the same output.

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

## gnovm/adr/pr6058_faux_block_shadowing.md:161-166 [gh](https://github.com/gnolang/gno/blob/391840aa7/gnovm/adr/pr6058_faux_block_shadowing.md?plain=1#L161-L166) · [↗](../../../../../.worktrees/gno-review-6058/gnovm/adr/pr6058_faux_block_shadowing.md#L161)
Nit: master rejects the case-body form, `switch v := 1; v { case 1: println(v); const v = "c" }`, with `StaticBlock.Define2(v) cannot change const status`, so the nested block shares the mechanism and not the reachability.

```suggestion
  const". The mechanism is not new: an ordinary nested block
  (`v := 1; { println(v); const v = "c" }`) misbehaves identically on master,
  because the shadow's `Consts` entry exists from `initStaticBlocks` on while
  `GetIsConst` has no notion of statement position. What is new is that a case
  body reaches it at all: master rejects the same program outright with
  `StaticBlock.Define2(v) cannot change const status`. #6060 closes it for both
  block kinds.
```
