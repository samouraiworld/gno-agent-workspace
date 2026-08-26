# Review: [#6072](https://github.com/gnolang/gno/pull/6072)
Event: COMMENT

## Body
[Automatic AI review]

Automated pass over the diff, scoped to correctness, state safety and test coverage of the new grc721 core. It makes no design judgement on the Token/PrivateLedger/Teller split and carries no merge verdict.

## examples/gno.land/p/demo/tokens/grc721/token.gno:271 [gh](https://github.com/jinoosss/gno/blob/refactor/grc721-core/examples/gno.land/p/demo/tokens/grc721/token.gno#L271) · [↗](../../../../../.worktrees/gno-review-6072/examples/gno.land/p/demo/tokens/grc721/token.gno#L271)
`SetApprovalForAll(op, false)` on an operator never approved stores an AVL node instead of removing one, so entries accumulate unbounded and never release, where this file's `Approve` revoke and `setBalance` both remove on clear.

<details><summary>repro</summary>

Writes 1084 bytes of permanent storage for a relationship that does not exist, charged to the issuing realm, and emits an `ApprovalForAll ... approved=false` event as though state changed. `isApprovedForAll` reads a missing key and a stored `false` identically, and no method removes an operator entry.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6072 -R gnolang/gno
cat > gno.land/pkg/integration/testdata/grc721_sgd.txtar <<'TXTAR'
loadpkg gno.land/p/demo/tokens/grc721
loadpkg gno.land/r/test/nftstore $WORK/nftstore
adduser alice
gnoland start
gnokey maketx send -send 90000000ugnot -to $alice_user_addr -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid tendermint_test test1
stdout 'OK!'
gnokey maketx call -pkgpath gno.land/r/test/nftstore -func Deny -args g1us8428u2a5satrlxzagqqa5m6vmuze025anjlj -gas-fee 5000000ugnot -gas-wanted 50000000 -chainid=tendermint_test alice
stdout 'STORAGE DELTA'
-- nftstore/gnomod.toml --
module = "gno.land/r/test/nftstore"
gno = "0.9"
-- nftstore/nftstore.gno --
package nftstore
import "gno.land/p/demo/tokens/grc721"
var token *grc721.Token
var ledger *grc721.PrivateLedger
func init(cur realm) { token, ledger = grc721.NewToken("Store", "STO", 0, cur) }
func Deny(cur realm, op address) {
	if err := token.CallerTeller().SetApprovalForAll(0, cur, op, false); err != nil {
		panic(err)
	}
}
TXTAR
go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc721_sgd' -count=1 -v 2>&1 | grep -E 'STORAGE DELTA|bytes_delta'
rm gno.land/pkg/integration/testdata/grc721_sgd.txtar
```

A never-approved operator denied once:

```
STORAGE DELTA:  1084 bytes
"bytes_delta":1084 ... "pkg_path":"gno.land/r/test/nftstore"
```

Ready-to-add regression, red at head and green once the `false` path removes:

```go
func TestSGDFalseDoesNotStore(cur realm, t *testing.T) {
	_, led := newTestToken("Foo", "FOO", 0, cur)
	uassert.NoError(t, led.SetApprovalForAll(alice, bob, false))
	uassert.Equal(t, 0, led.operatorApprovals.Size()) // IS 1 at head; SHOULD be 0
}
```
</details>
