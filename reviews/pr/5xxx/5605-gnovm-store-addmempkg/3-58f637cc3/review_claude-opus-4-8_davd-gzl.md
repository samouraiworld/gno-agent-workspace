# PR #5605: fix(gnovm/store): body-first AddMemPackage ordering + fail-fast IterMemPackage

URL: https://github.com/gnolang/gno/pull/5605
Author: moul | Base: master | Files: 2 | +208 -40
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: `58f637cc3` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5605 58f637cc3`

Round 3 (deep, multi-angle): head advanced `9452fbcc6` → `58f637cc3` by a master merge only; patch-id of the two PR files is unchanged from round 2, so the code is identical. Re-examined under red-team / blue-team / correctness lenses plus a critic pass. Round-2 verdict and findings carry; one warning is extended with the durability-layer analysis, one nit gains a concrete failure mode.

**TL;DR:** When the node persists a smart-contract package it writes three things across two databases (the package body, an index entry, and a counter). The old write order could, after a hard crash, leave the counter pointing at a package whose body was never written, and the node would then crash-loop on restart. This PR reorders the writes and makes the restart-time reader fail loudly on any leftover inconsistency instead of crashing with a useless stack trace.

**Verdict: APPROVE** — Write reordering is correct, gas-neutral, and consensus-safe; the fail-fast reader is the right call and its panic is live on the production restart path. No code blocker. The one thing worth fixing before merge is the `AddMemPackage` comment, which oversells the crash-consistency guarantee and points at a consumer-side nil skip that does not exist.

## Summary

`defaultStore.AddMemPackage` previously wrote counter → index → body across two substores (baseStore holds counter + index, iavlStore holds the body). A crash between substore commits could leave a counter naming a package whose body is absent; on restart `IterMemPackage` yielded `nil` into `ParseMemPackage`, which SIGSEGV'd and crash-looped the node (PR body reports replay walltime ~12 min → ~36 s post-fix). The PR reorders to body → index → counter and converts `IterMemPackage` from a lazy goroutine+channel iterator that could yield `nil` into an eager O(N) load that panics at the call site, naming the slot and telling the operator to replay from a clean snapshot. `nil` leaves the type's value set for the one production consumer.

```
old: ctr→idx→body    crash ⇒ counter says "package N exists", body absent → SIGSEGV on restart
new: body→idx→ctr    counter (the visibility gate) is the last write
                     fail-fast panic is the real safety net if a divergence is still observed
