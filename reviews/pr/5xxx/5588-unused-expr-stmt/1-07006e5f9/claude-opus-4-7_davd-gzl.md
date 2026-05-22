# PR #5588: fix(gnovm): detect unused expression statements at preprocess time

**URL:** https://github.com/gnolang/gno/pull/5588
**Author:** aronpark1007 | **Base:** master | **Files:** 4 | **+43 -0**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)

## Summary

Closes #3426. Gno's preprocessor previously accepted invalid expression statements (bare `1`, `x + 1`, `x`, etc.) and let them reach the VM, where they could trigger runtime panics or — combined with infinite recursion — OOM crashes. This PR closes the gap by adding a `*ExprStmt` case in [preprocess.go:2519](.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2519) (`TRANS_LEAVE`):

```go
case *ExprStmt:
    if _, ok := n.X.(*CallExpr); !ok {
        panic(fmt.Sprintf("invalid operation: %s is not used", n.X.String()))
    }
```

Three filetests (`expr_stmt0..2.gno`) cover literal-as-statement, name-as-statement, and binary-expr-as-statement.

## Test Results

- **Existing tests:** PASS (filetest suite excluded — see Note). The PR's own three fixtures pass; broader `TestFiles` failures all pre-exist on master (Go-toolchain wording diffs in `slice_2/3/5`, `varg_12`, `addressable_1b/d_err`, `invalid_labels0`, `persist_native`, `redeclaration_global5`).
- **CI:** all jobs pass; `Merge Requirements` red is the codeowner gate. Codecov 100% on patched lines.
- **Adversarial filetests:** 5 written. 3 reveal expected behavior, 2 reveal **a real correctness gap** (see Critical):
  - `expr_stmt_complit.gno` — `T{x: 1}` standalone: PR correctly errors ✓
  - `expr_stmt_iife.gno` — `func() {…}()` standalone: allowed (CallExpr) ✓
  - `expr_stmt_selector.gno` (selector `t.x`) — PR correctly errors ✓
  - `expr_stmt_typeconv.gno` — `int64(x)` standalone: **PR allows, Go rejects** ✗
  - `expr_stmt_append.gno` — `append(s, 3)` standalone: **PR allows, Go rejects** ✗

## Critical (must fix)

- [ ] [preprocess.go:2517-2521](.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2517-L2521) — type conversions slip through. The check `n.X.(*CallExpr)` matches both true function calls (`f()`) AND type conversions (`int64(x)`, `T(x)`). In the gno AST, type conversion is parsed as `*CallExpr` and stays a `*CallExpr` after `TRANS_LEAVE` — the type-conversion branch at [preprocess.go:1491](.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L1491) sets `ATTR_TYPEOF_VALUE` but doesn't change the node kind. Go rejects standalone type conversions ("`int64(x)` (value of type int64) is not used"). PR doesn't.
  - Fix: when `n.X` is a `*CallExpr`, also evaluate `evalStaticTypeOf(store, last, cx.Func)` and reject if it is a `*TypeType`. Alternative: hook into the `case *TypeType:` branch at line 1491 and tag the node so the `ExprStmt` check can short-circuit.
  - Add a fixture (`expr_stmt3.gno` mirroring `tests/expr_stmt_typeconv.gno`) once fixed.
- [ ] PR description claims "Only call expressions … are valid as expression statements." That's not the full Go rule — Go further requires the call to be a *function call*, not a type conversion, and modern Go also rejects bare `append(...)`. Update the description (and ideally the comment in code) to call out the type-conversion gap or the chosen scope.

## Warnings (should fix)

- [ ] [preprocess.go:2520](.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2520) — error format `"invalid operation: %s is not used"` interpolates `n.X.String()`, which dumps internal AST decoration (e.g. `(const (1 <untyped> bigint))`, `x<VPBlock(1,0)>`). Compare with Go's clean `"1 (untyped int constant) is not used"`. The current pinning means every test fixture's `// Error:` line is unreadable. Either trim the AST dump from the message (use the source slice via `n.X.GetSpan()` / source bytes) or align with the typecheck wording. Test fixtures already show the disparity:
  ```
  // Error:        main/expr_stmt0.gno:6:2-3: invalid operation: (const (1 <untyped> bigint)) is not used
  // TypeCheckError: main/expr_stmt0.gno:6:2: 1 (untyped int constant) is not used
  ```
