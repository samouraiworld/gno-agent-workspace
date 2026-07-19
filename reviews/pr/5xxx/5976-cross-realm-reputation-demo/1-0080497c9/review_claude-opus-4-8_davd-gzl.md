# PR [#5976](https://github.com/gnolang/gno/pull/5976): feat(examples): add cross-realm reputation demo realm

URL: https://github.com/gnolang/gno/pull/5976
Author: zardozmonopoly | Base: master | Files: 3 | +131 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 0080497c9 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5976 0080497c9`

**TL;DR:** A new example realm that keeps a points ledger for addresses. Other realms award points to a user; the ledger records who awarded them and refuses to take points from a person calling it directly.

**Verdict: REQUEST CHANGES** — the "only another realm can award points" guard is bypassed by `gnokey maketx run`, and a score can wrap negative (1 Critical, 1 Warning, 3 Nits, 2 Missing tests, 2 Suggestions).

## Summary

The realm stores two [`avl.Tree`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L10-L13) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L10-L13) ledgers: `scores` keyed by `address|category|issuer`, and `totals` keyed by address alone. [`AddPoints`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L15) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L15) is a crossing function, so its `cur` is minted by the VM and `cur.Previous()` is the unforgeable caller identity; the realm reads `PkgPath()` off it and tags every entry with it. That part is right.

The gate on top of it is not. [`caller.IsUserCall()`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L17) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L17) is true only when the caller's pkgPath is [empty](https://github.com/gnolang/gno/blob/0080497c9/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107), which a `gnokey maketx run` script is not: it executes in an ephemeral realm at `gno.land/e/<addr>/run`. One command credits any address any amount. Separately, the [`points <= 0`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) check bounds one award, not the running sum, so two `MaxInt64` awards leave the score at `-2`.

## Glossary

- crossing / `cross`: a call into a crossing function (`func F(cur realm, ...)`), invoked as `cross(cur)`; the callee identifies its caller through `cur.Previous()`.
- ephemeral realm: the short-lived code realm a `maketx run` script executes under (`gno.land/e/<addr>/run`); `IsUser()` accepts it alongside true EOA calls while `IsUserCall()` accepts only a direct EOA call.
- realm: a stateful on-chain package under `r/` whose objects persist across transactions; also the VM builtin type threaded as a `cur realm` parameter.

## Critical (must fix)

- **[anyone can award themselves points]** `examples/gno.land/r/demo/reputation/reputation.gno:16-19` — [`IsUserCall()`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L16-L19) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L16-L19) does not reject a `gnokey maketx run` script, so any account credits any address any amount. Fix: reject `caller.IsUser()`, which also covers the run path.
  <details><summary>details</summary>

  `IsUserCall()` returns true only when the caller's pkgPath is [the empty string](https://github.com/gnolang/gno/blob/0080497c9/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/stdlibs/chain/runtime/frame.gno#L105-L107). A `maketx run` script runs in an ephemeral realm whose pkgPath is `gno.land/e/<addr>/run`, so the guard passes and the script becomes the recorded issuer. `IsUser()` is [`IsUserCall() || IsUserRun()`](https://github.com/gnolang/gno/blob/0080497c9/gnovm/stdlibs/chain/runtime/frame.gno#L83-L85) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/stdlibs/chain/runtime/frame.gno#L83-L85) and is the predicate used by realms that want to catch both entries, e.g. [boards2](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L42-L44) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L42-L44). Confirmed against a running node: a `maketx run` script credited 1000000 points to an arbitrary address, [repro](comment_claude-opus-4-8.md); regression txtar at [`tests/reputation_msgrun_bypass.txtar`](tests/reputation_msgrun_bypass.txtar). Fix: reject `caller.IsUser()`.
  </details>

## Warnings (should fix)

- **[score wraps to a negative number]** `examples/gno.land/r/demo/reputation/reputation.gno:30` — the [per-award positivity check](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L20-L22) does not bound the stored sum, so two `MaxInt64` awards land the score at `-2`. Fix: abort when the addition would overflow.
  <details><summary>details</summary>

  Both [`scores.Set(key, current+points)`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L30) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L30) and [`totals.Set(totalKey, currentTotal+points)`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L37) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L37) add without a bound check. gno `int64` addition wraps silently, so a score stops being a positive running total with no panic and no way to unwind it. Ran two `MaxInt64` awards from one issuer realm: `GetScore` returned `-2`, [repro](comment_claude-opus-4-8.md). Combined with the Critical above, an EOA reaches this in one transaction. Test at [`tests/reputation_overflow_test.gno`](tests/reputation_overflow_test.gno), red at 0080497c9. Fix: abort when the addition would overflow.
  </details>

## Nits

- **[render breaks once the ledger is large]** `examples/gno.land/r/demo/reputation/reputation.gno:65` — [`totals.Iterate("", "", ...)`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L65-L68) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L65-L68) walks every address with no bound; the output grows about 46 bytes per address.
  <details><summary>details</summary>

  Measured the rendered string at three ledger sizes: 696 bytes at 10 addresses, 10,006 at 200, 98,206 at 2000. `Render` has no upper bound, and the ledger only grows, so `vm/qrender` eventually returns an unusable page. [`IterateByOffset`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/p/nt/avl/v0/tree.gno#L111) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/p/nt/avl/v0/tree.gno#L111) is the paging primitive, and the unused `path` argument is the natural place to take the page number. Fix: page the output.
  </details>

- **[stale spelling of the empty interface]** `examples/gno.land/r/demo/reputation/reputation.gno:65` — the callback writes [`interface{}`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L65) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L65) where [`IterCbFn`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/p/nt/avl/v0/tree.gno#L21) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/p/nt/avl/v0/tree.gno#L21) is declared with `any`.

- **[exported surface has no doc comments]** `examples/gno.land/r/demo/reputation/reputation.gno:15` — none of [`AddPoints`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L15) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L15), [`GetScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L40) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L40), [`GetTotalScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L48) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L48) or [`Render`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L60) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L60) carries one, and a demo realm is read as a pattern. The caller-only contract on `AddPoints` in particular deserves a sentence at the declaration, not only in the panic string.

- **[unused parameter reads as a bug]** `examples/gno.land/r/demo/reputation/reputation.gno:60` — [`Render(path string)`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L60) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L60) never reads `path`; [`r/demo/counter`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/counter/counter.gno#L12) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/counter/counter.gno#L12) writes `_ string` for the same shape. No enabled linter covers this and the meaning is unchanged, so not posted, no change needed; it is subsumed by the paging nit above if `path` becomes the page argument.

## Missing Tests

- **[the realm's core claim is unasserted]** `examples/gno.land/r/demo/reputation/reputation_test.gno:9` — [no test](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation_test.gno#L9-L31) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation_test.gno#L9-L31) has two different issuer realms write the same category for the same target.
  <details><summary>details</summary>

  The whole point of folding the issuer into the key is that one realm cannot overwrite or read another's award. Every existing test uses a single issuer, so a regression that dropped `issuer` from [`scoreKey`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) would keep the suite green. The ready-to-add case switches realms mid-test and asserts both per-issuer scores plus the combined total; it passes at 0080497c9, so it is a guard, not a bug proof. Cases are in [`comment_claude-opus-4-8.md`](comment_claude-opus-4-8.md).
  </details>

- **[the rejected entry path is untested]** `examples/gno.land/r/demo/reputation/reputation_test.gno:33` — [`TestAddPointsDirectUserCallPanics`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation_test.gno#L33-L42) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation_test.gno#L33-L42) covers direct MsgCall only, which is why the `maketx run` entry went unnoticed.
  <details><summary>details</summary>

  `testing.NewCodeRealm` [rejects](https://github.com/gnolang/gno/blob/0080497c9/gnovm/tests/stdlibs/testing/context_testing.gno#L145-L150) · [↗](../../../../../.worktrees/gno-review-5976/gnovm/tests/stdlibs/testing/context_testing.gno#L145-L150) an `/e/` path, so the run-script entry cannot be reached from a package unit test. The integration harness can: [`tests/reputation_msgrun_bypass.txtar`](tests/reputation_msgrun_bypass.txtar) boots a node and drives `gnokey maketx run` against the realm, asserting the current (accepted) result with the post-fix assertion commented alongside. Fix: land it under `gno.land/pkg/integration/testdata/` with the post-fix assertion active.
  </details>

## Suggestions

- **[read side can address another issuer's entry]** `examples/gno.land/r/demo/reputation/reputation.gno:56-58` — the author asked about the key shape: [concatenating with `|`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L56-L58) is safe on the write path but not on the read path.
  <details><summary>details</summary>

  Writes cannot be forged. The issuer segment is appended last and comes from `cur.Previous().PkgPath()`, which the caller does not control and which contains no `|`, so the final segment of a key is always the true issuer no matter what a caller puts in `category`. Reads are looser: [`GetScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L40-L46) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L40-L46) takes `issuerPkgPath` verbatim from its caller, so a category holding a `|` makes two distinct argument triples hit one entry. Confirmed behaviorally: an award written as category `a|b` from `gno.land/r/demo/issuerx` reads back as 42 both for `("a|b", "gno.land/r/demo/issuerx")` and for `("a", "b|gno.land/r/demo/issuerx")`. Nesting a per-issuer `avl.Tree` under each address removes the concatenation entirely and gives cheap per-issuer enumeration for `Render`. Fix: nest the trees, or length-prefix the segments.
  </details>

