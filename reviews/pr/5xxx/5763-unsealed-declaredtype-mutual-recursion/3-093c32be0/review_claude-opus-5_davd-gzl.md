# PR [#5763](https://github.com/gnolang/gno/pull/5763): fix(gnovm): allow unsealed *DeclaredType during mutual type-decl recursion

URL: https://github.com/gnolang/gno/pull/5763
Author: ltzmaxwell | Base: master | Files: 3 | +82 -7
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: `093c32be0` (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5763 093c32be0`

Round 3, deep, over the same commit round 2 reviewed. Four lenses ran in parallel: red team, blue team, correctness with Go-parity pairs, and consensus impact. It overturns round 2's APPROVE. Round 2 checked two-type cycles and found them correct, which holds. Nobody built a three-type cycle, and that is where the fix stops working. Two lenses independently cleared the process-global uverse write that a third then measured as a data race, so the disagreement was settled by running it rather than by counting votes.

## Overview

Gno lets you declare two named types that refer to each other, `type T1 struct{Next *T2}` next to `type T2 T1`. The compiler used to abort on this with an internal `panic("should not happen")`. This PR removes that abort and adds a helper, `fillTypeInPlace`, that completes the second type's underlying struct from the first one's once both are known. For two types the result is right: both halves resolve, they stay distinct named types, and illegal cycles are still rejected with a better message than before.

The helper runs at the end of every named type declaration, not only recursive ones, and it works by overwriting an existing type object in place. Those two facts are where the remaining problems come from: a cycle of three types finishes in an order the helper does not handle and leaves a type with no underlying type at all, and a declaration whose underlying type is a shared global gets that global written.

**Verdict: REQUEST CHANGES** — a three-type cycle leaves `DeclaredType.Base` nil, which `gno lint` no longer reports at all where master named the file and line, and the failure resurfaces unpositioned in the type-persistence path; separately the fill writes the process-global uverse `error` object, adding a data race no CI job runs (1 Critical, 3 Warnings, 2 Missing tests, 2 Nits, 1 Suggestion).

## Verify first

- [`preprocess.go:3077`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077) — the `dstT.Base != nil` guard silently skips finalization instead of rejecting it. Run [`tests/adv_decltype_chain3_dependent_first.gno`](tests/adv_decltype_chain3_dependent_first.gno) as a filetest and confirm it prints `7 8 9` rather than nil-dereferencing.
- [`types.go:1577`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1577) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1577) — confirm the destination is never a type this package does not own. `go test -race -run TestZZUverseBaseRace` with [`tests/consensus/uverse_base_race_test.go`](tests/consensus/uverse_base_race_test.go) must report no frame naming `fillTypeInPlace`.
- [`preprocess.go:5504`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504) — this widens what a node accepts. Confirm no failed `addpkg` of a mutual-recursion package sits in the history the chain replays.

## Summary

`tryPredefine` aborted whenever a type declaration's right-hand side resolved to a `*DeclaredType` that was not yet sealed, which is a normal state midway through mutual recursion. The PR drops that abort and adds [`fillTypeInPlace`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1553-L1592) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1553-L1592), which copies a completed underlying type into an existing pointer of the same kind so a dependent that captured that pointer while it was empty observes the finished contents. For every two-type shape this is correct across all seven base kinds, in both declaration orders, matching real Go byte for byte. The fix is real and the round-1 corruption is genuinely gone.

What it does not cover is a cycle long enough that some member finalizes before its own base exists. With three types the finalize order can put the dependent first, its source's base is still nil, the `dstT.Base != nil` guard skips the fill, and `*dstT = *tmp2` writes a nil `Base` into the slot. Nothing downstream asserts a non-nil base, so the type reaches the persistence layer and the developer's own gate stays quiet.

## Fix

Before, [`tryPredefine`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5501-L5511) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5501-L5511) panicked on an unsealed `*DeclaredType`; now it reads the slot and continues. The load-bearing addition is at the `*DeclaredType` finalize branch, [`preprocess.go:3070-3079`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3070-L3079) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3070-L3079), where the completed base is copied into the pointer the dependent already aliases and that pointer is then reused. Filling in place rather than swapping is the constraint the whole design rests on: a swap would leave the dependent holding the empty original. The gap is that the guard treats "the source has no base yet" as a case to skip rather than a case to reject, so the one shape the design cannot serve is also the one shape it does not diagnose.

## Critical (must fix)

- **[nil underlying type survives finalization]** [`preprocess.go:3077`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077) — a three-type cycle whose dependent is declared first leaves `DeclaredType.Base` nil; `gno lint` drops from a positioned error to exit 0, and the failure resurfaces with no file or line.
  <details><summary>details</summary>

  The shape is ordinary Go and compiles under go1.26.5, printing `7 8 9`:

  ```go
  type T2 T1
  type T1 struct { Next *T3; Val int }
  type T3 T2
  ```

  T3 finalizes before T1. Its source's base is still nil, so the `dstT.Base != nil` guard skips the fill, and `*dstT = *tmp2` two lines below copies the nil straight into the slot. Nothing between there and the store asserts a non-nil base. The first use reaches [`elideCompositeElements`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5823) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5823), where `baseOf(clt)` returns nil, the type switch falls to `default`, and `clt.String()` dereferences a nil `Type`.

  Measured on both trees with the same toolchain:

  | | merge-base `0397fc87f` | head `093c32be0` |
  |---|---|---|
  | `gno lint` on a package holding the shape | exit 1, `nilbase.gno:3:6-11: should not happen (code=gnoPreprocessError)` | **exit 0, silent** |
  | `gno test` on that package | exit 1, same positioned error | `panic: cannot copy nil types`, [`realm.go:1501`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/realm.go#L1501) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/realm.go#L1501), no position |
  | the shape as a filetest | positioned `should not happen` at 15:6-11 | unpositioned Go nil dereference |

  Master rejects this shape too, so it is not a regression in what compiles. It is a regression in what the author is told: the gate that named their file and line now says nothing, and the same nil arrives later in `copyTypeWithRefs` where no source position is available to report. On chain it is contained, read from the code rather than exercised: `AddPackage` has `defer doRecover(m2, &err)` at [`keeper.go:731`](https://github.com/gnolang/gno/blob/093c32be0/gno.land/pkg/sdk/vm/keeper.go#L731) · [↗](../../../../../.worktrees/gno-review-5763/gno.land/pkg/sdk/vm/keeper.go#L731), so the transaction fails rather than the node.

  A census at the fill site over the stock `gnovm/tests/files` corpus counted 3,182 calls, of which 3 had a nil destination, so the skipped-guard path is reachable from the existing suite and not only from a constructed case.

  The shape is a member of the class this PR sweeps: the title is mutual type-decl recursion, and this is mutual type-decl recursion one hop longer than the case that was fixed. Fix: reject rather than skip. When `dstT.Base` is nil at finalize, raise a positioned preprocessor error naming the declaration instead of storing the nil, or order the finalize so a dependent never runs before its source has a base. Repro: [`tests/adv_decltype_chain3_dependent_first.gno`](tests/adv_decltype_chain3_dependent_first.gno), package form in [`tests/nilbase_package_repro.sh`](tests/nilbase_package_repro.sh).
  </details>

### Repro

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > gnovm/tests/files/chain3.gno <<'EOF'
package main

type T2 T1

type T1 struct {
	Next *T3
	Val  int
}

type T3 T2

func main() {
	var a T1
	a.Next = &T3{Val: 7}
	println(a.Next.Val)
	var b T2
	b.Val = 8
	println(b.Val)
	var c T3
	c.Val = 9
	println(c.Val)
}

// Output:
// 7
// 8
// 9
EOF
go test -run 'TestFiles/chain3.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/chain3.gno
```

The run fails, and the failure is the finding: the file is valid Go that prints `7 8 9`, and the head aborts with an unpositioned runtime fault instead.

```
--- FAIL: TestFiles/chain3.gno (0.00s)
    files_test.go:111: unexpected panic: runtime error: invalid memory address or nil pointer dereference
    …
    gnolang.elideCompositeElements(…) preprocess.go:5863
    gnolang.preprocess1.func1.2(…)    preprocess.go:1249
```

## Warnings (should fix)

- **[data race on a process-global type]** [`types.go:1577`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1577) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1577) — `type E error` makes the fill write the uverse `error` singleton, which master never wrote; two goroutines preprocessing it race.
  <details><summary>details</summary>

  For `type E error` the destination and the source are the same pointer, because both come from `baseOf` of the same uverse object. `fillTypeInPlace` still executes `*dst = *src`, so it writes a process-global that the declaring package does not own. The value is unchanged, but an unsynchronised write racing a concurrent read is a race regardless.

  Two goroutines each preprocessing `type E error`, same toolchain, `go test -race`:

  | Tree | `WARNING: DATA RACE` | frames naming `fillTypeInPlace` |
  |---|---|---|
  | merge-base `0397fc87f` | 2, both pre-existing `TypeID()` memoization and `SetCacheType` | 0 |
  | head `093c32be0` | 4 | 3 |
  | head plus the guard below | 2 | 0 |

  The reported address is a fixed static one, which is what a process-global looks like to the detector. No job catches this: gno CI runs no `-race`, and `go test -race ./gnovm/pkg/gnolang/` on the head is clean by itself because nothing in the suite declares such a type concurrently.

  Fix: add the pointer-inequality test the helper's own contract already assumes, `if dstT.Base != nil && dstT.Base != tmp2.Base && fillTypeInPlace(dstT.Base, tmp2.Base)`. It is a no-op whenever the pointers are equal, which is the only case that touches a foreign object. Verified on this tree: races drop to the merge-base level with zero `fillTypeInPlace` frames, and `decltype_mutual.gno` plus the all-seven-kinds fixture still pass. Reports: [`tests/race_base_mine.txt`](tests/race_base_mine.txt), [`tests/race_head_mine.txt`](tests/race_head_mine.txt), [`tests/race_guard_mine.txt`](tests/race_guard_mine.txt).
  </details>