- [ ] No comment explains *why* only `*CallExpr` is accepted. The PR description mentions channel receives, but the code doesn't. Add a one-line comment so the next maintainer doesn't re-add `<-ch` thinking it was missed (Gno doesn't have channels yet, but might).
- [ ] `append(s, x)` standalone: PR allows it; recent Go rejects it. Decide: match Go strictly (reject any CallExpr where the result is unused and there are no observable side effects beyond return value) or keep this pragmatic loophole. Either way, document it in the PR / a comment.

## Nits

- [ ] [preprocess.go:2517](.worktrees/gno-review-5588/gnovm/pkg/gnolang/preprocess.go#L2517) — the new case follows a `// TRANS_LEAVE -----------------------` comment block; the existing comment style is consistent. Minor: insert one blank line above for readability.
- [ ] Filetests `expr_stmt0/1/2.gno` are sequentially numbered and not very descriptive. Renaming to `expr_stmt_literal.gno` / `expr_stmt_name.gno` / `expr_stmt_binary.gno` would help future contributors find the right file when adding a case.
- [ ] PR description mentions "channel receive operations are also valid expression statements per the Go spec, but are omitted here as Gno does not support channels." Move that note into a code comment near the new case so it survives if the description is rewritten.

## Missing Tests

- [ ] **Type conversion as statement** (`int64(x)`) — see Critical. (`gnovm/tests/files/expr_stmt_typeconv.gno`)
- [ ] **Composite literal as statement** (`T{}`) — currently caught (CompositeLit, not CallExpr). Pin in a fixture. (`gnovm/tests/files/`)
- [ ] **Selector/index as statement** (`t.x`, `arr[0]`) — currently caught. Pin in fixtures. (`gnovm/tests/files/`)
- [ ] **IIFE as statement** (`func() {…}()`) — must continue to work. Pin so the regression is loud. (`gnovm/tests/files/`)
- [ ] **Method call as statement** (`obj.M()`) — `*CallExpr` with selector Func; should be allowed. Add fixture for completeness.
- [ ] **Builtins as statement** (`recover()`, `panic(x)`, `len(x)`) — `recover()` and `panic(x)` are clearly call-with-side-effect; `len(x)` is the same kind of "discarded value" case as `append`. Decide and pin.
- [ ] **Realm `cross()` as statement** — gno-specific; verify the wrapping is recognized as `*CallExpr`. Add fixture.

## Suggestions

- The change is in `TRANS_LEAVE`, which means the preprocessor runs the rest of the pass on a known-broken statement before erroring. For a stricter early exit, consider `TRANS_BLOCK` for `*ExprStmt` — but `TRANS_LEAVE` matches the surrounding code style. Up to the author.
- The PR fixes a concrete OOM/panic-via-runtime crash. Consider linking the original crash report (#3426) directly in the new code as a `// Fixes #3426.` comment so the regression context survives forever.
- The error message could be more helpful when it points at a type conversion. After fixing the Critical above, "T(x) is not used" reads as "expected; you probably want a function call here" rather than "internal AST string dump."

## Questions for Author

- Was the type-conversion case considered? If yes, intentional scope decision or oversight?
- The error string includes the internal AST representation. Was matching Go's wording considered? If yes, why prefer the AST form?
- Should the check also reject `append(s, x)` standalone? Modern Go rejects it; the current PR allows it.

## Verdict

**REQUEST CHANGES** — the direction is right and the fix is small, but the check is incomplete: type conversions (`int64(x)`, `T(x)`) are still accepted because they share the `*CallExpr` AST kind with function calls. That's the same class of unused-result problem the PR is fixing, so leaving it open re-introduces a partial gap that will likely become the next bug report. Once the type-conversion case is rejected (and ideally the error message is cleaned up to match Go's wording), this is a clean, focused fix worth merging.