```

## Glossary

- `baseStore` — dbadapter-backed flat KV substore (`pkgidx:counter`, `pkgidx:<N>` → path). Writes go straight to its DB. Not versioned.
- `iavlStore` — iavl-tree versioned substore (`pkg:<path>` → amino body). Commits as a tree version.
- `AddMemPackage` — append-only writer: writes body, index slot, counter. Runs on the tx hot path (`AddPackage`), not only at restart.
- `IterMemPackage` — restart-time replay iterator; now eager + fail-fast. Sole production caller: `Machine.PreprocessAllFilesAndSaveBlockNodes` at boot.
- `pkgGetter` — optional stdlib-fabrication hook on the store; can re-materialise a missing body via `RunMemPackage`. `nil` on the production restart path.

## Fix

`AddMemPackage` writes the iavl body first ([`store.go:984-985`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L984-L985) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L984-L985)), reads the counter without committing it, writes the index slot at `ctr+1` ([`store.go:991-1002`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L991-L1002) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L991-L1002)), then bumps the counter as the last visible write ([`store.go:1003-1005`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1003-L1005) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1003-L1005)). `incGetPackageIndexCounter` is deleted (no other caller). `IterMemPackage` loads eagerly on the caller's goroutine and panics on a missing index slot ("corrupt package index", [`store.go:1117-1121`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1117-L1121) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1117-L1121)) or a missing body ("substore divergence", [`store.go:1124-1129`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1124-L1129) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1124-L1129)), then drains a buffered channel of the validated slice ([`store.go:1132-1137`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1132-L1137) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1132-L1137)).

Verified on `58f637cc3`: the new ordering writes the same key/value multiset as the deleted `incGetPackageIndexCounter` path (one counter Get, one counter Set, one index Set, one body Set), so per-tx gas and committed state are byte-identical to before; the reorder cannot fork consensus. The "substore divergence" panic is reachable in production: the restart store at [`keeper.go:149`](https://github.com/gnolang/gno/blob/58f637cc3/gno.land/pkg/sdk/vm/keeper.go#L149) · [↗](../../../../../.worktrees/gno-review-5605/gno.land/pkg/sdk/vm/keeper.go#L149) sets no `pkgGetter`, so a body-less slot reaches the panic rather than being silently fabricated. The self-healing retry path (`ctr = prevCtr+1` overwrites a dangling slot N+1) was traced through the code and holds.

## Critical (must fix)

None.

## Warnings (should fix)

- **[comment claims a crash guarantee the storage layer doesn't give, and names a defense that doesn't exist]** [`gnovm/pkg/gnolang/store.go:961-976`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L961-L976) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L961-L976) — The write-order comment lays out a per-statement crash taxonomy ("crash after body, before index: orphaned body, harmless"; "crash after index, before counter: self-heals on retry") and then states "WAL flush ordering across substores is still non-deterministic," which negates the taxonomy it just gave. It also says "the consumer-side nil skip must be retained as belt-and-braces," but no such skip exists.
  <details><summary>details</summary>

  Within a block the three writes never tear at statement boundaries: during `DeliverTx` the substores are cache-wrapped (baseapp `MultiCacheWrap`), each substore's cache flushes to its DB in one atomic batch, and a mid-block crash discards the whole uncommitted block and replays it. The only real crash window is at block `Commit`, where `commitStores` commits the base and iavl substores in non-deterministic Go-map order with no batch spanning both ([`tm2/pkg/store/rootmulti/store.go:498`](https://github.com/gnolang/gno/blob/58f637cc3/tm2/pkg/store/rootmulti/store.go#L498) · [↗](../../../../../.worktrees/gno-review-5605/tm2/pkg/store/rootmulti/store.go#L498)). So the index (baseStore) can become durable before the body (iavlStore) regardless of the in-code statement order, and the comment's "orphaned body is harmless" guarantee does not hold at the durability layer. The body-first ordering is a cheap best-effort posture; the actual safety net is `IterMemPackage`'s fail-fast panic, exactly as the PR body frames it ("body-first ordering plus loud failure is the correct posture"). Separately, the sole consumer [`machine.go:329-331`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/machine.go#L329-L331) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/machine.go#L329-L331) pipes `mpkg` straight into `FilterMemPackage`/`ParseMemPackage` with no nil guard, so the "consumer-side nil skip" the comment says must be retained is not there. A maintainer reading this comment learns a durability model the layer doesn't provide and a defense that doesn't exist. Fix: rewrite the comment to state the real posture, that the statement ordering is best-effort and the fail-fast panic in `IterMemPackage` is the guarantee, and drop the consumer-side-nil-skip sentence and the `b15ffde6e` reference (next finding).
  </details>

- **[commit ref unreachable from a gnolang/gno clone]** [`gnovm/pkg/gnolang/store.go:972-974`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L972-L974) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L972-L974) — `"see commit b15ffde6e and the defensive consumer in machine.go"` cites a hash that lives only on the `aeddi/chain/test13-rc*` branches, never upstreamed; from a gnolang/gno clone `git show b15ffde6e` fails with "unknown revision". The defensive consumer it describes is also gone. Raised round 1 and round 2; unchanged.
  <details><summary>details</summary>

  `git branch -r --contains b15ffde6e` resolves only to aeddi remotes, no `origin/*`. Fix: drop the reference or replace it with a self-contained note, e.g. "the previous counter→index→body ordering could leave a counter pointing at a missing body after a crash; see the PR #5605 description for the root-cause analysis." Folds into the comment rewrite above.
  </details>

## Nits

- [`gnovm/pkg/gnolang/store.go:1109`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1109) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1109) and [`store.go:929`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L929) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L929) — A corrupt counter value bypasses the friendly panic contract the PR establishes. The counter is read with signed `strconv.Atoi`; a negative value makes `make([]*std.MemPackage, 0, ctr)` ([`store.go:1113`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1113) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1113)) panic with an opaque "makeslice: cap out of range" (a `runtime.errorString`, not the `string` the new tests assert), and `NumMemPackages` returns the negative count so [`keeper.go:152`](https://github.com/gnolang/gno/blob/58f637cc3/gno.land/pkg/sdk/vm/keeper.go#L152) · [↗](../../../../../.worktrees/gno-review-5605/gno.land/pkg/sdk/vm/keeper.go#L152)'s `> 0` guard silently skips preprocessing instead of failing loud. Unreachable in practice (the counter is only ever written unsigned), so it stays a nit; extends round 2's Atoi/FormatUint asymmetry with the concrete failure. Fix: read with `strconv.ParseUint(s, 10, 64)` in both functions and wrap the parse error with the same "replay from a clean snapshot" message.

- [`gnovm/pkg/gnolang/store_test.go:281-282`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store_test.go#L281-L282) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L281-L282) — `"Verified by snapshotting each substore between calls and asserting the order of key appearance"` is false: `TestAddMemPackage_WriteOrderIsBodyFirst` snapshots nothing and asserts no ordering (see Missing Tests). A reader chasing the snapshot logic finds none. Fix: delete the sentence, or make the test do what it claims.

- [`gnovm/pkg/gnolang/store_test.go:7`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store_test.go#L7) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L7) + [`store_test.go:331`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store_test.go#L331) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L331) — `_ = strconv.Itoa // keep the import used` is the entire use of the `strconv` import. After `incGetPackageIndexCounter` was dropped no test needs strconv; delete both lines.

- [`gnovm/pkg/gnolang/store.go:1113-1136`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1113-L1136) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1113-L1136) — `IterMemPackage` builds `pkgs []*std.MemPackage`, then a buffered channel of `len(pkgs)`, drains the slice into it, closes, returns — every entry is held twice for the iteration. The channel signature only buys API stability for the single non-test caller; `iter.Seq[*std.MemPackage]` or a plain slice would be cleaner. Out of scope; flagging only.

## Missing Tests

- **[the headline write-order test does not test write order]** [`gnovm/pkg/gnolang/store_test.go:283-332`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store_test.go#L283-L332) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store_test.go#L283-L332) — `TestAddMemPackage_WriteOrderIsBodyFirst` asserts only the final post-state (body present, counter=1, index→path) plus a round-trip; it never records the `Set` call sequence. A regression back to counter→index→body still passes it.
  <details><summary>details</summary>

  The load-bearing invariant of the PR is the per-call write ordering, not the end state. Confirmed behaviorally: swapping the body `Set` to after the counter bump (restoring the old order) leaves `TestAddMemPackage_WriteOrderIsBodyFirst` green. Adversarial regression-anchor written for this review: [`tests/addmempkg_write_order_test.go`](tests/addmempkg_write_order_test.go) wraps both substores with a recorder and asserts iavl body → base index → base counter; it passes on `58f637cc3` and fails with "REGRESSION: body ... must be written before index ..." when the order is flipped. Fix: add a recorder-based ordering assertion (or fold one into the existing test) so the invariant is pinned.
  </details>

- **[self-healing retry uncovered]** [`gnovm/pkg/gnolang/store.go:967-970`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L967-L970) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L967-L970) — The comment promises that after a crash-between-index-and-counter, a retried `AddMemPackage` recomputes `ctr = prevCtr+1` and overwrites the dangling slot. No test pins it. Forge an index slot at N+1 with the counter left at N, call `AddMemPackage` again, assert the slot was overwritten with the new path and the counter is N+1 (not N+2), then iterate to confirm no orphan surfaces. Round 1 and round 2 both flagged it.

- **[`NumMemPackages` vs `IterMemPackage` divergence under corruption]** [`gnovm/pkg/gnolang/store.go:923-935`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L923-L935) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L923-L935) — `NumMemPackages` reads only the counter and never validates index/body, so under the corruption that `IterMemPackage` now panics on it returns a dangling count silently. Worth a test pinning the divergent behaviour so a future "make them agree" change doesn't break the boot guard at `keeper.go:152` unnoticed.

## Suggestions

- [`gnovm/pkg/gnolang/store.go:1088-1102`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1088-L1102) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1088-L1102) — The "substore divergence" panic only fires when `pkgGetter == nil`. On stores with a `pkgGetter` (`gno lint`, test imports) a body-less slot is silently re-materialised via `GetMemPackage` → `RunMemPackage` ([`store.go:1032-1036`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1032-L1036) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1032-L1036)) — which itself calls `AddMemPackage`, mutating the counter mid-iteration. Production restart sets no `pkgGetter`, so this is not a node-safety issue, but `IterMemPackage` reading via `GetMemPackage` makes the panic non-deterministic across store configs. Consider reading the body via the raw `iavlStore` inside `IterMemPackage` so the divergence panic is independent of `pkgGetter`, and/or a doc line noting the dependency.
  <details><summary>details</summary>

  The two corrupt-state tests construct the store with `NewStore(nil, ...)` (nil pkgGetter), so they exercise the panic path; a node or CLI with a pkgGetter would take the fabrication path instead. Pinning both configs would make the contract explicit.
  </details>

- [`gnovm/pkg/gnolang/store.go:1088-1098`](https://github.com/gnolang/gno/blob/58f637cc3/gnovm/pkg/gnolang/store.go#L1088-L1098) · [↗](../../../../../.worktrees/gno-review-5605/gnovm/pkg/gnolang/store.go#L1088-L1098) — The doc explains the eager load but not that the panic now surfaces synchronously on the *caller's* goroutine (the previous design panicked inside an orphan producer goroutine, unrecoverable in tests). The new tests rely on `recover()` catching it at the call site. One line — "panic propagates synchronously to the caller" — closes the loop.

## Open questions

- The replay speedup (~12 min → ~36 s): how much is from eliminating crash-retry duplicate work (the ordering fix) vs removing the producer-goroutine/channel scheduling in `IterMemPackage` (the eager-load fix)? Nothing in this diff changes steady-state write count, so the win must be retry-avoidance, but it's not benchmarked. Asked rounds 1-2; not posted, sizing-only.
- The PR body says "a dedicated ADR will follow." Without it the rationale (panic-on-corruption over best-effort skip) lives only in commit messages. Not posted; contribution-policy, out of review scope.
