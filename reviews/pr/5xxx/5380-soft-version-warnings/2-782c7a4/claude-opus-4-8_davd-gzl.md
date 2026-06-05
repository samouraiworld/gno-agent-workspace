# PR #5380: feat(gnovm): add `vm/qlatestversion` query and soft version warnings for gnokey addpkg

URL: https://github.com/gnolang/gno/pull/5380
Author: davd-gzl | Base: master | Files: 10 | +727 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `782c7a4` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5380 782c7a4`
Note: self-authored PR — flagged for transparency; findings stand on their own.
Round 2 of 2. Prior review: `reviews/pr/5xxx/5380-soft-version-warnings/1-2d0babd/`.

**Verdict: APPROVE** — both round-1 critical findings are fixed and covered by tests: the keeper now re-anchors on `base == basePath` so nested siblings stop inflating the version count, and the CLI skips silently on any ABCI response error so a deploy against an older node no longer hard-blocks. All four prior Warnings and the two doc/help Nits are also addressed. One non-blocking gap remains: `checkVersionGap`'s network/response branches still have no end-to-end coverage, but the load-bearing keeper path is now tested directly.

## Summary

Tooling-only soft-warning system for non-sequential `/vN` deployments: `ParseVersionSuffix` helper, `vm/qlatestversion` ABCI query, and `gnokey maketx addpkg --force` that warns on a missing predecessor and blocks when the gap from latest exceeds 5. Round 1 (REQUEST CHANGES) flagged two correctness bugs. Round 2 fixes both, adds a query timeout, surfaces previously-swallowed parse failures, warns on backwards deploys, rejects zero-padded suffixes, and documents the soft-check behavior. Net delta since `2d0babd`: 7 files, +96 / -7, all addressing review feedback.

## Round-1 resolution

| Round-1 finding | Status | Where |
|---|---|---|
| Critical: nested path collision over-counts versions | Fixed | keeper re-anchors on `base != basePath` |
| Critical: old-node ABCI error blocks `vN` deploys | Fixed | CLI returns nil on any `Response.Error` |
| Warning: silent strconv failure on `Latest` | Fixed | warns to stderr, then skips |
| Warning: query swallows context (no timeout) | Fixed | 3s `context.WithTimeout` |
| Warning: backwards deploy emits no warning | Fixed | `gap < 0` warns "going backwards" |
| Warning: zero-padded `v01` ≡ `v1` | Fixed | rejects leading-zero suffix |
| Warning: no end-to-end test for `checkVersionGap` | Partial | keeper path now tested; CLI network branch still indirect |
| Nit: help text "> 5" drifts from constant | Fixed | `Sprintf(...exceeds %d, maxVersionGap)` |
| Nit: docs omit `--force` / warning behavior | Fixed | added paragraph in `interact-with-gnokey.md` |

## Fix

Keeper nested-collision: the prefix scan `basePath + "/v"` still pulls in deeper packages, but the loop now discards any path whose `ParseVersionSuffix` base does not equal the queried base, at [`gno.land/pkg/sdk/vm/keeper.go:1239-1245`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/sdk/vm/keeper.go#L1239-L1245) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1239-L1245). Old-node block: `qres.Response.Error != nil` now returns nil unconditionally rather than routing into `evalVersionGap(..., nil, ...)`, at [`gno.land/pkg/keyscli/addpkg.go:210-217`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/keyscli/addpkg.go#L210-L217) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L210-L217).

## Verification

Round-2 fixes verified against the source and exercised by tests added in this round:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5380 -R gnolang/gno
go test -run 'TestVmHandlerQuery_LatestVersion|TestParseVersionSuffix' ./gno.land/pkg/sdk/vm/ ./gnovm/pkg/gnolang/
go test -run 'TestEvalVersionGap' ./gno.land/pkg/keyscli/
```

```
ok  	github.com/gnolang/gno/gno.land/pkg/sdk/vm	3.432s
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.010s
ok  	github.com/gnolang/gno/gno.land/pkg/keyscli	0.030s
```

The nested-collision repro from round 1 (`avl/v1` + `avl/v1/sub/v3`, query `avl`) is now codified as the `nested_sibling_versions` subtest in `TestVmHandlerQuery_LatestVersion`, asserting `"latest":"v1"`, `"missing":0`, and absence of `first_missing` ([`gno.land/pkg/sdk/vm/handler_test.go:585-617`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/sdk/vm/handler_test.go#L585-L617) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/handler_test.go#L585-L617)). Build of the three affected packages passes; CI `check` jobs are green (the failing "Merge Requirements" gate is the standard pending-approval bot, not a test failure).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gno.land/pkg/keyscli/addpkg.go:254-259`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/keyscli/addpkg.go#L254-L259) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L254-L259) — the `gap < 0` branch dereferences `result.Latest`, which is safe only by arithmetic: a `nil` result yields `latestVersion = -1`, so `gap >= 1` and the branch is unreachable. No guard enforces it. Today's sole production caller feeds `version >= 1`, so nothing to fix, but a future direct caller relying on the `result == nil` signature should know the protection is incidental, not explicit.
- [`gno.land/pkg/sdk/vm/keeper.go:1253-1254`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/sdk/vm/keeper.go#L1253-L1254) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/sdk/vm/keeper.go#L1253-L1254) — keeper still returns `error` for the no-versions case rather than an empty result (round-1 Suggestion not taken). Fine: the CLI's "skip on any response error" fix makes the choice moot for this PR. If a future consumer (gnoweb banner) wants to distinguish "no versions" from "endpoint unknown", revisit then.

## Missing Tests

- **[CLI network branch]** [`gno.land/pkg/keyscli/addpkg.go:184-226`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/keyscli/addpkg.go#L184-L226) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L184-L226) — `checkVersionGap` (timeout, response-error-skip, unmarshal-warn) is still only exercised through its pure helper `evalVersionGap`. Downgraded from round 1: the keeper-side bug it fed is now tested directly, so the residual risk is small. A `.txtar` under `gno.land/pkg/integration/testdata/` driving `gnokey maketx addpkg` against a live node would close it (deploy v1 + v1/sub/v3, attempt v2, assert the warning text; and assert no block against a node without the query).

## Suggestions

- [`gno.land/pkg/keyscli/addpkg.go:174`](https://github.com/gnolang/gno/blob/782c7a4/gno.land/pkg/keyscli/addpkg.go#L174) · [↗](../../../../../.worktrees/gno-review-5380/gno.land/pkg/keyscli/addpkg.go#L174) — round-1 suggestion to make `maxVersionGap` a `--max-version-gap` flag still stands. Optional, out of scope for this round.

## Questions for Author

- None outstanding. The round-1 question on the old-node fallback is answered by the round-2 design: any response error now falls back to silent skip.
