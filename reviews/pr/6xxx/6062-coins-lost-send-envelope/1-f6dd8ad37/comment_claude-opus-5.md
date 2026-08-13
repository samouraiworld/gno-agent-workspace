# Review: [#6062](https://github.com/gnolang/gno/pull/6062)
Event: COMMENT

## Body
Four routes past the persistence walk abort with `cannot persist realm value`: a closure capture, a pointer inside a struct inside a slice, a map value and a bound method value. The closure route succeeds on the merge base.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6062-coins-lost-send-envelope/1-f6dd8ad37/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## gnovm/stdlibs/chain/runtime/unsafe/unsafe.go:39 [gh](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/stdlibs/chain/runtime/unsafe/unsafe.go#L39) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/stdlibs/chain/runtime/unsafe/unsafe.go#L39)
`m.Realm` is the borrowed realm, not the realm whose code is running, so a realm reading its own envelope through an object another realm owns has its payment refused while the identical direct read is accepted. Pin whichever answer you keep with a fixture, since [`payable_callee_read.txtar`](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/integration/testdata/payable_callee_read.txtar#L12) covers only the plain callee.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6062 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/zz_payable_borrow.txtar <<'TXTAR'
loadpkg gno.land/p/test/tok $WORK/tok
loadpkg gno.land/r/test/host $WORK/host
loadpkg gno.land/r/test/helper $WORK/helper
loadpkg gno.land/r/test/payee $WORK/payee
loadpkg gno.land/r/test/shop $WORK/shop

adduser alice

gnoland start

## OVER-CREDIT. payee is the entry realm and never reads the envelope. It
## calls helper, which reads it through an object payee owns, so the read
## is attributed to payee. Accepted, and the 400ugnot strands in payee.
gnokey maketx call -pkgpath gno.land/r/test/payee -func Ignores -send 400ugnot -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test alice
stdout OK!                            # IS:     the read is credited to payee, which never looked
# ! gnokey maketx call -pkgpath gno.land/r/test/payee -func Ignores -send 400ugnot -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test alice
# stderr 'never read the send-envelope'  # SHOULD: same outcome as payable_callee_read.txtar

## The coins are sitting in payee, which has no idea it was paid.
gnokey maketx call -pkgpath gno.land/r/test/payee -func Balance -gas-fee 1000000ugnot -gas-wanted 5000000 -chainid=tendermint_test alice
stdout 'balance=400'

## FALSE REFUSAL. shop is the entry realm, is paid, and reads its own
## envelope, through an object host owns. The read is attributed to host,
## so the payment is refused.
! gnokey maketx call -pkgpath gno.land/r/test/shop -func Buy -send 400ugnot -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test alice
stderr 'never read the send-envelope'  # IS:     shop read its own envelope and is refused
# gnokey maketx call -pkgpath gno.land/r/test/shop -func Buy -send 400ugnot -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test alice
# stdout 'bought=400'                  # SHOULD: same outcome as BuyDirect below

## Same realm, same coins, same requirement, read directly: accepted.
gnokey maketx call -pkgpath gno.land/r/test/shop -func BuyDirect -send 400ugnot -gas-fee 1000000ugnot -gas-wanted 8000000 -chainid=tendermint_test alice
stdout 'bought=400'

-- tok/gnomod.toml --
module = "gno.land/p/test/tok"
gno = "0.9"

-- tok/tok.gno --
package tok

import "chain/runtime/unsafe"

// T is an ordinary shared type. Its method reads the ambient envelope.
type T struct{ n int }

func New() *T { return &T{} }

func (t *T) Peek() int64 {
	return unsafe.OriginSend().AmountOf("ugnot")
}

-- host/gnomod.toml --
module = "gno.land/r/test/host"
gno = "0.9"

-- host/host.gno --
package host

import "gno.land/p/test/tok"

var shared = tok.New()

func Get(cur realm) *tok.T { return shared }

-- payee/gnomod.toml --
module = "gno.land/r/test/payee"
gno = "0.9"

-- payee/payee.gno --
package payee

import (
	"strconv"

	"chain/banker"
	"chain/runtime/unsafe"

	"gno.land/p/test/tok"
	"gno.land/r/test/helper"
)

var mine = tok.New()

// Ignores never reads the envelope. It hands helper an object this realm
// owns; helper's read is attributed here by the borrow rules.
func Ignores(cur realm) string {
	return helper.Use(cross(cur), mine)
}

func Balance(cur realm) string {
	me := unsafe.CurrentRealm().Address()
	return "balance=" + strconv.FormatInt(
		banker.NewReadonlyBanker().GetCoins(me).AmountOf("ugnot"), 10)
}

-- helper/gnomod.toml --
module = "gno.land/r/test/helper"
gno = "0.9"

-- helper/helper.gno --
package helper

import (
	"strconv"

	"gno.land/p/test/tok"
)

func Use(cur realm, t *tok.T) string {
	return "helper-saw=" + strconv.FormatInt(t.Peek(), 10)
}

-- shop/gnomod.toml --
module = "gno.land/r/test/shop"
gno = "0.9"

-- shop/shop.gno --
package shop

import (
	"strconv"

	"chain/runtime/unsafe"

	"gno.land/r/test/host"
)

// Buy reads this realm's own envelope through an object host owns.
func Buy(cur realm) string {
	amt := host.Get(cross(cur)).Peek()
	if amt < 400 {
		panic("underpaid")
	}
	return "bought=" + strconv.FormatInt(amt, 10)
}

// BuyDirect is the same requirement, read directly.
func BuyDirect(cur realm) string {
	amt := unsafe.OriginSend().AmountOf("ugnot")
	if amt < 400 {
		panic("underpaid")
	}
	return "bought=" + strconv.FormatInt(amt, 10)
}
TXTAR
go test ./gno.land/pkg/integration/ -run 'TestTestdata/zz_payable_borrow$' -v
rm gno.land/pkg/integration/testdata/zz_payable_borrow.txtar
```

`shop.Buy` and `shop.BuyDirect` read the same envelope and only the direct read is accepted.

```
> gnokey maketx call -pkgpath gno.land/r/test/payee -func Ignores -send 400ugnot ...
("helper-saw=400" string)
> gnokey maketx call -pkgpath gno.land/r/test/payee -func Balance ...
("balance=400" string)
> ! gnokey maketx call -pkgpath gno.land/r/test/shop -func Buy -send 400ugnot ...
Data: coins were sent but the called function never read them
    0  gno/gno.land/pkg/sdk/vm/errors.go:99 - 400ugnot sent to gno.land/r/test/shop.Buy, which never read the send-envelope
> gnokey maketx call -pkgpath gno.land/r/test/shop -func BuyDirect -send 400ugnot ...
("bought=400" string)
```

`Machine.Realm` follows the receiver object's owner at [`machine.go:2607-2618`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/gnolang/machine.go#L2607-L2618) and the closure's capture realm at [`machine.go:2641-2650`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/gnolang/machine.go#L2641-L2650). At the merge base the same file reports `bought=400` for `shop.Buy`.
</details>

## gnovm/pkg/test/test.go:66 [gh](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/pkg/test/test.go#L66) · [↗](../../../../../.worktrees/gno-review-6062/gnovm/pkg/test/test.go#L66)
Missing test: the harness never consults `OriginSendObserved`, so a `SEND:` filetest whose code never reads the envelope passes while `MsgCall` refuses the same call.

<details><summary>test cases</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6062 -R gnolang/gno
cat > gnovm/tests/files/zz_unobserved_send_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/test/zzunobserved
// SEND: 400ugnot

// A SEND: envelope no code reads passes under `gno test`, while MsgCall
// refuses the same call with "never read the send-envelope". The 400ugnot
// strands in the realm's address and the harness reports green.

package zzunobserved

import (
	"chain/banker"
	"chain/runtime/unsafe"
)

// ignores has no notion of being paid.
func ignores(cur realm) {}

func main(cur realm) {
	ignores(cur)
	me := unsafe.CurrentRealm().Address()
	println("stranded=", banker.NewReadonlyBanker().GetCoins(me).AmountOf("ugnot"))
}

// Output:
// stranded= 400
GNO
go test -run 'TestFiles/zz_unobserved_send_filetest.gno$' ./gnovm/pkg/gnolang/ -v
rm gnovm/tests/files/zz_unobserved_send_filetest.gno
```

The 400ugnot strands in the realm's address and the test is green.

```
--- PASS: TestFiles/zz_unobserved_send_filetest.gno (2.89s)
```
</details>

## gno.land/pkg/gnoclient/integration_test.go:684 [gh](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/gnoclient/integration_test.go#L684) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/gnoclient/integration_test.go#L684)
Nit: the breakdown still subtracts `Send 1000000`, and its total matches neither the old assertion nor the new one.

## gno.land/pkg/sdk/vm/keeper.go:745 [gh](https://github.com/gnolang/gno/blob/f6dd8ad37/gno.land/pkg/sdk/vm/keeper.go#L745) · [↗](../../../../../.worktrees/gno-review-6062/gno.land/pkg/sdk/vm/keeper.go#L745)
Nit: `AddPackage` sets `OriginSendRecipient` and leaves `OriginSendRecipientPath` empty, which [`context_testing.go:150-152`](https://github.com/gnolang/gno/blob/f6dd8ad37/gnovm/tests/stdlibs/testing/context_testing.go#L150-L152) says must never happen.

## docs/MANIFESTO.md:188-189 [gh](https://github.com/gnolang/gno/blob/f6dd8ad37/docs/MANIFESTO.md?plain=1#L188-L189) · [↗](../../../../../.worktrees/gno-review-6062/docs/MANIFESTO.md#L188-L189)
Nit: the rewrap is unrelated to the three fixes and leaves a double space after `weapons.`.
