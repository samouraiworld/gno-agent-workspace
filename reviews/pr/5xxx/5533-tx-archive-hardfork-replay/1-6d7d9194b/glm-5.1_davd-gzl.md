# PR #5533: feat(contribs/tx-archive): hardfork-replay readiness

**URL:** https://github.com/gnolang/gno/pull/5533
**Author:** moul | **Base:** feat/genesis-replay-upgrade3 | **Files:** 9 | **+455 -18**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR enhances `contribs/tx-archive` so its output is directly usable as input to the genesis-replay hardfork mechanism (#5511). It is stacked on the `feat/genesis-replay-upgrade3` branch. Five logical changes:

1. **Hardfork replay metadata** — adds `block_height`, `chain_id`, and `failed` fields to `GnoTxMetadata`. `Client` gains `GetChainID()`; RPC impl reads it from `/status.NodeInfo.Network`.

2. **Chain stdlib amino registration** — blank-imports `gnovm/stdlibs/chain` in both backup and restore clients so amino can decode events from chain-stdlib syscalls.

3. **Per-tx SignerInfo population** (`signerinfo.go`, 246 lines new) — the core new logic. For each signer in each exported tx, resolves `(account_num, pre-tx sequence)` via brute-force signature verification. Strategy: query `auth/accounts/<addr>` at halt height for anchor `(accNum, finalSeq)`, brute-force sequences in `[0, finalSeq]` on first successful tx, increment on subsequent successes, buffer failed txs and back-patch. This switches the writer from streaming to buffered mode (all txs held in memory until `Finalize()`).

4. **Info-level progress log** — one-line status every ~5s during long pulls.

5. **CLI flag** `-no-populate-signer-info` — opt-out of SignerInfo resolution.

End-to-end validated: 2637 txs on gnoland1 (halt @704052) produce 0/2715 replay failures after hardfork genesis assembly.

## Test Results

- **Existing tests:** PASS — `go test ./backup/...` completes in 0.08s. New assertions for `BlockHeight`, `ChainID`, `Failed` fields pass.
- **Edge-case tests:** Skipped — signerinfo logic is untested (see Missing Tests).
- **CI:** gnogenesis/lint FAIL (gofmt issues in `gnogenesis/internal/fork/` — from base branch, not this PR's files). All other checks pass (tx-archive lint, test, build all PASS).

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `contribs/tx-archive/backup/signerinfo.go:199` — `bruteForceSignerSequence` loop `for seq := lo; seq <= hi` does nothing (0 iterations) when `lo > hi`. This can happen when `ss.seq > ss.finalSeq` (account was created after halt-height query, or sequence rolled past finalSeq). The function returns `lo` with an error, and the caller at line 133 silently falls back to `ss.seq`. No data corruption (failed txs are skipped on replay), but SignerInfo will carry a stale/incorrect sequence with no warning. Consider logging a warning when brute-force fails.

- [ ] `contribs/tx-archive/backup/backup.go:87` — When `populateSignerInfo` is enabled, **all txs** are buffered in `bufferedTxs` until `Finalize()`. For a full-chain backup (e.g. 700k blocks, 2715 txs on gnoland1), this is modest. But chains with heavy throughput could accumulate millions of txs in memory. There's no memory-budget limit or documented constraint. Consider adding a documented memory caveat or a streaming-friendly alternative for large chains.

- [ ] `contribs/tx-archive/backup/client/rpc/rpc.go:86` — `GetAccountAtHeight` returns `(0, 0, nil)` when `res.Response.Error != nil`. A real ABCI error (e.g. node pruned that height) is silently treated as "account doesn't exist yet." This will produce wrong SignerInfo (accNum=0, seq=0) instead of failing the backup. Should distinguish between "empty response" and "error response."

## Nits

- [ ] `contribs/tx-archive/backup/client/rpc/rpc.go:65-70` — `GetChainID()` makes a separate `/status` RPC call, but `GetLatestBlockNumber()` (line 52) also calls `/status`. During `ExecuteBackup`, both are called sequentially at startup. A single `/status` call could cache both values, saving one round-trip.

- [ ] `contribs/tx-archive/backup/signerinfo.go:53-54` — `pendingFailedTx.ownerSS` is set but never read outside of assignment. It appears to be dead code from an earlier design iteration (commit message confirms: "drop unused 'buffered' slice on signerResolver"). The `ownerSS` field should be removed.

## Missing Tests

- [ ] `signerinfo.go` — 0 test coverage (Codecov: 13.76%). The entire `signerResolver`, `Populate`, `Finalize`, `bruteForceSignerSequence`, `assignFailedTxSequences`, and `assignTrailingFailedTxSequences` are untested. Key scenarios needing coverage:
  - Single signer, all successful txs — sequence increments linearly.
  - Single signer, mix of failed and successful txs — back-patching correctness.
  - Signer only has failed txs (no successful tx) — `Finalize` trailing path.
  - Multi-signer tx — both signers resolved independently.
  - `bruteForceSignerSequence` — correct sequence found, not found, empty range (`lo > hi`).
  - `GetAccountAtHeight` — account exists, doesn't exist, ABCI error.
- [ ] `backup.go:89` — Watch mode disables SignerInfo (`!cfg.Watch`), but no test verifies SignerInfo is absent in watch mode output.
- [ ] `backup.go:266-280` — Buffered flush path (resolver != nil) is exercised by existing tests (mock `GetAccountAtHeight` returns 0,0), but only with empty SignerInfo. Need test with non-trivial mock data.
- [ ] `options.go:32-36` — `WithPopulateSignerInfo(false)` is untested.

## Suggestions

- Log a warning in `signerinfo.go:130-133` when `bruteForceSignerSequence` fails, so operators can investigate replay issues instead of silently carrying wrong SignerInfo.
- In `rpc.go:86`, distinguish `res.Response.Error != nil` (real error) from `len(res.Response.Data) == 0` (account doesn't exist). Return an error for the former, `(0, 0, nil)` only for the latter.
- Consider a `MaxBufferedTxs` option or at least a documented upper bound on memory usage when SignerInfo is enabled, since the buffering architecture holds the entire chain's tx data in memory.

## Questions for Author

- The PR is based on `feat/genesis-replay-upgrade3`, not `master`. What's the merge plan — will this be squashed into a single commit targeting `master`, or rebased onto `master` after #5511 lands?
- `signerinfo.go:53-54` — Is `ownerSS` on `pendingFailedTx` intentionally kept for future use, or is it leftover dead code?
- For chains with very high throughput (millions of txs), is the all-in-memory buffering model acceptable, or should there be a streaming fallback?

## Verdict

REQUEST CHANGES — The SignerInfo logic is the heart of this PR and has 0% dedicated test coverage. The `GetAccountAtHeight` ABCI-error-silencing bug could produce wrong metadata on pruned nodes. The core algorithm is sound and the end-to-end validation is strong, but the lack of unit tests for the most complex new code is a blocking gap.
