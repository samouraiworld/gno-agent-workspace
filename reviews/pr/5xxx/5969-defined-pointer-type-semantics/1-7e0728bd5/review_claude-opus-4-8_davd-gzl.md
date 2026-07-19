# PR [#5969](https://github.com/gnolang/gno/pull/5969): fix(gnovm): match Go semantics for defined pointer types (selectors, embedding)

URL: https://github.com/gnolang/gno/pull/5969
Author: Romainua | Base: master | Files: 10 | +357 -21
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7e0728bd5 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5969 7e0728bd5`

**TL;DR:** In Go you can write `type D1 *D2` to give a pointer type its own name. Such a name gets no methods of its own, and Go says so with a normal compile error when you try to call one. GnoVM instead crashed with an internal `should not happen` message. This PR makes GnoVM answer like Go, and also rejects putting such a name inside a struct as an embedded field, which Go has always refused.

**Verdict: REQUEST CHANGES** — the fix is correct and regression-free for every shape it covers, but it changes which types satisfy an interface with no test asserting it, and two neighbouring shapes still reach the same internal panic (2 Warnings, 2 Missing Tests, 1 Nit).

## Summary

`type D1 *D2` is a defined type whose underlying type is a pointer. Go gives it an empty method set: only fields promote through it, via the `x.f` → `(*x).f` shorthand, which the spec restricts to selectors denoting a field and not a method. GnoVM's embedded lookup promoted `D2`'s methods through `D1`, so phase 1 of the lookup reported a hit and phase 2 hit a `default:` branch it had no trail shape for, panicking `should not happen`.

The fix threads a `fieldsOnly` flag through the breadth-first lookup: crossing a defined type's pointer base switches method lookups off, and a second crossing exposes nothing. A separate declaration-time check in `fillEmbeddedName` rejects an embedded field that is still of pointer kind after one dereference, matching Go's "embedded field type cannot be a pointer".

Both changes land on the code master rewrote in [#5721](https://github.com/gnolang/gno/pull/5721); the branch's original patch targeted the old recursive lookup and was re-authored inside the merge commit, so the whole of [`types.go`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go) in this diff is first-round content.

```
    var x D1                type D1 *D2       type D2 struct{ E; A int }
       |
  root = D1 (DeclaredType) ---- fieldsOnly=false
       |  Base
    *D2 (PointerType) ---------- crossing: fieldsOnly := true
       |  canonEmbeddedType
     D2 (DeclaredType) --------- methods skipped  (x.Foo  -> not found)
       |  Base
   struct{E; A} -------------- fields still hit   (x.A    -> found)
       |  embedded field E, inherits fieldsOnly=true
      E ---------------------- methods skipped    (x.Bar  -> not found)
```

## Examples

| Written as | Go | GnoVM at 7e0728bd5 |
|---|---|---|
| `type D1 *D2; var x D1; x.Foo` (method) | `x.Foo undefined` | `missing field Foo in main.D1` |
| `type D1 *D2; var x D1; x.A` (field) | promotes | promotes |
| `type D1 *D2; var x D1; x.E.Bar()` | works | works |
| `type B *C; type A *B; var x A; x.F` | `x.F undefined` | `missing field F in main[...].A` |
| `type S struct{ D1 }` | rejected at declaration | rejected at declaration |
| `type P = *D2; type S struct{ P }` | accepted | accepted |
| `var x D1; var f Fooer = x` | `D1 does not implement Fooer` | `main.D1 does not implement main.Fooer` |
| `var p *D1; p.A` | `p.A undefined` | panic `should not happen` |
| `type S struct{ *I }; s.M` | rejected at declaration | panic `should not happen` |

## Glossary

- defined type: a type introduced by `type N U`, distinct from its underlying type and from an alias; `type N *T` has an empty method set.
- promoted field/method: a name of an embedded type reachable on the embedder, resolved at the shallowest embedding depth.
- selector: an `x.f` expression, resolved by the preprocessor to a ValuePath after walking embedding.
- ValuePath: the resolved access step a selector compiles to, carrying a kind (`VPField`, `VPDerefField`, `VPInterface`, method variants) and an embedding depth.
- filetest: a file under `gnovm/tests/files/` run by the VM and asserted against `// Output:` / `// Error:` golden directives.
- preprocess: the static pass that resolves names, types, and blocks before execution.

## Fix

