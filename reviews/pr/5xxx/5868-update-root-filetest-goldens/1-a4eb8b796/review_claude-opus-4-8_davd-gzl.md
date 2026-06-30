# PR #5868: fix(gnovm): update root-level filetest goldens

URL: https://github.com/gnolang/gno/pull/5868
Author: notJoon | Base: master | Files: 2 | +46 -1
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a4eb8b796 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5868 a4eb8b796`

**TL;DR:** `gno test -update-golden-tests` rewrites the `// Output:` block of a `*_filetest.gno` in place. The writer always aimed at `filetests/<name>`, so a filetest sitting at the package root (no `filetests/` subdir) ran fine but crashed on update with `no such file or directory`. The fix probes where the file actually lives and writes there.

**Verdict: APPROVE** — small, correctly targeted fix; both source layouts now covered by tests.

## Summary
The filetest loop builds the write-back path for golden updates. The old line hardcoded `filepath.Join(fsDir, "filetests", testFileName)` ([test.go:419](https://github.com/gnolang/gno/blob/a4eb8b796/gnovm/pkg/test/test.go#L419) · [↗](../../../../../.worktrees/gno-review-5868/gnovm/pkg/test/test.go#L419)), correct only when the package keeps its filetests in a `filetests/` subdir. Root-level filetests have no such subdir, so the write target did not exist and `os.WriteFile` failed. The new `filetestPath` helper ([test.go:462-468](https://github.com/gnolang/gno/blob/a4eb8b796/gnovm/pkg/test/test.go#L462-L468) · [↗](../../../../../.worktrees/gno-review-5868/gnovm/pkg/test/test.go#L462-L468)) stats the subdir path and falls back to the root path when it is absent. The probe targets the source filetest's own file, which always exists because it was just loaded, so the fallback is reliable.

## Fix
`filetestPath(fsDir, name)` returns `fsDir/filetests/name` when that file exists on disk, else `fsDir/name`. Routing is decided per filetest at [test.go:419](https://github.com/gnolang/gno/blob/a4eb8b796/gnovm/pkg/test/test.go#L419) · [↗](../../../../../.worktrees/gno-review-5868/gnovm/pkg/test/test.go#L419) and consumed only by the golden write at [test.go:438](https://github.com/gnolang/gno/blob/a4eb8b796/gnovm/pkg/test/test.go#L438) · [↗](../../../../../.worktrees/gno-review-5868/gnovm/pkg/test/test.go#L438). No behavior change when `opts.Sync` is false (`changed == ""`).

Verified on a4eb8b796: reverting the helper call back to the hardcoded `filetests/` join makes the new txtar fail with the exact pre-fix error `could not fix golden file: open filetests/x_filetest.gno: no such file or directory`; with the fix the root-level golden is written and `cmp` matches. The existing `filetests/`-subdir sync tests (`output_sync`, `error_sync`, `realm_sync`) still route to the subdir, so both branches of the helper are exercised.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. The new `output_sync_root_filetest.txtar` covers the root-level branch; the pre-existing `output_sync`/`error_sync`/`realm_sync` txtars cover the `filetests/`-subdir branch.

## Suggestions
- `gnovm/pkg/test/test.go:462` — the helper rediscovers the source location with an `os.Stat` probe instead of the loader recording where each filetest was read from.
  <details><summary>details</summary>

  The package loader already knew whether each filetest came from `filetests/` or the root, but `MemPackage.File` carries only `Name`, so that fact is dropped and reconstructed here by stat-ing the disk. The probe is correct (it targets the source file, which exists), so this is purely an altitude note, not a defect. If filetest provenance ever needs to be authoritative, plumbing the original relative path through the loader would remove the filesystem round-trip. No change needed for this PR.
  </details>

## Open questions
- A package holding both a root-level `x_filetest.gno` and a `filetests/x_filetest.gno` would, on update of the root one, write to the subdir copy (same `Name`, subdir wins the probe). Pathological layout, not worth guarding; noting it in case the loader is ever changed to merge both.
