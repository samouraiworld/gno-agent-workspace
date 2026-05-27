# PR #5605: fix(gnovm/store): body-first AddMemPackage ordering + fail-fast IterMemPackage

URL: https://github.com/gnolang/gno/pull/5605
Author: moul | Base: master | Files: 2 | +208 -40
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5605 9452fbcc6` (then `gh -R gnolang/gno pr checkout 5605` inside it)

Verdict: APPROVE — Write-ordering fix is correct and the fail-fast contract is the right call. Two cosmetic blockers from round 1 still stand (stale comment + phantom commit ref + unused-import hack) and one missing regression-anchor adversarial test would harden a load-bearing invariant; none are merge blockers.

## Summary

`defaultStore.AddMemPackage` previously wrote counter→index→body across two substores (baseStore for index/counter, iavlStore for body). A SIGKILL during the gap between substore commits left a counter pointing at an index pointing at no body; on restart `IterMemPackage` yielded nil into `ParseMemPackage` and SIGSEGV'd, crash-looping the node (replay walltime reported in the PR body dropped from ~12 min to ~36 s post-fix). PR reorders to body→index→counter so every crash window resolves to either (i) orphaned-but-invisible iavl body, (ii) dangling index slot that gets overwritten on retry, or (iii) consistent. `IterMemPackage` switches from goroutine+channel lazy iteration to eager O(N) load with loud panic-at-source on observed substore divergence — `nil` is no longer in the type's value set for downstream consumers.

```
old: ctr→idx→body    crash anywhere ⇒ counter says "package N exists", body absent → SIGSEGV
new: body→idx→ctr    crash between body+idx ⇒ orphan body (invisible)
                     crash between idx+ctr  ⇒ slot at N+1 silently overwritten on retry
                     crash after ctr        ⇒ consistent
