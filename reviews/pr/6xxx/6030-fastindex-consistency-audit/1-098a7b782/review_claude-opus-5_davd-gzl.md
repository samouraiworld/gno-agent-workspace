# PR [#6030](https://github.com/gnolang/gno/pull/6030): feat(tm2/bptree): add fast-index consistency audit and gnoland fastindex verify

URL: https://github.com/gnolang/gno/pull/6030
Author: moul | Base: master | Files: 32 | +2704 -155
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 098a7b782 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6030 098a7b782`

**TL;DR:** The node keeps a side table of recent values to answer lookups in one read instead of walking the tree. A bug fixed in [#6018](https://github.com/gnolang/gno/pull/6018) could leave that table disagreeing with the real data, silently, until nodes forked. This adds a tool that reads a stopped node's database and says whether the table still agrees.

**Verdict: REQUEST CHANGES** — the audit itself is right, but the command reports success in two states where it audited nothing, which is the failure mode that matters for its stated use as a CI gate (3 Warnings, 3 Missing tests, 2 Nits, 2 Suggestions).

Scope: the branch carries [#6018](https://github.com/gnolang/gno/pull/6018)'s commits, already reviewed at `5ceafd2c5` ([review](../../6018-fastindex-snapshot-isolation/1-5ceafd2c5/review_claude-opus-5_davd-gzl.md)). This round covers only the six files added on top, `git diff 5ceafd2c5..098a7b782`, +504 and no deletions.

## Verify first

- [`gno.land/cmd/gnoland/fastindex_verify.go:100-103`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L100-L103) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L100-L103) — point the command at a directory holding an empty `db/` and confirm what it prints and returns. It reports `version=0 entries=0`, says nothing to verify, and exits 0.
- [`gno.land/cmd/gnoland/fastindex_verify.go:18`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L18) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L18) — confirm `s/_/` is still the prefix gno.land's main store lands under. It is set in [`constructStore`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/store/rootmulti/store.go#L593-L599) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/store/rootmulti/store.go#L593-L599) and depends on the store being mounted with its own DB handle, which [`app.go:106`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/pkg/gnoland/app.go#L106) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/pkg/gnoland/app.go#L106) does. A drift here turns the audit into a silent no-op, per the first Warning.

## Summary

[#6011](https://github.com/gnolang/gno/issues/6011) was a persisted `'F'` entry that kept a value the tree had already moved past, which a fast read then served, so nodes could diverge on the app hash with nothing in the logs. [#6018](https://github.com/gnolang/gno/pull/6018) removes the cause. This adds detection: [`VerifyFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L90) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L90) walks every `'F'` entry and compares its inlined value against the tree's own committed value, resolved through [`treeLookup` plus `getCommittedValue`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L119-L131) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L119-L131) rather than through `Get`, which would consult the index under audit. `gnoland fastindex verify` wraps it for a stopped node and maps the report onto an exit status. The audit is read-only in the sense that matters most, it never rebuilds or heals, which [`TestVerifyFastIndex_ReadOnly`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify_test.go#L85-L103) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify_test.go#L85-L103) pins.

## Diagram

```
'F' entries ──┐
              ├─ per entry: verifyChecksum ─> corrupt
              │              treeLookup(root, key) ─ not found ─> orphan
              │              getCommittedValue(vk) ─ differs ──> stale-value   (the #6011 signature)
stamp ────────┘
                              │
   exit status <── stamp absent ─> "nothing to verify", 0   <-- also the empty-store verdict
                  stamp > version ─> rewound DB, 1
                  stamp < version ─> WARN, 0
                  mismatches > 0 ─> CORRUPTION, 1
```

## Warnings (should fix)

- **[an audit that walked nothing reports OK]** `gno.land/cmd/gnoland/fastindex_verify.go:100-103` — a data directory holding a `db/` folder but no gnoland store passes with exit 0 instead of failing.
  <details><summary>details</summary>

  [`LoadReadonly`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/mutable_tree.go#L519-L528) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/mutable_tree.go#L519-L528) returns `(0, nil)` when there is no latest version, so the report carries `version=0 stamp=0 entries=0`, and the [`!rep.StampPresent`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L101-L103) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L101-L103) branch reads that as "feature disabled or never built" and returns nil. Nothing distinguishes a store audited and found clean from a store that was never opened. The command is offered for CI gates and snapshot providers, where the whole value is the exit status: a wrong `-data-dir`, a `-db-backend` that does not match what the node ran with, or a directory layout change all leave the gate green forever. Measured by [`fastindex_verify_gaps_test.go`](tests/fastindex_verify_gaps_test.go), [repro](comment_claude-opus-5.md). Fix: fail when no committed version was loaded.
  </details>

- **[a populated index reported as no index]** `gno.land/cmd/gnoland/fastindex_verify.go:101` — an index whose stamp was deleted but whose entries remain prints its mismatches and then says no fast index is present.
  <details><summary>details</summary>

  The stamp branch is evaluated before the mismatch branch, so with entries present and the stamp gone the output is a list of mismatches followed by "no fast index present (feature disabled or never built) — nothing to verify". That state is exactly what this command's own remedy produces: [line 107](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L105-L107) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L105-L107) tells the operator to drop the stamp to force a rebuild, and [`dropFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L209-L215) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L209-L215) deletes the stamp first and clears the entries after, so an interrupted drop leaves the same shape. Exit 0 is the right verdict there, since [`ensureFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L246-L248) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L246-L248) rebuilds on the next load; the sentence is what is wrong. Fix: with entries present and no stamp, report it the way the behind case is reported.
  </details>

- **[the read-only tool writes to the directory]** `gno.land/cmd/gnoland/fastindex_verify.go:69` — [`dbm.NewDB`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L69) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L69) opens read-write and creates the store when it is absent.
  <details><summary>details</summary>

  Running the command against a directory with an empty `db/` leaves a `gnolang.db` behind, observed in the same repro as the first Warning. The [command help](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L36) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L36) says READ-ONLY and the PR body says the audit inspects a captured node state, so an operator auditing the one copy they kept of a diverged node has no reason to expect the directory to change. `tm2/pkg/db` exposes no read-only open, so the honest fix is in the wording: say the audit performs no writes, while the database is still opened read-write, and keep the existing advice to run against a copy.
  </details>

