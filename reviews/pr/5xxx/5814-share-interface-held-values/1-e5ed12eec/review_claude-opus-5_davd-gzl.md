# PR [#5814](https://github.com/gnolang/gno/pull/5814): perf(gnovm): share interface-held values when copying arrays

URL: https://github.com/gnolang/gno/pull/5814
Author: thehowl | Base: master | Files: 5 | +397 -3
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: e5ed12eec (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5814 e5ed12eec`

## Overview

When a gno array is copied, the VM used to duplicate every element down to the
last field. For an array whose elements are interfaces that duplication is
invisible work: the language gives no way to write into a value once it sits
inside an interface, so the duplicate can never diverge from the original. This
change makes the copy share those elements instead. The array itself is still
fresh, so writing a whole new element into one copy leaves the other alone;
only the boxed contents are now held in common. Because the duplicates were
also being billed, allocation gas for the affected shapes drops, which is why
one gas golden in the test corpus moves.

**Verdict: COMMENT** — every program `go/types` accepts behaves identically before and after, and the change also removes a merge-base inconsistency where the same source persisted two different layouts depending on read order; the one Warning is that the VM never enforced the addressability rule the diff's comment relies on, which predates the branch and is now louder (1 Warning, 2 Missing tests, 1 Nit, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/values.go:463-472`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L463-L472) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L463-L472) — the whole change and its stated premise. Drop [`tests/zz_iface_array_alias_matrix.gno`](tests/zz_iface_array_alias_matrix.gno) into `gnovm/tests/files/` and run it at this sha and at 754780601: six of its ten cases flip.
- [`gnovm/tests/files/gas/nested_alloc.gno:12`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/gas/nested_alloc.gno#L12) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/gas/nested_alloc.gno#L12) — the consensus-visible number. `go test -run 'TestFiles/gas/nested_alloc.gno$' ./gnovm/pkg/gnolang/` reproduces 17013961 here and 8559690224 at the merge base.

## Summary

[`ArrayValue.Copy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L454) walked every element through [`TypedValue.Copy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L1345), which deep-copies exactly three carriers: `BigintValue`, `*ArrayValue` and `*StructValue`. Those three are the change's entire surface, since pointers, slice headers and maps already fell through the `default` arm and were shared. For an array whose element kind is `InterfaceKind` the new branch runs [`copy(cp.List, av.List)`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L472) instead, so `x = [1]any{x}` stops being quadratic.

The soundness argument in the comment has three premises. Two hold in the VM: [`TypedValue.Assign`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L2040) copies on entry, and extraction assigns and therefore copies again. The third, that interface-held values are not addressable, is enforced by `go/types` and not by the VM. On chain that is enough: both [`AddPackage`](https://github.com/gnolang/gno/blob/e5ed12eec/gno.land/pkg/sdk/vm/keeper.go#L654) and [`Run`](https://github.com/gnolang/gno/blob/e5ed12eec/gno.land/pkg/sdk/vm/keeper.go#L1035) type-check before the machine sees the package, and [`gno lint`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/cmd/gno/lint.go#L425) reports it. `gno run` does not, and executes the write.

The measurement that most argues for the change is not in the description. At the merge base the deep copy only happened when the element was a live object; an element still sitting in the store as a `RefValue` fell through `default` and was shared. So the same realm code persisted one shared object or two owned ones depending on whether anything had read the element earlier in the transaction. `tests/zrealm_iface_array_share_stored.gno` and its untouched twin differ only by one `println`, and at 754780601 they persist different object graphs. At e5ed12eec they persist the same one.

## Examples

Ten shapes, each an array of interface copied and then written through the copy at the illegal position `go/types` rejects. Values read back as `source copy`.

| element held in the interface slot | 754780601 | e5ed12eec |
|---|---|---|
| `int` | `1 1` | `1 1` |
| `S{F int}` | `1 9` | `9 9` |
| `*S` | `9 9` | `9 9` |
| `[]int` | `9 9` | `9 9` |
| `map[string]int` | `9 9` | `9 9` |
| `[2]int` | `1 9` | `9 9` |
| `W{A [2]int}` | `1 9` | `9 9` |
| `T` in a named interface `I` | `1 9` | `9 9` |
| `S` reached through a struct field copy | `1 9` | `9 9` |
| `S` written through a by-value parameter | `1` | `9` |

## Benchmarks / Numbers

Measured on this machine, `go1.25.9`, merge base 754780601 against e5ed12eec, both binaries built from the same worktree with only the [`values.go`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L463-L472) hunk swapped.

| measurement | 754780601 | e5ed12eec |
|---|--:|--:|
| `gas/nested_alloc.gno` gas | 8,559,690,224 | 17,013,961 |
| `gas/nested_alloc.gno` wall | 102.96s | 0.04s |
| [`recurse1.gno`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/recurse1.gno#L5-L7) wall | 107.54s | 0.02s |
| `TestFiles` user CPU | 569.95s | 359.84s |
| `TestFiles` wall | 148.67s | 130.53s |
| 100,000 copies of `[8]any` of 4-field structs | 1.42s | 0.51s |

The suite CPU delta is **210.11s** and those two filetests are **210.50s** of the merge base's CPU, so the whole suite win is the two files built to exercise the quadratic chain. Running the merge-base VM against this branch's goldens moves 4 filetests out of 2454: `gas/nested_alloc.gno` and the three the branch adds.

## Warnings (should fix)

- **[unenforced premise]** [`gnovm/pkg/gnolang/values.go:466`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L466) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L466) — the addressability rule the sharing rests on lives in `go/types`, and the VM executes the write it forbids, now into both copies.
  <details><summary>details</summary>

  The comment reads as a statement about the VM. It is a statement about the type checker. Feeding the same program to [`gno run`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/cmd/gno/run.go#L185), which never calls [`TypeCheckMemPackage`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/gotypecheck.go#L193), prints `orig: 1 copy: 9` at 754780601 and `orig: 9 copy: 9` at e5ed12eec, with no diagnostic on either. A developer who writes `arr2[0].(S).F = 9` used to lose the write and now silently corrupts the array they copied from. Nothing reaches a transaction, since [`AddPackage`](https://github.com/gnolang/gno/blob/e5ed12eec/gno.land/pkg/sdk/vm/keeper.go#L654) and [`Run`](https://github.com/gnolang/gno/blob/e5ed12eec/gno.land/pkg/sdk/vm/keeper.go#L1035) both type-check, which is why this does not block. The VM already refuses the sibling construct: a pointer-receiver method call on an interface-held value reaches [`panicIllegalPointerLHS`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/machine.go#L2904). Fix: reject the assignment in the same place, or say in the comment that the guarantee comes from the type checker.
  </details>

## Nits

- **[stale figures]** [`gnovm/tests/files/gas/nested_alloc.gno:12`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/gas/nested_alloc.gno#L12) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/gas/nested_alloc.gno#L12) — the description's `IMPORTANT` callout quotes 8,559,690,088 and 17,013,825; the golden and the merge-base run give 8,559,690,224 and 17,013,961, both 136 higher, after [926a1390f](https://github.com/gnolang/gno/commit/926a1390f6afaa55a09b4a0e824e6216757f071a) re-derived the golden over the block-pool merge.

## Missing Tests

- **[already-persisted array]** [`gnovm/tests/files/zrealm_iface_array_share.gno:19-20`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/zrealm_iface_array_share.gno#L19-L20) — all three added tests build the element in the same transaction that copies it, so nothing covers the shape realm code actually has.
  <details><summary>details</summary>

  `share.gno` creates the element in `main`, and `share_drop_one.gno` and `share_drop_both.gno` create it in `init` and only drop it in `main`. None copies an array that was already in the store. That is the case where the merge base disagrees with itself: an element still held as a `RefValue` was shared there too, and only a prior read turned it into a live object and triggered the deep copy. [`tests/zrealm_iface_array_share_stored.gno`](tests/zrealm_iface_array_share_stored.gno) is that test, passing here and failing at 754780601 with a second 216-byte object where this branch escapes the first.
  </details>

- **[realm boundary]** [`gnovm/tests/files/zrealm_iface_array_share.gno:1`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/tests/files/zrealm_iface_array_share.gno#L1) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/tests/files/zrealm_iface_array_share.gno#L1) — no test hands the array to another realm, where sharing takes the element out of its owner tree for good.
  <details><summary>details</summary>

  Passing a `[1]any` by value into [`crossrealm_b.SetObject`](https://github.com/gnolang/gno/blob/e5ed12eec/examples/gno.land/r/tests/vm/crossrealm_b/crossrealm.gno#L40) leaves the callee's persisted array holding a `RefValue` into the caller's object, which loses its `OwnerID` and goes to `RefCount 2`. Escape is one-way, so a later drop back to `RefCount 1` leaves it escaped, as `share_drop_one.gno` already pins within one realm. [`tests/zrealm_iface_array_share_crossrealm.gno`](tests/zrealm_iface_array_share_crossrealm.gno) pins the cross-realm layout; at 754780601 the callee gets its own 215-byte object instead. Authority is unchanged either way, because [`getDeclaredPkgID`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L622) stamps the merge base's fresh copy with the declaring realm's `PkgID` too, so [`IsReadonlyBy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/ownership.go#L461) answers the same on both sides.
  </details>

## Suggestions

- **[the shape on chain]** [`gnovm/pkg/gnolang/uverse.go:880`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/uverse.go#L880) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/uverse.go#L880) — `copy` and `append` over `[]any` still deep-copy every element, and that is the shape realm code uses.
  <details><summary>details</summary>

  Both builtins reach [`TypedValue.Copy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L1345) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L1345) by different routes and so keep deep-copying interface-held structs: `append` through [`unrefCopy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L1364) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L1364) at four call sites, `copy` through [`Assign2`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/uverse.go#L1213) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/uverse.go#L1213). After this branch `dst = src` shares and `copy(dst, src)` does not, while Go shares in both. Under `examples/gno.land` and `gnovm/stdlibs`, 322 lines mention `[]any` or `[]error` and not one declares a fixed-size array of an interface element type, so the reach of the branch as written is the two filetests and future code. The description names [`StructValue.Copy`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/values.go#L611) · [↗](../../../../../.worktrees/gno-review-5814/gnovm/pkg/gnolang/values.go#L611) as follow-up; these five call sites belong on the same list.
  </details>

## Verified

- Sharing is unobservable to any program the toolchain accepts. Ten element shapes copied and written through at the illegal position, run at both shas: the six that flip are exactly the ones `go/types` rejects, and `gno lint` reports every one of them.
- The three added filetests bite. Restoring the merge base's `values.go` in the worktree fails all three on their `Realm:` goldens, and restoring the branch's passes them.
- The branch moves one pre-existing golden and no more. The merge-base binary run over the whole `TestFiles` corpus at this branch's goldens fails 4 of 2454, three being the tests the branch adds.
- Invariant catalog walked. Gas is consensus-affecting and the description leads with it; realm state safety is covered by the three goldens plus the stored and cross-realm fixtures here; determinism improves, since the merge base's choice between sharing and copying depended on whether the element had been read; no global state, no caller-identity code, no coin or banker path, no new VM fault.

## Open questions

- `arr[0].(S).Bump()` with a pointer receiver raises a bare Go panic from [`panicIllegalPointerLHS`](https://github.com/gnolang/gno/blob/e5ed12eec/gnovm/pkg/gnolang/machine.go#L2904) that escapes `runOnce` uncatchable by gno `recover()`. It reproduces identically at 754780601, so it belongs to whichever change closes the VM-side addressability gap rather than to this one.
