# gnolang/gno [#6111](https://github.com/gnolang/gno/pull/6111): fix(gpao): debit max-spend at the broadcast, not at the decision to send

URL: https://github.com/gnolang/gno/pull/6111
Author: gfanton | Base: master | Files: 3 | +317 -7
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: b51a78a8a (latest)
Local checkout: `git -C gno worktree add ../.worktrees/gno-review-6111 b51a78a8a`
Overview: [overview](../overview.md)

## Overview

gpao is a daemon holding an approver key on a chain whose `CodeSubmissionPolicy` is `inert`: submitted packages are parked until gpao typechecks them off-chain and sends a `MsgEnablePackage`. `-max-spend` caps what one run will pay in gas fees for those approvals. The daemon counted a fee against that cap before entering the send, and [four returns inside the send](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L772-L825) built no transaction, so the counter drifted above what the key had paid. Enough drift and the daemon refuses well-formed packages as `blocked` while holding every coin it started with. The branch moves the count to the statement before `BroadcastTxCommit` and hands it back for a `CheckTx` rejection, which is the one refusal a chain gives for free.

**Verdict: APPROVE** — the accounting matches the money on every path out of `enable` the client can resolve, over-counting only the nil result that stands for a free refusal and a lost answer alike, and the four findings are a missing chain-level assertion, a stale line of `--help` text, a third copy of the node bring-up and a debit that fits in fewer lines (1 missing test, 2 Nits, 1 Suggestion).

## Verify first

- [`contribs/gpao/oracle.go:823-831`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L823-L831) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/oracle.go#L823) — the refund fires for exactly one outcome, and refunding a charged one walks the counter under the balance. Run [`zz_delivertxcharge_test.go`](tests/zz_delivertxcharge_test.go), which sends an enable that clears `CheckTx`, runs out of gas inside a block, and drops the balance by the flat fee.
- [`contribs/gpao/oracle.go:757-758`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L757-L758) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/oracle.go#L757) — `broadcastWasFree` reads `res.CheckTx` off a result that may be nil. Check the two shapes against [`client_txs.go:460-470`](https://github.com/gnolang/gno/blob/b51a78a8a/gno.land/pkg/gnoclient/client_txs.go#L460-L470): a nil result whenever the round trip failed, a non-nil one only on the two `IsErr` branches.

## Summary

`handleCandidate` did `o.spent += o.enableFee` before calling `enable`, and `enable` has [four returns that broadcast nothing](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L772-L825), the likeliest being a simulate that says the enable would fail. A package whose `init()` panics takes exactly that return, since gpao's verifier preprocesses rather than executes. Resubmitting one under `inert` is cheap. The branch deletes that line, adds the same debit at [`oracle.go:823`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L823) one statement before the broadcast, and refunds it at [`oracle.go:827-829`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L827-L829) when `broadcastWasFree` says the node refused the transaction before any block.

## Fix

Before, the count was a property of deciding to approve. After, it is a property of handing bytes to a node, which is the event the ante charges for. A client cannot always tell a free refusal from a lost answer. `gnoclient.BroadcastTxCommit` returns a nil result for a pre-mempool refusal, which costs nothing, and for a transaction whose answer was lost after it was handed over, which may have committed. The branch counts both, which stops the run early rather than letting it overspend.

## Benchmarks / Numbers

Every path from the `-max-spend` check at [`oracle.go:518`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L518) to a return from `enable`. Rows 1 to 4 return before any transaction is built; rows 5 to 8 are the four outcomes of one `BroadcastTxCommit`.

| Path out of `enable` | Bytes handed to a node | Chain charged | Counted before | Counted after |
| --- | --- | ---: | ---: | ---: |
| `-gas-fee` will not parse | no | 0 | one fee | 0 |
| signing the simulate probe failed | no | 0 | one fee | 0 |
| simulate says the enable would fail | no | 0 | one fee | 0 |
| signing the real transaction failed | no | 0 | one fee | 0 |
| no result came back | maybe | unknown | one fee | one fee |
| `CheckTx` refused it | yes | 0 | one fee | 0 |
| `DeliverTx` failed inside a block | yes | one fee | one fee | one fee |
| committed | yes | one fee | one fee | one fee |

