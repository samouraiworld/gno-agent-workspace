# PR #4944: feat(govdao): add proposal fee-based for non-member

URL: https://github.com/gnolang/gno/pull/4944
Author: davd-gzl | Base: master | Files: 9 | +351 -83
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]

**Verdict: REQUEST CHANGES** — fee is checked but never captured: with MsgRun the coins never leave the caller (zero-cost spam), with MsgCall they land on the `r/gov/dao` proxy address with no withdrawal path; PR description claims they go to "GovDAO implementation" which is wrong for both paths.

## Summary

The PR opens proposal creation to non-members behind a 1 GNOT fee (settable via member-only proposal, 0 disables it), while keeping member-only `requireCallerMember` fast-fails on sensitive helpers (law change, impl upgrade, fee change). The `PreCreateProposal` gate validates that non-members send exactly one coin in `ugnot` of the configured amount, then lets `dao.CreateProposal` proceed. A `[non-member]` badge is also added to the rendered proposal author. The functional contribution is real (anti-spam toll on the spam vector that was just opened); what is missing is the second half of the design — actually collecting and accounting the fee so it ends up somewhere usable (treasury) and so MsgRun is not a free bypass.

```
MsgCall  → -send 1000000ugnot → r/gov/dao.CreateProposal
   bank.SendCoins(caller, pkgAddr=r/gov/dao, send)   <-- coins parked on PROXY
   dao.PreCreateProposal()                            <-- banker.OriginSend() == 1000000ugnot, OK
   <no SendCoins to treasury/impl>                    <-- fee stranded on proxy

MsgRun   → -send 1000000ugnot → caller's run package
   bank.SendCoins(caller, pkgAddr=caller, send)       <-- caller self-send, zero net flow
   dao.PreCreateProposal()                            <-- banker.OriginSend() == 1000000ugnot, OK
   <no SendCoins>                                     <-- caller keeps the "fee"
```

## Glossary

- `ProposalFeeAmount` — top-level `var int64` in `r/gov/dao/v3/impl`, mutable via `NewProposalFeeAmountRequest`. Single source of truth for the non-member toll.
- `PreCreateProposal` — `GovDAO` hook called by the `r/gov/dao` proxy before persisting a proposal. Returns `(author, error)`; the only place the fee is checked.
- `requireCallerMember` — new helper in `types.gno` that panics if `runtime.OriginCaller()` is not in `memberstore`. Used by member-only request builders as a spam fast-fail (not a security boundary; package-private `var`s are the real boundary).
- `OriginSend` — `banker.OriginSend()` returns the `Send` field of the originating `MsgCall`/`MsgRun`. Independent of where the bank actually moved the coins.

## Fix

Before: `PreCreateProposal` rejected every non-member with `"only members can create new proposals"`, and individual request helpers (`NewChangeLawRequest`, `NewUpgradeDaoImplRequest`, `NewAddMemberRequest`) each duplicated their own `"proposer is not a member"` panic. After: `PreCreateProposal` lets non-members through if `banker.OriginSend()` is exactly one `ugnot` coin equal to `ProposalFeeAmount` (or if the fee is 0); the member-only helpers route through a single `requireCallerMember` (spam fast-fail). `NewAddMemberRequest` keeps the invitation-point deduction for member proposers, skips it for non-members. A new member-only `NewProposalFeeAmountRequest` updates the global `ProposalFeeAmount`. Render now appends `` `[non-member]` `` next to the author.

## Critical (must fix)

