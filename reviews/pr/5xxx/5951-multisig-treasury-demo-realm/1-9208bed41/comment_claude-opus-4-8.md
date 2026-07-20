# Review: PR [#5951](https://github.com/gnolang/gno/pull/5951)
Posted: https://github.com/gnolang/gno/pull/5951#pullrequestreview-4733411800
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Reproduced on 9208bed41. The realm has no way back from a bad input: `initialized` is one-shot, a proposal cannot be cancelled, and the treasury address is fixed by the pkgpath. Every gap below therefore lands as permanent state rather than a retryable error, worth closing before this ships as the reference example.

The red Merge Requirements check is the approval bot, not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5951-multisig-treasury-demo-realm/1-9208bed41/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/multisig/multisig.gno:120-122 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L120) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029318)
Critical: `TreasuryAddress` returns the caller's realm address, not the treasury's, so a realm that composes with this one funds itself. [`unsafe.CurrentRealm()`](https://github.com/gnolang/gno/blob/9208bed41/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L38-L40) stack-walks, so it resolves to whoever asked. Deriving the address from this package's own pkgpath, or from a threaded `cur realm`, makes both read paths agree.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5951 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/treasury_addr.txtar <<'EOF'
loadpkg gno.land/r/demo/multisig
gnoland start
gnokey query vm/qeval --data 'gno.land/r/demo/multisig.TreasuryAddress()'
gnokey maketx addpkg -pkgdir $WORK/wrapper -pkgpath gno.land/r/demo/wrapper -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test test1
gnokey query vm/qeval --data 'gno.land/r/demo/wrapper.Treasury()'
-- wrapper/gnomod.toml --
module = "gno.land/r/demo/wrapper"
gno = "0.9"
-- wrapper/wrapper.gno --
package wrapper
import "gno.land/r/demo/multisig"
func Treasury() string { return multisig.TreasuryAddress().String() }
EOF

go test -v -run 'TestTestdata/treasury_addr' ./gno.land/pkg/integration/ 2>&1 | grep -E 'qeval|data:'
rm gno.land/pkg/integration/testdata/treasury_addr.txtar
```

```
> gnokey query vm/qeval --data 'gno.land/r/demo/multisig.TreasuryAddress()'
data: ("g17vd2lug0kdaeahm9sv3y0udz7pac9kc0kqs0aa" .uverse.address)
> gnokey query vm/qeval --data 'gno.land/r/demo/wrapper.Treasury()'
data: ("g1ndz9e3hkgyz2xgzs9zuj5au9lf8hfncftf7vwh" string)
```

The second address is the wrapper's own package address.
</details>

## examples/gno.land/r/demo/multisig/multisig.gno:47-55 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L47) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029325)
`Setup` accepts three owner addresses without checking they are distinct, then records `ownerCount = 3` whatever the tree holds. `Setup(A, A, A, 2)` leaves one owner and an unreachable threshold, so anything sent to the treasury is locked for good behind the one-shot `initialized`. The threshold bound at [line 47](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L47) checks the literal 3 rather than the owner set, so `Setup(A, A, B, 3)` fails the same way.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5951 -R gnolang/gno
cd examples/gno.land/r/demo/multisig
mv multisig_test.gno /tmp/multisig_test.gno.bak

cat > dup_test.gno <<'EOF'
package multisig_test

import (
	"testing"

	"gno.land/r/demo/multisig"
)

func TestDupOwners(cur realm, t *testing.T) {
	a := address("g1owner1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	multisig.Setup(cross(cur), a, a, a, 2)
	println("Setup(A,A,A,2) accepted")

	testing.SetRealm(testing.NewUserRealm(a))
	id := multisig.Propose(cross(cur), "payout", a, 100)
	multisig.Approve(cross(cur), id)
	println(multisig.Render(""))
	multisig.Execute(cross(cur), id)
}
EOF

gno test -v -run TestDupOwners .
rm dup_test.gno && mv /tmp/multisig_test.gno.bak multisig_test.gno
```

```
Setup(A,A,A,2) accepted
Multisig Treasury
Initialized: true
Owners: 3 | Threshold: 2

Proposal #1: payout | to: g1owner1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx | amount: 100 | approvals: 1/2 [pending]

=== RUN   TestDupOwners
panic: not enough approvals
# …
```
</details>

## examples/gno.land/r/demo/multisig/multisig.gno:67-81 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L67) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029331)
`Propose` stores the payout amount without checking its sign, so a negative or zero payout is only refused by the bank keeper inside [`Execute`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L117). Both owners pay for an approval transaction on a proposal that can never settle, and with no cancel path it stays in the tree and in `Render` for good.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5951 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/negamt.txtar <<'EOF'
adduser ownerA
adduser ownerB
loadpkg gno.land/r/demo/multisig
gnoland start
gnokey maketx call -pkgpath gno.land/r/demo/multisig -func Setup -args $test1_user_addr -args $ownerA_user_addr -args $ownerB_user_addr -args 2 -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/demo/multisig -func Propose -args 'drain' -args $ownerA_user_addr -args -500000 -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/demo/multisig -func Approve -args 1 -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test test1
gnokey maketx call -pkgpath gno.land/r/demo/multisig -func Approve -args 1 -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test ownerA
! gnokey maketx call -pkgpath gno.land/r/demo/multisig -func Execute -args 1 -gas-fee 1000000ugnot -gas-wanted 10000000 -chainid=tendermint_test test1
stderr 'invalid coins error'
EOF

go test -v -run 'TestTestdata/negamt' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/negamt.txtar
```

```
> gnokey maketx call ... -func Propose -args 'drain' ... -args -500000 ... test1
(1 int)
OK!
> gnokey maketx call ... -func Approve -args 1 ... test1
OK!
> gnokey maketx call ... -func Approve -args 1 ... ownerA
OK!
> ! gnokey maketx call ... -func Execute -args 1 ... test1
Data: invalid coins error
Execute at gno.land/r/demo/multisig/multisig.gno:117
```
</details>

## examples/gno.land/r/demo/multisig/multisig.gno:83-100 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L83) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029337)
Nothing in the realm removes an approval or a proposal. Once a payout reaches the threshold, [`Execute`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L102-L118) settles it at any later block for any caller, so a proposal approved while the treasury was empty fires the moment someone funds the address. cw3-flex-multisig, the model the description names, expires proposals for this reason.

## examples/gno.land/r/demo/multisig/multisig_test.gno:8-24 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig_test.gno#L8) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029344)
Missing test: a threshold reached by distinct owners, and a payout that moves coins. Every test here is in-package and calls through the same `cur`, so [`TestApproveIncrementsCount`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig_test.gno#L62-L71) never passes 1 of 2 and only the manual Test13 run exercises the state machine. The suite is also order-dependent: `gno test -run TestProposeCreatesProposal` panics with `multisig not initialized`, and six of the eight fail standalone.

<details><summary>test cases</summary>

An external test package lets callers differ. Setup is one-shot and package state is shared across files, so this replaces `multisig_test.gno` rather than sitting beside it.

```go
package multisig_test

import (
	"chain"
	"chain/banker"
	"testing"

	"gno.land/r/demo/multisig"
)

func TestExecutePaysRecipient(cur realm, t *testing.T) {
	owner1 := address("g1owner1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	owner2 := address("g1owner2xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	owner3 := address("g1owner3xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	recipient := address("g1recipientxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

	multisig.Setup(cross(cur), owner1, owner2, owner3, 2)

	treasury := chain.PackageAddress("gno.land/r/demo/multisig")
	testing.IssueCoins(treasury, chain.Coins{{Denom: "ugnot", Amount: 1000}})

	testing.SetRealm(testing.NewUserRealm(owner1))
	id := multisig.Propose(cross(cur), "payout", recipient, 400)
	multisig.Approve(cross(cur), id)

	testing.SetRealm(testing.NewUserRealm(owner2))
	multisig.Approve(cross(cur), id)

	testing.SetRealm(testing.NewUserRealm(owner3))
	multisig.Execute(cross(cur), id)

	got := banker.NewReadonlyBanker().GetCoins(recipient)
	if got.String() != "400ugnot" {
		t.Fatalf("recipient balance = %s, want 400ugnot", got.String())
	}
}
```

The same flow across three real keys on a node, asserting `bank/balances`, is at [`tests/multisig_flow.txtar`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5951-multisig-treasury-demo-realm/1-9208bed41/tests/multisig_flow.txtar).
</details>

## examples/gno.land/r/demo/multisig/multisig.gno:14-22 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L14) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029348)
Nit: `Proposal` is exported but every field is unexported and there are no accessors, so the only way for another realm to read a proposal is to scrape `Render` output. `approveCnt` also duplicates what [`approvals.Size()`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/p/nt/avl/v0/tree.gno#L39) already knows, and `Execute` trusts the counter.

## examples/gno.land/r/demo/multisig/multisig.gno:79 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L79) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029352)
Nit: proposal ids are stored as decimal strings, so once there are ten proposals [`Render`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L132-L144) lists them as 1, 10, 11, 2.

## examples/gno.land/r/demo/multisig/multisig.gno:124 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L124) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029357)
Nit: `Render` takes `path` and never reads it, so `:1` and `:anything` both return the full list.

## examples/gno.land/r/demo/multisig/multisig.gno:34-37 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L34) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029362)
Suggestion: realms under `examples/` are uploaded at genesis, so on a public chain the account `init` records is the genesis deployer and no reader can run the demo. `Setup` refuses everyone else, so one owner set exists for the life of the realm. A factory that mints a multisig per caller would make it something people can try.

## examples/gno.land/r/demo/multisig/multisig.gno:102-118 [↗](../../../../../.worktrees/gno-review-5951/examples/gno.land/r/demo/multisig/multisig.gno#L102) [posted](https://github.com/gnolang/gno/pull/5951#discussion_r3613029368)
Suggestion: nothing here emits an event, so following the treasury means polling [`Render`](https://github.com/gnolang/gno/blob/9208bed41/examples/gno.land/r/demo/multisig/multisig.gno#L124-L146). A [`chain.Emit`](https://github.com/gnolang/gno/blob/9208bed41/gnovm/stdlibs/chain/emit_event.gno#L18) on `Propose` and `Execute` would let a wallet or indexer show pending payouts.
