# PR #5697: fix(gnofaucet): apply reward debit after claim succeeds, atomically

URL: https://github.com/gnolang/gno/pull/5697
Author: ajnavarro | Base: master | Files: 4 | +123 -8
Reviewed by: davd-gzl | Model: claude-opus-4.7

**Verdict: APPROVE** — two real bugs fixed (debit-before-drip destroys earned balance on any downstream rejection; non-atomic `Get`+`Set` loses concurrent debits — empirically ~85% lost under contention). Pre-existing cap-bypass TOCTOU between `GetReward` and `Apply` is out of scope.

## Summary

`gitHubClaimRewardsMiddleware` debited the user's earned contribution-reward counter *before* calling the downstream chain. If the cooldown gate, drip, or any later middleware rejected, the debit had already landed and the `Rewarder` interface has no `Refund` — the user's earned balance was permanently zeroed. A 2-line reorder moves `Apply` to run only after `next(ctx, req)` returns without an error. Independently, `RedisRewarder.Apply` was `Get`-then-`Set`, which two concurrent claims for the same user can interleave so one debit is silently dropped; the fix is `IncrBy`, atomic in Redis.

```
old: GetReward → Apply (DEBIT) → cooldown → drip   ← any failure here = lost balance
new: GetReward → cooldown → drip → Apply (DEBIT)   ← debit only on full success
```

## Glossary

- `Apply` — debits the user's cumulative-rewarded counter in Redis (`reward:<user>` key).
- `GetReward` — computes claimable amount = `min(score - previouslyRewarded, MaxReward - previouslyRewarded)`.
- `claimRPCMethod` (`"claim"`) — frontend method; the middleware rewrites it to `DefaultDripMethod` before forwarding.

## Fix

In [`gh.go:251-266`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/gh.go#L251-L266), `next(ctx, req)` now runs first; the response's `Error` field is inspected and the debit is skipped on any downstream error, returning the original downstream response unchanged. In [`github/rewarder.go:80-82`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github/rewarder.go#L80-L82), the racy `getCount` + `Set(amount + previouslyRewarded)` pair becomes a single `IncrBy(amount)`. Two new tests pin the behavior: [`gh_test.go:298-362`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/gh_test.go#L298-L362) drives the real chain via `getMiddlewares` (same wiring as production at [`github.go:171-178`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github.go#L171-L178)) and asserts zero `Apply` calls on cooldown/drip rejection and exactly one on success; [`rewarder_test.go:152-178`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github/rewarder_test.go#L152-L178) fires 50 concurrent debits and asserts the final counter equals the expected sum.

## Empirical confirmation of the race

The author's PR notes "lost 11 of 50 debits in a sanity-check run." Reverting just `Apply` to the old `Get`+`Set` form and running the new test 5× under `-race`:

```bash
cd /home/davd/Projects/gno-skills/.worktrees/gno-review-5697
git show HEAD~1:contribs/gnofaucet/github/rewarder.go > /tmp/old-rewarder.go
cp /tmp/old-rewarder.go contribs/gnofaucet/github/rewarder.go
go test -count=5 -race -v -run TestApply_ConcurrentDebitsAreAtomic ./contribs/gnofaucet/github/
git checkout HEAD -- contribs/gnofaucet/github/rewarder.go
```

Old impl: expected 350, observed 49 / 56 / 63 / 98 / 350. So `Apply` loses ~70-85% of overlapping debits under miniredis, far worse than the author's conservative estimate. New impl: 5/5 pass.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`gh.go:258-264`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/gh.go#L258-L264) — if `Apply` fails after a successful drip, response is rewritten to an "unable to apply reward" error, masking the fact the user already received tokens. Bounded by cooldown so impact is small, and Redis `IncrBy` rarely fails in practice; not worth additional logic, but a log line on the `Apply` failure (with username and amount) would let operators reconcile.

## Missing Tests

None — the two new tests are well-targeted. The timing test goes through the real `getMiddlewares` chain (not a stub), so it actually exercises the production wiring at [`github.go:171`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github.go#L171). The concurrency test verifies the race is gone (empirically demonstrated above).

## Suggestions

- [`github/rewarder.go:41-75`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github/rewarder.go#L41-L75) — pre-existing, not introduced here: the `GetReward` → `Apply` flow still has a TOCTOU on `MaxReward`. Two concurrent claims that both read `previouslyRewarded = 0` will each compute a reward up to `MaxReward`, then both `IncrBy`, pushing the counter past `MaxReward` (and making subsequent `GetReward` calls return negative via `total = MaxReward - previouslyRewarded` at [line 68](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/github/rewarder.go#L68)). Cooldown is the de facto guard today. A follow-up could either move the cap enforcement into `Apply` (e.g. Lua script that reads + checks + increments), or clamp `previouslyRewarded` in `GetReward`. Out of scope for this PR.

## Questions for Author

- The PR description mentions "lifetime-cap rejection" as one of the failure modes the new ordering protects against, but the cap is enforced in `GetReward` *before* `next`/`Apply` (`reward == 0` short-circuit at [`gh.go:238-244`](../../../../../.worktrees/gno-review-5697/contribs/gnofaucet/gh.go#L238-L244)). Is there a downstream lifetime-cap path I'm missing, or is the description referring to a hypothetical future check?
