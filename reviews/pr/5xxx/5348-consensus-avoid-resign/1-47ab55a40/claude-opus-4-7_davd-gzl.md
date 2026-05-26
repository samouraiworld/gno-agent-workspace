# PR #5348: fix(tm2/consensus): avoid re-signing known self votes

URL: https://github.com/gnolang/gno/pull/5348
Author: D4ryl00 | Base: master | Files: 4 | +336 -2
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]

Verdict: APPROVE — small, well-scoped consensus-layer guard against re-signing already-cast self votes; only nit is the conflict-vote branch logs and returns silently when called for a different `BlockID` at the same `H/R/T`, which is the correct safety choice but could be surfaced via a metric for fleet observability.

## Summary

`signAddVote` blindly called `privValidator.SignVote` every time, even when the validator already had a vote in the current round's vote set. Two failure modes flowed from that: wasted [`cs.wal.FlushAndSync()`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1712) on every re-entry (WAL replay, timer re-fires, round re-evaluation), and an outright vote-cast failure with strict remote signers (KMS/HSM) that reject duplicate-`H/R/S` requests — which can stall consensus. A secondary nil-deref bug existed for observer nodes because [`cs.privValidator.PubKey().Address()`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1770) ran before the `nil` guard. Fix: look up the current round's vote set first — reuse on `BlockID` match, refuse on conflict, otherwise sign as before; and reorder the nil check.

## Glossary

- `signAddVote` — sign a prevote/precommit and dispatch it on the internal queue.
- `cs.Votes` — `HeightVoteSet`, authoritative per-round prevote/precommit sets for this height.
- `replayMode` — set during WAL replay so signing errors don't spam Error logs.
- `sameHRS` — local `PrivValidator`'s tolerance: if the next sign request matches the last persisted `(Height, Round, Step)`, return the cached signature instead of re-signing.

## Fix

Before this PR, [`signAddVote`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1764-L1827) computed the validator address, checked membership, then called `signVote` → `privValidator.SignVote`. After, it checks `cs.Votes.{Prevotes,Precommits}(cs.Round).GetByAddress(addr)` first: same `BlockID` → log Info and return without signing; different `BlockID` → log Error (or Warn under `replayMode`) and return without signing; nil → fall through to the original path. The new helper [`existingSignedVote`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1749-L1761) takes the precomputed address to avoid recomputing it inside the helper, and the `cs.privValidator == nil` guard is moved above `PubKey().Address()` so observer nodes don't panic. The vote set is the right gate here — it's already the authoritative record of what this validator committed to in the round.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`tm2/pkg/bft/consensus/state.go:1789-L1802`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1789-L1802) — conflict-vote path is silent at the protocol layer; consider an `evsw` event or counter so an operator can detect a node refusing to vote.
  <details><summary>details</summary>

  When `existingSignedVote` returns a vote with a different `BlockID`, the new code logs and returns. That's the correct safety call — never sign a conflicting vote — but the only observable signal is a log line. In a fleet, this can sit unnoticed until consensus stalls. A bumped counter or a fired `cstypes.Event...` would make it dashboard-visible. Not blocking; the log line is acceptable for this PR's scope. Fix: optionally fire a `cstypes.EventConflictingVoteRefused` or bump a counter alongside the log.
  </details>

- [`tm2/pkg/bft/consensus/state.go:1757`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1757) — `panic("unknown vote type")` includes no context. Existing precedent in the file uses `fmt.Sprintf("Unexpected vote type %X", type_)` (e.g. [`height_vote_set.go:171`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/types/height_vote_set.go#L171)). Aligning the message helps when the panic surfaces in a goroutine stack.

- [`tm2/pkg/bft/consensus/state.go:1776`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1776) — already discussed with `@tbruyelle`: a one-line comment pointing to `tm2/adr/pr5348_consensus_avoid_resign.md` would help future readers skip the archeology. Author left it to preference; not blocking.

## Missing Tests

- [`tm2/pkg/bft/consensus/state_test.go`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state_test.go) — no test exercises the `cs.privValidator == nil` nil-deref fix.
  <details><summary>details</summary>

  The PR fixes a real nil-pointer for observer/light-client nodes (privValidator == nil) but adds no regression test that calls `signAddVote` on a `ConsensusState` with a nil privValidator. The original bug would have crashed; the new code returns early. A 5-line test (`cs.SetPrivValidator(nil); cs.signAddVote(types.PrevoteType, nil, types.PartSetHeader{})` and assert no panic) locks the fix down. Fix: add `TestStateSignAddVoteNoPrivValidator`.
  </details>

- [`tm2/pkg/bft/consensus/state_test.go`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state_test.go) — precommit path isn't covered; only `PrevoteType` is tested for both reuse and conflict branches.
  <details><summary>details</summary>

  `existingSignedVote` switches on `type_` and pulls the matching vote set. A `PrecommitType` test would be near-identical to the prevote one but would exercise the second case in the switch, locking it against accidental refactors that break the precommit path. Cost is two near-duplicate tests; value is moderate. Optional.
  </details>

## Suggestions

- [`tm2/pkg/bft/consensus/state.go:1749`](../../../../../.worktrees/gno-review-5348/tm2/pkg/bft/consensus/state.go#L1749) — fold `existingSignedVote` into `signAddVote` if it's never called from anywhere else.
  <details><summary>details</summary>

  `existingSignedVote` has a single caller. Extraction makes sense if the helper aids testing or is reused; here it's a 12-line function with one call site whose body could be inlined as a 4-line switch. Either keep it (cheap; future test seam) or inline it (slightly clearer call flow). Not blocking.
  </details>

- [`tm2/adr/pr5348_consensus_avoid_resign.md:64`](../../../../../.worktrees/gno-review-5348/tm2/adr/pr5348_consensus_avoid_resign.md#L64) — note in the ADR that the conflict-vote refusal is permanent for the round (the validator simply doesn't vote until the next round resets the set). Today the ADR only documents the happy paths.

## Questions for Author

- The conflict branch logs Error normally and Warn under `replayMode`. During replay, conflicting self-votes shouldn't be possible (we're replaying our own WAL), so Warn implies the path is reachable there — is the expectation that out-of-order WAL entries or peer-injected self-votes can land before `signAddVote` runs in replay? If yes, that's worth a one-sentence note in the ADR; if no, the Warn could stay Error and the path treated as a bug.
