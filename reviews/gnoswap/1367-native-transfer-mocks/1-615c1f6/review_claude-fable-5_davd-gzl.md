# PR [#1367](https://github.com/gnoswap-labs/gnoswap/pull/1367): test: pay tokens via native transfers instead of the grc20 registry hub

URL: https://github.com/gnoswap-labs/gnoswap/pull/1367
Author: moul | Base: main | Files: 41 | +463 -150
Reviewed by: davd-gzl | Model: claude-fable-5 (deep) | Commit: 615c1f6 (latest)
Local worktree: `git clone https://github.com/gnoswap-labs/gnoswap && gh pr checkout 1367 -R gnoswap-labs/gnoswap`

**TL;DR:** A test-only sweep across 24 test files and 17 scenario filetests: test callbacks and helpers stop moving tokens through gnoswap's shared registry helper (`common.SafeGRC20Transfer` / `SafeGRC20Approve`) and call each token's own `Transfer`/`Approve` instead, via small per-package switch helpers. Two suites also change who the tests pretend to be: gov/staker and launchpad tests now run module-owned transfer flows as the module's code realm instead of as the admin user. No production code and no assertions change.

**Verdict: REQUEST CHANGES** — the mechanical replacements are behavior-preserving and the two realm-context fixes are genuinely right, but the comments shipped with the new helpers attribute the change to a grc20 "origin-guard" that does not exist in the gno tip CI pins, a claim a revert run disproves (2 Warnings, 3 Nits, 3 Suggestions).

## Verify first