- **[the accept boundary moves with no gate]** [`preprocess.go:5504`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504) — an unupgraded node rejects an `addpkg` an upgraded node accepts, and a historical failed `addpkg` of this shape re-executes as a success on replay.
  <details><summary>details</summary>

  Nine of twenty-four declaration shapes move from rejected to accepted, none move the other way, and `gno.land` carries no height gate or language-version switch on this path. The divergence needs one ordinary signed transaction from any funded account.

  The replay direction is the one a rollout note tends to miss. A block holding a *failed* `addpkg` of a mutual-recursion package re-executes as a success under the new binary, so a node syncing from genesis computes a different AppHash and halts at [`validation.go:70`](https://github.com/gnolang/gno/blob/093c32be0/tm2/pkg/bft/state/validation.go#L70) · [↗](../../../../../.worktrees/gno-review-5763/tm2/pkg/bft/state/validation.go#L70). This one is derived from the accept-boundary measurement rather than exercised end to end; a replay was not run.

  Existing programs are unaffected: a type-heavy package costs 34164 gas on both trees, and a corpus differential over 136 packages and 343 types found zero differences in TypeID or persisted amino bytes. Fix: say in the PR body that this widens what a node accepts and needs a coordinated upgrade, and confirm no failed `addpkg` of this shape sits in replayed history; gate on the mempackage's declared `gno` version if one does. Evidence: [`tests/consensus/accept_boundary_test.go`](tests/consensus/accept_boundary_test.go), [`tests/consensus/addpkg_boundary_test.go`](tests/consensus/addpkg_boundary_test.go).
  </details>

- **[the description omits the fix]** — the PR body's "## Changes" lists two files; the diff is three, and the missing one is where the fix lives.
  <details><summary>details</summary>

  The body was written for `61fc396e4` and never updated after `fillTypeInPlace` landed. It describes the change as dropping a stale sanity panic. Dropping that panic alone is exactly round 1, which produced the silent empty-base corruption; what makes the case actually resolve is [`types.go:1545-1592`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1545-L1592) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1545-L1592), +49 lines the body does not mention.

  A maintainer reading it approves a one-line revert of a stale assertion and merges a change that re-points `Base` for every named type in every gno package: 2,848 of 3,182 finalizations over the filetest corpus keep the predefine-time pointer instead of the one built at finalize. Fix: rewrite the body around `fillTypeInPlace`, what it fills, why in place rather than by swapping, and that the predefine base pointer is canonical from that point. The body's `realm.go:1788` citation is also off by one section; the sealed gate is at `realm.go:1810`.
  </details>

