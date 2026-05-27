# PR #5478: fix(validators): handle duplicate validator entries in same block

URL: https://github.com/gnolang/gno/pull/5478
Author: omarsy | Base: master | Files: 6 | +453 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5478 922d6d3` (then `gh -R gnolang/gno pr checkout 5478` inside it)

Verdict: APPROVE — realm-level dedup is correct and well-tested; minor issues (tautological execution-time guard, ADR placement under `tm2/adr/` despite no tm2 changes) are non-blocking.

## Summary

`r/sys/validators/v2` has no `UpdateValidator()` — power changes must be expressed as remove + re-add. Two separate DAO proposals (one remove, one re-add of the same address) executed in the same block cause [`saveChange()`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/validators.gno#L72-L96) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/validators.gno#L72-L96) to append both entries to the block's change list as `[{X, 0}, {X, 30}]`. These reach tm2's `processChanges()` which rightly rejects duplicates (because `applyUpdates` runs before `applyRemovals`, a same-address entry in both lists would add-then-remove the validator — wrong outcome). Result: the node panics with `Error changing validator set: duplicate entry ...` and halts the chain.

The fix is layered at the realm: `saveChange()` now overwrites prior entries for the same address within the same block (last-writer-wins), and `NewPropRequest()` panics if `changesFn()` returns duplicate addresses. tm2's strict rejection is intentionally left intact as a safety net.

```
before:                              after:
  tx1: remove(X) -> append {X,0}      tx1: remove(X) -> append {X,0}
  tx2: add(X,30) -> append {X,30}     tx2: add(X,30) -> overwrite -> [{X,30}]
  block list: [{X,0},{X,30}]          block list: [{X,30}]
  processChanges() -> PANIC, halt     processChanges() -> ok
