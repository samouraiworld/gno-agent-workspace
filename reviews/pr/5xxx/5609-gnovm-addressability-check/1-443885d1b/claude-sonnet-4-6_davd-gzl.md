# PR #5609: fix(gnovm): validate addressability of & operand at preprocess stage

**URL:** https://github.com/gnolang/gno/pull/5609
**Author:** aronpark1007 | **Base:** master | **Files:** 27 | **+135 -26**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Fixes #5586: the GnoVM executed `&expr` for non-addressable expressions without error; Go rejects these at compile time. Moves validation from a runtime crash to a structured preprocess-stage panic. Adds `isAddressable()` in `preprocess.go` covering: `*NameExpr` (true), `*StarExpr` (true), `*CompositeLitExpr` (true), `*IndexExpr` on Slice (true), Map (false), Array (recursive), `*SelectorExpr` (recursive or true for pointer base), all others (false). Also adds slice-of-unaddressable-array check in `TRANS_LEAVE *SliceExpr`.

## Test Results
- **Existing tests:** PASS (full gnolang suite + all 41 addressable_*.gno file tests)
- **Edge-case tests:** skipped

## Critical (must fix)
None.

## Warnings (should fix)

- [ ] `preprocess.go:4395` — `*NameExpr` is unconditionally addressable, but Go rejects `&funcName` for named functions (`invalid operation: cannot take address of bar`). Gno silently accepts it. Pre-existing gap, not a regression. Fix: check `evalStaticTypeOf(...).Kind() == FuncKind` and return `false`.
- [ ] `preprocess.go:4395` — `*NameExpr` unconditionally true means `&str[0]` (string byte index) also passes: `*IndexExpr` falls through to `isAddressable(cx.X)` for StringKind, `*NameExpr` base returns `true`. Go rejects this. Fix: add `StringKind: return false` in the `*IndexExpr` switch. File `addressable_4a_err.gno` only has `// TypeCheckError:`, confirming no preprocess check.

## Nits

- [ ] `preprocess.go:4393` — `isAddressable` lacks a doc comment explaining Go spec correspondence and known gaps.
- [ ] Error messages still expose internal Gno representation (`VPBlock(3,2)`, `(const-type struct{})`) — pre-existing style issue.

## Missing Tests

- [ ] `&functionName` for package-level function (currently accepted, should be rejected).
- [ ] `&str[0]` for string variable byte index (no `// Error:` annotation in `addressable_4a_err.gno`).

## Suggestions

- Add `StringKind: return false` and `FuncKind`/`TypeKind` exclusions for `*NameExpr` in a follow-up PR to complete Go spec alignment.

## Questions for Author

- Was `*NameExpr → always true` intentional to keep function-value-in-variable patterns working, or are the named-function and string-index gaps tracked elsewhere?
- `baseOf()` calls were removed in the simplification commit; is there any `DeclaredType` wrapping case where Kind() could differ from the unwrapped base's Kind()?

## Verdict

APPROVE — Correctly fixes #5586 by moving addressability validation to preprocess for all targeted expression classes; all tests pass. Two residual gaps (`&funcName`, `&str[0]`) pre-date this PR and should be tracked as follow-up.