```

## Glossary

- `baseStore` — dbadapter-backed flat KV substore; holds `pkgidx:counter` and `pkgidx:<N>` → path entries. NOT versioned.
- `iavlStore` — iavl-tree-backed versioned substore; holds `pkg:<path>` → amino-marshalled MemPackage body.
- `AddMemPackage` — append-only writer; bumps counter, writes path index entry, writes body.
- `IterMemPackage` — restart-time replay iterator; now eager + fail-fast.
- `pkgGetter` — optional stdlib fabrication hook on the store; can re-materialise a missing body via `RunMemPackage` if set.

## Fix

`AddMemPackage` writes the iavl body first ([`gnovm/pkg/gnolang/store.go:929-930`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L929-L930) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L929-L930)), then computes the next counter via a read-without-write ([`store.go:934-946`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L934-L946) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L934-L946)), writes the index slot at `ctr+1`, then bumps the counter as the last visible operation ([`store.go:948-950`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L948-L950) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L948-L950)). `incGetPackageIndexCounter` is gone. `IterMemPackage` validates eagerly: missing slot ⇒ panic with "corrupt package index, slot N, replay"; missing body ⇒ panic with "substore divergence, slot N, replay" ([`store.go:1062-1074`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L1062-L1074) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1062-L1074)). The channel is preserved for API compatibility but is now a buffered drain of a fully-validated slice ([`store.go:1077-1082`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L1077-L1082) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1077-L1082)).

## Critical (must fix)

None.

## Warnings (should fix)

- [stale comment, internally contradicts new design] [`gnovm/pkg/gnolang/store.go:917-921`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L917-L921) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L917-L921) — `AddMemPackage` doc-block still says the consumer-side nil skip "must be retained as belt-and-braces" and references "the defensive consumer in machine.go" — both wrong post-99e762bb4. Raised in round 1; unchanged.
  <details><summary>details</summary>

  The phrase "WAL flush ordering across substores is still non-deterministic, so the consumer-side nil skip must be retained as belt-and-braces" presupposes a consumer that skips nil. The consumer ([`gnovm/pkg/gnolang/machine.go:283-307`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/machine.go#L283-L307) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/machine.go#L283-L307)) has no nil-skip: it pipes `mpkg` straight into `MPFProd.FilterMemPackage` and `m.ParseMemPackage`. The current design relies on `IterMemPackage` panicking before any consumer sees a nil; the comment describes the abandoned earlier draft. A reviewer onboarding to this code reads it and concludes the design tolerates nil consumers — false. Fix: rewrite the trailing line to "WAL flush ordering across substores can still produce divergence; `IterMemPackage` panics on it, refusing to feed nil to consumers."
  </details>

- [phantom commit ref unreachable from gnolang/gno] [`gnovm/pkg/gnolang/store.go:917-919`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L917-L919) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L917-L919) — `"see commit b15ffde6e and the defensive consumer in machine.go"` cites a hash that exists only on `aeddi/chain/test13-rc5` and `aeddi/chain/test13-rc6` (the original test13-rc series, not upstreamed); from a gnolang/gno clone `git show b15ffde6e` resolves only because the local clone happens to have the aeddi remote — public readers will see "unknown revision".
  <details><summary>details</summary>

  Confirmed via `git branch -r --contains b15ffde6e` (only aeddi remotes, no `origin/*`). The defensive consumer described in that commit's message is also no longer present in machine.go (the design moved to panic-at-source). Either drop the reference entirely or replace with a descriptive note: "the previous counter→index→body ordering left a counter pointing at a missing body across SIGKILL; see PR #5605 description for the root-cause analysis."
  </details>

## Nits

- [`gnovm/pkg/gnolang/store_test.go:331`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store_test.go#L331) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L331) — `_ = strconv.Itoa` plus the strconv import on [`store_test.go:7`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store_test.go#L7) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L7) is the entire usage. After dropping `incGetPackageIndexCounter` from production, no test calls strconv. Delete the import + the comment-line.

- [`gnovm/pkg/gnolang/store_test.go:281-282`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store_test.go#L281-L282) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L281-L282) — `"Verified by snapshotting each substore between calls"` is aspirational; the test only checks pre/post-conditions. A reader chasing the snapshot expectation finds nothing.

- [`gnovm/pkg/gnolang/store.go:1048-1082`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L1048-L1082) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1048-L1082) — `IterMemPackage` builds `pkgs []*std.MemPackage`, then makes a buffered channel of `len(pkgs)`, drains the slice into the channel, closes, returns. The slice and the channel hold every entry twice for the duration of the iteration. The channel signature only buys API stability for the single non-test caller — `iter.Seq[*std.MemPackage]` or a plain `[]*std.MemPackage` would be cleaner. Out of scope for this PR; flagging only.

- [`gnovm/pkg/gnolang/store.go:937-944`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L937-L944) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L937-L944) vs [`store.go:950`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L950) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L950) — counter is read with `strconv.Atoi` (signed int, max ~9.2×10^18) but written with `strconv.FormatUint(ctr, 10)`. Round 1 raised this. Unreachable in practice (no node will reach 2^63 mempackages) but the asymmetry is cheap to fix: read with `strconv.ParseUint(s, 10, 64)`.

## Missing Tests

- [recorded write-order regression anchor] [`gnovm/pkg/gnolang/store_test.go:283`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store_test.go#L283) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L283) — `TestAddMemPackage_WriteOrderIsBodyFirst` proves the postcondition (all three writes happen and end up consistent) but does not pin the actual call ordering. A future refactor that reverts to counter→index→body would still pass it as long as all three writes ultimately land.
  <details><summary>details</summary>

  The load-bearing invariant of this PR is the per-substore call ordering, not the final state. The fail-fast crash-consistency story only holds while writes happen in body→index→counter order. A regression-anchor test should wrap base/iavl substores with a recorder and assert the observed Set() sequence. Adversarial test written for this review: [`reviews/pr/5xxx/5605-gnovm-store-addmempkg/2-9452fbcc6/tests/addmempkg_write_order_test.go`](tests/addmempkg_write_order_test.go) — passes on `9452fbcc6`; flipping `iavlStore.Set` and `baseStore.Set(idxkey, …)` in `store.go` makes it fail with `REGRESSION: body … must be written before index …`. The existing tests miss this.
  </details>

- [idempotent retry after index-before-counter crash] [`gnovm/pkg/gnolang/store_test.go:283`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store_test.go#L283) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L283) — Round 1 already flagged this. The PR body claims "slot at N+1 with counter still N gets overwritten on next add" but no test pins it: write index slot at ctr+1, leave counter at ctr, call AddMemPackage again, assert slot N+1 was overwritten cleanly and counter is now N+1 (not N+2). The self-healing path is uncovered.

- [`NumMemPackages` vs `IterMemPackage` agreement under corruption] [`gnovm/pkg/gnolang/store.go:868-880`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L868-L880) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L868-L880) — `NumMemPackages` reads only the counter; it never validates index/body. Under the corruption scenarios that `IterMemPackage` now panics on, `NumMemPackages` returns the dangling count silently. Not a bug per se (one call panics, the other returns a stale-but-typed value) but worth a test pinning the divergent behaviour so a future "make NumMemPackages also panic" change doesn't silently break something else.

## Suggestions

- [`gnovm/pkg/gnolang/store.go:1033-1047`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L1033-L1047) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1033-L1047) — Doc comment is good but doesn't mention that the panic now surfaces on the *caller's* goroutine (an important behavioural difference from the previous design, where panics surfaced inside an orphan producer goroutine and were unrecoverable in tests). One line: "panic propagates synchronously to the caller; tests may `recover()` from it" would close the loop.
  <details><summary>details</summary>

  This is also load-bearing: the PR body and the new tests rely on `recover()` surfacing the panic in the test goroutine. Anyone reading just the doc-comment wouldn't know the previous behaviour was different.
  </details>

- [`gnovm/pkg/gnolang/store.go:956-993`](https://github.com/gnolang/gno/blob/9452fbcc6/gnovm/pkg/gnolang/store.go#L956-L993) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L956-L993) — `IterMemPackage` calls `GetMemPackage` which has a `pkgGetter` rescue fallback (calls `GetPackage` to fabricate a missing body via `RunMemPackage`). When the store has `pkgGetter != nil` (genesis bootstrap, `cmd/gno` lint), a "missing body" under an existing index slot is silently fabricated rather than panicking. Production restart paths set `pkgGetter == nil` so the panic fires as intended, but the divergence is not pinned by any test. Worth a comment in `IterMemPackage` doc explaining the implicit dependency: "Assumes `pkgGetter == nil` for the panic path to fire; if a pkgGetter is installed, missing bodies are silently re-materialised."

## Questions for Author

- The replay-speedup ("~12 min → ~36 s") — is the bulk of it from eliminating crash-retry duplicate work (write-ordering fix) or from removing the producer-goroutine + channel scheduling cost in `IterMemPackage` (eager-load fix)? Round 1 asked; still unanswered. Useful for sizing the contribution of each half to the PR's claimed win.
- Was the lack of a nil-guard in `machine.go:PreprocessAllFilesAndSaveBlockNodes` intentional (the panic-at-source contract is the only safety net) or carried over from the earlier draft that paired with #5606? If intentional, the line in the `AddMemPackage` doc-comment about "consumer-side nil skip" needs to go (see Warning above).
- The PR description mentions "A dedicated ADR will follow." Is that ADR landed or planned for a follow-up PR? Without it, the design rationale (why panic-on-corruption over best-effort skip) lives only in the commit message and is hard to discover.
- Duplicate-path semantics: `AddMemPackage(path="X", v1)` then `AddMemPackage(path="X", v2)` produces two index slots both pointing at path "X", and the iavl body is the latest v2 — so `IterMemPackage` yields the *same* mpkg twice. Pre-existing (not introduced by this PR), and `vm/keeper.AddPackage` rejects duplicates at message handling time, but private-package re-deploy hits this path. Worth a test pinning the intended semantics so future de-duplication work has a baseline.
