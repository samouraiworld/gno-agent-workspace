# PR #5522: fix(r/sys/validators/v2): Render was showing oldest changes instead of newest

**URL:** https://github.com/gnolang/gno/pull/5522
**Author:** D4ryl00 | **Base:** master | **Files:** 2 | **+65 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

The `Render` function in `r/sys/validators/v2` was displaying the oldest valset changes instead of the newest. The root cause was an incorrect `offset` argument to `changes.ReverseIterateByOffset(size-maxDisplay, maxDisplay, ...)`.

`ReverseIterateByOffset(offset, count, ...)` internally calls `TraverseByOffset(offset, count, ascending=false, ...)`. With `ascending=false`, offset 0 means "start from the highest key" (newest block). So `offset = size - maxDisplay` was skipping the newest entries and iterating the oldest — the exact opposite of intent. A secondary bug: when `size < maxDisplay`, the offset goes negative, which is invalid.

The fix changes the call to `ReverseIterateByOffset(0, maxDisplay, ...)`, which correctly starts from the newest end and returns up to `maxDisplay` entries. The `maxDisplay` count naturally caps iteration when `size > maxDisplay`, and when `size < maxDisplay` it just returns all entries.

Two new tests validate both scenarios (over-limit and under-limit).

## Test Results
- **Existing tests:** Could not run (`gno` binary not installed in the review environment). CI passes all checks.
- **Edge-case tests:** 2 new tests added by the PR author covering the exact bug scenarios.

## Critical (must fix)
None

## Warnings (should fix)
None

## Nits
- [ ] `validators_test.gno:159` — `const maxDisplay = 10` is redeclared locally in the test. If the package-level `maxDisplay` ever changes, this test constant would silently drift. Consider referencing or exporting the real value instead of duplicating it.

## Missing Tests
None — the two new tests directly cover the fixed behavior for both boundary conditions.

## Suggestions
None

## Questions for Author
None

## Verdict
APPROVE — Clean, minimal fix for a clear off-by-one logic error, with good test coverage for both edge cases.
