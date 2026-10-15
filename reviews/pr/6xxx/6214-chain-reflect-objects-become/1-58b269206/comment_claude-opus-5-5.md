# Review: [#6214](https://github.com/gnolang/gno/pull/6214)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. One persisted object yields several unequal `reflect.Object` values depending on the pointer type passed to `Of`, which breaks the documented equality and map-key guarantee of a stdlib this branch introduces.
Model: claude-opus-5-5, standard review
Commit: 58b2692067d6545410b99c71f33bdf506caadaa0
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6214 58b269206`
Round: 1. 8 finders, one reflector, 22 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 2 refuted, one of them the reflector's Warning.

## gnovm/stdlibs/chain/reflect/reflect.go:37 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L37) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L37) · Warning
`declaredTypeName(v)` takes `typ` from [the caller's pointer type](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L128), so `Of(tok)`, `Of((*Alias)(tok))` and `Of((*struct{ n int })(tok))` share an address yet compare unequal. A realm keying a map by `Object` then counts one object once per pointer type, against [the promise](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.gno#L47-L48) that two Objects naming one object are equal.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6214 -R gnolang/gno
cat > gnovm/tests/files/zrealm_reflect_eq_repro.gno <<'GNO'
// PKGPATH: gno.land/r/demo/b3eq
package b3eq

import "chain/reflect"

type Token struct{ n int }
type Alias Token

var tok = &Token{n: 1}

func main(cur realm) {
	a, _ := reflect.Of(tok)
	b, _ := reflect.Of((*Alias)(tok))
	c, _ := reflect.Of((*struct{ n int })(tok))
	println("same address:", a.Address() == b.Address(), a.Address() == c.Address())
	println("a == b:", a == b, "a == c:", a == c)
	println("types:", a.Type(), "|", b.Type(), "|", c.Type())
	seen := map[reflect.Object]bool{a: true, b: true, c: true}
	println("map entries for one object:", len(seen))
}

// Output:
// same address: true true
// a == b: true a == c: true
// types: gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Token
// map entries for one object: 1
GNO
cat > gnovm/tests/files/zrealm_reflect_conv_repro.gno <<'GNO'
// PKGPATH: gno.land/r/demo/reflectconv
package reflectconv

import (
	"chain/reflect"
	"strings"
)

type Mine strings.Builder

var b = &strings.Builder{}

func main(cur realm) {
	x1, _ := reflect.Of(b)
	x2, _ := reflect.Of((*Mine)(b))
	println("cross-package equal:", x1 == x2)
	println("type via Mine:", x2.Type())
}

// Output:
// cross-package equal: true
// type via Mine: strings.Builder
GNO
(cd gnovm/pkg/gnolang && go test -run 'TestFiles/zrealm_reflect_(eq|conv)_repro.gno$' -count=1 .)
rm gnovm/tests/files/zrealm_reflect_eq_repro.gno gnovm/tests/files/zrealm_reflect_conv_repro.gno
```

Both tests fail because one object, reached through several pointer types, gives unequal Objects, one map key per type, and a `Type()` naming the type the caller converted to, a realm's own `Mine` over `strings.Builder` included:

```
--- FAIL: TestFiles/zrealm_reflect_eq_repro.gno (0.01s)
            -a == b: true a == c: true
            -types: gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Token
            -map entries for one object: 1
            +a == b: false a == c: false
            +types: gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Alias |
            +map entries for one object: 3
--- FAIL: TestFiles/zrealm_reflect_conv_repro.gno (0.34s)
            -cross-package equal: true
            -type via Mine: strings.Builder
            +cross-package equal: false
            +type via Mine: gno.land/r/demo/reflectconv.Mine
FAIL
```
</details>

