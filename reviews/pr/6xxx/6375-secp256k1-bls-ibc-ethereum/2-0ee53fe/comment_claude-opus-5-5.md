# Review: [#6375](https://github.com/gnolang/gno/pull/6375)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Two Warnings the branch introduces block it: a node upgraded in place to this binary panics at boot because its store lacks `crypto/bls12381`, and both natives charge `msg` flat gas while its hashing and its Go copy grow with its length, on a free query too; nine missing tests post beside them.
Model: claude-opus-5-5, deep review, high effort
Commit: 0ee53fefc (latest), BLS natives slice only (`gnovm/stdlibs/crypto/bls12381/*`, their rows in `native_gas.go` and `generated.go`, `gnovm/adr/pr6375_bls12381_native.md`)
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6375 0ee53fefc`
Round: 2, on the same head as round 1, so no prior finding was resolved: round 1's `msg` Warning carries with the free-query case added, its two missing tests and its ADR SKIP carry unchanged, and its calibrate-bench SKIP folds into the `native_gas.go:193` Suggestion. 9 finders, one reflector, 18 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 1 refuted, none of them above Nit.

## Body

- This binary panics at boot on a node upgraded in place: [`VMKeeper.Initialize`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/keeper.go#L217-L218) type-checks [`crypto/bls12381`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/generated.go#L2733) against a store whose stdlibs were written once, at genesis, by [`InitChainer`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/gnoland/app.go#L418).
- Missing test: `crypto/bls12381` is absent from the amd64 and arm64 [determinism corpus](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/misc/multiarch-determinism/corpus.gno#L11-L23) and from every txtar, where its gnark sibling `crypto/bn254` has [corpus cases](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/misc/multiarch-determinism/corpus.gno#L261-L269) and a [restart txtar](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/integration/testdata/stdlib_ibc_crypto_determinism.txtar#L1-L2).
- Missing test: no test pins which parameter the [`crypto/bls12381` rows](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L196-L201) charge per byte on, and pricing the 48-byte aggregate instead of the excluded keys keeps every test green.

<details><summary>repro: in-place upgrade boot</summary>

The upgrade is simulated by deleting the package's stored mempackage, which is what a store written by a release without `crypto/bls12381` lacks, then booting a fresh keeper on it as `gnoland.NewAppWithOptions` does.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6375 -R gnolang/gno
cat > gno.land/pkg/sdk/vm/upgrade_boot_test.go <<'EOF'
package vm

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestUpgradeBootWithoutNewStdlib(t *testing.T) {
	env := setupTestEnv()
	ms := env.ctx.MultiStore()
	ms.GetStore(env.vmk.iavlKey).Delete(nil, []byte("pkg:_/crypto/bls12381"))

	vmk2 := NewVMKeeper(env.vmk.baseKey, env.vmk.iavlKey, env.acck, env.bankk, env.prmk)
	var rec any
	func() {
		defer func() { rec = recover() }()
		vmk2.Initialize(log.NewNoopLogger(), ms.MultiCacheWrap())
	}()
	require.Nil(t, rec, "booting on a store that predates crypto/bls12381 panics")
}

func TestUpgradeBootControl(t *testing.T) {
	env := setupTestEnv()
	ms := env.ctx.MultiStore()
	vmk2 := NewVMKeeper(env.vmk.baseKey, env.vmk.iavlKey, env.acck, env.bankk, env.prmk)
	require.NotPanics(t, func() { vmk2.Initialize(log.NewNoopLogger(), ms.MultiCacheWrap()) })
}
EOF
go test ./gno.land/pkg/sdk/vm -run 'TestUpgradeBoot' -count=1 -v -timeout 2m 2>&1 | grep -E '^(=== RUN|--- )|Error:'
rm gno.land/pkg/sdk/vm/upgrade_boot_test.go
```

The boot without the stored package panics on a nil dereference, while the control with it boots.

```text
=== RUN   TestUpgradeBootWithoutNewStdlib
        	Error:      	Expected nil, but got: "invalid memory address or nil pointer dereference"
--- FAIL: TestUpgradeBootWithoutNewStdlib (1.53s)
=== RUN   TestUpgradeBootControl
--- PASS: TestUpgradeBootControl (0.79s)
```

