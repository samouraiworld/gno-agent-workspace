# PR [#6000](https://github.com/gnolang/gno/pull/6000): docs: complete the effective-gno roadmap topics

URL: https://github.com/gnolang/gno/pull/6000
Author: davd-gzl | Base: master | Files: 1 | +560 -32
Reviewed by: davd-gzl | Model: claude-fable-5 (deep, self-review) | Commit: 94f259ec (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6000 94f259ec`

Self-review of own PR before un-drafting. A 2-round adversarial loop (accuracy, consistency, prose lenses, then a combined critic) ran pre-push and its findings are already folded in; this round adds the compile check on the doc's code samples.

**TL;DR:** Effective Gno ended with a TODO comment listing two dozen topics that were never written. This PR writes all of them, from file layout and versioning through gas, oracles, forking, and the frame stack, and removes the TODO block.

**Verdict: APPROVE** — docs-only, every named API and path verified against master, all embedded samples compile; ready to un-draft (0 findings).

## Summary
The guide's trailing TODO block tracked by [issue #5031](https://github.com/gnolang/gno/issues/5031) is replaced by 20 sections. New code samples use the current interrealm API: `cur realm` parameters, `cross(cur)`, `cur.Previous()` behind a `cur.IsCurrent()` guard. Quarantined packages (pausable, minisocial, entropy, subscription, chess) are not referenced; deployed substitutes are (`r/sys/validators` for versioning, `atomicswap` for derived state, gnorkle + ghverify for oracles). Two small rider fixes to existing text: the avl section's map bullet and code comment claimed unspecified iteration order where Gno guarantees insertion order.

## Fix
None. Additive docs plus the two-line map-order correction.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None (docs-only).

## Suggestions
None.

## Verified
- All three embeddable sample groups compile at the reviewed sha: copied verbatim into a scratch realm and run through `go run ./gnovm/cmd/gno lint` (typecheck confirmed live: it rejects an undefined symbol). Covers the [subscription sample](https://github.com/gnolang/gno/blob/94f259ec/docs/resources/effective-gno.md?plain=1#L1336-L1360) · [↗](../../../../../.worktrees/gno-review-6000/docs/resources/effective-gno.md#L1336) (zero-value `avl.Tree`, single-value `Get`, `time.Time` expiry), the [state-machine entrypoint](https://github.com/gnolang/gno/blob/94f259ec/docs/resources/effective-gno.md?plain=1#L1270-L1301) · [↗](../../../../../.worktrees/gno-review-6000/docs/resources/effective-gno.md#L1270) (crossing signature, `IsCurrent` guard, `address` comparison), and the [Must* wrapper](https://github.com/gnolang/gno/blob/94f259ec/docs/resources/effective-gno.md?plain=1#L122-L141) · [↗](../../../../../.worktrees/gno-review-6000/docs/resources/effective-gno.md#L122).
- The run-script form `func main(cur realm)` + `Increment(cross(cur))` matches the in-tree txtar precedent [`params_valset_keeprunning_race.txtar`](https://github.com/gnolang/gno/blob/94f259ec/gno.land/pkg/integration/testdata/params_valset_keeprunning_race.txtar#L78) · [↗](../../../../../.worktrees/gno-review-6000/gno.land/pkg/integration/testdata/params_valset_keeprunning_race.txtar#L78) and [`counter.Increment(_ realm)`](https://github.com/gnolang/gno/blob/94f259ec/examples/gno.land/r/demo/counter/counter.gno#L7) · [↗](../../../../../.worktrees/gno-review-6000/examples/gno.land/r/demo/counter/counter.gno#L7).
- Storage-deposit paragraph checked against [storage-deposit.md](https://github.com/gnolang/gno/blob/94f259ec/docs/resources/storage-deposit.md?plain=1#L16-L17) · [↗](../../../../../.worktrees/gno-review-6000/docs/resources/storage-deposit.md#L16) (lock on store, refund on delete, cleanup reward) and gas numbers against [gas-fees.md](https://github.com/gnolang/gno/blob/94f259ec/docs/resources/gas-fees.md?plain=1#L156-L158) · [↗](../../../../../.worktrees/gno-review-6000/docs/resources/gas-fees.md#L156).
- `cur.IsCurrent()` semantics match [uverse.go](https://github.com/gnolang/gno/blob/94f259ec/gnovm/pkg/gnolang/uverse.go#L584) · [↗](../../../../../.worktrees/gno-review-6000/gnovm/pkg/gnolang/uverse.go#L584); the `unsafe.PreviousRealm` caveat matches the SECURITY comment in [unsafe.gno](https://github.com/gnolang/gno/blob/94f259ec/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L21) · [↗](../../../../../.worktrees/gno-review-6000/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L21).
- Docs linter (`make -C docs lint`): zero findings against the file; all in-page anchors resolve; relative links (`../CONSTITUTION.md`, `../../examples/...`) target existing paths.
- CI green at 94f259ec: 10 checks pass, 2 skipped (deploy, save-pr-number; normal for fork drafts).

## Open questions
- The untouched body still teaches the removed API (`runtime.PreviousRealm()` ×7, `runtime.OriginCaller()` ×3, `banker.OriginSend()` ×3, one-arg `NewBanker`, string-id `grc20.NewToken`) right next to the new sections. All covered by the in-flight unsafe-split docs branch (`fix/docs-runtime-unsafe`, worktree gno-fix-5939); whichever PR lands second rebases. Not posted: deliberate scope split, noted in the PR description.
- The `TODO: explain when to use runtime.OriginCaller` line is intentionally kept; the unsafe-split branch replaces that exact line. Not posted: stated in the PR description.
