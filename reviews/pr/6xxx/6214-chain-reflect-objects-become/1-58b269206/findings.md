# Findings in posting order, from round assemble: 13 to post, 7 SKIP, 1 refuted kept out

## Body

- SKIP Suggestion: gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:196 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L196), outside the diff, so no anchor
  State: UNVERIFIED, band: Suggestion, angle: catalog (consensus/upgrade), on the finder's read only
  TL;DR: Adding the chain/reflect stdlib moves the genesis app hash (the test's own expected hash is bumped), so a running chain needs a coordinated upgrade path for a stdlib absent from its state; the PR names no such path
  Check: grep how stdlibs are written to state (gno.land/pkg/sdk/vm keeper LoadStdlib / initStdlibs) and whether an existing store missing a stdlib loads it on demand; run an addpkg importing chain/reflect against a store initialised at the base commit
  Evidence: not run: on the finder's read

## gnovm/stdlibs/chain/reflect/reflect.go:37 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L37) · Warning
State: CONFIRMED, band: Warning, angle: catalog + reach
TL;DR: Converting the pointer before calling Of gives one persisted object two unequal Objects and a type chosen by the caller, so dedup or map keys by Object can count it twice
Check: cp tests/b1-reach-conversion-splits-object.gno gnovm/tests/files/zrealm_reflect_conversion.gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect_conversion.gno' -count=1: expect 'same object equal: true', observe false | cp tests/b1-catalog-object-type-varies.gno gnovm/tests/files/zrealm_reflect_b1_typevaries.gno && go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect_b1_typevaries' -v; the pinned Output passes with `a == b: false a == c: false` and `distinct keys for one object: 3`
Details: X_objectInfo derives id and addr from the heap item that ownObject resolves, but it takes typ from declaredTypeName(v) (reflect.go:37), which reads tv.T, the static type of the pointer the caller passed. So Of(tok) and Of((*Alias)(tok)) carry the same id and address but different typ. Object is a struct of four strings compared field by field, so the two compare unequal. This breaks the doc's promise at reflect.gno:47, 'Two Objects are equal if and only if they name the same object' and 'safe as a map key'. The same thing happens across packages, with type Mine strings.Builder over a *strings.Builder. Fix: take typ from the heap item's own value type (hiv.Value.T) rather than the argument's static type, or drop typ from the compared struct, and pin the equality with a filetest.
Evidence: In a scratch worktree at 58b269206, go test -run 'TestFiles/zrealm_reflect_conversion.gno$' with the finder's file failed with the actual output '+same object equal: false', '+map entries for one object: 2', '+type via alias: gno.land/r/demo/reflectconv.Alias', '+cross-package equal: false', '+type via Mine: gno.land/r/demo/reflectconv.Mine'. Merge base: gnovm/stdlibs/chain/reflect does not exist, so the PR introduces this.
Artifact: tests/b1-reach-conversion-splits-object.gno

## gnovm/stdlibs/chain/reflect/reflect.go:128 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L128) · Warning
State: CONFIRMED, band: Warning, angle: claims
TL;DR: The documented equality and map-key guarantee fails: one object read through *Token, *Alias and *struct{ n int } gives three Objects with one address
Check: cp tests/b3-claims-object-equality.gno gnovm/tests/files/zrealm_b3claims_equality.gno && cd gnovm/pkg/gnolang && go test -run 'TestFiles/zrealm_b3claims_equality' -count=1 . ; it passes at head with 'a == b: false a == c: false' and 'map entries for one object: 3'
Details: declaredTypeName begins with t := tv.T (reflect.go:128), the argument's static pointer type, so the typ field follows whatever pointer type the caller converted to. A conversion to an unnamed struct pointer even gives typ "". The three Objects have the same Address() and ok is true for each, yet they compare unequal and take three slots in a map[reflect.Object]bool. That contradicts 'Two Objects are equal if and only if they name the same object' at reflect.gno:47. A realm that registers or dedupes by Object therefore admits the same object again under each pointer type. This is the same root cause as candidate 6 and is to be reconciled with it. Fix: read the type from the heap item's own TypedValue in X_objectInfo, which also makes the type P *Token unwrap unnecessary.
Evidence: In a scratch worktree at 58b269206, go test -run 'TestFiles/zrealm_b3claims_equality.gno$' -count=1 -v with the finder's file passed against the pinned Output 'ok: true true true' / 'same address: true true' / 'a == b: false a == c: false' / 'types: gno.land/r/demo/b3eq.Token | gno.land/r/demo/b3eq.Alias |' / 'map entries for one object: 3'. Merge base: gnovm/stdlibs/chain/reflect does not exist.
Artifact: tests/b3-claims-object-equality.gno

## SKIP gnovm/cmd/calibrate/reflect_bench_test.go:54 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/cmd/calibrate/reflect_bench_test.go#L54) · Missing test
State: PLAUSIBLE, band: Missing test, angle: catalog + lines + reach, on the finder's read only
TL;DR: the stamped bench that sets objectInfo's flat 1200 price passes a TypedValue with no T, so declaredTypeName returns early and the typed path every production call takes is never measured | the stamped benchmark passes the param with a nil T, so declaredTypeName returns "" at its first type assertion and the run that prices the native never builds the type name | the Stamped bench fixture sets V only and leaves T nil, so declaredTypeName returns "" at its first type assertion and the price never includes the type-name step the header says it measures
Check: add a bench case with T: &gno.PointerType{Elt: &gno.DeclaredType{PkgPath: reflectBenchPkgPath, Name: "Account"}} and a cold-cache variant without tx.GetPackageRealm; go test ./gnovm/cmd/calibrate -run x -bench Reflect_ObjectInfo; compare ns/op against the 1196ns behind Base 1200 | in a scratch worktree set T: &gno.PointerType{Elt: &gno.DeclaredType{PkgPath: reflectBenchPkgPath, Name: "Token", Base: &gno.StructType{}}} on line 54, then go test ./gnovm/cmd/calibrate -run '^$' -bench 'Reflect_ObjectInfo_Stamped' -count 5 before and after; also plant panic("reached") after the PointerType assertion in declaredTypeName and confirm the unmodified bench stays green | plant panic("reached") after the PointerType assertion in declaredTypeName (reflect.go:133) and run go test ./gnovm/cmd/calibrate -run '^$' -bench Reflect_ObjectInfo_Stamped: expect a panic, observe the bench stays green
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gnovm/tests/files/zrealm_reflect_foreign_finalize.gno:30 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_foreign_finalize.gno#L30) · Missing test
State: PLAUSIBLE, band: Missing test, angle: claims + reach, on the finder's read only
TL;DR: The foreign-finalize filetest never reaches touchForeignRealm: 'theirs' is a package var, stamped by this realm at package load before SetFooer runs | the foreign-finalize fixture hands crossrealm an object its own init finalize already stamped, so the foreign tick branch it claims to exercise never runs
Check: Plant panic("touchForeignRealm reached") before targetRlm = rlm.touchForeignRealm(store, oid.PkgID) at gnovm/pkg/gnolang/realm.go:2038, then go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect_foreign_finalize' -count=1: expect green, so the test pins nothing. Move theirs into main and re-run: expect the sentinel to fire. | at 58b269206: sed -i '2038s/targetRlm = rlm.touchForeignRealm(store, oid.PkgID)/_ = rlm.touchForeignRealm(store, oid.PkgID)/' gnovm/pkg/gnolang/realm.go; cd gnovm/pkg/gnolang && go test . -v -run 'TestFiles/zrealm_reflect_(foreign_finalize|zprobe_foreign)': expect the PR fixture red, observe PASS; the probe (fresh object created in main, handed to crossrealm) goes red
Evidence: not run: no verifier reached it; the check is still to run

## SKIP gnovm/tests/files/zrealm_reflect_identity.gno:49 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_identity.gno#L49) · Missing test
State: PLAUSIBLE, band: Missing test, angle: reach, on the finder's read only
TL;DR: the kinds fixture pins three true shapes; bound method values, new(T), &packageVar and /p/ or stdlib func values answer ok=true with no test, though the ADR says no answer may ever be withdrawn
Check: delete the `case *gno.BoundMethodValue:` arm in gnovm/stdlibs/chain/reflect/reflect.go, then go test ./gnovm/stdlibs/chain/reflect && go test ./gnovm/pkg/gnolang -run 'TestFiles/zrealm_reflect': expect red, observe green; the shapes probe goes red
Evidence: not run: no verifier reached it; the check is still to run

## docs/resources/gno-stdlibs.md:748 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/docs/resources/gno-stdlibs.md#L748) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: Docs say PkgPath is the creating realm, but the VM re-stamps adopted objects to the realm that persists them
Check: Add a txtar under gno.land/pkg/integration/testdata with a realm r/foo that has an exported Token type and a crossing Register(cur realm, t *Token) storing t. Run gnokey maketx run with a script doing foo.Register(cross, &foo.Token{}), then a call in r/foo printing reflect.Of(stored).PkgPath(). Expect gno.land/r/foo although the run script created the object.
Details: assignNewObjectID re-stamps the PkgID to the finalizing realm when the PkgID is zero, belongs to an ephemeral /e/.../run realm, or belongs to a stdlib. creatingRealmPath resolves that re-stamped PkgID. So an object built in a maketx run script, or returned by a stdlib helper, reports the adopting realm as its PkgPath, which the docs line calls the creator. reflect.gno:178-179 already adds 'the realm whose storage clock minted its identity and which owns its persistence', but the docs page keeps only the 'created' half. Fix: in gno-stdlibs.md, describe PkgPath as the realm that minted the identity and owns the object, and say that adopted objects report the adopter.
Evidence: Read only. realm.go:2005-2013: `} else if oid.PkgID != rlm.ID && isPkgEphemeralFromPkgID(store, oid.PkgID) { ... adopt it: re-stamp PkgID to the persisting realm ... oo.SetPkgID(rlm.ID)`; same for zero PkgID (1999-2004) and stdlib PkgID (2014-2029). reflect.go creatingRealmPath: `rlm := m.Store.GetRealmByID(oid.PkgID)`. Docs line 748: '`PkgPath` is the realm that created the object.' Not run: no txtar was built under the read-only contract.

## SKIP gnovm/adr/chain_reflect_object_identity.md:50 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/adr/chain_reflect_object_identity.md#L50) · Nit
State: UNVERIFIED, band: Nit, angle: claims, on the finder's read only
TL;DR: The consensus preimage is objectid:RID<UPPERHEX>:<n> through PkgID.String(), not objectid:<ID()>, so an address cannot be recomputed from the ID() spelling the docs offer for storage dumps
Check: Read misc.go:222, realm.go:74-75 and ownership.go:47-49; or in a Go test print gno.DeriveObjectCryptoAddr(oid) next to crypto.AddressFromPreimage([]byte("objectid:"+oid.String())) for one oid: expect two different addresses
Evidence: not run: on the finder's read

## gnovm/adr/chain_reflect_object_identity.md:164 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/adr/chain_reflect_object_identity.md#L164) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: the ADR says the binding maps any to gno.AnyT(), but the generated binding is gno.X("any")
Check: grep -n 'chain/reflect' -A22 gnovm/stdlibs/generated.go | grep p0 ; expect gno.X("any"), not gno.AnyT()
Details: GnoTypeExpression returns gno.AnyT() only when the type expression renders as "interface{}" (misc/genstd/mapping.go:39-41), and the Gno-side `any` renders as "any", so generated.go:1210 carries `Type: gno.X("any")`. The ADR's Consequences bullet names code that does not exist. Fix: change the ADR to gno.X("any").
Evidence: generated.go:1210 `{NameExpr: *gno.Nx("p0"), Type: gno.X("any")},`; mapping.go:40-41 `if s == "interface{}" { return "gno.AnyT()" }`; ADR:163-164 '`GnoTypeExpression` maps it to `gno.AnyT()`'

## gnovm/stdlibs/chain/reflect/reflect.go:26 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L26) · Nit
State: CONFIRMED, band: Nit, angle: catalog
TL;DR: the godoc says IsFinalized is enough to avoid the panic, but the guard also panics on a zero PkgID; that case cannot happen today
Check: grep every SetNewTime/SetObjectID site in gnovm/pkg/gnolang for one that stamps NewTime with a zero PkgID
Details: DeriveObjectCryptoAddr panics on `!oid.IsFinalized() || oid.PkgID.IsZero()`, and its godoc tells callers to check IsFinalized, which is only `NewTime != 0`. The only NewTime stamps outside unmarshal are realm.go:2031 and :2041 inside assignNewObjectID, after the zero-PkgID fix-up at :1995-2003. So X_objectInfo never hits the second half, and the only cost is godoc advice that covers half the guard. Fix: say in the godoc that both halves must be set, or have X_objectInfo check PkgID.IsZero too.
Evidence: grep SetNewTime/.NewTime = in gnovm/pkg/gnolang non-test: ownership.go:65 (UnmarshalAmino), ownership.go:296 (setter), realm.go:2031, realm.go:2041; realm.go:1995-2003 `if oid.PkgID.IsZero() { oo.SetPkgID(rlm.ID) ...}`; ownership.go:107-108 IsFinalized `return oid.NewTime != 0`

## gnovm/stdlibs/chain/reflect/reflect.go:104 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L104) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: The m.Store == nil guard in creatingRealmPath can never fire, and ownObject already uses m.Store unguarded
Check: sed -n 185,204p gnovm/pkg/gnolang/machine.go; grep -n 'SetNewTime(' gnovm/pkg/gnolang/*.go (only realm.go:2031, 2041 inside assignNewObjectID)
Details: NewMachineWithOptions sets a bare store whenever opts.Store is nil, then assigns mm.Store = store. In non-test code no other line assigns Machine.Store. X_objectInfo calls ownObject first, which runs cv.GetBase(m.Store) without a guard, so a nil store would already be handled or fail there before creatingRealmPath runs. The guard is dead code that looks like protection. Fix: delete the `if m.Store == nil { return "" }` block in creatingRealmPath (reflect.go:104-106), which is the only site.
Evidence: Read only. machine.go:185-188: `store := opts.Store; if store == nil { // bare store, no stdlibs. store = NewStore(alloc, nil, nil) }`; machine.go:204 `mm.Store = store`; grep 'm.Store = |mm.Store = |.Store = nil' over gnovm/pkg/gnolang/*.go non-test: only machine.go:204. reflect.go ownObject: `hiv, ok := cv.GetBase(m.Store).(*gno.HeapItemValue)` unguarded.

## gnovm/stdlibs/native_gas.go:157 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/native_gas.go#L157) · Nit
State: CONFIRMED, band: Nit, angle: lines + reach
TL;DR: The row comment's 'nothing here scales with caller input' is false, since the type name is concatenated per call, but gas tracks the extra work within about 1.5x, not several times
Check: bash tests/b2-reach-typename-flat-gas.sh from a clone at 58b269206: compare per-iteration gas and wall deltas of short vs 60 KB type name; also the Go bench tests/b2-reach-typename-flat-gas_test.go copied into gnovm/cmd/calibrate, -bench 'Reflect_ObjectInfo_(Stamped$|TypeName)'
Details: declaredTypeName returns dt.PkgPath + "." + string(dt.Name) on every stamped call (reflect.go:143), so the work grows with a type name the caller declares, while the objectInfo row is SizeFlat and its comment says nothing scales with caller input. The returned string is charged allocation gas, so gas does rise with the name: +9,740 gas per call for a 60 KB name. Three reruns of the finder's script measured 0.62-0.63 ns/gas for the short name and 0.71-0.91 ns/gas for the 60 KB name, about 0.84-1.25 ns/gas at the margin. That gap is within 1.5x of the base rate and does not reproduce the finder's 2.3-3.6 ns/gas or the claim that a block runs several times longer than its gas. What remains is a row comment that misdescribes the native. Fix: reword the native_gas.go row comment to say the type-name string grows with the declared name and is covered by allocation gas, or have the calibrate bench add a long-name case.
Evidence: bash tests/b2-reach-typename-flat-gas.sh at 58b269206, three runs: short0 gas=18934, short20000 gas=249923693, big0 gas=378913, big20000 gas=445083700 (gas identical to the finder's); wall ms short0/short20000/big0/big20000 = 51/205/89/406, 57/215/58/457, 58/216/57/462. Per iteration: short 7.7-7.9 us over 12,495 gas, big 15.9-20.2 us over 22,235 gas. Merge base: gnovm/stdlibs/chain/reflect does not exist.
Artifact: tests/b2-reach-typename-flat-gas.sh

## gnovm/pkg/gnolang/misc.go:217 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L217) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: the two coupled PRs export one preimage under two names, so landing both leaves duplicate exported derivers
Check: gh pr diff 6139 -R gnolang/gno | grep -n 'func DeriveObject' ; grep -n 'func DeriveObject' gnovm/pkg/gnolang/misc.go in the head worktree
Details: Head declares `func DeriveObjectCryptoAddr(oid ObjectID)` at misc.go:217 and #6139 declares `func DeriveObjectIDCryptoAddr(objectID ObjectID)` with the identical `"objectid:" + PkgID.String() + ":" + strconv.FormatUint(NewTime, 10)` body. The godoc's 'interchangeable' holds for the address but not for callers: whichever PR lands second conflicts or carries a second exported name for one function. Fix: agree on one symbol name across #6214 and #6139 before either merges.
Evidence: gh pr diff 6139: line 2075 `+func DeriveObjectIDCryptoAddr(objectID ObjectID) crypto.Address {`, line 2080 same preimage; head misc.go:217 `func DeriveObjectCryptoAddr(oid ObjectID) crypto.Address {`

## gnovm/pkg/gnolang/misc.go:222 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/pkg/gnolang/misc.go#L222) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog + reach + reach (ephemeral run realm)
TL;DR: hashing the id that Object.ID() returns, as the docs suggest, does not give the chain's address, and the consensus preimage depends on a debug Stringer
Check: read misc.go:222, realm.go:74-76, ownership.go:45-48, reflect.gno:60 | go test snippet comparing AddressFromPreimage("objectid:"+oid.String()) against DeriveObjectCryptoAddr(oid)
Details: The preimage uses PkgID.String(), which is `fmt.Sprintf("RID%X", pid.Hashlet[:])` (realm.go:74-75), uppercase with an RID prefix. X_objectInfo returns oid.String() as the id, and ObjectID.String() is MarshalAmino, `hex.EncodeToString(oid.PkgID.Hashlet[:])` lowercase with no prefix (ownership.go:47-48, 69-71). reflect.gno:60 documents addr as 'bech32, derived from id', so an indexer hashing `objectid:`+ID() gets a different address. Shared with #6139. Fix: build the preimage from the hashlet bytes or from oid.MarshalAmino rather than a fmt Stringer, or document the exact spelling.
Evidence: realm.go:74-75 `return fmt.Sprintf("RID%X", pid.Hashlet[:])`; ownership.go:47 `pid := hex.EncodeToString(oid.PkgID.Hashlet[:])`, :69-70 ObjectID.String returns MarshalAmino; misc.go:222 uses `oid.PkgID.String()`; reflect.go:34 returns `oid.String()` as id

## SKIP gnovm/stdlibs/chain/reflect/reflect.gno:184 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.gno#L184) · Suggestion
State: UNVERIFIED, band: Suggestion, angle: reach, on the finder's read only
TL;DR: PkgPath of a /p/ or stdlib package-level object names that package, not a realm, against its doc 'the path of the realm that created the object'
Check: gno test a filetest printing reflect.Of(io.EOF).PkgPath() and reflect.Of(strings.ToUpper).PkgPath(): observed 'io' and 'strings'
Evidence: not run: on the finder's read

## gnovm/stdlibs/chain/reflect/reflect.go:65 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L65) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: a stored map gets no identity although every holder shares one *MapValue, and the documented reason for excluding it (a by-value copy) does not apply to maps
Check: run the shapes probe: `map ok: false has: false`; read values.go:1339-1352 for the copy cases
Details: ownObject handles only PointerValue, *FuncValue and *BoundMethodValue, so a *MapValue falls to `return nil`. TypedValue.Copy deep-copies only BigintValue, *ArrayValue and *StructValue and unrefCopy only arrays and structs; a map hits `default: cp = tv`, so it is a reference exactly like the func values the comment admits. reflect.gno:106 and reflect.go:55 group map with struct and array as 'may be ... a copy', which is false for maps. Fix: either add a `case *gno.MapValue` arm alongside FuncValue, or reword the exclusion so it names the real reason maps are left out.
Evidence: values.go Copy: `case BigintValue / *ArrayValue / *StructValue ... default: cp = tv`; unrefCopy copies only *ArrayValue and *StructValue; finder's probe output line 64 `map             ok: false has: false pkg:  type:  str: <no object>`
Artifact: tests/b4-lines-removed-reach-catalog-identity-shapes.gno

## gnovm/stdlibs/chain/reflect/reflect.go:83 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L83) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines + claims
TL;DR: a realm following the docs on `tok.Method` waits for an address that never comes, because each evaluation is a fresh unstamped object that reports HasIdentity true
Check: filetest with persisted tok and Method; print reflect.Of(tok.Method) ok and HasIdentity before and after a crossing call; expect 'ok: false has: true' both times
Details: Each evaluation of a method selector allocates a new `&BoundMethodValue{...}` and stamps only its PkgID (values.go:2253-2260), so NewTime stays 0, X_objectInfo takes the `!oid.IsFinalized()` branch and returns ok=false, hasIdentity=true. Asking again after the boundary evaluates tok.Method again and gets another fresh value, so the 'ask again after the call returns' answer in reflect.gno:70-74 never resolves. The method-expression half ((*Token).Method) was not traced. Fix: report HasIdentity false for a BoundMethodValue that is not persisted, or document that a method value only gets an identity once it is stored.
Evidence: values.go:2253-2260 `alloc.AllocateBoundMethod(); bmv := &BoundMethodValue{Func: mv, Receiver: rtv}; ... alloc.stampPkgID(&bmv.ObjectInfo, nil)`; reflect.go:26-31 returns `"", "", "", "", false, true` when !IsFinalized; ownership.go:107-108 IsFinalized is `oid.NewTime != 0`

## gnovm/stdlibs/chain/reflect/reflect.go:113 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L113) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: Of(strings.ToUpper) returns ok with an address and PkgPath "strings", although no realm created that object
Check: tests/b1-lines-probe-shapes.gno: copy to gnovm/tests/files, run with -update-golden-tests, read the strings.ToUpper Output line
Details: creatingRealmPath returns rlm.Path for any record GetRealmByID finds, and GetRealmByID (store.go:495-503) resolves any package path through pkgPathFromPkgID and GetPackageRealm with no realm-path check. A stdlib func value is therefore Of-ok, with PkgPath documented as 'the realm that created the object' (reflect.gno:61) naming a stdlib. Because the docs say an answer once given cannot be withdrawn, this decision becomes permanent once it ships. Fix: decide explicitly, either return no identity for objects whose PkgID does not resolve to a realm path or document that package objects of p/ and stdlib packages are included.
Evidence: reflect.go:108-113 `rlm := m.Store.GetRealmByID(oid.PkgID); if rlm == nil { return "" }; return rlm.Path`; store.go:499-503 no IsRealmPath filter; probe Output line 59 `strings.ToUpper ok: true has: true pkg: strings ... (g1q37e4qy20davf35rwzuuu3662fnhf7f3v7npzr)`
Artifact: tests/b1-lines-probe-shapes.gno

## gnovm/stdlibs/chain/reflect/reflect.go:143 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/stdlibs/chain/reflect/reflect.go#L143) · Suggestion
State: CONFIRMED, band: Suggestion, angle: catalog
TL;DR: the flat gas price assumes nothing scales with input, but the returned type name is built at the length of the declared name, so long names run under-charged
Check: Benchmark X_objectInfo on a stamped object with DeclaredType Name of 8 B, 1 KB, 64 KB, 1 MB; compare with 1200 + allocGas(len)
Details: native_gas.go:157 prices objectInfo at Base 1200, SizeFlat, with the comment 'nothing here scales with caller input'. declaredTypeName returns `dt.PkgPath + "." + string(dt.Name)`, an allocation and copy proportional to the identifier length, which the flat row does not bill. The finder measured 827 us at a 1 MB name against roughly 196k gas charged; it only matters at extreme identifier lengths. Fix: charge a SizeReturnLen post-slope on the type-name result, or correct the gas comment.
Evidence: reflect.go:143 `return dt.PkgPath + "." + string(dt.Name)`; native_gas.go:157 `{Pkg: "chain/reflect", Fn: "objectInfo", Base: 1200, SlopeIdx: -1, SlopeKind: SizeFlat}, // ... nothing here scales with caller input`; finder bench /1048576 827394 ns/op
Artifact: tests/b2-reach-typename-flat-gas_test.go

## SKIP gnovm/tests/files/zrealm_reflect_proposal_fund.gno:8 [gh](https://github.com/gnolang/gno/blob/58b2692067d6545410b99c71f33bdf506caadaa0/gnovm/tests/files/zrealm_reflect_proposal_fund.gno#L8) · Suggestion
State: UNVERIFIED, band: Suggestion, angle: lines, on the finder's read only
TL;DR: the flagship example sells an object address as a proposal's fund and renders a funds link, without saying coins sent there can never be released
Check: grep -n 'release\|cannot' gnovm/tests/files/zrealm_reflect_proposal_fund.gno: no warning; sed -n 233,240p gnovm/stdlibs/chain/banker/banker.gno
Evidence: not run: on the finder's read
