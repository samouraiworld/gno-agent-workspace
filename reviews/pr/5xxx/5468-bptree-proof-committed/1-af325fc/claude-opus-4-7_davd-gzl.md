# PR #5468: fix(bptree): use lastSaved in immutableForProof to generate proofs from committed state

URL: https://github.com/gnolang/gno/pull/5468
Author: notJoon | Base: feat/jae/bp32tree | Files: 5 | +133 -27
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5468 af325fc` (then `gh -R gnolang/gno pr checkout 5468` inside it)

Verdict: APPROVE — fix is correct and narrowly scoped; the only public proof-API path that touched `MutableTree` returned proofs unverifiable against `MutableTree.Hash()`, and `t.lastSaved` is the right snapshot to anchor them to. One open thread from [@clockworkgr](https://github.com/gnolang/gno/pull/5468#issuecomment-4212077149) about preferring the `GetImmutable(t.version)` pattern deserves a one-line answer in the PR before merge.

## Summary

Before this PR, [`MutableTree.immutableForProof`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof.go#L227-L246) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof.go#L227-L246) built an `ImmutableTree` from `t.root` — the working tree, which may include uncommitted `Set()`/`Remove()` calls. The resulting proof's leaf/inner hashes anchored to a root that didn't match `MutableTree.Hash()` (which returns `lastSaved.Hash()` per [`mutable_tree.go:439-445`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/mutable_tree.go#L439-L445) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/mutable_tree.go#L439-L445)), so any external caller doing `Set(); proof := tree.GetMembershipProof(k); ics23.VerifyMembership(spec, tree.Hash(), proof, ...)` got `false`. The fix switches the snapshot to `t.lastSaved`, returns a new `ErrNoCommittedState` sentinel when no version has been saved, and updates every test that previously verified working-tree proofs against `WorkingHash()` to commit first and verify against `Hash()`. Blast radius is narrow: the production proof path (`Store.Query`) takes [`tree.GetImmutableTree(res.Height)`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/store/bptree/store.go#L303) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/store/bptree/store.go#L303) and never touches `immutableForProof`.

```
before:                            after:
  Set("a")                           Set("a")
  SaveVersion()                      SaveVersion()
  Set("b")    ── dirty root ──>      Set("b")    ── dirty root, but proof anchors to lastSaved ──>
  proof = GetMembershipProof("a")    proof = GetMembershipProof("a")
  VerifyMembership(Hash(), proof)    VerifyMembership(Hash(), proof)
  → false (root mismatch)            → true
