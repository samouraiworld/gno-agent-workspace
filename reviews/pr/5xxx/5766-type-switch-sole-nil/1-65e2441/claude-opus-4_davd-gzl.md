# PR #5766: fix(gnovm): allow type-switch with sole `case nil:` (preprocess single-case branch must tag-type-only when ct is nil)

URL: https://github.com/gnolang/gno/pull/5766
Author: ltzmaxwell | Base: master | Files: 2 | +88 -2
Reviewed by: davd-gzl | Model: claude-opus-4 | Commit: 65e2441 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5766 65e2441`

**Verdict: APPROVE** — minimal, correct one-line preprocess fix; behavior now matches Go; regression covered by a filetest that panics on master and passes here. No blocking concerns.

## Summary
A type-switch clause whose sole case is `nil` (`case nil:`) panicked at preprocess with `name xx not declared` when the switch bound a variable (`switch xx := x.(type)`). The single-case shortcut defined the clause variable as `anyValue(ct)` with `ct == nil`, leaving the variable effectively undeclared so any later use of it panicked. Go binds the variable to the tag's static type in this case. The fix gates the shortcut on `ct != nil`, so a nil case type falls through to the same tag-type define already used for multi-case clauses.

## Glossary
- `ss.VarName` — the variable bound by the type-switch guard (`xx` in `switch xx := x.(type)`); empty for the bare `x.(type)` form.
- `ct` — the case's static type; `nil` for `case nil:`.
- tag type — static type of the guard expression `ss.X`, computed via `evalStaticTypeOf`.
- `n.Cases` — the case expressions of a single `SwitchClauseStmt` (one clause), not the whole switch; `case nil:` is one clause with `len(n.Cases) == 1`.

## Fix
In [`preprocess.go:1045`](https://github.com/gnolang/gno/blob/65e2441/gnovm/pkg/gnolang/preprocess.go#L1045) · [↗](../../../../../.worktrees/gno-review-5766/gnovm/pkg/gnolang/preprocess.go#L1045) the condition changed from `len(n.Cases) == 1` to `len(n.Cases) == 1 && ct != nil`. Before: a sole `case nil:` clause hit the "define with case type" branch and called `last.Define(ss.VarName, anyValue(nil))`, which does not produce a usable declaration. After: `ct == nil` routes to the else branch, defining the variable with the tag type ([`preprocess.go:1055`](https://github.com/gnolang/gno/blob/65e2441/gnovm/pkg/gnolang/preprocess.go#L1055) · [↗](../../../../../.worktrees/gno-review-5766/gnovm/pkg/gnolang/preprocess.go#L1055)), identical to the multi-case path. The per-clause framing is the load-bearing detail: every `case nil:` is a single-case clause, so this path is hit whether nil is the only clause in the switch or sits beside other clauses.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- [`typeswitch1.gno:1-5`](https://github.com/gnolang/gno/blob/65e2441/gnovm/tests/files/typeswitch1.gno#L1-L5) · [↗](../../../../../.worktrees/gno-review-5766/gnovm/tests/files/typeswitch1.gno#L1-L5) — the header comment says preprocess "registered the type-switch var with a nil static type when ct was nil"; precise, but the user-visible symptom (`name xx not declared` panic) is the more useful anchor and is already named. Fine as-is.

## Missing Tests
- None blocking. The fix is exercised: the `case nil:` clauses in `whatis` ([`typeswitch1.gno:37-38`](https://github.com/gnolang/gno/blob/65e2441/gnovm/tests/files/typeswitch1.gno#L37-L38) · [↗](../../../../../.worktrees/gno-review-5766/gnovm/tests/files/typeswitch1.gno#L37-L38)) are single-case clauses with `ct == nil`, the exact buggy path. `check(nil, "nil <nil>")` ([`typeswitch1.gno:82`](https://github.com/gnolang/gno/blob/65e2441/gnovm/tests/files/typeswitch1.gno#L82) · [↗](../../../../../.worktrees/gno-review-5766/gnovm/tests/files/typeswitch1.gno#L82)) asserts the runtime value. Confirmed: the test panics on master and passes on this PR (repro below).
  <details><summary>optional strengthening</summary>

  The clause sits inside a multi-clause switch, so it reads as "nil alongside other cases" rather than "literally the only clause". The two are the same code path (per-clause `n.Cases`), so this is not a real gap, but a 5-line filetest with `switch v := x.(type) { case nil: _ = v }` as the single clause would make the title's "sole `case nil:`" claim self-evident to a future reader. Optional.
  </details>

## Suggestions
None.

## Questions for Author
None.

---

### Repro: master panics, PR passes

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5766 -R gnolang/gno
# PR head: passes
go test -run 'TestFiles/typeswitch1.gno$' ./gnovm/pkg/gnolang/
# revert just the preprocess gate to master and re-run: panics
git checkout origin/master -- gnovm/pkg/gnolang/preprocess.go
go test -run 'TestFiles/typeswitch1.gno$' ./gnovm/pkg/gnolang/
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go
```

```
# PR head:
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.486s

# master preprocess:
files_test.go:111: unexpected panic: main/typeswitch1.gno:38:29-31: name xx not declared
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.382s
```

Go parity (`whatis(nil)` in real Go) prints `"nil <nil>"`, matching the filetest's `check(nil, "nil <nil>")` — the fix reproduces Go's tag-type binding for a sole `case nil:`.