- **[page describes a trust property the ledger does not have]** `examples/gno.land/r/demo/reputation/reputation.gno:63` — the [rendered text](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L63) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L63) says points can only be awarded by other realms, which reads as a barrier; deploying a one-line realm is the whole barrier.
  <details><summary>details</summary>

  Even with the Critical fixed, anyone can `addpkg` a realm that calls `AddPoints` and mint without limit, and [`GetTotalScore`](https://github.com/gnolang/gno/blob/0080497c9/examples/gno.land/r/demo/reputation/reputation.gno#L48-L54) · [↗](../../../../../.worktrees/gno-review-5976/examples/gno.land/r/demo/reputation/reputation.gno#L48-L54) sums across every issuer, so the headline number on the page is fully attacker-controlled. The per-issuer score is the only number a consumer can reason about, and only if the consumer already trusts that issuer. Since the realm is a demo others will copy, saying that plainly on the page matters more than the guard does. `runtime.AssertOriginCall()` is [the primitive that forbids MsgRun entirely](https://github.com/gnolang/gno/blob/0080497c9/docs/resources/effective-gno.md?plain=1#L852-L856) · [↗](../../../../../.worktrees/gno-review-5976/docs/resources/effective-gno.md#L852-L856) if a harder gate is wanted, but it does not change this: the barrier is one `addpkg`. Fix: describe the guarantee as per-issuer attribution rather than as a restriction on who can award.
  </details>

## Verified

- The `maketx run` bypass reproduces against a live node, not just in unit tests: [`tests/reputation_msgrun_bypass.txtar`](tests/reputation_msgrun_bypass.txtar) boots `gnoland start`, broadcasts a run script, and the script's `GetTotalScore` reads back 1000000 credited to an address the script chose.
- Score overflow wraps rather than aborting: two `MaxInt64` awards from one issuer realm leave `GetScore` at `-2`; [`tests/reputation_overflow_test.gno`](tests/reputation_overflow_test.gno) asserts the post-fix abort and is red at 0080497c9.
- `Render` output measured at 696 / 10,006 / 98,206 bytes for 10 / 200 / 2000 ledger addresses, confirming the growth is per-address and unbounded.
- Key aliasing measured, not inferred: the same stored award reads back as 42 through both `GetScore(target, "a|b", "gno.land/r/demo/issuerx")` and `GetScore(target, "a", "b|gno.land/r/demo/issuerx")`.
- The realm's own suite passes at 0080497c9 (`gno test ./gno.land/r/demo/reputation`, 5 tests), and `gno lint` is clean on the package; lint was sanity-checked by injecting an undefined symbol, which it reported.

## Open questions

- `AddPoints` has no way to revoke or decay a score, so a buggy issuer realm permanently poisons a target's ledger. Deliberate for a demo; not posted because nothing in this PR forces the decision.
- The realm ships without a `README.md`, matching the other `r/demo` realms, none of which have one. Not posted, no change needed.
