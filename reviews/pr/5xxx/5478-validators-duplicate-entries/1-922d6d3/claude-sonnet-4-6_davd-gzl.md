# PR #5478: fix(validators): handle duplicate validator entries in same block

**URL:** https://github.com/gnolang/gno/pull/5478
**Author:** omarsy | **Base:** master | **Files:** 6 | **+329 -3**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

This PR fixes a node-crashing bug where removing and re-adding the same validator address in the same block produces duplicate entries that reach `tm2/pkg/bft/types/validator_set.go`'s `processChanges()`, which rejects them and halts the chain with `"Error changing validator set: duplicate entry Validator{...}"`.

The root cause is that `r/sys/validators/v2` lacks an `UpdateValidator()` operation — power changes must be expressed as remove + re-add. When two separate DAO proposals (one remove, one re-add) execute in the same block, `saveChange()` in `validators.gno` was blindly appending both entries to the block's change list, producing `[{X, 0}, {X, 30}]`. `processChanges()` sorts by address before deduplication, so both entries collide.

The fix is layered at the realm level:

1. **`saveChange()` dedup** (`validators.gno:84-92`): When saving a change for the same block, the function now scans for an existing entry with the same address and overwrites it (last-writer-wins) instead of appending. This handles the cross-tx case.

2. **`checkDuplicateAddresses()` guard** (`poc.gno:12-20`): A new helper panics if `changesFn()` returns a validator list with duplicate addresses. It is called both at proposal creation time (`poc.gno:47`) and inside the execution callback (`poc.gno:66`).

Test additions:
- `validators_test.gno`: Three new `.gno` unit tests covering `saveChange` dedup, duplicate-address panic in `NewPropRequest`, and distinct-address success.
- `validator_duplicate_address.txtar`: Integration test for the within-proposal duplicate rejection.
- `validator_cross_proposal_dedup.txtar`: Integration test for cross-proposal dedup via `saveChange` last-writer-wins.
- `tm2/adr/pr5478_validator_set_dedup.md`: ADR documenting the decision and alternatives.

**Go build and tests:** `gno.land/...` builds cleanly. `gno.land/pkg/gnoland` tests pass.

## Test Results
- **Existing tests:** PASS — `go test ./gno.land/pkg/gnoland/... ` passes (93s). `go build ./gno.land/...` clean.
- **Edge-case tests:** 0 written (existing unit test coverage is adequate for the dedup logic)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `examples/gno.land/r/sys/validators/v2/poc.gno:66` — The second `checkDuplicateAddresses(changes)` call inside the execution callback is tautological. The `changes` variable is captured from `changesFn()` at proposal-creation time (line 37) and never mutated. Because it was already validated at line 47, the execution-time check will always pass and can never catch new duplicates introduced after creation. The comment in the ADR claims this guard defends against stale-closure tricks, but there is no code path that could produce different content in `changes` at execution vs. creation time in the current implementation. This adds confusion about what invariant is being enforced. Either remove it, or change the callback to re-invoke `changesFn()` and validate the fresh result if re-evaluation is the actual intent.

- [ ] `tm2/adr/pr5478_validator_set_dedup.md` — The ADR is placed under `tm2/adr/`, but per `AGENTS.md`, ADRs for `gno.land` changes belong in `gno.land/adr/`. The fix is entirely within `gno.land/r/sys/validators/v2`; `tm2` is unchanged. Move to `gno.land/adr/pr5478_validator_set_dedup.md`.

## Nits

- [ ] `examples/gno.land/r/sys/validators/v2/validators.gno:84-91` — The linear scan over `set` to find a duplicate address is O(n) where n is the number of existing changes for the block. For the expected scale (< 40 validators per proposal, few proposals per block) this is harmless, but if a future change raises the per-block change volume, a map would be cleaner. Not blocking; worth a comment acknowledging the trade-off.

- [ ] `gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar:89-95` — The `val_helper` comment explains the pubkey/address mismatch, which is necessary because a real matching-pubkey test would require a multi-node cluster. This is a structural limitation of the txtar harness and is honestly documented, but the test does **not** exercise the actual crash path (EndBlocker filtering drops the update before it reaches `processChanges()`). The test only verifies realm-level dedup via `Render`. This is an acceptable trade-off given the testing infrastructure constraints, but it should be noted in the PR description so reviewers understand that the chain-halt path has not been exercised in CI.

## Missing Tests

- [ ] No test covers the case where `saveChange()` is called with add-then-remove (rather than remove-then-add) for the same address in the same block. The last-writer-wins semantics should produce a removal entry `{X, power=0}`, which is the correct outcome. The current `TestSaveChange_DeduplicatesSameBlock` only tests remove-then-re-add. A complementary test for add-then-remove would strengthen confidence in both orderings. (Reference: `examples/gno.land/r/sys/validators/v2/validators_test.gno`)

## Suggestions

- Consider adding `UpdateValidator()` to the realm as the cleanest long-term fix (mentioned in ADR as future work). With an explicit update primitive, remove + re-add is no longer required, eliminating the duplicate-entry scenario entirely at the source. The ADR correctly identifies this but frames it only as future work. A follow-up issue would be valuable to track it.

- The `val_helper` realm in `validator_cross_proposal_dedup.txtar` embeds a hardcoded test address (`g1ut590acnamvhkrh4qz6dz9zt9e3hyu499u0gvl`). If this address ever appears elsewhere in integration tests, it could produce confusing cross-test interactions. Consider generating it dynamically or using a clearly test-scoped constant with a comment tying it to the test mnemonic used in `adduserfrom`.

## Questions for Author

- The `checkDuplicateAddresses` guard at execution time (callback in `poc.gno:66`) checks the same immutable `changes` slice captured at creation. Was the intent to re-call `changesFn()` to detect mutations to the closure's captured state, or was this meant as a pure defensive copy? If the former, the implementation needs to change; if the latter, the guard is unnecessary and confusing.

- Is there a scenario where `saveChange()` for the same address in the same block would represent a **bug** rather than a valid cross-tx dedup? For example, if two completely independent proposals (neither aware of the other) both try to remove the same validator in the same block, `saveChange` silently deduplicates to one removal. This is safe, but could mask a governance coordination error. Should there be a logged warning in that case?

## Verdict

APPROVE — The core fix is correct and well-reasoned: `saveChange()` last-writer-wins prevents duplicates from reaching `processChanges()`, and the within-proposal guard blocks the obvious same-proposal duplicate at creation time. The secondary execution-time check is redundant but harmless, and the ADR misplacement is a nit. Test coverage is adequate for the realm-level fix, with the acknowledged caveat that the actual tm2 crash path cannot be exercised in the txtar harness.
