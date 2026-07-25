# Contribution draft: PR [5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule

Not a review round. Rule + fixtures live in `.worktrees/gno-review-5835`, checked out on head `96cce07a2` (moul's `loadGnoSource` / `src.hit` line-map idiom). Postable comment: [`comment_realm-only-gate.md`](comment_realm-only-gate.md).

## State (all uncommitted in the worktree)

- `run.go`: `realm_only_gate` case + `realmOnlyGateHits` in the new idiom.
- `run_test.go`: `TestRealmOnlyGateRule`.
- `expected/realm-only-gate.yaml`, `fixtures/realm-only-gate/{vulnerable,fixed}` (`reputation.gno` + `gnomod.toml`).
- Harness suite green, including `TestAgentPatternContract/realm-only-gate` and `TestAgentPatternContractWithGNO/realm-only-gate` (both fixtures compile under `gno`).
- Run against 5976's realm file: reports `reputation.gno:17`.

## Gap it closes

Ten rules, none fire on [5976](https://github.com/gnolang/gno/pull/5976)'s access gate. `reputation.gno:17` writes `if caller.IsUserCall() { panic }` to mean realms-only. `IsUserCall()` is `pkgPath == ""`, false inside the `maketx run` realm, so a user script passes. `payment_user_call` is the only rule reading `IsUserCall()`, and it fires only alongside an `OriginSend()`; 5976 has none.

## Next

Offer to moul via [`comment_realm-only-gate.md`](comment_realm-only-gate.md), or open a PR against `codex/audit-guide-examples`. Both gated on explicit approval (fork push + `gh pr create`).
