# PR [#5998](https://github.com/gnolang/gno/pull/5998): feat(gov/dao): named sentinel errors for VoteOnProposal

URL: https://github.com/gnolang/gno/pull/5998
Author: ygd58 | Base: master | Files: 2 | +118 -5
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cf75d982a (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5998 cf75d982a`

**TL;DR:** When a govDAO vote fails, the DAO used to hand back a plain sentence like "already voted on proposal", so anything calling it had to match on that sentence to know what went wrong. This PR gives each failure its own named error value, so a caller can ask "was it this one?" instead of reading the text.

**Verdict: NEEDS DISCUSSION** — no correctness defect found; the open call is where the sentinels should live, because declared on the versioned implementation they stop matching the moment govDAO is upgraded (1 Warning, 2 Nits, 1 Missing test, 1 Suggestion).

## Summary

[`VoteOnProposal`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L107-L151) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L107-L151) built each of its five failures with `errors.New` at the return site, so the only way to tell them apart was the message string. The PR hoists five of them into package-level sentinels and turns the closed-proposal case, which carries the accepted/denied flag, into a `*ProposalClosedError` with an [`Is(error) bool`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L45-L47) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L45-L47) method so it matches a single sentinel regardless of outcome. Every `.Error()` string is byte-identical to before, so `MustVoteOnProposal`'s panic text does not move.

The three questions the change turns on all check out. gno's `errors.Is` does honour the extension point: [`wrap.gno:55`](https://github.com/gnolang/gno/blob/cf75d982a/gnovm/stdlibs/errors/wrap.gno#L55) · [↗](../../../../../.worktrees/gno-review-5998/gnovm/stdlibs/errors/wrap.gno#L55) type-asserts `interface{ Is(error) bool }` before unwrapping, exactly as Go does, and a side-by-side gno/Go run of the same program prints identical output. Sentinels as package-level realm state are stable: a pointer read out of the impl realm in one transaction still compares equal to a fresh read in the next. Returning a pointer with an exported field is not a write leak either: the readonly taint rejects `pce.Accepted = false` from the caller.

## Glossary

- realm: a stateful on-chain package under `r/` whose objects persist across transactions.
- ephemeral realm: the short-lived `gno.land/e/<addr>/run` code realm a `maketx run` script executes under.
- readonly taint: the VM guard rejecting a direct assignment into an object owned by another realm.
- txtar: testscript integration tests under `gno.land/pkg/integration/testdata/`.
- crossing / `cross`: a call into a crossing function, where the callee reads its caller through `cur.Previous()`.

## Fix

Five `errors.New` calls move from inside [`VoteOnProposal`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L107-L151) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L107-L151) up to a package-level [`var` block](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L19-L30) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L19-L30), so each return site hands back a value a caller can compare against instead of a fresh object. The closed-proposal branch cannot be a plain sentinel because its message embeds `status.Accepted`, so it becomes a [concrete type](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L37-L47) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L37-L47) whose `Is` maps both outcomes onto one sentinel. The load-bearing constraint the author held to is message stability: `MustVoteOnProposal` panics with `err.Error()`, and existing txtar tests assert on that text.

## Critical (must fix)

None.

## Warnings (should fix)

- **[caller checks stop matching after a govDAO upgrade]** `examples/gno.land/r/gov/dao/v3/impl/govdao.gno:19-30` — the sentinels are objects of the versioned implementation realm, so a caller's `errors.Is` returns false once `dao.UpdateImpl` swaps the implementation, while the message stays identical.
  <details><summary>details</summary>

  Callers reach voting through [`gno.land/r/gov/dao`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L101-L106) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/proxy.gno#L101-L106), which forwards to whichever implementation [`UpdateImpl`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L146-L160) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/proxy.gno#L146-L160) last installed. Each implementation declares its own `errors.New` objects, and identity is per-object, so `errors.Is(err, v3impl.ErrProposalNotFound)` flips to false after a swap even when the new implementation returns the same sentence. A caller that moved off string matching onto `errors.Is` therefore loses the branch silently: nothing panics, nothing logs, the `if` just stops firing. The govDAO is built to be swapped, with a proposal type ([`NewUpgradeDaoImplRequest`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L33-L51) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L33-L51)) and a rollback story written into the proxy's [`allowedDAOs`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L14-L19) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/proxy.gno#L14-L19) comment, so this is the mechanism the realm exists to support, not a hypothetical. Demonstrated on chain with [`dao5998_implswap.txtar`](tests/dao5998_implswap.txtar): before the swap `v3-sentinel-matches: true`, after it `msg: proposal not found` is unchanged and `v3-sentinel-matches: false` ([repro](comment_claude-opus-4-8.md)). Fix: declare the sentinels on `gno.land/r/gov/dao` next to the `VoteOnProposal` entry point, and have implementations return those values.
  </details>

## Nits

- **[realm callers cannot reach any sentinel]** `examples/gno.land/r/gov/dao/v3/impl/govdao.gno:108-110` — the only failure a code realm can trigger is the one branch with no sentinel, so nothing changes for realm callers.
  <details><summary>details</summary>

  [`isValidCall`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L214-L229) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L214-L229) admits only a direct user call or the ephemeral realm a `maketx run` script executes under; every other code realm is rejected at the top of `VoteOnProposal`, before any sentinel branch, with an ad-hoc `errors.New`. Confirmed on chain in [`dao5998_sentinels.txtar`](tests/dao5998_sentinels.txtar): a realm calling `dao.VoteOnProposal` gets `realm-err: proposal voting must be done directly by a user` and `realm-is-notfound: false`, while a run script in the same test matches the sentinel. The unit suite cannot show this, because under the gno test harness `cur.Previous()` reports an empty package path whatever `testing.SetRealm` was given, so `isValidCall` passes from any realm. Fix: give that rejection its own sentinel too, so the one error a realm caller can observe is also comparable.
  </details>

- **[bare issue reference does not resolve]** `examples/gno.land/r/gov/dao/v3/impl/govdao.gno:16` — `#3104` in a source comment is not a link and names no repository; [`govdao_test.gno:194`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L194) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L194) in the same package writes the full URL. The same bare form appears at [`govdao_test.gno:337`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L337) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L337). Cosmetic, no linter enforces it; not posted, no change needed.

## Missing Tests

- **[a broken Is() keeps the suite green]** `examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno:338-402` — every assertion is positive, so nothing pins that a sentinel matches only itself.
  <details><summary>details</summary>

  [`TestVoteOnProposalErrors`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L338-L402) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L338-L402) asserts each failure matches its own sentinel and stops there. The whole value of the change is discrimination, and discrimination is what goes untested: replacing the body of [`ProposalClosedError.Is`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L45-L47) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L45-L47) with `return true` leaves the test green, and so would collapsing all five sentinels onto one value. Verified both directions at cf75d982a: the mutated build passes `TestVoteOnProposalErrors` and fails [`TestSentinelsAreDisjoint`](tests/sentineldisjoint_test.gno) with three `expected: false / actual: true` mismatches. Fix: add the disjointness assertions, one table over all six sentinels plus the concrete closed error.
  </details>

