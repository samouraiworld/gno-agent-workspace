# PR #5682: fix(gnovm): allow `fallthrough` from non-last default clause

URL: https://github.com/gnolang/gno/pull/5682
Author: davd-gzl | Base: master | Files: 8 | +266 -94
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `b25f19c` (stale — +71 commits since)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5682 b25f19c`

**Verdict: APPROVE** — Minimal-shape fix that aligns Gno expression-switch semantics with Go for `fallthrough` from a non-last `default:`; runtime/type-switch defers default-selection cleanly and the preprocess fallthrough check is now correct by construction. Only nits and a small ADR doc drift.

## Summary

Go allows `fallthrough` from any clause that is not textually last, including an earlier `default:`. Gno rejected it because [`toClauses`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/go2gno.go#L906-L920) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/go2gno.go#L906-L920) reordered `default` to the end of `SwitchStmt.Clauses`, so the preprocess "final clause" check ([`preprocess.go:2770`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/preprocess.go#L2770) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/preprocess.go#L2770)) fired on the wrong clause. The fix preserves textual order in `toClauses` and shifts "default runs only when no case matches" to evaluation time: [`doOpTypeSwitch`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L844-L923) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L844-L923) records `defaultIdx` and only uses it after the matching pass; [`doOpSwitchClause`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L925-L972) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L925-L972) skips default during the per-clause match loop and falls back to a linear scan once cases are exhausted. With textual order preserved, the existing `i == len(swch.Clauses)-1` check at [`preprocess.go:2770`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/preprocess.go#L2770) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/preprocess.go#L2770) is correct by construction.

```
Before:                          After:
toClauses reorders [default→end]  toClauses preserves source order
+---------------+                 +---------------+
| case1         | i=0             | default       | i=0  (textually first)
| case2         | i=1             | case1         | i=1
| default       | i=2 (moved!)    | case2         | i=2  (textually last)
+---------------+                 +---------------+
fallthrough from default:        fallthrough from default:
  i=2 = len-1 → REJECT             i=0 ≠ len-1 → ACCEPT, jump to i+1
```

## Glossary

- `toClauses`: AST builder that maps Go `*ast.CaseClause` slices into `[]SwitchClauseStmt` for `SwitchStmt`.
- `doOpSwitchClause`: per-clause dispatcher for expression switches; re-enters once per clause via the OpSwitchClause stack op.
- `doOpTypeSwitch`: single-pass type-switch dispatcher; iterates all clauses inline.
- `FALLTHROUGH BodyIndex`: index assigned during preprocess; runtime at [`op_exec.go:721-723`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L721-L723) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L721-L723) jumps to `BodyIndex+1` (next textual clause) on fallthrough.

## Fix

[`toClauses`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/go2gno.go#L906-L920) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/go2gno.go#L906-L920) now appends every clause in source order and tracks `sawDefault` only to preserve the duplicate-default panic. Default-selection moves to runtime: [`doOpTypeSwitch`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L857-L895) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L857-L895) records a `defaultIdx`, breaks out of the match loop on the first hit via a labeled break, and assigns `matchedIdx = defaultIdx` only if no case matched. [`doOpSwitchClause`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L932-L971) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L932-L971) advances past any default clauses on entry, and when the clause cursor runs off the end it scans once for the default and dispatches its body. The `fallthrough` preprocess check at [`preprocess.go:2769-2779`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/preprocess.go#L2769-L2779) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/preprocess.go#L2769-L2779) is unchanged — with textual ordering preserved, `i == len(swch.Clauses)-1` now picks out the actually-last clause.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/pkg/gnolang/op_exec.go:963`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L963) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L963) — `caiv.SetInt(0)` is redundant. Initial entry pushes `0` at [`op_exec.go:783`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L783) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L783) and re-entry from `doOpSwitchClauseCase` resets it at [`op_exec.go:1020`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L1020) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L1020). Harmless and arguably defensive, but worth a one-line comment if kept.
- [`gnovm/adr/pr5682_fallthrough_from_default.md:40`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/adr/pr5682_fallthrough_from_default.md#L40) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/adr/pr5682_fallthrough_from_default.md#L40) — ADR says "a single backward scan finds the default" but the code does a forward scan at [`op_exec.go:939-944`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L939-L944) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L939-L944) (`for i := range ss.Clauses { if len == 0 { defaultIdx = i; break } }`). Either flip the loop or fix the doc — forward is fine since there is only one default by construction (`toClauses` panics on duplicates).
- [`gnovm/adr/pr5682_fallthrough_from_default.md:1`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/adr/pr5682_fallthrough_from_default.md#L1) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/adr/pr5682_fallthrough_from_default.md#L1) — Heading is still `PRxxxx`. Replace with `PR5682` (the filename is already correct).
- [`gnovm/pkg/gnolang/op_exec.go:938-944`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L938-L944) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L938-L944) — The default-scan inside `doOpSwitchClause` duplicates the same `len(cs.Cases) == 0` predicate used twice already in this function. A tiny helper like `defaultClauseIndex(ss.Clauses)` would deduplicate three call sites (here, the skip loop at [937](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L937) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L937), and the typeSwitch loop at [862-865](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L862-L865) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L862-L865)) without growing node state. Optional.

## Missing Tests

- [`gnovm/tests/files/`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/tests/files/) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/tests/files/) — Coverage for chained fallthrough through a middle `default:` is missing.
  <details><summary>details</summary>

  The PR covers the three "shapes" that matter for the preprocess/runtime split: default-first + fallthrough (switch42), case → middle default (switch43), default-last still rejected (switch44). What's not exercised is a chain like `case 1: fallthrough; default: fallthrough; case 2:` where fallthrough crosses a middle default *and* continues into a later case. I added this locally as `switch_default_mid.gno` and `switch_chain.gno` and both pass; worth landing one of them to lock the behavior in the test suite.
  </details>

## Suggestions

- [`gnovm/pkg/gnolang/op_exec.go:847`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L847) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L847) — Type-switch CPU metering is unchanged (`OpCPUSlopeTypeSwitchCase * len(ss.Clauses)`) and still bounds the new loop, so no metering change needed. Expression-switch dispatch (`doOpSwitchClause`) was unmetered before and remains unmetered; the new skip-default and end-scan loops add O(N) work per call but the surrounding per-clause dispatch was already O(N) total, so asymptotics are unchanged. Worth a one-line comment near [932](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/op_exec.go#L932) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/op_exec.go#L932) noting the assumption.
- [`gnovm/pkg/gnolang/nodes_string.go:400-408`](https://github.com/gnolang/gno/blob/b25f19c/gnovm/pkg/gnolang/nodes_string.go#L400-L408) · [↗](../../../../../.worktrees/gno-review-5682/gnovm/pkg/gnolang/nodes_string.go#L400-L408) — `SwitchStmt.String()` now renders clauses in source order, which is a positive behavior change (previously printed `default:` last regardless of source). Not a regression, but if any printer-snapshot test exists it may need re-baselining. Quick grep finds none.

## Questions for Author

- Did you consider gating the duplicate-default detection on `clause.Cases == nil` (which is what `static_analysis.go:172` uses) instead of `len(clause.Cases) == 0`? Both work today, but the codebase mixes the two — picking one in `toClauses` and the runtime would help future readers. Not a blocker.
