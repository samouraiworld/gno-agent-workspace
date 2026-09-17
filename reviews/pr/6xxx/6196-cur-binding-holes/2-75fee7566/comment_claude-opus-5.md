# Review: [#6196](https://github.com/gnolang/gno/pull/6196)
Posted: https://github.com/gnolang/gno/pull/6196#pullrequestreview-5238487667
Verdict: COMMENT. The compile break this round opened on is fixed at this head by 379fc44c5; what is left is four polish findings and two suggestions, none of them blocking.
Event: COMMENT
Model: claude-opus-5, standard review
Commit: 75fee7566
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/75fee7566bb1d0f1d0f83765cbe34b9c54007339) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/75fee7566bb1d0f1d0f83765cbe34b9c54007339)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6196-new 75fee7566`
Round: 2. The head moved from 6d88deba7 to 75fee7566 while round 1 ran. Findings are carried verbatim, anchors re-cut at this head; three sections 379fc44c5 and the ADR rename closed are marked SKIP.

## Body
> AI review, claude-opus-5, standard review, [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6196-cur-binding-holes/overview.md) · Status: COMMENT

- Read at 6d88deba7, re-checked at 75fee7566: the `cur` write-rule findings are closed by 379fc44c5 and what follows survives it.
## SKIP gnovm/pkg/gnolang/preprocess.go:6111 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6111) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L6111) · Warning
`isCrossingCurParam` consults no type, so every body-level `cur` in a crossing function with an unnamed first realm parameter is refused [at its own `:=`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3009-L3015), [under `&cur`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2583-L2584) and [in a range clause](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3145-L3151), and the realm package stops compiling.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6196 -R gnolang/gno
cat > gnovm/tests/files/zz_b1_local_int.gno <<'EOF'
// PKGPATH: gno.land/r/demo/b1local
package b1local

// An int local named `cur` in the body of a crossing function whose first
// realm parameter is unnamed. Prints 2 at the merge base b7845fb2d; at
// 6d88deba7 it panics at the `cur := 1` line.
func F(_ realm) int {
	cur := 1
	cur = 2
	return cur
}

func main(cur realm) {
	println(F(cur))
}

// Output:
// 2
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_local_int.gno$'
rm gnovm/tests/files/zz_b1_local_int.gno
```

The declaration fails, not the assignment under it:

```
--- FAIL: TestFiles/zz_b1_local_int.gno (0.00s)
    files_test.go: unexpected panic: gno.land/r/demo/b1local/zz_b1_local_int.gno: cannot reassign the crossing `cur` parameter: it names the realm this frame is executing as, and that binding is fixed for the life of the frame
# …
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang
```

Mechanism: a function body's own top-level declarations live in the `FuncDecl` block, so `GetBlockNodeForPath` resolves such a `cur` to the `FuncDecl` itself. [`IsCrossing()`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/types.go#L1393-L1399) compares `Params[0].Type` to `gRealmType` and nothing else, so an unnamed first realm parameter leaves the name `cur` free for any declaration in that block while the function still answers true. The three rules calling it each tested the written name's static type against `gRealmType` before, which is the test that kept an int out. The same predicate drives all three, so the class covers assignment, definition, the address form and the range clause. Shapes affected are the common `func F(_ realm, ...)` signature used by r/gnoland/blog, r/gnoland/boards2/v0, r/demo/counter and r/sys/users.

</details>


Skipped: 379fc44c5 rewrote the write rules to key on the static type at every site, and the round's own fixture passes at this head.
## SKIP gnovm/pkg/gnolang/preprocess.go:2584 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2584) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L2584) · Warning
`&cur` on a plain int parameter named `cur` aborts the realm package with a message calling the binding realm-typed: `func F(_ realm, cur int)` is legal Gno, [the FuncTypeExpr rule](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2819-L2833) skips every parameter whose type is not `gRealmType`, and the guard here now asks only where the name was declared.

At the merge base the same rule read `xt == gRealmType` and the address was taken.

Skipped: one edit to `isCrossingCurParam` clears this together with the section anchored on that function, and the author's next edit is the same edit.