At the merge base the store holds every listed stdlib and the same boot succeeds. A store missing `crypto/bech32` panics the same way on both trees, so the missing nil check predates the branch, and the branch reaches it by listing a stdlib the running chain never stored. `NewAppWithOptions` runs `Initialize` at [`app.go:290`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/gnoland/app.go#L290), and the type check runs only when the store already holds packages, [`keeper.go:188`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/keeper.go#L188), so a fresh genesis is unaffected. On `gnoland-1`, `vm/qfile` answers for all 50 stdlibs the merge base lists and returns `InvalidPackageError` for `crypto/bls12381`, so this is the first listed stdlib the running chain never stored.

</details>

<details><summary>test cases: determinism txtar</summary>

```txtar
# crypto/bls12381 from a realm on a running node: addpkg, a tx, a vm/qeval
# query, a 512-key call in a tx, then a restart and the same call again.

gnoland start

gnokey maketx addpkg -pkgdir $WORK/blsreach -pkgpath gno.land/r/test/blsreach -gas-fee 1000001ugnot -gas-wanted 10_000_000 -chainid=tendermint_test test1
stdout OK!

gnokey maketx call -pkgpath gno.land/r/test/blsreach -func Verify -gas-fee 5000001ugnot -gas-wanted 50_000_000 -chainid=tendermint_test test1
stdout OK!
stdout '\(true bool\)'

gnokey query vm/qeval --data 'gno.land/r/test/blsreach.Check()'
stdout '\(true bool\)'

gnokey maketx call -pkgpath gno.land/r/test/blsreach -func Big -gas-fee 10000001ugnot -gas-wanted 100_000_000 -chainid=tendermint_test test1
stdout OK!
stdout '\(false bool\)'
stdout 'GAS USED:'

gnoland restart

gnokey maketx call -pkgpath gno.land/r/test/blsreach -func Verify -gas-fee 5000001ugnot -gas-wanted 50_000_000 -chainid=tendermint_test test1
stdout OK!
stdout '\(true bool\)'

-- blsreach/gnomod.toml --
module = "gno.land/r/test/blsreach"
gno = "0.9"

-- blsreach/blsreach.gno --
package blsreach

import (
	"crypto/bls12381"
	"encoding/hex"
)

// Vectors copied from gnovm/stdlibs/crypto/bls12381/bls12381_test.gno.
const (
	pubKeys = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68" +
		"b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"ac7563bb556a1687d1dfc2b5fa2d5891c0756ba960f3cfc62de348a5549ddf65121efdbc06050988704674d7820f4a3a"
	sig = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
	msg        = "gno.land bls12381 test vector"
	aggregate  = "8b38c72176718cbaa7ba3fb02c062ce2a116cbd72791c1af8d4fc9ff84f0ba265dfac905e4cbc8120b5cd01ac95c3aaf"
	excluded   = "b6a005f54655f0a670599f6b9de9edffc39c99799a482f62fa5e42bfe986a93ae2c52360a96139ed82c4c4d6672695e5" +
		"b900fd910bb898cd4fb2a689bd59751e450105efc3b2be03c6a9545b7221fa2f5320d284cc731dc49c93262459138da0"
	partialSig = "951e05eff3af6074e89a657c9d36f9d01c0b463e33ea5cfb64c19d822b52055c816414e78bd833da3ad50364bb1e358b" +
		"0e322f576108346548a273637c42b7056e6d0777458215800ed23ff4d320ea3f5350df732af8d2b9d26eb6397292403f"
)

func mustDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func Check() bool {
	return bls12381.FastAggregateVerify(mustDecode(pubKeys), []byte(msg), mustDecode(sig)) &&
		bls12381.FastAggregateVerifyExcluding(mustDecode(aggregate), mustDecode(excluded), []byte(msg), mustDecode(partialSig))
}

func Verify(cur realm) bool {
	return Check()
}

func Big(cur realm) bool {
	one := mustDecode(pubKeys)[:bls12381.PubKeySize]
	pks := make([]byte, 0, 512*bls12381.PubKeySize)
	for i := 0; i < 512; i++ {
		pks = append(pks, one...)
	}
	return bls12381.FastAggregateVerify(pks, []byte(msg), mustDecode(sig))
}
```

The script passes at the head as `gno.land/pkg/integration/testdata/bls12381_reach.txtar`. The corpus half is a `FastAggregateVerify` and a `FastAggregateVerifyExcluding` case on the same vectors in `misc/multiarch-determinism/corpus.gno`. Without either, a gnark-crypto bump whose arm64 field code changes a verdict splits validators by architecture and nothing in the repository compares the two.

</details>

<details><summary>test cases: gas row shape</summary>

```go
// Both BLS natives do one key decoding and subgroup check per 48-byte key, so
// each must meter its key list per byte: pubkeys (param 0) for
// fastAggregateVerify, excluded (param 1) for fastAggregateVerifyExcluding.
func TestBLS12381GasMetersKeys(t *testing.T) {
	t.Parallel()
	want := map[string]int8{"fastAggregateVerify": 0, "fastAggregateVerifyExcluding": 1}
	for i := range calibratedNativeGas {
		e := &calibratedNativeGas[i]
		idx, ok := want[e.Fn]
		if e.Pkg != "crypto/bls12381" || !ok {
			continue
		}
		delete(want, e.Fn)
		if e.SlopeKind != SizeLenBytes || e.SlopeIdx != idx || e.Slope == 0 {
			t.Errorf("%s must meter param %d per byte; got Slope=%d SlopeIdx=%d SlopeKind=%d",
				e.Fn, idx, e.Slope, e.SlopeIdx, e.SlopeKind)
		}
	}
	if len(want) != 0 {
		t.Fatalf("crypto/bls12381 rows missing from calibratedNativeGas: %v", want)
	}
}
```

It passes at the head and fails with `SlopeIdx: 0` on the `fastAggregateVerifyExcluding` row: `fastAggregateVerifyExcluding must meter param 1 per byte; got Slope=1962000 SlopeIdx=0 SlopeKind=1`. That edit leaves `go test ./gnovm/stdlibs/ ./gnovm/stdlibs/crypto/bls12381/` green, and the existing rows are checked by [`TestEveryRowValidates`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas_validate_test.go#L85-L99) for their kind alone.

</details>

## gnovm/stdlibs/native_gas.go:196 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L196) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/native_gas.go#L196) · Warning

`msg` is unbounded in the [wrapper](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.gno#L37) and unpriced here, so a 64 MiB message runs 25x longer for a 32-byte call's gas, in a transaction or a free [`vm/qeval`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/handler.go#L98) query.

<details><summary>repro: transaction</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6375 -R gnolang/gno
mkdir -p /tmp/blsmsg
printf 'module = "gno.land/p/demo/blsmsg"\ngno = "0.9"\n' > /tmp/blsmsg/gnomod.toml
cat > /tmp/blsmsg/msg_test.gno <<'EOF'
package blsmsg

import (
	"crypto/bls12381"
	"encoding/hex"
	"testing"
)

const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
)

func run(n, calls int) {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	for i := 0; i < calls; i++ {
		bls12381.FastAggregateVerify(pk, msg, sig)
	}
}

func TestMsg32x1(t *testing.T)  { run(32, 1) }
func TestMsg32x5(t *testing.T)  { run(32, 5) }
func TestMsg1Mx1(t *testing.T)  { run(1<<20, 1) }
func TestMsg1Mx5(t *testing.T)  { run(1<<20, 5) }
func TestMsg16Mx1(t *testing.T) { run(16<<20, 1) }
func TestMsg16Mx5(t *testing.T) { run(16<<20, 5) }
func TestMsg64Mx1(t *testing.T) { run(64<<20, 1) }
func TestMsg64Mx5(t *testing.T) { run(64<<20, 5) }
EOF
(cd gnovm && go build -o /tmp/gno-bls ./cmd/gno) && GNOROOT=$PWD /tmp/gno-bls test -v /tmp/blsmsg 2>&1 | grep -E 'PASS|GAS'
rm -rf /tmp/blsmsg /tmp/gno-bls
```

Gas per call, (x5 - x1) / 4, is 2130789 at 32 B and 2130908 at 64 MiB: the charge does not follow `msg`.

```text
--- PASS: TestMsg32x1 (0.00s)
--- GAS:  4712724
--- PASS: TestMsg32x5 (0.00s)
--- GAS:  13235880
--- PASS: TestMsg1Mx1 (0.00s)
--- GAS:  4712724
--- PASS: TestMsg1Mx5 (0.01s)
--- GAS:  13235880
--- PASS: TestMsg16Mx1 (0.02s)
--- GAS:  4712748
--- PASS: TestMsg16Mx5 (0.07s)
--- GAS:  13235904
--- PASS: TestMsg64Mx1 (0.06s)
--- GAS:  4712748
--- PASS: TestMsg64Mx5 (0.27s)
--- GAS:  13236382
```

`verifyAggregate` passes `msg` whole to [`bls.HashToG2`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L123), whose `expand_message_xmd` runs SHA-256 over every byte. A transaction's [500 MB allocation limit](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/keeper.go#L50) admits the 64 MiB `msg`. [`native_input_bounds.md`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/native_input_bounds.md?plain=1#L82) asks for each variable-length parameter to be bounded in the wrapper or sloped. The allocation itself costs 24 gas more at 16 MiB than at 32 bytes.

Native time of one call with one key, from a Go benchmark beside the package (`BenchmarkMsgGrowth`, 5 iterations), as a ratio to `msg` = 32 B. The growth starts past the 64 KiB of the [earlier flat measurement](https://github.com/gnolang/gno/pull/6375#discussion_r4335346193):

| `msg` | `fastAggregateVerify` | `fastAggregateVerifyExcluding` |
| --- | --- | --- |
| 1 MiB | 1.3x | 1.3x |
| 16 MiB | 7.2x | 6.8x |
| 64 MiB | 25.2x | 25.0x |

</details>

<details><summary>repro: free query</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6375 -R gnolang/gno
cat > gno.land/pkg/sdk/vm/bls_query_msg_test.go <<'EOF'
package vm

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

const blsQueryRealm = `package blsq

import (
	"crypto/bls12381"
	"encoding/hex"
)

const (
	pkHex  = "820ad0f24a42c82129fef2a137f7b7c230c2aaffb78ffd82f6cbdcd2bfbf3560435a35c62d3ff66ad696b78f8c6c6c68"
	sigHex = "857cbc9c0976c88648089b0598cdc7b8c75a96f8e4f06ea0f0bbd0e2d5ac74d9dae7d4c479a7b573f0d7f7e3b875a6b9" +
		"039083a95ea6f3af6d432056337bc115ae50023fa289c320322941a3b07db7a710b438f643432a1a56f64b121931357e"
)

// Spin holds one msg of n bytes and calls the native k times on it.
func Spin(n int, k int) int {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	c := 0
	for i := 0; i < k; i++ {
		if !bls12381.FastAggregateVerify(pk, msg, sig) {
			c++
		}
	}
	return c
}

// Hold allocates the same msg and makes no native call.
func Hold(n int) int {
	pk, _ := hex.DecodeString(pkHex)
	sig, _ := hex.DecodeString(sigHex)
	msg := make([]byte, n)
	return len(pk) + len(sig) + len(msg)
}
`

func TestBLSQueryMsg(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, std.MustParseCoins(ugnot.ValueString(20000000)))
	pkgPath := "gno.land/r/test/blsq"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "x.gno", Body: blsQueryRealm},
	}
	require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)))
	env.vmk.CommitGnoTransactionStore(ctx)

	q := func(expr string) (time.Duration, uint64, error) {
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		start := time.Now()
		_, err := env.vmk.QueryEval(env.ctx, pkgPath, expr)
		d := time.Since(start)
		runtime.ReadMemStats(&b)
		return d, b.TotalAlloc - a.TotalAlloc, err
	}

	oog := map[int]string{}
	for _, n := range []int{32, 4 << 20} {
		for _, k := range []int{1300, 1500} {
			_, _, err := q(fmt.Sprintf("Spin(%d, %d)", n, k))
			oog[n] += fmt.Sprintf("k=%d err=%v; ", k, err)
		}
		t.Logf("msg=%d: %s", n, oog[n])
	}

	const k = 3
	var base time.Duration
	for _, n := range []int{32, 1 << 20, 16 << 20, 64 << 20} {
		d, spinAlloc, err := q(fmt.Sprintf("Spin(%d, %d)", n, k))
		require.NoError(t, err)
		_, holdAlloc, err := q(fmt.Sprintf("Hold(%d)", n))
		require.NoError(t, err)
		if n == 32 {
			base = d
		}
		perCall := (float64(spinAlloc) - float64(holdAlloc)) / k
		t.Logf("msg=%9d B: wall-time ratio to 32 B %6.1fx; Go heap per call beyond Hold %.2f x len(msg)",
			n, float64(d)/float64(base), perCall/float64(n))
	}
	if oog[32] == oog[4<<20] {
		t.Errorf("per-call gas is flat in len(msg) on the free query path")
	}
}
EOF
go test ./gno.land/pkg/sdk/vm -run 'TestBLSQueryMsg' -count=1 -v -timeout 2m 2>&1 | grep -E 'msg=|flat|^--- '
rm gno.land/pkg/sdk/vm/bls_query_msg_test.go
```

The test fails because calls run out of gas at the same count at 32 B and at 4 MiB. Each call still allocates one more copy of `msg` outside the VM's allocator; at 32 B the call's fixed allocations dominate the per-call figure.

```text
    bls_query_msg_test.go:84: msg=32: k=1300 err=<nil>; k=1500 err=out of gas in location: CPUCycles; 
    bls_query_msg_test.go:84: msg=4194304: k=1300 err=<nil>; k=1500 err=out of gas in location: CPUCycles; 
    bls_query_msg_test.go:98: msg=       32 B: wall-time ratio to 32 B    1.0x; Go heap per call beyond Hold 1239.08 x len(msg)
    bls_query_msg_test.go:98: msg=  1048576 B: wall-time ratio to 32 B    1.3x; Go heap per call beyond Hold 1.03 x len(msg)
    bls_query_msg_test.go:98: msg= 16777216 B: wall-time ratio to 32 B    7.4x; Go heap per call beyond Hold 1.00 x len(msg)
    bls_query_msg_test.go:98: msg= 67108864 B: wall-time ratio to 32 B   32.8x; Go heap per call beyond Hold 1.00 x len(msg)
    bls_query_msg_test.go:102: per-call gas is flat in len(msg) on the free query path
--- FAIL: TestBLSQueryMsg (13.08s)
```

The dispatcher copies `msg` through [`Gno2GoValue`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/generated.go#L1426-L1428), whose [`make([]byte, svl)`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/pkg/gnolang/gonative.go#L365) takes no allocator. A query charges no fee and runs under a [1.5 GB allocation limit](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/keeper.go#L51), three times the transaction's, so a 1 GiB `msg`, counted once against that limit, adds an uncounted 1 GiB copy per call. Every `[]byte` native shares the uncounted copy; rows such as `crypto/sha256` slope on the length, and only the `crypto/bls12381` rows leave it unpriced.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:58 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L58) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L58) · Missing test

Missing test: no case passes the same key twice, a shape Altair [sync committees allow](https://github.com/ethereum/consensus-specs/blob/master/specs/altair/beacon-chain.md#get_next_sync_committee), so gnark's [doubling branch](https://github.com/Consensys/gnark-crypto/blob/v0.20.1/ecc/bls12-381/g1.go#L459-L462), reached from [`agg.AddMixed`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L57), never runs under these tests.

<details><summary>test cases</summary>

```go
// keyFrom returns the compressed public key of secret s and s*H(msg).
func keyFrom(tb testing.TB, s int64, msg []byte) (pk []byte, sig bls.G2Affine) {
	tb.Helper()
	g1 := g1Generator()
	h, err := bls.HashToG2(msg, ethDST)
	require.NoError(tb, err)
	var p bls.G1Affine
	p.ScalarMultiplication(&g1, big.NewInt(s))
	b := p.Bytes()
	sig.ScalarMultiplication(&h, big.NewInt(s))
	return b[:], sig
}

// sigBytes returns the compressed sum of points.
func sigBytes(points ...bls.G2Affine) []byte {
	var sum bls.G2Jac
	for i := range points {
		sum.AddMixed(&points[i])
	}
	var a bls.G2Affine
	a.FromJacobian(&sum)
	b := a.Bytes()
	return b[:]
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func TestFastAggregateVerifyRepeatedKey(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	q, sq := keyFrom(t, 0x89abcdef, msg)
	require.True(t, X_fastAggregateVerify(cat(pk, pk, q), msg, sigBytes(s, s, sq)))
	require.False(t, X_fastAggregateVerify(cat(pk, pk, q), msg, sigBytes(s, sq)))
}
```

It passes at the head.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:76 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L76) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L76) · Missing test

Missing test: every length case here is short, and gnark's [decoder](https://github.com/Consensys/gnark-crypto/blob/v0.20.1/ecc/bls12-381/marshal.go#L1163) ignores bytes past the 96th, so a [signature length check](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L43) of `<` instead of `!=` fails no test.

<details><summary>test cases</summary>

```go
// keyFrom, sigBytes and cat as in the repeated-key cases; keySum returns the
// compressed sum of compressed keys.
func keySum(tb testing.TB, keys ...[]byte) []byte {
	tb.Helper()
	var sum bls.G1Jac
	for _, k := range keys {
		var p bls.G1Affine
		_, err := p.SetBytes(k)
		require.NoError(tb, err)
		sum.AddMixed(&p)
	}
	var a bls.G1Affine
	a.FromJacobian(&sum)
	b := a.Bytes()
	return b[:]
}

func TestTrailingByte(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	q, _ := keyFrom(t, 0x89abcdef, msg)
	sig := sigBytes(s)
	require.True(t, X_fastAggregateVerify(pk, msg, sig))
	require.False(t, X_fastAggregateVerify(pk, msg, cat(sig, []byte{0})))
	aggregate := keySum(t, pk, q)
	require.True(t, X_fastAggregateVerifyExcluding(aggregate, q, msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(cat(aggregate, []byte{0}), q, msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, q, msg, cat(sig, []byte{0})))
}
```

It passes at the head. With `!=` turned into `<` for `sig` at lines 43 and 82 and for `aggregate` at line 82, `X_fastAggregateVerify(pk, msg, sig||0x00)` returns true and this case fails, while `TestFastAggregateVerify` and `TestFastAggregateVerifyExcluding` still pass. The wrapper's own `!=` at [`bls12381.gno:37`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.gno#L37) still refuses it for a realm.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:78-83 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L78-L83) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L78) · Missing test

Missing test: apart from the infinity key, no case feeds a key outside the prime-order subgroup, so nothing pins the subgroup check the [`X_fastAggregateVerify`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L39-L40) godoc promises.

<details><summary>test cases</summary>

```go
// offSubgroupG1 returns the encoding of a point on y^2 = x^3 + 4 that is not
// in the r-order subgroup. Needs "github.com/consensys/gnark-crypto/ecc/bls12-381/fp".
func offSubgroupG1(t *testing.T) []byte {
	var p bls.G1Affine
	var rhs, four fp.Element
	four.SetUint64(4)
	for i := uint64(1); ; i++ {
		p.X.SetUint64(i)
		rhs.Square(&p.X).Mul(&rhs, &p.X).Add(&rhs, &four)
		if p.Y.Sqrt(&rhs) == nil {
			continue
		}
		require.True(t, p.IsOnCurve())
		if !p.IsInSubGroup() {
			b := p.Bytes()
			return b[:]
		}
	}
}

func TestOutsideSubgroup(t *testing.T) {
	msg := make([]byte, 32)
	pubkeys, aggregate, sig := committee(t, 4, msg, func(int) bool { return true })
	bad := offSubgroupG1(t)
	require.False(t, X_fastAggregateVerify(append(append([]byte{}, pubkeys[0]...), bad...), msg, sig))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, bad, msg, sig))
}
```

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:104 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L104) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L104) · Missing test

