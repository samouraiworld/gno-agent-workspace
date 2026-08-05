# PR [#6037](https://github.com/gnolang/gno/pull/6037): fix(preprocess): map composite-literal keys resolution inside for loop

URL: https://github.com/gnolang/gno/pull/6037
Author: Villaquiranm | Base: master | Files: 2 | +23 -4
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 390bffe90 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6037 390bffe90`

**TL;DR:** Writing `map[int]int{i: i+1}` inside a `for i := ...` loop used to be rejected with `name i not declared`, even though the same line compiles in Go. This makes the map key resolve to the loop variable, the way it already did everywhere else in the loop body.

**Verdict: APPROVE** — the fix holds on every literal shape I could build and matches Go on each one; land the stale [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L194-L197) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L194-L197) contract comment and the nested-literal filetest with it (1 Warning, 2 Nits, 2 Missing Tests, 1 Suggestion).

## Verify first

- [`gnovm/pkg/gnolang/preprocess.go:1279-1281`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279-L1281) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279-L1281) — the trim is the whole safety net for struct keys, and one existing filetest is the only thing holding it. Delete those two lines and run `go test -run 'TestFiles/loopvar_struct_field_2.gno$' ./gnovm/pkg/gnolang/`; it must fail with `struct type struct{i int} has no field i.loopvar`.
- [`gnovm/pkg/gnolang/preprocess.go:2379-2381`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L2379-L2381) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L2379-L2381) — one of the two places that read a struct composite key by name rather than by path, and both must see the trimmed name. This one sits on the parent `*CompositeLitExpr`'s `TRANS_LEAVE`, which children leave before; the other is the runtime duplicate-field panic at [`op_expressions.go:660`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/op_expressions.go#L660) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/op_expressions.go#L660), long after preprocessing.

## Summary

Correct fix. [`initStaticBlocks1`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L200) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L200) gives each iteration of a `for` loop its own binding by suffixing the loop variable with `.loopvar` at its declaration ([`preprocess.go:298`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L298) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L298)) and at every reference ([`preprocess.go:364`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L364) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L364)). Composite-literal keys were excluded wholesale, because a struct key such as `T{i: 1}` is a field name and renaming it corrupts the field lookup. A map key is not a field name, so the exclusion left it pointing at a name that no longer exists.

The diff drops `TRANS_COMPOSITE_KEY` from the exclusion list ([`preprocess.go:354-358`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L354-L358) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L354-L358)) and undoes the rename later, in the one branch that knows the key is a field name ([`preprocess.go:1273-1281`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1273-L1281) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1273-L1281)). The type is resolved by then, so the ambiguity the exclusion was working around no longer exists at that point.

## Diagram

```
initStaticBlocks1                     preprocess1 (TRANS_LEAVE *NameExpr)
─────────────────                     ────────────────────────────────────
for i := ... {                        ftype == TRANS_COMPOSITE_KEY
  m := M{ i : ... }                     └─ baseOf(clx.Type)
         │                                   ├─ *StructType  → TrimSuffix ".loopvar"  ← diff
         └── rename → i.loopvar  ← diff      │                  n.Path = field path
             (was: skipped)                  ├─ *Array/Slice → const key or reject
                                             └─ (map, other) → resolve as a variable
```

## Examples

| Written inside `for i := 0; i < 2; i++` | Before | After |
|---|---|---|
| `map[int]int{i: i + 1}` | `name i not declared` | `1`, `2` |
| `M{i: i + 1}`, `type M map[int]int` | `name i not declared` | `1`, `2` |
| `map[int]T{i: {i: i}}` | `name i not declared` | `0`, `1` |
| `T{i: i * 10}`, field `i` | `0`, `10` | `0`, `10` |
| `[3]int{i: 1}` | `name i not declared` | `slice/array literals may not contain non-const keys` |

Go rejects the last row too, with `index i must be integer constant`; every accepted row prints what Go prints.

## Critical (must fix)

None.

## Warnings (should fix)

- **[the comment tells the next reader to undo the fix]** [`gnovm/pkg/gnolang/preprocess.go:194-197`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L194-L197) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L194-L197) — the `initStaticBlocks1` contract still names composite-literal keys as a position the rename skips, which is what the diff removed.
  <details><summary>details</summary>

  The doc comment states the rename applies "iff its name is in the top frame and its position is not a declaration site ([composite-literal key](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L196) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L196), var-name, range key/value, or assign-LHS in a DEFINE)". After this diff composite keys are renamed like any other reference, and the exclusion list at [`preprocess.go:354-358`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L354-L358) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L354-L358) no longer holds `TRANS_COMPOSITE_KEY`. This is the one paragraph a maintainer reads before touching the pass, and it now describes the pre-fix behavior, so the obvious reading is that the missing case is a bug to restore. Restoring it reopens [#5910](https://github.com/gnolang/gno/issues/5910). The comment also has to say where the rename is undone, since the pass alone no longer tells the whole story. Fix: rewrite the sentence to say composite-literal keys are renamed and that [`preprocess1`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279-L1280) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279-L1280) strips the suffix back off once the literal's type resolves to a struct.
  </details>

## Nits

- **[the test's stated purpose is now the opposite of what it checks]** [`gnovm/tests/files/loopvar_struct_field_2.gno:4`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/loopvar_struct_field_2.gno#L4) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/loopvar_struct_field_2.gno#L4) — says the test checks that composite keys are not renamed; after the diff they are renamed, and what the test checks is that the rename is undone.
  <details><summary>details</summary>

  The header reads "Tests that TRANS_COMPOSITE_KEY NameExprs are not renamed to i.loopvar" ([`loopvar_struct_field_2.gno:4`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/loopvar_struct_field_2.gno#L4) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/loopvar_struct_field_2.gno#L4)). This file is the only regression guard on the new trim: removing the two trim lines and leaving everything else makes it fail with `struct type struct{i int} has no field i.loopvar`. A reader who trusts the header reads it as covering a mechanism the code dropped, which is exactly the way a guard gets deleted as obsolete. Fix: say the field name is renamed by the loop-var pass and restored once the literal's type is known.
  </details>

- **[two names for one value]** [`gnovm/pkg/gnolang/preprocess.go:1279-1286`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279-L1286) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279-L1286) — `fname` and `n.Name` hold the same string from line 1280 onward, and the block then uses both.
  <details><summary>details</summary>

  [`n.Name = Name(fname)`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1280) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1280) makes the two interchangeable, yet [line 1281](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1281) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1281) reads `n.Name` while [lines 1283 and 1286](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1283-L1286) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1283-L1286) read `fname`. The diff's changes to the `isUpper` argument and the panic argument are then no-ops, which costs a reader a check to establish. Fix: assign `n.Name` directly and drop `fname`, leaving the two `isUpper`/panic lines untouched by the diff.
  </details>

## Missing Tests

- **[the half-fix passes every shipped test]** [`gnovm/tests/files/maplit15.gno:1-13`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/maplit15.gno#L1-L13) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/maplit15.gno#L1-L13) — no test fails when the exclusion is dropped without the suffix trim, or when the trim lands without the exclusion drop; one nested literal covers both directions.
  <details><summary>details</summary>

  [`maplit15.gno`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/maplit15.gno#L1-L13) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/maplit15.gno#L1-L13) covers a bare `map[int]int` key and nothing else. It says nothing about the named map type the linked issue names, nor about the elided nested literal where a map key and a struct field key sit in the same expression and have to be treated differently. `M{i: {i: i + 4}}` with `type M map[int]T` fails at the merge base with `name i not declared` and, with the exclusion dropped but no trim, fails with `struct type struct{i int} has no field i.loopvar`. Only the full diff passes it. Fix: add the ready filetest at [`tests/loopvar_composite_key_nested.gno`](tests/loopvar_composite_key_nested.gno).
  </details>

- **[a wrong rename here changes a value, not a compile result]** [`gnovm/tests/files/maplit15.gno:1-13`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/maplit15.gno#L1-L13) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/maplit15.gno#L1-L13) — nothing covers a map key inside a closure that also captures the loop variable, the one shape where a mis-resolved key yields silently wrong output.
  <details><summary>details</summary>

  Every shipped case fails loudly when the rename is wrong, because the name stops resolving. The heap-promoted case does not: a key bound to the wrong iteration still compiles and prints the wrong number. Confirmed behaviorally on 390bffe90, `println(map[int]int{i: i * 2}[i])` inside a captured closure prints `0 2 4`, matching Go 1.22 per-iteration semantics; at the merge base it fails to preprocess. Fix: add the ready filetest at [`tests/loopvar_map_key_closure.gno`](tests/loopvar_map_key_closure.gno).
  </details>

## Suggestions

- **[one string, five hand-written copies]** [`gnovm/pkg/gnolang/preprocess.go:1279`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279) — the trim adds a fifth literal `".loopvar"`, and the rename half and the undo half now sit a thousand lines apart with nothing tying them together.
  <details><summary>details</summary>

  The suffix is written out at [`preprocess.go:295`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L295) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L295), [`:298`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L298) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L298), [`:364`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L364) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L364), [`:1279`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279) and [`debugger.go:773`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/debugger.go#L773) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/debugger.go#L773). Changing it in four of the five leaves a silent mismatch: the debugger one already fails open, returning `name X not declared` rather than an error naming the cause. A package-level `const loopvarSuffix = ".loopvar"` makes the set greppable and the two halves of this fix visibly paired.
  </details>

## Verified

- Deleting the two trim lines at [`preprocess.go:1279-1280`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1279-L1280) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1279-L1280) while keeping the exclusion-list change breaks [`loopvar_struct_field_2.gno`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/tests/files/loopvar_struct_field_2.gno#L1-L19) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/tests/files/loopvar_struct_field_2.gno#L1-L19) with `struct type struct{i int} has no field i.loopvar`, and leaves both new filetests failing the same way. The trim is load-bearing and that one file is the only committed guard on it.
- Seventeen literal shapes built and run under a `gno` binary from this worktree against the same source under `go run`: named and unnamed map types, elided struct and pointer-to-struct values, nested maps, a map keyed by a struct literal, multi-variable `for` init, a `switch` clause inside the loop, `defer`, a labeled `continue` with the loop variable's address taken, and closure capture. Output is identical to Go on every shape that compiles.
- The two rejection cases stay rejected and their messages did not regress: `[3]int{i: 1}` now reports [`slice/array literals may not contain non-const keys`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/preprocess.go#L1295) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/preprocess.go#L1295) instead of `name i not declared`, closer to Go's `index i must be integer constant`, and `T{i: 1}` on a struct with no field `i` reports `struct type struct{a int} has no field i` at both the merge base and this head, so the trim keeps the untrimmed name out of the error. `T{i: 1, i: 2}` inside the loop reaches the runtime message at [`op_expressions.go:660`](https://github.com/gnolang/gno/blob/390bffe90/gnovm/pkg/gnolang/op_expressions.go#L660) · [↗](../../../../../.worktrees/gno-review-6037/gnovm/pkg/gnolang/op_expressions.go#L660) and prints `duplicate field name i in struct literal` on both, covering the second reader of the key by name.
- `go test ./gnovm/pkg/gnolang/ -run TestFiles` green at 390bffe90 (491s), so no `// Preprocessed:` golden moved.

## Open questions

- The linked issue traces this to `map.go` in Go's test corpus, which is not in the tree; whether that corpus file now passes is only checkable in whatever branch imports it. Not posted: it asks the author for work outside this PR.
- The undo is a string operation on a name the parser cannot produce, so it cannot collide with a real field name. Recording that the safety rests on `.` being unparseable in an identifier, rather than on the trim, would help the next reader. Not posted: no change needed, and the Suggestion above already asks for the constant that would carry it.