## SKIP gnovm/pkg/gnolang/preprocess.go:6112 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6112) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L6112) · Warning
The test is the name and the path kind, then the declaring node, so `cur := 1` inside `func F(_ realm) int` resolves to the crossing `FuncDecl` and panics at the declaration line, with [the write rule](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3009-L3015) now running for `DEFINE` as well.

Skipped: the declaration-site symptom of the missing type test, cleared by the same edit as the section anchored on `isCrossingCurParam`.

## SKIP gnovm/pkg/gnolang/preprocess.go:3009 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3009) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L3009) · Warning
A same-block `cur, x := cur.Previous(), 1` that forwards the rebound name nowhere is refused at preprocess, where the merge base compiled and ran it: [`installInheritedCur`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L513)'s identity check fires on a crossing entry and nowhere else, so a rebind that reaches no crossing entry was never checked.

The shape needs a multi-name `:=`, since a bare `cur := x` at the same level is already `no new variables on left side of :=`, and a sweep of `examples/`, `gno.land/` and the stdlibs finds no realm-typed `cur` declared outside a parameter.

Skipped: the only action is a correction to the description's Compatibility sentence, which never reaches the draft; the two runs stay in `claims.md`.

## SKIP gnovm/pkg/gnolang/preprocess.go:6116-6119 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6116-L6119) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L6116) · Warning
Already raised: https://github.com/gnolang/gno/pull/6196#discussion_r4037111651