Missing test: no case pins the refusal of a non-canonical key or signature, such as an x at or above the field modulus, which gnark [refuses today](https://github.com/Consensys/gnark-crypto/blob/v0.20.1/ecc/bls12-381/marshal.go#L961).

<details><summary>test cases</summary>

```go
// keyFrom, sigBytes and cat as in the repeated-key cases.
// Needs "github.com/consensys/gnark-crypto/ecc/bls12-381/fp".
func TestNonCanonical(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	sig := sigBytes(s)
	modulus := fp.Modulus().FillBytes(make([]byte, fp.Bytes))

	xIsP := append([]byte{}, modulus...) // x = p, compressed flag set
	xIsP[0] |= 0x80
	infPayload := make([]byte, pubKeySize) // infinity flag with a nonzero x
	infPayload[0], infPayload[pubKeySize-1] = 0xc0, 0x01
	badMask := make([]byte, pubKeySize) // compressed, infinity and sign all set
	badMask[0] = 0xe0
	for name, key := range map[string][]byte{"x = p": xIsP, "infinity payload": infPayload, "invalid mask": badMask} {
		require.False(t, X_fastAggregateVerify(cat(pk, key), msg, sig), name)
	}

	sigXIsP := cat(modulus, make([]byte, fp.Bytes)) // x.A1 = p
	sigXIsP[0] |= 0x80
	sigInfPayload := make([]byte, signatureSize)
	sigInfPayload[0], sigInfPayload[signatureSize-1] = 0xc0, 0x01
	sigBadMask := make([]byte, signatureSize)
	sigBadMask[0] = 0xe0
	for name, bad := range map[string][]byte{"x = p": sigXIsP, "infinity payload": sigInfPayload, "invalid mask": sigBadMask} {
		require.False(t, X_fastAggregateVerify(pk, msg, bad), name)
	}
}
```

It passes at the head. A key with a second encoding fails the caller's Merkle membership proof, so the signature is where a second encoding would get through.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:185 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L185) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L185) · Missing test

Missing test: no excluded set names one key twice, so nothing pins that a member holding two committee seats and absent from both is [subtracted once per seat](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L91-L98).

<details><summary>test cases</summary>

```go
// keyFrom, sigBytes and cat as in the repeated-key cases, keySum as in the
// trailing-byte cases.
func TestFastAggregateVerifyExcludingRepeatedMember(t *testing.T) {
	msg := []byte("signing root")
	k1, s1 := keyFrom(t, 0x1234567, msg)
	k2, _ := keyFrom(t, 0x89abcdef, msg)
	k3, s3 := keyFrom(t, 0x7654321, msg)
	aggregate := keySum(t, k1, k2, k2, k3)
	require.True(t, X_fastAggregateVerifyExcluding(aggregate, cat(k2, k2), msg, sigBytes(s1, s3)))
	require.False(t, X_fastAggregateVerifyExcluding(aggregate, k2, msg, sigBytes(s1, s3)))
}
```

It passes at the head.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:197 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L197) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L197) · Missing test

