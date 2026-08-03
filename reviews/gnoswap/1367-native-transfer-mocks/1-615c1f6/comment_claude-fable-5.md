# Review: PR [#1367](https://github.com/gnoswap-labs/gnoswap/pull/1367)
Event: REQUEST_CHANGES

## Body
Putting the hub call back in the pool swap mock still passes pool/v1 at 615c1f6 against the gno master CI pins, so the breakage the new helper comments describe does not reproduce. Details in the first inline comment.

- The PR body says production never funds the pool through the hub. The [router's own callback does](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/swap_callback.gno#L89-L92) when the router is the payer, debiting its own address. The invariant the tests mirror is narrower: no EOA ever pays through the hub.
- tlin-check is red on a checkout fetch flake and never ran. Not a code problem; re-run the job.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/gnoswap/1367-native-transfer-mocks/1-615c1f6/review_claude-fable-5_davd-gzl.md [↗](review_claude-fable-5_davd-gzl.md)

## contract/r/gnoswap/pool/v1/_helper_test.gno:435-439 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L435-L439)
This comment describes a grc20 guard the gno master CI pins do not have: [`CallerTeller`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L11-L22) still resolves `rlm.Previous()` with only [`IsCurrent` checks](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L115-L135) on the write paths, and restoring `common.SafeGRC20Transfer` here still passes pool/v1. The same claim ships in three wordings, here, in [position](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/position/v1/_helper_test.gno#L493-L495) and in [fuzz](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L214-L216), while 36 scenario filetests keep the replaced idiom, for example [`position_decrease_increase_filetest.gno:227-239`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/scenario/position/position_decrease_increase_filetest.gno#L227-L239). State the fidelity rationale instead, that production drives the hub only as a module realm debiting its own address, or point at the pending grc20 change in future tense.

<details><summary>repro</summary>

```bash
# from a local clone of gnoswap-labs/gnoswap:
gh pr checkout 1367 -R gnoswap-labs/gnoswap
git clone --depth 1 https://github.com/gnoswap-labs/gno.git gno -b master
(cd gno && make install.gno)
python3 setup.py -w .
# put the pre-PR hub call back in the pool swap-callback mock:
sed -i \
  -e 's|transferToken(cur, token0Path, poolAddr, amount0Delta)|common.SafeGRC20Transfer(cross(cur), token0Path, poolAddr, amount0Delta)|' \
  -e 's|transferToken(cur, token1Path, poolAddr, amount1Delta)|common.SafeGRC20Transfer(cross(cur), token1Path, poolAddr, amount1Delta)|' \
  contract/r/gnoswap/pool/v1/_helper_test.gno
(cd gno/examples && gno test ./gno.land/r/gnoswap/pool/v1)
git checkout -- contract/r/gnoswap/pool/v1/_helper_test.gno
```

```
ok      ./gno.land/r/gnoswap/pool/v1 	137.14s
```
</details>

## SKIP contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno:538 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L538)
Already raised: https://github.com/gnoswap-labs/gnoswap/pull/1367#discussion_r3699491673
The scenario rows keep `from: admin` while the debit actor is now `govStakerAddr`, so [`transferToken`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward.gno#L265-L299) prechecks one account and debits another, a split production cannot reach. Set `from: govStakerAddr` in those rows.

## contract/r/scenario/position/position_unclaimed_fee_filetest.gno:314-315 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/scenario/position/position_unclaimed_fee_filetest.gno#L314-L315)
Nit: 16 of the 17 converted filetests add a second blank line before `// Output:` and `gno fmt` deletes it. Run `gno fmt` over `contract/r/scenario/position/`.

## contract/r/gnoswap/pool/v1/_helper_test.gno:440-476 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L440-L476)
Suggestion: the pool switches omit `wugnotPath` while [the file declares it](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L32) and [`approveTokensForPool`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/pool_test.gno#L991-L996) feeds any `pool.Token0Path()` into them. The old hub accepted any registered token, so the first wugnot pool test now panics; no current test hits it. Add the `wugnotPath` case, matching the position helper.

## contract/r/gnoswap/pool/v1/_helper_test.gno:459 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L459)
Suggestion: `approveToken` lands beside [`TokenApprove`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L164), which silently switches realm to its `owner` argument, and the router files [reused `TokenApprove`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/base_test.gno#L134) for the same problem. Position needs the frame-preserving variant for its [code-realm caller](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/position/v1/position_test.gno#L806-L812). Pick one per file, or say on each helper when to use which.

## contract/r/gnoswap/test/fuzz/_helper_test.gno:230 [↗](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L230)
Suggestion: the new helpers panic on an unknown path while [`mintTestToken`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L209) silently returns on one. If [`defaultTokenPaths`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L77-L83) grows, the mint no-ops and the run fails later at approve, pointing at the wrong stage. Make `mintTestToken` panic too.
