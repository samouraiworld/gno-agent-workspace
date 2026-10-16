# Review: [#6317](https://github.com/gnolang/gno/pull/6317)
Event: APPROVE
Verdict: APPROVE. Two Nits on the activation step remain open, the open-proposal refusal with nothing to check and no activation route for a chain booted fresh, and neither blocks a design record.
Model: claude-opus-5-5, quick review, solo: finder at xhigh, judge and writer at high
Commit: cffd989cd
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/clockworkgr/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070) · [vscode.dev](https://vscode.dev/github/clockworkgr/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6317 cffd989cd`
Round: 1. One finder, then one judge and writer answering the completeness question, four candidates, each read from scratch by an agent that was not its finder, all of them Nits on the ADR's own text; one refuted, one confirmed and skipped.

## Body

## gno.land/adr/pr6317_name_transfer.md:155 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L155) · [↗](../../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L155) · Nit

Nit: activation has no list of open address-bound proposals to refuse on, since [`ProposeUpdateName`](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/examples/gno.land/r/sys/users/admin.gno#L143-L166) · [↗](../../../../../.worktrees/gno-review-6317/examples/gno.land/r/sys/users/admin.gno#L143) and [`ProposeDeleteUser`](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/examples/gno.land/r/sys/users/admin.gno#L170-L184) · [↗](../../../../../.worktrees/gno-review-6317/examples/gno.land/r/sys/users/admin.gno#L170) record none in `r/sys/users`.

<details><summary>where the refusal lands</summary>

- Both functions only return `dao.NewProposalRequest(..., dao.NewSimpleExecutor(0, cur, cb, ""))`, and [`SimpleExecutor`](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/examples/gno.land/r/gov/dao/types.gno#L185-L188) · [↗](../../../../../.worktrees/gno-review-6317/examples/gno.land/r/gov/dao/types.gno#L185) keeps the callback, a description and the creation realm, so finding an open proposal means walking GovDAO's proposal store.
- Once the activation tx does refuse, the chain does not boot: the strict replay the ADR requires [panics on any failed genesis tx](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/pkg/gnoland/app.go#L651-L658) · [↗](../../../../../.worktrees/gno-review-6317/gno.land/pkg/gnoland/app.go#L651).

</details>

## gno.land/adr/pr6317_name_transfer.md:414 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L414) · [↗](../../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L414) · Nit

Nit: activation needs a genesis tx at height 0, and nothing appends one for gnodev, a fresh testnet or the Testing section's `.txtar`. Those chains refuse `OfferTransfer` and `AcceptTransfer` for good, while the keeper rule applies from block 1.

<details><summary>where activation can come from</summary>

- The fork tool's `--migration-tx` is the one path that appends a tx after history, in [`generate.go`](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/contribs/gnogenesis/internal/fork/generate.go#L373-L377) · [↗](../../../../../.worktrees/gno-review-6317/contribs/gnogenesis/internal/fork/generate.go#L373).
- No loader under `contribs/gnodev` or `gno.land/pkg/integration` references a migration tx, and the ADR names no other route.
- Moving activation into the `r/sys/users` `init()` does not help, since a replay upgrade runs that `init()` at its historical position and would end dormancy early.

</details>

## SKIP gno.land/adr/pr6317_name_transfer.md:91 [gh](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/gno.land/adr/pr6317_name_transfer.md#L91) · [↗](../../../../../.worktrees/gno-review-6317/gno.land/adr/pr6317_name_transfer.md#L91) · Nit

Nit: `ResolveID` is listed as read-only API without saying it returns the stored record, like [`ResolveAddress`](https://github.com/gnolang/gno/blob/cffd989cd91b34383b16dd902b41796b736a1070/examples/gno.land/r/sys/users/users.gno#L21-L32) · [↗](../../../../../.worktrees/gno-review-6317/examples/gno.land/r/sys/users/users.gno#L21), rather than a `makeUserDataSafe` copy, which the authoritative guard refuses.

Not posted: the offer and accept filetest the Testing section requires fails on a copy, so an implementation cannot ship the wrong reading unnoticed.
