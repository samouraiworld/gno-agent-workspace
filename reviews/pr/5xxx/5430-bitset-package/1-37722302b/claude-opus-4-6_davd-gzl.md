# PR #5430: feat(examples): add `bitset` package

**URL:** https://github.com/gnolang/gno/pull/5430
**Author:** jeronimoalbi | **Base:** master | **Files:** 5 | **+551 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

Adds a new `gno.land/p/jeronimoalbi/bitset` package implementing an arbitrary-size bit set (bit array) backed by a `[]uint64` slice. The package provides `Set`, `Clear`, `ClearAll`, `Test`, `Size`, `Len`, `And`, `Or`, `Xor`, `Equals`, and `String` operations. The BitSet auto-grows its backing slice as needed via `grow()`. The `New()` constructor pre-allocates capacity.

The PR includes 322 lines of tests covering all public methods, a filetest for the README example, and a README with API documentation. All CI checks pass. This is a pure-additive package under the author's namespace with no impact on existing code.

## Test Results
- **Existing tests:** PASS (all 11 test functions + 1 filetest pass via `gno test -v`)
- **Edge-case tests:** skipped (findings documented below)

## Critical (must fix)

- [ ] `bitset.gno:68-75` — **`And` does not clear bits beyond `other`'s range.** The loop `for i := range other.words` only ANDs the words that exist in `other`. If `b` has more words than `other` (e.g., `b` has bits at position 200 and `other` only has bits in the first word), the trailing words in `b` are left untouched. Correct AND semantics require that any bit in `b` not present in `other` should be cleared. The fix is to zero out `b.words[len(other.words):]` after the loop.

- [ ] `bitset.gno:21-24` — **`Set` panics on negative indices >= -64.** `Set(-1)` computes `wordIndex(-1) = 0`, `bitMask(-1) = 1 << uint(-1 % 64)`. In Go, `-1 % 64 = -1`, and `uint(-1)` is `MaxUint64`, so `1 << MaxUint64 = 0` (shift >= 64 yields 0). The bit is silently not set. However, `Set(-65)` computes `wordIndex(-65) = -1` (Go integer division truncates toward zero), `grow(-1)` is a no-op, and `b.words[-1]` panics with index out of range. `Clear` and `Test` have the same issue. All methods accepting `int` positions should either document that negative indices are invalid or guard against them (e.g., panic or return early).

## Warnings (should fix)

- [ ] `bitset.gno:124-131` — **`String` produces incorrect output for multi-word bitsets.** `strconv.FormatUint(w, 2)` strips leading zeros from each word. For a bitset with bit 0 and bit 64 set, `words = [1, 1]`, and `String()` returns `"11"` — which looks like bits 0 and 1 are set, not bits 0 and 64. Each word after the first should be zero-padded to 64 characters to preserve bit positions. Without this, `String()` output is misleading and cannot be round-tripped.

- [ ] `bitset.gno:46` — Minor: typo "setted" should be "set" in the comment `// Bit has not been setted yet`.

## Nits

- [ ] `bitset.gno:5` — Consider naming the constant to clarify its purpose, e.g., `bitsPerWord` is fine but some implementations use `wordSize`. This is subjective.

## Missing Tests

- [ ] **`And` with different-length bitsets where receiver is longer.** E.g., `a.Set(200); b.Set(0); a.And(b)` — should clear bit 200 in `a`, but the current implementation does not.
- [ ] **Negative index behavior.** No tests for `Set(-1)`, `Set(-65)`, `Test(-1)`, `Clear(-65)`. These would expose the panic.
- [ ] **`String` with bits across multiple words.** E.g., `Set(0); Set(64)` — the current output `"11"` is incorrect; a test asserting the correct 128-character string would catch this.

## Suggestions

- Consider adding `Not()` (complement) and `Flip(i)` (toggle bit) operations, which are common in bitset APIs. Not blocking.
- `countSetBits` using Kernighan's algorithm is O(k) per word where k = popcount. For GnoVM this is fine, but worth documenting the choice. `bitset.gno:151-158`.
- `ClearAll` zeros words but doesn't shrink the slice. Consider adding a `Reset()` that also reclaims memory, or document that `ClearAll` preserves capacity. `bitset.gno:36-40`.

## Questions for Author

- Is the `And` behavior (not clearing trailing bits) intentional? Standard bitwise AND should only retain bits present in both sets.
- Is the `String()` representation intended for debugging or for serialization? If the latter, zero-padding is essential.

## Verdict

REQUEST CHANGES — `And` has incorrect semantics (does not clear bits beyond `other`'s range), and negative indices cause panics for positions <= -65.