## docs/resources/gno-stdlibs.md:748 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/docs/resources/gno-stdlibs.md?plain=1#L748) · [↗](../../../../../.worktrees/gno-review-6214/docs/resources/gno-stdlibs.md#L748) · Nit
Nit: `PkgPath` is not always the creating realm. [`assignNewObjectID`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/realm.go#L2004-L2010) re-stamps an object built by a `maketx run` script, or [by a stdlib](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/realm.go#L2012-L2026), to the realm that persists it.

## gnovm/adr/chain_reflect_object_identity.md:164 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/adr/chain_reflect_object_identity.md?plain=1#L164) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/adr/chain_reflect_object_identity.md#L164) · Nit
Nit: the generated binding is [`gno.X("any")`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/generated.go#L1210), not `gno.AnyT()`, because [`GnoTypeExpression`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/misc/genstd/mapping.go#L39-L41) returns `gno.AnyT()` only for a type spelled `interface{}`.

## gnovm/stdlibs/chain/reflect/reflect.go:26 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L26) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L26) · Nit
Nit: `!oid.IsFinalized()` is the whole guard, as [the godoc](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L210) advises, while [`DeriveObjectCryptoAddr`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L218) also panics on a zero `PkgID`. No stamp sets `NewTime` on a zero `PkgID` today.

## gnovm/stdlibs/chain/reflect/reflect.go:104 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L104) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L104) · Nit
Nit: this `m.Store == nil` block is dead code to delete. [`NewMachineWithOptions`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/machine.go#L185-L188) always sets a store, and [`ownObject`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L69) reads `m.Store` unguarded first.

## gnovm/stdlibs/native_gas.go:157 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/native_gas.go#L157) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/native_gas.go#L157) · Nit
Nit: the comment says nothing scales with caller input, yet [`declaredTypeName`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L143) builds a string that grows with the declared type name on every call.

## gnovm/pkg/gnolang/misc.go:222 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L222) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/pkg/gnolang/misc.go#L222) · Suggestion
Suggestion: build the preimage from the hashlet bytes, or document its spelling. [`PkgID.String()`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/realm.go#L74-L75) is uppercase with an `RID` prefix, while `Object.ID()` is [lowercase hex](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/ownership.go#L47-L49), so hashing `objectid:` plus `ID()` gives another address.

## gnovm/stdlibs/chain/reflect/reflect.go:65 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L65) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L65) · Suggestion
Suggestion: give a stored map an identity with a `case *gno.MapValue` arm, or name the real reason for leaving it out. [`TypedValue.Copy`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/values.go#L1339-L1352) shares one `*MapValue` between holders, so a map is never a by-value copy.

## gnovm/stdlibs/chain/reflect/reflect.go:83 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L83) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L83) · Suggestion
Suggestion: report `HasIdentity` false for an unpersisted method value, or document that it needs storing first. Every `tok.Method` [allocates a fresh `BoundMethodValue`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/values.go#L2253-L2260), so asking again after the call never yields an address.

## gnovm/stdlibs/chain/reflect/reflect.go:113 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L113) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L113) · Suggestion
Suggestion: decide whether `p/` and stdlib objects get an identity, and document it. [`GetRealmByID`](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/store.go#L495-L503) resolves any package, so `Of(strings.ToUpper)` answers `ok` with `PkgPath` `strings`, an answer that [cannot be withdrawn](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L43-L45) later.

## gnovm/stdlibs/chain/reflect/reflect.go:143 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L143) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.go#L143) · Suggestion
Suggestion: this concatenation grows with the declared type name, while [the `objectInfo` row](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/native_gas.go#L157) charges a flat base with no `SizeReturnLen` post-slope.

## SKIP gnovm/pkg/gnolang/misc.go:217 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L217) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/pkg/gnolang/misc.go#L217) · Suggestion
Suggestion: `DeriveObjectCryptoAddr` exports a preimage that another open change exports as `DeriveObjectIDCryptoAddr`, so whichever lands second carries two exported names for one function.

Not posted: it depends on a symbol name a sibling pull request also defines, which the two changes have to settle together.

## SKIP gnovm/cmd/calibrate/reflect_bench_test.go:54 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/cmd/calibrate/reflect_bench_test.go#L54) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/cmd/calibrate/reflect_bench_test.go#L54) · Missing test
Missing test: the stamped benchmark leaves `T` nil, so `declaredTypeName` returns `""` at its first type assertion and the run that sets the 1200 price never builds a type name.

Not posted: PLAUSIBLE, no verifier ran the planted-panic check.

## SKIP gnovm/tests/files/zrealm_reflect_foreign_finalize.gno:30 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_foreign_finalize.gno#L30) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/tests/files/zrealm_reflect_foreign_finalize.gno#L30) · Missing test
Missing test: `theirs` is a package variable that this realm stamps at load, before `SetFooer` runs, so the fixture never reaches `touchForeignRealm`.

Not posted: PLAUSIBLE, no verifier ran the planted-panic check.

## SKIP gnovm/tests/files/zrealm_reflect_identity.gno:49 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_identity.gno#L49) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/tests/files/zrealm_reflect_identity.gno#L49) · Missing test
Missing test: no fixture pins bound method values, `new(T)`, `&packageVar` or stdlib func values. Deleting the `*gno.BoundMethodValue` arm keeps the suite green.

Not posted: PLAUSIBLE, no verifier ran the deletion check.

## SKIP gnovm/adr/chain_reflect_object_identity.md:50 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/adr/chain_reflect_object_identity.md?plain=1#L50) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/adr/chain_reflect_object_identity.md#L50) · Nit
Nit: the address preimage is `objectid:RID<UPPERHEX>:<n>`, not `objectid:` plus the `ID()` spelling offered for storage dumps.

Not posted: UNVERIFIED, on the finder's read only, and the `misc.go:222` section carries the same mismatch.

## SKIP gnovm/stdlibs/chain/reflect/reflect.gno:184 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.gno#L184) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/stdlibs/chain/reflect/reflect.gno#L184) · Suggestion
Suggestion: `PkgPath` of a `p/` or stdlib package-level object names that package rather than a realm.

Not posted: UNVERIFIED, and the `reflect.go:113` section carries the decision.

## SKIP gnovm/tests/files/zrealm_reflect_proposal_fund.gno:8 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_proposal_fund.gno#L8) · [↗](../../../../../.worktrees/gno-review-6214/gnovm/tests/files/zrealm_reflect_proposal_fund.gno#L8) · Suggestion
Suggestion: the example uses an object address as a proposal's fund without saying that coins sent there can never be released.

Not posted: UNVERIFIED, on the finder's read only.

## SKIP gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:196 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L196) · [↗](../../../../../.worktrees/gno-review-6214/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L196) · Suggestion
Suggestion: adding the `chain/reflect` stdlib moves the genesis app hash, so a running chain needs an upgrade path that loads a stdlib absent from its state.

Not posted: UNVERIFIED, and the line sits outside the diff, so it would go in the Body only.
