# PR #5573: fix(tm2): fix three CodeQL high-severity alerts

**URL:** https://github.com/gnolang/gno/pull/5573
**Author:** thehowl | **Base:** master | **Files:** 3 | **+8 -2**
**Reviewed by:** davd-gzl | **Model:** glm-5.1

## Summary

This PR addresses three CodeQL high-severity alerts on master:

1. **`tm2/pkg/amino/tests/common.go`** — `AminoMarshalerInt5.UnmarshalAmino` calls `strconv.Atoi` (returns arch-dependent `int`) and casts the result directly to `int32` without bounds checking. The fix adds a `math.MinInt32`/`math.MaxInt32` guard that returns an error on overflow.

2. **`tm2/pkg/crypto/xsalsa20symmetric/symmetric.go`** — `EncryptSymmetric` computes `nonceLen+secretbox.Overhead+len(plaintext)` as the ciphertext allocation size, which could overflow `int` on 32-bit platforms or with pathological inputs. Replaced with `overflow.Addp` (panics on overflow).

3. **`tm2/pkg/store/prefix/store.go`** — `cloneAppend` computes `len(bz)+len(tail)` as the allocation size, which could overflow `int`. Replaced with `overflow.Addp` (panics on overflow).

The `overflow.Addp` function is a well-tested generic that panics on overflow — consistent with how the codebase already uses it in `tm2/pkg/store/cache/store.go` and `tm2/pkg/store/types/gas.go`.

## Test Results
- **Existing tests:** PASS — all three packages pass (`tm2/pkg/amino/tests`, `tm2/pkg/crypto/xsalsa20symmetric`, `tm2/pkg/store/prefix`)
- **Edge-case tests:** skipped

## Critical (must fix)
None

## Warnings (should fix)
None

## Nits
- [ ] `tm2/pkg/crypto/xsalsa20symmetric/symmetric.go:31` — `overflow.Addp(overflow.Addp(nonceLen, secretbox.Overhead), len(plaintext))` is technically correct, but the inner `Addp(nonceLen, secretbox.Overhead)` adds two small constants (24+16=40) and can never overflow. A single `overflow.Addp(nonceLen+secretbox.Overhead, len(plaintext))` would be clearer and equivalent in safety, since the compiler evaluates the constant addition at compile time. Minor style preference, not a correctness issue.

## Missing Tests
- No test for the new int32 overflow guard in `AminoMarshalerInt5.UnmarshalAmino` (`tm2/pkg/amino/tests/common.go:304`). Codecov flagged 0% patch coverage on these lines. A test with a string representing a value outside `[-2^31, 2^31-1]` (e.g. `"2147483648"`) would confirm the error path works.

## Suggestions
- Consider using `int32(i)` with a range check instead of `math.MinInt32`/`math.MaxInt32` constants for the amino bounds check. Alternatively, use `overflow.Addp` pattern for consistency, though returning an error (rather than panicking) is the correct choice here since this is deserialization input. Current approach is fine.

## Questions for Author
- The `EncryptSymmetric` function now panics on overflow in `Addp`. Is this the desired behavior for a crypto function receiving caller-controlled `plaintext`? In `DecryptSymmetric`, malformed ciphertext returns an error instead. Should `EncryptSymmetric` also return an error on overflow rather than panic? (In practice, overflow here implies an absurdly large plaintext that would fail at `make` anyway, so this is theoretical.)

## Verdict
APPROVE — Clean, minimal fixes for three genuine CodeQL high-severity alerts. The `overflow.Addp` usage is consistent with existing codebase patterns. The int32 bounds check is correct and returns an error (appropriate for deserialization). Only minor gap is the lack of a test for the new amino overflow path.
