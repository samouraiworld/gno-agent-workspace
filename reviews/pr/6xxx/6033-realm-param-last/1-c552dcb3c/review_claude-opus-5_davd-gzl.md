# PR [#6033](https://github.com/gnolang/gno/pull/6033): refactor(examples)!: drop the placeholder argument by keeping the realm off first

URL: https://github.com/gnolang/gno/pull/6033
Author: davd-gzl | Base: master | Files: 145 | +799 -679
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: c552dcb3c (stale — +11 commits since, all applying this review; head cea49947b, ready for review, CI green)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6033 c552dcb3c`

**TL;DR:** A helper that wants a realm value but must not become a crossing function used to take a discarded `0` as its first argument. This moves the realm to the end of the parameter list instead, which does the same job with no placeholder, across 116 signatures and 422 call sites.

**Verdict: APPROVE** — the rewrite is order-preserving at every call site checked, and the remaining findings are all in the prose that describes it (4 Warnings, 3 Nits, all applied on the branch).

## Verify first

- [`gnovm/pkg/gnolang/types.go:1340-1347`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/pkg/gnolang/types.go#L1340-L1347) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/pkg/gnolang/types.go#L1340-L1347) — the whole change rests on `IsCrossing` reading `Params[0]` and nothing else. Confirm no rewritten signature left a realm at index 0: `grep -rnE '^func (\([^)]*\) )?[A-Za-z0-9_]+\((cur|rlm|r) realm\)' --include=*.gno examples gnovm/tests` returns nothing.
- [`examples/gno.land/p/demo/tokens/grc20/tellers.gno:137`](https://github.com/gnolang/gno/blob/c552dcb3c/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L137) · [↗](../../../../../.worktrees/gno-review-6033/examples/gno.land/p/demo/tokens/grc20/tellers.gno#L137) — `TransferFrom(owner, to address, amount int64, rlm realm)` has two adjacent `address` parameters, so a swap here compiles and moves someone else's money. Read the four rewritten call sites and confirm the owner still precedes the recipient.

## Summary

A crossing function is one whose first parameter is a `realm`, tested by [`FuncType.IsCrossing`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/pkg/gnolang/types.go#L1340-L1347) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/pkg/gnolang/types.go#L1340-L1347) reading `Params[0]`. A non-crossing helper that still needs a realm had to keep it off that slot, and the convention prepended a discarded `int`. Declaring the realm last keeps it off index 0 by the same rule, at no cost: parameter position is part of `FuncType.TypeID`, so the property travels with the type through interfaces and func values. 68 signatures keep the sentinel: their realm is the only parameter, the only other one is variadic, or the other one is a callback that belongs last. The figures are differences between the merge base and the head, and the ADR carries the commands that produce them.

Reading order: [`gnovm/adr/prxxxx_realm_param_last.md`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/adr/prxxxx_realm_param_last.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/adr/prxxxx_realm_param_last.md#L1) for the decision and the two shapes it cannot reach, then [`examples/gno.land/p/moul/authz/authz.gno:200-218`](https://github.com/gnolang/gno/blob/c552dcb3c/examples/gno.land/p/moul/authz/authz.gno#L200-L218) · [↗](../../../../../.worktrees/gno-review-6033/examples/gno.land/p/moul/authz/authz.gno#L200-L218) as the representative signature pair, then the call sites.

## Diagram

```
func F(_ int, rlm realm, x T)      func F(x T, rlm realm)
        │                                      │
        └─ Params[0] = int                     └─ Params[0] = T
           IsCrossing() == false                  IsCrossing() == false
           call: F(0, cur, x)                     call: F(x, cur)

shapes with nowhere to trail, sentinel kept (186 signatures):

