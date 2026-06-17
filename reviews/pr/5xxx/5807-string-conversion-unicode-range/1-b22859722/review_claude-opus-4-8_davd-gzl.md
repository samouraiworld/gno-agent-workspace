# PR #5807: fix(gnovm): respect Unicode range in 64-bit integer-to-string conversions

URL: https://github.com/gnolang/gno/pull/5807
Author: omarsy | Base: master | Files: 3 | +135 -4
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `b22859722` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5807 b22859722`

**TL;DR:** Converting a 64-bit integer to a string in Gno (e.g. `string(someInt64)`) is supposed to give back the character at that Unicode code point, and anything out of range should give the replacement character `�`. Gno chopped the number down to 32 bits first, so huge values could land back on a real character instead of `�`. This PR adds two small helpers that detect the out-of-range case up front and return `�`, matching Go exactly.

**Verdict: APPROVE** — correct, minimal, byte-for-byte verified against the Go toolchain across every range boundary; no open concerns.

## Summary
`string(x)` for `int`/`int64`/`uint`/`uint64` ran `string(rune(x))`, and `rune(...)` truncates to int32 *before* Go's own code-point range check, so a value like `uint64(0x10001F600)` wrapped onto a valid code point (`😀`) instead of yielding `�`. The fix routes those four conversion sites through `runeStrFromInt64` / `runeStrFromUint64`, which return `string(utf8.RuneError)` when the value doesn't fit in an int32 and otherwise defer to Go's native `string(rune)`. Both the runtime path ([`op_expressions.go:800`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/op_expressions.go#L800) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/op_expressions.go#L800)) and constant evaluation ([`preprocess.go:4893`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/preprocess.go#L4893) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/preprocess.go#L4893)) flow through `ConvertTo`, so one fix covers both.

## Fix
Before, the four 64-bit-capable string-conversion arms narrowed to int32 with `rune(...)` and lost the out-of-range signal. After, [`runeStrFromInt64`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/values_conversions.go#L19-L24) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/values_conversions.go#L19) detects out-of-range via the round-trip `v != int64(rune(v))` (catches both positive overflow and negatives below MinInt32) and [`runeStrFromUint64`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/values_conversions.go#L27-L32) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/values_conversions.go#L27) via `v > math.MaxInt32`; both return `�`, and in-int32 values fall through to Go's native conversion, which already maps negatives, surrogate halves, and values above 0x10FFFF to `�`. The load-bearing constraint: once a value fits in int32, the truncation hole is the only divergence left, so no separate `0..0x10FFFF` range check is needed.

## Verification

Differential check against the Go toolchain (`go run`) over the PR's full boundary table: out-of-range 64-bit values, the truncation-aliasing values (`0x10001F600`, `-4294967231`, `0x100000041`), in-range glyphs, in-int32 invalids (`-1`, `0xD800`, `0x110000`), and the uint32 cases — Gno output matches Go byte-for-byte. The new filetest `str_conv_overflow.gno` passes with the fix and fails on the old truncating behavior (reverting the two helper bodies to `return string(rune(v))` reproduces master's `A`/`😀`/`A` aliasing).

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5807 -R gnolang/gno
go test -run 'TestFiles/str_conv_overflow.gno$' -v ./gnovm/pkg/gnolang/
```

```
=== RUN   TestFiles/str_conv_overflow.gno
--- PASS: TestFiles (0.18s)
    --- PASS: TestFiles/str_conv_overflow.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
```

Note on CI: the only red check is "Merge Requirements" (bot waiting on a review-team approval), not a code failure. The broader `TestFiles -test.short` suite shows unrelated error-message mismatches (`switch13.gno`, `redeclaration3.gno`, `or_f0.gno`, etc.); these fail on `origin/master` alone — none of those files are touched by this PR (only 3 files changed) — so they are pre-existing base noise, not a regression here.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- None.

## Missing Tests
None. The filetest covers truncation-aliasing values, in-range values, in-int32 invalids, the uint32 path, named types (`int64`/`int`/`uint64`/`uint`), and the untyped rune constant `'A' + 0x100000000` (constant-eval path). Boundaries (surrogate halves, 0x10FFFF/0x110000, MaxInt32±1, ±2^32) are exercised.

## Suggestions
- `gnovm/pkg/gnolang/values_conversions.go:19-32` — the two helpers use different but equivalent out-of-range tests (round-trip vs `> MaxInt32`); optional to unify, current form is clear and correct.
  <details><summary>details</summary>

  [`runeStrFromInt64`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/values_conversions.go#L19-L24) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/values_conversions.go#L19) uses `v != int64(rune(v))` because a signed value can be negative, so a one-sided `> MaxInt32` would miss values below MinInt32; [`runeStrFromUint64`](https://github.com/gnolang/gno/blob/b22859722/gnovm/pkg/gnolang/values_conversions.go#L27-L32) · [↗](../../../../../.worktrees/gno-review-5807/gnovm/pkg/gnolang/values_conversions.go#L27) is non-negative so `> MaxInt32` suffices. The asymmetry is justified; not a defect. Leaving as-is is fine.
  </details>

## Open questions
- `Uint32Kind` keeps `string(rune(tv.GetUint32()))` unchanged. This is correct: for uint32, values in `(MaxInt32, MaxUint32]` reinterpret as negative runes and values in `(0x10FFFF, MaxInt32]` exceed the code point range, so every invalid uint32 already yields `�` (confirmed against Go: `string(uint32(0x80000041))` and `string(uint32(0x7FFFFFFF))` both give `�`). No change needed; noted only so a future reader doesn't mistake the asymmetry for an oversight. Not posted.