Both arms ask only `IsCrossing()`, which compares [`Params[0].Type`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/types.go#L1393-L1399) to `gRealmType`, so a named result `cur realm` of `func F(_ realm) (cur realm)` is declared in the `FuncDecl`'s own block, resolves to it, and `cur = nil` is refused as this frame's realm identity, against the godoc three lines above it.

The shape is well formed because a blank first parameter is [renamed to `.arg_0`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L557-L560) and [the crossing-name rule](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2827) accepts any `.arg` prefix. `ft.IsCrossing() && ft.Params[0].Name == "cur"` in both arms restores the documented behaviour.

Skipped: the open thread on these lines states the claim and carries this patch; the run confirming it stays in `claims.md`.

## SKIP gnovm/pkg/gnolang/op_call.go:567 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L567) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L567) · Missing test
Missing test: [`doOpFuncLit`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_expressions.go#L775-L780) builds every literal's `FuncValue` with an empty `Name` and a set `Source`, so a literal always reaches the source-location arm at the identity panic, and no test asserts that the rendered text carries a file and a line.

Skipped: the same edit as the section anchored on `TestFuncDisplayName`.

## SKIP gnovm/pkg/gnolang/op_call_test.go:82 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call_test.go#L82) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call_test.go#L82) · Missing test
Missing test: no subtest puts a `testing` frame below the immediate caller, so widening [`harnessSeedsCur`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L558-L561) to any ancestor frame leaves all three cases green and stops the stale-capture check firing for every crossing call made under `gno test`.

<details>
<summary>test cases</summary>

```go
	t.Run("testing frame below the caller", func(t *testing.T) {
		m := &Machine{}
		m.Frames = append(m.Frames,
			Frame{Func: &FuncValue{PkgPath: TestingBasePkgPath}},
			Frame{Func: &FuncValue{PkgPath: "gno.land/r/test/realm_a"}},
			Frame{Func: &FuncValue{}},
		)
		require.False(t, m.harnessSeedsCur())
	})
```

With `harnessSeedsCur` replaced by a loop over `PeekCallFrame(n)` that returns true at the first `testing` frame it reaches, `TestHarnessSeedsCur`, `TestFiles/zrealm_cur_backstop.gno`, `TestFiles/zrealm_cur_defer.gno`, `TestFiles/zrealm_cur_method_backstop.gno` and `TestStdlibs/test-testing` all stay ok. The case above is the one that reddens.

</details>


Skipped: a mutation nobody ran, on a branch that adds six test files; the check stays in `claims.md`.
## SKIP gnovm/pkg/gnolang/op_call_test.go:110 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call_test.go#L110) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call_test.go#L110) · Missing test
Missing test: `TestFuncDisplayName` pins the [`fv.Source == nil` fallback](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L575), which only a hand-built `FuncValue` reaches, and never the [source-location arm](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L571-L573) every literal from [`doOpFuncLit`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_expressions.go#L775-L780) takes.

<details>
<summary>test cases</summary>

A crossing function literal entered without `cross()` with a cur captured in another frame trips the identity panic through that arm, so the rendered location is asserted by the filetest itself:

```go
// PKGPATH: gno.land/r/demo/curlitname
package curlitname

// Pins funcDisplayName's source-location arm: a crossing function literal
// entered without cross() carries a cur captured in another frame, so the
// identity panic names the literal by file and line.

func exec(cur realm, cb func()) {
	cb()
}

func main(cur realm) {
	f := func(cur realm) { println("f:", cur.PkgPath()) }
	cb := func() { f(cur) } // captures main's cur, calls the crossing literal
	exec(cross(cur), cb)    // ...but runs inside exec's frame
}

// Error:
// crossing function gno.land/r/demo/curlitname func literal at zrealm_cur_literal_name.gno:27 was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
```

`grep -rn 'func literal at' gnovm/` matches [op_call.go:573](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L573) alone, and the three filetests reaching the identity panic, `zrealm_cur_defer.gno`, `zrealm_cur_backstop.gno` and `zrealm_cur_method_backstop.gno`, all name their functions and take the `Name != ""` arm. Deleting the four lines of the source-location arm leaves every one of those green.

</details>


Skipped: the uncovered arm is one branch of a two-branch helper, which costs the author a case and no behaviour.
## SKIP gnovm/pkg/gnolang/op_call_test.go:113 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call_test.go#L113) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call_test.go#L113) · Missing test
Missing test: nothing in the tree asserts the `%s func literal at %s:%d` arm of [`funcDisplayName`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L571-L573), so an edit dropping the file and line from it leaves the identity panic without a location and the suite green.

<details>
<summary>test cases</summary>

`gnovm/tests/files/zrealm_cur_litname.gno`, run with `go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_litname.gno$'`:

```go
// PKGPATH: gno.land/r/demo/curlitname
package curlitname

// Pins the func-literal arm of funcDisplayName: the callee of the identity
// panic is a crossing function literal, so the message has to carry its file
// and line. Mutating that format string reddens this file and leaves
// TestFuncDisplayName ok.

func exec(cur realm, cb func()) {
	cb()
}

func main(cur realm) {
	tgt := func(cur realm) { println("tgt:", cur.PkgPath()) }
	cb := func() { tgt(cur) } // captures main's cur
	exec(cross(cur), cb)      // ...but runs inside exec's frame
}

// Error:
// crossing function gno.land/r/demo/curlitname func literal at zrealm_cur_litname.gno:14 was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
```

The named-func arm is already pinned by `zrealm_cur_backstop.gno`; the fallback arm is pinned by the assertion here. Only the middle arm has no cell.

</details>


Skipped: the same missing arm as the section above it, so one case covers both.
## SKIP gnovm/tests/files/zrealm_cur_shadow.gno:19 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/files/zrealm_cur_shadow.gno#L19) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/files/zrealm_cur_shadow.gno#L19) · Missing test
Missing test: the file ends on `println("ok")` without reading `cur` back, so its four writes assert only that the block preprocesses: a shadow aliasing the parameter's heap item would null main's identity with every line here passing.

<details>
<summary>test cases</summary>

Two lines close it: cur-call a crossing helper with the parameter after the block, which reads `true` at the reviewed head.

```go
func probe(cur realm) {
	println(cur.IsCurrent())
}

func main(cur realm) {
	{
		cur := cur
		cur = nil
		p := &cur
		*p = nil
		_ = cur
	}
	probe(cur)
	println("ok")
}

// Output:
// true
// ok
```

The static half is self-pinning: a shadow resolving to the parameter would make [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6111) answer true and the file would go red at preprocess. The runtime half, a slot that reads block-scoped statically while aliasing the parameter's `*HeapItemValue`, is what no assertion here covers.

</details>


Skipped: the fixture pins what the branch changed, and the read-back is an addition rather than a gap.
## SKIP gnovm/tests/files/zrealm_cur_other_legal.gno:9 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/files/zrealm_cur_other_legal.gno#L9) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/files/zrealm_cur_other_legal.gno#L9) · Missing test
Missing test: the package-level arm is spelled `pkgCur`, a name [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6112) rejects before it looks at the declaring block, so no fixture in the tree covers a package-level `var cur realm`.

<details>
<summary>test cases</summary>

`gnovm/tests/files/zrealm_cur_pkgvar.gno`, run with `go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_pkgvar.gno$'`:

```go
// PKGPATH: gno.land/r/demo/curpkgvar
package curpkgvar

// A package-level realm var named cur is not any frame's identity, so writes
// to it compile. Keying the write rule on the name alone reddens this file;
// the pkgCur spelling stays green under that same mutation.

var cur realm

func set(x int, r realm) {
	cur = r
	cur = nil
}

func main() {
	set(0, nil)
	println("ok")
}

// Output:
// ok
```

</details>


Skipped: the fixture covers the class and spells one name differently, which costs a rename and no behaviour.
## SKIP gnovm/tests/files/zrealm_cur_other_legal.gno:17 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/files/zrealm_cur_other_legal.gno#L17) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/files/zrealm_cur_other_legal.gno#L17) · Missing test
Missing test: this named result belongs to `namedResult`, which is not crossing, so nothing covers `cur` as the named result of a crossing `func F(_ realm) (cur realm)`, where [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6115-L6119) refuses the write as the crossing `cur` parameter.

<details>
<summary>test cases</summary>

`gnovm/tests/files/zz_b1_named_result.gno`, run with `go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zz_b1_named_result.gno$'`:

```go
// PKGPATH: gno.land/r/demo/b1named
package b1named

// A named result `cur` in a crossing function whose first realm parameter is
// blank. A named result is not any frame's identity, so the file has to print
// ok; the write rule refuses the assignment at preprocess instead.

func F(_ realm) (cur realm) {
	cur = nil
	return cur
}

func main(cur realm) {
	println("ok")
}

// Output:
// ok
```

The fixture asserts `ok` and the run stops at preprocess, which is the uncovered cell:

```
unexpected panic: gno.land/r/demo/b1named/zz_b1_named_result.gno:9:2-11: cannot reassign the crossing `cur` parameter: …
# …
```

A blank first realm parameter is renamed `.arg`, so the declaration is legal and the refusal is not introduced by the fixtures added here: it predates them, and the class they sweep is the one it belongs to.

</details>


Skipped: the shape it asks for is the one the open thread on `isCrossingCurParam` already argues.
## SKIP gnovm/adr/prxxxx_cur_binding_followups.md:1 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/adr/prxxxx_cur_binding_followups.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/adr/prxxxx_cur_binding_followups.md#L1) · Nit
Nit: the filename keeps `prxxxx` and the [Status](https://github.com/gnolang/gno/blob/75fee7566/gnovm/adr/prxxxx_cur_binding_followups.md?plain=1#L5) names the branch it was implemented on, so deleting the branch leaves the record with no pointer to the change it decides.


Skipped: the file is renamed pr6196_cur_binding_followups.md at this head.
## SKIP gnovm/pkg/gnolang/debugger.go:811 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/debugger.go#L811) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/debugger.go#L811) · Nit
Related nit: the non-method arm formats `PkgPath` and `Name` unconditionally and a function literal's `FuncValue` carries an empty `Name`, so a stack line through a closure frame renders as `gno.land/r/demo/x.` with nothing after the separator, where the identity panic names the literal's file and line.

SKIP: `debugger.go` carries no hunk in this diff, so an inline anchor on it is rejected at post time.

## SKIP gnovm/pkg/gnolang/op_call.go:452 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L452) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L452) · Nit
Nit: the `NOTE` carrying the deferred unmetered walk points at a tracker outside this repository, the only cross-repo issue reference in the VM source, where the comparable one beside it, [`nocompile_on_32bits.go:7`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/nocompile_on_32bits.go#L7), names an issue in this repository.

Skipped: whether a contributor outside the organisation can open that tracker is the whole finding and no check here settles it, and a finding on a code comment's wording earns no inline slot either way.

## gnovm/pkg/gnolang/op_call.go:560 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L560) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L560) · Nit [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090797)
Nit: `caller.Func != nil` cannot be false, because [`PeekCallFrame`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/machine.go#L2908-L2930) returns a frame only from inside its `fr.IsCall()` branch and [`IsCall`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/frame.go#L78-L80) is `return fr.Func != nil`.

```suggestion
	return caller != nil && caller.Func.PkgPath == TestingBasePkgPath
```

## gnovm/pkg/gnolang/op_call.go:573 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L573) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L572) · Suggestion [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090826)
Suggestion: `loc.File` is the basename alone, so the panic renders a literal as `<pkgpath> func literal at <basename>:<line>` while [the stacktrace](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/frame.go#L168) from the same frame renders the `<pkgpath>/<basename>:<line>` [`Location.String()`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/nodes_location.go#L304) already produces; 2 lines become 1.

```suggestion
		return fmt.Sprintf("func literal at %s", fv.Source.GetLocation())
```

`Location.String()` also carries the column span, which the stacktrace line omits.

## gnovm/pkg/gnolang/op_call_test.go:87 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call_test.go#L87) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call_test.go#L87) · Nit [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090835)
Nit: `Crossing: true` reads as a condition of the exemption, and [`harnessSeedsCur`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L559-L560) tests the caller frame's `PkgPath` alone, so the callee fixture here and in the other two subtests needs nothing beyond a non-nil [`Func`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/frame.go#L78-L80).

```suggestion
			Frame{Func: &FuncValue{}},
```

## SKIP gnovm/pkg/gnolang/preprocess.go:3113 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3145) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L3145) · Nit
Nit: the surviving `n.Op != DEFINE` carve-out cannot change an outcome, since a range `DEFINE` [reserves its names against the `RangeStmt`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L589-L603) and [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6115-L6121) switches on `*FuncDecl` and `*FuncLitExpr` only. The rule then reads two ways across its two sites, and a reader has to redo the block-push argument to see that this one is not a second hole.

Skipped: the same edit as the suggestion anchored on this rule, which deletes the wrapper outright.

## SKIP gnovm/tests/files/zrealm_cur_other_legal.gno:21 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/files/zrealm_cur_other_legal.gno#L21) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/files/zrealm_cur_other_legal.gno#L21) · Nit
Nit: the leading `x int` is what keeps `rangeLocal` non-crossing, since [`FuncType.IsCrossing`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/types.go#L1393-L1399) tests the first parameter alone and a crossing declaration then [requires the name `cur`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2828), so deleting the parameter as filler takes the file red at preprocess.

SKIP: the fix is a clause on the file's own header comment, which changes no behaviour.

## gnovm/tests/stdlibs/testing/cur_subtest_test.gno:12-14 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/stdlibs/testing/cur_subtest_test.gno#L12-L14) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/stdlibs/testing/cur_subtest_test.gno#L12) · Nit [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090841)
Nit: a crossing function's own first `cur` is the topmost crossing frame's `Cur`, which is what [`IsCurrent`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/uverse.go#L585-L588) compares against, so the panic arm cannot fire and deleting it leaves the same regression in 22 lines instead of 25.

```suggestion
```

<details>
<summary>what still reddens</summary>

With the guard dropped, deleting the `harnessSeedsCur()` term from the stale-capture check still fails `TestStdlibs/test-testing`: the sub-test closure panics at its own crossing entry, before `curSubtestTarget` is reached. Flipping the guard to `if cur.IsCurrent() { panic(...) }` in a copy of the file panics on every run, which is the same fact from the other side.

</details>

## SKIP gnovm/pkg/gnolang/op_call.go:153 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L153) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L153) · Suggestion
Suggestion: a realm minted with a nil or non-pointer prev keeps `gOriginRealmTV` in `Fields[2]`, because both constructors default that field to the placeholder and replace it only for a `PointerValue` prev, so such a value compares equal here, is rebuilt from `buildOriginRealm`, and skips the caller-identity check through `rebuilt`.

The header's sentence about the materialized per-tx origin holds: `buildOriginRealm` passes `TypedValue{}` to `newRealmHIVPointer`, leaving `Fields[2].V` nil so the pointer assertion fails. The constructor a caller can hand its own prev today is `MakeRealmValue` through the testing stdlib, so the loss is latent.

Skipped: op_call.go:153 sits outside this file's diff hunks, which cover lines 75-143 and 407-597, so an inline comment on it is rejected at submit; the finding stays in `claims.md`.

## gnovm/pkg/gnolang/op_call.go:521-523 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/op_call.go#L521-L523) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/op_call.go#L521) · Suggestion [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090850)
Related suggestion: the identity guard raises a Go string panic, so a realm wrapping the call in `defer`/`recover()` aborts the whole message; `m.PanicString` puts it on gno's exception path instead.

<details>
<summary>mechanism</summary>

The guard raises a Go string rather than an `*Exception`, which the VM loop re-raises past the op loop, and the fault predates the highlighted lines. A crossing function reached through `F(cross(cur))` whose body calls `Target(cur)` with a named result `cur` trips the guard, and a deferred `recover()` in `F` never runs: the Go stack unwinds past the op loop, so the filetest reports `unexpected panic` and prints nothing from the recover.

</details>

## SKIP gnovm/pkg/gnolang/preprocess.go:2154-2177 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2154-L2177) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L2154) · Suggestion
Suggestion: the `GetBlockNodeForPath` switch here resolves `nx.Path` and tests `IsCrossing()` on the same two node kinds as [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6111-L6122), returning the same verdict, so 24 lines fold to 6 and the declaration test lives in one function instead of two.

```suggestion
								// Check `cur` directly from parent crossing function's argument.
								// NOTE: TRANS_ENTER *FuncTypeExpr ensures that `cur realm` is the
								// first argument of the crossing function.
								if !isCrossingCurParam(store, last, nx) {
									panic("only the `cur` argument of a containing crossing function maybe passed by cross-call")
								}
```

<details>
<summary>what the fold changes</summary>

The helper's two extra guards are already satisfied at this site: `nx.Name == "cur"` is the enclosing `Name("cur")` arm, and the `nx.Path.Type != VPBlock` early return only replaces [`GetBlockNodeForPath`'s own `expected block type value path` panic](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/nodes.go#L1869) for a `NameExpr` that cannot reach here with a non-block path. The `default` arm's message is the one the helper's `false` return now produces through the `if`.

</details>


Skipped: the fold was against `isCrossingCurParam`, which 379fc44c5 deletes.
## gnovm/pkg/gnolang/preprocess.go:3145 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L3141-L3153) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L3109) · Suggestion [posted](https://github.com/gnolang/gno/pull/6196#discussion_r4039090855)
Suggestion: a range `DEFINE` binds its key and value into [the `RangeStmt`'s own block](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L589-L603), which is still on the stack here, so [`isCrossingCurParam`](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6115-L6121) answers false for both whatever `n.Op` is and the `n.Op != DEFINE` guard decides nothing; 13 lines become 7.

```suggestion
				for _, lh := range []Expr{n.Key, n.Value} {
					ne, ok := lh.(*NameExpr)
					if !ok || !isCrossingCurParam(store, last, ne) {
						continue
					}
					panic("cannot assign to a realm-typed `cur` in a range clause: it names the realm this frame is executing as, and that binding is fixed for the life of the frame")
				}
```

The `ASSIGN` rejection the rule exists for is untouched.

## SKIP gnovm/pkg/gnolang/preprocess.go:6121 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L6121) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/pkg/gnolang/preprocess.go#L6121) · Suggestion
Suggestion: the unknown-kind case returns false here while [the provenance switch's default](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2175-L2176) panics, which is the right pair for every kind `GetBlockNodeForPath` returns today: a package-level `var cur realm` resolves to a `FileNode`, writable and not cur-callable at once. The godoc's "the same declaration test" overstates it, since the two share a lookup and not a verdict.

Skipped: the read settles the wording and not a behaviour, and a finding on a comment's own words earns no inline slot; confirming the behavioural half needs a third function-shaped block node kind, which no branch has.

## SKIP gnovm/tests/files/zrealm_cur_other_legal.gno:29 [gh](https://github.com/gnolang/gno/blob/75fee7566/gnovm/tests/files/zrealm_cur_other_legal.gno#L29) · [↗](../../../../../.worktrees/gno-review-6196-new/gnovm/tests/files/zrealm_cur_other_legal.gno#L29) · Suggestion
Suggestion: this call and the two under it pass nil, so the fixture shows the write rules staying silent and not that the three bindings can hold a realm, which a crossing declaration [refuses one level up](https://github.com/gnolang/gno/blob/75fee7566/gnovm/pkg/gnolang/preprocess.go#L2828).

SKIP: the fix is a line on the file's own header comment, which changes no behaviour.