## Missing Tests

- **[six of seven kinds unpinned]** [`decltype_mutual.gno:1-19`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L1-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L1-L19) — the PR ships one filetest covering the `*StructType` arm; deleting the other six arms leaves the whole suite green.
  <details><summary>details</summary>

  Proved by mutation, twice and independently. Delete all six non-struct arms and `TestFiles` stays at exactly the failures it already has. Delete the `*StructType` arm alone and `decltype_mutual.gno` fails with `struct type struct{} has no field Val`, which is the round-1 bug. So one arm of seven is pinned and six could regress to the round-1 corruption unnoticed.

  Fix: extend the filetest to the slice, map, array, func, pointer and interface bases. A ready fixture covering all seven passes at `093c32be0` and aborts `should not happen` at the merge-base: [`tests/decltype_mutual_kinds.gno`](tests/decltype_mutual_kinds.gno), with the per-arm mutation table in [`tests/branch-necessity/`](tests/branch-necessity/).
  </details>

- **[nothing pins what the cycle writes to state]** [`decltype_mutual.gno:17-19`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno?plain=1#L17-L19) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L17-L19) — the filetest asserts stdout only, and the change's whole point is that two named types now share one underlying object.
  <details><summary>details</summary>

  A `// PKGPATH:` variant with `// Types:` and `// Realm:` freezes what actually matters: that the two halves keep distinct TypeIDs despite the shared `*StructType`, and that the persisted object diff and size are what they are. Fix: ship [`tests/consensus/zdecltype_mutual_realm.gno`](tests/consensus/zdecltype_mutual_realm.gno) beside the existing filetest; it fails at the merge-base and passes at head.
  </details>

## Nits

- **[comment scopes the branch to recursion, the code runs on everything]** [`preprocess.go:3070-3076`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3070-L3076) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3070-L3076) — the comment opens "During mutual type-decl recursion"; measured over `gnovm/tests/files` the branch fires 3,182 times, 2,848 of them with a real pointer change, and exactly one program in the suite is mutually recursive. Someone reading it to size the blast radius of a later change would be wrong by three orders of magnitude. Not posted, per the standing rule that a finding about a comment's own wording stays in the review file.