Missing test: no case excludes every member and sends the identity signature, the one input that `verifyAggregate`'s identity refusal alone stops from verifying any message.

<details><summary>test cases</summary>

```go
	t.Run("nobody signed, identity signature", func(t *testing.T) {
		t.Parallel()
		var all []byte
		for _, pk := range pubkeys {
			all = append(all, pk...)
		}
		var inf bls.G2Affine
		infSig := inf.Bytes()
		require.False(t, X_fastAggregateVerifyExcluding(aggregate, all, []byte("any message"), infSig[:]))
	})
```

With `bls12381.go` lines 116-118 deleted, this case fails and every existing `TestFastAggregateVerifyExcluding` case still passes.

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:205 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L205) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L205) · Missing test

Missing test: the signature mismatch alone refuses this identity aggregate, so deleting the [aggregate's identity check](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L86) keeps every test green while letting one key holder sign for an identity aggregate.

<details><summary>test cases</summary>

```go
// keyFrom and sigBytes as in the repeated-key cases.
func TestFastAggregateVerifyExcludingIdentityAggregate(t *testing.T) {
	msg := []byte("signing root")
	pk, s := keyFrom(t, 0x1234567, msg)
	var p, neg bls.G1Affine
	_, err := p.SetBytes(pk)
	require.NoError(t, err)
	neg.Neg(&p)
	nb := neg.Bytes()
	var inf bls.G1Affine
	ib := inf.Bytes()
	require.False(t, X_fastAggregateVerifyExcluding(ib[:], nb[:], msg, sigBytes(s)))
}
```

It passes at the head and fails with `|| agg.IsInfinity()` deleted, while `TestFastAggregateVerify` and `TestFastAggregateVerifyExcluding` still pass. Without the clause, the infinity aggregate decodes to Z = 0, `AddMixed` copies the negated excluded key into that empty sum, and the signers' key is `pk_a`.

</details>

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381_test.go:53 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L53) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L53) · Missing test

