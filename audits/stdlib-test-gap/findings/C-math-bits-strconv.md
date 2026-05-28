# Stdlib Test Gap — Findings (math, math/bits, strconv)

Bugs found by porting upstream Go 1.25.9 tests into Gno's
`gnovm/stdlibs/math`, `gnovm/stdlibs/math/bits`, and
`gnovm/stdlibs/strconv` and running them under `TestStdlibs`.

**Baseline**: `gnolang/gno@master` with PR
[#5723](https://github.com/gnolang/gno/pull/5723) cherry-picked.

Ports live in the worktree at
[`.worktrees/gno-stdlib-test-port/`](../../.worktrees/gno-stdlib-test-port/):

- [`gnovm/stdlibs/math/all_test.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/math/all_test.gno) (ported content appended to the existing file)
- [`gnovm/stdlibs/math/bits/bits_test.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/math/bits/bits_test.gno) (ported content appended to the existing file)

How to reproduce locally:

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/math$'       ./gnovm/pkg/gnolang/
go test -count=1 -v -run 'TestStdlibs/math-bits$'  ./gnovm/pkg/gnolang/
go test -count=1 -v -run 'TestStdlibs/strconv$'    ./gnovm/pkg/gnolang/
```

---

## Result: no behavioural divergences found

| Package     | Upstream-missing | Ported | Pass | Fail | Skipped (unportable) | Skipped (missing API) |
| ----------- | ---------------: | -----: | ---: | ---: | -------------------: | --------------------: |
| `math`      |               64 |     35 |   35 |    0 |                   29 |                     0 |
| `math/bits` |               37 |     37 |   37 |    0 |                    0 |                     0 |
| `strconv`   |                5 |      0 |    0 |    0 |                    0 |                     5 |

All ported tests pass cleanly against `master + #5723`. No divergence
from upstream Go 1.25.9 surfaced in any of the three packages.

### `math` — what was ported

- 31 of the 32 upstream Examples (`ExampleAcos`, `ExampleCos`,
  `ExampleSin`, `ExampleHugeSinCos`, ..., `ExampleModf`), rewritten as
  `TestExample*` functions that assert `fmt.Sprintf` output matches the
  upstream `// Output:` block. (See "Note on Examples" below.)
- All 4 huge-argument trig tests from upstream `huge_test.go` —
  `TestHugeCos`, `TestHugeSin`, `TestHugeSinCos`, `TestHugeTan`. These
  exercise Gno's trig argument-reduction path on values up to
  `MaxFloat64` and compare against reference values computed at
  4096 bits of precision (upstream uses ivy). Tolerance: `1e-14`.
  Failure messages include `math.Float64bits` of both sides so any
  ULP-level drift would surface as `Expected: 0x... Actual: 0x...`.

### `math` — what was skipped and why

The 17 `Test*Novec` tests in upstream `arith_s390x_test.go` are
s390x-only — they call `HasVX` / `CosNoVec` / `SinNoVec` / `TanNoVec`
/ ... none of which exist in Gno (or anywhere off s390x). Likewise
the 12 `TestLarge*Novec` cases. Together: **29 tests skipped as
missing-API**.

### `math/bits` — what was ported

- All 36 upstream Examples from `example_test.go` and
  `example_math_test.go` (LeadingZeros/TrailingZeros/OnesCount/
  RotateLeft/Reverse/ReverseBytes/Len/Add/Sub/Mul/Div, sizes 8/16/32/64),
  rewritten as `TestExample*`.
- `TestUintSize`, adapted: upstream uses `unsafe.Sizeof(uint)*8`;
  Gno is always 64-bit, so the ported test asserts `UintSize == 64`
  directly.

### `strconv` — all 5 missing tests skipped (missing API)

`grep '^func ParseComplex\|^func FormatComplex'
gnovm/stdlibs/strconv/*.gno` returns nothing — Gno's `strconv` does
not implement complex-number parsing/formatting. The 5 upstream tests
(`TestParseComplex`, `TestParseComplexIncorrectBitSize`,
`TestFormatComplex`, `TestFormatComplexInvalidBitSize`, and the
malloc-counting `TestCountMallocs`, which also touches `ParseComplex`)
are all skipped on this basis.

Filing an `API gap` follow-up rather than a bug: Gno's `strconv/doc.gno`
explicitly mentions "FormatComplex and ParseComplex" in its package
docstring while neither function is defined, so the documented surface
diverges from the implemented surface. Low priority — no chain code
calls these.

---

## Note on Examples

Gno's `TestStdlibs` harness only enumerates `Test*` functions
(see [`gnovm/pkg/test/test.go:693`](../../.worktrees/gno-stdlib-test-port/gnovm/pkg/test/test.go#L693)):

```go
if strings.HasPrefix(fname, "Test") {
    ...
}
```

`Example*` functions are still parsed and type-checked, but never
executed. To actually exercise the Example bodies and verify their
`// Output:` blocks, each upstream `ExampleX` was rewritten as
`TestExampleX` that:

1. Calls the same expression(s) the Example would have called.
2. Captures `fmt.Sprintf` output instead of writing to stdout.
3. Compares against the literal expected-output string.

So `func ExampleAcos() { fmt.Printf("%.2f", math.Acos(1)) // Output: 0.00 }`
becomes
`func TestExampleAcos(t *testing.T) { checkExample(t, "Acos", fmt.Sprintf("%.2f", math.Acos(1)), "0.00") }`.

This is the only adaptation that diverges from a strict
upstream-cherry-pick.

---

## Notable

- **No `math/big` dependency in any missing math test.** The upstream
  missing-test set is dominated by Examples and the s390x `*Novec`
  variants, neither of which touches `math/big`.
- **No complex-number dependency in math.** Missing complex-number
  coverage is contained entirely in `strconv` (`Parse/FormatComplex`).
- **Float-eps**: the four huge-trig tests pass at the upstream `1e-14`
  tolerance without adjustment — Gno's `Cos`/`Sin`/`Tan` reduction
  preserves enough precision at `MaxFloat64` to match upstream.
- **`math/bits` is 100% portable**: every missing test is either a
  pure-integer Example or `TestUintSize`. Gno's `UintSize = 64`
  invariant makes the 32-bit branches in upstream tests dead code,
  which is fine.
- **No matching open PRs** for any of the test areas covered
  (`gh pr list --search 'math/bits'`, `'stdlib math trig'`,
  `'ParseComplex FormatComplex'` all return `[]`).
