# Contribution draft: PR [5835](https://github.com/gnolang/gno/pull/5835) — `realm_only_gate` rule

Not a review round. Worktree `.worktrees/gno-review-5835` is kept at pristine head `96cce07a2` so the review anchors in comment.md stay correct; the rule edits are parked, not applied in place. Postable comment: [`comment_realm-only-gate.md`](comment_realm-only-gate.md).

## State

- Rule + test edits parked in [`realm-only-gate.patch`](realm-only-gate.patch): `realm_only_gate` case + `realmOnlyGateHits` (moul's `loadGnoSource` / `src.hit` idiom) in `run.go`, `TestRealmOnlyGateRule` in `run_test.go`. `run.go`/`run_test.go` in the worktree are reverted to pristine.
- `expected/realm-only-gate.yaml`, `fixtures/realm-only-gate/{vulnerable,fixed}` (`reputation.gno` + `gnomod.toml`) remain untracked in the worktree.
- With the patch applied: harness suite green including `TestAgentPatternContract/realm-only-gate` and `TestAgentPatternContractWithGNO/realm-only-gate` (both fixtures compile under `gno`); run against 5976's realm reports `reputation.gno:17`.

## Gap it closes

Ten rules, none fire on [5976](https://github.com/gnolang/gno/pull/5976)'s access gate. `reputation.gno:17` writes `if caller.IsUserCall() { panic }` to mean realms-only. `IsUserCall()` is `pkgPath == ""`, false inside the `maketx run` realm, so a user script passes. `payment_user_call` is the only rule reading `IsUserCall()`, and it fires only alongside an `OriginSend()`; 5976 has none.

## Next

Offer to moul via [`comment_realm-only-gate.md`](comment_realm-only-gate.md), or open a PR against `codex/audit-guide-examples`. Both gated on explicit approval (fork push + `gh pr create`).
