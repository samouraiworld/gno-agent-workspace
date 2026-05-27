# PR #5350: feat(gnovm): display storage usage after running file tests

URL: https://github.com/gnolang/gno/pull/5350
Author: davd-gzl | Base: master | Files: 12 | +66 -34
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `5b7bdc9` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5350 5b7bdc9`

Verdict: APPROVE — clean, scoped feature; only concerns are an optional-regex gap in the txtar fixtures and a stale rebase against master. Already approved by [@Villaquiranm](https://github.com/Villaquiranm) and [@notJoon](https://github.com/notJoon); reviewer feedback (type alias, AGENT docs) addressed.

## Summary

Filetest output and `--print-runtime-metrics` now report per-realm storage byte deltas alongside gas: `--- PASS: ./x_filetest.gno (elapsed: 0.00s, gas: 64733, storage: gno.land/r/xx:+5b)`. Closes [#5347](https://github.com/gnolang/gno/issues/5347). The diff threads `gno.StorageDiffs` (a new type alias for `map[string]int64`) through `RunFiletest`/`runFiletest` as a 4th return value, lifts per-realm diffs from `m.Store.RealmStorageDiffs()` after each run, and renders them via a new `fmtStorageDiffs` helper that omits zero entries and signs each byte count with `%+d`. Seven txtar fixtures were widened to make the new `, storage: ...` suffix optional in the regex.

## Glossary

- `StorageDiffs` — `map[string]int64` alias keyed by realm path; values are signed byte deltas accumulated during a message.
- `RealmStorageDiffs()` — `Store` method returning the live diff map; mutated by `Realm.FinalizeRealmTransaction` ([`gnovm/pkg/gnolang/realm.go:394-396`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/gnolang/realm.go#L394-L396) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/gnolang/realm.go#L394-L396)) and consumed by `processStorageDeposit` in production ([`gno.land/pkg/sdk/vm/keeper.go:1244-1245`](https://github.com/gnolang/gno/blob/5b7bdc9/gno.land/pkg/sdk/vm/keeper.go#L1244-L1245) · [↗](../../../../../.worktrees/gno-review-5350/gno.land/pkg/sdk/vm/keeper.go#L1244-L1245)).
- `fmtStorageDiffs` / `realmDiffsString` — two formatters; the former for the PASS/FAIL line, the latter for the `// Storage:` directive.

## Fix

