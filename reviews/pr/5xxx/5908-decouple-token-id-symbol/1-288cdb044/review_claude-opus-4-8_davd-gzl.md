# PR [#5908](https://github.com/gnolang/gno/pull/5908): fix(grc20)!: decouple token id from symbol

URL: https://github.com/gnolang/gno/pull/5908
Author: notJoon | Base: master | Files: 19 | +94 -71
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 288cdb044 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5908 288cdb044`

**TL;DR:** `grc20.NewToken` gains an explicit `id` parameter so a token's event identifier (`Token.ID()`) no longer derives from its `symbol`. Two tokens minted with the same display symbol in one realm now emit distinguishable `Transfer`/`Approval`/`Mint`/`Burn` events.

**Verdict: REQUEST CHANGES** — the change deterministically breaks two integration goldens ([`storage_deposit_price_change`](https://github.com/gnolang/gno/blob/288cdb044/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) · [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72), [`govdao_proposal_change_law`](https://github.com/gnolang/gno/blob/288cdb044/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L16) · [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L16)) because grc20 sits in the gov/dao/v3 genesis graph and grew; the goldens must be updated in this PR before CI can go green.

## Summary
The PR adds an `id` field to `grc20.Token` and a required `id` parameter to `NewToken`, changing [`Token.ID()`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/p/demo/tokens/grc20/token.gno#L111-L113) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/p/demo/tokens/grc20/token.gno#L111-L113) from `origRealm + "." + symbol` to `origRealm + "." + id`. This lets a realm mint several tokens sharing a display `symbol` while keeping distinct stable ids, so a backend reading events alone can tell instances apart. All 18 in-repo callers and both doc examples are updated, and every already-deployed realm (wugnot, foo20, test20) passes its old symbol as the new id, so their on-chain event identifiers are unchanged. The blocking problem is a side effect: grc20 is imported by [`gov/dao/v3/treasury`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6), which is loaded into the genesis graph of two gas/storage-exact integration tests, so the added source shifts their golden values.

## Examples
Realm `gno.land/r/x` minting two same-symbol tokens:

| Call | `Token.ID()` |
|------|-------------|
| new: `NewToken(0, cur, "foo-v1", "Foo", "FOO", 4)` | `gno.land/r/x.foo-v1` |
| new: `NewToken(0, cur, "foo-v2", "Foo", "FOO", 4)` | `gno.land/r/x.foo-v2` |
| old: `NewToken(0, cur, "Foo", "FOO", 4)` ×2 | `gno.land/r/x.FOO` (both — collided) |

## Glossary
- storage deposit: per-realm refundable charge for on-chain storage, locked on a positive byte delta; `processStorageDeposit` in `gno.land/pkg/sdk/vm`.
- pure package: importable, stateless package under `p/`; grc20 is one.

## Critical (must fix)
- **[PR breaks two integration goldens, CI red because of this change]** `gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar:16`, `gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar:72` — grc20 grew inside the gov/dao/v3 genesis graph, shifting exact gas and storage-deposit golden values.
  <details><summary>details</summary>

  Both tests `loadpkg gno.land/r/gov/dao/v3/impl`, which imports [`v3/treasury`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L14) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L14), which imports [`grc20`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/r/gov/dao/v3/treasury/treasury.gno#L6). Adding the `id` field, the extra validation, and the new error/comment lines makes the deployed grc20 source larger, so genesis storage accounting and preprocessing gas both rise. In [`govdao_proposal_change_law.txtar:16`](https://github.com/gnolang/gno/blob/288cdb044/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L16) · [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/govdao_proposal_change_law.txtar#L16) the `maketx run` that creates the proposal now uses 41,006,774 gas against the test's `-gas-wanted 41_000_000`, an out-of-gas failure. In [`storage_deposit_price_change.txtar:72`](https://github.com/gnolang/gno/blob/288cdb044/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) · [↗](../../../../../.worktrees/gno-review-5908/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) the final-balance golden `"coins": "9999854` no longer matches: the user ends with 9,999,853,978,400ugnot, 21,600 lower. Verified on 288cdb044: both tests pass on the parent d1489dc5c and fail on this head, so the drift is from this change, not pre-existing flake. See [repro](comment_claude-opus-4-8.md). Fix: bump `-gas-wanted` in the govdao txtar above 41,006,774 and update the storage-deposit balance golden to the new value.
  </details>

## Warnings (should fix)
None.

## Nits
- `examples/gno.land/p/demo/tokens/grc20/types.gno:84` — new field comment `// ID slug of the token (e.g., "dummy").` uses a lowercase example while the adjacent symbol comment uses `"DUMMY"`; both are the same string in practice.

## Missing Tests
None. [`TestNewTokenAllowsDuplicateSymbolInSameRealm`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L31-L43) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L31-L43) covers the core claim that id, not symbol, drives `ID()`, and the validation table exercises `ErrInvalidID` across empty/too-long/charset cases.

## Suggestions
- `examples/gno.land/p/demo/tokens/grc20/token.gno:27` — `NewToken(0, rlm, id, name, symbol, decimals)` places three adjacent string parameters (`id`, `name`, `symbol`) in a row; several updated call sites already pass `id == symbol`, and transposing `id` and `symbol` produces a valid-but-wrong token with no compile error. A doc-comment ordering reminder or grouping the metadata into a struct would cut the footgun. No change strictly required.
  <details><summary>details</summary>

  Both `id` and `symbol` accept the same charset and length cap via [`validSymbol`](https://github.com/gnolang/gno/blob/288cdb044/examples/gno.land/p/demo/tokens/grc20/token.gno#L78-L88) · [↗](../../../../../.worktrees/gno-review-5908/examples/gno.land/p/demo/tokens/grc20/token.gno#L78-L88), so a swap between them passes validation and only surfaces as a wrong event identifier downstream.
  </details>

## Open questions
- Non-breaking alternative: keep `NewToken(0, rlm, name, symbol, decimals)` with `id` defaulting to `symbol` and add `NewTokenWithID` for the id-aware case. That preserves every downstream grc20 caller, in-repo and external. Raised as a design question in the comment because the breaking signature is deliberate (`!` title, defended in the description) and serves the linked gnoswap request, so it is the author's call, not a blocker.