Before, [`resolveEmbedNode`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3134) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3134) walked a declared type's wrapper spine and counted every method it met, including those behind the type's pointer base. Now the walk carries a `fieldsOnly` flag: the [`*PointerType` case](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3162-L3174) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3162-L3174) sets it on the first crossing and returns not-found on a second, and the declared-type and interface cases consult it before counting a method. The flag rides the breadth-first walk through [`embedLookupEntry.fieldsOnly`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L2985) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L2985) and a parallel [`childFOs`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3019) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3019) slice. The load-bearing constraint is that phase 1 must never report a winner phase 2 cannot express as a trail, which is what the `default: panic` in [`buildEmbeddedTrail`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3265-L3266) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3265-L3266) enforces.

## Critical (must fix)

None.

## Warnings (should fix)

- **[same panic, one shape away]** `gnovm/pkg/gnolang/types.go:3129-3133` — a pointer to a defined pointer type still panics `should not happen`, because the outer pointer is stripped before the walk starts, so the crossing is entered one level too late.
  <details><summary>details</summary>

  The doc comment at [`types.go:3130-3132`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3130-L3132) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3130-L3132) rests on root pointers being stripped by [`canonEmbeddedType`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3106-L3115) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3106-L3115) before the walk. For root `*D1` that strip yields `D1`, the walk starts with `fieldsOnly` false, and a field of `D2` is reported found. Go rejects `p.A` outright: the `(*x).f` shorthand requires the operand type not to be a pointer type, and `*D1` is one. Phase 2 then builds a `VPDerefField`-headed trail and feeds it to [`applyPointerDeref`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L731-L732) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L731-L732), whose `default:` branch panics. The method form `p.Foo` is already correct, so only the field path is exposed. This reproduces at the merge base 959cefd91 as well, so it is not a regression; it is the same panic class the PR sets out to close, and [`struct64b.gno:10`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/struct64b.gno#L10) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/struct64b.gno#L10) already puts `*D1` in the PR's test surface. Filetest asserting the post-fix state: [`tests/ptr14.gno`](tests/ptr14.gno), red at 7e0728bd5, [repro](comment_claude-opus-4-8.md). Fix: make the root strip in [`findEmbeddedFieldType`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L2927-L2932) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L2927-L2932) report nothing when the element it uncovers is itself of pointer kind.
  </details>

- **[adjacent rule left out]** `gnovm/pkg/gnolang/types.go:2564-2566` — an embedded pointer to a defined interface type is accepted, and selecting through it panics `should not happen`; Go rejects it at declaration from the same check that rejects an embedded pointer type.
  <details><summary>details</summary>

  Go's struct-field check dereferences the field type once and then rejects a pointer underlying type and a pointer-to-interface separately, with "embedded field type cannot be a pointer" and "embedded field type cannot be a pointer to an interface". The new guard ports the first. For `type S struct{ *I }` with `I` an interface, `unwrapPointerType` yields `I`, whose kind is interface, so the guard passes. [`canonEmbeddedType`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L3106-L3115) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L3106-L3115) then returns the declared interface type rather than nil, the lookup reports `M` found, and phase 2 hands a `VPInterface`-headed trail to [`applyPointerDeref`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/types.go#L731-L732) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/types.go#L731-L732), which panics. Reproduces at 959cefd91 too, so it is pre-existing. Filetest asserting the post-fix state: [`tests/struct65.gno`](tests/struct65.gno), red at 7e0728bd5, [repro](comment_claude-opus-4-8.md). Fix: extend the same guard so a pointer whose element resolves to an interface is rejected with Go's second message.
  </details>

## Nits

- **[test comment restates the golden]** `gnovm/tests/files/method47.gno:12` — the trailing `// want: method not found, not "should not happen"` duplicates what the `// Error:` golden already asserts, and `want:` is not a filetest directive. Same shape at [`method48.gno:14`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/method48.gno#L14) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/method48.gno#L14), [`struct64.gno:10`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/struct64.gno#L10) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/struct64.gno#L10), [`struct64b.gno:10`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/struct64b.gno#L10) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/struct64b.gno#L10). No enabled linter covers filetest comments ([`.github/golangci.yml`](https://github.com/gnolang/gno/blob/7e0728bd5/.github/golangci.yml?plain=1#L13) · [↗](../../../../../.worktrees/gno-review-5969/.github/golangci.yml#L13) runs `default: none`), and it changes no meaning. Not posted, no change needed.

## Missing Tests

- **[behavior change with no assertion]** `gnovm/pkg/gnolang/types.go:3153-3160` — nothing asserts the assignability change: a defined pointer type no longer satisfies an interface through its base's methods.
  <details><summary>details</summary>

  `VerifyImplementedBy` resolves interface methods through the same lookup, so suppressing method hits past the pointer crossing changes which types are assignable to which interfaces. That is a semantic change to the type system, and no filetest in the PR covers it: `method47-50`, `ptr12-13`, and `struct64/64b` all exercise selectors and declarations only. Confirmed behaviorally: at the merge base 959cefd91 `var f Fooer = x` with `x` of type `D1` panics `should not happen`; at 7e0728bd5 it reports `main.D1 does not implement main.Fooer (missing method Foo)`, matching the Go compiler's verdict. Ready-to-add filetest: [`tests/method51.gno`](tests/method51.gno), green at 7e0728bd5 and red at the merge base. Fix: add it under `gnovm/tests/files/`.
  </details>

- **[new rejection has no legal-case guard]** `gnovm/tests/files/struct64b.gno:10` — the new embedded-pointer rejection has no test for the legal case it must not catch, an alias of a pointer type.
  <details><summary>details</summary>

  `struct64.gno` and `struct64b.gno` assert the rejection; nothing asserts the boundary on the other side. `type P = *D2; type S struct{ P }` is legal Go, since an alias introduces no defined type and the guard's one dereference reaches the struct. The guard runs on every embedded field of every struct in every package, so a false positive there rejects valid programs wholesale, and the aliased spelling is the only nearby input that reaches the same branch. Confirmed behaviorally: the case compiles under Go and prints `3` under GnoVM at 7e0728bd5. Ready-to-add filetest: [`tests/struct64c.gno`](tests/struct64c.gno), green at 7e0728bd5. Fix: add it under `gnovm/tests/files/`.
  </details>

## Suggestions

None.

## Verified

- GnoVM's verdicts match the Go compiler on the shapes the PR covers, checked by compiling each shape with `go build` rather than reasoning about the spec: method through a defined pointer type, field through one, an alias of a pointer type embedded, a defined pointer type embedded, and interface satisfaction. Harness: [`tests/go_parity_test.go`](tests/go_parity_test.go), all seven cases green.
- The two shapes in the Warnings diverge in the same harness: Go rejects `p.A` for `p` of type `*D1` and rejects `struct{ *I }` at declaration, while GnoVM panics on both.
- The assignability change is real, not incidental: the same program panics `should not happen` at the merge base 959cefd91 and reports the correct "does not implement" error at 7e0728bd5.
- No defined pointer type exists anywhere under `examples/` or `gnovm/stdlibs/`, so the new declaration-time rejection cannot break a package already in the tree.
- Green at 7e0728bd5: the eight new filetests individually; `go test ./gnovm/pkg/gnolang/... -test.short -skip TestFiles`; `go test ./gnovm/tests/... -test.short`. `go test ./gnovm/pkg/gnolang/ -run Files -test.short` fails 10 goldens, an identical set to the merge base 959cefd91 (go/types message drift from a local Go newer than CI).

## Existing threads

- [@notJoon](https://github.com/gnolang/gno/pull/5969#issuecomment-4992103025) asked for the nested `type B *C; type A *B` case; the author added [`ptr12.gno`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/tests/files/ptr12.gno#L1) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/tests/files/ptr12.gno#L1) and it passes. No overlap with own findings; thread not formally resolved, awaiting a re-review the author requested.

## Open questions

- The `fieldsOnly` flag can only ever be set at the root now that `fillEmbeddedName` rejects embedded pointer-kind fields, so `embedLookupEntry.fieldsOnly` and the `childFOs` slice carry a value constant across the whole walk. It is still the right shape as defence for struct types decoded from the store, which bypass `fillEmbeddedName` entirely ([`pb3_gen.go:14442`](https://github.com/gnolang/gno/blob/7e0728bd5/gnovm/pkg/gnolang/pb3_gen.go#L14442) · [↗](../../../../../.worktrees/gno-review-5969/gnovm/pkg/gnolang/pb3_gen.go#L14442) sets `Embedded` straight from the wire). Not posted: no defect, and pointing at redundancy would invite removing the safety net.
- CI on this PR ran only the bot and PR-metadata jobs; the test suites are gated behind initial maintainer approval. Not posted: routine, and nothing for the author to do.
