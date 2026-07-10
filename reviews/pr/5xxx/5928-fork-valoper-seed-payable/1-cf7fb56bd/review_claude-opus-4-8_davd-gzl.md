# PR [#5928](https://github.com/gnolang/gno/pull/5928): fix(gnogenesis): make fork valoper-seed output valid and payable

URL: https://github.com/gnolang/gno/pull/5928
Author: aeddi | Base: master | Files: 4 | +92 -20
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cf7fb56bd (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5928 cf7fb56bd`

**TL;DR:** `gnogenesis fork valoper-seed` writes a file of validator-registration transactions that a hardfork ceremony replays into genesis. Before this change those transactions could not actually run: they had no signatures and a zero fee, both of which the transaction validator rejects. This makes the output replayable and adds a `--caller` flag naming the account that pays the tiny fee.

**Verdict: APPROVE** — correct, minimal fix; only a missing regression test for the core claim and one imprecise flag-help string, neither blocking.

## Summary
The tool emitted `valopers.Register` transactions with an empty signature slice and a zero-amount fee. Genesis replay runs each transaction through the [ante handler](https://github.com/gnolang/gno/blob/cf7fb56bd/tm2/pkg/sdk/auth/ante.go#L104) · [↗](../../../../../.worktrees/gno-review-5928/tm2/pkg/sdk/auth/ante.go#L104), which calls `std.Tx.ValidateBasic`; that rejects a transaction with no signatures and one whose fee denom is empty. The fix gives each transaction one zero-value signature per signer and a 1 ugnot fee, and adds `--caller` so a funded account pays the fee while the operator address moves to `Args[3]`. The squat guard in the realm's `Register` only fires when `ChainHeight() > 0`, so at genesis the caller need not equal the operator. The same signature fix is applied to the sibling `fork addpkg` output, which had the identical empty-slice bug.

## Glossary
- Amino: gno's deterministic serialization codec. A zero-amount `std.Coin` marshals to `""`, dropping its denom, so a round-trip yields an invalid coin.
- ante handler: the pre-execution stage that runs for every tx, including genesis replay; it verifies signatures, deducts the fee, and calls `Tx.ValidateBasic`.

## Fix
`std.NewCoin("ugnot", 0)` marshals through its `MarshalAmino` to the empty string and decodes back to `Coin{Denom:"", Amount:0}`, which fails `Fee.GasFee.IsValid()`; `Amount=1` preserves the denom. See [`coin.go:51-65`](https://github.com/gnolang/gno/blob/cf7fb56bd/tm2/pkg/std/coin.go#L51-L65) · [↗](../../../../../.worktrees/gno-review-5928/tm2/pkg/std/coin.go#L51-L65) and the transaction rules in [`tx.go:44-58`](https://github.com/gnolang/gno/blob/cf7fb56bd/tm2/pkg/std/tx.go#L44-L58) · [↗](../../../../../.worktrees/gno-review-5928/tm2/pkg/std/tx.go#L44-L58). The signature slice is sized to `len(GetSigners())`, mirroring [`genesis.go:241`](https://github.com/gnolang/gno/blob/cf7fb56bd/gno.land/pkg/gnoland/genesis.go#L241) · [↗](../../../../../.worktrees/gno-review-5928/gno.land/pkg/gnoland/genesis.go#L241). Signature verification is skipped at genesis under `--skip-genesis-sig-verification`, so the placeholder is never checked.

## Examples
| CSV row / flag | Emitted MsgCall field | Value |
|---|---|---|
| `--caller g1jg8…qf5` | `Caller` | `g1jg8…qf5` (fee payer) |
| `operator_addr` col | `Args[3]` | operator address |
| fee | `Fee.GasFee` | `1ugnot` |
| signers | `Signatures` | one zero-value entry |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- **[test aliases two roles onto one address]** [`valoper_seed_test.go:32-35`](https://github.com/gnolang/gno/blob/cf7fb56bd/contribs/gnogenesis/internal/fork/valoper_seed_test.go#L32-L35) · [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed_test.go#L32-L35) — `testCaller = opAddrA`, and `opAddrA` is also an operator row in the same tests, so the "fee payer independent of the operator rows" comment describes an address that is not independent. Decoupling is still proven by the second transaction, where caller `opAddrA` differs from operator `opAddrB`. Not posted: the test is correct, only its comment overstates the setup.

## Missing Tests
- **[core fix has no regression guard]** [`valoper_seed.go:367-372`](https://github.com/gnolang/gno/blob/cf7fb56bd/contribs/gnogenesis/internal/fork/valoper_seed.go#L367-L372) · [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed.go#L367-L372) — no test asserts the emitted transaction passes `ValidateBasic`.
  <details><summary>details</summary>

  The existing tests check `Caller`, `Args`, `Metadata`, and `Reason`, but none round-trips an emitted line and calls `ValidateBasic`, which is the whole point of the change. A revert to a zero fee or an empty signature slice would leave every test green while re-breaking replay. Fix: add a test that unmarshals each emitted line and requires `at.Tx.ValidateBasic()` to succeed. The [`tests/valoper_seed_validatebasic_test.go`](tests/valoper_seed_validatebasic_test.go) shipped here does exactly that; it passes at cf7fb56bd and fails if either fix is reverted.
  </details>

## Suggestions
- **[help understates the balance the caller needs]** [`valoper_seed.go:148-155`](https://github.com/gnolang/gno/blob/cf7fb56bd/contribs/gnogenesis/internal/fork/valoper_seed.go#L148-L155) · [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed.go#L148-L155) — the flag help says the caller needs `>= 1 ugnot`, but every CSV row becomes its own transaction and each deducts 1 ugnot from the same caller.
  <details><summary>details</summary>

  `execValoperSeed` emits one transaction per row ([`valoper_seed.go:185-186`](https://github.com/gnolang/gno/blob/cf7fb56bd/contribs/gnogenesis/internal/fork/valoper_seed.go#L185-L186) · [↗](../../../../../.worktrees/gno-review-5928/contribs/gnogenesis/internal/fork/valoper_seed.go#L185-L186)), all sharing the single `--caller`. The ante handler deducts `tx.Fee.GasFee` for every delivered genesis transaction with no genesis-mode bypass ([`ante.go:173-184`](https://github.com/gnolang/gno/blob/cf7fb56bd/tm2/pkg/sdk/auth/ante.go#L173-L184) · [↗](../../../../../.worktrees/gno-review-5928/tm2/pkg/sdk/auth/ante.go#L173-L184)). An N-row CSV therefore needs the caller funded with N ugnot; funding it with exactly 1 ugnot per the help would abort replay on the second registration. Fix: state that the caller needs balance covering one 1 ugnot fee per row. In practice the usual `--caller` is test1 with a large genesis balance, so this bites only a purpose-funded caller.
  </details>

## Verified
- Amino round-trip (not covered by any existing test): a zero-amount fee marshals to `gas_fee:""` and the decoded transaction fails `ValidateBasic` with "insufficient fee"; a 1 ugnot fee decodes to `1ugnot` and passes. Reverting the signature slice to empty fails `ValidateBasic` with "no signatures". These are the two reverts the fix guards against.
- `MsgCall.GetSigners()` returns exactly the caller ([`msgs.go:167-169`](https://github.com/gnolang/gno/blob/cf7fb56bd/gno.land/pkg/sdk/vm/msgs.go#L167-L169) · [↗](../../../../../.worktrees/gno-review-5928/gno.land/pkg/sdk/vm/msgs.go#L167-L169)), so `make([]std.Signature, len(msg.GetSigners()))` is length 1 and matches `Tx.GetSigners()`.
- Argument order matches the realm: `Register(cur, moniker, description, serverType, addr, pubKey)` at [`valopers.gno:137`](https://github.com/gnolang/gno/blob/cf7fb56bd/examples/gno.land/r/gnops/valopers/valopers.gno#L137) · [↗](../../../../../.worktrees/gno-review-5928/examples/gno.land/r/gnops/valopers/valopers.gno#L137) lines up with `Args[0..4]`, and the squat guard at [`valopers.gno:139`](https://github.com/gnolang/gno/blob/cf7fb56bd/examples/gno.land/r/gnops/valopers/valopers.gno#L139) · [↗](../../../../../.worktrees/gno-review-5928/examples/gno.land/r/gnops/valopers/valopers.gno#L139) is `ChainHeight() > 0 && OriginCaller() != addr`, so `caller != operator` is fine at genesis.
- Consume path: `fork generate` appends migration txs and `deliverGenesisTx` runs each through `baseApp.Deliver` ([`app.go:810`](https://github.com/gnolang/gno/blob/cf7fb56bd/gno.land/pkg/gnoland/app.go#L810) · [↗](../../../../../.worktrees/gno-review-5928/gno.land/pkg/gnoland/app.go#L810)), so `ValidateBasic` really does gate these transactions.
- `go test ./internal/fork/...` in `contribs/gnogenesis` passes at cf7fb56bd, plus the added `TestValoperSeed_EmittedTxPassesValidateBasic`.

## Open questions
- The tool cannot verify the `--caller` account has a genesis balance; that lives in a separate balances file. Not posted: the tool has no visibility into it, and the flag help already states the requirement.