## Suggestions

- **[the same closed-proposal state is still a string elsewhere]** `examples/gno.land/r/gov/dao/v3/impl/govdao.gno:158-160` — `PreExecuteProposal` tests the identical `status.Denied || status.Accepted` condition and returns an ad-hoc error, so "this proposal is closed" is programmatically detectable through voting and not through execution.
  <details><summary>details</summary>

  [`PreExecuteProposal`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L153-L173) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L153-L173) feeds [`executeProposal`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/proxy.gno#L180-L206) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/proxy.gno#L180-L206), which panics with `err.Error()`, so the message is all a caller has there. Its text differs ("proposal already executed. Accepted: %v"), so reusing `ProposalClosedError` would change it; a separate sentinel keeps the text and closes the gap. Deferred scope: the PR is scoped to `VoteOnProposal` and the issue asks only about voting.
  </details>

## Verified

- gno's `errors.Is` honours the `Is(error) bool` extension point the way `ProposalClosedError` assumes: [`wrap.gno:55`](https://github.com/gnolang/gno/blob/cf75d982a/gnovm/stdlibs/errors/wrap.gno#L55) · [↗](../../../../../.worktrees/gno-review-5998/gnovm/stdlibs/errors/wrap.gno#L55) tries the assertion before unwrapping, and a [gno filetest](tests/errors_is_custom_filetest.gno) mirroring the PR's shape prints output byte-identical to [the same program under Go 1.26.5](tests/errors_is_custom.go), across the match, the cross-sentinel miss, the reverse direction, the nil error, and `Unwrap`.
- Sentinels are safe as package-level realm state across both boundaries that matter. A foreign realm reads `impl.ErrProposalNotFound`, stores it, and in a later transaction the stored pointer still compares equal to a fresh read (`cached-eq: true cached-is: true`, [`dao5998_sentinels.txtar`](tests/dao5998_sentinels.txtar) step 4). `errorString` exposes no field and no mutating method, so the returned pointer carries no write path.
- Returning `*ProposalClosedError` with an exported `Accepted` field hands the caller no write: `pce.Accepted = false` from a run script aborts the transaction with `cannot directly modify readonly tainted object (use a method or crossing function)` ([`dao5998_closed.txtar`](tests/dao5998_closed.txtar)).
- Message text is preserved end to end on chain, not only in unit tests: voting on a closed proposal returns `msg: proposal closed. Accepted: true` through the proxy, matching the pre-PR `ufmt.Sprintf` output.
- Green at cf75d982a: `gno test ./gno.land/r/gov/dao/...` (4 packages), `go test -run 'TestTestdata/govdao' ./gno.land/pkg/integration/`, `go test -run 'TestFiles/govdao' ./gnovm/pkg/gnolang/`, and `gno lint ./gno.land/r/gov/dao/v3/impl` (sanity-checked by feeding it an undefined symbol, which it reports).

## Open questions

- Only the bot and labelling jobs ran on cf75d982a; `ci-dir-examples` never executed, so no upstream workflow has tested this diff. Standard for a first-time contributor's fork PR awaiting workflow approval, nothing for the author to do.
- Walked the invariant catalog: the only class the diff touches beyond error handling is the realm audit pattern `exported-pointer-leak`, which fires on `var ErrX = errors.New(...)` as an exported pointer into package state. Inspected and dropped: the readonly taint blocks foreign writes and `errorString` has no exported field, so there is no mutation path. Not posted, no change needed.
- The pre-existing [`ErrMemberNotFound`](https://github.com/gnolang/gno/blob/cf75d982a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L13) · [↗](../../../../../.worktrees/gno-review-5998/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L13) has the same placement problem as the Warning, so moving the sentinels to the proxy is a small migration rather than a one-line change. Worth deciding once, here, before this becomes the pattern other realms copy.
