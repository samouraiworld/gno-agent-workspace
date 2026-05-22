# PR #5490: chore(gnovm): remove unreachable panic in doOpEnterCrossing

**URL:** https://github.com/gnolang/gno/pull/5490
**Author:** thehowl | **Base:** master | **Files:** 1 | **+0 -2**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4.6

## Summary

This PR removes two dead lines from `doOpEnterCrossing()` in `gnovm/pkg/gnolang/op_call.go`:

```go
//nolint:govet // detected as unreachable
panic("should not happen") // defensive
```

The function `doOpEnterCrossing` (lines 91-133) contains an infinite `for i := 1; ; i++` loop starting at line 102 that has three exit paths, all via `return` or `panic`:

1. **Line 117**: `return` when `fri == nil` (frame index beyond stack boundary, i.e. stage-add/run context)
2. **Line 125**: `return` when `fri.WithCross || fri.DidCrossing` (valid crossing found)
3. **Line 131**: `panic(...)` when `fri.LastRealm != m.Realm` (invalid crossing state)

If none of the three conditions match (i.e. `fri != nil`, `!fri.WithCross && !fri.DidCrossing`, and `fri.LastRealm == m.Realm`), the loop continues to the next iteration — it never breaks. There is no code path that exits the `for` loop normally, so the code after it is truly unreachable.

The `//nolint:govet` directive was ineffective because `go vet` (which detects unreachable code) does not honor golangci-lint suppression comments. As the PR author notes, this creates spurious LSP diagnostics.

## Test Results
- **Existing tests:** PASS (the one failure in `gnovm/pkg/gnolang` tests is pre-existing on master — a `varg_12.gno` error message format mismatch unrelated to this change)
- **CI:** All checks pass
- **Edge-case tests:** skipped (no behavioral change)

## Critical (must fix)
None

## Warnings (should fix)
None

## Nits
None

## Missing Tests
None — this is a pure dead code removal with no behavioral change.

## Suggestions
None

## Questions for Author
None

## Verdict
APPROVE — Correct removal of provably unreachable dead code. The infinite `for` loop has no break/fallthrough path, making the panic after it impossible to reach. Clean, zero-risk change.
