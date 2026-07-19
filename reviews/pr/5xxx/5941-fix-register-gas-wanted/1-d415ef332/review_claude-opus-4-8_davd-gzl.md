# PR [#5941](https://github.com/gnolang/gno/pull/5941): docs: fix Register gas-wanted and add log-level tip (test13)

URL: https://github.com/gnolang/gno/pull/5941
Author: coinsspor | Base: chain/test13 | Files: 1 | +6 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: d415ef332 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5941 d415ef332`

**TL;DR:** The test13 validator onboarding guide told operators to send the valoper registration transaction with a gas budget that is now too small, so the transaction failed. This raises the documented budget and adds a note about turning down the node's log verbosity.

**Verdict: APPROVE** — both documented values check out against the chain and the code; only a stray blank line and two out-of-diff doc gaps left over (1 Nit).

## Summary

[`VALIDATOR.md`](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L106-L132) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L106-L132) walks a new test13 operator through registering on `gno.land/r/gnops/valopers`. The `Register` call now costs about 82M gas, so the 50M budget the guide used to document aborted with out-of-gas. The diff raises [that line](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L127) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L127) to 100M and adds a [blockquote tip](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L99-L101) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L99-L101) recommending `--log-level info` on the start command. Both are the author's own onboarding findings as an active-set test13 validator, and both hold up: the cited transaction is on chain, and the log-level default and option list match the flag as registered.

## Glossary

- gas-wanted: the per-transaction gas budget the sender commits to; exceeding it aborts the transaction and still charges the fee.
- storage deposit: per-realm refundable charge for on-chain storage, locked on positive byte delta, governed by the `storage_price` and `default_deposit` VM params.

## Benchmarks / Numbers

| Quantity | Value | Source |
| --- | --- | --- |
| `Register` gas used, height 741195 | 81,783,643 | test13 RPC `/tx` |
| Documented gas-wanted, before | 50,000,000 | removed line |
| Documented gas-wanted, after | 100,000,000 | [`VALIDATOR.md:127`](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L127) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L127) |
| Headroom over measured cost | 22% | derived |
| Block gas ceiling | 3,000,000,000 | [`params.go:28`](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/bft/types/params.go#L28) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/bft/types/params.go#L28) |
| Storage deposit locked by that transaction | 1,289,700ugnot | test13 RPC `/tx` events |

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- **[stray whitespace]** [`misc/deployments/test13.gno.land/VALIDATOR.md:102-103`](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L102-L103) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L102-L103) — the tip is followed by two blank lines instead of one. No enabled linter covers markdown in this repo: `.github/workflows/` has no markdownlint job and the [misc CI workflow](https://github.com/gnolang/gno/blob/d415ef332/.github/workflows/ci-dir-misc.yml#L18-L34) · [↗](../../../../../.worktrees/gno-review-5941/.github/workflows/ci-dir-misc.yml#L18-L34) only builds Go programs under `misc/`. Renders identically either way. Not posted, no change needed.

## Missing Tests

None. Documentation-only change with no test surface.

## Suggestions

None.

## Verified

- The documented flag pair works on the live chain, not just in principle: the transaction the PR body cites (`NRk+9eIVNf+fEb8THBzHGSWWmYSQfYI9LcVxOGzBgaw=`, height 741195) comes back from the test13 RPC with `GasWanted: 100000000`, `GasUsed: 81783643`, `Error: null`, and a decoded fee of `1000000ugnot`. That rules out the concern that doubling gas-wanted while holding the fee flat halves the effective gas price below a node's minimum: this exact combination was accepted and included.
- 100M is far inside the block ceiling. [`MaxBlockMaxGas`](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/bft/types/params.go#L28) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/bft/types/params.go#L28) is 3B and is what [`DefaultBlockParams`](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/bft/types/params.go#L46-L52) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/bft/types/params.go#L46-L52) installs as `Block.MaxGas`, so a 100M budget cannot make the transaction unincludable.
- The tip's two factual claims match the flag as registered. The default is [`zapcore.DebugLevel.String()`](https://github.com/gnolang/gno/blob/d415ef332/gno.land/cmd/gnoland/start.go#L147-L152) · [↗](../../../../../.worktrees/gno-review-5941/gno.land/cmd/gnoland/start.go#L147-L152), and the four options the tip lists are exactly the flag's own help text. There is no competing `log_level` key in the tm2 config, so the CLI flag is the only knob and the tip cannot conflict with the `config.toml` table earlier in the guide.
- No other document carries the stale value: `--gas-wanted 50000000` appears nowhere else in the repository, and the only other `--gas-wanted` uses under `misc/deployments/test13.gno.land/` are in [`gen-genesis.sh`](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/gen-genesis.sh#L713-L714) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/gen-genesis.sh#L713-L714), which reads the value per transaction from `meta.json` rather than hardcoding it.

## Existing threads

- notJoon, APPROVED, "the change seems good". No overlap with anything here.

## Open questions

- The 100M figure is calibrated against one sample whose description is 118 bytes, while the realm accepts up to [`DescriptionMaxLength = 2048`](https://github.com/gnolang/gno/blob/d415ef332/examples/gno.land/r/gnops/valopers/valopers.gno#L26) · [↗](../../../../../.worktrees/gno-review-5941/examples/gno.land/r/gnops/valopers/valopers.gno#L26). Registration gas scales with the bytes written, so an operator with a near-maximal description eats into the 22% headroom. Not posted: quantifying it needs a measurement on test13 state that a local node cannot reproduce, and an unmeasured warning would be speculation.
- [`VALIDATOR.md:114`](https://github.com/gnolang/gno/blob/d415ef332/misc/deployments/test13.gno.land/VALIDATOR.md?plain=1#L114) · [↗](../../../../../.worktrees/gno-review-5941/misc/deployments/test13.gno.land/VALIDATOR.md#L114) says registration "costs a gas fee", but the cited transaction also locked 1,289,700ugnot of storage deposit via [`lockStorageDeposit`](https://github.com/gnolang/gno/blob/d415ef332/gno.land/pkg/sdk/vm/keeper.go#L1568) · [↗](../../../../../.worktrees/gno-review-5941/gno.land/pkg/sdk/vm/keeper.go#L1568), charged to the caller on top of the 1 GNOT fee. Not posted: the faucet's [default max send amount is 10 GNOT](https://github.com/gnolang/gno/blob/d415ef332/contribs/gnofaucet/serve.go#L122-L127) · [↗](../../../../../.worktrees/gno-review-5941/contribs/gnofaucet/serve.go#L122-L127), which covers both with room to spare, and the sentence predates this diff.
- `gnokey maketx --simulate only` reports estimated gas usage, which would keep the guide from going stale again. It does not work as a discovery tool here: the ante handler [seeds the meter from `tx.Fee.GasWanted`](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/sdk/auth/ante.go#L71) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/sdk/auth/ante.go#L71) and [`SetGasMeter` only goes infinite at height 0 or under replay](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/sdk/auth/ante.go#L499-L512) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/sdk/auth/ante.go#L499-L512), so a simulated call is metered against the same budget, and [the estimate is only computed when simulation succeeds](https://github.com/gnolang/gno/blob/d415ef332/tm2/pkg/crypto/keys/client/broadcast.go#L127-L136) · [↗](../../../../../.worktrees/gno-review-5941/tm2/pkg/crypto/keys/client/broadcast.go#L127-L136), so an under-budgeted call reports out-of-gas rather than the number needed. Not posted: the suggestion does not actually solve the staleness it would claim to.
- `valopers.Register` panics when [`sysparams.GetValoperRegisterFee()`](https://github.com/gnolang/gno/blob/d415ef332/examples/gno.land/r/sys/params/valoper.gno#L30-L36) · [↗](../../../../../.worktrees/gno-review-5941/examples/gno.land/r/sys/params/valoper.gno#L30-L36) is non-zero and no coins are sent, and the documented command has no `--send`. The default is 0 and governance has not raised it, so the guide is correct today. Not posted: out of scope for a gas-value fix, and it becomes a real doc bug only if governance sets the fee.
