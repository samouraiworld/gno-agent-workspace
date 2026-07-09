# PR [#5917](https://github.com/gnolang/gno/pull/5917): feat(govdao-scripts): add test13 updated scripts

URL: https://github.com/gnolang/gno/pull/5917
Author: aeddi | Base: master | Files: 4 | +311 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: c8a5a2f2a (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5917 c8a5a2f2a`

**TL;DR:** Three shell scripts for govDAO operators on the test13 testnet: batch-add validators in a single proposal, push the updated valoper registration instructions, and unlock ugnot transfers chain-wide. Each writes a small Gno program on the fly and runs it through `gnokey maketx run` to create, vote YES on, and execute a govDAO proposal.

**Verdict: APPROVE** — generated Gno is API-accurate and matches the shapes exercised by passing integration tests; only two comment nits and one gas-note suggestion, plus an adjacent staleness in the older singular v3 scripts worth a decision.

## Summary
Adds three scripts to `misc/govdao-scripts/`, in the directory's existing env-parameterized (`GNOKEY_NAME`/`CHAIN_ID`/`REMOTE`) style. Each generates an ephemeral-realm `main(cur realm)` program in a temp dir and hands it to `gnokey maketx run`, which coerces the package to `gno.land/e/<caller>/run` so the crossing `main` is legal on-chain. `add-validators-v3.sh` batches up to 40 operators into one `r/sys/validators/v3` proposal so the valset-update cooldown is consumed once instead of per validator. `set-valoper-instructions.sh` embeds the `r/gnops/valopers` registration guide from [#5842](https://github.com/gnolang/gno/pull/5842) and pushes it through governance to an already-deployed realm. `unlock-transfer.sh` clears the bank `restricted_denoms` param chain-wide via `ProposeUnlockTransferRequest`. No `.gno` files are committed, so this is a tooling-plus-docs change; the invariant catalog does not apply, but the generated Gno's crossing convention was checked against the realm facade and the integration suite.

## Glossary
- ephemeral realm — the short-lived `gno.land/e/<addr>/run` code realm a `maketx run` script executes under.
- crossing / `cross` — a call into a `func F(cur realm, ...)` function, invoked as `cross(cur)`.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`set-valoper-instructions.sh:11-12`](https://github.com/gnolang/gno/blob/c8a5a2f2a/misc/govdao-scripts/set-valoper-instructions.sh#L11-L12) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/set-valoper-instructions.sh#L11-L12) — the comment says "init.gno on this branch still carries the pre-PR text", but [#5842](https://github.com/gnolang/gno/pull/5842) merged 2026-06-22 and the branch's [`init.gno`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/gnops/valopers/init.gno#L21) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gnops/valopers/init.gno#L21) already carries the post-#5842 text, byte-identical to the string the script embeds. The pre-PR text lives only on the deployed test13 realm, which is what the script actually updates. Fix: reword to reference the deployed realm, not the branch.
- [`unlock-transfer.sh:9`](https://github.com/gnolang/gno/blob/c8a5a2f2a/misc/govdao-scripts/unlock-transfer.sh#L9) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/unlock-transfer.sh#L9) — "Use lock-transfer (ProposeLockTransferRequest) to re-lock" points at a `lock-transfer` command that does not exist in the directory; an operator following it finds nothing. Fix: name the `ProposeLockTransferRequest` proposal path directly, or add the script.

## Missing Tests
None. The directory has no in-repo harness that compiles these MsgRun scripts; the three proposal paths they drive are each covered by passing integration tests (see Verified).

## Suggestions
- [`add-validators-v3.sh:28`](https://github.com/gnolang/gno/blob/c8a5a2f2a/misc/govdao-scripts/add-validators-v3.sh#L28) · [↗](../../../../../.worktrees/gno-review-5917/misc/govdao-scripts/add-validators-v3.sh#L28) — a 40-operator batch stores the largest proposal payload of the three scripts, yet only `set-valoper-instructions.sh` carries the "if the tx runs out of gas, raise `GAS_WANTED`" note. Consider adding the same note here.
  <details><summary>details</summary>

  `GAS_WANTED` already defaults to 50000000 and is overridable, so this is documentation, not a bug. The batch is the most likely of the three to exceed the default at 40 entries.
  </details>

## Verified
- Generated Gno compiles and runs against the on-chain realm APIs: each script's emitted create/vote/execute shape matches MsgRun scripts that passing integration tests drive through `gnokey maketx run` — `add-validators-v3` against [`params_valset_multi_entry_same_op.txtar`](https://github.com/gnolang/gno/blob/c8a5a2f2a/gno.land/pkg/integration/testdata/params_valset_multi_entry_same_op.txtar#L48) · [↗](../../../../../.worktrees/gno-review-5917/gno.land/pkg/integration/testdata/params_valset_multi_entry_same_op.txtar#L48) and [`params_valset_proposal_power_update.txtar`](https://github.com/gnolang/gno/blob/c8a5a2f2a/gno.land/pkg/integration/testdata/params_valset_proposal_power_update.txtar#L86) · [↗](../../../../../.worktrees/gno-review-5917/gno.land/pkg/integration/testdata/params_valset_proposal_power_update.txtar#L86), `unlock-transfer` against [`transfer_unlock.txtar`](https://github.com/gnolang/gno/blob/c8a5a2f2a/gno.land/pkg/integration/testdata/transfer_unlock.txtar#L62) · [↗](../../../../../.worktrees/gno-review-5917/gno.land/pkg/integration/testdata/transfer_unlock.txtar#L62), `set-valoper-instructions` against [`valopers.txtar`](https://github.com/gnolang/gno/blob/c8a5a2f2a/gno.land/pkg/integration/testdata/valopers.txtar#L109) · [↗](../../../../../.worktrees/gno-review-5917/gno.land/pkg/integration/testdata/valopers.txtar#L109).
- Every API signature matches the branch source: `NewValidatorProposalRequest(cur, changes, title, desc)` and `NewValoperChange(address, uint64)` in [`proposal.gno:69`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/sys/validators/v3/proposal.gno#L69) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/sys/validators/v3/proposal.gno#L69), `MustVoteOnProposalSimple(cur, int64, "YES")` where [`YesVote = "YES"`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/gov/dao/types.gno#L23) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gov/dao/types.gno#L23), `ProposeUnlockTransferRequest(cur)` in [`unlock.gno:14`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/sys/params/unlock.gno#L14) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/sys/params/unlock.gno#L14), and `ProposeNewInstructionsProposalRequest(cur, string)` in [`proposal.gno:73`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L73) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gnops/valopers/proposal/proposal.gno#L73).
- The instructions text `set-valoper-instructions.sh` embeds is byte-identical to the branch's current [`init.gno`](https://github.com/gnolang/gno/blob/c8a5a2f2a/examples/gno.land/r/gnops/valopers/init.gno#L21) · [↗](../../../../../.worktrees/gno-review-5917/examples/gno.land/r/gnops/valopers/init.gno#L21): a line-by-line diff of the two raw strings differs only on the Register link, where the script swaps the realm-relative `txlink.Call("Register")` for the absolute `txlink.Realm("gno.land/r/gnops/valopers").Call("Register")`. Both resolve to `/r/gnops/valopers$help&func=Register`; the absolute form is required because the ephemeral run realm is not `r/gnops/valopers`, so the relative call would resolve against the wrong realm.
- Argument parsing in `add-validators-v3.sh` rejects the failure shapes: empty address, missing/zero/non-integer power, and more than 40 operators; a duplicate operator is caught downstream at proposal creation (`duplicate operator in proposal`, asserted by `params_valset_multi_entry_same_op.txtar`).

## Open questions
- The older singular scripts `add-validator-v3.sh` and `rm-validator-v3.sh` (not in this diff) still generate the pre-[#5669](https://github.com/gnolang/gno/pull/5669) three-argument `NewValidatorProposalRequest([]ValoperChange{...}, title, desc)` call. The current realm signature takes `cur realm` first, so that call no longer compiles against `r/sys/validators/v3`. #5669 added `cross(cur)` to their vote/execute calls but left the builder call on the old form. This PR's batch script uses the correct four-argument form. Since the PR updates the test13 script set, whether to fix the two singular v3 scripts here is a decision for the author — raised as a Body question in comment.md.
