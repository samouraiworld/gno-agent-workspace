# PR #5504: fix(grc721): migrate royalty to basis points per EIP-2981

**URL:** https://github.com/gnolang/gno/pull/5504
**Author:** notJoon | **Base:** master | **Files:** 4 | **+62 -27**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR migrates the GRC721 royalty calculation from integer percentages (denominator 100) to basis points (denominator 10,000), aligning with the [EIP-2981](https://eips.ethereum.org/EIPS/eip-2981) specification. This enables fractional royalties (e.g., 2.5%, 0.5%) that were impossible with the old integer-percentage model.

**Changes across 4 files:**

1. **`igrc721_royalty.gno`** — The `RoyaltyInfo` struct field `Percentage int64` is renamed to `Bps int64`, with updated documentation explaining basis-point semantics.
2. **`grc721_royalty.gno`** — Introduces `const bpsDenominator = 10000`. The `royaltyNFT` struct field `maxRoyaltyPercentage` becomes `maxRoyaltyBps`. The `SetTokenRoyalty` validation now checks `Bps ∈ [0, maxRoyaltyBps]` (adds a missing lower-bound check for negative values). The `calculateRoyaltyAmount` helper is inlined into `RoyaltyInfo()`. The royalty formula becomes `overflow.Mul64p(salePrice, Bps) / bpsDenominator`.
3. **`errors.gno`** — `ErrInvalidRoyaltyPercentage` renamed to `ErrInvalidRoyaltyBps`; `ErrCannotCalculateRoyaltyAmount` removed (no longer needed after inlining).
4. **`grc721_royalty_test.gno`** — Existing tests updated to use `Bps` instead of `Percentage`. A new `TestRoyaltyBpsCompliance` test validates fractional royalties (2.5%, 0.5%, 7.5%, 10%, 100%) with table-driven cases.

**Blast radius:** No external callers — no realm or package outside `p/demo/tokens/grc721` uses `NewNFTWithRoyalty`, `SetTokenRoyalty`, `RoyaltyInfo`, or the `RoyaltyInfo` struct. The `RoyaltyInfo` struct is exported so this is technically a breaking API change, but it has zero consumers today.

## Test Results
- **Existing tests:** PASS (all 19 tests pass, including 5 subtests in `TestRoyaltyBpsCompliance`)
- **CI:** All green (build, lint, fmt, test, codecov — all pass)
- **Edge-case tests:** Skipped (see Missing Tests below for gaps)

## Critical (must fix)

None

## Warnings (should fix)

- [ ] `grc721_royalty_test.gno:34-38` — **Pre-existing bug: invalid-token-ID test is a no-op.** Line 34 assigns the error from `SetTokenRoyalty(TokenID("3"), ...)` to `_` (discards it), but line 38 asserts `derr` (the variable from the *previous* successful call, which is `nil`). The assertion `uassert.ErrorIs(t, derr, ErrInvalidTokenId)` always compares `nil` against `ErrInvalidTokenId` — this should fail but `ErrorIs(nil, ErrInvalidTokenId)` in the Gno uassert implementation may silently pass. The PR touched this code (renaming `Percentage` → `Bps`) and should fix it: capture the return value and assert the correct variable.

- [ ] `grc721_royalty.gno:74` — **Panic on large `salePrice * Bps` products.** `overflow.Mul64p` panics on overflow. With `bpsDenominator = 10000`, the product `salePrice * Bps` overflows `int64` when `salePrice > ~922_337_203_685` (≈922 billion) at `Bps = 10000`. While unrealistic for most use cases, the old code had the same issue with a smaller threshold. Consider either: (a) documenting the panic behavior, or (b) using `Div64p(salePrice, bpsDenominator) * Bps` for large sale prices (trades some precision for range). At minimum, this should be documented or tested.

## Nits

- [ ] `grc721_royalty.gno:74` — The division `/ bpsDenominator` truncates toward zero (integer division). For `salePrice=1, Bps=250`, the royalty is `250/10000 = 0`. This is consistent with EIP-2981 (Solidity also truncates), but a brief comment noting "truncation is intentional per EIP-2981" would help future readers.

- [ ] `grc721_royalty_test.gno:55-60` — The test for `Bps > maxRoyaltyBps` uses `int64(10001)` but the caller is `addr2` (set on line 40), not the token owner. Since `SetTokenRoyalty` validates payment address first, then bps, then ownership, the test actually hits `ErrCallerIsNotOwner` or `ErrInvalidTokenId` (token "5" doesn't exist) before reaching the bps check. The test passes because `uassert.ErrorIs` may check that the returned error wraps `ErrInvalidRoyaltyBps` loosely, but it's testing the wrong thing. Mint token "5" to the caller first and ensure the caller is the owner before testing the bps validation path.

## Missing Tests

- [ ] **Negative bps value.** The new validation `royaltyInfo.Bps < 0` at `grc721_royalty.gno:43` is untested. Add a test case with `Bps: -1` that expects `ErrInvalidRoyaltyBps`.
- [ ] **Zero bps.** `Bps = 0` is allowed by the validation. Add a test confirming `RoyaltyInfo()` returns `royaltyAmount = 0`.
- [ ] **Rounding/truncation edge cases.** Small sale prices (e.g., `salePrice=1, Bps=250`) where the royalty truncates to 0. This is correct per EIP-2981 but worth documenting with a test.
- [ ] **Large salePrice near int64 max.** Test behavior when `salePrice * Bps` would overflow — should panic via `Mul64p`. Having an explicit test documents this contract.

## Suggestions

- Consider adding a `0%` (Bps=0) test case to `TestRoyaltyBpsCompliance` to verify zero royalties work. `grc721_royalty_test.gno:84-90`.
- The PR body mentions `FeeDenominator` as an alternative name. `bpsDenominator` is clear and matches EIP-2981 terminology well; the current naming is fine.
- The commit structure (first commit adds a failing test, second fixes it) is clean and appreciated — makes the change easy to verify.

## Questions for Author

- The `AGENTS.md` in the gno repo notes "Break `gno.land/p/demo/` backwards-compat — needs discussion." The `RoyaltyInfo.Percentage` → `RoyaltyInfo.Bps` rename is a breaking change to an exported type. Since there are zero consumers today, this is likely fine, but was this discussed with maintainers? The PR body suggests it was ("we considered keeping `Percent`... we decided to rename"), so this seems intentional.

## Verdict

**APPROVE** — Clean, well-motivated migration to EIP-2981 basis points. The core logic is correct and well-tested. The warnings are about test quality rather than production correctness; the pre-existing test bug (discarded error variable) should be fixed in a follow-up.
