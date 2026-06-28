# PR #5763: fix(gnovm): allow unsealed *DeclaredType during mutual type-decl recursion

URL: https://github.com/gnolang/gno/pull/5763
Author: ltzmaxwell | Base: master | Files: 3 | +82 -7
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: `093c32be0` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5763 093c32be0`

Round 2 (head advanced `61fc396e4` → `093c32be0`; PR content changed). The round-1 REQUEST CHANGES is addressed: the author added `fillTypeInPlace` to complete the aliased base in place, so the `type T2 T1` empty-base corruption no longer occurs. All round-1 findings are resolved. Verdict flips to APPROVE.

**TL;DR:** In Gno you can write two types that refer to each other (`type T1 struct{Next *T2}; type T2 T1`). The compiler used to crash with `panic("should not happen")` on this, and the first fix attempt left `T2` silently broken so any field read on it panicked at runtime. This round fills `T2`'s underlying struct from `T1`'s completed one, so both halves of the cycle now resolve to the right type.

**Verdict: APPROVE** — `fillTypeInPlace` resolves the round-1 empty-base corruption; mutual type-decl recursion now produces the correct underlying type for both halves, illegal value-cycles still reject cleanly, and named-type identity is preserved despite the shared base pointer. Only a minor extra-coverage Suggestion remains.

## Summary

`tryPredefine` panicked `"should not happen"` whenever a type-decl's RHS resolved to a still-unsealed `*DeclaredType`, a state that is legitimate during mutual type-decl recursion. Round 1 removed the panic but the derived type `type T2 T1` then collapsed to `struct{}` because `T2`'s base aliased `T1`'s base struct pointer, captured while still empty. This round adds `fillTypeInPlace`: at `T2`'s finalize step it copies the completed underlying type into that original aliased base pointer in place and reuses it, so `T2.Base` and `T1.Base` end up the same fully-populated `*StructType`. Reading a field through `T2` now works in both declaration orders, and the two named types stay distinct.

```
master (guard present):            61fc396e4 (round 1):                093c32be0 (round 2):
  type T1 struct{Next *T2;Val int}   type T1 struct{Next *T2;Val int}   type T1 struct{Next *T2;Val int}
  type T2 T1                         type T2 T1                         type T2 T1
        |                                  |                                  |
  panic "should not happen" <- loud  T2.Base = struct{} <- silent       fillTypeInPlace(T1.Base, completed)
                                     corruption                          T2.Base == T1.Base, both filled
                                           |                                  |
                                     runtime panic on field read        T2.Val reads 9 <- correct
```

## Examples

| Source | Round 1 (`61fc396e4`) | Round 2 (`093c32be0`) |
|--------|-----------------------|------------------------|
| `type T1 struct{Next *T2;Val int}; type T2 T1`, read `a.Next.Val` | runtime panic `struct type struct{} has no field Val` | prints `9` |
| same, read `var b T2; b.Val` | runtime panic | prints the value |
| `type T2 T1` declared before `type T1 ...` | already worked | still works |
| `type T1 struct{Self T2;...}; type T2 T1` (finite-size cycle) | rejected | rejected, same as Go |

## Glossary
- TypeID: a type's canonical string identity that decides type equality and is persisted in on-chain object state.
- `tryPredefine` — preprocessor pass that partially defines package-level names so recursion resolves; `preprocess.go`.
- `DeclaredType` — a named type (`type Foo ...`); holds `Base` (underlying type), `Methods`, a `sealed` bit, and a name-derived `typeid`; `types.go`.
- `sealed` — one-bit lifecycle marker meaning "Base/Methods filled in"; recursion-protection only, not a slot-lookup precondition; `types.go`.
- TRANS_LEAVE TypeDecl — the handler that finalizes a named type: evaluates the RHS, builds a fresh `*DeclaredType`, seals it, copies it back into the predefined pointer; `preprocess.go`.

## Fix

Round 1 dropped the `panic("should not happen")` in `tryPredefine` ([`preprocess.go:5504-5508`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5504-L5508) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5504-L5508)). This round adds the actual resolution: a new `fillTypeInPlace` helper ([`types.go:1553-1592`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/types.go#L1553-L1592) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/types.go#L1553-L1592)) copies a completed underlying type into an existing same-kind pointer. In the `*DeclaredType` finalize branch, after sealing the fresh `tmp2`, the handler fills `dstT.Base` (the original, possibly-empty base captured by the dependent) in place and reuses it ([`preprocess.go:3077-3079`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3077-L3079) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3077-L3079)). The load-bearing constraint is that the dependent's base must be filled in place, not swapped for a fresh pointer, because the dependent already aliases the original by pointer; swapping would leave it observing the empty original. Value-kind bases (PrimitiveType) return false and are skipped, since they are immutable and never alias.

Verified on `093c32be0`: the round-1 empty-base repro (which panicked on `61fc396e4`) now prints `9`; a runtime probe of the finalized types confirms `T1.Base == T2.Base` (same pointer, 2 fields, struct typeid `struct{Next *test.T2;Val int}`) while the named types stay distinct (`test.T1` vs `test.T2`, typeids unequal). See [repro](#repro).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

None.

## Missing Tests

None blocking. The PR filetest now reads a field through `T2` (`a.Next.Val` where `a.Next` is `*T2`), the exact access that panicked on `61fc396e4`, so the core regression is pinned. See the Suggestion for optional extra coverage.

## Suggestions

- [`gnovm/tests/files/decltype_mutual.gno:10-13`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/tests/files/decltype_mutual.gno#L10-L13) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/tests/files/decltype_mutual.gno#L10-L13) — the filetest covers the T1-first order and a pointer-field read; the declaration-order-swapped variant (`type T2 T1` before `type T1`) and a direct `var b T2; b.Val` value read are not pinned.
  <details><summary>details</summary>

  Both extra shapes work today (verified on `093c32be0`: order-swap and direct-`T2`-value read both print correctly; see [`tests/decltype_mutual_order_swap.gno`](tests/decltype_mutual_order_swap.gno) and [`tests/decltype_derived_empty_base.gno`](tests/decltype_derived_empty_base.gno)). Adding the swapped order and a value-access line to the filetest would lock in that both ends of the cycle and both declaration orders agree. Optional; the core path is already covered.
  </details>

## Open questions

- Was the `should not happen` panic ever observed firing in the wild? Not load-bearing for the verdict; the fix is correct regardless, so not posted.

### Repro

The round-1 Critical repro, now passing on `093c32be0`. Pairs the field read (the bug shape) with a direct `T2` value read.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5763 -R gnolang/gno
cat > gnovm/tests/files/decltype_derived_empty_base.gno <<'EOF'
package main

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

func main() {
	var a T1
	a.Next = &T2{Val: 9}
	println(a.Next.Val)
	var b T2
	b.Val = 7
	println(b.Val)
	println("ok")
}

// Output:
// 9
// 7
// ok
EOF
go test -v -run 'TestFiles/decltype_derived_empty_base.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/decltype_derived_empty_base.gno
```