- **[fee never reaches the DAO]** [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L80-L88`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L80-L88) — PR body says "Coins are stored in GovDAO implementation, and can be use by the DAO" but no code path moves the fee there.
  <details><summary>details</summary>

  `PreCreateProposal` only *checks* `banker.OriginSend()`. It never calls `banker.SendCoins(...)` to forward the fee anywhere. Where the coins actually end up is decided entirely by the transport layer in [`gno.land/pkg/sdk/vm/keeper.go`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/sdk/vm/keeper.go#L770-L799): for `MsgCall` the bank credits `pkgAddr := gno.DerivePkgCryptoAddr(pkgPath)` = the proxy `r/gov/dao`, not `r/gov/dao/v3/impl`; for `MsgRun` [`pkgAddr := caller`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/sdk/vm/keeper.go#L942) so it's a self-send and the fee never leaves the user's account. Either way `r/gov/dao/v3/impl` never sees the coins, and `r/gov/dao/v3/treasury` uses `BankerTypeRealmSend` against its own address ([`treasury.gno:39`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L39)) so there's no way to extract from the proxy through governance either. Fix: in `PreCreateProposal` (non-member branch), construct a `BankerTypeRealmSend` banker from `impl` and `SendCoins` the fee from the proxy/run-pkg address into the treasury's banker address (queryable via `treasury.Address("Coins")`), or to a dedicated `impl` vault address. Without that, "non-member fee" is a checksum on `OriginSend`, not a fee.
  </details>

- **[MsgRun is a free bypass]** [`gno.land/pkg/sdk/vm/keeper.go#L942`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/sdk/vm/keeper.go#L942) — the existing integration test exercises exactly this path and passes without any net debit to the non-member.
  <details><summary>details</summary>

  `gno.land/pkg/sdk/vm/keeper.go` line 942 sets `pkgAddr := caller` for `MsgRun`, so `bank.SendCoins(ctx, caller, pkgAddr, send)` is a no-op transfer. `OriginSend` is still populated to `send`, so [`banker.OriginSend()`](../../../../../.worktrees/gno-review-4944/gnovm/stdlibs/chain/banker/banker.gno#L173-L180) returns the declared amount and the fee check passes — but the coins never moved. The new [`govdao_proposal_nonmember.txtar`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/integration/testdata/govdao_proposal_nonmember.txtar#L19) uses `gnokey maketx run -send "1000000ugnot" ...` and the proposal succeeds; the test never asserts the non-member balance dropped by 1 GNOT, because it wouldn't. So once this PR ships, anyone with a positive ugnot balance can spam proposals via `MsgRun` for the gas cost alone. Fix: requires the same change as above (an actual `SendCoins` inside `PreCreateProposal`) — once `impl` pulls the coins from the OriginCaller into a vault, MsgRun and MsgCall become symmetric. Add a balance-before/balance-after assertion in the txtar so a future regression on this is caught.
  </details>

## Warnings (should fix)

- **[fee equality is a footgun for users]** [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L82`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L82) — overpayment by 1ugnot rejects, so any UX that pre-funds a rounded amount blows up.
  <details><summary>details</summary>

  The check is `sent[0].Amount != ProposalFeeAmount`. This was tightened in response to [ajnavarro's review comment on overpayment absorption](https://github.com/gnolang/gno/pull/4944#discussion_r2056130612), which is fair as long as the fee is captured. With the current "no SendCoins" implementation, exact-match buys nothing (the coins don't move anyway); it just turns a typo into a failed tx the user paid gas for. Once Critical #1 is fixed and the fee is actually transferred, the better pattern is `sent[0].IsGTE(minFee)` plus an explicit refund of `sent[0] - minFee` back to `OriginCaller`, mirroring [`r/gnops/valopers/valopers.gno:82-89`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gnops/valopers/valopers.gno#L82-L89) (which only does the min-fee half). At minimum, document the exact-amount requirement in the error message, which already does — but also surface it on the dao realm Render so wallets/dapps can pre-fill.
  </details>

- **[ProposalFeeAmount is a public mutable var, racy with proposals in flight]** [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L17`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L17) — `var ProposalFeeAmount int64 = 1_000_000` is exported and writable by any code in package `impl`; the fee-change executor mutates it directly.
  <details><summary>details</summary>

  Because the var is exported, any future code in `impl` (or any file added later, intentionally or by mistake) can rewrite it without going through `NewProposalFeeAmountRequest`. The "members only" gate exists only in the request builder, not on the variable. Fix: make the variable lowercase (`proposalFeeAmount`) and expose `GetProposalFeeAmount() int64` for external reads; the executor still writes via package-internal access. Also: the description string in [`prop_requests.gno:248-256`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L248-L256) captures `ProposalFeeAmount` at *request creation* time, so if another fee-change proposal lands between creation and execution of this one, the rendered "from X to Y" text becomes stale. Either snapshot at execution time or be explicit that X is "the value when the proposal was filed". Minor in practice but worth a sentence in the description.
  </details>

- **[member badge is read-at-render, decays on departure]** [`examples/gno.land/r/gov/dao/v3/impl/render.gno#L204-L210`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/render.gno#L204-L210) — `renderMembershipBadge(p.Author())` queries memberstore at render time, not at proposal time.
  <details><summary>details</summary>

  A member who proposes, then is removed via `NewWithdrawMemberRequest`, will retroactively get the `[non-member]` label on every past proposal they authored — the historical truth ("they were a member when they filed this") is lost. Conversely, a non-member who paid the fee and later joined the DAO will lose the badge on their old proposals. Tiny issue, but the rendered page is a public artifact and should reflect the state at proposal time. Fix: snapshot `wasMember bool` on the `Proposal` at creation (in `PostCreateProposal`, or pass it through `ProposalRequest`) and render from that.
  </details>

- **[stale comment]** [`examples/gno.land/r/gov/dao/v3/impl/types.gno#L67-L70`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/types.gno#L67-L70) — `requireCallerMember` docstring says "real protection is package privacy on sensitive state" but the state it gates (`ProposalFeeAmount`, `law`) is exposed via uppercase exported names.
  <details><summary>details</summary>

  `ProposalFeeAmount` is exported (`govdao.gno:17`), and `law` in `impl.go` is lowercase but the `Law` struct it points at is exported and reassignable through the executor pattern. The comment overstates how much "package privacy" buys here. Fix: align the comment with reality, or fix the underlying export (see Warning above) so the claim becomes true.
  </details>

## Nits

- [`examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L223-L257`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L223-L257) — `NewProposalFeeAmountRequest` panics on `< 0` then later on `reason == ""`; flip the cheap string check first so callers get the cheap failure faster (matches the ordering in `NewWithdrawMemberRequest` / `NewTreasuryPaymentRequest`).
- [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L17`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L17) — `// 1 GNOT` comment is fine but the spelling elsewhere in this repo is `gnot` / `ugnot` lower-case; minor consistency.
- [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L82`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L82) — string check `sent[0].Denom != "ugnot"` repeats a literal already used by the error message; consider a `const proposalFeeDenom = "ugnot"` at the top of the file.
- [`gno.land/pkg/integration/testdata/govdao_proposal_set_fee_amount.txtar#L13-L15`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/integration/testdata/govdao_proposal_set_fee_amount.txtar#L13-L15) — gas-wanted of 19M for vote and 22M for execute matches the sibling txtars (per the last commit), but it'd be worth a one-line comment "matches govdao_proposal_*.txtar after #5291/#5415" so the next bump is a single git-grep away.

## Missing Tests

- **[bypass coverage]** [`gno.land/pkg/integration/testdata/govdao_proposal_nonmember.txtar`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/integration/testdata/govdao_proposal_nonmember.txtar) — txtar passes for the MsgRun path but never asserts the non-member account actually paid 1 GNOT to anyone.
  <details><summary>details</summary>

  Add a `gnokey query bank/balances g1rfznvu6qfa0sc76cplk5wpqexvefqccjunady0` after the successful proposal and assert the balance dropped (or the proxy/impl/treasury balance rose) by 1 GNOT minus gas. This is the test that would have caught Critical #1 / #2; without it the txtar pattern of "command stdout OK!" silently endorses a no-op.
  </details>

- **[overpay rejection in txtar]** [`gno.land/pkg/integration/testdata/govdao_proposal_nonmember.txtar`](../../../../../.worktrees/gno-review-4944/gno.land/pkg/integration/testdata/govdao_proposal_nonmember.txtar) — the unit test covers overpay/multi-denom/wrong-denom but the txtar does not.
  <details><summary>details</summary>

  Unit tests run against `testing.SetOriginSend`, which is a different code path than the actual bank send. Add Case 4 (overpayment), Case 5 (multi-denom), Case 6 (wrong denom) to the txtar, each asserting the fee message. Cheap and protects the contract end-to-end.
  </details>

- **[fee-disabled end-to-end]** new — `TestNonMemberProposalFeeDisabled` covers the unit path but no txtar exercises `ProposalFeeAmount = 0` + non-member proposal at the integration level.
  <details><summary>details</summary>

  Add a 4th case to `govdao_proposal_set_fee_amount.txtar`: vote a fee change to 0, then have the non-member create a proposal with no `-send`. Confirms the disable path survives a real chain round-trip.
  </details>

## Suggestions

- [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L64-L91`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L64-L91) — once Critical #1 is fixed, emit a `chain.Emit("ProposalFeePaid", ...)` event from the same place that does the `SendCoins`. Right now there's no audit trail for fees, which is the kind of thing block explorers and governance dashboards want to graph.
  <details><summary>details</summary>

  Pattern: `chain.Emit("ProposalFeePaid", "from", caller.String(), "amount", strconv.FormatInt(ProposalFeeAmount, 10))`. Keeps the same shape as `ProposalCreated` in `proxy.gno:89-91`.
  </details>

- [`examples/gno.land/r/gov/dao/v3/impl/render.gno#L97`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/render.gno#L97) — also render the current `ProposalFeeAmount` on the dao home page so prospective non-member proposers know what to send before composing the tx.
- [`examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L223`](../../../../../.worktrees/gno-review-4944/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L223) — `NewProposalFeeAmountRequest` doesn't reject equal-to-current. A fee-change proposal that doesn't change the fee should fail at request-creation time, not waste a vote cycle.

## Questions for Author

- Where do you intend the collected fee to actually live? "GovDAO implementation" in the description doesn't match either MsgCall (proxy address) or MsgRun (caller address) — was the plan to add `SendCoins` to treasury in this PR and it got dropped, or in a follow-up?
- Why did `NewProposalFeeAmountRequest` end up under `requireCallerMember` instead of using `PreCreateProposal`'s fee path? A non-member willing to pay 1 GNOT can't propose a fee change, which is intentional — but the same is true for `NewChangeLawRequest` / `NewUpgradeDaoImplRequest`. Is there a reason to leave `NewAddMemberRequest` / `NewWithdrawMemberRequest` / `NewPromoteMemberRequest` / `NewTreasuryPaymentRequest` open to non-members but gate these three? Worth one sentence in the `PreCreateProposal` doc-comment naming the included vs. excluded request types so the policy doesn't drift.
- Was an upper bound on `ProposalFeeAmount` considered? Currently a single accepted proposal can set the fee to `math.MaxInt64`, effectively closing non-member proposals forever (until another supermajority votes it back down). A `maxProposalFeeAmount` sanity check would prevent a malicious or buggy proposal from softlocking the path.