func F(_ int, rlm realm)                     nothing to follow
func F(_ int, rlm realm, xs ...T)            variadic must stay last
func(_ int, rlm realm)                       callback type, realm is its only parameter
```

## Fix

Every `_ int, rlm realm` prefix becomes a trailing `rlm realm` wherever another parameter exists to carry the position, and every call drops its leading `0`. The load-bearing constraint is that the realm must not land at index 0, which is why the variadic and sole-parameter shapes are excluded rather than rewritten. Applied by a `go/ast` codemod that resolves each call per package and rewrites only when the second argument is a realm by syntax, so `ulist.Set(0, v)` is left alone.

## Warnings (should fix)

- **[record names a PR that does not exist]** `gnovm/adr/prxxxx_realm_param_last.md:1` — the ADR filename keeps the `prxxxx_` placeholder although the PR number is now known.
  <details><summary>details</summary>

  [`AGENTS.md:119-120`](https://github.com/gnolang/gno/blob/c552dcb3c/AGENTS.md?plain=1#L119-L120) · [↗](../../../../../.worktrees/gno-review-6033/AGENTS.md#L119-L120) sets the naming as `pr<number>_<description>.md` and allows `prxxxx_` only while the number is unknown. Every numbered sibling in [`gnovm/adr/`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/adr/pr5890_realm_sub.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/adr/pr5890_realm_sub.md#L1) uses the real number. Fix: rename to `pr6033_realm_param_last.md`.

  Applied on the branch.
  </details>

- **[figure counts only half the call sites it moved]** `gnovm/adr/prxxxx_realm_param_last.md:26` — the ADR says 419 call sites, counted before the five inside txtar-embedded realms were found.
  <details><summary>details</summary>

  The codemod walks `.gno` files. Five call sites live in realms embedded in `gno.land/pkg/integration/testdata/*.txtar` and were rewritten in a later commit, so the total is 424. The same figure appears in the PR body and in the first commit message. Fix: 424 in the ADR; the PR body needs the same edit.

  Applied on the branch for the ADR, and on the PR body.
  </details>

- **[claim promises a build failure that does not happen]** `gnovm/adr/prxxxx_realm_param_last.md:88-90` — Consequences says the compiler catches every stale call site, which is not true of the shapes that actually broke.
  <details><summary>details</summary>

  A realm embedded in a `.txtar` archive is compiled when the integration test deploys it, not when the tree builds. `main / lint` and `main / build` were both green while five such call sites were stale; the failure arrived from `gno.land/pkg/integration` as a panic carrying `cannot use 0 (untyped int constant) as string value in argument to a.DoByPrevious`. The panic aborts the test binary, so the first red run named one failure where there were five. Fix: say where the rejection lands, and record the blind spot in Verification.

  Applied on the branch.
  </details>

- **[warning tells the next reader to distrust a gate that works]** `gnovm/adr/prxxxx_realm_param_last.md:122-126` — Verification says `gno lint` exits 0 while printing type errors. It exits 1.
  <details><summary>details</summary>

  The claim matters because it tells whoever revisits this that `gno-checks / lint` cannot gate a signature desync, which is the exact failure class this change risks. Three deliberate breaks under the job's own command, `gno lint -C examples -v ./...`, all exit 1: a type error inside one package, a call site carrying the old arity across packages, and an implementation left behind by its interface.

  ```
  gno.land/r/demo/defi/foo20/foo20.gno:50:53: too many arguments in call to userTeller.TransferFrom
  	have (number, realm, address, address, int64)
  	want (grc20.address, grc20.address, int64, grc20.realm) (code=gnoTypeCheckError)
  exit status 1
  ```

  Fix: state what the gate does catch, and name the one shape it cannot, a call site inside a `.txtar` archive.

  Applied on the branch.
  </details>

## Nits

- **[guide teaches the shape the same guide replaces]** `gnovm/adr/migration_guide.md:323` and `:351` — two snippets in §11 still call `tdao.vote(0, cur, ...)`.
  <details><summary>details</summary>

  [`migration_guide.md:20-23`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/adr/migration_guide.md?plain=1#L20-L23) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/adr/migration_guide.md#L20-L23) teaches the trailing form in the same file. §11 is about `SetRealm` frames, so the argument order there is incidental, which is exactly why it was missed.

  Applied on the branch.
  </details>

- **[the sweep moved a signature it should have skipped]** `examples/gno.land/p/demo/tests/tests.gno:81` — `ExecRlm` moved its own realm last, which evicts the callback from the final argument.
  <details><summary>details</summary>

  The callback cannot drop the sentinel: a realm is its only parameter, so `func(realm)` puts a realm at `Params[0]` and every literal would type as crossing. So the move leaves one expression carrying both conventions, three lines from a sibling call that does not:

  ```go
  rtests.ExecSwitchRlm(cross(cur), func(_ int, rlm realm) {
  	// ...
  })                    // crossing, realm first, closure last

  rtests.ExecRlm(func(_ int, rlm realm) {
  	// ...
  }, cur)               // swept, closure evicted from last
  ```

  Fix: leave `ExecRlm` alone. A trailing realm is expressible here, so this is a judgement rather than an impossibility, and the ADR records it as a third shape that keeps the sentinel.

  Reverted on the branch, at the three declarations and the three call sites in [`gnovm/tests/files/zrealm_crossrealm11.gno:95`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/tests/files/zrealm_crossrealm11.gno#L95) · [↗](../../../../../.worktrees/gno-review-6033/gnovm/tests/files/zrealm_crossrealm11.gno#L95).
  </details>

- **[a README quoting code the sweep moved]** `examples/quarantined/gno.land/p/samcrew/daokit/README.md:297` — the page shows `DAO.Vote(0, cur, proposalID, vote)` against a `DAOWrapper` that now takes the realm last.
  <details><summary>details</summary>

  It quotes `daodemo.gno` verbatim, and that file moved. Same class as the txtar and docs misses: the codemod reads `.gno`. Found by sweeping every `.md` and `.txtar` for calls to the 92 function names that dropped the sentinel; it was the only real hit, the rest being name collisions on unrelated `Render` and `Get`, and frozen test5 genesis snapshots whose `0` is a proposal id.

  Applied on the branch.
  </details>

## Verified

- No rewritten signature left a realm at index 0. `grep -rnE '^func (\([^)]*\) )?[A-Za-z0-9_]+\((cur|rlm|r) realm\)' --include=*.gno examples gnovm/tests gnovm/stdlibs misc` returns nothing, so no helper silently became a crossing function.
- Every rewritten call site preserved the relative order of its non-realm arguments. A parser over the diff paired each removed call with its replacement, dropped the sentinel and the realm from both sides, and compared the remainder as ordered lists: 399 single-line call sites, zero mismatches. The five multi-line `NewSimpleExecutor` calls and the nine parser false positives were read by hand.
- The families where a swap would compile were read individually: `TransferFrom(owner, to address, ...)`, `getThread`/`getComment`/`setThreadReadonly` on adjacent `uint64` and `boards.ID`, `NewBasicNFT`/`NewNFTWithMetadata`/`NewNFTWithRoyalty` on adjacent `string`, and `CreateForm` on five consecutive `string` parameters. All preserve order.
- The apphash pin moved for the reason the branch claims. [`TestAppHashCrossrealm38`](https://github.com/gnolang/gno/blob/c552dcb3c/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L125) · [↗](../../../../../.worktrees/gno-review-6033/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L125) still produces the old hash at the merge base ddb752cac and the new one on the branch, so the shift comes from the branch and not from master.
- The sweep left no reachable signature behind. Every `_ int, rlm realm` declaration surviving outside `examples/quarantined/` falls into the two shapes the ADR excludes: 63 of `(_ int, rlm realm)` with nothing to trail, and one of `(_ int, rlm realm, addrs ...address)` where a realm cannot follow a variadic. No declaration keeps the sentinel while carrying another parameter that could hold the realm.
- `gno lint` gates on a signature desync. Three deliberate breaks under `gno lint -C examples -v ./...` each exit 1: an in-package type error, a cross-package call site with the old arity, and `*fnTeller` left behind by the `Teller` interface. This refutes the Verification claim in the ADR and is the fourth Warning above.
- Every doc and archive was swept, not just the ones the failure named. Every `.md` and `.txtar` in the tree was checked for calls to the 92 function names that dropped the sentinel, and for calls to the 58 that kept it written without one. One real hit, the daokit README above.
- Green at c552dcb3c: `gno.land/pkg/integration` (514s), `gno.land/pkg/sdk/vm` (64s), `TestFiles/zrealm_crossrealm11.gno`, and all 102 PR checks.

## Open questions

- `inviteMembers(boardID boards.ID, rlm realm, invites ...Invite)` puts the realm second-to-last, not last, because a variadic must stay final. The title said "last" and now says "keeping the realm off first", which is the rule [`IsCrossing`](https://github.com/gnolang/gno/blob/c552dcb3c/gnovm/pkg/gnolang/types.go#L1340-L1347) actually applies. The ADR heading still reads "last"; its Decision section carries the qualifier, so it was left.
