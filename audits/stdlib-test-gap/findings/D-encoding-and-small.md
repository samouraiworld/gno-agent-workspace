# Stdlib Test Gap — Findings (Wave D: encoding + small packages)

> **Version baseline caveat**: this audit used Go 1.25.9 as the
> upstream baseline. Per `gno/docs/resources/go-gno-compatibility.md`,
> **Gno is modeled after Go 1.17** with selective forward cherry-picks.
> The `regexp/syntax.String()` finding below matches Go 1.17 exactly —
> upstream rewrote `writeRegexp` in Go 1.20+ (CL 444817). It's a
> port-forward request, not a bug in Gno's port.
> See [`../bugs.md`](../bugs.md) for reclassification.

Bugs and divergences uncovered by porting Go 1.25.9 stdlib tests into the
matching Gno packages and running them via `TestStdlibs`.

**Baseline**: `gnolang/gno@master` with PR
[#5723](https://github.com/gnolang/gno/pull/5723) cherry-picked.

Ports live in the worktree at
[`.worktrees/gno-stdlib-test-port/`](../../../.worktrees/gno-stdlib-test-port/),
each in the upstream-named test file inside the relevant
`gnovm/stdlibs/<pkg>/` directory (e.g. `bits_test.gno`, `binary_test.gno`,
`example_test.gno` for the Example-only ports, `adler32_test.gno` for
`hash/adler32`).

How to reproduce all findings locally:

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/regexp-syntax$' ./gnovm/pkg/gnolang/
```

---

## #1 — `regexp/syntax: (*Regexp).String()` does not merge adjacent same-flag subexpressions

**Package**: `gnovm/stdlibs/regexp/syntax`
**Severity**: correctness divergence (round-trip via `Parse` → `String` ≠ upstream Go)
**Existing PR**: none found
**Found by**: port of upstream `TestString` in
[parse_test.gno](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/regexp/syntax/parse_test.gno)

### Summary

When a parsed regexp contains adjacent subexpressions that share the same
`FoldCase` (case-insensitive) flag — or when a single `(?i:...)` group
expands into multiple `OpLiteral` children that all carry `FoldCase` —
upstream Go's `(*Regexp).String()` factors the common flag out into a
single `(?i:...)` wrapper around the merged run. Gno's `String()`
re-emits the flag on every literal/concat element, producing strings that
are functionally equivalent but lexically diverge from upstream.

### Repro

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/regexp-syntax$' ./gnovm/pkg/gnolang/
```

### Expected (upstream Go 1.25.9)

```
Parse(`x(?i:ab*c|d?e)1`).String() == `x(?i:AB*C|D?E)1`
Parse(`[Aa][Bb]*[Cc]`).String()   == `(?i:AB*C)`
Parse(`(?i:ab)[123](?i:cd)`).String() == `(?i:AB[1-3]CD)`
Parse(`A(?:[Bb][Cc]|[Dd])[Zz]`).String() == `A(?i:(?:BC|D)Z)`
```

### Actual (Gno)

```
Parse(`x(?i:ab*c|d?e)1`).String() == `x(?:(?i:A)(?i:B)*(?i:C)|(?i:D)?(?i:E))1`
Parse(`[Aa][Bb]*[Cc]`).String()   == `(?i:A)(?i:B)*(?i:C)`
Parse(`(?i:ab)[123](?i:cd)`).String() == `(?i:AB)[1-3](?i:CD)`
Parse(`A(?:[Bb][Cc]|[Dd])[Zz]`).String() == `A(?:(?i:BC)|(?i:D))(?i:Z)`
```

All 15 cases in upstream `stringTests` fail; full output in the
`TestString` log.

### Root cause

`writeRegexp` in
[`gnovm/stdlibs/regexp/syntax/regexp.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/regexp/syntax/regexp.gno)
emits the `(?i:` / `)` wrappers on the `OpLiteral` case unconditionally
whenever `re.Flags & FoldCase != 0`, and the `OpConcat` / `OpAlternate`
cases simply iterate their children without considering shared flags.

Upstream rewrote this in 2022 (CL 444817, Go 1.20+) to take a
`printFlags` map computed in a pre-pass: each subtree publishes the
flags it would emit, then `writeRegexp` factors any flag shared by all
children up to their parent. Gno's copy predates that refactor.
Reference: `/usr/lib/go/src/regexp/syntax/regexp.go` upstream.

### Impact

- Any tool/test that serializes a parsed regexp and compares to a
  golden string diverges from upstream Go.
- `Regexp.Equal`-style structural comparisons are not affected (the
  parsed tree itself is identical); the issue is purely cosmetic at the
  String() boundary.
- Cannot affect matching behavior at runtime.

### Existing PR

None found on `gnolang/gno`. Searches tried:
`regexp syntax String`, `writeRegexp printFlags`, `regexp String FoldCase`.

---

# Per-package summary

| Package          | Ported | Pass | Fail | Skip (unportable) | Skip (missing API) |
|------------------|-------:|-----:|-----:|------------------:|-------------------:|
| `encoding/binary`|     7  |   7  |   0  |                12 |                  0 |
| `encoding/hex`   |     6  |   6  |   0  |                 0 |                  0 |
| `encoding/base64`|     6  |   6  |   0  |                 0 |                  0 |
| `encoding/csv`   |     5  |   5  |   0  |                 0 |                  0 |
| `html`           |     2  |   2  |   0  |                 0 |                  0 |
| `hash/adler32`   |     2  |   2  |   0  |                 0 |                  1 |
| `hash`           |     0  |   0  |   0  |                 0 |                  1 |
| `unicode/utf8`   |     0  |   0  |   0  |                 1 |                  0 |
| `unicode/utf16`  |     1  |   1  |   0  |                 0 |                  3 |
| `regexp`         |     1  |   1  |   0  |                 1 |                  1 |
| `regexp/syntax`  |     1  |   0  |   1  |                 0 |                  0 |

### Skipped — unportable (Go feature gap)

- `encoding/binary`: `ExampleRead`, `ExampleRead_multi`, `ExampleWrite`,
  `ExampleWrite_multi`, plus tests
  `TestLittleEndianRead`, `TestLittleEndianWrite`,
  `TestLittleEndianPtrWrite`, `TestBigEndianRead`, `TestBigEndianWrite`,
  `TestBigEndianPtrWrite`, `TestReadSlice`, `TestWriteSlice`,
  `TestReadBool`, `TestReadBoolSlice`, `TestSliceRoundTrip`, `TestWriteT`,
  `TestBlankFields`, `TestSizeStructCache`, `TestSizeInvalid`,
  `TestUnexportedRead`, `TestReadErrorMsg`, `TestReadTruncated`,
  `TestReadInvalidDestination`, `TestNoFixedSize`, `TestAppendAllocs`,
  `TestSizeAllocs`, `TestNativeEndian` — all rely on `binary.Read` /
  `Write` / `Encode` / `Decode` / `Append` / `Size`, which use
  `reflect` features that Gno's `reflect` does not yet drive (and
  `TestNativeEndian` needs `unsafe`).
- `unicode/utf8`: `TestRuneCountNonASCIIAllocation` — uses
  `testing.AllocsPerRun`, not implemented in Gno's `testing`.
- `regexp`: `TestRE2Exhaustive` — upstream skips on `-test.short`.

### Skipped — missing API

- `hash/adler32`: `AppendBinary` subtest inside `TestGoldenMarshal` —
  Gno's `encoding` package has no `BinaryAppender` and Gno's
  `adler32.digest` has no `AppendBinary` method. The rest of
  `TestGoldenMarshal` (MarshalBinary / UnmarshalBinary) is ported and
  passes.
- `hash`: `Example_binaryMarshaler` — uses `crypto/sha256`, but Gno's
  `crypto/sha256` does not implement `encoding.BinaryMarshaler` /
  `BinaryUnmarshaler` (no MarshalBinary on its digest type).
- `unicode/utf16`: `TestConstants` (no exported `MaxRune` /
  `ReplacementChar`) and `TestRuneLen` (no `RuneLen` function). Gno's
  `utf16` package keeps these as lowercase package-private symbols.
- `regexp`: `TestUnmarshalText` — Gno's `regexp` has no `MarshalText`,
  `UnmarshalText`, or `AppendText` methods on `*Regexp`.

### Notable

- `hash/adler32` shipped with zero tests in Gno. The port adds
  `TestGolden` (37 fixed-string + huge-repeat cases) and
  `TestGoldenMarshal` (round-trip via `MarshalBinary` /
  `UnmarshalBinary`). Both pass — adler32 itself is fine; it just had
  no upstream coverage in Gno until now.
- `encoding/binary` still has wide API gaps: roughly half of upstream's
  test surface is unreachable from Gno because `reflect` cannot yet
  drive `binary.Read` / `Write` / `Encode` / `Decode` / `Append` /
  `Size`. Worth tracking as a single follow-up against
  `gnovm/stdlibs/encoding/binary/binary.gno`, not as separate findings.
- The new `binary.TestByteOrder` exercises every `Put*` / `Append*` /
  `*` round-trip across both endianness implementations for offsets
  that span and miss alignment boundaries; all pass.
- The new `binary.TestEarlyBoundsChecks` confirms Gno's `Uint64` /
  `PutUint64` panic on short slices (as upstream does).