- [`pool/v1/_helper_test.gno:435-439`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L435-L439) against the fork's [`grc20/tellers.gno:11-22`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L11-L22): read `CallerTeller` and its write paths and confirm no guard forbids a foreign realm from debiting the origin — then decide what the comment should actually say before this wording lands in three files.
- [`staker_reward_test.gno:538`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L538) and sibling rows: confirm `from: admin` rows should become `from: govStakerAddr` now that the debit actor is the staker code realm (CodeRabbit's finding; the fix is one field per row).
- [`router/v1/swap_callback.gno:89-92`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/swap_callback.gno#L89-L92): confirm the PR body's "production never funds the pool that way" is narrower than stated — the router's own callback funds the pool through the hub when it is the payer.

## Summary

Every replaced call is a pure actor-preserving rewrite: the old path crossed into `common`, whose `CallerTeller` debits `Previous()` — the realm set by the test; the new path crosses into the token realm directly, whose own `CallerTeller` debits the same `Previous()`. One hop removed, same address debited, both panic on failure. The two realm-context changes are real fidelity fixes: launchpad refund tests previously debited the admin EOA while the `obl` funding parked at `launchpadAddr` sat unused (masked by admin's large balance), and both changed suites now run as the proxy realm that production actually threads to the debit ([`gov/staker/proxy.gno:54`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/proxy.gno#L54), [`launchpad/proxy.gno:49`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/launchpad/proxy.gno#L49), fund holders per [`rbac/consts.gno:25`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/rbac/consts.gno#L25) and [`:27`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/rbac/consts.gno#L27)). The flaw is the stated rationale: the helper comments claim a patched grc20 rejects the old pattern and that it "broke EOA-context test callbacks", but no such guard exists in the gno master CI pins, 36 sibling filetests still use the old pattern and pass on this head, and reverting the pool mock to the hub helper still passes the suite.

Reading order: [`pool/v1/_helper_test.gno`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L434-L476) (helper shape and the contested comment), then [`staker_reward_test.gno`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L413-L420) and [`launchpad_project_test.gno`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/launchpad/v1/launchpad_project_test.gno#L1155-L1158) (the two realm-context changes), then router tests (reuse of existing `TokenApprove`), then fuzz helpers, then the 17 filetests (mechanical `payToken` copies).

## Diagram

```
before                                          after
------                                          -----
callback frame (SetRealm: admin)                callback frame (SetRealm: admin)
  └─ cross → common hub                           └─ cross → token realm (bar)
       SafeGRC20Transfer                               Transfer
       └─ token CallerTeller                           └─ token CallerTeller
          debits Previous(common) = admin                 debits Previous(bar) = admin
```

Same debited address either side; the diff only removes the middle hop. The gov/staker and launchpad changes move the `SetRealm` value from admin to the module's proxy realm — there the debited address does change, from the admin EOA to the module address production debits.

## Fix

Not a fix PR; the change is a test-fidelity refactor. The load-bearing constraint is that `CallerTeller` debits `rlm.Previous().Address()` ([`tellers.gno:17-19`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L17-L19)), so removing the hub hop preserves the actor, and `testing.SetRealm` mutates the frame's captured realm in place ([`context_testing.go:57-125`](https://github.com/gnoswap-labs/gno/blob/959cefd/gnovm/tests/stdlibs/testing/context_testing.go#L57-L125)), so a later `cross(cur)` carries the overridden identity in both the old and new shape.

## Critical (must fix)

None.

## Warnings (should fix)

- **[comments document a guard that does not exist]** `pool/v1/_helper_test.gno:435-439` — the new helpers' comments attribute the change to a grc20 origin-guard absent from the gno tip CI pins, and the "broke EOA-context test callbacks" claim fails a revert run.
  <details><summary>details</summary>

  Three files carry the claim in three different phrasings: [`pool/v1/_helper_test.gno:435-439`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L435-L439) ("The patched grc20 forbids a CallerTeller from a foreign realm ... which broke EOA-context test callbacks"), [`position/v1/_helper_test.gno:493-495`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/position/v1/_helper_test.gno#L493-L495) ("avoids the grc20 origin-guard that rejects ..."), [`test/fuzz/_helper_test.gno:214-216`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L214-L216) ("required by the grc20 origin-guard ..."). On gnoswap-labs/gno master 959cefd — the exact ref CI clones ([`run_test.yml:71`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/.github/workflows/run_test.yml#L71)) — `CallerTeller` resolves `rlm.Previous().Address()` with only `IsCurrent` spoof checks in the write paths ([`tellers.gno:11-22`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L11-L22), [`:115-135`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L115-L135)); `ErrNotRealm` guards only [`NewToken`](https://github.com/gnoswap-labs/gno/blob/959cefd/examples/gno.land/p/demo/tokens/grc20/token.gno#L39). No open fork PR adds such a guard. Counter-evidence to "broke": 36 scenario filetests keep the exact replaced idiom and pass CI on this head (e.g. [`position_decrease_increase_filetest.gno:227-239`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/scenario/position/position_decrease_increase_filetest.gno#L227-L239), same directory as the 17 converted ones), and reverting the pool mock to `common.SafeGRC20Transfer` passes `pool/v1` locally at this head ([repro](comment_claude-fable-5.md)). A maintainer who greps for the guard will not find it, and a future reader will treat a nonexistent invariant as load-bearing. The fidelity motivation is real and sufficient on its own: production drives the hub only as a module realm debiting its own address ([`swap_callback.gno:89-92`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/swap_callback.gno#L89-L92), [`staker_reward.gno:299`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward.gno#L299)), never as an EOA. Note the PR body's "production never funds the pool that way" overshoots for the same reason: the router's own callback funds the pool via `SafeGRC20Transfer` when it is the payer. Fix: reword the three comments (and the PR body) to the fidelity rationale, or to future tense with a link to the pending grc20 change that will add the guard.
  </details>

- **[balance check and debit now hit different addresses]** [@coderabbitai](https://github.com/gnoswap-labs/gnoswap/pull/1367#discussion_r3699491673) `gov/staker/v1/staker_reward_test.gno:538` — scenario rows keep `from: admin` while the debit actor is now `govStakerAddr`, so the guard is checked against one account and the transfer debits another.
  <details><summary>details</summary>

  [`transferToken`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward.gno#L265-L299) prechecks `common.BalanceOf(tokenPath, from)` then debits the caller realm via `common.SafeGRC20Transfer(cross(rlm), ...)`. After the [`SetRealm` change at :625](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L625), `TestStakerReward_TokenTransferScenarios` rows with [`from: admin`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L538) check admin's balance but debit `govStakerAddr` — a from/actor split production cannot reach, since production always passes `from = rlm.Address()` ([`staker_reward.gno:53`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward.gno#L53)). Safe today only because the setup funds `govStakerAddr` far above the row amounts ([`:767-773`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L767-L773)); if it ever were not, the function would panic inside `SafeGRC20Transfer` instead of returning the error the rows assert. The companion [`TestStakerReward_transferToken`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/gov/staker/v1/staker_reward_test.gno#L343-L407) already uses `from: govStakerAddr` throughout and is coherent. Raised by CodeRabbit (SKIPped in the draft). Fix: set `from: govStakerAddr` in the affected rows; a balance-movement assertion on source and destination would also restore what the suite lost — no test now verifies the debited account is the checked account.
  </details>

## Nits

- **[formatter undoes 16 files]** `contract/r/scenario/position/position_unclaimed_fee_filetest.gno:314-315` — 16 of the 17 converted filetests add a double blank line before `// Output:`; `gno fmt` collapses it (run locally: one line deleted, nothing else). Base files are clean, so the diff introduces it; the repo's `make fmt` (gofumpt) would rewrite all 16, and tlin never ran on this PR (its checkout step failed). Fix: run `gno fmt` over `contract/r/scenario/position/`.
- **[literals beside their own constants]** 13 filetests define `barPath`/`fooPath` yet their `payToken` switches on raw strings (e.g. [`position_collect_fee_filetest.gno:251-259`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/scenario/position/position_collect_fee_filetest.gno#L251-L259)); only `contract_owned_position_lifecycle_filetest.gno` uses the constants, and the panic text carries two wordings across the PR (`"test: ..."` vs `"mockSwapCallback: ..."`). Cosmetic; no enabled linter enforces it — not posted.
- **[silent-continue became abort]** [`router/v1/swap_single_test.gno:86-87`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/swap_single_test.gno#L86-L87) — old `common.Approve` returned an error the tests discarded; the token realms' `Approve` panics on failure. Only reachable on invalid inputs these tables never pass. No change needed — not posted.

## Missing Tests

None beyond the balance-movement assertion folded into the second Warning.

## Suggestions

- **[first wugnot pool test will panic]** [`pool/v1/_helper_test.gno:440-476`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L440-L476) — the pool switches omit `wugnotPath` while the file declares it ([`:32`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L32)) and [`approveTokensForPool`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/pool_test.gno#L991-L996) feeds arbitrary `pool.Token0Path()` into them.
  <details><summary>details</summary>

  No pool/v1 test uses wugnot today, so the panic is latent; the old hub accepted any registered token, the new switch panics on the first GNOT-pool test someone adds in pool/v1. Fix: add the `wugnotPath` case now, matching the position helper.
  </details>
- **[unknown-token policy diverges within one helper file]** [`test/fuzz/_helper_test.gno:209`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L209) — pre-existing `mintTestToken` silently returns on an unknown path while the new sibling `approveToken`/`transferToken` panic ([`:230`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L230), [`:249`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/test/fuzz/_helper_test.gno#L249)).
  <details><summary>details</summary>

  If `defaultTokenPaths` ever grows, the mint no-ops silently and the run fails later at approve with a message pointing at the wrong stage. No current input triggers it. Fix: make `mintTestToken` panic too.
  </details>
- **[two approve dispatchers with different realm semantics]** [`pool/v1/_helper_test.gno:459`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L459) — pool and position each now carry the new `approveToken` (uses the caller's current frame) beside the pre-existing `TokenApprove` ([`:164`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/pool/v1/_helper_test.gno#L164), position [`:165`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/position/v1/_helper_test.gno#L165)) which silently `SetRealm`s to its `owner` argument; the router files resolved the identical problem by reusing `TokenApprove` instead ([`base_test.gno:134`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/router/v1/base_test.gno#L134)).
  <details><summary>details</summary>

  Two near-identical helpers per file, one of which switches realm as a side effect, invite picking the wrong one. Position genuinely needs the frame-preserving variant (one caller runs under a code realm, [`position_test.gno:806-812`](https://github.com/gnoswap-labs/gnoswap/blob/615c1f68/contract/r/gnoswap/position/v1/position_test.gno#L806-L812), which `TokenApprove`'s forced `NewUserRealm` would break). Either converge on one resolution per file or add a one-line comment on each `approveToken` saying when to use which.
  </details>

## Verified

- Reverting the change in the pool swap mock — putting `common.SafeGRC20Transfer(cross(cur), ...)` back in `mockSwapCallback` — still passes the full `pool/v1` suite at 615c1f6 against gnoswap-labs/gno master 959cefd, the ref CI pins. The "broke EOA-context test callbacks" comment does not reproduce.
- `gno fmt` over a converted filetest deletes exactly the doubled blank line, confirming the formatting nit is real and mechanical.
- All touched suites pass locally at 615c1f6 against the fork tip: `gov/staker/v1`, `protocol_fee/v1`, `launchpad/v1`, `pool/v1`, `position/v1`, `router/v1`, `scenario/position` (filetests, 55s). The `test/fuzz` suite was still executing locally at drafting (the seed rotates per run); its CI job passes at this head in 18m26s.
- Static equivalence of every mechanical replacement (same debited address before and after) traced through the fork's `testing.SetRealm` in-place frame mutation and `cross` identity check; the two suites where the debited address deliberately changes are gov/staker and launchpad, both matching what their production proxies thread.

## Existing threads

- CodeRabbit, `staker_reward_test.gno:625`: scenario rows validate the wrong source address — overlaps the [balance check and debit now hit different addresses] Warning; [thread](https://github.com/gnoswap-labs/gnoswap/pull/1367#discussion_r3699491673).

## Open questions

- The old hub helpers implicitly asserted every test token is grc20reg-registered; the direct dispatch never touches the registry. Registration is still exercised by production-path tests, so no gap today — not posted, no change needed.
- If the anticipated grc20 origin-guard lands with realm-owner-only semantics, the 36 remaining hub-idiom filetests and the txtar integration wrappers become the exposure, not these converted files — deferred scope, not this PR's problem.
- tlin-check is red on an `actions/checkout` fetch flake (three git exit-1 retries, lint never ran) — needs a job re-run, not a code fix.
