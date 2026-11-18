# Review: [#6375](https://github.com/gnolang/gno/pull/6375)

Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. The `msg` argument of both BLS natives is charged flat gas while the hashing it buys grows with its length, a Warning this branch introduces; two missing tests post beside it.
Model: claude-opus-5-5, standard review
Commit: 0ee53fefc (latest), BLS natives slice only (`gnovm/stdlibs/crypto/bls12381/*`, their rows in `native_gas.go` and `generated.go`, `gnovm/adr/pr6375_bls12381_native.md`)
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6375 0ee53fefc`
Round: 1. Solo round over the BLS slice: 2 finders, 4 candidates judged from scratch by an agent that was not their finder (1 refuted), and 2 more added and run by that judge's completeness pass.

## Body

## gnovm/stdlibs/native_gas.go:196 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L196) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/native_gas.go#L196) · Warning

`msg` is unbounded in [`bls12381.gno`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.gno#L37) and unpriced here, so a 64 MiB message, inside a transaction's [500 MB allocation limit](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gno.land/pkg/sdk/vm/keeper.go#L50), pays a 32-byte call's gas and runs 25x longer.

<details><summary>repro</summary>

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

`verifyAggregate` passes `msg` whole to [`bls.HashToG2`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L123), whose `expand_message_xmd` runs SHA-256 over every byte, and the generated dispatcher copies `msg` into Go first. [`native_input_bounds.md`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/native_input_bounds.md#L82) asks for each variable-length parameter to be bounded in the wrapper or sloped. The allocation itself is nearly free in gas: the 16 MiB slice above costs 24 more than the 32-byte one.

Native time of one call with one key, from a Go benchmark beside the package (`BenchmarkMsgGrowth`, 5 iterations), as a ratio to `msg` = 32 B. The growth starts past the 64 KiB of the [earlier flat measurement](https://github.com/gnolang/gno/pull/6375#discussion_r4335346193):

| `msg` | `fastAggregateVerify` | `fastAggregateVerifyExcluding` |
| --- | --- | --- |
| 1 MiB | 1.3x | 1.3x |
| 16 MiB | 7.2x | 6.8x |
| 64 MiB | 25.2x | 25.0x |

</details>

## gnovm/stdlibs/crypto/bls12381/bls12381_test.go:78-83 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L78-L83) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/crypto/bls12381/bls12381_test.go#L78) · Missing test

Missing test: beside the infinity key, no case feeds a key outside the prime-order subgroup, so nothing pins the subgroup check the [`X_fastAggregateVerify`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/crypto/bls12381/bls12381.go#L39-L40) godoc promises.

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

## SKIP gnovm/adr/pr6375_bls12381_native.md:72 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/adr/pr6375_bls12381_native.md#L72) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/adr/pr6375_bls12381_native.md#L72) · Nit

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

## SKIP gnovm/stdlibs/native_gas.go:195 [gh](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/stdlibs/native_gas.go#L195) · [↗](../../../../../.worktrees/gno-review-6375/gnovm/stdlibs/native_gas.go#L195) · Suggestion

Suggestion: the [calibrate benches](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/cmd/calibrate/ibc_native_bench_test.go) and [`gen_native_table.py`](https://github.com/gnolang/gno/blob/0ee53fefc33a77f709709151ea84958dbca7e5f9/gnovm/cmd/calibrate/gen_native_table.py#L217-L225) know no BLS native, so the regeneration both rows' comment asks for emits neither row.

Not posted: a Suggestion ships only once applied and run both ways, and no bench was added or run in this round.
