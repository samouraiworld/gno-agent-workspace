# Review: [#6111](https://github.com/gnolang/gno/pull/6111)
Event: APPROVE

## Body
The counter follows the ante on every path out of [`enable`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L772) that the client can resolve, and over-counts only the nil result that stands for a free refusal and a lost answer alike.

- `-max-spend`'s flag help at [`main.go:141-142`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/main.go#L141-L142) still says every approval costs a fee whether or not it succeeds, without the "that reaches a block" the branch added to [`README.md:216`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/README.md?plain=1#L216).

## contribs/gpao/spenddebit_test.go:236 [gh](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L236) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L236)
Missing test: nothing asserts that a `DeliverTx` failure is charged, the only failure that costs anything.

<details><summary>test cases</summary>

An enable at 1,000,000 `GasWanted` clears `CheckTx`, runs out of gas inside a block, and leaves the balance 1,000,000ugnot lighter, so [`broadcastWasFree`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L757-L758) returns false against a charge that happened. The value picks the arm: 100,000 runs the ante out of gas in `CheckTx`, the refunded arm, and 5,000,000 lets the enable succeed.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6111 -R gnolang/gno
curl -fsSL -o contribs/gpao/zz_delivertxcharge_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6111-debit-max-spend-at-broadcast/1-b51a78a8a/tests/zz_delivertxcharge_test.go
cd contribs/gpao && go test -v -run 'TestDeliverTxFailureIsCharged' .
rm zz_delivertxcharge_test.go
```
</details>

## contribs/gpao/spenddebit_test.go:150-175 [gh](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L150-L175) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L150)
Nit: this 26-line node bring-up repeats [`spenddebit_test.go:39-64`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L39-L64) byte for byte and [`endtoend_test.go:37-63`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/endtoend_test.go#L37-L63) except for one comment line.

## contribs/gpao/oracle.go:816-831 [gh](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L816-L831) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/oracle.go#L816)
Refactor: taking the debit after the call removes the refund arm.

```suggestion
	// Counted for what the node was charged for. The paths above broadcast
	// nothing, so the ante charges nothing; of what is sent, only a CheckTx
	// rejection is free.
	res, err := o.client.BroadcastTxCommit(signed)
	if !broadcastWasFree(res) {
		o.spent += o.enableFee
	}
	if err != nil {
		return fmt.Errorf("broadcast: %w", err)
	}
```

<details><summary>equivalence</summary>

`broadcastWasFree` answers for all four outcomes on its own and needs no `err`: a nil result and a success both give false, which is a debit. `err == nil` already implies a non-nil result whose `CheckTx` is OK, at [`client_txs.go:465-472`](https://github.com/gnolang/gno/blob/b51a78a8a/gno.land/pkg/gnoclient/client_txs.go#L465-L472). [`oracle.go:747-756`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L747-L756) already gives the reason a no-result error stays counted, so the replaced comment stated it twice.

With the replacement applied, all 34 tests in `contribs/gpao` stay green, `gofmt` and `go vet` included.
</details>
