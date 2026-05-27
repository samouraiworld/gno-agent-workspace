# PR #5216: fix(consensus): handle conflicting votes instead of panicking

URL: https://github.com/gnolang/gno/pull/5216
Author: davd-gzl | Base: master | Files: 6 | +170 -39
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]
Local worktree: `git -C gno worktree add .worktrees/gno-review-5216 aed66be4a` (then `gh -R gnolang/gno pr checkout 5216` inside it)

**Verdict: REQUEST CHANGES** — fix swaps a `panic("not yet implemented")` for an unconditional `cs.privValidator.PubKey().Address()` deref that nil-panics on every non-validator full node when any Byzantine validator double-signs; same blast radius as the bug being patched, larger fleet. Plus: `don't merge` label still on, blocking on `@jaekwon` review (`@thehowl` CHANGES_REQUESTED), and no real evidence pool so double-signing is still consequence-free (acknowledged by author).

## Summary

Replaces `panic("not yet implemented")` inside `tryAddVote()` with a code path that (a) logs and ignores conflicting votes from self, (b) forwards conflicting votes from peers to a placeholder `NoOpEvidencePool`. The panic ran inside the consensus `receiveRoutine`, so a single double-signing validator on the network halted consensus on every node — the gnoland1 chain halt cited by `@thehowl`. Fix introduces an `evidencePool` interface, ships a `NoOpEvidencePool`, wires it through `NewConsensusState`, and adds two unit tests around the peer/self branches.