Missing test: no test sends a repeated key, so the doubling branch of gnark's `G1Jac.AddMixed`, reached from `bls12381.go:57`, never runs under the package's tests.

Not posted: the same gap as the `bls12381_test.go:58` section, whose one added case closes both.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381_test.go:84 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L84) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L84) · Missing test

Missing test: no test repeats a key, so the valid sync-committee shape `pk||pk`, which only gnark's doubling branch keeps at `2pk`, is unpinned.

Not posted: the same gap as the `bls12381_test.go:58` section, whose one added case closes both.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381_test.go:163 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L163) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L163) · Missing test

Missing test: no test puts the same key twice in `excluded`, so nothing pins the per-entry subtraction that committees with a twice-sampled absent member depend on.

Not posted: the `bls12381_test.go:185` section carries it, anchored on the excluded-set case it extends rather than on the `func` line.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381_test.gno:9 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.gno#L9) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.gno#L9) · Missing test

Missing test: the vectors here name their secret keys but not the signer of `vecSig`, so no accepting vector is known to come from an implementation other than gnark.

Not posted: PLAUSIBLE; signing the vector with py_ecc or blst would settle whether `vecSig` is gnark-made, and neither was available to this round.

## SKIP gnovm/stdlibs/generated.go:1433 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/generated.go#L1433) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/generated.go#L1433) · Missing test

