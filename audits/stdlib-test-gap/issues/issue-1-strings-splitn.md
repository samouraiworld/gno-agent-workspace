# [TITLE] strings.SplitN(s, "") asymmetrically mangles invalid UTF-8 bytes (port forward from Go 1.20+)

## Description

> **Snapshot context**: Gno is modeled after Go 1.17. The behavior
> described below matches Go 1.17 exactly — the `if ch == utf8.RuneError`
> branch is present in `release-branch.go1.17/src/strings/strings.go`.
> Upstream Go removed it between 1.18 and 1.20. This is therefore a
> port-forward request to bring Gno in line with modern Go, **not a
> bug introduced by Gno's port**.

When the separator is the empty string, `strings.SplitN` splits by Unicode
code point. For inputs containing invalid UTF-8 bytes, Gno's implementation
(matching Go 1.17) silently replaces each *non-last* invalid byte with the
3-byte U+FFFD replacement encoding (`\xef\xbf\xbd`), but leaves the *last*
invalid byte unchanged. Modern upstream Go preserves the original bytes
for every element.

Two problems with carrying the Go 1.17 behavior forward:

1. **Divergence from modern Go.** Realm code shared between Go and Gno
   gets different results: `strings.Split("\xff-\xff", "")` returns three
   1-byte strings on modern Go but `["\xef\xbf\xbd", "-", "\xff"]` on Gno.
2. **Internal inconsistency.** The first and last invalid bytes are treated
   differently within the same call.

As a consequence, `strings.Join(strings.Split(s, ""), "")` is no longer
guaranteed to equal `s` when `s` contains invalid UTF-8.

## Reproducer

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

## Root cause

`explode` in `gnovm/stdlibs/strings/strings.gno` (lines 20-38) has an
extra branch that upstream does **not**:

```go
for i := 0; i < n-1; i++ {
    ch, size := utf8.DecodeRuneInString(s)
    a[i] = s[:size]
    s = s[size:]
    if ch == utf8.RuneError {
        a[i] = string(utf8.RuneError)  // <-- not in upstream
    }
}
if n > 0 {
    a[n-1] = s  // last element keeps raw bytes (this matches upstream)
}
```

Upstream `src/strings/strings.go` at `go1.25.9` (lines 23-38):

```go
for i := 0; i < n-1; i++ {
    _, size := utf8.DecodeRuneInString(s)
    a[i] = s[:size]
    s = s[size:]
}
if n > 0 {
    a[n-1] = s
}
```

## Suggested fix

Delete the `if ch == utf8.RuneError { a[i] = string(utf8.RuneError) }` block
and the now-unused `ch` capture (rename to `_`). This restores upstream
semantics and removes the last-element asymmetry.

## Test coverage

An upstream-derived `TestSplit` ported into the worktree is currently
failing against `master` and isolates the bug — see the audit's worktree
at `gnovm/stdlibs/strings/strings_test.gno`.

## Impact

Affects `strings.Split(s, "")`, `strings.SplitN(s, "", n)`, and any code
that uses either to deconstruct binary or non-UTF-8 text. Realm code that
parses attacker-controlled bytes via `Split(s, "")` may see byte content
silently transformed without warning.

## How this was found

Discovered by porting the upstream `TestSplit` from `src/strings/strings_test.go @ go1.25.9`
into `gnovm/stdlibs/strings/` as part of a broader stdlib test-gap audit.
