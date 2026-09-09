# Review: [#6139](https://github.com/gnolang/gno/pull/6139)
Event: REQUEST_CHANGES

## Body
The `grc20reg` re-key to `rlmPath.slug` is a third breaking change the title names neither half of, and its own commit lands six commits after the identity work was already complete and consistent at 089519b45, so it can go out on its own:

- The key of every registration passing an empty slug moves from `<realm>.<SYMBOL>` to `<realm>`, because [`fqname.Construct(rlmPath, slug)`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L46) returns the path bare: [`wugnot`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L27), [`foo20`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/foo20/foo20.gno#L25) and [`test20`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/tests/vm/test20/test20.gno#L23) in tree, while [`grc20factory`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L51) passes the symbol as the slug and keeps its keys.
- One realm can now register two tokens under one symbol, where master's key was [the overwrite and alias guard](https://github.com/gnolang/gno/blob/d4bb7ab93/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30-L31) against exactly that, and [`grc20reg_test.gno:48-52`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L48-L52) asserts the second `TST` succeeds, so a lookup by symbol has no single answer.
- The symbol left the provenance check: master compared `token.ID()` against `rlmPath + "." + symbol`, [line 42](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L42) compares realm paths alone, and `symbol` still rides in the `register` event bound to nothing.

## gnovm/stdlibs/chain/runtime/native.go:60 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.go#L60) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/chain/runtime/native.go#L60)
[`Register`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59) announces one `token_id` for two ledgers that emit two, the ambiguity [issue 6026](https://github.com/gnolang/gno/issues/6026) exists to remove: a `grc20.Token` copied into a struct field or an array element reports the same [`ID()`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/token.gno#L145-L147) as its sibling copy, since [`GetFirstObject`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L415-L420) answers with the base container for a pointer and the backing array for a slice. Resolve the value itself rather than its container, or reject a value whose resolved object is not that value, and drop the copy sentence from [`native.gno:18-19`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.gno#L18-L19), [`gno-stdlibs.md:690-691`](https://github.com/gnolang/gno/blob/011afff91/docs/resources/gno-stdlibs.md?plain=1#L690-L691) and [the ADR's Consequences](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L108).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
P=examples/gno.land/r/demo/grc20container
mkdir -p $P
printf 'module = "gno.land/r/demo/grc20container"\ngno = "0.9"\n' > $P/gnomod.toml
cat > $P/token_id_container_filetest.gno <<'EOF'
// PKGPATH: gno.land/r/demo/grc20container
package grc20container

import "gno.land/p/demo/tokens/grc20"

type Pair struct {
	A grc20.Token
	B grc20.Token
}

var (
	origA, origB *grc20.Token
	pair         Pair
	arr          [2]grc20.Token
	varA, varB   grc20.Token
)

func init(cur realm) {
	origA, _ = grc20.NewToken("Alpha", "ALPHA", 4, cur)
	origB, _ = grc20.NewToken("Beta", "BETA", 4, cur)

	pair.A, pair.B = *origA, *origB
	arr[0], arr[1] = *origA, *origB
	varA, varB = *origA, *origB
}

func main(cur realm) {
	// The shape NewToken hands back, correctly distinct.
	println("pointers differ:", origA.ID() != origB.ID())
	// One copy per package-level var, the case the doc comment describes.
	println("separate vars differ:", varA.ID() != varB.ID())
	// Two copies inside one container.
	println("struct fields differ:", pair.A.ID() != pair.B.ID())
	println("array elements differ:", arr[0].ID() != arr[1].ID())
}

// Output:
// pointers differ: true
// separate vars differ: true
// struct fields differ: true
// array elements differ: true
EOF
(cd $P && GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v .)
rm -rf $P
```

The two container rows are the finding: each pair of tokens carries its own ledger and answers with one address.

```
pointers differ: true
separate vars differ: true
struct fields differ: false
array elements differ: false
=== RUN   ./token_id_container_filetest.gno
--- FAIL: ./token_id_container_filetest.gno (elapsed: 0.03s, gas: 749286)
Output diff:
--- Expected
+++ Actual
@@ -2,3 +2,3 @@
 separate vars differ: true
-struct fields differ: true
-array elements differ: true
+struct fields differ: false
+array elements differ: false
# … the runner's own FAIL summary lines follow
```

At the GRC20 level with the supplies visible, `pair.A` and `pair.B` holding tokens `ONE` and `TWO` with supplies 111 and 222 both answer `g12vuu8a49g6vdzs7cxsargfkkhdvujsmkdx8ehj`. Without GRC20 in the picture, `&h.A` and `&h.B` of one struct both answer `g1xg33mkdhght44wzdccuns7470ude5cvlns8mxz` and `&arr[0]` and `&arr[1]` both answer `g1g96sx2rwe2jzwzq46hl3u0xr2ed89gkl82dz00`, while `ObjectID(&h)` answers `g18808jrvpvzp89efeegj033ajaf7wxh5c8e9ypc`, so one variable has two names depending on how the pointer is taken. A `[]grc20.Token` behaves the same way: the slice and `&sl[0]` answer with one address, while `&iA`, `&iB` and `&sA`, three vars of one package block, answer with three, so the collapse is bounded to containers and slices.

The derivation function is not involved: `objectid:<pkgid>:<newtime>` is injective in both halves, and every address above is the right address for the object the resolver picked. `origRealm` copies with the struct, so both copies pass `Register`'s provenance check and land as two public entries whose `register` events carry one `token_id`, while the two ledgers' `Transfer` events carry two. At the merge base the same four shapes answer true on all four, since `Token.ID()` there returns the stored `tok.id`.
</details>

## gnovm/stdlibs/native_gas.go:143 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/native_gas.go#L143) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/native_gas.go#L143)
This flat 200 is priced against [`getSessionInfo`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/native_gas.go#L144), which does no hashing, where [`chain.packageAddress`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/native_gas.go#L106) charges exactly 908 gas on a 24-character path for the same truncated-SHA256-plus-bech32 derivation, and the ratio between the two natives puts the floor at 527 for any dispatch envelope. Add the benchmark to the [`chain/runtime` section](https://github.com/gnolang/gno/blob/011afff91/gnovm/cmd/calibrate/native_machine_bench_test.go#L826) every other row in that block was fitted from and refit, or record the borrowed base in the header block with the reason 200 cannot undercharge.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_objid_bench_test.go <<'EOF'
package gnolang

import "testing"

var zzSink string

// DerivePkgBech32Addr is the whole body of chain.packageAddress, whose
// calibrated row charges Base 552 + Slope 15201*N/1024, or exactly 908 gas on
// a 24-character path. ObjectID.DerivePath is the body of the row under
// review, priced flat at 200.
func BenchmarkZZDerivePkgBech32Addr(b *testing.B) {
	for b.Loop() {
		zzSink = string(DerivePkgBech32Addr("gno.land/r/demo/objectid"))
	}
}

func BenchmarkZZObjectIDDerivePath(b *testing.B) {
	oid := ObjectID{PkgID: PkgIDFromPkgPath("gno.land/r/demo/objectid"), NewTime: 7}
	for b.Loop() {
		zzSink = oid.DerivePath()
	}
}
EOF
go test ./gnovm/pkg/gnolang/ -run '^$' -bench 'BenchmarkZZ' -benchtime=3s -count=9
rm gnovm/pkg/gnolang/zz_objid_bench_test.go
```

The row priced at 200 runs at 0.59 of the row charging 908.

```
BenchmarkZZDerivePkgBech32Addr-6   	 1000000	      3177 ns/op
# … seven more samples, median 3077
BenchmarkZZDerivePkgBech32Addr-6   	 1209783	      3040 ns/op
BenchmarkZZObjectIDDerivePath-6    	 2019961	      1795 ns/op
# … seven more samples, median 1819
BenchmarkZZObjectIDDerivePath-6    	 2101833	      1754 ns/op
```

Those medians are single-session figures on a shared box and a re-run does not land on them, so the claim rests on the ratio, which held between 0.581 and 0.622 over four sessions. Writing the dispatch envelope both rows pay as `E`, `908 = E + D_p` puts the fair price at `ratio * 908 + (1 - ratio) * E`, at least 527 for any `E` at or above zero. That is a floor: the harness holds a `*StructValue` in the block, so `GetFirstObject`'s `RefValue` branch, the store read the row's comment says the store charges, never runs.
</details>

## gnovm/adr/prxxxx_objectid_derived_ids.md:101 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L101) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/adr/prxxxx_objectid_derived_ids.md#L101)
Consequences covers the signature change, the registry re-key and the genesis apphash move, and never says the derivation is fixed once an address is referenced: a scan of the whole file for stable, immutable, upgrade, consensus, migrate, rollback, fund, balance and account returns [one line](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L13), the sentence calling objects potential accounts, and [`PkgIDFromPkgPath`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L97) is never named as an input although its [flag nibble](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L103-L116) moves every address in a realm. Add a bullet naming every input to the address and saying that changing one after an address is referenced needs a migration.

## gnovm/stdlibs/chain/runtime/native.gno:13-20 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.gno#L13-L20) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/chain/runtime/native.gno#L13)
This returns a `g1…` a realm author cannot tell from a spendable account, and coins sent to one are locked forever: [`NewBanker`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/banker/banker.gno#L149) binds `pkgAddr` to `rlm.Address()`, which is always a `pkgPath:` derivation, and [`SendCoins`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/banker/banker.gno#L237-L240) panics unless `from` equals it, so nothing on chain produces a banker for an object address; the one constructor that takes an address verbatim is [`MakeRealmValue`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/uverse.go#L474), in the test-only `testing` stdlib. Say in this doc comment that no key exists for the address, since the [worked example](https://github.com/gnolang/gno/blob/011afff91/docs/resources/gno-stdlibs.md?plain=1#L677-L697) and the [ADR](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L13) both read the other way.

## misc/genstd/mapping.go:140-149 [gh](https://github.com/gnolang/gno/blob/011afff91/misc/genstd/mapping.go#L140-L149) · [↗](../../../../../.worktrees/gno-review-6139/misc/genstd/mapping.go#L140)
Missing test: the widened predicate has no fixture, and [`fieldListsMatch`](https://github.com/gnolang/gno/blob/011afff91/misc/genstd/mapping.go#L354-L358) consults it to skip the Gno-to-Go parameter type check, so `func foo(n int) string` against `func X_foo(m *Machine, n any) string` now links silently for every native in the stdlib and fails at run time rather than at generation.

<details><summary>test cases</summary>

Two cases, the widened rule and its edge, beside [`linkFunctions_noMatchSig`](https://github.com/gnolang/gno/blob/011afff91/misc/genstd/mapping_test.go#L161), the case that guards the unwidened rule. Both pass at this head, and the first panics against the merge base's `mapping.go`.

`misc/genstd/zz_mapping_any_test.go`:

```go
package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_linkFunctions_anyParam pins the widened rule: an empty-interface Go
// parameter links against any gno parameter type, unchecked, and arrives as
// the TypedValue. `any` and `interface{}` are one type and both must qualify.
func Test_linkFunctions_anyParam(t *testing.T) {
	chdir(t, "testdata/linkFunctions_anyParam")

	pkgs, err := walkStdlibs(".")
	require.NoError(t, err)

	mappings := linkFunctions(pkgs)
	require.Len(t, mappings, 2)

	for _, m := range mappings {
		require.Len(t, m.Params, 1, "%s", m.GoFunc)
		assert.True(t, m.Params[0].IsTypedValue,
			"%s: an empty-interface Go parameter must receive the TypedValue", m.GoFunc)
	}
}

// Test_linkFunctions_namedIface pins the edge. Only the empty interface is
// exempt; a widening that swallowed a named interface would leave every
// native's Go signature unchecked against its gno declaration.
func Test_linkFunctions_namedIface(t *testing.T) {
	chdir(t, "testdata/linkFunctions_namedIface")

	pkgs, err := walkStdlibs(".")
	require.NoError(t, err)

	defer func() {
		r := recover()
		require.NotNil(t, r, "a named interface parameter linked without a type check")
		assert.Contains(t, fmt.Sprint(r), "doesn't match signature of go function")
	}()

	linkFunctions(pkgs)
}
```

`misc/genstd/testdata/linkFunctions_anyParam/std/std.gno`:

```go
package std

// Both declare a concrete gno type. The Go side takes the empty interface, so
// the parameter reaches the implementation as the raw gno.TypedValue.
func AnyParam(n int) string

func AnyParamLong(n int) string
```

`misc/genstd/testdata/linkFunctions_anyParam/std/std.go`:

```go
package std

import (
	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func AnyParam(m *gno.Machine, n any) string { return "" }

func AnyParamLong(m *gno.Machine, n interface{}) string { return "" }
```

`misc/genstd/testdata/linkFunctions_namedIface/std/std.gno`:

```go
package std

// A named interface is not the empty interface, so the gno and Go parameter
// types still have to match.
func ErrParam(n int) string
```

`misc/genstd/testdata/linkFunctions_namedIface/std/std.go`:

```go
package std

import (
	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func ErrParam(m *gno.Machine, n error) string { return "" }
```
</details>

## gnovm/pkg/gnolang/misc_test.go:126-136 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc_test.go#L126-L136) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/pkg/gnolang/misc_test.go#L126)
Nit: three of the five cases here compute `want` by calling `DeriveObjectIDCryptoAddr`, as do [two of the four native cases](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native_test.go#L275-L279), so the package that owns the derivation asserts nothing about the bytes it produces, and [`assignNewObjectID`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/realm.go#L1988-L1990) makes every address it already produced permanent. Pin the preimage layout and the resulting addresses as literals.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_objectid_golden_test.go <<'EOF'
package gnolang

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/crypto"
)

// TestZZObjectIDGoldenVectors pins the derivation end to end. Once an object's
// address is written into an event, a registry or a balance, changing any input
// to it is a state break: the same object answers differently and whatever sat
// at the old address is unreachable.
func TestZZObjectIDGoldenVectors(t *testing.T) {
	t.Parallel()

	// PkgID is the first input, and it is not a plain hash: PkgIDFromPkgPath
	// clears the top nibble and writes IsStdlib/IsImmutable/IsInternal flags
	// into it, with bit 0x10 reserved. Claiming that bit, or reclassifying a
	// path, moves every object address in that realm.
	for path, want := range map[string]string{
		"gno.land/r/demo/foo20":        "RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80",
		"gno.land/p/demo/tokens/grc20": "RID4FBD19DF645A50B847D72ED838E39F596646CAE1",
		"chain/runtime":                "RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50",
	} {
		require.Equal(t, want, PkgIDFromPkgPath(path).String(),
			"PkgID of %q moved; every object address in that realm moved with it", path)
	}

	vectors := []struct {
		pkgPath  string
		newTime  uint64
		preimage string
		addr     string
	}{
		{"gno.land/r/demo/foo20", 1, "objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:1", "g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape"},
		{"gno.land/r/demo/foo20", 2, "objectid:RID05A95D3B90BADA501E6136CC5D7F4EF98DAB5B80:2", "g1wr733ezlqykpj87fvl3p5n63u9nnq6vyuhju2a"},
		{"gno.land/p/demo/tokens/grc20", 7, "objectid:RID4FBD19DF645A50B847D72ED838E39F596646CAE1:7", "g1dmfquaplgaf0wakjhx5ftlnkdntua67j6dtyyj"},
		// The clock is a uint64 rendered in decimal, so the top of the range
		// is part of the contract.
		{"chain/runtime", 1 << 32, "objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:4294967296", "g1x53ple3fg0nghfnx0vu2xddnmd99h5mn5xv3nk"},
		{"chain/runtime", ^uint64(0), "objectid:RIDC09C8277A76BF0C457FDF56BD592EDCDCF839A50:18446744073709551615", "g18de5twlh8fpwgll5925uyf3faul4xwmn7uluhp"},
	}
	for _, v := range vectors {
		t.Run(v.pkgPath+":"+strconv.FormatUint(v.newTime, 10), func(t *testing.T) {
			t.Parallel()

			oid := ObjectID{PkgID: PkgIDFromPkgPath(v.pkgPath), NewTime: v.newTime}
			require.Equal(t, v.addr, DeriveObjectIDCryptoAddr(oid).String())
			require.Equal(t, v.addr, oid.DerivePath())

			// The preimage spelled out independently of the function under
			// test, so a changed prefix or separator reports as a layout
			// change rather than as a hash miss.
			require.Equal(t, v.addr, crypto.AddressFromPreimage([]byte(v.preimage)).String(),
				"preimage layout changed: expected %q", v.preimage)
		})
	}
}

// TestZZObjectIDIsSeparateFromPkgAddresses pins the domain separation. A realm
// address and an object address are drawn from one 20-byte space, and
// chain.PackageAddress lets a realm pick the pkgPath half of it freely, so the
// two preimage prefixes are what stop a realm naming an object it does not own.
func TestZZObjectIDIsSeparateFromPkgAddresses(t *testing.T) {
	t.Parallel()

	oid := ObjectID{PkgID: PkgIDFromPkgPath("gno.land/r/demo/foo20"), NewTime: 1}
	object := DeriveObjectIDCryptoAddr(oid).String()

	forged := "objectid:" + PkgIDFromPkgPath("gno.land/r/demo/foo20").String() + ":1"
	require.NotEqual(t, object, DerivePkgCryptoAddr(forged).String(),
		"a pkgPath-derived address reached an object address")
	require.NotEqual(t, object, DerivePkgCryptoAddr("gno.land/r/demo/foo20").String())
	require.NotEqual(t, object, DeriveStorageDepositCryptoAddr("gno.land/r/demo/foo20").String())
}
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestZZObjectID' -count=1
mkdir -p /tmp/mut6139
sed 's/"objectid:"/"objectid-v2:"/' gnovm/pkg/gnolang/misc.go > /tmp/mut6139/misc.go
printf '{"Replace":{"%s":"/tmp/mut6139/misc.go"}}' "$PWD/gnovm/pkg/gnolang/misc.go" > /tmp/mut6139/overlay.json
echo "--- branch tests under the prefix mutation:"
go test -overlay=/tmp/mut6139/overlay.json ./gnovm/pkg/gnolang/ -run 'TestObjectIDDerivePath|TestDeriveObjectIDCryptoAddrRejectsIncompleteIDs' -count=1
echo "--- the golden test under the same mutation:"
go test -overlay=/tmp/mut6139/overlay.json ./gnovm/pkg/gnolang/ -run 'TestZZObjectIDGoldenVectors' -count=1
rm -rf gnovm/pkg/gnolang/zz_objectid_golden_test.go /tmp/mut6139
```

Renaming the preimage prefix moves every address and the branch's own tests stay green.

```
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.060s
--- branch tests under the prefix mutation:
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.051s
--- the golden test under the same mutation:
--- FAIL: TestZZObjectIDGoldenVectors (0.00s)
    --- FAIL: TestZZObjectIDGoldenVectors/gno.land/r/demo/foo20:1 (0.00s)
        # … Error Trace, Error: Not equal: and Diff: lines cut
        	            	expected: "g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape"
        	            	actual  : "g1k30p3apsud5tjahmy985du3gshld6p27tgazvp"
# … the other four vectors fail the same way: foo20:2, grc20:7,
# … chain/runtime:4294967296 and chain/runtime:18446744073709551615
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.062s
```

Setting the reserved `0x10` bit in the `PkgID` flag nibble behaves the same way: the branch's unit tests are unmoved, and the golden test reports `RID15A95…` against `RID05A95…` with the message naming the realm whose addresses moved. The integration fixtures do catch both mutations, two directories away: [`grc20_object_id_events.txtar`](https://github.com/gnolang/gno/blob/011afff91/gno.land/pkg/integration/testdata/grc20_object_id_events.txtar#L25) pins `g1ej4f8h7qhwyxy7ys2mat3vcj0g72x8phv2k0w5` on lines 25, 31, 41 and 46, [`grc20_object_id_events_init.txtar`](https://github.com/gnolang/gno/blob/011afff91/gno.land/pkg/integration/testdata/grc20_object_id_events_init.txtar#L41) pins `g1fppxyq8rzamcz7a5rk0mhtvcf23pxuv8krrmz2` on lines 31, 41 and 46, [`token_identity_filetest.gno`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/filetests/token_identity_filetest.gno#L117) pins two more and [`event_provenance_filetest.gno`](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/p/demo/tokens/grc20/filetests/event_provenance_filetest.gno#L113) a third. What no fixture gives is an assertion in the package that owns the derivation.
</details>

## gnovm/pkg/gnolang/ownership.go:78-81 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L78-L81) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/pkg/gnolang/ownership.go#L78)
Nit: `""` is shared by every unstamped object, against the doc line above saying such an object is unnamed rather than sharing an address, so a realm keying a map by `ID()` merges them all into one entry and [`DeriveObjectIDCryptoAddr`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc.go#L207-L217) raises for the same absence of identity. The comment has to name that shared value.

<details><summary>measured</summary>

Thirteen of the fifteen shapes driven through the native inside a gno `recover()` at this head split the two behaviours, and an author who tests `ObjectID(42)` meets only the loud one:

```
int    PANIC-RECOVERED: value has no object identity
string PANIC-RECOVERED: value has no object identity
nilptr PANIC-RECOVERED: value has no object identity
nilifc PANIC-RECOVERED: value has no object identity
nilmap PANIC-RECOVERED: value has no object identity
nilfn  PANIC-RECOVERED: value has no object identity
err    ->
struct ->
&struct ->
method -> g1js0ae3haarqdgk69246mh49vfu7dp73q3h8mpy
closure ->
slice  ->
emptysl ->
nilslice PANIC-RECOVERED: value has no object identity
chanish -> g19jt5lxr0g04mpt50ywlm3hjuasxx4v8ey2rrhk
```

The remaining two are a third outcome: a top-level func value and the native `runtime.ObjectID` itself resolve through [`GetFirstObject`'s `*FuncValue` case](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L423-L424) and answer with a stamped address. The ADR accepts the unstamped window for events and does not cover a realm using the value as a key.
</details>

## gnovm/pkg/gnolang/ownership.go:74 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L74) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/pkg/gnolang/ownership.go#L74)
Nit: `DerivePath` returns an address rather than a path, and [`runtime.ObjectID`](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.gno#L13-L21) returns an address rather than an object ID, which is why both doc comments open by correcting the name and `docs/resources/gno-stdlibs.md` spends a paragraph on it. Rename them for what they return.

## gnovm/adr/prxxxx_objectid_derived_ids.md:1 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/prxxxx_objectid_derived_ids.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/adr/prxxxx_objectid_derived_ids.md#L1)
Nit: the filename is wrong. Rename to `pr6139_objectid_derived_ids.md`.

## gnovm/pkg/gnolang/misc.go:206-220 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/misc.go#L206-L220) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/pkg/gnolang/misc.go#L206)
Refactor: [`IsZero()`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L113) is `PkgID.IsZero() && NewTime == 0`, so no input trips the first panic without tripping the second, and the fifteen lines carry two conditions.

```suggestion
func DeriveObjectIDCryptoAddr(objectID ObjectID) crypto.Address {
	if objectID.PkgID.IsZero() || objectID.NewTime == 0 {
		panic("objectID is not fully stamped: " + objectID.String())
	}

	return crypto.AddressFromPreimage([]byte("objectid:" + objectID.PkgID.String() + ":" + strconv.FormatUint(objectID.NewTime, 10)))
}
```

<details><summary>equivalence</summary>

Both forms over the four cells of zero-or-set `PkgID` against zero-or-set `NewTime`, from a local clone of gnolang/gno:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_derive_equiv_test.go <<'EOF'
package gnolang

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
)

func TestZZDeriveEquiv(t *testing.T) {
	pid := PkgIDFromPkgPath("gno.land/r/demo/foo20")
	var zero PkgID
	for _, c := range []struct {
		name string
		oid  ObjectID
	}{
		{"both zero", ObjectID{}},
		{"pkgID zero, newTime 7", ObjectID{PkgID: zero, NewTime: 7}},
		{"pkgID set, newTime 0", ObjectID{PkgID: pid}},
		{"both set (newTime 1)", ObjectID{PkgID: pid, NewTime: 1}},
	} {
		fmt.Printf("%-22s three-guard=%-42s merged=%s\n", c.name,
			zzRun(zzThreeGuardForm, c.oid), zzRun(zzMergedForm, c.oid))
	}
}

func zzRun(f func(ObjectID) crypto.Address, oid ObjectID) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("panic(%v)", r)
		}
	}()
	return f(oid).String()
}

func zzThreeGuardForm(objectID ObjectID) crypto.Address {
	if objectID.IsZero() {
		panic("objectID cannot be zero")
	}
	if objectID.PkgID.IsZero() {
		panic("pkgID cannot be zero")
	}
	if objectID.NewTime == 0 {
		panic("newTime cannot be zero")
	}
	return zzPreimage(objectID)
}

func zzMergedForm(objectID ObjectID) crypto.Address {
	if objectID.PkgID.IsZero() || objectID.NewTime == 0 {
		panic("objectID is not fully stamped: " + objectID.String())
	}
	return zzPreimage(objectID)
}

func zzPreimage(objectID ObjectID) crypto.Address {
	return crypto.AddressFromPreimage([]byte("objectid:" + objectID.PkgID.String() + ":" + strconv.FormatUint(objectID.NewTime, 10)))
}
EOF
go test ./gnovm/pkg/gnolang/ -run 'TestZZDeriveEquiv|TestObjectIDDerivePath|TestDeriveObjectIDCryptoAddrRejectsIncompleteIDs' -count=1 -v 2>&1 | grep -E 'three-guard=|^(ok|--- )'
rm gnovm/pkg/gnolang/zz_derive_equiv_test.go
```

The two forms reject the same three cells and derive the same address on the fourth, so the merge drops four lines and no behaviour.

```
both zero              three-guard=panic(objectID cannot be zero)             merged=panic(objectID is not fully stamped: 0000000000000000000000000000000000000000:0)
pkgID zero, newTime 7  three-guard=panic(pkgID cannot be zero)                merged=panic(objectID is not fully stamped: 0000000000000000000000000000000000000000:7)
pkgID set, newTime 0   three-guard=panic(newTime cannot be zero)              merged=panic(objectID is not fully stamped: 05a95d3b90bada501e6136cc5d7f4ef98dab5b80:0)
both set (newTime 1)   three-guard=g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape   merged=g1un39xxhnkdlj46586xvclnsk5xaw04jlrnyape
--- PASS: TestZZDeriveEquiv (0.00s)
--- PASS: TestDeriveObjectIDCryptoAddrRejectsIncompleteIDs (0.00s)
--- PASS: TestObjectIDDerivePath (0.00s)
ok  	github.com/gnolang/gno/gnovm/pkg/gnolang	0.038s
```

The merged message carries the id, so a caller reads which half was missing rather than which branch fired. The only other thing the dropped branch does is the `debug`-build invariant assertion inside `IsZero`, which every other caller of it still performs.
</details>

## gnovm/stdlibs/chain/runtime/native.go:54-66 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.go#L54-L66) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/chain/runtime/native.go#L54)
Refactor: [`GetFirstObject`](https://github.com/gnolang/gno/blob/011afff91/gnovm/pkg/gnolang/ownership.go#L449-L450) already returns nil for every value with no object behind it, the zero `TypedValue` a failed type assertion leaves included, so the `tv.V == nil` branch rejects nothing the nil check below rejects.

```suggestion
func X_objectID(m *gno.Machine, v any) string {
	tv, _ := v.(gno.TypedValue)
	oo := tv.GetFirstObject(m.Store)
	if oo == nil {
		m.PanicString("value has no object identity")
	}

	return oo.GetObjectID().DerivePath()
}
```

<details><summary>equivalence</summary>

Both forms over every shape the parameter arrives as, the non-`TypedValue` the transpiled-Go path passes included:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
cat > gnovm/stdlibs/chain/runtime/zz_xobjectid_equiv_test.go <<'EOF'
package runtime

import (
	"fmt"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func TestZZXObjectIDEquiv(t *testing.T) {
	m := gno.NewMachineWithOptions(gno.MachineOptions{})
	for _, c := range []struct {
		name string
		v    any
	}{
		{"not a TypedValue (int)", 42},
		{"not a TypedValue (string)", "abc"},
		{"nil interface", nil},
		{"TypedValue zero", gno.TypedValue{}},
		{"TypedValue primitive", typedString("not an object")},
		{"nil PointerValue", gno.TypedValue{V: gno.PointerValue{}}},
		{"unstamped StructValue", gno.TypedValue{V: gno.NewAllocator(1 << 30).NewStruct(nil, nil)}},
	} {
		fmt.Printf("%-26s two-guard=%-12s one-guard=%s\n", c.name, zzTwoGuard(m, c.v), zzOneGuard(m, c.v))
	}
}

func zzOneGuard(m *gno.Machine, v any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "panic"
		}
	}()
	tv, _ := v.(gno.TypedValue)
	oo := tv.GetFirstObject(m.Store)
	if oo == nil {
		m.PanicString("value has no object identity")
	}
	return "ok:" + oo.GetObjectID().DerivePath()
}

func zzTwoGuard(m *gno.Machine, v any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = "panic"
		}
	}()
	tv, ok := v.(gno.TypedValue)
	if !ok || tv.V == nil {
		m.PanicString("value has no object identity")
	}
	oo := tv.GetFirstObject(m.Store)
	if oo == nil {
		m.PanicString("value has no object identity")
	}
	return "ok:" + oo.GetObjectID().DerivePath()
}
EOF
go test ./gnovm/stdlibs/chain/runtime/ -count=1 -v -run 'TestZZXObjectIDEquiv' 2>&1 | grep -E 'two-guard=|^(ok|--- )'
go test ./gnovm/stdlibs/chain/runtime/ -count=1 2>&1 | tail -1
rm gnovm/stdlibs/chain/runtime/zz_xobjectid_equiv_test.go
```

All seven shapes agree, so the four lines come off without moving a rejection.

```
not a TypedValue (int)     two-guard=panic        one-guard=panic
not a TypedValue (string)  two-guard=panic        one-guard=panic
nil interface              two-guard=panic        one-guard=panic
TypedValue zero            two-guard=panic        one-guard=panic
TypedValue primitive       two-guard=panic        one-guard=panic
nil PointerValue           two-guard=panic        one-guard=panic
unstamped StructValue      two-guard=ok:          one-guard=ok:
--- PASS: TestZZXObjectIDEquiv (0.00s)
ok  	github.com/gnolang/gno/gnovm/stdlibs/chain/runtime	0.031s
ok  	github.com/gnolang/gno/gnovm/stdlibs/chain/runtime	0.076s
```
</details>

## SKIP gnovm/stdlibs/native_gas.go:143 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/native_gas.go#L143) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/native_gas.go#L143)
Nit: [`crypto.Address.String()`](https://github.com/gnolang/gno/blob/011afff91/tm2/pkg/crypto/crypto.go#L114-L116) is `AddressToBech32`, and this row's comment names a hex encoding. Skipped: a finding about a code comment's own wording changes no behaviour, and the line already carries a posted comment about its price.

## SKIP gnovm/stdlibs/chain/runtime/native.go:54 [gh](https://github.com/gnolang/gno/blob/011afff91/gnovm/stdlibs/chain/runtime/native.go#L54) · [↗](../../../../../.worktrees/gno-review-6139/gnovm/stdlibs/chain/runtime/native.go#L54)
Nit: the native is unconditional, where [#6101](https://github.com/gnolang/gno/pull/6101)'s equivalent reads [`RealmIDEnabled`](https://github.com/gnolang/gno/blob/20d2a9f2e/gnovm/stdlibs/internal/execctx/context.go#L46) off the execution context, and the `.gno` source declaring this one is genesis state, so removing it after realms call it is a hard fork. Skipped: a maintainer decides whether to gate this native, and the ADR comment above already asks for the permanence decision in writing.

## SKIP examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:39 [gh](https://github.com/gnolang/gno/blob/011afff91/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39) · [↗](../../../../../.worktrees/gno-review-6139/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39)
`Register` is a crossing function, so its `cur` is [the current one by construction](https://github.com/gnolang/gno/blob/011afff91/gnovm/adr/interrealm_v2.md?plain=1#L338), and an `IsCurrent()` check here fires on nothing: a captured realm, a realm held in a package var, `cross()` over an adjacent frame's realm and a non-crossing call from outside are each rejected before the body runs.

<details><summary>probe</summary>

The guard inserted at the top of `Register`, then attacked from a second realm:

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6139 -R gnolang/gno
REG=examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno
P=examples/gno.land/r/demo/regprobe
python3 - "$REG" <<'PY'
import sys, pathlib
p = pathlib.Path(sys.argv[1]); s = p.read_text()
old = 'func Register(cur realm, token *grc20.Token, slug string) string {\n\tif token == nil {'
new = ('func Register(cur realm, token *grc20.Token, slug string) string {\n'
       '\tif !cur.IsCurrent() {\n\t\tpanic("grc20reg: PROBE-GUARD-FIRED")\n\t}\n\n'
       '\tif token == nil {')
assert old in s
p.write_text(s.replace(old, new))
PY
mkdir -p $P
printf 'module = "gno.land/r/demo/regprobe"\ngno = "0.9"\n' > $P/gnomod.toml
head='// PKGPATH: gno.land/r/demo/regprobe
package regprobe

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)
'
# the call the doc comment prescribes, then four routes carrying a realm that is not cur
printf '%s\nfunc main(cur realm) {\n\ttoken, _ := grc20.NewToken("Probe", "PRB", 4, cur)\n\tprintln("key:", grc20reg.Register(cross(cur), token, "probe"))\n\tprintln("cur.IsCurrent():", cur.IsCurrent())\n}\n' "$head" > /tmp/a.gno
printf '%s\nfunc main(cur realm) {\n\ttoken, _ := grc20.NewToken("Probe", "PRB", 4, cur)\n\tcaptured := cur\n\tprintln(grc20reg.Register(captured, token, "probe"))\n}\n' "$head" > /tmp/b.gno
printf '%s\nvar held realm\n\nfunc main(cur realm) {\n\ttoken, _ := grc20.NewToken("Probe", "PRB", 4, cur)\n\theld = cur\n\tprintln(grc20reg.Register(cross(held), token, "probe"))\n}\n' "$head" > /tmp/c.gno
printf '%s\nfunc main(cur realm) {\n\ttoken, _ := grc20.NewToken("Probe", "PRB", 4, cur)\n\tstale := cur.Previous()\n\tprintln(grc20reg.Register(cross(stale), token, "probe"))\n}\n' "$head" > /tmp/d.gno
printf '%s\nfunc main(cur realm) {\n\ttoken, _ := grc20.NewToken("Probe", "PRB", 4, cur)\n\tprintln(grc20reg.Register(cur, token, "probe"))\n}\n' "$head" > /tmp/e.gno
for n in a b c d e; do
	cp /tmp/$n.gno $P/probe_${n}_filetest.gno
	printf '%s ' "$n:"
	(cd $P && GNOROOT=$(git rev-parse --show-toplevel) go run ../../../../../gnovm/cmd/gno test -v . 2>&1) |
		grep -oE 'unexpected panic: .*|cur\.IsCurrent\(\): .*|key: .*' | awk '!seen[$0]++'
	rm $P/probe_${n}_filetest.gno
done
rm -rf $P /tmp/[a-e].gno
git checkout -- $REG
```

No input reaches the guard and answers false: the ordinary call reports `IsCurrent()` true, and each of the four hostile routes dies before the body.

```
a: key: gno.land/r/demo/regprobe.probe
cur.IsCurrent(): true
b: unexpected panic: gno.land/r/demo/regprobe/probe_b.gno:12:10-53: only `cur` or `cross(rlm)` are allowed as the first argument to a crossing function but got captured<VPBlock(1,2)>
c: unexpected panic: cannot persist realm value: realm values are ephemeral and tied to a call frame
d: unexpected panic: cross: rlm is not the current cur (stale capture or sibling frame)
e: unexpected panic: gno.land/r/demo/regprobe/probe_e.gno:11:10-48: cannot cur-call to external realm function gno.land/r/demo/defi/grc20reg.grc20reg<VPBlock(2,1)>.Register from gno.land/r/demo/regprobe
```
</details>

Skipped: it answers another reviewer's open thread rather than asking the author for an edit, and the branch carries no such check for a finding to anchor on.