Missing test: no CI job compares `fastAggregateVerify`'s verdict between gnark's amd64 and arm64 field code.

Not posted: the same gap as the Body's determinism-corpus bullet.

## SKIP gnovm/adr/pr6375_bls12381_native.md:15 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/pr6375_bls12381_native.md?plain=1#L15) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/adr/pr6375_bls12381_native.md#L15) · Nit

Nit: `bls12381.go` ships two natives, while the Decision opens on one and lines 30 to 47 name or price the second before line 49 introduces it.

Not posted: the gno delta keeps the ADR, the change's governing document, out of the posted findings.

## SKIP gnovm/adr/pr6375_bls12381_native.md:72 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/pr6375_bls12381_native.md?plain=1#L72) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/adr/pr6375_bls12381_native.md#L72) · Nit

Nit: 47M is the per-key slope alone, and with the row's 2.02M base a 512-key call is charged about 49.1M.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6375 -R gnolang/gno
mkdir -p /tmp/blsgas
printf 'module = "gno.land/p/demo/blsgas"\ngno = "0.9"\n' > /tmp/blsgas/gnomod.toml
cat > /tmp/blsgas/gas_test.gno <<'EOF'
package blsgas

import (
	"crypto/bls12381"
	"testing"
)

func call(keys int) {
	bls12381.FastAggregateVerify(make([]byte, keys*48), make([]byte, 32), make([]byte, 96))
}

