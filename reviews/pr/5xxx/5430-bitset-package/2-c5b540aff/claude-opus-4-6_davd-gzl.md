# PR #5430: feat(examples): add `bitset` package

**URL:** https://github.com/gnolang/gno/pull/5430
**Author:** jeronimoalbi | **Base:** master | **Files:** 5 | **+766 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This is a follow-up review of the incremental changes between `fd010c64` and `c5b540af` (9 new commits by the author plus master merges). The PR adds a `gno.land/p/jeronimoalbi/bitset` package implementing an arbitrary-size bit array backed by `[]uint64`.

Since the previous review (round 1 at `37722302b`), all critical and warning findings have been addressed:

1. **`And` now clears trailing bits** (`bitset.gno:70-78`) — The loop was rewritten to iterate over `b.words` and zero out words beyond `other`'s range. This is correct.
2. **Negative index panic eliminated** — All position parameters changed from `int` to `uint64`, making negative indices impossible at the type level. `New()` also takes `uint64`.
3. **`String()` output fixed** — Replaced per-word `strconv.FormatUint` (which dropped leading zeros) with a new `PaddedString()` method that builds a zero-padded MSB-first buffer. `String()` trims leading zeros from `PaddedString()`. Multi-word bitsets now render correctly.
4. **`grow()` now takes length directly** (`bitset.gno:160-164`) — Callers pass the desired slice length rather than an index, eliminating the off-by-one confusion.
5. **Typo fixed** — "setted" changed to "set" (`bitset.gno:48`).
6. **`Equals` renamed to `Equal`** — Follows Go convention (e.g., `time.Time.Equal`).

Additional changes in this range:
- New `TestBitSetSet` table tests covering word boundaries, cross-word bits, and idempotent set.
- `TestBitSetPaddedString` and `TestBitSetString` now include `max word value` (bit 63) cases.
- All test slices changed from `[]int` to `[]uint64`.
- `Set()` now inlines the grow check (`if idx >= len(b.words)`) instead of delegating to `grow()`, which is slightly cleaner.
- `Or` and `Xor` store `otherLen := len(other.words)` to avoid re-evaluating inside the condition — minor but clean.

## Test Results
- **Existing tests:** PASS (13 test functions + 1 filetest, all pass via `gno test -v`)
- **Edge-case tests:** skipped (all previous findings resolved, no new fragile code paths identified)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `bitset.gno:160-163` — **`grow` always allocates a new slice**, even when the existing slice already has sufficient capacity. If a caller does `grow(len(b.words))` (same length), it still allocates and copies. This is not a bug since current callers guard with `idx >= len(b.words)`, but `grow` itself is not defensive. Consider adding `if length <= len(b.words) { return }` at the top for robustness, or at minimum, use `if length <= cap(b.words) { b.words = b.words[:length]; return }` to exploit pre-allocated capacity from `New()`.

## Nits

- [ ] `bitset.gno:140` — The cast `uint64(bitsPerWord-1-j)` is unnecessary since `j` is an `int` and the shift operand is automatically promoted. `w >> (bitsPerWord - 1 - j) & 1` would suffice.
- [ ] `bitset_test.gno:439-441` — The `Equal` test's assertion logic is inverted from the usual pattern: `if tc.want == got { return }`. While correct, it reads slightly backwards compared to the other tests which use `if got != tc.want`. Not worth changing but noted for consistency.

## Missing Tests

- [ ] **Multi-word `PaddedString`/`String` test.** All current `PaddedString` tests use bits within a single word (0-63). A test with e.g. bits 0 and 64 set would verify the word-ordering logic across multiple words in `PaddedString`. The `TestBitSetSet` "bit across word boundary" case covers `String()` for this scenario, but `PaddedString` lacks an equivalent.
- [ ] **`New` with non-aligned size.** `New(65)` should allocate 2 words (128 bits). `New(1)` should allocate 1 word. These boundary cases are not explicitly tested.

## Suggestions

- Consider adding `Not()` (complement) and `Flip(i uint64)` (toggle bit) operations, which are common in bitset APIs. Not blocking.
- The README API section only lists `New()`. Consider listing at least the core methods (`Set`, `Clear`, `Test`, `And`, `Or`, `Xor`, `Equal`, `String`, `PaddedString`, `Len`, `Size`) for discoverability.
- `ClearAll` zeros words but doesn't shrink the slice. Consider documenting this explicitly or adding a `Reset()` that reclaims memory. `bitset.gno:37-42`.

## Questions for Author

- None — all previous questions have been addressed.

## Verdict

APPROVE — All critical and warning findings from round 1 have been resolved. The `uint64` transition, `And` fix, `String` rewrite, and `grow` simplification are all correct. The package is well-tested (514 lines of tests for 182 lines of implementation). Remaining items are minor nits and potential enhancements.
