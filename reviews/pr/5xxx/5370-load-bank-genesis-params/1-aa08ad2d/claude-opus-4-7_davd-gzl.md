# PR #5370: feat(gno): load bank param from genesis_param.toml

URL: https://github.com/gnolang/gno/pull/5370
Author: mvallenet | Base: master | Files: 4 | +129 -2
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: APPROVE** — small, well-scoped TODO-completion + denom-typo fix; only gap is a missing test for the new `std.ValidateDenom` error path on the loader.

## Summary

`["bank"]` in [`gno.land/genesis/genesis_params.toml`](../../../../../.worktrees/gno-review-5370/gno.land/genesis/genesis_params.toml) was silently dropped because [`LoadGenesisParamsFile`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L79) only handled `vm` / `vm:<realm>`. The TOML also used `"gnot"` instead of `ugnot.Denom` (`"ugnot"`), so even a future wire-up would have been a no-op against [`keeper.go:1303`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/sdk/vm/keeper.go#L1303) (`slices.Contains(..., ugnot.Denom)`). The PR adds a `bank.restricted_denoms` parser with `std.ValidateDenom` checking (matching the suggestion left on commit `13743df6`), fixes the denom string, leaves explicit comments in the testscript wiring documenting why `tsGenesis.Bank` is intentionally not promoted into the test node genesis, and adds 6 subtests + 1 real-file test.

## Glossary

- `LoadGenesisParamsFile` — TOML-to-`GnoGenesisState` decoder used by integration test setup (`node_testing.go:LoadDefaultGenesisParamFile`); not currently called from any production gnoland entrypoint.
- `RestrictedDenoms` — bank-keeper list consulted by `canSendCoins` ([`bank/keeper.go:122`](../../../../../.worktrees/gno-review-5370/tm2/pkg/sdk/bank/keeper.go#L122)) and `processStorageDeposit` ([`vm/keeper.go:1303`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/sdk/vm/keeper.go#L1303)).
- `splitTypedName` — helper that strips a `.<type>` suffix from a TOML key (`foo.strings` → `foo`).

## Fix

Before: `LoadGenesisParamsFile` had a `XXX Write onto ggs for other keeper params` placeholder and the `["bank"]` section was unread. After: a `bank` branch parses `restricted_denoms` as `[]any`, asserts each element is a `string`, runs `std.ValidateDenom` per entry, and writes onto `ggs.Bank.Params.RestrictedDenoms` ([`genesis.go:96-121`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L96-L121)). Unknown sub-keys error out via `default:`. The TOML file is updated to `"ugnot"` to match `ugnot.Denom` ([`genesis_params.toml:5`](../../../../../.worktrees/gno-review-5370/gno.land/genesis/genesis_params.toml#L5)). In [`testscript_gnoland.go:304-306`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/integration/testscript_gnoland.go#L304-L306), two commented lines (`genesis.VM.Params = ...`, `genesis.Bank = ...`) make the non-propagation explicit. CI: 101/101 green; local `go test ./gno.land/pkg/gnoland/ -run 'TestLoadGenesisParamsFile|TestGenesis_Verify'` passes.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`genesis.go:96-121`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L96-L121) — the new bank branch uses error returns with `%T` for bad shapes; the surrounding `vm` branch ([`genesis.go:126-138`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L126-L138)) still uses bare `value.(string)` assertions that will panic on a malformed file. Not part of this PR's scope, but the inconsistency is worth a follow-up.
- [`genesis.go:123`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L123) — `TODO: Write onto ggs for auth and tm2 keeper params` replaces the old `XXX`. Fine; flagging only because the file still has no caller in production (only tests via `node_testing.go:168`), so the TODO will likely outlive its usefulness if/when the production path lands.

## Missing Tests

- **[validation path uncovered]** [`genesis_test.go:70-154`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis_test.go#L70-L154) — no case exercises the new `std.ValidateDenom` error branch added in direct response to the prior review comment.
  <details><summary>details</summary>

  The `["bank"] restricted_denoms = ["INVALID!!!"]` case would fail at the loader (line 111-113) before reaching `bank.ValidateGenesis`. Today the only "invalid denom" coverage is `TestGenesis_Verify/invalid_GenesisState_Bank`, which constructs the state in-Go and exercises `ValidateGenState` → `bank.ValidateGenesis` → `Params.Validate`. The TOML loader's validation call is therefore untested. Codecov flags 3 missing + 3 partial lines on `genesis.go`; this is one of them. Fix: add a subtest `name: "invalid restricted denom"`, `toml: '["bank"]\n  restricted_denoms = ["INVALID!!!"]\n'`, `expectErr: "bank parameter restricted_denoms[0]"`.
  </details>
- **[non-array type uncovered]** [`genesis_test.go:70-154`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis_test.go#L70-L154) — no case exercises `restricted_denoms = "ugnot"` (string instead of array) or `restricted_denoms = [1, 2]` (int elements), which hit the two `%T` branches at [`genesis.go:103`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L103) and [`genesis.go:109`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/gnoland/genesis.go#L109). One subtest each closes the remaining coverage gap.

## Suggestions

- [`testscript_gnoland.go:304-306`](../../../../../.worktrees/gno-review-5370/gno.land/pkg/integration/testscript_gnoland.go#L304-L306) — the "Not propagated" comment is helpful but reads like temporary scaffolding. If "production defaults from `genesis_params.toml` would break tests" is the long-term contract, a one-liner naming the concrete break (e.g. "RestrictedDenoms=ugnot would block all `adduser` balance grants") would save a future reader the bisect.

## Questions for Author

- Is there a planned follow-up that actually wires `LoadGenesisParamsFile` into the production `gnoland start` / `gnogenesis` path? Today the only call site is `integration/node_testing.go:168`, so the fix's user-visible impact is bounded to tests until that lands.

## Past reviews

None. First review of this PR. The prior `genesis.go:114` review comment by `davd-gzl` on commit `13743df6` ("add `std.ValidateDenom` per entry") was addressed in commit `ad5f6da9`.
