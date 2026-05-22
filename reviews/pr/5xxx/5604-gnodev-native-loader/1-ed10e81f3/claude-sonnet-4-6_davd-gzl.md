# PR #5604: feat: gnodev native loader

**URL:** https://github.com/gnolang/gno/pull/5604
**Author:** gfanton | **Base:** master | **Files:** 56 | **+2280 -1789**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Introduces a native filesystem-based package loader for `gnodev`, replacing the previous approach of fetching all packages from the running gnoland node. The new `Loader` struct (`contribs/gnodev/pkg/packages/`) delegates bulk workspace loading to `gnovm/pkg/packages/Load` and handles per-path lazy resolution locally. Lock discipline: RLock fast-path for cache hits, Write-lock for FS walk, no-lock for RPC fetch. `KindUnknown` as zero value provides defensive default.

Key design: packages discovered from the local filesystem are served directly from disk rather than via RPC, significantly reducing dev-loop latency. Remote override flag allows hybrid local+remote resolution.

## Test Results
- **Existing tests:** ALL PASS (111/111 across gnodev + gnovm/pkg/packages)
- **Edge-case tests:** skipped

## Critical (must fix)
None.

## Warnings (should fix)
- [ ] `loader.go:475-493` — `stripStdlibs` uses `pkgs[:0]` aliasing (same backing array); mutates `p.Imports` in-place. Safe today but fragile for future callers. Use `pkgs[:0:0]` to make no-alias intent explicit.
- [ ] `package.go:62-64` — `packageFromMemPackage` sets `Kind: KindFS` with comment saying classification happens at call site; default to `KindUnknown` instead to make misuse visible.
- [ ] `loader.go:271-321` — `Reload` may double-resolve paths in both `l.tracked` and workspace result; wasteful but not incorrect.
- [ ] `app.go:337-401` — Lock-order `LockEmit → muNode → l.mu` is consistent but undocumented; add comment to prevent future inversion.

## Nits
- [ ] Discovery banner writes to raw `os.Stderr` instead of `cio.Err()`.
- [ ] README has pre-existing typos.
- [ ] `nodeHolder` test accesses unexported `n.paths` directly; fragile.

## Missing Tests
- [ ] Reload with workspace+tracked path overlap.
- [ ] `LookupFS` multi-root cold-path.
- [ ] `scanRoot` with `module = ""`.
- [ ] `-remote-override` wiring end-to-end.

## Suggestions
- Document the lock-order invariant in a comment near `app.go:337`.
- Consider a `KindUnknown` sentinel check at `Loader.Lookup` entry to catch mis-classified packages early.

## Questions for Author
- Is there a plan to support workspace-relative symlinks in `scanRoot`?
- Does the `-remote-override` flag have precedence over local FS hits, or vice versa?

## Verdict
APPROVE — Architecture is sound, lock discipline is well-designed, all 111 tests pass. Warnings are minor and non-blocking.
