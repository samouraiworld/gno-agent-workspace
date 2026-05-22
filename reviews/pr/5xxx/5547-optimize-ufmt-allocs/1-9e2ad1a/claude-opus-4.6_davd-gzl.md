# PR #5547: perf(ufmt): reduce allocations and eliminate nested Sprintf in print paths

**URL:** https://github.com/gnolang/gno/pull/5547
**Author:** notJoon | **Base:** master | **Files:** 1 | **+134 -140**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR optimizes `gno.land/p/nt/ufmt/v0` for gas reduction across all formatting paths. The single changed file is `ufmt.gno`. Key changes:

1. **Byte-level format parsing** — `doPrintf` no longer converts the format string to `[]rune`. It iterates by byte index. Since all format metacharacters (`%`, digits, `.`, verb letters) are ASCII, this is safe — non-ASCII bytes cannot alias with them.

2. **`parseNumber` helper** — Replaces an inline closure + `strconv.Atoi` with a simple accumulator loop that avoids substring allocation.

3. **`strconv.Append*` throughout** — `AppendInt`, `AppendUint`, `AppendFloat`, `AppendQuote` write directly into the `buffer` (which is `[]byte`), eliminating intermediate strings from `Itoa`/`FormatXxx` + `writeString`.

4. **`doPrint` inlines integer/float formatting** — Previously delegated to nested `Sprintf("%d", v)` / `Sprintf("%f", v)`, now uses direct `strconv.Append*` calls per type.

5. **`writeFallback` replaces `fallback` + `typeToString`** — Single function builds the `"%!verb(type=value)"` mismatch diagnostic via string concatenation + one `writeString`, removing multi-step construction and nested Sprintf.

6. **Padding loop** — `writeStringWithLength` writes spaces with a byte loop instead of `strings.Repeat`.

7. **`writeChar` uses `string(rune(v))` + `writeString`** — Cheaper in GnoVM than `writeRune`'s `utf8.AppendRune` path.

8. **`writeValue` inlines bool** — Avoids function call to `writeBool`.

The PR reports ~5.7% total gas reduction across all test cases. All CI checks pass.

## Test Results
- **Existing tests:** PASS (all 14 test functions)
- **Edge-case tests:** skipped (no new test code added by PR, no test changes)

## Critical (must fix)
None

## Warnings (should fix)
- [ ] `ufmt.gno:242-251` — `parseNumber` has no overflow protection. A format string like `%99999999999999999999s` will silently overflow `int` and produce a nonsensical width value. The old code using `strconv.Atoi` would have panicked with "invalid length specification". While this is an edge case unlikely in practice, it's a behavioral regression — the old code rejected overflow, the new code silently wraps. Consider adding a cap (e.g., `if num > 999999 { panic("ufmt: width/precision overflow") }`).

## Nits
- [ ] `ufmt.gno:274` — `int64(v)` where `v` is already `int64` is a no-op cast. Minor redundancy.
- [ ] `ufmt.gno:360` — Same: `int64(v)` where `v` is `int64` in `writeInt`.

## Missing Tests
- [ ] Format string with very large width/precision (e.g., `%999999999999999999s`) to verify overflow behavior — relevant given `parseNumber` change.
- [ ] Multi-byte UTF-8 characters in format string literals (e.g., `Sprintf("日本語%d", 1)`) — byte-level parsing should handle this correctly but there's no explicit test for it.
- [ ] `%%` at end of format string (e.g., `Sprintf("100%%")`) — edge case for the new trailing-`%` logic.

## Suggestions
- The repeated type-switch patterns for `int`, `int8`, ..., `uint64` appear in `doPrint`, `writeValue`, `writeInt`, and `writeFallback`. The PR description explains why extracting a helper regressed gas (+10,600). A code-generation comment noting this tradeoff would help future maintainers understand the intentional duplication. (`ufmt.gno:68-87`, `ufmt.gno:265-284`, `ufmt.gno:350-370`, `ufmt.gno:573-593`)
- `writeFallback` line 600: the 4-part concatenation `"%!" + string(verb) + "(" + body + ")"` allocates intermediate strings. Since the PR notes that 3+ fragment concat beats multi-segment append, this is intentional, but a brief inline comment would clarify the rationale for reviewers.

## Questions for Author
- The `writeHex` and `writeQuotedString` default cases still emit `"(unhandled)"` rather than calling `writeFallback`. Was this intentional (to match prior behavior) or an oversight? Standard Go's `fmt` would produce `%!x(string=foo)` for a type mismatch, not `(unhandled)`.

## Verdict
APPROVE — Clean, well-motivated gas optimization with no correctness regressions. The `parseNumber` overflow is a minor behavioral difference worth addressing but not a blocker.
