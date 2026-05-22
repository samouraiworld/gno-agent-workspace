# PR #5443: fix(gnobr): read LastResultsHash from block header instead of ABCI responses

**URL:** https://github.com/gnolang/gno/pull/5443
**Author:** moul | **Base:** master | **Files:** 2 | **+88 -6**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR fixes a crash in the `gnobr` (Gno Block Replayer) tool that occurs when rolling back chains that contain blocks with custom ABCI event types (e.g., `tm.StorageDepositEvent`). The crash happens because `LoadABCIResponses()` uses amino to unmarshal stored responses, and unregistered custom event types cause amino to call `osm.Exit(1)` — a hard process exit with no recovery possible.

The fix changes how `LastResultsHash` is obtained during rollback. Instead of loading ABCI responses for block N and recomputing the results hash, the tool now reads the `LastResultsHash` directly from block N+1's header (which Tendermint already computed and stored at commit time). This is both simpler and avoids the amino unmarshalling entirely.

Key implementation details:
- `contribs/gnobr/main.go:74-81` — The `nextMeta` (block N+1's metadata) is now loaded BEFORE the blockstore is trimmed. This is critical because `TrimToHeight()` deletes blocks above the target, so block N+1 would be gone after trimming.
- `contribs/gnobr/main.go:117-125` — Instead of `LoadABCIResponses()`, the code reads `nextMeta.Header.LastResultsHash` and assigns it to the state.
- `contribs/gnobr/main.go:78-81` — When `nextMeta` is nil (target height is the blockstore tip — no block N+1 exists), a warning is printed and `LastResultsHash` is not updated. This is an edge case where the old code would also have crashed (since it called `LoadABCIResponses` which fails on custom events).

The PR also adds a comprehensive `README.md` documenting the two-pass approach for handling app-hash-breaking changes during chain rollback.

## Test Results
- **Existing tests:** No tests exist for gnobr (it's a standalone tool in `contribs/`).
- **Build:** PASS — `go build .` succeeds in `contribs/gnobr/`.
- **CI status:** All checks pass.
- **Edge-case tests:** Skipped — gnobr requires a full chain database to test meaningfully; unit testing would require significant mocking infrastructure.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `contribs/gnobr/main.go:78-81` — **Silent data loss when target is at blockstore tip.** When `nextMeta` is nil, the code prints `fmt.Println("Warning: no next block meta found...")` but continues execution. The `LastResultsHash` in the saved state will retain whatever value it had before, which may be stale or incorrect. Downstream consumers of the rolled-back state that depend on `LastResultsHash` being accurate (e.g., light client verification, IBC) could encounter mismatches. Consider: (a) making this a hard error if `LastResultsHash` accuracy is important, or (b) documenting explicitly in the README that rollback to the tip height is a no-op / unsupported scenario, or (c) falling back to recomputing from ABCI responses with a more graceful error handler.

- [ ] `contribs/gnobr/main.go:117-125` — **No validation that `nextMeta.Header.LastResultsHash` is non-empty.** If the stored block header somehow has an empty/nil `LastResultsHash` (e.g., corrupted DB, genesis block edge case), it will silently overwrite the state with a zero hash. A sanity check (e.g., `if len(nextMeta.Header.LastResultsHash) == 0`) with a warning would make this more robust.

## Nits

- [ ] `contribs/gnobr/main.go:78` — The warning uses `fmt.Println` while other messages in the file use `fmt.Printf`. Should be consistent — either all `Println` or all `Printf` with newlines.

- [ ] `contribs/gnobr/main.go:74` — The variable name `nextMeta` is clear in context but `nextBlockMeta` would be more self-documenting for readers skimming the code.

## Missing Tests

- [ ] No test coverage for gnobr exists at all. While adding integration tests for a chain rollback tool is non-trivial, at minimum a test for the `nextMeta == nil` edge case (line 78) would help prevent regressions — `contribs/gnobr/main.go:74-81`.

## Suggestions

- The `README.md` is well-written and documents the two-pass approach clearly. Consider adding a note about the `nextMeta == nil` edge case — specifically that rolling back to the exact blockstore tip is effectively unsupported because no N+1 header exists.

- Long-term, the amino registration problem that caused this issue (custom event types not being registered) should be fixed at the amino/ABCI layer rather than worked around in gnobr. The current fix is pragmatic and correct, but other tools that call `LoadABCIResponses()` will hit the same crash. Consider filing an issue to track amino event registration for custom types.

## Questions for Author

- Is there a scenario where a user would legitimately want to roll back to the exact blockstore tip (triggering the `nextMeta == nil` path)? If so, what should `LastResultsHash` be set to?

- The old code loaded ABCI responses and recomputed `LastResultsHash`. The new code reads it from the header. Is there any scenario where these two values could differ (e.g., a bug in the original block commit that stored a wrong hash in the header)? If so, is the header value always authoritative?

## Verdict

**APPROVE** — Clean, targeted fix for a real crash. The approach of reading from the next block's header is sound and avoids the amino unmarshalling entirely. The `nextMeta == nil` edge case deserves a bit more attention (error vs. warning) but doesn't block merging since it represents an edge case that also crashed before this PR.
