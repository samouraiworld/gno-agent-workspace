# PR #4731: feat(GovDAO): add activity page to highlight inactive GovDAO's members

URL: https://github.com/gnolang/gno/pull/4731
Author: davd-gzl | Base: master | Files: 9 | +182 -10
Reviewed by: davd-gzl | Model: claude-opus-4-7[1m]

Verdict: REQUEST CHANGES — heavy master drift forces a full rewrite (interrealm signatures, `cross(rlm)`, `bptree`); loader re-adds members already removed by merged PR #4716; `Available` counter increments for tier-ineligible members on filtered proposals; orphan activity rows on member removal; render rebuilds the pager tree every request, defeating the pager.

## Summary

Adds a `:activity` sub-page to `/r/gov/dao/v3/impl` showing each member's vote/available ratio across proposals, plus a member-list bootstrap in `loader.gno` (Jae, Manfred, Milos, ..., 25 entries). Activity is tracked with a global `avl.Tree[address -> {Available, Votes}]`: `Votes` increments in `VoteOnProposal`, `Available` increments for every member of every tier in `PostCreateProposal`. The render path collects all members into a fresh AVL tree per request, then paginates 15 per page.

```
PostCreateProposal(pid)
  -> incrementAvailableVotesForAllMembers(cross)
      iterate tiers -> iterate members -> Available++   (regardless of tatv)

VoteOnProposal(r)
  -> incrementMemberActivity(caller)                     (Votes++)

renderActivityPage(path)
  -> iterate tiers -> iterate members -> collect
  -> rebuild avl.Tree (Set N times)
  -> pager.MustGetPageByPath -> 15 entries
```

## Glossary

- `tatv` — TiersAllowedToVote, per-proposal whitelist set by `FilterByTier`.
- `memberActivityTree` — global activity store, `address -> MemberActivity`.
- `Available` — counter intended to equal "proposals this member could vote on".
- `Votes` — counter intended to equal "proposals this member actually voted on".
- `FilterByTier` — proposal filter that restricts voting to a subset of tiers.

## Fix