The bug **is not fully closed**: the self-detection branch derefs [`cs.privValidator.PubKey()`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L1544) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L1544) without a nil check, but `privValidator` is `nil` on every non-validator node — full nodes, sentries, RPC nodes — and they all receive gossiped votes via [`reactor.go:310`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/reactor.go#L310) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/reactor.go#L310). Same Byzantine double-vote now crashes the entire non-validator fleet instead of the validator fleet. Verified with a one-line repro (see Critical below).

## Glossary

- `tryAddVote` — wraps `addVote`; classifies the error (height mismatch / conflicting sig / other) and decides whether to ignore, forward to evidence pool, or just log.
- `receiveRoutine` — single goroutine processing peer messages and timeouts; an unhandled panic here halts the node's consensus.
- `evidencePool` — interface added in this PR with one method `ReportConflictingVotes`. No real implementation exists in tm2.
- `NoOpEvidencePool` — placeholder that drops conflicting votes silently; wired in [`node.go:344`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/../node/node.go#L344) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/../node/node.go#L344).
- `privValidator` — local node's signing key; `nil` on non-validator nodes per [`node.go:347`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/node/node.go#L347) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/node/node.go#L347).
- `VoteConflictingVotesError` — wraps `DuplicateVoteEvidence` with `VoteA`/`VoteB`; raised by `voteSet.addVote` at [`vote_set.go:205`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/types/vote_set.go#L205) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/types/vote_set.go#L205).

## Fix

Before: `tryAddVote` panicked on every `VoteConflictingVotesError`, killing `receiveRoutine` permanently on **every node** receiving the gossiped conflicting vote pair. After: the error is split into self / peer branches — self-conflict logs at `Error` level and returns; peer-conflict forwards to `cs.evpool.ReportConflictingVotes` (currently a no-op) and logs at `Warn`. `tryAddVote` also lost its `error` return (consolidated to `bool`), which simplifies `handleMsg`'s switch but unconditionally re-enables the previously-commented generic error log at [`state.go:732-735`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L732-L735) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L732-L735) for `ProposalMessage` / `BlockPartMessage` errors. The `evidencePool` interface lets a future PR plug in a real evidence handler without touching `state.go` again.

## Critical (must fix)

- **[nil deref re-crashes the non-validator fleet]** [`tm2/pkg/bft/consensus/state.go:1544`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L1544) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L1544) — `cs.privValidator.PubKey().Address()` runs before any nil check; non-validator nodes have `privValidator == nil` and crash on the first gossiped conflicting vote.
  <details><summary>details</summary>

  Reproduced in a worktree: set `cs.privValidator = nil` on a stock `randConsensusState`, feed two conflicting prevotes through `handleMsg` — second call panics with `runtime error: invalid memory address or nil pointer dereference` at the same line. The whole point of this PR is "a double-signing validator currently break every node's consensus goroutine permanently"; that property is preserved for the entire non-validator fleet, which is strictly larger than the validator set (sentries, RPC nodes, archive nodes, indexers, gnodev users tailing mainnet). Gate the deref on `cs.privValidator != nil` before doing the self-check — non-validator nodes should fall straight into the `ReportConflictingVotes` branch.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5216 -R gnolang/gno
  cat > tm2/pkg/bft/consensus/nil_pv_repro_test.go <<'EOF'
  package consensus

  import (
      "testing"

      "github.com/gnolang/gno/tm2/pkg/bft/types"
      p2pmock "github.com/gnolang/gno/tm2/pkg/p2p/mock"
  )

  func TestNilPrivValidatorOnConflictingVote(t *testing.T) {
      cs, vss := randConsensusState(2)
      cs.SetPrivValidator(nil) // non-validator full node
      peer := p2pmock.Peer{}
      voteA := signVote(vss[1], types.PrevoteType, []byte("blockA"), types.PartSetHeader{})
      cs.handleMsg(msgInfo{&VoteMessage{voteA}, peer.ID()})
      voteB := signVote(vss[1], types.PrevoteType, []byte("blockB"), types.PartSetHeader{})
      defer func() {
          if r := recover(); r != nil {
              t.Logf("PANIC: %v", r)
              t.FailNow()
          }
      }()
      cs.handleMsg(msgInfo{&VoteMessage{voteB}, peer.ID()})
  }
  EOF
  go test -v -run TestNilPrivValidatorOnConflictingVote ./tm2/pkg/bft/consensus/
  # observed: --- FAIL with PANIC: runtime error: invalid memory address or nil pointer dereference
  rm tm2/pkg/bft/consensus/nil_pv_repro_test.go
  ```

  Fix: hoist the nil-check before the address comparison, e.g.
  ```go
  var voteErr *types.VoteConflictingVotesError
  if goerrors.As(err, &voteErr) {
      if cs.privValidator != nil {
          addr := cs.privValidator.PubKey().Address()
          if vote.ValidatorAddress == addr {
              cs.Logger.Error("Found conflicting vote from ourselves...", ...)
              return added
          }
      }
      cs.evpool.ReportConflictingVotes(voteErr.VoteA, voteErr.VoteB)
      ...
  }
  ```
  Add a `TestConflictingVoteOnNonValidator` test that mirrors the repro above so this regression is caught.
  </details>

## Warnings (should fix)

- **[`don't merge` label + CHANGES_REQUESTED]** PR-level — bot lists `Must not contain the "don't merge" label` as a hard fail; `@thehowl` left a CHANGES_REQUESTED "blocking until @jaekwon's review". Both predate the most recent commit and are still in effect.
  <details><summary>details</summary>

  The PR is technically mergeable (`mergeable: MERGEABLE`, all CI green except the gating bot), but the merge requirements bot is the one check that fails, and the human gate (`@thehowl`) hasn't been cleared. Don't drop these without `@jaekwon`'s signoff and an explicit decision on the label.
  </details>

- **[silent evidence loss]** [`tm2/pkg/bft/consensus/state.go:97`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L97) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L97) — `NoOpEvidencePool.ReportConflictingVotes` drops `voteA`/`voteB` on the floor; only a `Warn` log line remains. The PR body and `@tbruyelle`'s comment ("There's no punishment. […] Allowing validators to double vote for free is dangerous, because this promotes fork of the chain") acknowledge this is a stopgap.
  <details><summary>details</summary>

  Splitting the consensus-halt fix from the punishment is a defensible engineering call — the immediate liveness bug is independent of slashing. But landing the placeholder without a tracked follow-up (issue link, ADR, or at least a `TODO(#NNNN)` in the comment) means the "TODO: replace with a real evidence pool that persists double-signing evidence and enables validator slashing" comment can sit there indefinitely while validators free-ride. Fix: open the follow-up issue, link it from the `TODO`, and from the PR description; this also answers `@julienrbrt`'s "How is submitted an evidence then? Can you do that manually?" — it isn't, and there is no manual path either.
  </details>

- **[generic-error log restored, but now noisier than the original silenced it for]** [`tm2/pkg/bft/consensus/state.go:732-735`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L732-L735) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L732-L735) — the upstream tendermint comment `// Causes TestReactorValidatorSetChanges to timeout` was the reason this log was commented out; re-enabling it without verifying the underlying race is fixed risks reintroducing that flake.
  <details><summary>details</summary>

  The previous code wrapped the conflicting-vote case under `ErrAddingVote` and then suppressed the log entirely with the `//nolint:staticcheck` block. After this PR, `tryAddVote` returns no error, so only `ProposalMessage` / `BlockPartMessage` errors flow into this log path — which is what the author argues for in their reply to `@julienrbrt`. But the original comment cited `TestReactorValidatorSetChanges` flakiness as the reason for suppression; that test still exists and the upstream tendermint issue [#3406](https://github.com/tendermint/tendermint/issues/3406) was never fully closed. Fix: run `go test -run TestReactorValidatorSetChanges -count=20 ./tm2/pkg/bft/consensus/` and confirm no timeouts before declaring this safe; if it still flakes, scope the log to `BlockPartMessage` errors that aren't wrong-round (the only realistic new error source) instead of the generic `if err != nil`.
  </details>

## Nits

- [`tm2/pkg/bft/consensus/state.go:85-90`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L85-L90) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L85-L90) — comment says "interface to the evidence pool" but lists only one method; align with `txNotifier`'s style — single sentence, no method enumeration in the godoc since the signature is right below.
- [`tm2/pkg/bft/consensus/state.go:92-97`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L92-L97) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L92-L97) — `NoOpEvidencePool` is exported but the interface `evidencePool` is unexported; that's fine for the wiring in [`node.go:344`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/node/node.go#L344) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/node/node.go#L344), but a future caller that wants to inject a real pool will need either the interface exported or a constructor. Worth a one-line `// satisfies the evidencePool interface` doc on the type to telegraph intent.
- [`tm2/pkg/bft/consensus/state.go:1546-1551`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L1546-L1551) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L1546-L1551) — `cs.Logger.Error("Found conflicting vote from ourselves...")` includes the full `voteA`/`voteB` objects; these are large amino structs that will balloon the log line. Either drop them (the round/height already pin the event) or log their hashes only.