`CheckTx` charges nothing because `runTx` returns on an ante abort at [`baseapp.go:917-919`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/sdk/baseapp.go#L917-L919), before the `MultiWrite` that flushes the ante's fee deduction at [`baseapp.go:928-931`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/sdk/baseapp.go#L928-L931). Every `Commit` rebuilds the check state from the committed store, at [`baseapp.go:1026`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/sdk/baseapp.go#L1026).

No path debits twice, and no path that moved money goes uncounted. Three conditions sit outside the table:

- **Context cancellation.** `enable` takes no context, and `gnoclient.BroadcastTxCommit` passes `context.Background()` at [`client_txs.go:460`](https://github.com/gnolang/gno/blob/b51a78a8a/gno.land/pkg/gnoclient/client_txs.go#L460). A shutdown during a broadcast neither cancels it nor reaches the counter.
- **Panic.** The only `recover` on this path is at [`verifier.go:79-83`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/verifier.go#L79-L83), inside `verifyPackage`. A panic inside `enable` ends the process, and `spent` lives in memory only, so the debit dies with the process.
- **Concurrent callers.** `enable` has one call site, [`oracle.go:531`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L531), reached only from the single `runVerifier` goroutine started at [`oracle.go:290`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L290). The debit and its refund cannot interleave with another approval.

Row 5 covers a partial send. `Mempool.CheckTx` refusing the transaction outright comes back as a nil result at [`mempool.go:77-83`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/bft/rpc/core/mempool.go#L77-L83), and two of its causes cost nothing: a full mempool at [`clist_mempool.go:231`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/bft/mempool/clist_mempool.go#L231) and a transaction already in the cache at [`clist_mempool.go:263`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/bft/mempool/clist_mempool.go#L263). The branch over-counts both by one fee, and the alternative is matching on an error string.

## Missing Tests

- **[accounting]** `contribs/gpao/spenddebit_test.go:236` — nothing asserts that a `DeliverTx` failure is charged, the one failure the chain charges for.
  <details><summary>details</summary>

  [`TestBroadcastWasFree`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L236) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L236) asserts the function returns `false` for a `ResponseDeliverTx` carrying an error, which is a restatement of the function body rather than a check on the chain. The two node tests cover a return before the broadcast and a `CheckTx` rejection, so nothing here confirms the premise all three tests rest on: that a transaction which clears `CheckTx` and fails inside a block was charged. The premise holds, and an ante change would break it silently.

  [`tests/zz_delivertxcharge_test.go`](tests/zz_delivertxcharge_test.go) signs an enable with `GasWanted` at 1,000,000, enough for the ante and short of running the enable, and reads the balance either side. Measured at b51a78a8a: `CheckTx` OK, `DeliverTx` out of gas, balance down by 1,000,000ugnot, `broadcastWasFree` false. 100,000 runs the ante out of gas in `CheckTx`, the refunded arm, and 5,000,000 lets the enable succeed.

  Fix: add the chain-level assertion for the `DeliverTx` arm beside the other two.
  </details>

## Nits

- **[operator-facing text]** `contribs/gpao/main.go:141-142` — `-max-spend`'s flag help still says every approval costs a fee whether or not it succeeds, the sentence the branch corrected in the README.
  <details><summary>details</summary>

  [`README.md:216-217`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/README.md?plain=1#L216-L217) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/README.md#L216) now reads "Every approval that reaches a block costs the full gas fee". The same claim sits in the flag help at [`main.go:141-142`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/main.go#L141-L142) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/main.go#L141), which is the text `gpao --help` prints. `main.go` carries no hunk in this diff, so the finding goes in the comment Body rather than on an anchor.

  Fix: end the help at "every approval that reaches a block costs a fee".
  </details>

- **[test duplication]** `contribs/gpao/spenddebit_test.go:150-175` — this 26-line node bring-up repeats `spenddebit_test.go:39-64` byte for byte and `endtoend_test.go:37-63` except for one comment line.
  <details><summary>details</summary>

  [`spenddebit_test.go:39-64`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L39-L64) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L39) and [`spenddebit_test.go:150-175`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L150-L175) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L150) repeat [`endtoend_test.go:37-63`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/endtoend_test.go#L37-L63) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/endtoend_test.go#L37): the same genesis balance, the same `inert` policy, the same single approver, the same in-memory node and client. Diffing the two new ranges prints nothing, and diffing either against the third prints one comment line. One shared helper folds 52 lines out of the new file and gives the next node test one place to start from: it returns the chain ID, the approver address and the client.

  The same file's new `testIO` helper at [`spenddebit_test.go:280-286`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/spenddebit_test.go#L280-L286) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/spenddebit_test.go#L280) duplicates inline copies at [`oracle_test.go:98-100`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle_test.go#L98-L100) and [`processblock_test.go:228-230`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/processblock_test.go#L228-L230), which can now call the helper instead.

  Fix: extract the bring-up into one helper the three tests share.
  </details>

## Suggestions

- **[fewer lines]** `contribs/gpao/oracle.go:816-831` — taking the debit after the call removes the refund arm.
  <details><summary>details</summary>

  The current shape at [`oracle.go:816-831`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L816-L831) · [↗](../../../../../.worktrees/gno-review-6111/contribs/gpao/oracle.go#L816) adds a fee, calls the node, and subtracts the fee again on one arm. `broadcastWasFree` already answers for all four outcomes on its own and needs no `err`: a nil result and a success both give `false`, which is a debit. Nothing reads `spent` between the two mutations: the only readers are [`oracle.go:526`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L526) and [`oracle.go:614`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L614), on the same goroutine. The win is the line count and one arm fewer, not a bug fix.

  ```go
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

  Sixteen lines to ten, one mutation instead of two, and the reason a no-result error is counted is stated once, at [`oracle.go:747-756`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L747-L756). Equivalence rests on `err == nil` implying a non-nil result whose `CheckTx` is OK, at [`client_txs.go:465-472`](https://github.com/gnolang/gno/blob/b51a78a8a/gno.land/pkg/gnoclient/client_txs.go#L465-L472). The patch is [`tests/oracle-single-debit.patch`](tests/oracle-single-debit.patch).

  Fix: replace the debit and the refund with the conditional debit.
  </details>

## Verified

- The kept debit matches a real charge. [`tests/zz_delivertxcharge_test.go`](tests/zz_delivertxcharge_test.go) sends an enable that clears `CheckTx`, runs out of gas in a block and leaves the balance 1,000,000ugnot lighter. No job reaches it: the test is not in the branch.
- With [`tests/oracle-single-debit.patch`](tests/oracle-single-debit.patch) applied, all 34 tests in `contribs/gpao` stay green, `gofmt` and `go vet` included.

## Open questions

- Three code comments still carry the claim the branch corrected in the README, so a reader who trusts them concludes the counter tracks approvals rather than blocks. [`oracle.go:512-513`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L512-L513) and [`oracle.go:610-611`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L610-L611) both say the fee is charged whether or not the message succeeds. [`main.go:33`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/main.go#L33) says every approval costs the full gas fee. Not posted: a finding on a comment's own wording stays in the review file. The `--help` copy of the same sentence is posted, because an operator reads it.
- `broadcastWasFree`'s comment at [`oracle.go:751`](https://github.com/gnolang/gno/blob/b51a78a8a/contribs/gpao/oracle.go#L751) says a `DeliverTx` failure "ran in a block and was charged", and an ante abort inside `DeliverTx` runs in a block without being charged: `runTx` returns at [`baseapp.go:917-919`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/sdk/baseapp.go#L917-L919) before the checkpoint at [`baseapp.go:941-950`](https://github.com/gnolang/gno/blob/b51a78a8a/tm2/pkg/sdk/baseapp.go#L941-L950) that flushes the fee. Read from source, not measured. Two shapes reach that abort, a balance or a sequence moving between the two runs of the ante, and both are hard to provoke on a chain whose only spender is this daemon. It over-counts, which is the direction the branch already chose for a nil result, so the code needs no change. Not posted: it is a comment's wording, and the branch's own accounting is unaffected.
