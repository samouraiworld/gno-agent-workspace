# PR #5446: fix(gnovm): use side-table for PkgID flags instead of stealing hash bits

**URL:** https://github.com/gnolang/gno/pull/5446
**Author:** albttx | **Base:** dev/jae/gas-model-improvements-storage2 | **Files:** 64 | **+886 -850**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR replaces the approach of stealing 4 high bits from the Murmur3 hash in `PkgID` to encode package flags (stdlib, immutable) with a `sync.Map` side-table (`pkgIDFlags`). Previously, `PkgIDFromPkgPath()` would compute the hash and then mangle the top nibble to encode whether a package was stdlib or immutable. This caused the stored PkgID to differ from the raw hash, which was confusing and fragile.

The new approach stores the 32-bit hash unmodified in `PkgID` and maintains a separate `sync.Map` (`pkgIDFlags`) keyed by `PkgID` that stores a `pkgIDFlag` bitfield. Two flag bits are defined: `pkgIDFlagStdlib` (bit 0) and `pkgIDFlagImmutable` (bit 1). The `IsStdlibPkg()` and `IsImmutablePkg()` methods on `PkgID` now look up the side-table instead of inspecting hash bits.

A second `sync.Map` (`pkgIDFromPkgPathCache`) caches `pkgPath -> PkgID` mappings to avoid recomputing the hash. Both maps are populated in `PkgIDFromPkgPath()`.

The PR also updates `gno.land/adr/gas_refactor.md` to document the new approach, and regenerates ~62 golden test files whose expected PkgID values changed because the hash is no longer mangled (e.g., `09b6b63c` reverts to `f9b6b63c`).

Key files affected:
- `gnovm/pkg/gnolang/realm.go` — Core implementation (lines 87-156): side-table types, `PkgIDFromPkgPath()`, `IsStdlibPkg()`, `IsImmutablePkg()`, helper constants.
- `gnovm/pkg/gnolang/store.go:448` — Call site for `IsStdlibPkg()` in stdlib key bytes cache.
- `gnovm/pkg/gnolang/realm.go:271,292,539,634` — Call sites for `IsImmutablePkg()` in refcount optimization.
- `gno.land/adr/gas_refactor.md` — ADR documentation update.
- `gnovm/tests/files/*.go` — ~62 regenerated golden test files.

## Test Results
- **Existing tests:** PASS — `go test ./gnovm/pkg/gnolang/ -run Files -test.short` passed in the worktree.
- **CI status:** Multiple failures (lint, build, tests) — however this targets `dev/jae/gas-model-improvements-storage2` (not master), so failures likely originate from the base branch rather than this PR's changes.
- **Edge-case tests:** Skipped — the logic is straightforward map lookups; existing filetests cover the affected packages.

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `gnovm/pkg/gnolang/realm.go:108-127` — **Race condition window between two sync.Map stores.** `PkgIDFromPkgPath()` calls `pkgIDFromPkgPathCache.LoadOrStore` (line 123) BEFORE `pkgIDFlags.LoadOrStore` (line 125). A concurrent goroutine that calls `PkgIDFromPkgPath()` for the same path will get a cache hit at line 109, returning the `PkgID` immediately — but `pkgIDFlags` may not yet be populated. This means `IsStdlibPkg()` and `IsImmutablePkg()` will return `false` for that PkgID during the window. The current behavior is documented as a "conservative default" (line 142: "For unknown PkgIDs, return false (conservative default)"), and returning `false` is safe: `IsImmutablePkg()` returning false just means refcounting isn't skipped (extra work, no correctness issue), and `IsStdlibPkg()` returning false means the stdlib key cache is bypassed (slower, no correctness issue). However, this is still a latent bug — if any future caller requires accurate flags, it will silently misbehave. **Fix: swap the order — populate `pkgIDFlags` before `pkgIDFromPkgPathCache`**, so any goroutine that can see the cached PkgID will also see its flags.

- [ ] `gnovm/pkg/gnolang/realm.go:87-91` — **`pkgIDFlags` and `pkgIDFromPkgPathCache` are package-level `sync.Map` globals that grow monotonically.** In long-running processes or test suites that create many packages, these maps accumulate entries indefinitely. There is no eviction, reset, or size limit. This is probably fine for the current use case (package paths are bounded), but worth noting as a design constraint.

## Nits

- [ ] `gnovm/pkg/gnolang/realm.go:140-142` — The comment "For unknown PkgIDs, return false (conservative default)" on `IsStdlibPkg()` is helpful but should be replicated on `IsImmutablePkg()` (line 149) for consistency. Currently `IsImmutablePkg()` has no equivalent comment.

- [ ] `gnovm/pkg/gnolang/realm.go:95-96` — The constants `pkgIDFlagStdlib` and `pkgIDFlagImmutable` use explicit shift notation (`1 << 0`, `1 << 1`) which is clear, but the type `pkgIDFlag` is `uint8` — a comment noting that only 2 of 8 bits are used and the rest are reserved for future flags would document the design intent.

## Missing Tests

- [ ] No unit test for the race condition scenario (concurrent `PkgIDFromPkgPath` calls for the same path). A test with `-race` and multiple goroutines calling `PkgIDFromPkgPath` simultaneously would validate thread safety — `gnovm/pkg/gnolang/realm.go:108-127`.

- [ ] No test verifies that `IsStdlibPkg()` / `IsImmutablePkg()` return `false` for a `PkgID` that was never registered via `PkgIDFromPkgPath()` — this is the documented "conservative default" behavior but isn't explicitly tested — `gnovm/pkg/gnolang/realm.go:138-156`.

## Suggestions

- Swap the `sync.Map` store order in `PkgIDFromPkgPath()` (populate flags before cache) to eliminate the race window. This is a one-line swap at `realm.go:123-125` and makes the code correct-by-construction rather than relying on "conservative defaults."

- Consider adding a brief doc comment on the `pkgIDFlags` and `pkgIDFromPkgPathCache` variables explaining their purpose and lifecycle, since they are package-level global state that other developers need to understand — `gnovm/pkg/gnolang/realm.go:87-91`.

## Questions for Author

- The PR includes filetest changes (e.g., `addressable_1b_err.gno`, `addressable_1d_err.gno`) that appear to be Go 1.25 error message updates — the same changes present in PR #5441. Is this because the base branch (`dev/jae/gas-model-improvements-storage2`) already incorporates those changes, or will this create merge conflicts when either PR lands?

- Is there a reason the flags side-table uses `sync.Map` rather than a regular `map` with a `sync.RWMutex`? For a read-heavy workload with bounded keys, `RWMutex` can be faster and uses less memory. `sync.Map` is optimal when keys are mostly written once and read many times by many goroutines, which may or may not match the actual access pattern here.

## Verdict

**APPROVE** — The core change is a clear improvement: the side-table approach is simpler, removes hash bit mangling, and the "conservative default" behavior makes the race window safe in practice. The one-line fix (swapping store order) would eliminate the race entirely and should be easy for the author to address.
