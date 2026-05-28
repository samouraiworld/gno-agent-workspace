# Stdlib Test Gap — Aggregated Findings

Bugs and divergences found by porting upstream Go 1.25.9 stdlib tests
into Gno's `gnovm/stdlibs/*` packages and running them under
`TestStdlibs`.

**Baseline**: `gnolang/gno@master` with PR
[#5723](https://github.com/gnolang/gno/pull/5723) cherry-picked.
Without it, allocator-overflow class bugs surface as unrecoverable host
panics that mask everything downstream.

Per-batch raw findings live under
[`findings/`](findings/) — this file is the curated aggregate.

Ports live in [`.worktrees/gno-stdlib-test-port/`](../../.worktrees/gno-stdlib-test-port/),
under each package's existing test files.

Reproduce locally (per package):

```bash
cd .worktrees/gno-stdlib-test-port
go test -count=1 -v -run 'TestStdlibs/<pkg>$' ./gnovm/pkg/gnolang/
# e.g. TestStdlibs/strings, TestStdlibs/regexp-syntax (slash → dash)
```

---

## Coverage achieved

| Package         | Ported | Pass | Fail | Notes                                 |
| --------------- | -----: | ---: | ---: | ------------------------------------- |
| `strings`       |    ~40 |   39 |    1 | TestSplit fails (finding #1)          |
| `bytes`         |   1+5E |   1+ |    0 | mostly blocked on missing API (#5676) |
| `path`          |     8E |    8 |    0 | all examples (no Test gaps remain)    |
| `sort`          |   1+14E|    1 |    0 | blocked on missing `Slice*`/`Find`    |
| `net/url`       |     30 |   30 |    0 | clean                                 |
| `math`          |     35 |   35 |    0 | clean (29 s390x-only skipped)         |
| `math/bits`     |     37 |   37 |    0 | clean (every missing test ported)     |
| `strconv`       |      0 |    0 |    0 | only Complex tests missing, no API    |
| `encoding/binary`|    7  |   7  |    0 | ~12 need reflect Gno doesn't drive    |
| `encoding/hex`  |      6 |    6 |    0 | clean                                 |
| `encoding/base64`|     6 |    6 |    0 | clean                                 |
| `encoding/csv`  |      5 |    5 |    0 | clean                                 |
| `html`          |      2 |    2 |    0 | clean                                 |
| `hash/adler32`  |      2 |    2 |    0 | package had 0 tests before this audit |
| `unicode/utf16` |      1 |    1 |    0 | 3 blocked on unexported API           |
| `regexp`        |      1 |    1 |    0 | TestUnmarshalText blocked, no Marshal*|
| `regexp/syntax` |      1 |    0 |    1 | TestString fails (finding #2)         |

> "E" = Example tests rewritten as Test funcs (Gno's harness doesn't
> execute `Example*`; see meta-finding M1).

(E = rewrites of upstream `Example*` so they actually run.)

---

# Tier 1 — Novel correctness bugs (no open PR addresses)

## #1 — `strings.SplitN(s, "", n)` mangles invalid UTF-8 bytes (asymmetric)

**Package**: `gnovm/stdlibs/strings`
**Severity**: correctness divergence + data corruption
**Existing PR**: none

When the separator is the empty string, `strings.SplitN` splits by
Unicode code point. For inputs containing invalid UTF-8 bytes, Gno
replaces each *non-last* invalid byte with the 3-byte U+FFFD
replacement encoding (`\xef\xbf\xbd`), but leaves the *last* invalid
byte unchanged. Upstream Go preserves the original bytes for every
element. `Join(Split(s, ""), "")` is no longer guaranteed to recover `s`.

### Repro

```gno
package main

import (
    "fmt"
    "strings"
)

func main() {
    s := "\xff-\xff"
    parts := strings.SplitN(s, "", -1)
    for i, p := range parts {
        fmt.Printf("[%d] len=%d bytes=% x\n", i, len(p), []byte(p))
    }
}
```

### Expected output (upstream Go 1.25.9)

```
[0] len=1 bytes=ff
[1] len=1 bytes=2d
[2] len=1 bytes=ff
```

### Actual output (Gno, current `master`)

```
[0] len=3 bytes=ef bf bd
[1] len=1 bytes=2d
[2] len=1 bytes=ff
```

### Root cause

[`gnovm/stdlibs/strings/strings.gno:20-38`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/strings/strings.gno#L20-L38) —
Gno's `explode` has an extra `if ch == utf8.RuneError { a[i] = string(utf8.RuneError) }`
branch that upstream does not. Removing it restores upstream
semantics. Last element doesn't go through the loop body, so the bug
manifests as asymmetric output.

### Impact

Affects `Split(s, "")`, `SplitN(s, "", n)`, and any code that uses
either to deconstruct binary or non-UTF-8 text. Realm code that parses
attacker-controlled bytes via `Split(s, "")` may see byte content
silently transformed.

---

## #2 — `regexp/syntax.(*Regexp).String()` does not merge adjacent same-flag subexpressions

**Package**: `gnovm/stdlibs/regexp/syntax`
**Severity**: correctness divergence (`Parse` → `String` round-trip ≠ upstream Go)
**Existing PR**: none

When a parsed regexp contains adjacent subexpressions that share the
same `FoldCase` flag — or when a single `(?i:...)` group expands into
multiple `OpLiteral` children — upstream Go's `String()` factors the
common flag out into a single `(?i:...)` wrapper. Gno's `String()`
re-emits the flag on every literal/concat element. Result is
*functionally* equivalent (same matching behavior) but lexically
divergent.

### Repro

```gno
package main

import (
    "fmt"
    "regexp/syntax"
)

func main() {
    for _, src := range []string{
        `x(?i:ab*c|d?e)1`,
        `[Aa][Bb]*[Cc]`,
        `(?i:ab)[123](?i:cd)`,
    } {
        re, _ := syntax.Parse(src, syntax.Perl)
        fmt.Printf("Parse(%q).String() = %q\n", src, re.String())
    }
}
```

### Expected output (upstream Go 1.25.9)

```
Parse("x(?i:ab*c|d?e)1").String()       = "x(?i:AB*C|D?E)1"
Parse("[Aa][Bb]*[Cc]").String()         = "(?i:AB*C)"
Parse("(?i:ab)[123](?i:cd)").String()   = "(?i:AB[1-3]CD)"
```

### Actual output (Gno)

```
Parse("x(?i:ab*c|d?e)1").String()       = "x(?:(?i:A)(?i:B)*(?i:C)|(?i:D)?(?i:E))1"
Parse("[Aa][Bb]*[Cc]").String()         = "(?i:A)(?i:B)*(?i:C)"
Parse("(?i:ab)[123](?i:cd)").String()   = "(?i:AB)[1-3](?i:CD)"
```

All 15 cases in upstream `stringTests` fail in
[regexp/syntax/parse_test.gno](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/regexp/syntax/parse_test.gno).

### Root cause

`writeRegexp` in
[`gnovm/stdlibs/regexp/syntax/regexp.gno`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/regexp/syntax/regexp.gno)
emits the `(?i:` / `)` wrappers on every `OpLiteral` whenever
`re.Flags & FoldCase != 0`; the `OpConcat` / `OpAlternate` cases simply
iterate children without considering shared flags.

Upstream rewrote this in 2022 (CL 444817, Go 1.20+) to take a
`printFlags` map computed in a pre-pass: each subtree publishes the
flags it would emit, then `writeRegexp` factors any flag shared by all
children up to their parent. Gno's copy predates that refactor.

### Impact

- Any tool/test that serializes a parsed regexp and compares to a
  golden string diverges from upstream Go.
- Cannot affect matching behavior at runtime.
- Trivially fixable by porting the upstream `printFlags`/`writeRegexp`
  pair verbatim — both are pure Gno-able code.

---

# Tier 2 — Incomplete fixes / partial PRs

## #3 — Allocator-overflow paths NOT covered by PR #5723 still produce unrecoverable host panics

**Package**: `gnovm/pkg/gnolang/alloc.go`
**Severity**: DOS — host panic not catchable from `.gno` code
**Existing PR**: [#5723](https://github.com/gnolang/gno/pull/5723) (OPEN) — *partial fix only*

[PR #5723](https://github.com/gnolang/gno/pull/5723) converts
`overflow.Addp`/`Mulp` host panics into recoverable `*Exception` panics
with message `runtime error: makeslice: len out of range`. It does so
for **two** methods only: `AllocateDataArray` and `AllocateListArray`.

The same `overflow.Addp`/`Mulp` calls remain in four other allocator
methods, so any large `.gno` allocation that flows through one of them
still triggers a host-side `panic: addition overflow` that `recover()`
in `.gno` cannot catch.

### Unfixed call sites in [`alloc.go`](../../.worktrees/gno-stdlib-test-port/gnovm/pkg/gnolang/alloc.go)

| Line | Method                                | Triggered by                                       |
| ---- | ------------------------------------- | -------------------------------------------------- |
| 304  | `Allocate` (guard against `maxBytes`) | every allocation                                   |
| 337  | `AllocateString`                      | giant strings (`string(make([]byte, N))`, `Repeat`)|
| 374  | `AllocateStruct`                      | structs with many fields                           |
| 382  | `AllocateMap`                         | `make(map[K]V, N)` with huge hint                  |
| 398  | `AllocateBlock`                       | scope blocks with many bindings                    |
| 402  | `AllocateBlockItem`                   | block enlargement                                  |

### Repro (`AllocateString`)

```gno
package main

func main() {
    defer func() {
        if r := recover(); r != nil {
            println("recovered:", r)
            return
        }
        println("no panic")
    }()
    _ = string(make([]byte, int(^uint(0)>>1))) // maxInt-byte string
}
```

### Expected (upstream semantics, matches #5723's pattern for the two fixed methods)

```
recovered: runtime error: makeslice: len out of range
```

### Actual (Gno, current `master`, even with #5723 applied)

```
panic: addition overflow
github.com/gnolang/gno/tm2/pkg/overflow.Addp(...)
    tm2/pkg/overflow/overflow.go:53
github.com/gnolang/gno/gnovm/pkg/gnolang.(*Allocator).AllocateString(...)
    gnovm/pkg/gnolang/alloc.go:337
```

### Fix sketch

Replicate the `overflow.Add`/`Mul` + `Exception` pattern that #5723
uses on the remaining six call sites in the table above.

### Impact

After #5723 merges:

- `string(make([]byte, N))` or `string(rune)`-built giant strings → line 337
- `make(map[K]V, N)` with huge hint → line 382
- Heavy function calls (many locals in a block) → line 398

are still attacker-controllable VM-crash vectors.

`strings.Repeat(maxInt)` from earlier in the audit IS fixed by #5723
(flows through `AllocateDataArray`). This finding is the *delta* — the
paths #5723 doesn't touch.

---

# Tier 3 — Already addressed by open PRs (informational)

## I1 — `errors`: `Is`/`As`/`Unwrap`/`Join` missing

**Existing PR**: [#5385](https://github.com/gnolang/gno/pull/5385) (OPEN, author of this audit)

11 upstream tests (`TestIs`, `TestAs`, `TestUnwrap`, `TestJoin`, etc.)
cannot be ported because the API doesn't exist. PR #5385 adds it.
Once merged, those tests become portable in a follow-up.

## I2 — `bytes`: `Cut`/`CutPrefix`/`CutSuffix`, `Clone`, `ContainsFunc`, `AvailableBuffer` missing

**Existing PR**: [#5676](https://github.com/gnolang/gno/pull/5676) (OPEN)

A batch of upstream test ports for `bytes` is blocked on these
helpers. PR #5676 adds them end-to-end.

## I3 — `Example*` functions never executed by `TestStdlibs`

**Existing PR**: [#5188](https://github.com/gnolang/gno/pull/5188) (OPEN)

`gnovm/pkg/test/test.go:693` `loadTestFuncs` discovers only `Test*`
prefixed funcs. Every `example_test.gno` under `gnovm/stdlibs/` parses
and type-checks but is never run, so `// Output:` blocks are never
verified against actual output. This audit worked around it by
rewriting every ported `Example*` as `TestExample_*`. PR #5188 fixes
the harness.

---

# Tier 4 — Missing API / coverage gaps, no open PR

These are not bugs in existing Gno code; they're absent features whose
upstream tests can't be ported. Worth filing as separate enhancements.

## E1 — `sort.Find` not exposed (likely oversight)

`sort.Slice`/`SliceStable`/`SliceIsSorted` are intentionally omitted —
Gno's `sort/sort.gno` carries a comment explaining the `reflect`
constraint. But `sort.Find` (binary search by predicate, no `reflect`
needed) appears to be an unintentional omission. A one-line port adds
upstream `TestFind`/`TestFindExhaustive`.

## E2 — `bytes.Lines`, `bytes.SplitSeq`/`SplitAfterSeq`/`FieldsSeq`/`FieldsFuncSeq` missing

Needs the language-level range-over-func feature (Go 1.23+). Tracking
upstream Gno work on iter.

## E3 — `net/url.ParseQuery` has no `defaultMaxParams` DOS guard

**Severity**: defence-in-depth

Upstream Go 1.25 introduced a guard against query strings with too
many `&`-separated params, configurable via
`GODEBUG=urlmaxqueryparams=N`, default `defaultMaxParams = 10000`.
Gno's [`gnovm/stdlibs/net/url/url.gno:925`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/net/url/url.gno#L925)
has no such check. A realm exposing a path that calls
`url.ParseQuery(attackerString)` on unbounded input has no built-in
upper bound (only allocator + gas).

## E4 — `unicode/utf16`: `MaxRune`, `ReplacementChar`, `Surr1`/`Surr2`/`Surr3`, `RuneLen` unexported

Upstream exports these constants and function; Gno keeps them
lowercase package-private. `TestConstants` and `TestRuneLen` can't be
ported without an API change.

## E5 — `regexp.*Regexp`: no `MarshalText`/`UnmarshalText`/`AppendText`

`TestUnmarshalText` can't be ported.

## E6 — `crypto/sha256`: no `MarshalBinary`/`UnmarshalBinary` on digest

Entire hash-state marshaling surface is missing from Gno's `crypto/*`
hashes other than the one just added by this audit's adler32 port.
`hash.Example_binaryMarshaler` can't be ported.

## E7 — `hash/adler32`: no `AppendBinary` method, no `encoding.BinaryAppender` interface

The `AppendBinary` subtest inside `TestGoldenMarshal` was skipped. The
rest of TestGoldenMarshal (MarshalBinary / UnmarshalBinary) is now
ported and passes — this audit added the first tests this package
ever had in Gno.

## E8 — `encoding/binary.Read`/`Write`/`Encode`/`Decode`/`Append`/`Size` blocked on `reflect` features

About a dozen upstream tests can't be ported. Not a behavioral bug in
Gno's `encoding/binary`, just a Gno `reflect` capability gap.

---

# Tier 5 — Doc/code drift (low priority)

## D1 — `strconv` docstring advertises `ParseComplex`/`FormatComplex` that don't exist

[`gnovm/stdlibs/strconv/doc.gno:58`](../../.worktrees/gno-stdlib-test-port/gnovm/stdlibs/strconv/doc.gno#L58)
still mentions these functions in the package docstring; they were
never implemented. Either implement (low priority — `complex64`/`128`
support unclear in Gno) or trim the docstring.

---

# Tier 6 — Meta-observation

## M1 — `Example*` tests in stdlib are dead code (covered by I3 / #5188)

`gnovm/pkg/test/test.go:693` `loadTestFuncs` only matches funcs whose
name starts with `Test`. Every `example_test.gno` file across Gno's
stdlib (lots of them) is currently *parsed but never executed*. PR
#5188 addresses this — see I3.

---

# Negative space

Packages where all portable upstream tests passed verbatim — strong
signal that Gno's port tracks upstream behavior, at least for what
this audit could exercise: `net/url`, `math`, `math/bits`,
`encoding/hex`, `encoding/base64`, `encoding/csv`, `html`, the
non-`String()` parts of `regexp/syntax`, the non-`Marshal*` parts of
`regexp`, the portable bits of `bytes` and `path`. Worth highlighting
since the audit was specifically looking for divergences.

---

# Suggested next step

The ported test files themselves are valuable independent of the
findings — they're net-new test coverage that match upstream's
behavior. Prepare a per-package upstream PR that contributes these to
`gnolang/gno`'s `gnovm/stdlibs/*`, separately from the bug-fix PRs for
#1, #2, and #3.
