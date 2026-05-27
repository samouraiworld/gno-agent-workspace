# PR #5493: fix(gnoland): `--genesis-txs-file` flag causes double package additions in `gnoland start --lazy`

URL: https://github.com/gnolang/gno/pull/5493
Author: aronpark1007 | Base: master | Files: 1 | +24 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `1ffcb0a` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5493 1ffcb0a`

Verdict: REQUEST CHANGES — fix is directionally correct and resolves the reported `package already exists` panic from #5140, but lands without any unit test for the new dedup path (codecov: 45% patch coverage, the entire `slices.DeleteFunc` body is uncovered) and silently swallows two adjacent edge cases (intra-`genesisTxs` duplicates, multi-message txs containing a dup `MsgAddPackage`).

## Summary

When `gnoland start --lazy --genesis-txs-file <file>` is used, `generateGenesisFile` concatenated `LoadPackagesFromDir(examples/)` and `LoadGenesisTxsFile(<file>)` without deduplication. If the user's file re-added a package already present in `examples/` (e.g. `gno.land/p/nt/ufmt`), `InitChain` panicked at the second deploy with `vm.PkgExistError`. The fix collects package paths from the examples-derived txs into a `map[string]struct{}`, then `slices.DeleteFunc`s any tx in `genesisTxs` whose first matching `MsgAddPackage.Package.Path` is in the set. Examples win; user txs lose. Net: one new dependency (`slices`, `vm`), 24 lines, single file.

## Fix

Before: [`gno.land/cmd/gnoland/start.go:472`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L472) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L472) did a bare `append(pkgsTxs, genesisTxs...)`, letting duplicate package paths reach the VM keeper and trip [`PkgExistError`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/pkg/sdk/vm/errors.go) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/pkg/sdk/vm/errors.go) on the second deploy. After: [`gno.land/cmd/gnoland/start.go:450-470`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L450-L470) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L450-L470) builds a `pkgPaths` set from `pkgsTxs` and filters `genesisTxs` in place via `slices.DeleteFunc` before the same concat. The load-bearing assumption is that `pkgsTxs` always contains single-message `MsgAddPackage` txs (true by construction in [`LoadPackagesFromDir`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/pkg/gnoland/genesis.go#L172-L220) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/pkg/gnoland/genesis.go#L172-L220) → [`LoadPackage`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/pkg/gnoland/genesis.go#L223-L244) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/pkg/gnoland/genesis.go#L223-L244)), so the inner `for _, msg := range tx.Tx.Msgs` over `pkgsTxs` collapses to one iteration in practice.

## Warnings (should fix)

- **[no test for the new behavior]** [`gno.land/cmd/gnoland/start.go:450-470`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L450-L470) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L450-L470) — patch coverage is 45% (codecov); the dedup loop and `DeleteFunc` body are completely uncovered.
  <details><summary>details</summary>

  The PR adds a behavior change to `generateGenesisFile` — a function that has zero direct unit coverage (`TestStart_Lazy` in [`gno.land/cmd/gnoland/start_test.go`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start_test.go) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start_test.go) exercises `--lazy` without `--genesis-txs-file`). The fix is correct for the happy path described in #5140, but a regression that silently drops a non-duplicate tx (see warnings below) or stops filtering would not be caught by CI. A future refactor of `genesisTxs` assembly could trivially undo the dedup with nothing red. Fix: add a `TestGenerateGenesisFile_DedupsAgainstExamples` (or extend `TestStart_Lazy` with a `--genesis-txs-file` case) that writes a txs file containing one duplicate and one unique `MsgAddPackage`, calls `generateGenesisFile` against a fixture `examples/` dir, and asserts the resulting genesis has exactly one copy of each path. Bonus assertion: ordering — examples first, then non-duplicate user txs.
  </details>

- **[intra-genesisTxs duplicates still panic]** [`gno.land/cmd/gnoland/start.go:461-470`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L461-L470) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L461-L470) — dedup is one-direction (examples vs txs file); duplicates within the txs file itself still reach the keeper.
  <details><summary>details</summary>

  The fix populates `pkgPaths` from `pkgsTxs` only. If the user's `--genesis-txs-file` contains two `MsgAddPackage` lines for the same path (e.g. a copy-paste error, or a generated file with overlap), both pass through and the second triggers `vm.PkgExistError` at `InitChain` — the exact symptom #5140 reports. The issue describes "different deployers" for the duplicate, which is consistent with examples-vs-file overlap, but the underlying class of bug ("two deploys for the same path crash the node") is broader. Fix: collapse the two passes into one by populating `pkgPaths` as you walk both `pkgsTxs` and `genesisTxs` in order, dropping any tx whose path is already in the map. Same shape as the current code, two extra lines.
  </details>

- **[multi-msg tx drops entire tx on a single dup]** [`gno.land/cmd/gnoland/start.go:461-470`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L461-L470) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L461-L470) — `DeleteFunc` returns `true` on the first dup `MsgAddPackage` it finds, discarding every other message in the same tx.
  <details><summary>details</summary>

  The predicate iterates `tx.Tx.Msgs`, returns `true` as soon as one `MsgAddPackage` matches `pkgPaths`. If a future genesis tx bundles `[MsgAddPackage{dup}, MsgAddPackage{unique}]` or `[MsgAddPackage{dup}, MsgCall{...}]`, the whole tx is silently dropped — the unique add-package and the call vanish. In practice today every genesis-deploy tx produced by [`LoadPackage`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/pkg/gnoland/genesis.go#L223-L244) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/pkg/gnoland/genesis.go#L223-L244) and [`keyscli/addpkg.go`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/pkg/keyscli/addpkg.go) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/pkg/keyscli/addpkg.go) carries exactly one message, so this is latent. But there is no validator preventing a hand-written `--genesis-txs-file` from carrying multi-msg txs, and `std.Tx` happily accepts them. Fix: either (a) filter at the message level by rewriting `tx.Tx.Msgs` and only dropping the tx when its msg slice becomes empty, or (b) add an explicit comment + invariant check that asserts `len(tx.Tx.Msgs) == 1` for any tx carrying `MsgAddPackage`, with a clear error if it doesn't hold.
  </details>

## Nits

- [`gno.land/cmd/gnoland/start.go:452`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L452) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L452) — `make(map[string]struct{}, len(pkgsTxs))` over-allocates if any `pkgsTxs` tx is not a `MsgAddPackage` (none today, but the capacity hint is sized to txs, not to msgs). Minor; harmless.
- [`gno.land/cmd/gnoland/start.go:450-451`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L450-L451) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L450-L451) — the comment says "filter out any `MsgAddPackage` txs", but the filter operates at tx granularity (it drops the entire tx, not just the msg). Phrasing currently implicates the message but acts on the tx; consider "drop any tx in `genesisTxs` that contains a duplicate `MsgAddPackage`" so the behavior described in the warning above is at least visible at the call site.
- The PR description mentions a separate `TypeCheckError` still triggered by the same reproducer file. Worth either (a) confirming this is tracked in a follow-up issue, or (b) noting it in #5140's resolution comment so the issue isn't closed prematurely.

## Missing Tests

- **[dedup happy path]** [`gno.land/cmd/gnoland/start_test.go`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start_test.go) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start_test.go) — no test exercises `--lazy --genesis-txs-file <file>` with a deliberate duplicate.
  <details><summary>details</summary>

  The bug from #5140 is reproducible with a 1-line jsonl: an `m_addpkg` tx for any path already present in `examples/`. A test that writes such a file to a tempdir, runs `generateGenesisFile`, parses the resulting `genesis.json`, and asserts the deduped package appears exactly once would lock in the fix. Without it, the next refactor of genesis assembly can silently regress.
  </details>

- **[intra-file dup regression]** [`gno.land/cmd/gnoland/start_test.go`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start_test.go) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start_test.go) — should also cover a `--genesis-txs-file` whose own contents have two `MsgAddPackage` lines for the same path.
  <details><summary>details</summary>

  Pairs with the intra-genesisTxs warning above. If the fix is extended to dedup within the user file too, the test asserts no panic and exactly one copy in the final genesis. If the fix stays one-direction, the test documents the limitation explicitly (asserts `PkgExistError` still fires, so the next reader knows).
  </details>

## Suggestions

- [`gno.land/cmd/gnoland/start.go:450-470`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L450-L470) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L450-L470) — consider lifting the dedup into a helper (`dedupAddPackageTxs(existing []TxWithMetadata, incoming []TxWithMetadata) []TxWithMetadata`) in `gno.land/pkg/gnoland/genesis.go` next to `LoadPackagesFromDir` and `LoadGenesisTxsFile`. Keeps `generateGenesisFile` short, makes the helper unit-testable in isolation, and surfaces the rule (examples win) at the API boundary where future callers will find it.
- [`gno.land/cmd/gnoland/start.go:455`](https://github.com/gnolang/gno/blob/1ffcb0a/gno.land/cmd/gnoland/start.go#L455) · [↗](../../../../../.worktrees/gno-review-5493/gno.land/cmd/gnoland/start.go#L455) — when a duplicate is dropped, log it (`io.Println` or a `WARN` line) so operators using `--lazy` with a custom txs file can see that their tx was filtered. Silent dropping is the kind of thing that produces "why is my package not deployed" bug reports six months later.

## Questions for Author

- Was the choice of "examples win" intentional, or would the opposite ("user-supplied txs override examples") have been the user-facing expectation? The current direction means a user can never overwrite an examples package via `--genesis-txs-file` — only add fresh ones. Worth a one-liner in the PR description.
- Is the separate `TypeCheckError` mentioned in the test plan filed as a follow-up issue? If yes, link it from #5140 so the original report doesn't get closed while the secondary symptom is still live.