Before: filetest output stopped at gas; the `// XXX: store changes` TODO in [`gnovm/pkg/test/test.go:524`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L524) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L524) (pre-patch) acknowledged the gap. After: `runFiletest` collects `m.Store.RealmStorageDiffs()` immediately after `runTest` ([`gnovm/pkg/test/filetest.go:87-88`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/filetest.go#L87-L88) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/filetest.go#L87-L88)) and returns it; the `Test` driver formats and appends the suffix only when at least one realm has a non-zero diff ([`gnovm/pkg/test/test.go:653-671`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L653-L671) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L653-L671)). The unit-test path under `--print-runtime-metrics` calls `fmtStorageDiffs(m.Store.RealmStorageDiffs())` once per test ([`gnovm/pkg/test/test.go:536`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L536) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L536)). The type alias `StorageDiffs = map[string]int64` ([`gnovm/pkg/gnolang/store.go:39-40`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/gnolang/store.go#L39-L40) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/gnolang/store.go#L39-L40)) names the existing shape without changing any production-store type, keeping the keeper and realm sites untouched.

## Critical (must fix)

None.

## Warnings (should fix)

- **[stale rebase, not a code issue]** branch base — PR last merged master on 2026-04-14; current master has moved 230+ commits ahead.
  <details><summary>details</summary>

  Running `go test -timeout 300s -run TestFiles ./gnovm/pkg/gnolang/` on the PR HEAD against the local toolchain produces ~6 unrelated `TypeCheckError` diffs (`invalid operation: cannot slice p` vs `cannot slice p`, `cannot make int; type must be ...` vs `cannot make int: type must be ...`). Same tests pass on `origin/master`. Cause: Go's stdlib `go/types` updated those error strings in a newer release, and the PR branch carries stale `// TypeCheckError:` golden lines. Not introduced by this PR — but a rebase will be needed before merge or CI on a fresh runner may go red. Fix: rebase on current master; the golden lines have already been updated upstream.
  </details>

## Nits

- [`gnovm/cmd/gno/testdata/test/realm_correct.txtar:7`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/cmd/gno/testdata/test/realm_correct.txtar#L7) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/cmd/gno/testdata/test/realm_correct.txtar#L7) — `(, storage: .+)?` makes the storage segment optional even on a realm filetest that *always* writes state. The regex passes whether storage is rendered or silently dropped, so a regression where the suffix disappears for realm tests would slip through. Applies identically to the other six modified txtars. Replace with a required alternative when the test exercises a realm (`gas: \d+, storage: \S+`) to lock in non-empty rendering for the realm cases.
- [`gnovm/cmd/gno/testdata/test/flag_print-runtime-metrics.txtar:6`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/cmd/gno/testdata/test/flag_print-runtime-metrics.txtar#L6) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/cmd/gno/testdata/test/flag_print-runtime-metrics.txtar#L6) — the existing regex anchors only `runtime: cycle=… allocs=…(\d\.\d\d%)` and doesn't mention storage; the new suffix (`, storage: gno.test/p/integ/flag_print_runtime_metrics:+2468b`, observed locally) is functionally untested at the txtar level. Add a `stderr` assertion for the storage segment so the metric path has integration coverage.
- [`gnovm/pkg/test/filetest.go:219-232`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/filetest.go#L219-L232) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/filetest.go#L219-L232) vs [`gnovm/pkg/test/test.go:650-671`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L650-L671) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L650-L671) — two formatters for the same map now coexist with different shapes: `realmDiffsString` emits `path: 5\n` (for `// Storage:` directive matching), `fmtStorageDiffs` emits ` path:+5b` (for the PASS line). Intentional, but worth a 1-line comment on each so future contributors don't unify them and break the golden directive.
- [`gnovm/pkg/test/test.go:653-659`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L653-L659) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L653-L659) — small redundancy: the loop filters keys by `v != 0` then re-reads `diffs[k]` later. Single pass with `for k, v := range diffs` and a slice of `(key, value)` pairs would avoid the double map lookup. Micro-optimisation, ignore unless touching the file.

## Missing Tests

- **[no direct unit test]** [`gnovm/pkg/test/test.go:653`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/test/test.go#L653) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/test/test.go#L653) — `fmtStorageDiffs` has no targeted Go test covering: (a) empty map, (b) all-zero diffs, (c) mixed positive/negative, (d) deterministic sort across multiple realms. Coverage is purely incidental via the txtars. A 15-line table-driven test in `gnovm/pkg/test/` would lock the format contract (the sign, the ordering, the leading `, storage:` prefix) so future formatting tweaks can't silently change the public-facing output.
- Codecov already flagged the patch at 81.6% with the 7 missing lines living entirely in `gnovm/pkg/test/filetest.go` (the `nil` error returns added in the 4-tuple). Low cost to bring up; nothing security-sensitive.

## Suggestions

- [`gnovm/pkg/gnolang/store.go:39-40`](https://github.com/gnolang/gno/blob/5b7bdc9/gnovm/pkg/gnolang/store.go#L39-L40) · [↗](../../../../../.worktrees/gno-review-5350/gnovm/pkg/gnolang/store.go#L39-L40) — consider whether `StorageDiffs` should be a *defined* type rather than an alias (drop the `=`). Alias buys readability but no type safety; a defined type would force the keeper and realm sites to use the name and make method receivers possible (e.g. `(d StorageDiffs) NonZero() iter.Seq2[string, int64]`). Trade-off is a wider diff. Status quo is fine; flagging only because the original review comment was about clarity at the signature level.
- The PR body says `Closes #5347`, which is good. Consider adding a one-line note in the body about the `// Storage:` directive being orthogonal to this change (the directive existed before; this PR adds the `--- PASS` line rendering), to pre-empt a "why two formats?" question on read.

## Questions for Author

- Was the choice of `+5b` (bytes) over `5 bytes` / `5B` deliberate for compactness with multiple realms on one line? If so, worth a one-line comment on `fmtStorageDiffs` since the format becomes part of tooling output that downstream scripts may grep.
- Any plan to surface storage diffs in `gno test` integration mode for `_test.gno` files outside `--print-runtime-metrics`, or is the PASS-line treatment intentionally restricted to filetests?
