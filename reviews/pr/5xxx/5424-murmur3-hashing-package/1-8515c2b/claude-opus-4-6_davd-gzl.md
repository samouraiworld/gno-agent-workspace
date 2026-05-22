# PR #5424: feat(examples): add `murmur3` hashing package

**URL:** https://github.com/gnolang/gno/pull/5424
**Author:** jeronimoalbi | **Base:** master | **Files:** 9 | **+623 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR adds a new MurmurHash3 package at `gno.land/p/jeronimoalbi/murmur3`, implementing Austin Appleby's non-cryptographic hash algorithm. The package provides two hash variants:

1. **32-bit** (`murmur3_32.gno`): A faithful implementation of MurmurHash3_x86_32 that implements the `hash.Hash32` interface. Supports seeded and unseeded hashing, incremental writes, and the standard `Sum`/`Sum32`/`Reset` methods. The algorithm follows the canonical reference implementation with correct constants (c1=0xcc9e2d51, c2=0x1b873593), rotation values (15, 13), and finalization mix.

2. **64-bit** (`murmur3_64.gno`): A **non-standard** 64-bit variant constructed by concatenating two independent 32-bit hashes with different seeds (upper 32 bits = seed1, lower 32 bits = seed2). Implements `hash.Hash64`. Note: this is NOT the official MurmurHash3_x86_128 or MurmurHash3_x64_128 variant.

A utility function `EncodeToString(uint64)` converts hash values to hex strings. The package includes comprehensive tests against Wikipedia reference vectors for the 32-bit variant, plus incremental write, reset, and state-immutability tests for both variants. A filetest and README with embedded example are also included.

The only unrelated change is removing an extra blank line in `examples/gno.land/r/sys/users/admin.gno`.

## Test Results
- **Existing tests:** PASS (all 12 tests + 1 filetest pass)
- **Edge-case tests:** skipped (algorithm correctness verified against Wikipedia reference vectors)

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `murmur3_64.gno:17,32` and `murmur3_32.gno:30,41` — Typo: "provability raises" should be "probability increases/rises". Appears in 4 doc comments across both files.
- [ ] `murmur3_64.gno:12-86` — The 64-bit variant is a custom construction (two concatenated 32-bit hashes), NOT a standard MurmurHash3 variant. This should be clearly documented in the doc comments and README. Users expecting standard MurmurHash3_128 behavior will be misled. The Wikipedia reference cited in the 64-bit test file (`murmur3_64_test.gno:10`) only covers 32-bit vectors, making the comment misleading.
- [ ] `murmur3_32.gno:28,38` — Off-by-one in doc: "ranging from 0 to 4_294_967_296" — a 32-bit hash ranges from 0 to 4,294,967,295 (2^32 - 1). The "2 to the power of 32 unique values" part is correct, but the explicit range upper bound is wrong.
- [ ] `murmur3_64.gno:72` — Return value of `d.d1.Write(bz)` is silently discarded. While the current implementation never errors, this is fragile if the internal API ever changes.
- [ ] `murmur3_64_test.gno` — No test for `Sum64WithSeed` or `NewWithSeed64` with custom seeds. No test that `NewWithSeed64` panics when seeds are equal.

## Nits
- [ ] `murmur32.gno` — Filename inconsistency. Other files use the pattern `murmur3_32.gno`/`murmur3_64.gno`, but this file is `murmur32.gno`. Since it contains the package doc and the `EncodeToString` utility (not 32-bit-specific code), renaming to `murmur3.gno` or `encode.gno` would be more consistent.
- [ ] `murmur32.gno:18` — `EncodeToString` does not zero-pad hex output. `strconv.FormatUint(0, 16)` returns `"0"`, not `"00000000"` or `"0000000000000000"`. For hash functions, fixed-width output is typically expected. Additionally, the function requires callers to cast `uint32` to `uint64` for 32-bit hashes, which is ergonomically awkward (visible in the README example).
- [ ] `murmur3_32.gno:143` — Finalization shift reuses `r2` (13) which is semantically a block rotation constant, not a finalization constant. They share the same value by coincidence. If `r2` were ever changed for block processing, the finalization would silently break.
- [ ] `murmur3_32_test.gno`, `murmur3_64_test.gno` — Inconsistent use of hex vs decimal for expected values. `TestHash32Sum32` uses hex (`0xc0363e43`), but `TestHash32IncrementalWrite` uses decimal (`3224780355`) for the same value.

## Missing Tests
- [ ] `NewWithSeed64` panic on equal seeds — `murmur3_64_test.gno`
- [ ] `Sum64WithSeed` with non-default seeds — `murmur3_64_test.gno`
- [ ] `Sum32()` convenience function (currently only `Sum32WithSeed` is directly tested) — `murmur3_32_test.gno`
- [ ] Data with length exactly divisible by block size (4 bytes) — both test files
- [ ] `Write(nil)` and `Write([]byte{})` — both test files
- [ ] `Sum(existingSlice)` — append to non-nil slice — both test files

## Suggestions
- Consider adding a separate `EncodeToString32(uint32) string` to avoid the `uint64` cast in 32-bit usage. The current API forces `sum32 := uint64(h32.Sum32()); murmur3.EncodeToString(sum32)` which is unnecessary ceremony — `murmur32.gno`.
- Add a note about thread safety in the README. The hash objects are stateful and not safe for concurrent use — `README.md`.
- The `digest64.Write` method writes to both `d1` and `d2` sequentially. If performance matters, this could be optimized, but for Gno's use case this is likely fine — `murmur3_64.gno:72-74`.

## Questions for Author
- Is the non-standard 64-bit variant intentional? If users need a standard 128-bit MurmurHash3, this package would not serve that purpose. Would it be worth documenting this explicitly or renaming to something like `Sum64Dual` to signal the concatenation approach?
- Any plans to add the 128-bit variants (MurmurHash3_x86_128 / MurmurHash3_x64_128)?

## Verdict
**APPROVE** — The 32-bit implementation is correct and well-tested against reference vectors. The 64-bit variant is a reasonable pragmatic approach. The issues are all documentation/naming concerns and missing edge-case tests, none of which are blockers. The typos and doc inaccuracies should be fixed before merge.
