# PR #5780: fix(gnovm): avoid panic on assertion over nil slice

URL: https://github.com/gnolang/gno/pull/5780
Author: Villaquiranm | Base: master | Files: 2 | +27 -2
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `4b00b2b44` (stale)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5780 4b00b2b44`

**Verdict: APPROVE** — minimal, correct fix; nil-slice guard added in the one spot that did an unchecked type assertion, regression test reproduces the exact panic on master and passes with the fix. No blocking concerns.

## Summary
`string(s)` where `s` is a nil slice of a named rune type (`var z []MyRune; string(z)`) panicked with `interface conversion: gnolang.Value is nil, not *gnolang.SliceValue`. The conversion logic in `ConvertTo` already handled the nil case (it emits `""`), but the per-N gas charge in `doOpConvert` ran first and did `xv.V.(*SliceValue)` unconditionally, blowing up on the nil `V`. Fix wraps that assertion in an `if xv.V != nil` guard.

## Glossary
- `doOpConvert` — VM opcode handler for type conversions; charges CPU gas then calls `ConvertTo`.
- `ConvertTo` — does the actual value conversion; `values_conversions.go`.
- `OpCPUSlopeConvertRunesStr` — per-element gas slope for `[]rune → string`.

## Fix
Before: the `[]rune → string` gas branch asserted `xv.V.(*SliceValue)` to read the length, which panics when `xv.V` is nil (a declared-but-unallocated slice). After: the assertion and `incrCPU` run only when `xv.V != nil`. Charge is unchanged for every non-nil case, and the skipped case would have charged `slope * 0 == 0` anyway, so gas accounting is identical. See [`op_expressions.go:789-794`](https://github.com/gnolang/gno/blob/4b00b2b44/gnovm/pkg/gnolang/op_expressions.go#L789-L794) · [↗](../../../../../.worktrees/gno-review-5780/gnovm/pkg/gnolang/op_expressions.go#L789-L794).

## Verification
Reproduced the panic on master and confirmed the fix + test:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5780 -R gnolang/gno
# 1. test passes with the fix
go test ./gnovm/cmd/gno/ -run 'Test_Scripts/test/issue_5776' -v 2>&1 | tail -3
# 2. revert only the source fix, keep the test -> reproduces the reported panic
git checkout origin/master -- gnovm/pkg/gnolang/op_expressions.go
go test ./gnovm/cmd/gno/ -run 'Test_Scripts/test/issue_5776' -v 2>&1 | grep -iE "panic|FAIL"
git checkout HEAD -- gnovm/pkg/gnolang/op_expressions.go
```

```
--- PASS: Test_Scripts/test/issue_5776 (0.01s)
# after reverting the fix:
panic running expression main(): interface conversion: gnolang.Value is nil, not *gnolang.SliceValue
FAIL: .../issue_5776.txtar:1: unexpected gno command outcome (err=false expected=true)
```

`go test ./gno.land/pkg/sdk/vm/ -run Gas` and the gnolang conversion tests both pass — no gas/determinism regression.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- The named byte case from the same issue (`var y []MyByte; string(y)`) never reached this branch — there is no per-N gas charge for `[]byte → string` in `doOpConvert`, only for `[]rune → string` — so only the rune leg actually panicked. The test correctly covers both legs regardless; no change needed.

## Missing Tests
None. The txtar covers both the byte and rune legs, and the regression is pinned by a test that fails on master.

## Suggestions
- `gnovm/cmd/gno/testdata/test/issue_5776.txtar` only runs `gno run .` with no stdout assertion, so it asserts "does not panic" but not the produced value. Since both conversions yield `""` and `main` discards them, that is acceptable; an `// Output:` filetest under `gnovm/tests/files/` could assert the empty-string result more directly, but it is not required.

## Questions for Author
- Pre-existing, out of scope: `[]byte → string` charges no per-element CPU while `[]rune → string` does. Intentional (rune decoding does more work than a byte copy), or an accidental asymmetry worth a follow-up? Not a blocker for this PR.
