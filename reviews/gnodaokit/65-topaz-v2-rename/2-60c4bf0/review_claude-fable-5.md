# PR [#65](https://github.com/samouraiworld/gnodaokit/pull/65): feat: republish fork packages at /v2 paths for Topaz (supersedes #64)

URL: https://github.com/samouraiworld/gnodaokit/pull/65
Author: zxxma | Base: main | Files: 45 | +193 -289
Reviewed by: davd-gzl | Model: claude-fable-5 | Commit: 60c4bf0 (latest)
Local checkout: `.worktrees/gnodaokit-review-65` (plain clone, PR 65 checked out)

Round 2. Head advanced 0612859 → 60c4bf0 (branch reworked to a single commit atop 15dbc83, 7 files, +10 -9): the `/v2` renames are dropped and the fork republishes at its original paths, leaving only the avl repoints and the svg-boundary alias. Round 1's headline Warning and its README nit are resolved. The port commit 15dbc83 is byte-identical, so its findings carry; every one was re-run or re-read against this head rather than ported on trust, which corrected two of them. The PR title still says `/v2`.

Corrections to round 1: the README member-gate Warning claimed the example no longer compiles, which was wrong. Its `gno.land/r/some/dao` import is a placeholder the reader supplies, not a shipped demo, so only the authentication half survives. The caller-identity Warning claimed a flat "resolves to the empty string"; measuring two further call shapes showed the resolved value tracks call shape, so the finding is reworded around that.

**TL;DR:** Repoints gnodaokit's tree storage to the vendored `p/samcrew/avl` fork so the two-value `Get` keeps working on topaz-1, keeping one import on the genesis avl where the svg package's type demands it. Sits on the earlier commit that ported the whole framework to interrealm v2.

**Verdict: REQUEST CHANGES** — the avl work is confirmed correct against the live chain and is ready; the blocking items are the two realm-threading defects in the interrealm-v2 port commit underneath, which no review on either PR has covered (4 Warnings, 2 Nits).

## Summary