## Nits

- **[a private prefix rule, copied]** `gno.land/cmd/gnoland/fastindex_verify.go:18` — [`mainStorePrefix`](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L18) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L18) restates the rule inside [`constructStore`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/store/rootmulti/store.go#L593-L599) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/store/rootmulti/store.go#L593-L599), which also branches on whether the store was mounted with its own DB. Exporting a helper there keeps the two from drifting, and a drift is silent under the first Warning.
- **[cache size out of nowhere]** `gno.land/cmd/gnoland/fastindex_verify.go:78` — the tree is built with a node cache of [10000](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L76-L79) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L76-L79) with nothing saying why, while the package's own test helper uses 1000.

## Missing Tests

- **[the exit-1 branch the command exists for]** `gno.land/cmd/gnoland/fastindex_verify_test.go:104` — the three CLI tests cover healthy, stamp-behind and a missing DB. Nothing covers a stamp-current index that disagrees with the tree, which is the whole point of the command.
  <details><summary>details</summary>

  The damage does not need bptree internals: reading the persisted `'F'` record back through the prefixed DB, flipping one byte and writing it again leaves the stamp current and makes the record fail its checksum, which is also the only path that produces the [`MismatchCorrupt`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L26) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L26) kind. Shipped as `TestFastindexVerify_CorruptRecord` in [`fastindex_verify_gaps_test.go`](tests/fastindex_verify_gaps_test.go); it passes at 098a7b782, so it is a guard, not a bug report.
  </details>

- **[the rewound-DB branch]** `gno.land/cmd/gnoland/fastindex_verify.go:104-107` — nothing covers a stamp ahead of the latest version, the other exit-1 verdict.
  <details><summary>details</summary>

  Save two versions with the index on, then rewrite the stamp to a higher number through the prefixed DB. The assertion is that the command errors and names the rewind, which also pins that `LoadReadonly` reaches the audit at all in that state rather than failing the way [`ensureFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L249-L254) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L249-L254) does on a live load. That distinction is the reason the command uses `LoadReadonly`, and no test states it.
  </details>

- **[the sample cap]** `tm2/pkg/bptree/verify.go:11` — [`verifyMismatchCap`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L11) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L11) bounds the sample while `MismatchCount` keeps the true total, and no test drives more than one mismatch.
  <details><summary>details</summary>

  Doctoring 300 entries and asserting `MismatchCount == 300` with `len(Mismatches) == 256` pins the contract the CLI's ["and N more"](https://github.com/gnolang/gno/blob/098a7b782/gno.land/cmd/gnoland/fastindex_verify.go#L96-L98) · [↗](../../../../../.worktrees/gno-review-6030/gno.land/cmd/gnoland/fastindex_verify.go#L96-L98) line depends on.
  </details>

## Suggestions

- **[one ordered walk, not a descent per entry]** `tm2/pkg/bptree/verify.go:119` — the audit restarts from the root for every `'F'` entry via [`treeLookup`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/verify.go#L119) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/verify.go#L119).
  <details><summary>details</summary>

  Both the index scan and the tree are sorted by user key, so the same audit is a merge of two ordered streams. [`rebuildFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L285-L301) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L285-L301) already walks the tree that way with `iterateNodeResolved`, and reusing the shape turns a random descent per entry into one pass, which is what a snapshot provider auditing a full mainnet store pays for. It also makes the opposite direction visible, a key the tree holds with no entry in the index, which the current shape cannot see at all.
  </details>

- **[a rebuild command has a natural home]** `gno.land/cmd/gnoland/fastindex.go:23` — the PR body names an offline `fastindex rebuild` as future work, and both this command and [`ensureFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L250-L253) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L250-L253) already tell operators to delete a `PrefixMeta` key by hand as the remedy.
  <details><summary>details</summary>

  Hand-editing a meta key is the step most likely to be done wrong on a node that is already in a bad state, and [`dropFastIndex`](https://github.com/gnolang/gno/blob/098a7b782/tm2/pkg/bptree/fast_index.go#L198-L216) · [↗](../../../../../.worktrees/gno-review-6030/tm2/pkg/bptree/fast_index.go#L198-L216) already implements the safe ordering. Not a blocker for this PR; worth deciding before the advice reaches release notes.
  </details>

## Verified

- A directory holding an empty `db/` returns exit 0 from the command and gains a `gnolang.db`, which is the first and third Warnings measured together rather than argued.
- A stamp-current index with a byte flipped inside its persisted record is classified `corrupt` and exits 1, so the detection path works and only its test is missing.
- `go test ./tm2/pkg/bptree/... ./gno.land/cmd/gnoland/...` is green at 098a7b782, and all 80 GitHub checks pass.

## Open questions

- The audit treats a key held by the tree with no `'F'` entry as fine, which matches the trust contract, since a miss falls back to the authoritative walk. A completeness count would still tell a snapshot provider whether the accelerator is actually accelerating. Not posted: it is a different tool, not a gap in this one.
