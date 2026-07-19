# PR [#5951](https://github.com/gnolang/gno/pull/5951): feat(examples): add N-of-M multisig treasury demo realm

URL: https://github.com/gnolang/gno/pull/5951
Author: zardozmonopoly | Base: master | Files: 3 | +249 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 9208bed41 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5951 9208bed41`

**TL;DR:** Adds a demo realm where three named owners jointly control a pot of coins: any owner can propose a payment, and once enough of them approve it, the money moves. The realm holds the coins itself, so its own address is the wallet.

**Verdict: REQUEST CHANGES** — the address accessor people are told to fund resolves to the caller's realm rather than the treasury, and `Setup`/`Propose` accept inputs that leave the treasury unspendable (1 Critical, 3 Warnings, 1 Missing test, 4 Nits, 4 Suggestions).

## Summary

The realm keeps an owner set, a threshold, and a proposal tree, and pays out through a `BankerTypeRealmSend` banker on its own address. The state machine itself holds up: [`Execute`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L114) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L114) marks the proposal executed before it calls the banker, so re-entering it mid-send finds `executed` already set, and a 2-of-3 payout settles for real on a live node.

Three input paths are unguarded. [`TreasuryAddress`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L120-L122) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L120-L122) reads the realm identity by walking the frame stack, so it answers with whoever asked. [`Setup`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L50-L53) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L50-L53) writes three addresses into a set and then records the count as 3 regardless of how many landed. [`Propose`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L67-L81) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L67-L81) stores the payout amount without looking at its sign. Each one ends in coins that cannot be recovered or a proposal owners waste approvals on.

## Examples

| Call | Result at 9208bed41 |
|---|---|
| `qeval multisig.TreasuryAddress()` | `g17vd2lug0kdaeahm9sv3y0udz7pac9kc0kqs0aa`, the multisig address |
| same accessor read from `r/demo/wrapper` | `g1ndz9e3hkgyz2xgzs9zuj5au9lf8hfncftf7vwh`, the wrapper's own address |
| `Setup(A, A, A, 2)` | accepted; one owner in the tree, `ownerCount` 3, threshold unreachable |
| `Propose("drain", A, -500000)` | accepted; reverts only inside `Execute`, after two approvals |

## Glossary

- realm: a stateful on-chain package under `r/`; `cur.Previous()` is the caller, bare `cur.Address()` is the realm's own address.
- crossing / `cross`: a call into `func F(cur realm, ...)`, invoked as `cross(cur)`.
- banker: `chain/banker`, the stdlib API a realm uses to move coins.
- unsafe: `chain/runtime/unsafe`, the quarantined stack-walking primitives.
- addpkg: the `maketx addpkg` transaction that uploads a realm.

## Critical (must fix)

- **[funds routed to the wrong address]** `examples/gno.land/r/demo/multisig/multisig.gno:120-122` — [`TreasuryAddress`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L120-L122) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L120-L122) returns the calling realm's address, not the treasury's, so a realm that composes with this one sends deposits to itself.
  <details><summary>details</summary>

  The accessor reads the realm identity through [`unsafe.CurrentRealm()`](https://github.com/gnolang/gno/blob/9208bed41/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L38-L40) · [↗](../../../../../.worktrees/gno-review-5951/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L38-L40), which walks the frame stack rather than naming the package it is written in. `TreasuryAddress` is non-crossing, so [`GetRealm`](https://github.com/gnolang/gno/blob/9208bed41/gnovm/stdlibs/internal/execctx/realm.go#L9-L41) · [↗](../../../../../.worktrees/gno-review-5951/gnovm/stdlibs/internal/execctx/realm.go#L9-L41) skips its frame and resolves the last realm seen instead.

  On a live node the direct `vm/qeval` read is right and the cross-realm read is wrong: a wrapper realm forwarding the call got back `g1ndz9e3hkgyz2xgzs9zuj5au9lf8hfncftf7vwh`, which is that wrapper's own package address, while the treasury sits at `g17vd2lug0kdaeahm9sv3y0udz7pac9kc0kqs0aa`. Coins sent to the answer are unrecoverable, since the wrapper has no banker for them. This is the case [§5.8 of the security guide](https://github.com/gnolang/gno/blob/9208bed41/docs/resources/gno-security-guide.md?plain=1#L381-L412) · [↗](../../../../../.worktrees/gno-review-5951/docs/resources/gno-security-guide.md#L381-L412) calls a red flag: `chain/runtime/unsafe` imported by a realm that also declares crossing functions. Repro in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md); test artifact at [`tests/treasury_address.txtar`](tests/treasury_address.txtar).

  Fix: derive the address from something scoped to this package, either `chain.PackageAddress` on its own pkgpath or a `cur realm` parameter whose `cur.Address()` the runtime mints for this frame.
  </details>

## Warnings (should fix)

- **[treasury can be locked at setup]** `examples/gno.land/r/demo/multisig/multisig.gno:47-55` — [`Setup`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L47-L55) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L47-L55) accepts repeated owner addresses and still records `ownerCount = 3`, so a threshold no owner set can reach becomes permanent.
  <details><summary>details</summary>

  The three `owners.Set` calls collapse duplicates into one tree entry, then [line 53](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L53) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L53) hardcodes the count. `Setup(A, A, A, 2)` leaves `owners.Size() == 1` while `ownerCount` reports 3, verified by running it against the realm. [`Approve`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L95-L99) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L95-L99) rejects a second approval from the same address, so `approveCnt` stops at 1 and `Execute` always panics with `not enough approvals`.

  `initialized` is one-shot with no recovery path, so the realm and everything ever sent to its address are stuck. The [threshold bound](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L47) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L47) compares against the literal 3 rather than the owner set, so `Setup(A, A, B, 3)` has the same shape. Test artifact at [`tests/setup_validation_test.gno`](tests/setup_validation_test.gno).

  Fix: reject a Setup whose owner addresses are not distinct, and bound the threshold by the number of owners actually stored.
  </details>

- **[approvals spent on a proposal that cannot settle]** `examples/gno.land/r/demo/multisig/multisig.gno:67-81` — [`Propose`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L67-L81) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L67-L81) stores the payout amount without checking its sign, so a negative or zero payout is only refused at `Execute`.
  <details><summary>details</summary>

  The bank keeper rejects non-positive amounts in [`SubtractCoins`](https://github.com/gnolang/gno/blob/9208bed41/tm2/pkg/sdk/bank/keeper.go#L210-L213) · [↗](../../../../../.worktrees/gno-review-5951/tm2/pkg/sdk/bank/keeper.go#L210-L213) via [`Coins.validate`](https://github.com/gnolang/gno/blob/9208bed41/tm2/pkg/std/coin.go#L243-L252) · [↗](../../../../../.worktrees/gno-review-5951/tm2/pkg/std/coin.go#L243-L252), so no coins move the wrong way. The cost is process: on a live node `Propose("drain", ownerA, -500000)` returned proposal id 1, both owners paid for an `Approve` transaction, and only `Execute` reverted with `invalid coins error` out of [`banker.SendCoins`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L117) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L117).

  With no cancel path the dead proposal also stays in the tree and in `Render` forever. Repro in [comment_claude-opus-4-8.md](comment_claude-opus-4-8.md); test artifact at [`tests/multisig_flow.txtar`](tests/multisig_flow.txtar).

  Fix: reject a non-positive amount in `Propose`, before any owner can approve it.
  </details>

- **[approved payout stays live forever]** `examples/gno.land/r/demo/multisig/multisig.gno:83-118` — an approval can never be withdrawn and a proposal never expires, so a payout the owners changed their mind about remains executable by anyone.
  <details><summary>details</summary>

  [`Approve`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L83-L100) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L83-L100) only ever adds, and there is no `Revoke` or `Cancel` anywhere in the file. [`Execute`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L102-L118) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L102-L118) checks only the approval count, and takes no caller check by design, so once a proposal reaches threshold any address can settle it at any later block.

  A payout proposed when the treasury was empty stays armed and fires the moment someone funds the address, which is a surprising outcome for a demo teaching the pattern. cw3-flex-multisig, the model the PR description names, expires proposals for this reason.

  Fix: let an owner withdraw an approval before execution, or give a proposal a deadline after which it can no longer settle.
  </details>

## Nits

- **[proposals listed out of order]** [`examples/gno.land/r/demo/multisig/multisig.gno:79`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L79) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L79) — proposal ids are stored as decimal strings, so `Render` lists #1, #10, #11, #2 once there are ten of them. Confirmed behaviorally by proposing eleven payouts and reading `Render`. Zero-padding the key or sorting numerically before printing fixes the display.
- **[two sources of truth for one count]** [`examples/gno.land/r/demo/multisig/multisig.gno:21`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L21) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L21) — `approveCnt` tracks what `approvals.Size()` already knows, and `Execute` trusts the counter. They agree today; dropping the field removes the chance they stop agreeing.
- **[render ignores its argument]** [`examples/gno.land/r/demo/multisig/multisig.gno:124`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L124) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L124) — `Render` takes `path` and never reads it, so `:1` and `:anything` all render the full list. A per-proposal view is the usual second case.
- **[proposals unreadable from other realms]** [`examples/gno.land/r/demo/multisig/multisig.gno:14-22`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L14-L22) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L14-L22) — `Proposal` is exported but every field is unexported and there are no accessors, so the only way to read a proposal is to scrape `Render` output.

## Missing Tests

- **[the multi-owner path is untested]** `examples/gno.land/r/demo/multisig/multisig_test.gno:8-101` — every test runs as the same caller and the suite depends on file order, so nothing covers a threshold reached by distinct owners or a payout that actually moves coins.
  <details><summary>details</summary>

  The tests are in-package and each one calls `Propose`/`Approve` with the same `cur`, so [`TestApproveIncrementsCount`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig_test.gno#L62-L71) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig_test.gno#L62-L71) only ever reaches 1 of 2. The state machine's whole point, distinct approvals crossing the threshold, is exercised nowhere; the PR description covers it with a manual Test13 run instead.

  The suite also only passes as a whole. `gno test -run TestProposeCreatesProposal` panics with `multisig not initialized`, because the package state that makes it work is left behind by [`TestSetupInitializesCorrectly`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig_test.gno#L8-L24) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig_test.gno#L8-L24). Six of the eight fail standalone for the same reason. `Setup` being one-shot makes this hard to fix in place: a second configuration cannot be built in the same test binary.

  The closing cases live in [`tests/multisig_flow.txtar`](tests/multisig_flow.txtar), which runs the 2-of-3 payout across three real keys on a node and asserts the recipient balance, and [`tests/setup_validation_test.gno`](tests/setup_validation_test.gno), which is an external test package so callers can differ.
  </details>

## Suggestions

- **[activity invisible to indexers]** `examples/gno.land/r/demo/multisig/multisig.gno:39-118` — nothing emits an event, so there is no way to follow the treasury without polling `Render`.
  <details><summary>details</summary>

  `Setup`, `Propose`, `Approve`, and `Execute` all mutate state silently. `chain.Emit` on at least `Propose` and `Execute` would let an indexer or a wallet show pending payouts, and a demo realm is where readers first see the pattern.
  </details>

- **[only the uploader can ever configure it]** `examples/gno.land/r/demo/multisig/multisig.gno:34-46` — the deployer gate plus one-shot `Setup` means a single owner set exists for the life of the realm, chosen by whoever ran addpkg.
  <details><summary>details</summary>

  [`init`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L34-L37) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L34-L37) records the uploading account and [`Setup`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L41-L46) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L41-L46) refuses anyone else, confirmed on a node where only the uploading key could call it. Realms under `examples/` are uploaded at genesis, so on a public chain that account is the genesis deployer and no user can try the demo.

  A factory that mints a multisig per caller, or per-instance state keyed by creator, would make it something a reader can actually run.
  </details>

- **[unbounded proposal list]** `examples/gno.land/r/demo/multisig/multisig.gno:132-144` — [`Render`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L132-L144) · [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L132-L144) walks every proposal ever created, and there is no cap on `Propose` and no way to remove a settled one. An owner can grow the page until rendering it is impractical. Not posted: for a demo the growth is theoretical, and the cancel path it needs is already the third Warning.

- **[fixed three-owner signature]** `examples/gno.land/r/demo/multisig/multisig.gno:39` — the PR description already raises generalizing `Setup` past three owners. Passing a comma-separated string and splitting it inside is the pattern other realms use for list arguments over `maketx call`. Not posted: the author raised it first, so a comment adds nothing.

## Verified

- The 2-of-3 flow settles for real. On an integration node with three separate keys: Setup, fund the realm address, propose, two approvals from distinct owners, execute; the recipient's `bank/balances` rose by exactly the payout. Artifact at [`tests/multisig_flow.txtar`](tests/multisig_flow.txtar).
- `TreasuryAddress()` disagrees with itself by caller. Read through `vm/qeval` it returns the multisig package address; read through a wrapper realm on the same node it returns the wrapper's package address, matching `DerivePkgBech32Addr("gno.land/r/demo/wrapper")`. Artifact at [`tests/treasury_address.txtar`](tests/treasury_address.txtar).
- A negative payout reverts, and only at the end. `Propose` returned an id, both owners approved, `Execute` failed with `invalid coins error`; no coins moved in either direction.
- Duplicate owners lock the treasury. `Setup(A, A, A, 2)` left `owners.Size() == 1` against `ownerCount == 3`, and `Execute` then panicked with `not enough approvals` after the single owner's only approval.
- No divergence from Go in the state machine. A Go mirror of Setup/Propose/Approve/Execute reproduces the duplicate-owner lockout, the unchecked amount, the ownerless `Execute`, and the lexicographic id order identically. Artifact at [`tests/statemachine_go_test.go`](tests/statemachine_go_test.go).
- `gno lint` is clean on the package, checked against a planted undefined symbol to confirm it typechecks.
- `gno test` green on `examples/gno.land/r/demo/multisig` at 9208bed41 (8 tests).

## Open questions

- `Execute` takes no caller check, so any address can settle an approved proposal. This matches cw3-flex and the PR describes it as intended; not posted.
- CI never ran the gno test suite here: the workflow jobs show `skipping` pending a maintainer's initial approval. Nothing for the author to do; not posted.
