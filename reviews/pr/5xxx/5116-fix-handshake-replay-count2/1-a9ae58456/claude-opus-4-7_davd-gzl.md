# PR #5116: test: fix TestHandshakeReplayXXX tests w/ count >= 2

URL: https://github.com/gnolang/gno/pull/5116
Author: aeddi | Base: master | Files: 5 (consensus pkg, ignoring drift) | +12 -13
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `a9ae58456` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5116 a9ae58456`

Verdict: APPROVE — the `_s` branch was mutating the package-global `config` via a shared pointer; the fix scopes `config` to the function. Author should still rebase (PR is CONFLICTING after months of drift) and address [@thehowl](https://github.com/gnolang/gno/pull/5116#discussion_r1962488800)'s naming nit on the shadow before merge.

## Summary

`TestHandshakeReplayXXX` passed with `-count=1` but failed with `-count=2` because `testHandshakeReplay`'s single-node branch was reusing the package-global `config` (a `*cfg.Config`) via its parameter, then calling `config.Consensus.SetWalFile(...)` on it. `ResetConfig` was invoked locally as `testConfig` but never wired in. On the second iteration the global still pointed at the first run's `RootDir`/`DBDir`, so the kvstore on disk had `App block height 11 > core 6` and the abci handshake aborted with "App block height (11) is higher than core (6)". Fix: declare `config` as a function-local in `testHandshakeReplay` and assign `config = testConfig` in the single-node branch; drop the now-unused parameter.

## Glossary

- `config` — package-global `*cfg.Config` in `tm2/pkg/bft/consensus`, set once by `TestMain` via `ResetConfig("consensus_reactor_test")`.
- `testConfig` — fresh per-run `*cfg.Config` produced by `ResetConfig(unique-name)`; was previously dead-assigned in the `_s` branch.
- `_s` / `_m` — single-node / multi-node sub-paths of `testHandshakeReplay`, picked via `sim == nil`.
- `WAL` — write-ahead log used to rebuild the chain in tests.

## Fix

Before: `testHandshakeReplay(t, config, ...)` callers passed the package global. In the single-node branch, `testConfig, gf := ResetConfig(...)` created a fresh config but the function kept using the parameter (still the global), so `config.Consensus.SetWalFile(walFile)` mutated the global's `WalFile` and `config.DBDir()` returned the global's directory — both leaked across `-count=N` iterations. After: parameter dropped; `config` declared inside `testHandshakeReplay` ([`replay_test.go:661`](https://github.com/gnolang/gno/blob/a9ae58456/tm2/pkg/bft/consensus/replay_test.go#L661) · [↗](../../../../../.worktrees/gno-review-5116/tm2/pkg/bft/consensus/replay_test.go#L661)); single-node path assigns `config = testConfig` ([`replay_test.go:680`](https://github.com/gnolang/gno/blob/a9ae58456/tm2/pkg/bft/consensus/replay_test.go#L680) · [↗](../../../../../.worktrees/gno-review-5116/tm2/pkg/bft/consensus/replay_test.go#L680)) so `SetWalFile` and `DBDir()` only touch the fresh `ResetConfig` instance. The multi-node path was already correct (it does `config = sim.Config`).

The PR also drops the `Flappy` prefix from `TestHandshakeReplayNone` — that test was only flappy because of this same bug, so removing the `testutils.FilterStability(t, testutils.Flappy)` gate is appropriate.

## Critical (must fix)

None.

## Warnings (should fix)

- **[merge conflict, needs rebase]** PR is `CONFLICTING` — TL;DR last sync with master was Apr 9; needs rebase before merge.
  <details><summary>details</summary>

  The PR was approved by [@jefft0](https://github.com/gnolang/gno/pull/5116#pullrequestreview-2588849756), [@sw360cab](https://github.com/gnolang/gno/pull/5116#pullrequestreview-2590487263), [@davd-gzl](https://github.com/gnolang/gno/pull/5116#pullrequestreview-2592887814) and [@thehowl](https://github.com/gnolang/gno/pull/5116#pullrequestreview-2628019541) in early-to-mid Feb 2026 and has been sitting open since. The merge in `66d7b98d3` dragged in 468 unrelated files (whitepaper, boards2 v1, gnoweb forms, etc.); reviewing the PR-relevant diff requires `git diff 66d7b98d3 HEAD`. Rebase to clean the history and resolve conflicts. Fix: `git fetch origin master && git rebase origin/master` (or merge + drop the stale merge commit), then force-push.
  </details>

- **[unaddressed review feedback]** [@thehowl](https://github.com/gnolang/gno/pull/5116#discussion_r1962488800) [`tm2/pkg/bft/consensus/replay_test.go:661`](https://github.com/gnolang/gno/blob/a9ae58456/tm2/pkg/bft/consensus/replay_test.go#L661) · [↗](../../../../../.worktrees/gno-review-5116/tm2/pkg/bft/consensus/replay_test.go#L661) — function-local `config` shadows the package global; rename one of them.
  <details><summary>details</summary>

  Inside `testHandshakeReplay` there is now both a function-local `config` (intentional, the fix) and an in-scope package-global `config` declared in `common_test.go:46`. Anyone reading the body now has to mentally track that `config.ChainID()` (used in unrelated funcs in the same file at lines 381, 405, 445, 502) is the global, while `config.DBDir()` at line 708 inside `testHandshakeReplay` is the local. The author themselves flagged this in the PR body: "It's complicated to understand the flow ... with the `config` global var, the `config` local var (parameter), the `sim.Config`, and the role of `testConfig`". Fix: either rename the global to `globalConfig` (touches multiple test files, but clean), or rename the function-local to `cfgLocal`/`tcCfg` and add a one-line comment explaining why it can't be the global. [@thehowl](https://github.com/gnolang/gno/pull/5116#discussion_r1962493452) also suggested an inline comment on the `config = testConfig` assignment to flag the deliberate shadowing — cheap and helpful.
  </details>

## Nits

- [`tm2/pkg/bft/consensus/replay_test.go:677`](https://github.com/gnolang/gno/blob/a9ae58456/tm2/pkg/bft/consensus/replay_test.go#L677) · [↗](../../../../../.worktrees/gno-review-5116/tm2/pkg/bft/consensus/replay_test.go#L677) — `testConfig` is now used only for its `RootDir` (for `os.RemoveAll`) and to source `config`. Could inline as `config, gf := ResetConfig(...)`, then `defer os.RemoveAll(config.RootDir)` — one fewer name in scope. Minor.

## Missing Tests

None — the fix is itself a test-only change targeting an existing test. Manual verification: `go test -v -count=2 -run 'TestHandshakeReplay' ./tm2/pkg/bft/consensus/` passes locally; previously failed on the second iteration with `App block height (11) is higher than core (6)`.

## Suggestions

- [`tm2/pkg/bft/consensus/replay_test.go:573`](https://github.com/gnolang/gno/blob/a9ae58456/tm2/pkg/bft/consensus/replay_test.go#L573) · [↗](../../../../../.worktrees/gno-review-5116/tm2/pkg/bft/consensus/replay_test.go#L573) — `TestHandshakeReplayNone` was de-flappied here, but the flappy filter elsewhere in `tm2/` may still hide similar global-mutation bugs. Consider auditing other tests in `tm2/pkg/bft/consensus/*_test.go` for `config.<X>.Set...` calls that mutate the package global. Out of scope for this PR but worth a follow-up issue.

## Questions for Author

- None — root cause is well understood and the fix is minimal.

## Repro

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5116 -R gnolang/gno
go test -v -count=2 -run 'TestHandshakeReplay' ./tm2/pkg/bft/consensus/
# Expected on PR head: all four PASS.
# To see the pre-fix failure, check out the parent of the fix commit:
git stash && git checkout dd9c926b9^ -- tm2/pkg/bft/consensus/replay_test.go
go test -v -count=2 -run 'TestHandshakeReplayOne$' ./tm2/pkg/bft/consensus/
# Expected: second iteration FAILs with "App block height (11) is higher than core (6)".
git checkout HEAD -- tm2/pkg/bft/consensus/replay_test.go
```