func TestKeys1(t *testing.T)   { call(1) }
func TestKeys512(t *testing.T) { call(512) }
func TestKeys0(t *testing.T)   { _ = make([]byte, 512*48); _ = make([]byte, 32); _ = make([]byte, 96) }
EOF
(cd gnovm && go build -o /tmp/gno-bls ./cmd/gno) && GNOROOT=$PWD /tmp/gno-bls test -v /tmp/blsgas 2>&1 | grep -E 'PASS|GAS'
rm -rf /tmp/blsgas /tmp/gno-bls
```

The 512-key call costs 49486865 - 360230 = 49126635 gas, the row's 2024000 + 1962000 * 24576 / 1024.

```text
--- PASS: TestKeys1 (0.00s)
--- GAS:  2490785
--- PASS: TestKeys512 (0.00s)
--- GAS:  49486865
--- PASS: TestKeys0 (0.00s)
--- GAS:  360230
```

</details>

Not posted: the gno delta keeps the ADR, the change's governing document, out of the posted findings.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381.go:86 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L86) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381.go#L86) · Nit

Nit: the identity check here refuses a committee of keys summing to the identity, such as `{pk, -pk}`, against the [godoc](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L67-L69)'s promise of the answer `FastAggregateVerify` gives over the signers.

Not posted: the clause stays, per the `bls12381_test.go:205` section, so the edit is the godoc's wording, and a finding about a comment's wording ships SKIP.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381_test.go:21 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L21) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L21) · Nit

Nit: the `signers` godoc says it returns n random secret keys, and the function returns only the public keys and the signature.

Not posted: UNVERIFIED, on the finder's read, and a finding about a comment's wording ships SKIP.

## SKIP gnovm/stdlibs/native_gas.go:98 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L98) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/native_gas.go#L98) · Nit

Nit: the table header lists `crypto/bls12381` among the Xeon Silver 4114 fits, against the row's own comment at line 193 naming an M4 Pro and a scaling factor.

Not posted: UNVERIFIED, on the finder's read, and a finding about a comment's wording ships SKIP.

## SKIP gnovm/stdlibs/crypto/bls12381/bls12381.gno:50 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.gno#L50) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381.gno#L50) · Suggestion

Suggestion: name the signer count here, the committee size minus the excluded keys, since a realm counting signers from the participation bits accepts one member's signature as the full committee.

<details><summary>check</summary>

One member signs alone and every other member goes in `excluded` with all participation bits set: membership holds for each excluded key, the bits pass the two-thirds threshold, and `X_fastAggregateVerifyExcluding` returns true for that single signer. No realm in the branch imports `crypto/bls12381` yet, so the case waits on the light client the [ADR](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/pr6375_bls12381_native.md?plain=1#L55-L57) describes.

</details>

Not posted: the edit is the wording of a doc comment, and a finding about a comment's wording ships SKIP; the misuse it guards against is the caller's, which the doc already makes responsible for membership.

## SKIP gnovm/stdlibs/native_gas.go:193 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L193) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/native_gas.go#L193) · Suggestion

Suggestion: derive both rows from the Ryzen 7 7840U behind the [2.3 factor](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/cmd/calibrate/ibc_native_bench_test.go#L77-L84), and add both natives to the [calibrate benches](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/cmd/calibrate/gen_native_table.py#L217-L225), since a regeneration emits neither row today.

<details><summary>check</summary>

The rows scale M4 Pro timings by a factor taken between a Ryzen and the Xeon. On a Ryzen 7 7840HS, the same Zen 4 part as the 7840U at a higher power limit, the branch's own `BenchmarkFastAggregateVerify` runs slower than the M4 figure the row scales, at the smallest and the largest key count, so the table's method, a Ryzen timing times the factor, projects both calls above what the rows charge. That the 7840U runs no faster than the 7840HS is an inference.

</details>

Not posted: a Suggestion ships only once applied and run both ways, and no row was rederived or bench added in this round; a gas rate drawn from a timing is the calibration's to set.
