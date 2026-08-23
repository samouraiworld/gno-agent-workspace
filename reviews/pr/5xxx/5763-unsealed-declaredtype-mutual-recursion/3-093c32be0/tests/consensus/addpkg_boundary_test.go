// Consensus artifact for PR 5763: what the real MsgAddPackage path does
// with a mutual-type-decl-recursion package, on the merge-base versus the
// PR head. Measured at 093c32be0 / merge-base 0397fc87f:
//
//   TestZZMutualAddPackage      base: REJECTED, gas 148   head: ACCEPTED, gas 20670
//   TestZZGasTypeHeavy          base: gas 34164           head: gas 34164
//   TestZZRejectedShapeGas      base: gas 0, same error   head: gas 0, same error
//   TestZZMutualPersistAcrossTx base: addpkg rejected     head: val 9 -> 10 -> 11
//
// The first row is the fork: an unupgraded validator writes a failed tx and
// deducts 148 gas where an upgraded one writes the package and deducts 20670.
//
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gno.land/pkg/sdk/vm/zz_consensus_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/consensus/addpkg_boundary_test.go
go test -count=1 -v -run 'TestZZ' ./gno.land/pkg/sdk/vm/
git checkout $(git merge-base origin/master HEAD) -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
go test -count=1 -v -run 'TestZZ' ./gno.land/pkg/sdk/vm/
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
rm gno.land/pkg/sdk/vm/zz_consensus_test.go
*/

package vm

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
)

// TestZZMutualAddPackage deploys a package containing mutual type-decl
// recursion through the real VM keeper AddPackage path, the same path a
// MsgAddPackage tx takes on chain. It prints the accept/reject outcome
// and, on accept, the gas consumed — so the two trees can be compared.
func TestZZMutualAddPackage(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/mutual"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "mutual.gno", Body: `package mutual

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

var Head = T1{Next: &T2{Val: 9}}

func Get(cur realm) int {
	return Head.Next.Val
}
`},
	}

	msg := NewMsgAddPackage(addr, pkgPath, files)
	gasBefore := ctx.GasMeter().GasConsumed()
	err := env.vmk.AddPackage(ctx, msg)
	gasAfter := ctx.GasMeter().GasConsumed()
	if err != nil {
		fmt.Printf("ZZRESULT: REJECTED err=%v\n", err)
		return
	}
	pv := env.vmk.getGnoTransactionStore(ctx).GetPackage(pkgPath, false)
	fmt.Printf("ZZRESULT: ACCEPTED pkg=%v gasDelta=%d\n", pv != nil, gasAfter-gasBefore)
}

// TestZZGasTypeHeavy deploys a type-heavy package (every base kind that
// fillTypeInPlace handles, plus derived and self-recursive types) through
// the real AddPackage path and prints the gas consumed, so the two trees
// can be compared for a gas-schedule change on an existing-program shape.
func TestZZGasTypeHeavy(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/typeheavy"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "t.gno", Body: `package typeheavy

type S struct {
	A int
	B string
}
type Derived S
type I interface{ M() int }
type Sl []int
type Ar [3]int
type Mp map[string]int
type Fn func(int) int
type Pt *int
type Pi int
type MyErr error
type Node struct {
	Val  int
	Next *Node
}

var Head = &Node{Val: 1, Next: &Node{Val: 2}}

func Get(cur realm) int { return Head.Next.Val }
`},
	}

	msg := NewMsgAddPackage(addr, pkgPath, files)
	before := ctx.GasMeter().GasConsumed()
	err := env.vmk.AddPackage(ctx, msg)
	after := ctx.GasMeter().GasConsumed()
	fmt.Printf("ZZGAS: err=%v gasDelta=%d\n", err, after-before)
}

// TestZZRejectedShapeGas deploys a package that BOTH trees reject (an
// illegal finite-size value cycle) and prints the gas consumed plus the
// error, so the fee deducted and the tx result can be compared.
func TestZZRejectedShapeGas(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/badcyc"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "zc.gno", Body: `package badcyc

type T1 struct {
	Self T2
	Val  int
}

type T2 T1

func Get(cur realm) int { return 1 }
`},
	}

	before := ctx.GasMeter().GasConsumed()
	err := env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files))
	after := ctx.GasMeter().GasConsumed()
	bal := env.bankk.GetCoins(ctx, addr)
	fmt.Printf("ZZREJECT: gasDelta=%d balance=%s\nZZREJECT-ERR: %+v\n", after-before, bal.String(), err)
}

// TestZZMutualPersistAcrossTx deploys a mutual-type-decl-recursion realm,
// commits the transaction store, then calls it from a SEPARATE transaction
// store — so the types come back from the object store through fillType
// rather than from the preprocess-time in-memory graph, where T1.Base and
// T2.Base were the same pointer.
func TestZZMutualPersistAcrossTx(t *testing.T) {
	env := setupTestEnv()
	ctx := env.vmk.MakeGnoTransactionStore(env.ctx)

	addr := crypto.AddressFromPreimage([]byte("addr1"))
	acc := env.acck.NewAccountWithAddress(ctx, addr)
	env.acck.SetAccount(ctx, acc)
	env.bankk.SetCoins(ctx, addr, initialBalance)

	const pkgPath = "gno.land/r/mutualp"
	files := []*std.MemFile{
		{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
		{Name: "m.gno", Body: `package mutualp

type T1 struct {
	Next *T2
	Val  int
}

type T2 T1

var Head = T1{Next: &T2{Val: 9}, Val: 1}

func Bump(cur realm) string {
	Head.Next.Val++
	var b T2
	b.Val = Head.Next.Val
	return "val=" + itoa(b.Val) + " same=" + boolstr(T2(Head) == T2(Head))
}

func itoa(i int) string {
	if i == 0 { return "0" }
	s := ""
	for i > 0 { s = string(rune('0'+i%10)) + s; i /= 10 }
	return s
}
func boolstr(b bool) string { if b { return "true" }; return "false" }
`},
	}

	if err := env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, files)); err != nil {
		fmt.Printf("ZZPERSIST: ADDPKG-REJECTED err=%v\n", err)
		return
	}
	env.vmk.CommitGnoTransactionStore(ctx)

	// Transaction 2: fresh store, types reloaded from the object store.
	ctx2 := env.vmk.MakeGnoTransactionStore(env.ctx)
	res, err := env.vmk.Call(ctx2, NewMsgCall(addr, nil, pkgPath, "Bump", nil))
	fmt.Printf("ZZPERSIST: tx2 res=%q err=%v\n", res, err)
	env.vmk.CommitGnoTransactionStore(ctx2)

	// Transaction 3: read it back again.
	ctx3 := env.vmk.MakeGnoTransactionStore(env.ctx)
	res3, err3 := env.vmk.Call(ctx3, NewMsgCall(addr, nil, pkgPath, "Bump", nil))
	fmt.Printf("ZZPERSIST: tx3 res=%q err=%v\n", res3, err3)
}