```

## Glossary

- `saveChange()` — per-block writer in the validators realm; the locus of the dedup fix.
- `changesFn()` — closure passed by the proposer; returns the target validator list.
- `NewPropRequest()` — proposal-creation entrypoint that wraps `changesFn` into a DAO callback.
- `processChanges()` — tm2's strict apply path; refuses to apply duplicate addresses and panics.
- `EndBlocker` — app layer that filters realm-emitted validator updates by pubkey before forwarding to tm2.

## Fix

[`saveChange()`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/validators.gno#L72-L96) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/validators.gno#L72-L96) was an unconditional append; it now scans the existing per-block slice and overwrites the matching-address entry if present, otherwise appends. This handles cross-proposal duplicates in the same block. [`checkDuplicateAddresses()`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L12-L20) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L12-L20) is a new helper that panics if any address repeats; it runs once at proposal creation ([`poc.gno:47`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L47) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L47)) and once inside the execution callback ([`poc.gno:66`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L66) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L66)) on the same immutable captured slice. tm2's `processChanges()` is untouched.

## Critical (must fix)

None.

## Warnings (should fix)

- [tautological re-check] [`examples/gno.land/r/sys/validators/v2/poc.gno:66`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L66) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L66) — execution-time `checkDuplicateAddresses(changes)` cannot fail.
  <details><summary>details</summary>

  The `changes` variable is captured at [`poc.gno:37`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L37) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L37) from `changesFn()` and validated at [`poc.gno:47`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L47) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L47). The callback at line 66 closes over the same slice header — no caller mutates it, and the slice is never re-derived from `changesFn()`. So the second call always sees the same content as the first and is guaranteed to pass. The ADR claims this is defense against stale closure tricks, but the closure captures `changes` (a `[]validators.Validator`), not `changesFn` — there is no path that produces fresh duplicates at execution time.

  Fix: either drop the duplicate call, or change the callback body to `changes := changesFn(); checkDuplicateAddresses(changes); ...` so it re-evaluates the closure at execution and the check is meaningful. The current shape is misleading code that suggests an invariant it does not actually enforce.
  </details>

- [misplaced ADR] [`tm2/adr/pr5478_validator_set_dedup.md`](https://github.com/gnolang/gno/blob/922d6d3/tm2/adr/pr5478_validator_set_dedup.md) · [↗](../../../../../.worktrees/gno-review-5478/tm2/adr/pr5478_validator_set_dedup.md) — ADR for a gno.land-only change lives under `tm2/adr/`.
  <details><summary>details</summary>

  The fix touches `examples/gno.land/r/sys/validators/v2/` exclusively; tm2 is explicitly unchanged (per the ADR's own "Decision" section). `gno.land/adr/` already exists and hosts PR-style ADRs of the same shape (`pr5325_cla_gnokey_helper.md`, `pr5265_agent_friendly_docs.md`). Misplaced ADRs decay search and code-ownership signals — a tm2 maintainer scanning `tm2/adr/` will misclassify the scope. Fix: move to `gno.land/adr/pr5478_validator_set_dedup.md`.
  </details>

## Nits

- [`examples/gno.land/r/sys/validators/v2/validators.gno:84-92`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/validators.gno#L84-L92) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/validators.gno#L84-L92) — linear scan for the matching address. Fine at current scale (≤40 validators per proposal, few proposals per block); flag for a future map-based rewrite if the per-block change volume grows.
- [`gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar:8-10`](https://github.com/gnolang/gno/blob/922d6d3/gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar#L8-L10) · [↗](../../../../../.worktrees/gno-review-5478/gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar#L8-L10) — comment honestly documents that the pubkey/address mismatch makes the EndBlocker drop the updates before tm2 sees them, so the test exercises realm-level dedup only and not the chain-halt path. Worth surfacing in the PR description so reviewers don't assume CI covers the crash.

## Missing Tests

- [add-then-remove ordering] [`examples/gno.land/r/sys/validators/v2/validators_test.gno:177-207`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/validators_test.gno#L177-L207) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/validators_test.gno#L177-L207) — only remove-then-re-add is tested.
  <details><summary>details</summary>

  `TestSaveChange_DeduplicatesSameBlock` covers `removeValidator -> addValidator` in the same block. The symmetric ordering (`addValidator -> removeValidator`) is also valid under the realm's API and should produce `[{X, 0}]` (a removal entry, since last-writer-wins keeps the second call). Adding a sibling test pins down both orderings and prevents a future refactor from silently changing semantics on the reverse path. Cheap to add (~15 lines).
  </details>

## Suggestions

- [`tm2/adr/pr5478_validator_set_dedup.md:64-65`](https://github.com/gnolang/gno/blob/922d6d3/tm2/adr/pr5478_validator_set_dedup.md#L64-L65) · [↗](../../../../../.worktrees/gno-review-5478/tm2/adr/pr5478_validator_set_dedup.md#L64-L65) — open a follow-up issue tracking the proposed `UpdateValidator()` realm primitive.
  <details><summary>details</summary>

  The ADR identifies `UpdateValidator()` as the cleanest long-term fix (power changes emit a single entry, no remove+re-add dance, no need for `saveChange()` dedup). Listing it under "Alternatives considered" with no linked tracking issue means it will likely decay into invisible tech debt. A short follow-up issue makes the eventual cleanup discoverable.
  </details>

- [`gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar:89-99`](https://github.com/gnolang/gno/blob/922d6d3/gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar#L89-L99) · [↗](../../../../../.worktrees/gno-review-5478/gno.land/pkg/integration/testdata/validator_cross_proposal_dedup.txtar#L89-L99) — the hardcoded `ValAddr`/`ValPubKey` constants in `val_helper` are tied to a specific test mnemonic; add a comment referencing the source mnemonic or move the constants closer to the user creation step to make the tie explicit.

## Questions for Author

- Was the intent of the second `checkDuplicateAddresses()` at [`poc.gno:66`](https://github.com/gnolang/gno/blob/922d6d3/examples/gno.land/r/sys/validators/v2/poc.gno#L66) · [↗](../../../../../.worktrees/gno-review-5478/examples/gno.land/r/sys/validators/v2/poc.gno#L66) to re-call `changesFn()` at execution time (and the missing re-call is the bug), or is it a defensive copy of the creation-time check? If the latter, it can be removed — see Warnings above.
- Two independent proposals (with no coordination) removing the same validator in the same block would now silently dedup to one removal in `saveChange()`. This is safe, but could mask a governance coordination error. Worth a log/event when overwrite happens?