## Missing Tests

- **[non-validator path]** [`tm2/pkg/bft/consensus/state_test.go:1820`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state_test.go#L1820) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state_test.go#L1820) — both new tests run against a validator (`cs.privValidator` is always set by `randConsensusState`). The Critical above is the missing third case — a `TestConflictingVotesOnNonValidator` that clears `privValidator` and asserts no panic + `ReportConflictingVotes` called.
  <details><summary>details</summary>

  This is the test that would have caught the bug. Pair it with the existing two (peer / self) and the matrix is complete: `{validator-self, validator-peer, non-validator-peer}`.
  </details>

- **[wiring to `node.go` is uncovered]** [`tm2/pkg/bft/node/node.go:344`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/node/node.go#L344) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/node/node.go#L344) — the only production caller of `NewConsensusState` passes `cs.NoOpEvidencePool{}` literally; no integration test verifies that wiring. Low priority (it's a one-liner) but worth a comment in the PR that this path is exercised exclusively via end-to-end node startup.

## Suggestions

- [`tm2/pkg/bft/consensus/state.go:1542-1543`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L1542-L1543) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L1542-L1543) — `goerrors.As` is the right call (per `@tbruyelle`'s thread); consider also using `goerrors.As` on `ErrVoteHeightMismatch` for symmetry instead of `goerrors.Is`. Functionally equivalent for sentinel errors, but consistent.
  <details><summary>details</summary>

  `errors.Is` is the conventional check for sentinels; the inconsistency reads as if one was added carefully and the other casually. Leave as-is if you prefer the sentinel convention — just a style harmonization.
  </details>

- Consider an ADR under `tm2/adr/` per the AGENTS.md rule "Every non-trivial AI-assisted PR must include an ADR." The decision to ship a `NoOpEvidencePool` ahead of the real evidence pool, and to drop `tryAddVote`'s error return, are exactly the kind of "what was understood" notes ADRs exist for. If this PR wasn't AI-assisted, ignore.

## Questions for Author

- Is there a tracking issue for the real evidence pool? The `TODO` at [`state.go:93-94`](https://github.com/gnolang/gno/blob/aed66be4a/tm2/pkg/bft/consensus/state.go#L93-L94) · [↗](../../../../../.worktrees/gno-review-5216/tm2/pkg/bft/consensus/state.go#L93-L94) is unanchored; linking it would let `@julienrbrt`'s and `@tbruyelle`'s concerns be tracked in one place.
- What's the plan for the `don't merge` label and the `@jaekwon` review block? The PR has been APPROVED by four reviewers (`mvallenet`, `julienrbrt`, `ltzmaxwell`, `tbruyelle`) and CHANGES_REQUESTED by one (`thehowl`); does `@jaekwon` need to chime in before this can land?
- Did you run `TestReactorValidatorSetChanges` repeatedly after un-silencing the generic error log? The upstream comment specifically cited that test as the reason for the suppression.