```
--- PASS: TestFiles/decltype_derived_empty_base.gno (0.00s)
PASS
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang
```

On `61fc396e4` the same test panicked `struct type struct{} has no field Val`; on `093c32be0` it prints `9` then `7`. Saved with a standalone `/* Run: */` header as [`tests/decltype_derived_empty_base.gno`](tests/decltype_derived_empty_base.gno); the order-swapped companion is [`tests/decltype_mutual_order_swap.gno`](tests/decltype_mutual_order_swap.gno).

## Invariant catalog walk

- VM semantics vs Go: covered. `type T1 .../type T2 T1` evaluates as in Go across declaration orders, multi-level chains (`type T3 T2`), conversions (`T2(t1)`), defined-type method non-inheritance, type-switch identity, and equality, all verified against a side-by-side Go run.
- Type-check & preprocess: covered. Legal mutual recursion resolves; an illegal finite-size value-cycle (`type T1 struct{Self T2}; type T2 T1`) is still rejected, matching Go's `invalid recursive type`.
- Realm state safety / persistence: spot-checked. The full `zrealm*` serialization filetests pass; the shared base pointer does not break object persistence, and sibling derived types keep independent values.
- Determinism: `fillTypeInPlace` is a straight type switch with no map iteration; TypeID of the named type is name-derived (not Base-derived), so the in-place fill cannot perturb it.
- All other catalog classes: the diff does not touch them (no gas, coin/banker, storage deposit, caller/access control, global mutable state, error/panic-recoverability changes).

## Round-1 disposition

- Critical (empty-base corruption, `preprocess.go:3066`): RESOLVED by `fillTypeInPlace`; round-1 repro now passes.
- Warning (test does not exercise the fix): RESOLVED; the filetest now reads `a.Next.Val` through `*T2`.
- Nit (comment pointed at `declareWith`): RESOLVED; comment now reads `tmp2.Seal()` ([`preprocess.go:5507`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L5507) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L5507)).
- Missing Tests (empty-base regression guard): RESOLVED for the core path; extra order-swap/value-read coverage downgraded to the Suggestion above.
- Suggestion (provisional-base comment): no longer applicable; the TRANS_LEAVE handler now carries an explanatory comment for the in-place fill at [`preprocess.go:3070-3079`](https://github.com/gnolang/gno/blob/093c32be0/gnovm/pkg/gnolang/preprocess.go#L3070-L3079) · [↗](../../../../../.worktrees/gno-review-5763/gnovm/pkg/gnolang/preprocess.go#L3070-L3079).

Note: the PR diff against `61fc396e4` appears to add a `comparable uint8` field to `StructType` in `types.go`; that field and its `type_check.go` use predate this branch and arrived via a master merge, not PR content.
