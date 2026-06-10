# PR #5793: chore: Add session_multi_sigs_test.go

URL: https://github.com/gnolang/gno/pull/5793
Author: jefft0 | Base: master | Files: 3 | +192 -13
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: b2e0b5641 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5793 b2e0b5641`

**Verdict: APPROVE** — test-only chore; the helper refactor is correct and all four tests pass under CI. Two things to reconcile before merge: the PR body says "the other tests fail," but the Go tests pass (they assert the *transaction* is rejected), and the rejection assertions check only the error *type*, so they will silently survive the eventual #5731 fix instead of flagging it.

## Summary
Adds `session_multi_sigs_test.go` (4 tests) plus a helper refactor for issue #5731, where `tx.GetSigners()` deduplicates master addresses while `tx.GetSignatures()` does not, so a tx with two session signatures under one master breaks the `len(signers) == len(signatures)` contract. `TestTwoSessionSignaturesTwoMasters` is the nominal case (two sessions, two distinct masters, two messages) and passes. The other three feed scenarios that currently violate the contract and assert the tx is *rejected* via `checkInvalidTx(..., std.UnauthorizedError{})`. The rejection comes from `ValidateBasic`'s count check ([`tx.go:56-57`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/std/tx.go#L56-L57) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/std/tx.go#L56-L57)) run inside the AnteHandler ([`ante.go:102`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante.go#L102) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante.go#L102)). The helper refactor factors `setupSessionFromEnv` out of `setupSessionEnv` (multiple masters in one env) and `NewSessionTestSignature` out of `NewSessionTestTx` (append extra session sigs to a tx). No production code changes; `NewSessionTestTx`'s signature is unchanged, so the `gno.land/pkg/gnoland` caller is unaffected.

```
GetSigners() dedups by address          GetSignatures() never dedups
  msg[Caller=M], msg[Caller=M]  → [M]      sig{S1}, sig{S2}        → [S1, S2]
                                  len 1                              len 2
                                     └────── len mismatch ──────────┘
                              ValidateBasic → UnauthorizedError("wrong number of signers")
```

## Glossary
- `GetSigners` — tx method returning the addresses that must sign; dedups by address ([`tx.go:83`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/std/tx.go#L83) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/std/tx.go#L83)).
- `SessionAddr` — sub-account address carried on a `std.Signature`; the master signs via a delegated session key.
- `checkInvalidTx` — test helper asserting the AnteHandler aborts with a given error *type* ([`ante_test.go:38-44`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante_test.go#L38-L44) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante_test.go#L38-L44)).

## Critical (must fix)
None.

## Warnings (should fix)
- **[PR description contradicts the green suite]** PR body — "the other tests fail" reads as "the Go tests are red," but all four pass.
  <details><summary>details</summary>

  The body says `TestTwoSessionSignaturesTwoMasters` passes "but the other tests fail because they show the edge cases which aren't handled." All four tests pass (verified locally and reflected by green CI), because the three edge-case tests use `checkInvalidTx` to assert the *transaction* is rejected, not that the Go test fails. A reviewer reading "the other tests fail" and then seeing a green check will be confused about what the PR demonstrates. Fix: reword to something like "the other three tests document edge cases that are currently rejected (the transaction fails `ValidateBasic`), pending the #5731 fix." The in-file comments are already clear; only the PR description is misleading.
  </details>

## Nits
- `session_multi_sigs_test.go:11` — "trasaction" typo in the `TestTwoSessionSignaturesTwoMasters` doc comment.

## Missing Tests
- **[assertion too coarse to detect the #5731 fix]** [`session_multi_sigs_test.go:77,110,154`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L77) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/session_multi_sigs_test.go#L77) — the three invalid tests assert only `UnauthorizedError`, which several distinct rejections share.
  <details><summary>details</summary>

  `checkInvalidTx` compares only the error *type* ([`ante_test.go:44`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante_test.go#L44) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante_test.go#L44)). `ValidateBasic`'s "wrong number of signers" ([`tx.go:57`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/std/tx.go#L57) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/std/tx.go#L57)), "unknown session" ([`ante.go:124`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante.go#L124) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante.go#L124)), and "signature verification failed" all return `std.UnauthorizedError`. I confirmed the current rejection is the count check (repro below), but the assertion does not pin that. The hazard is forward: when #5731 is fixed (moul's Option 3, `GetSignerInfos` deduped by `(caller, pubkey)`), these txs should become *accepted*, but if instead they start rejecting for a *different* `UnauthorizedError` reason (e.g. a session-resolution path that now runs), the test stays green and silently masks a regression. Fix: assert the log message too, e.g. `require.Contains(t, result.Log, "wrong number of signers")`, so the test breaks loudly the moment the rejection cause moves.

  Repro — current rejection is the count check, not session resolution:

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

## Suggestions
- `session_multi_sigs_test.go:49-78` — consider whether these should assert the *desired* behavior instead of the current rejection.
  <details><summary>details</summary>

  The PR's stated purpose is "a test for the problem." As written, the three edge-case tests lock in the current (buggy) rejection as if it were correct, so when #5731 is fixed they must be manually rewritten rather than simply flipping from red to green. An alternative that captures intent more faithfully: assert the *target* behavior (`checkValidTx`) and gate with `t.Skip("blocked on #5731")`, or keep `checkInvalidTx` but add a side-by-side commented assertion of the post-fix expectation (the pattern the review skill uses for adversarial tests). Either makes the "this isn't handled yet" status machine-visible. This is a judgment call for the author/maintainers, not a blocker — the in-file comments ("However, this would succeed if we deduplicate the combined signer-addr/signature-pubkey") already document the gap.
  </details>
- `session_multi_sigs_test.go:118` — `TestThreeSessionSignaturesTwoMasters` is worth a one-line comment that `ValidateBasic` is the *only* guard here.
  <details><summary>details</summary>

  With 2 deduped signers and 3 signatures, if `ValidateBasic` did not reject first, Phase 3 ([`ante.go:189-194`](https://github.com/gnolang/gno/blob/b2e0b5641/tm2/pkg/sdk/auth/ante.go#L189-L194) · [↗](../../../../../.worktrees/gno-review-5793/tm2/pkg/sdk/auth/ante.go#L189-L194)) iterates over all three `stdSigs` and indexes `signerAddrs[i]` (len 2) → out-of-range panic, which the AnteHandler's `defer`/`recover` does not catch (it only recovers `OutOfGasError`). This is moul's point in the issue thread: the count check is load-bearing. Noting it in the test makes clear why this scenario must never bypass `ValidateBasic`.
  </details>

## Questions for Author
- Is the intent of the three failing-scenario tests characterization (lock in today's rejection) or demonstration (drive the #5731 fix)? The answer decides whether `checkInvalidTx` or a skipped `checkValidTx` is the right shape — see the Suggestions.
