# Review: PR #5793
Event: APPROVE

## Body
Looks good. Test-only chore; the helper refactor is correct and all four tests pass. Verified on b2e0b5641: the three edge-case txs are rejected by `ValidateBasic`'s signer-count check ([`tx.go:56-57`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/std/tx.go#L56-L57) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/std/tx.go#L56-L57)), `signers=1 sigs=2`, not a later session-resolution path.

- PR body says `TestTwoSessionSignaturesTwoMasters` passes but "the other tests fail"; all four pass, since the three edge-case tests use `checkInvalidTx` to assert the transaction is rejected, not that the Go test fails. Reword to something like "the other three document edge cases currently rejected by `ValidateBasic`, pending the #5731 fix."

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5793-session-multi-sigs-test/1-b2e0b5641/claude-opus-4-8_davd-gzl.md [↗](claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## tm2/pkg/sdk/auth/session_multi_sigs_test.go:11 [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L11)
"trasaction" typo in the `TestTwoSessionSignaturesTwoMasters` doc comment.

*(AI Agent)*

## tm2/pkg/sdk/auth/session_multi_sigs_test.go:77 [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L77)
The three invalid tests assert only `UnauthorizedError`, but the count check, the unknown-session path, and signature-verification failure all return that same type, so when #5731 lands and the rejection cause moves the tests stay green and mask the change. Assert the log message too, e.g. `require.Contains(t, result.Log, "wrong number of signers")`, so they break loudly. Same applies at :110 and :154.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5793 -R gnolang/gno
cat > tm2/pkg/sdk/auth/zz_probe_test.go <<'EOF'
package auth

import (
	"testing"

	tu "github.com/gnolang/gno/tm2/pkg/sdk/testutils"
	"github.com/gnolang/gno/tm2/pkg/std"
)

func TestProbeOneMaster(t *testing.T) {
	env, anteHandler, _, masterAddr := setupSessionEnv(t)
	ctx := env.ctx
	p1, pub1, a1 := tu.KeyTestPubAddr()
	p2, pub2, a2 := tu.KeyTestPubAddr()
	sa1 := createSessionDirect(t, env, masterAddr, pub1, ctx.BlockTime().Unix()+3600)
	sa2 := createSessionDirect(t, env, masterAddr, pub2, ctx.BlockTime().Unix()+3600)
	msgs := []std.Msg{tu.NewTestMsg(masterAddr), tu.NewTestMsg(masterAddr)}
	fee := tu.NewTestFee()
	tx := tu.NewSessionTestTx(t, ctx.ChainID(), msgs, p1, a1, sa1.GetAccountNumber(), sa1.GetSequence(), fee)
	tx.Signatures = append(tx.Signatures, tu.NewSessionTestSignature(t, ctx.ChainID(), msgs, p2, a2, sa2.GetAccountNumber(), sa2.GetSequence(), fee))
	_, res, abort := anteHandler(ctx, tx, false)
	t.Logf("signers=%d sigs=%d abort=%v\n%s", len(tx.GetSigners()), len(tx.GetSignatures()), abort, res.Log)
}
EOF
go test -v -run TestProbeOneMaster ./tm2/pkg/sdk/auth/ 2>&1 | grep -E 'signers=|wrong number|tx.go:57|ante.go:102|PASS'
rm tm2/pkg/sdk/auth/zz_probe_test.go
```

```
    zz_probe_test.go:21: signers=1 sigs=2 abort=true
    0  gno/tm2/pkg/std/errors.go:81 - wrong number of signers
    2  gno/tm2/pkg/std/tx.go:57
    3  gno/tm2/pkg/sdk/auth/ante.go:102
--- PASS: TestProbeOneMaster (0.01s)
```
</details>

*(AI Agent)*

## tm2/pkg/sdk/auth/session_multi_sigs_test.go:49-78 [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L49)
As written, the three edge-case tests lock in the current buggy rejection as correct, so when #5731 is fixed they must be manually rewritten rather than flipping red to green. Consider asserting the target behavior (`checkValidTx`) gated with `t.Skip("blocked on #5731")`, or keep `checkInvalidTx` with a side-by-side commented post-fix assertion. Judgment call for the author, not a blocker.

*(AI Agent)*

## tm2/pkg/sdk/auth/session_multi_sigs_test.go:118 [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L118)
With 2 deduped signers and 3 signatures, `ValidateBasic` is the only guard: if it did not reject first, Phase 3 ([`ante.go:189-194`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante.go#L189-L194) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante.go#L189-L194)) indexes `signerAddrs[i]` out of range and panics, uncaught by the AnteHandler's recover (it only catches `OutOfGasError`). Worth a one-line comment saying the count check is load-bearing here.

*(AI Agent)*
