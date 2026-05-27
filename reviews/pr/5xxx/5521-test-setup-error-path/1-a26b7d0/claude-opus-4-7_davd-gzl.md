# PR #5521: fix(gnovm): show failing path on setup error in gno test

URL: https://github.com/gnolang/gno/pull/5521
Author: aronpark1007 | Base: master | Files: 2 | +30 -13
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5521 a26b7d0` (then `gh -R gnolang/gno pr checkout 5521` inside it)

Verdict: APPROVE — small, correct UX fix; only nit is a missing trailing newline in the new txtar file.

## Summary

Closes [#5507](https://github.com/gnolang/gno/issues/5507): when `gno test` is given multiple paths and one lacks `gnomod.toml`, the failing path was not identifiable in the output (the per-path `FAIL <dir>` line was skipped on load errors, leaving only a bare `FAIL` summary). This PR hoists the `prettyDir` computation to the top of the per-package loop so it is available on the error branch, then prints `FAIL    <path> \t[setup failed]` mirroring `go test`'s setup-failure line. Also adds a txtar regression test under [`gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar).

## Fix

Before: the loop printed each `pkg.Errors[i]` line then `continue`d on `len(pkg.Errors) != 0`, never emitting a per-path summary line. After: the loop computes `prettyDir` first (no side-effects beyond `os.Getwd` and a `filepath.Rel`), then on the same error branch prints `FAIL    <prettyDir> \t[setup failed]` before continuing. See [`gnovm/cmd/gno/test.go:244-270`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/cmd/gno/test.go#L244-L270) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/cmd/gno/test.go#L244-L270). The hoist is safe because `pkg.Dir` is unconditionally populated in [`loadSinglePkg`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/pkg/packages/load_matches.go#L40) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/pkg/packages/load_matches.go#L40) regardless of whether errors are subsequently appended.

Round 1 of review already happened on GitHub: [@thehowl](https://github.com/gnolang/gno/pull/5521#pullrequestreview-3438) flagged tab-vs-space formatting and a missing txtar test; both were addressed in the second commit ([a26b7d0e](https://github.com/gnolang/gno/pull/5521/commits/a26b7d0ed51039f0e4abf1a8b9042c1fbe4ab3fc)). The current HEAD's format matches the surrounding `?       %s \t[no test files]` / `FAIL    %s \t%s` / `ok      %s \t%s` lines exactly.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar:16`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar#L16) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/cmd/gno/testdata/test/no_gnomod_multi_path.txtar#L16) — file is missing a trailing newline; sibling txtars (`dir_not_exist.txtar`, `empty_dir.txtar`, `no_path_empty_dir.txtar`) all end with `\n`. One-byte append.

## Missing Tests

None — the new `no_gnomod_multi_path.txtar` covers the bug. Optional follow-up could pin the multi-package case where both paths lack `gnomod.toml` (two consecutive `FAIL ... [setup failed]` lines) but that is the same code path and adds little signal.

## Suggestions

- [`gnovm/cmd/gno/test.go:267-270`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/cmd/gno/test.go#L267-L270) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/cmd/gno/test.go#L267-L270) — the new `FAIL <prettyDir> [setup failed]` line now fires for any package with `len(pkg.Errors) != 0`, including dependency-only packages (where `len(pkg.Match) == 0`). Previously deps with load errors printed only their `pkg.Errors[i].Error()` text and were silently skipped without a summary line. The new behavior arguably mirrors `go test` (which also emits `FAIL <dep> [setup failed]` for dep failures), so this is likely intentional and useful — but worth a sentence in the commit body / PR description acknowledging the broader scope of the change.
  <details><summary>details</summary>

  The diff is technically scoped to "show failing path on setup error", but the implementation also surfaces dep-load failures that were previously summary-less. Since [`load_matches.go:31-46`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/pkg/packages/load_matches.go#L31-L46) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/pkg/packages/load_matches.go#L31-L46) populates `pkg.Errors` for any failed `loadSinglePkg` call (including deps loaded via [`load.go:114-150`](https://github.com/gnolang/gno/blob/a26b7d0/gnovm/pkg/packages/load.go#L114-L150) · [↗](../../../../../.worktrees/gno-review-5521/gnovm/pkg/packages/load.go#L114-L150)), the user will now see additional `FAIL    <dep-dir>    [setup failed]` lines when a dependency fails to load. Confirm this is desired, document it in the PR body, or move the new print inside an `if len(pkg.Match) != 0` guard if you want to keep the user-facing scope strictly to top-level paths. The former is probably right since the original bug arises from "you cannot tell which path failed" — that argument applies equally to deps.
  </details>

## Questions for Author

- Was the broadened scope (dep-load failures now also emit `FAIL [setup failed]` summary lines, not just top-level user paths) intentional? See Suggestions.

## Notes (out of scope for this PR)

- CI shows one red check — `gnokms / test` failing `TestRunSignerServer/listener_not_free` in `contribs/gnokms/internal/common`. Unrelated to this diff (no `gnovm/cmd/gno` interaction); appears to be a flaky listener-reuse test. Not a blocker for this PR.
- `pkg.Errors[i].Error()` already embeds the failing path via the `Pos` field (e.g. `parsing gnomod.toml at .: gnomod.toml doesn't exist`), so the new `FAIL <path>` line is a redundancy-for-clarity improvement, not a recovery of previously lost info. The actual UX win is the grep-friendly per-path summary line that matches `go test` conventions, which is the right design choice.