Before: no on-chain participation signal; reviewers must look at each proposal individually.
After: every proposal creation bumps a per-member `Available` counter for everyone in every tier; every vote bumps that member's `Votes`. A new `:activity` sub-page paginates members and renders `votes/available (X%)`.
Constraint: must work without an off-chain indexer, must avoid the O(members·proposals) full scan @moul flagged in [the earlier round](https://github.com/gnolang/gno/pull/4731#discussion_r2467540400). The PR replaces the per-proposal nested scan with per-vote increments, then performs an O(N) iteration over tiers+members on every render — the read-side hot loop is the same shape @moul rejected, just deferred to the render call.

## Critical (must fix)

- **[loader collision with merged PR #4716]** [`examples/gno.land/r/gov/dao/v3/loader/loader.gno:32-60`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/loader/loader.gno#L32-L60) — re-adds 25 hard-coded T1/T2/T3 members that PR #4716 deliberately removed.
  <details><summary>details</summary>

  PR [#4716](https://github.com/gnolang/gno/pull/4716) (merged 2025-09-03, `8441a1e81`) was titled `chore(govdao): relegate and remove members from genesis loader` and stripped exactly this initialization. The loader doc comment in this PR's own diff says "It intentionally does NOT add any members" — the diff then adds 25 of them, contradicting the surviving header. Bootstrap is now done via MsgRun (`govdao_prop1.gno` in `misc/deployments/`), not loader. Fix: drop all `SetMember` lines from `loader.gno`; if activity tracking needs members at genesis, move that concern into the bootstrap script, not the loader.
  </details>

- **[master drift — won't compile after rebase]** [`examples/gno.land/r/gov/dao/v3/impl/govdao.gno:92`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L92) — bare `cross`, old method signatures, `avl` pager — all changed on master.
  <details><summary>details</summary>

  Master refactored govdao interrealm-Phase-3 in [PR #5669](https://github.com/gnolang/gno/pull/5669): `PostCreateProposal(_ int, rlm realm, r, pid)`, `VoteOnProposal(_ int, rlm realm, r)`, etc. — every method on `*GovDAO` now takes the realm explicitly. Bare `cross` no longer compiles; usage is `cross(rlm)`. Pager moved from `avl/v0/pager` to `bptree/v0/pager`. The PR currently sits on a master snapshot from Oct 2025; rebasing against current master means rewriting every diff hunk in `govdao.gno`, `render.gno`, `activity.gno`. Fix: rebase, propagate `rlm` into `incrementAvailableVotesForAllMembers`/`incrementMemberActivity`, swap to `bptree` pager, drop bare `cross`.
  </details>

- **[Available counter ignores tatv]** [`examples/gno.land/r/gov/dao/v3/impl/activity.gno:34-54`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/activity.gno#L34-L54) — T1-only and T1+T2 proposals bump `Available` for ineligible tiers.
  <details><summary>details</summary>

  `PostCreateProposal` ([`govdao.gno:78-89`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L78-L89)) computes `tatv` (TiersAllowedToVote) — `{T1}` for `FilterByTier{T1}`, `{T1,T2}` for `FilterByTier{T2}`. It then calls `incrementAvailableVotesForAllMembers(cross)` which iterates every tier unconditionally and increments `Available` for everyone. Concrete consequence: every T1-member-add proposal bumps T2/T3 `Available` counters even though those members cannot vote on it. Their displayed `votes/available` ratio drops from 100% to 50% the moment a T1 proposal lands, despite them having no opportunity to vote. The whole point of the page — "highlight inactive members" — is contaminated by ineligible-vote noise. Fix: pass `tatv` into `incrementAvailableVotesForAllMembers`, iterate only those tiers.
  </details>

- **[member removal leaks activity rows]** [`examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno:108-111`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L108-L111) and [`prop_requests.gno:124`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L124) — `RemoveMember` doesn't touch `memberActivityTree`.
  <details><summary>details</summary>

  Two code paths remove members: `NewWithdrawMemberRequest` cb (line 109) and `NewPromoteMemberRequest` cb (line 124). Neither calls into `activity.gno`. Result: the activity tree accumulates rows for every member ever removed, with no way to GC them. On a long-running chain this is unbounded state growth and the render path keeps paying the O(N_ever_member) cost on every page load. Worse, the address may later be re-added (promote/withdraw cycle) and inherit stale counters — `Votes/Available` from a previous tenure carries over silently. Fix: add `removeMemberActivity(addr)` and call it from both removal paths; on promotion you may want to reset rather than carry over.
  </details>

## Warnings (should fix)

- **[render rebuilds the pager every request]** [`examples/gno.land/r/gov/dao/v3/impl/render.gno:138-205`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/render.gno#L138-L205) — full-tier iteration + N inserts into a fresh AVL tree per render call, regardless of page.
  <details><summary>details</summary>

  The PR description claims the goal was to remove the nested-iteration anti-pattern @moul flagged. The render path still does it: it iterates every tier, every member, builds `activities []memberActivity`, then iterates `activities` again to populate a brand-new AVL tree, then asks the pager for 15 items. The pager only saves output bytes, not work. For a 30-member GovDAO this is cheap, but the PR also includes loader entries scaling toward 100 — and the explicit design rationale in the author's [self-comment](https://github.com/gnolang/gno/pull/4731#discussion_r2511289253) said the implementation "does not highlight inactive members instantly. It requires an AVL tree for storing activities data" while choosing this approach precisely to avoid scanning. The current code reintroduces the scan on the read side. Fix: keep `memberActivityTree` as the pager source directly (its key is address, which is fine for stable pagination), drop the rebuild dance. Sort/filter at write time, not at every render.
  </details>

- **[no ADR for a state-bearing feature]** repo-wide — `AGENTS.md` requires an ADR for non-trivial AI-assisted PRs; this one adds persistent on-chain state (`memberActivityTree`) plus a new render endpoint, with no `gno.land/adr/pr4731_*.md` in the diff.
  <details><summary>details</summary>

  See [`AGENTS.md:83-101`](../../../../../.worktrees/gno-review-4731/AGENTS.md#L83-L101). Even if the implementation is small, the schema choice (separate global tree vs. embedded in `proposalStatus`, address-keyed vs. tier-keyed, increment-on-create vs. compute-on-read) is exactly the kind of decision an ADR records for future contributors. Fix: add `gno.land/adr/pr4731_govdao_activity.md` covering context (issue #4715), decision (incremental counters, separate global AVL), alternatives (per-proposal participation set, off-chain indexer, sorted-by-activity view), consequences (carry-over on promotion, ineligible-tier bump).
  </details>

- **[no tests for activity tracking]** [`examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno) and the filetests under `filetests/` — none of them assert on `memberActivityTree` state.
  <details><summary>details</summary>

  `TestCreateProposalAndVote` exercises the code indirectly (the new increments run because they're called from `PostCreateProposal`/`VoteOnProposal`), but nothing checks that `Votes` actually went up, that `Available` is per-tier-correct, or that withdraw/promote interacts predictably. The Critical findings above (tatv-ignoring, removal leak) would not be caught by the existing suite. Fix: add a filetest that creates a T1-filtered proposal and asserts T2/T3 `Available` is still 0; add a filetest that withdraws a member and asserts their row is gone (or document the carry-over decision in the ADR and assert that instead).
  </details>

- **[`ufmt.Sprintf("%05d", index)` breaks lexical order past 99999]** [`examples/gno.land/r/gov/dao/v3/impl/render.gno:175`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/render.gno#L175) — string-sorted AVL keys, fixed-width pad.
  <details><summary>details</summary>

  At 100000 members the key becomes 6 digits and `"100000"` sorts before `"99999"` lexically. Practically irrelevant for a GovDAO. Tag is parked here only because the rebuild dance is being defended on grounds that are themselves fragile — if you're going to build a fresh AVL on every render, at least key it by something that sorts naturally (address itself, or an integer-sortable encoding). Fix: drop the rebuild as per the Warning above; if kept, use a wider pad or skip pagination for the count expected.
  </details>

- **[`endIdx` underflow on empty page]** [`examples/gno.land/r/gov/dao/v3/impl/render.gno:199-202`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/render.gno#L199-L202) — `startIdx + len(page.Items) - 1` is `startIdx - 1` when items is empty.
  <details><summary>details</summary>

  Early-return at line 143 covers `tiers.Size() == 0` (no tier entries) but not the "tier exists, no members" case nor the "page beyond end" case. If `MustGetPageByPath` returns an empty page (e.g. someone manually requests `?page=99`), the footer renders `_Showing N-(N-1) of M members_`. Fix: guard `if len(page.Items) == 0`, render a "no members on this page" footer.
  </details>

## Nits

- [`examples/gno.land/r/gov/dao/v3/impl/activity.gno:34`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/activity.gno#L34) — `cur realm` named parameter is unused; convention in this file is `_ realm`. The crossing-function intent is the only reason it's named.

- [`examples/gno.land/r/gov/dao/v3/impl/activity.gno:8-11`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/activity.gno#L8-L11) — `MemberActivity` exported; `incrementMemberActivity`/`incrementAvailableVotesForAllMembers` not; `GetMemberActivity` exported but no external caller in-tree. Pick one: package-private or document the external contract.

- [`examples/gno.land/r/gov/dao/v3/impl/render.gno:67`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/render.gno#L67) — link label `[> View Activity <]` is decorative; consider `[View Activity →](...)` for consistency with the surrounding `[> Go to Memberstore <]` (or normalize both later).

- [`examples/gno.land/r/gov/dao/v3/memberstore/types.gno:21-25`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L21-L25) — `NewMember` constructor is fine, but the only motivation is `loader.gno` brevity; on master the loader doesn't seed members, so the constructor's reason to exist is the contested loader entries. Tied to the Critical above.

## Missing Tests

- **[no FilterByTier×Available assertion]** [`examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno) — would catch the Critical "Available ignores tatv".
  <details><summary>details</summary>

  Create a T1-filtered AddMember proposal, assert `GetMemberActivity(m4).Available == 0` (m4 is T3 in the existing fixture). Currently it'll be 1, which proves the bug.
  </details>

- **[no withdraw-then-check assertion]** — would catch the Critical "removal leaks activity rows".
  <details><summary>details</summary>

  Add a member, vote, withdraw via `NewWithdrawMemberRequest`, then either (a) assert `GetMemberActivity(addr) == zero` if the decision is "purge on remove" or (b) assert the carry-over is documented and bounded.
  </details>

## Suggestions

- [`examples/gno.land/r/gov/dao/v3/impl/activity.gno`](../../../../../.worktrees/gno-review-4731/examples/gno.land/r/gov/dao/v3/impl/activity.gno) — consider embedding activity in `proposalStatus` instead of a parallel global tree.
  <details><summary>details</summary>

  Storing the eligible-voter set per proposal (already in `proposalStatus.TiersAllowedToVote`) lets the render path compute participation by iterating `pss.Tree` once with a cap (e.g. last 50 proposals), avoiding both the per-create write-amplification (`incrementAvailableVotesForAllMembers` writes once per member per proposal) and the global tree leak on removal. Trade-off: read cost grows with proposal count, not member count. Worth an ADR row.
  </details>

## Questions for Author

- After PR #4716 deliberately moved member init out of `loader.gno`, why re-introduce it here? If the activity page needs a non-empty memberstore to be useful in tests, the bootstrap MsgRun (`misc/deployments/.../govdao_prop1.gno`) is the right place.

- Promote/demote: should activity carry over when a member's tier changes, or reset? Current code carries over silently. The ADR should pin this.

- `MemberActivity` is exported and `GetMemberActivity` is exported — is this intended as a public API for other realms (e.g. a future indexer/notifier realm)? If so, document the stability contract; if not, lowercase them.
