# Stdlib Test Gap — Batch A: bytes / path / sort

> **Version baseline caveat**: this audit used Go 1.25.9 as the
> upstream baseline. Per `gno/docs/resources/go-gno-compatibility.md`,
> **Gno is modeled after Go 1.17** with selective forward cherry-picks.
> Many "missing API" entries below are simply post-1.17 APIs (e.g.
> `bytes.Clone` Go 1.20, `bytes.Lines` Go 1.24, `sort.Find` Go 1.21).
> See [`../bugs.md`](../bugs.md) for reclassification.

Findings from porting upstream Go 1.25.9 tests for the `bytes`, `path`, and
`sort` packages into Gno's `gnovm/stdlibs/*` and running them under
`TestStdlibs`.

**Baseline**: `gnolang/gno@master` **with PR [#5723](https://github.com/gnolang/gno/pull/5723) cherry-picked**.

Ports live in [`.worktrees/gno-stdlib-test-port/`](../../../.worktrees/gno-stdlib-test-port/):

- [`gnovm/stdlibs/path/example_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/path/example_test.gno)
- [`gnovm/stdlibs/sort/sort_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/sort/sort_test.gno) and [`gnovm/stdlibs/sort/example_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/sort/example_test.gno)
- [`gnovm/stdlibs/bytes/bytes_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/bytes/bytes_test.gno) (ported content appended to the existing file)

How to reproduce:

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/path$'  ./gnovm/pkg/gnolang/
go test -count=1 -v -run 'TestStdlibs/sort$'  ./gnovm/pkg/gnolang/
go test -count=1 -v -run 'TestStdlibs/bytes$' ./gnovm/pkg/gnolang/
```

All ported tests **pass** under the baseline. No correctness or DOS bugs
were surfaced. The findings below are missing-API gaps: each one is a
class of upstream tests that cannot be ported until the corresponding
public symbol is added to Gno's stdlib.

---

## bytes: Cut/CutPrefix/CutSuffix, Clone, ContainsFunc, Buffer.AvailableBuffer missing

**Package**: `gnovm/stdlibs/bytes`
**Severity**: missing API (gap, not bug)
**Existing PR**: [#5676](https://github.com/gnolang/gno/pull/5676) (OPEN) — adds exactly these symbols and their tests

### Summary

Upstream Go has the following in `bytes` that Gno does not:

- `bytes.Cut`, `bytes.CutPrefix`, `bytes.CutSuffix` (Go 1.18)
- `bytes.Clone` (Go 1.20)
- `bytes.ContainsFunc` (Go 1.21)
- `(*bytes.Buffer).AvailableBuffer` (Go 1.21)

The corresponding upstream tests — `TestCut`, `TestCutPrefix`,
`TestCutSuffix`, `TestClone`, `TestContainsFunc`, `TestWriteAppend` —
all reference functions that don't exist in Gno and therefore cannot be
ported as written.

This is **not a new finding** — PR #5676 already ports the API plus the
tests, with the same "no `unsafe`, no `testing.AllocsPerRun`" deviations
this audit uses. Once it lands, those tests become a no-op delta.

### Action

Nothing to file as a new issue. Track #5676.

---

## bytes: `Lines` and `*Seq` (iterator) family missing

**Package**: `gnovm/stdlibs/bytes`
**Severity**: missing API (gap, not bug)
**Existing PR**: none found

### Summary

Go 1.23+ adds range-over-func iterator versions to `bytes`:

- `bytes.Lines`
- `bytes.SplitSeq`, `bytes.SplitAfterSeq`
- `bytes.FieldsSeq`, `bytes.FieldsFuncSeq`

None are present in Gno. The upstream `TestLines` and the corresponding
Examples cannot be ported.

Gno does not yet have range-over-func either, so adding these requires
language support, not just a port. Out of scope of the test-porting audit;
recording only as a known gap.

---

## sort: `Slice`, `SliceStable`, `SliceIsSorted`, `Find` missing

**Package**: `gnovm/stdlibs/sort`
**Severity**: missing API (gap, not bug)
**Existing PR**: none found

### Summary

Gno's `sort` package implements the classic `sort.Interface` family
(`Sort`, `Stable`, `Reverse`, `IntSlice`, `Float64Slice`, `StringSlice`,
`Ints`, `Strings`, `Float64s`, etc.) plus `Search` and the typed search
wrappers, but is missing:

- `sort.Slice(x any, less func(i, j int) bool)` (Go 1.8)
- `sort.SliceStable(x any, less func(i, j int) bool)` (Go 1.8)
- `sort.SliceIsSorted(x any, less func(i, j int) bool)` (Go 1.8)
- `sort.Find(n int, cmp func(i int) int) (int, bool)` (Go 1.19)

The upstream tests `TestFind`, `TestFindExhaustive` and the
`ExampleSlice`, `ExampleSliceStable`, `ExampleSliceIsSorted`,
`ExampleFind` examples therefore cannot be ported. Same for the
canonical top-level `Example` which calls `sort.Slice` in its second half.

### Root cause

`sort.Slice` upstream uses `reflect.Swapper` and `reflect.ValueOf` to
sort an arbitrary slice value without requiring `sort.Interface` methods.
The Gno repository has comments in `sort_test.gno` (line 20:
`/*a XXX removed slice due to reflect methods`) acknowledging this is
intentionally absent because Gno's `reflect` does not expose the needed
hooks.

`sort.Find` is a thin wrapper around `sort.Search` and could be added
directly without reflect; the omission appears to be an oversight rather
than an architectural constraint.

### Impact

- Realm authors cannot sort a slice by closure without first writing a
  `sort.Interface` adapter. This is a significant ergonomic gap from Go,
  and a frequent source of "why doesn't this compile?" friction.
- `sort.Find` is the only way upstream Go expresses three-way-comparison
  binary search without writing the index walk by hand.

### Action

File an issue requesting at minimum `sort.Find` (zero-blocker, pure Go).
`sort.Slice*` requires either the missing reflect surface or a generics-
based reimplementation; track separately.

---

## sort: `reverseRange` not exported via `export_test.go` equivalent

**Package**: `gnovm/stdlibs/sort`
**Severity**: missing test hook (gap, not bug)
**Existing PR**: none found

### Summary

Upstream `TestReverseRange` exercises the private `reverseRange` helper
via `export_test.go`:

```go
// upstream src/sort/export_test.go
func ReverseRange(data Interface, a, b int) { reverseRange(data, a, b) }
```

Gno's `sort` package does not include the equivalent
`export_test.gno`, and the underlying `reverseRange` is also absent
from the implementation. The upstream test cannot be ported.

This is small-scope — it's a single helper used only by upstream's
internal heapsort. Recording it for completeness; no urgent action.

---

## path: no findings

**Package**: `gnovm/stdlibs/path`

All 8 upstream Examples (`ExampleBase`, `ExampleClean`, `ExampleDir`,
`ExampleExt`, `ExampleIsAbs`, `ExampleJoin`, `ExampleMatch`,
`ExampleSplit`) were ported to
[`path/example_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/path/example_test.gno)
and the package passes under `TestStdlibs/path`. No divergences.

The only upstream test not ported is `TestCleanMallocs`, which depends on
`testing.AllocsPerRun` (not in Gno's `testing`).

Note: Gno's `TestStdlibs` driver only invokes `Test*` functions; `Example*`
functions are compiled but their `// Output:` blocks are not verified.
Porting the examples still catches API-shape regressions at compile time.

---

## Tests ported that PASS (no divergence)

Negative-space coverage — places where Gno matches upstream behavior on
previously-untested code paths.

**path**: `ExampleBase`, `ExampleClean`, `ExampleDir`, `ExampleExt`,
`ExampleIsAbs`, `ExampleJoin`, `ExampleMatch`, `ExampleSplit` (compile-
only; `// Output:` not driven by `TestStdlibs`).

**sort**: `TestBreakPatterns`, plus compile-only examples `ExampleInts`,
`ExampleIntsAreSorted`, `ExampleFloat64s`, `ExampleFloat64sAreSorted`,
`ExampleReverse`, `ExampleStrings`, `ExampleSearch`,
`ExampleSearch_descendingOrder`, `ExampleSearchInts`,
`ExampleSearchFloat64s`, `ExampleSearchStrings`, `Example_sortKeys`,
`Example_sortMultiKeys`, `Example_sortWrapper`.

**bytes**: `TestCountNearPageBoundary` (with `dangerousSlice` stubbed to
a plain 4 KiB buffer, mirroring the existing
[`boundary_test.gno`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/bytes/boundary_test.gno)
convention), plus compile-only examples `ExampleBuffer_Cap`,
`ExampleBuffer_Next`, `ExampleBuffer_Read`, `ExampleBuffer_ReadByte`,
`ExampleToValidUTF8`.

All ported into the upstream-named test files under
[`gnovm/stdlibs/`](../../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/);
upstreamable verbatim as a separate PR once the missing-API PRs (#5676,
plus a hypothetical `sort.Find` PR) land.