- **[the helper's contract lives in the caller]** [`types.go:1545-1552`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1545-L1552) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1545-L1552) — after a `true` return the caller must rebind `tmp2.Base = dstT.Base`; filling without rebinding leaves the dependent on the orphaned object. That requirement appears only in the call-site comment, so a second caller written from the helper's own doc would miss it. The `false` return also conflates "value kind, nothing to do" with "kinds disagree, something is wrong", and the mismatch branch is dead: 0 hits in 3,182 calls. Not posted, same rule.

## Suggestions

- **[alias above its target still aborts, now naming a different assertion]** [`preprocess.go:5504-5508`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504-L5508) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504-L5508) — `type A = B` written above `type B struct{...}` is valid Go and fails preprocessing on both trees. Put it in a mutual pair and the merge-base aborts with the very `should not happen` this PR removes, while the head aborts with `StaticBlock.Define2(A) cannot change .V`. The root cause predates the diff and reproduces with no recursion at all, so the head is never worse, but it is a member of the class the title sweeps and the user now gets an internal invariant's name instead of their own type's. Fix: say in the body that a forward alias is out of scope, or land a filetest pinning the ordering. Repros: [`tests/adv_decltype_alias_forward.gno`](tests/adv_decltype_alias_forward.gno) and [`tests/adv_decltype_alias_forward_plain.gno`](tests/adv_decltype_alias_forward_plain.gno).

## Verified

- The three-cycle Critical, run by me on both trees with the same toolchain: head gives an unpositioned nil dereference, merge-base gives `adv_decltype_chain3_dependent_first.gno:15:6-11: should not happen`, real Go accepts the program and prints `7 8 9`.
- `gno lint` exit code on a package carrying the shape: 0 at head, 1 with a positioned error at the merge-base. `gno test` at head panics `cannot copy nil types` inside `copyTypeWithRefs`.
- The uverse race, run by me: 2 races at the merge-base with zero `fillTypeInPlace` frames, 4 at head with 3, and back to 2 with zero once the pointer-inequality guard is applied. The guard leaves `decltype_mutual.gno` and the seven-kind fixture passing.
- Fill-site census over the stock corpus, my own instrumentation: 3,182 calls, 2,848 with a different destination pointer, 306 `PrimitiveType`, 3 with a nil destination, 0 kind mismatches.
- Round 2's four claims re-derived independently and all four held: `T1.Base == T2.Base`, named types stay distinct, illegal value cycles still reject, `PrimitiveType` correctly skipped. The diff also improves the value-cycle message from `should not happen` to `invalid recursive type`, which round 2 did not credit.
- Go parity over 51 program pairs against a real toolchain: 12 shapes fixed, 0 regressions, 0 cases where gno now accepts what Go rejects. The 7 divergences found are the forward-alias family and reject identically on both trees.
- Existing programs are byte-identical: 136 packages, 343 types, no difference in TypeID or persisted amino bytes between the trees.
- CI is green on `093c32be0`. The 10 `TestFiles` failures seen locally are go1.26.5 `go/types` wording against a repo pinned to 1.25.9, identical on both trees, and no comparison here rests on them.

## Existing threads

- davd-gzl, round 1 on `61fc396e4`, `CHANGES_REQUESTED`, still standing: the empty-base corruption for `type T2 T1`. Resolved by this head, and the resolution is confirmed. The verdict here is a different defect one hop further along the same class.
- davd-gzl, inline suggestion on `preprocess.go`, applied by the author verbatim in `114c6db`.

## Open questions

- The `false` return from `fillTypeInPlace` on a kind mismatch would silently restore the round-1 stale-base state. Zero mismatches in 3,182 calls and no reachable construction found, so it is latent only; not posted.
- `gno test -p` and any future parallel preprocessing widen the race surface beyond the constructed case. Worth a `-race` job in CI regardless of this PR; not this PR's to add.
