# PR #5016: docs: add new `r/docs/...` examples

URL: https://github.com/gnolang/gno/pull/5016
Author: davd-gzl | Base: master | Files: 28 | +1806 -12
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `877a57bd8` (stale — +59 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5016 877a57bd8`

Round 2 of 2. Prior review: [`1-a9e2631f2/claude-opus-4-7_davd-gzl.md`](../1-a9e2631f2/claude-opus-4-7_davd-gzl.md) (verdict REQUEST CHANGES).

**Verdict: APPROVE** — every round-1 blocker is resolved: all 11 new realms migrated to the post-#5669 capability API (`cur realm` / `cur.Previous()` / `cross(cur)` / `unsafe.*`), the merge conflict is gone (master's #5726 moved the conflicting `ownable` realm to `examples/quarantined/`), all CI lint+test jobs pass, and the realms compile + lint clean locally. One doc-prose typo remains (`cross` instead of `cross(cur)` in `userprofile`); not blocking. Self-review by the PR author; disclosed per AGENTS.md.

## Summary

The PR adds 11 documentation realms under `examples/gno.land/r/docs/` (composition + counter/logger sub-realms, crossing, factory, json, mistakes, registry, soliditypatterns/banker, soliditypatterns/reentrancy, tdd, userprofile). Round 1 blocked on a single root cause: the branch was 5 weeks stale and every realm used the pre-#5669 surface (`runtime.PreviousRealm()`, `banker.NewBanker(bt)` single-arg, `banker.OriginSend()`), so nothing compiled against master. Since then the author merged master twice and rewrote every realm against the current API. The net PR diff (vs merge-base `2c7f1abe3`) is now purely additive: 25 files, +1774, 0 deletions.

## What changed since `a9e2631f2`

Relevant doc commits on top of two master merges:
- `8eb8f3d26` — idiomatic `cur.Previous()` caller-auth + clearer crossing doc
- `9940005a8` — fix crossing-call form in tdd test, address review
- `38ac75e67` / `e61b07a3b` — `import "testing"` in test example; broad review fixups
- `09d9935ef` — package summaries, simplify json realm
- `42d1163e0` — fix doc-example bugs across r/docs realms
- `877a57bd8` — clarify `lastCreatedJSON` comment

## Round-1 findings — disposition

| # | Round-1 finding | Status |
|---|---|---|
| C1 | Every realm uses removed `runtime.PreviousRealm()`/`CurrentRealm()` — won't compile | Resolved — all realms use `cur.Previous()`, non-crossing helpers use `unsafe.CurrentRealm()` |
| C2 | banker uses removed `NewBanker(bt)` + `banker.OriginSend()` | Resolved — `NewBanker(bt, cur)`, `unsafe.OriginSend()`, `import "chain/runtime/unsafe"` |
| C3 | merge conflict on `soliditypatterns/ownable/render.gno` | Resolved — `mergeable=MERGEABLE`; #5726 relocated `ownable` to `examples/quarantined/`, conflict no longer exists |
| C4 | composition auth broken: `assertAuthorized()` reads `PreviousRealm()` in a non-crossing helper | Resolved — `assertAuthorized(cur realm)` reads `cur.Previous().PkgPath()`; callers use `cross(cur)` |
| W (tdd crossing) | mutating fns should be crossing; tests use wrong call form | Resolved — tests take `(cur realm, t *testing.T)` and call `Add(cross(cur), …)`; tdd tests pass |
| W (counter/logger crossing) | `Increment`/`Log` etc. should be crossing | Resolved — all take `cur realm` |
| W (userprofile `data.Address()`) | prose uses non-existent method | Resolved — prose uses `data.Addr()` |
| W (factory `g1caller`/old API) | `runtime.PreviousRealm().Address()` deprecated | Resolved — `cur.Previous().Address()` |
| W (reentrancy old API) | snippets teach `NewBanker(bt)` single-arg + `runtime.PreviousRealm()` | Resolved — `NewBanker(bt, cur)`, `cur.Previous().Address()` |
| W (json.Marshal panic) | defensive panic never triggers | Resolved — json realm simplified; `encode()` helper, panic is honest for the `Marshal` path |
| W (mistakes `IsValid`/"zero address") | Solidity vocabulary | Resolved — rephrased to "empty or malformed address" |
| Nit (factory `Itoa(int(uint64))`) | truncates | Resolved — `strconv.FormatUint(f.amount, 10)` |
| Nit (banker `init()` no-op) | drop `init` | Resolved — `var deposits avl.Tree`, no init |
| Nit (json unused `User` struct) | drop | Resolved — struct removed; realm builds nodes inline |
| Nit (reentrancy broken "See Also" link) | wrong path | Resolved — links to `/r/docs/soliditypatterns/banker` |
| Nit (`var count int64 = 0`) | drop `= 0` | Resolved — `var count int64` |
| Suggestion (tdd `import "testing"`) | show import | Resolved — first test snippet shows it |
| Suggestion (json `ParseJSON` → pretty name) | rename | Superseded — function removed entirely in the json simplification |

All other reviewer comments from [@mvallenet](https://github.com/gnolang/gno/pull/5016) and [@jeronimoalbi](https://github.com/gnolang/gno/pull/5016) carry author replies with real commit links (`e61b07a3b`, `9940005a8`, `42d1163e0`, `09d9935ef`). The round-1 note that "Fix links point to non-existent commit `a7b9043df`" no longer applies — those were the stale mvallenet-era replies; the actual fixes landed in the commits above.

## Critical (must fix)

None.

## Warnings (should fix)

- **[doc prose teaches wrong cross syntax]** [`userprofile/userprofile.gno:68-69`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/userprofile/userprofile.gno#L68-L69) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/userprofile/userprofile.gno#L68-L69) — the prose snippet calls `profile.SetStringField(cross, profile.DisplayName, "Alice")`, using bare `cross` instead of `cross(cur)`.
  <details><summary>details</summary>

  The PR's own crossing doc teaches the correct form: `other.ModifyState(cross(cur), "hello")` ([`crossing/crossing.gno:74`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/crossing/crossing.gno#L74) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/crossing/crossing.gno#L74)). `SetStringField` on master is `func SetStringField(cur realm, field, value string) bool` ([`r/demo/profile/profile.gno:72`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/demo/profile/profile.gno#L72) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/demo/profile/profile.gno#L72)), so a reader copying the snippet into a real realm and writing `cross` gets a compile error. This is in prose only — the `userprofile` realm code is render-only and never calls `SetStringField`, so nothing in the PR breaks. Fix: `profile.SetStringField(cross(cur), profile.DisplayName, "Alice")` on both lines 68 and 69.
  </details>

## Nits

- [`registry/registry.gno:142`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/registry/registry.gno#L142) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/registry/registry.gno#L142) — `address(parts[0])` casts a raw render-path segment to `address` without `.IsValid()`. Carried from round 1. Harmless (`Lookup` returns `nil` → "Service Not Found"), but a `.IsValid()` guard would make the error page the right one rather than an implicit fallthrough.
- [`registry/registry.gno:113-121`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/registry/registry.gno#L113-L121) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/registry/registry.gno#L113-L121) — [@mvallenet](https://github.com/gnolang/gno/pull/5016#discussion_r2661428401) noted the composite `caller:name` key already enforces ownership, so `Update`/`Deactivate`/`Delete` need no extra owner check (and the current code correctly has none). A one-line comment at the key construction explaining "the `caller:name` key is the auth gate" would make the model legible to a learner. Teaching-clarity only.
- [`userprofile/userprofile.gno:1-2`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/userprofile/userprofile.gno#L1-L2) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/userprofile/userprofile.gno#L1-L2) — the author's reply to the json `User`-struct comment ("`CreateUserJSON` now builds the node from a `User` value") is now stale: the later `simplify json realm` commit removed the struct, and the final code builds the map inline ([`json/json.gno:14-18`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/json/json.gno#L14-L18) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/json/json.gno#L14-L18)). End state is correct (no unused struct); only the comment thread is out of date. No code action needed.

## Missing Tests

- **[only `tdd` ships tests]** all new realms — `composition`, `crossing`, `factory`, `json`, `mistakes`, `registry`, `banker`, `reentrancy`, `userprofile` have no `_test.gno`.
  <details><summary>details</summary>

  Carried from round 1, downgraded. These are doc realms and they now compile + lint clean against master (verified: `gno test .` and `gno lint .` in each dir, all `ok`/no findings), so the "API drift would have been caught by a test" rationale no longer bites — CI already gates compilation. The remaining value is regression coverage for behavior, not compilation. The single highest-value addition is a `composition_test.gno` exercising the cross-realm bypass-rejection path, since "direct calls to `counter.Increment()` will fail with an authorization error" ([`composition/composition.gno:85`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/composition/composition.gno#L85) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/composition/composition.gno#L85)) is the load-bearing security claim of the example and nothing currently verifies it. Note jeronimoalbi's json-test suggestion was acted on but the test was removed alongside the json simplification, so json is now uncovered again.
  </details>

## Suggestions

- [`composition/composition.gno:74-77`](https://github.com/gnolang/gno/blob/877a57bd8/examples/gno.land/r/docs/composition/composition.gno#L74-L77) · [↗](../../../../../.worktrees/gno-review-5016/examples/gno.land/r/docs/composition/composition.gno#L74-L77) — the "Alternative: pure packages" callout is the correct recommendation but still isn't demonstrated. A parallel `/p/` variant would make the "bypass structurally impossible" claim concrete instead of asserted. Carried from round 1; optional.

## Questions for Author

- `mvallenet`'s approval predates the full API migration. Worth re-requesting review now that the rewrite has landed and CI is green.