```

## Glossary

- `MutableTree.root` — working tree, mutated in place (COW per [`insert.go:14-22`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/insert.go#L14-L22) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/insert.go#L14-L22)) by `Set`/`Remove`.
- `MutableTree.lastSaved` — snapshot pinned by the most recent `SaveVersion()`; matches `Hash()` and the on-disk root.
- `WorkingHash()` — hash of the in-progress tree; diverges from `Hash()` after any uncommitted mutation.
- `immutableForProof` — internal helper that materialises an `ImmutableTree` + value resolver for the `MutableTree.GetMembership/NonMembershipProof` wrappers.

## Fix

[`immutableForProof`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof.go#L227-L246) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof.go#L227-L246) now sources its `root` from `t.lastSaved` and returns `(*ImmutableTree, error)` — `ErrNoCommittedState` when `t.lastSaved == nil`. The two wrappers `MutableTree.GetMembershipProof` and `MutableTree.GetNonMembershipProof` propagate the error. All existing proof tests gained a `tree.SaveVersion()` before verification and switched their root from `WorkingHash()` to `Hash()`. [`TestProof_UsesCommittedState`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof_test.go#L299-L374) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof_test.go#L299-L374) is the new regression test, run in both in-memory and DB-backed variants, asserting (a) `ErrNoCommittedState` pre-save, (b) working hash diverges from committed hash after dirty `Set`, (c) proof for a committed key verifies against the committed root, (d) proof for an uncommitted key fails to generate. The `bptree` package tests pass in the worktree (`go test ./tm2/pkg/bptree/...` → ok 1.885s).

## Critical (must fix)

None.

## Warnings (should fix)

- **[unanswered review thread]** [@clockworkgr](https://github.com/gnolang/gno/pull/5468#issuecomment-4212077149) `tm2/pkg/bptree/proof.go:227` — Why not mirror IAVL's `GetImmutable(t.version)` pattern instead of `lastSaved` + new sentinel error?
  <details><summary>details</summary>

  Both designs land on the same logical state, but the choice has consequences the PR description should record. `GetImmutable(t.version)` for a DB-backed tree triggers `ndb.GetRoot` + `loadNode` (see [`mutable_tree.go:370-386`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/mutable_tree.go#L370-L386) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/mutable_tree.go#L370-L386)) — a disk round-trip on every proof — and would still need an early-return for `t.version == 0` (no version saved yet), which is structurally the same `ErrNoCommittedState` branch. Using `t.lastSaved` directly avoids the DB hit, reuses an in-memory snapshot the `MutableTree` already pins, and keeps the value resolver dispatch (ndb vs memValues) co-located. The trade-off is a divergent code path between the public versioned-proof flow (`Store.Query` → `GetImmutableTree`) and the direct `MutableTree.GetMembershipProof` flow — they no longer share a single source of truth. Fix: add a one-line comment on `immutableForProof` saying "we use lastSaved (not GetImmutable) to skip the DB round-trip; both reference the same committed root", and post that same explanation as a reply to the review comment.
  </details>

## Nits

- [`tm2/pkg/bptree/iavl_proof_ics23_test.go:53`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/iavl_proof_ics23_test.go#L53) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/iavl_proof_ics23_test.go#L53) — `tree.SaveVersion()` return values dropped silently; the helper already returns `error`, fold the save error into it: `if _, _, err := tree.SaveVersion(); err != nil { return nil, nil, err }`.
- [`tm2/pkg/bptree/proof_test.go:320`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof_test.go#L320) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof_test.go#L320) — `if err != ErrNoCommittedState` uses direct comparison; switch to `errors.Is(err, ErrNoCommittedState)` to keep the contract stable if the error ever gets wrapped.
- [`tm2/pkg/bptree/proof_test.go:356`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof_test.go#L356) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof_test.go#L356) — Second `tree.SaveVersion()` drops `(hash, version, err)`; if `SaveVersion` returns an error the test silently continues with a stale `tree.Hash()`. Capture the error.
- Commit message typo: `immutableFroProof` → `immutableForProof`. Worth fixing on rebase; no behavior impact.

## Missing Tests

- **[concurrent read during commit]** [`tm2/pkg/bptree/proof_test.go`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof_test.go) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof_test.go) — no test pins down the COW invariant the fix relies on.
  <details><summary>details</summary>

  The fix is only safe because `treeInsert` clones every node on the modification path ([`insert.go:14-22`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/insert.go#L14-L22) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/insert.go#L14-L22)), so `lastSaved` and `root` can share nodes after `SaveVersion` without a later `Set` corrupting `lastSaved`. The new test only checks hashes at three discrete points; it never holds a proof from `lastSaved` across a `Set` mutation and re-verifies. A test like: `SaveVersion; proof := GetMembershipProof("a"); Set("z", ...); Set("z2", ...); VerifyMembership(committedHash, proof, "a", "1")` — assert the proof still verifies after multiple intervening writes — would lock in the COW guarantee.
  </details>

- **[Remove path]** [`tm2/pkg/bptree/proof_test.go`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof_test.go) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof_test.go) — `TestProof_UsesCommittedState` only exercises `Set`. A symmetrical case for `Remove` (commit `{a,b}`, `Remove("b")` without `SaveVersion`, proof for `b` must still verify against committed hash) would cover the dirty-delete scenario, which is the other half of "working tree diverges".

## Suggestions

- [`tm2/pkg/bptree/proof.go:208-222`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof.go#L208-L222) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof.go#L208-L222) — Consider a tiny doc comment on the `MutableTree.GetMembershipProof` / `GetNonMembershipProof` wrappers stating "proves against the last committed state (`SaveVersion`), not the working tree". Without it, the next caller hitting `ErrNoCommittedState` will re-debug the same surprise. The doc on `immutableForProof` ([`proof.go:224-226`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/proof.go#L224-L226) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/proof.go#L224-L226)) already says it; surface it on the exported wrappers too.

- [`tm2/pkg/bptree/errors.go:14`](https://github.com/gnolang/gno/blob/af325fc/tm2/pkg/bptree/errors.go#L14) · [↗](../../../../../.worktrees/gno-review-5468/tm2/pkg/bptree/errors.go#L14) — `ErrNoCommittedState` is now a public sentinel; the message reads `"no committed state: call SaveVersion before generating proofs"`. Consider trimming the imperative tail ("call SaveVersion ...") since the rest of the error vars in this file stick to noun-phrase form (`"version does not exist"`, `"tree is empty"`); callers can format guidance separately.

## Questions for Author

- Was the choice of `lastSaved` over `GetImmutable(t.version)` driven by the DB round-trip cost, or was there another constraint? A sentence in the PR body would close the open thread.
- Any plan to mirror this on `tm2/pkg/iavl/MutableTree.GetMembershipProof`? It embeds `*ImmutableTree` so a direct call against the working tree has the same shape as the bug fixed here, even if no internal caller exercises it.
