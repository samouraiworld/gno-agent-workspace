# [TITLE] regexp/syntax: (*Regexp).String() does not merge adjacent same-flag subexpressions

## Description

When a parsed regexp contains adjacent subexpressions that share the same
`FoldCase` flag — or when a single `(?i:…)` group expands into multiple
`OpLiteral` children that all carry `FoldCase` — upstream Go's
`(*Regexp).String()` factors the common flag out into a single `(?i:…)`
wrapper around the merged run. Gno's `String()` re-emits the flag on every
literal / concat element.

The two strings are functionally equivalent (same matching behavior at
runtime), but they diverge lexically. Any tool, test, or downstream that
compares a serialized regexp to a golden upstream string will fail.

## Reproducer

```gno
package main

import (
	"fmt"
	"regexp/syntax"
)

func main() {
	srcs := []string{
		`x(?i:ab*c|d?e)1`,
		`[Aa][Bb]*[Cc]`,
		`(?i:ab)[123](?i:cd)`,
		`A(?:[Bb][Cc]|[Dd])[Zz]`,
	}
	for _, src := range srcs {
		re, _ := syntax.Parse(src, syntax.Perl)
		fmt.Printf("Parse(%q).String() = %q\n", src, re.String())
	}
}
```

### Expected output (upstream Go 1.25.9)

```
Parse("x(?i:ab*c|d?e)1").String()         = "x(?i:AB*C|D?E)1"
Parse("[Aa][Bb]*[Cc]").String()           = "(?i:AB*C)"
Parse("(?i:ab)[123](?i:cd)").String()     = "(?i:AB[1-3]CD)"
Parse("A(?:[Bb][Cc]|[Dd])[Zz]").String()  = "A(?i:(?:BC|D)Z)"
```

### Actual output (Gno, current `master`)

```
Parse("x(?i:ab*c|d?e)1").String()         = "x(?:(?i:A)(?i:B)*(?i:C)|(?i:D)?(?i:E))1"
Parse("[Aa][Bb]*[Cc]").String()           = "(?i:A)(?i:B)*(?i:C)"
Parse("(?i:ab)[123](?i:cd)").String()     = "(?i:AB)[1-3](?i:CD)"
Parse("A(?:[Bb][Cc]|[Dd])[Zz]").String()  = "A(?:(?i:BC)|(?i:D))(?i:Z)"
```

All 15 cases in upstream's `stringTests` table fail.

## Root cause

`writeRegexp` in `gnovm/stdlibs/regexp/syntax/regexp.gno` emits the `(?i:`
/ `)` wrappers on every `OpLiteral` whenever
`re.Flags & FoldCase != 0`, and the `OpConcat` / `OpAlternate` cases
simply iterate their children without considering shared flags.

Upstream Go rewrote this in 2022 ([CL 444817](https://go-review.googlesource.com/c/go/+/444817), released in Go 1.20)
to take a `printFlags` map computed in a pre-pass: each subtree publishes
the flags it would emit, then `writeRegexp` factors any flag shared by all
children up to their parent. Gno's copy predates that refactor.

## Suggested fix

Port the upstream `printFlags` map + the updated `writeRegexp` from
`src/regexp/syntax/regexp.go @ go1.25.9`. Both are pure Gno-able code (no
language features Gno lacks).

## Test coverage

An upstream-derived `TestString` ported into the worktree is currently
failing against `master` — see the audit's worktree at
`gnovm/stdlibs/regexp/syntax/parse_test.gno`.

## Impact

- Round-tripping (`syntax.Parse(s).String()`) produces visually different,
  noisier output than upstream Go.
- Cannot affect matching behavior at runtime — the parsed tree itself is
  identical; the divergence is purely in `String()`.
- Cosmetic but a real correctness divergence at the API boundary.

## How this was found

Discovered by porting the upstream `TestString` from
`src/regexp/syntax/parse_test.go @ go1.25.9` into
`gnovm/stdlibs/regexp/syntax/` as part of a broader stdlib test-gap audit.
