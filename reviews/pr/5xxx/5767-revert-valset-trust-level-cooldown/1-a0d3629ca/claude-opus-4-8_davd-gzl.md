# PR #5767: revert(validators): remove valset trust-level and cooldown limits

URL: https://github.com/gnolang/gno/pull/5767
Author: omarsy | Base: master | Files: 12 | +148 -1173
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `a0d3629ca` (stale — +38 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5767 a0d3629ca`

**Verdict: APPROVE** — clean, complete revert of the chain-side valset trust-level and cooldown checks added in #4834 by the same author; mechanically sound, no dangling references, all tests pass. The single open item is a design/security call, not a code defect: removing the chain-side trust-level guard shifts an IBC liveness property onto relayers, so a maintainer with IBC context should sign off on the rationale (see Warnings).

## Summary

#4834 added two gates to `r/sys/validators/v3`: an IBC light-client trust-level overlap check (a valset update had to retain a configurable fraction, default 1/3, of the previous set's voting power) and a 24h cooldown between updates. This PR removes both, deleting `limits.gno`, `limits_test.gno`, the trust-level integration txtar, and the four governance setters (`GetTrustLevel`, `NewTrustLevelPropRequest`, `GetCooldown`, `NewCooldownPropRequest`). The argument: the trust-level overlap rule is a property of non-adjacent (skipping) light-client verification and belongs to the relayer/light client, not chain governance. For adjacent headers CometBFT's `VerifyAdjacent` checks the committed next-validator hash and applies no overlap threshold, and when a skipped update can't be verified directly a relayer bisects with intermediate headers. The cooldown is likewise not required by the IBC/CometBFT spec and would block AtomOne ICS provider-valset mirroring. The independently-added safeguards (operator dedup, KeepRunning opt-out, signing-key re-resolution, empty-set liveness floor) are preserved.

## Glossary

- `trustLevelRatio` — fraction of previous voting power the surviving validators had to retain; removed.
- `VerifyAdjacent` / `VerifyNonAdjacent` — CometBFT light-client verification for consecutive vs skipped heights; only the non-adjacent path applies the trust-level overlap check.
- baseline — the effective valset read at execute time, against which the removed check measured retained power.
- ICS — AtomOne Interchain Security; the mirroring use case the cooldown would have blocked.

## Fix

`newValoperChangeExecutor` previously took a `snapshotTrustLevel` and, at execute time, summed each surviving baseline validator's baseline voting power and rejected the proposal if `retained * den <= total * num` (strict, matching CometBFT's `got <= needed`); `NewValidatorProposalRequest` also rejected creation and execution within the cooldown window and stamped `lastValsetUpdate` on success. After this PR the executor just applies the deltas, enforces the empty-set floor, and publishes via `SetValsetProposal` — see [`proposal.gno:140-206`](https://github.com/gnolang/gno/blob/a0d3629ca/examples/gno.land/r/sys/validators/v3/proposal.gno#L140-L206) · [↗](../../../../../.worktrees/gno-review-5767/examples/gno.land/r/sys/validators/v3/proposal.gno#L140-L206). The rationale lives in the new ADR [`pr5767_revert_valset_trust_level_cooldown.md`](https://github.com/gnolang/gno/blob/a0d3629ca/gno.land/adr/pr5767_revert_valset_trust_level_cooldown.md#L1) · [↗](../../../../../.worktrees/gno-review-5767/gno.land/adr/pr5767_revert_valset_trust_level_cooldown.md#L1), which replaces the deleted `pr4834_*` ADR.

## Verification

- `go run ./gnovm/cmd/gno test -C examples/gno.land/r/sys/validators/v3 . -v` — all pass, including the new `TestNewValidatorProposalRequest_AllowsFullValsetReplacement`.
- `go test ./gno.land/pkg/integration -run 'TestTestdata/(params_valset_proposal_e2e|params_valset_proposal_power_update|params_valset_proposal_remove|params_valset_keeprunning_optout_bypass)$' -count=1` — `ok` (10.6s).
- `bash -n misc/val-scenarios/scenarios/18_govdao_v3_add_remove_validator.sh` — syntax OK.
- Repo-wide grep for every removed symbol (`NewTrustLevelPropRequest`, `NewCooldownPropRequest`, `GetTrustLevel`, `GetCooldown`, `trustLevelRatio`, `valsetUpdateCooldown`, `lastValsetUpdate`, `errValsetUpdateCooldown`, `trustRatio`, `resetLimits`, `disable_cooldown`) across `.gno/.go/.txtar/.sh/.md`: zero matches outside the new ADR. The revert is complete.

## Critical (must fix)

None.

## Warnings (should fix)

- **[removes an IBC safety guard — wants maintainer sign-off, not a code fix]** [`proposal.gno:140`](https://github.com/gnolang/gno/blob/a0d3629ca/examples/gno.land/r/sys/validators/v3/proposal.gno#L140) · [↗](../../../../../.worktrees/gno-review-5767/examples/gno.land/r/sys/validators/v3/proposal.gno#L140) — a single approved GovDAO proposal can now fully replace the validator set between two adjacent blocks.
  <details><summary>details</summary>

  The technical premise is correct: `VerifyAdjacent` verifies the untrusted header's validators against the trusted header's committed `NextValidatorsHash` and applies no overlap threshold, so an adjacent full swap is verifiable; the overlap rule only governs the non-adjacent skip path, where a relayer can bisect with intermediate headers. So nothing here is verifier-unsound. What the chain gives up is defense-in-depth: the previous gate prevented governance from, in one transaction, moving the valset far enough that any light client relying on skip-verification with a configured trust level is forced to fall back to per-block intermediate headers it may not receive in time. The ADR's Consequences section names this tradeoff explicitly. The decision is reasonable and is the original author reverting their own #4834, but because it relaxes a consensus-adjacent safety property, a reviewer with IBC context should explicitly acknowledge the rationale rather than rubber-stamp the green diff. Nothing to change in code.
  </details>

## Nits

- [`params_valset_proposal_e2e.txtar:69`](https://github.com/gnolang/gno/blob/a0d3629ca/gno.land/pkg/integration/testdata/params_valset_proposal_e2e.txtar#L69) · [↗](../../../../../.worktrees/gno-review-5767/gno.land/pkg/integration/testdata/params_valset_proposal_e2e.txtar#L69) — `ExecuteProposal` gas-wanted dropped 35M → 24M, consistent with the executor no longer running the baseline tally; test passes at the lower budget, so it doubles as a sanity check that the work really was removed.

## Missing Tests

- **[optional]** [`proposal_test.gno:939`](https://github.com/gnolang/gno/blob/a0d3629ca/examples/gno.land/r/sys/validators/v3/proposal_test.gno#L939) · [↗](../../../../../.worktrees/gno-review-5767/examples/gno.land/r/sys/validators/v3/proposal_test.gno#L939) — `AllowsFullValsetReplacement` drives `newValoperChangeExecutor` directly and asserts a swap that the old trust-level rule would have rejected, which is the right regression to pin. It does not exercise the full `NewValidatorProposalRequest` → dao create/vote/execute path for a full replacement, but the existing e2e txtars already cover that wiring and a full-swap e2e would add little. Fine as-is.

## Suggestions

None.

## Questions for Author

- Confirm there is no in-flight or planned IBC relayer/light-client integration on gno.land today that was depending on the chain-side trust-level floor as a guarantee (vs. handling skip-verification itself). The revert is sound on protocol grounds; this is about whether any current consumer assumed the chain enforced it.