The rework is what round 1 asked for. `/v2` is gone from every module line and import, verified by grep at the head, and the remaining diff is the 8 avl repoints plus the aliased `ntavl` import in [`gno/p/basedao/utils.gno:9`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/utils.gno#L9).

topaz-1 is now live and the premise is settled against the chain itself rather than the genesis script. `vm/qpaths gno.land/p/samcrew` on the launched chain returns exactly piechart, tablesort and urlfilter, so the original module paths are free and the rename was correctly dropped. Both remaining changes are required by the live genesis: `p/demo/svg` there types `Canvas.Style` as `*p/nt/avl/v0.Tree`, and genesis `p/nt/avl/v0.Get` is single-value while gnodaokit reads two values at 12+ call sites.

What no round has reviewed is the commit underneath. 15dbc83 rewrote caller identity and realm threading across the framework, and it carries the two defects below. [zxxma's gate review](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5004710182) scoped itself to the rename commit, and [#64](https://github.com/samouraiworld/gnodaokit/pull/64) has no reviews at all.

## Critical (must fix)
None.

## Warnings (should fix)
- **[caller picks the realm the DAO acts as]** [`gno/p/daokit/daokit.gno:17`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/daokit.gno#L17) — `Execute(id, rlm)` takes the realm from its caller, so a DAO action crosses under whichever realm invoked it, not the DAO's own.
  <details><summary>details</summary>

  Carried from round 1, code unchanged. `Core.Execute` forwards `rlm` to the action handler ([`daokit.gno:91`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/daokit.gno#L91)), and `NewEditProfileHandler` writes with `setter(cross(rlm), k, v)` ([`basedao/actions.gno:146`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/actions.gno#L146)). `Execute` is on the exported `daokit.DAO` interface, so any holder supplies its own `cur` and the write lands on that realm's profile. Ran it: a foreign realm's EditProfile execution recorded `gno.land/r/test/caller|Bio=written-by-caller`, while the DAO's own `basedao.New` init on the same DAO recorded `gno.land/r/test/daorlm`; see [tests](../tests/README.md). The same realm flows into `InstantExecute` ([`daokit.gno:34-37`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/daokit.gno#L34-L37)), which `NewInstantExecuteAction` aims at a [caller-supplied target DAO](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/actions.gno#L136) held in a field commented ["Target DAO to execute on"](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/actions.gno#L114), so a parent DAO executing on a sub-DAO makes the sub-DAO's profile writes land on the parent. The deleted [`crossing.gno`](https://github.com/samouraiworld/gnodaokit/blob/b833296/gno/p/daokit/crossing.gno#L36-L40) routed these through the DAO realm's own `crossFn`, so the realm was the DAO's rather than the caller's; nothing replaced it, and `DAOPrivate.Realm` is a plain `runtime.Realm` tuple, not a usable realm capability. Latent in the shipped demos only because they keep `localDAO` unexported. Fix: have the handler cross under the DAO's own realm instead of the one the caller passed.
  </details>
- **[a DAO cannot identify a cross-realm caller]** [`gno/p/basedao/basedao.gno:69`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/basedao.gno#L69) — `CallerIDFn` takes no realm, so a caller from another realm resolves to the empty string, and `MembersStore.AddMember` stores an empty address as a member without validating it.
  <details><summary>details</summary>

  Carried from round 1, code unchanged. `Execute` gained realm threading but `Propose` and `Vote` did not, and `CallerID` stays a bare `func() string` defaulting to `realmid.Previous` ([`basedao.gno:120`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/basedao.gno#L120)), which reads [`unsafe.PreviousRealm()`](https://github.com/gnolang/gno/blob/fc4052651/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L26). That call is documented not to name the immediate caller in a non-crossing helper, and the `daokit.DAO` methods are non-crossing. What it resolves to depends on the call shape rather than on who called: measured at this head, a direct cross-realm query returns the empty string, the same query under an origin-caller override panics `frame not found: cannot seek beyond origin caller override`, and with an intermediate realm frame it returns the frame below that one. None of these is the calling realm. `assertCallerIsMember` ([`basedao.gno:244`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/basedao.gno#L244)) then tests `IsMember("")`, and [`AddMember`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/members.gno#L184) validates neither emptiness nor address shape, so one empty-address member authenticates every cross-realm caller. Ran that: a foreign realm proposed, voted and executed on the DAO; without the empty member the gate panics `caller is not a member`. See [tests](../tests/README.md). Fix: reject empty and malformed addresses in `AddMember`, and give `CallerIDFn` a realm to read.
  </details>
- **[the taught membership gate does not name its caller]** [`gno/p/basedao/README.md:410`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/README.md?plain=1#L410) — the member-only example authenticates with `unsafe.PreviousRealm()` inside a non-crossing `Post`.
  <details><summary>details</summary>

  The port rewrote this line from the v1 `std.PrevRealm().Addr()` to `unsafe.PreviousRealm().Address()` and left `Post` non-crossing, which is the shape the stdlib [warns about](https://github.com/gnolang/gno/blob/fc4052651/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L20-L25). It is the framework's only worked answer to "how do I gate on membership", so the pattern propagates. Scope check before shipping this: the snippet imports the placeholder `gno.land/r/some/dao`, so `dao.DAO` is a realm the reader writes, not a shipped demo, and round 1's claim that the example no longer compiles was wrong and is dropped. The compile concern is real only for readers who followed the Quick Start, which is the separate guidance split covered in the gno-pin Warning below. Fix: make the example a crossing function that reads its own `cur`.
  </details>
- **[supersession silently drops #64's tail]** [`Makefile:1`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/Makefile#L1) — this branch still carries only [#64](https://github.com/samouraiworld/gnodaokit/pull/64)'s first commit; closing #64 on merge loses its pin bump and its docs fix.
  <details><summary>details</summary>

  Carried from round 1, unchanged at the new head. The net delta of #64's head [523bf58](https://github.com/samouraiworld/gnodaokit/commit/523bf58) over 15dbc83 is the Makefile `GNOVERSION` bump 2c7f1abe → ba9da8eb and the `localDAO` README alignment. The docs half was already flagged in [zxxma's gate comment](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5004710182); the pin half was not. Neither pin is the launch tip: the deployer has since moved its own CI to [fc4052651](https://github.com/samouraiworld/samcrew-deployer/blob/9b2b22e/CHANGELOG.md?plain=1#L12).

  The docs half is load-bearing, not polish. [`README.md:245`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/README.md?plain=1#L245) still tells integrators to export `DAO`, while this commit unexported it to `localDAO` in all three demos ([`simple_dao.gno:15`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/r/daodemo/simple_dao/simple_dao.gno#L15)). That README line is not itself in the diff, so the contradiction comes from the code side moving. Exporting `DAO` is exactly what puts `Execute` in reach of any realm, so the dropped commit is the mitigation for the first Warning. Fix: cherry-pick 523bf58's README changes and decide which gno pin the port ships with.
  </details>

## Nits
- **[lambda actions cannot cross]** [`gno/p/daokit/actions.gno:93-94`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/actions.gno#L93-L94) — `NewExecuteLambdaHandler` discards its realm into `_` and invokes a `func()` payload, so a lambda action can never do what the threading was added for, while [`NewInstantExecuteHandler`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/daokit/actions.gno#L126) below it forwards the realm. Nothing in the type or the doc comment says lambdas are the non-crossing kind.
- **[New ignores the realm it was given]** [`gno/p/basedao/basedao.gno:128`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/basedao.gno#L128) — `New` stores `Realm: unsafe.CurrentRealm()` although it holds an authoritative `rlm realm` parameter that cannot lie. The two agree in the demos, where `New` is only called from `init(cur realm)`, so this is latent. `DAOPrivate.Realm` feeds the private-extension gate, so a borrowed call site would diverge.
- **[title still advertises the dropped rename]** the PR title and description headline still say "republish fork packages at /v2 paths", which the head no longer does. Cosmetic, and the [update comment](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5011167644) explains the change; not posted.

## Missing Tests
- **[nothing exercises a second realm]** [`gno/p/basedao/basedao_test.gno:1`](https://github.com/samouraiworld/gnodaokit/blob/60c4bf0/gno/p/basedao/basedao_test.gno#L1) — every test drives the DAO from inside its own realm, so neither realm-threading Warning is caught by the suite.
  <details><summary>details</summary>

  The port changed who supplies the realm and what caller identity resolves to, and both defects only appear when a second realm is involved. The packages pass at the launch tip with both bugs live. The adversarial realms in [tests/](../tests/README.md) are the shape to add: one realm owning the DAO, one calling in, asserting that a foreign caller is rejected and that action writes are attributed to the DAO realm. Adding them as package tests needs the multi-realm harness the repo does not have yet, so they ship here as runnable realm files rather than a drop-in patch.
  </details>

## Suggestions
None.

## Verified

- Queried the launched chain directly, not the genesis script: `https://rpc.topaz.testnets.gno.land` reports `chain_id: topaz-1`, and `vm/qpaths gno.land/p/samcrew` returns exactly `piechart`, `tablesort`, `urlfilter`. No `daokit`, `basedao`, `daocond` or `realmid`. This confirms the author's live check and round 1's offline reproduction, and settles the dropped rename.
- Both surviving changes are required by the live genesis, checked with `vm/qfile` against the chain: `p/demo/svg` imports `gno.land/p/nt/avl/v0` and declares `Style *avl.Tree`, and `p/nt/avl/v0` exposes single-value `Get`. The vendored [`p/samcrew/avl` `Get` is two-value](https://github.com/samouraiworld/samcrew-deployer/blob/9b2b22e/deps/avl/tree.gno#L51), which is what the 12+ two-value call sites need.
- `/v2` is fully gone: no `samcrew/{daokit,basedao,daocond}/v2` occurrence remains at 60c4bf0, and the three gnomod module lines are back to their original paths.
- The port commit is untouched by the rework: diffing 15dbc83 against the new head over the five finding files shows only `members.gno`, and only its avl import line. Every cited line still reads what round 1 cited it for.
- Re-ran the adversarial realms against 60c4bf0 itself rather than trusting the round-1 run ([tests/](../tests/README.md)): a DAO queried from a second realm reports its caller identity as the empty string, and an EditProfile action executed by that realm records the write under `gno.land/r/test/caller` while the DAO's own init records `gno.land/r/test/daorlm`. Removing the empty-address member turns the same run into `panic: caller is not a member`.
- Probed two further call shapes before wording the caller-identity Warning, rather than generalising from the first: under `testing.SetOriginCaller` the same query panics `frame not found: cannot seek beyond origin caller override`, and with an intermediate realm pushed via `testing.SetRealm` it returns the frame below that realm. The identity tracks call shape, not the caller. A stronger claim, that a calling realm can act as the signing EOA, was not reproduced in any shape and is not written up.
- Baselined the `unsafe` swap rather than attributing it: `runtime.PreviousRealm()` at the pre-port pin [4e80c37e](https://github.com/gnolang/gno/blob/4e80c37e8d1870aa5d3b01966ed4c5dfa18a7566/gnovm/stdlibs/chain/runtime/native.gno#L22-L25) is the same `getRealm(1)` stack-walk as today's [`unsafe.PreviousRealm()`](https://github.com/gnolang/gno/blob/fc4052651/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L26-L29), and `chain/runtime` no longer exports it at the launch tip. The rename was forced and changed no behavior on its own, so no finding is written against it.

## Existing threads

- [zxxma update comment](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5011167644): strip executed, live genesis verified, branch reworked to one commit. Resolves round 1's headline Warning; independently confirmed against the chain above.
- [zxxma gate comment](https://github.com/samouraiworld/gnodaokit/pull/65#issuecomment-5004710182): CI retool decision and the partial #64 supersession. The supersession half overlaps the Makefile Warning; the CI half is unresolved and unchanged.

## Open questions

- CI is still red on the pre-existing 2c7f1abe pin, and the retool is deferred to a follow-up. Worth deciding alongside the pin question in the Makefile Warning, since both are answered by moving to the launch tip.
- Namespace registration for `samcrew` on topaz-1: `names.Enable` is on from genesis, so the deploy ceremony depends on owning the namespace. The deployer gates on this already; deployment ops, not this diff.
