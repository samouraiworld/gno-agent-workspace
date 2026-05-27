# PR #5051: feat(govdao): upgrade UI/UX

URL: https://github.com/gnolang/gno/pull/5051
Author: davd-gzl | Base: master | Files: 22 | +310 -160
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5051 5dec5ee43` (then `gh -R gnolang/gno pr checkout 5051` inside it)

**Verdict: REQUEST CHANGES** — purely cosmetic PR but ships two real regressions: the new "Votes" column shows tier-bucket count (always 3) instead of actual votes, and `md.EscapeText` was dropped from title rendering. Executor metadata for treasury/token-update proposals is also silently degraded.

## Summary

Rewrites the GovDAO renderer (`r/gov/dao/v3/impl/render.gno` and `memberstore/`) from a flat bullet-list layout to a card/table layout: status badges, metadata tables, "Vote Summary" tables on `/votes` pages, navigation links between members/proposals. Pure UI change — no governance, voting, or execution logic touched. The risk surface is markdown rendering correctness and information lost from the old layout.

Two regressions slipped in. (1) The new table columns `| YES | %d | %.2f%% |` use `ps.YesVotes.Size()` for the count — but `YesVotes` is `MembersByTier`, an `*avl.Tree` keyed by tier name (T1/T2/T3) with member trees as values, so `Size()` returns the number of tier buckets (3 after `newEmptyVoteStore`), not the number of voters. Every rendered proposal — open or closed, with 0 or 100 votes — shows `YES | 3 | x%`. (2) `md.EscapeText(p.Title())` was removed from both `renderProposalListItem` and `renderProposalPage`. Currently no external caller has unrestricted title input (valoper monikers are regex-validated), but the escape was the only defense against future callers passing a title with `|`, `[`, or backtick into the new tables.

## Glossary

- `proposalStatus` — per-proposal vote state, has `YesVotes`/`NoVotes`/`AbstainVotes` of type `MembersByTier`.
- `MembersByTier` — `*avl.Tree` with tier-name keys (T1/T2/T3) and inner `*avl.Tree` of members. `Size()` = number of tiers, not number of members.
- `newEmptyVoteStore` — pre-populates a `MembersByTier` with empty T1/T2/T3 buckets, hence `.Size() == 3` from creation.
- `pss` — `ProposalsStatuses`, the per-DAO `*avl.Tree` mapping proposal-id to `*proposalStatus`.
- `tierColoredChip` — svg-coloured tier badge previously rendered in the members table.

## Fix

Before: GovDAO home was `# GovDAO` + bullet list of proposals (`### [Prop #N - title]`, `Author:`, `Status:`, `Tiers eligible to vote:`); per-proposal page was a `## Prop #N` header + free-form Description + `### Stats` bullets + `### Actions` line. Memberstore home rendered `- chip Tier T1 contains N members with power: P` rows; members list embedded `tierColoredChip(tn) tn` per row.

After: home becomes `# Governance DAO` + a `**N** proposals total | **N** active | **N** completed` summary line + card-style list items with inline vote counters. Per-proposal page gains a status-badge line, a `## Details` 2-col table (Author/Status/Eligible Tiers/YES%/NO%), a separate `## Description` section, an optional `## Execution Details` fenced block, the existing `## Voting Statistics` (now a vote table) and a `## Actions` section gated by status. Vote pages get their own header + `## Vote Summary` table + `## Individual Votes` body. Memberstore home becomes a `| Tier | Members | Power |` table; members list drops the colored chip and adds a `_Showing %d members_` line.

Executor metadata is unconditionally trimmed in `prop_requests.gno`: portfolio (AddMember), reason (Withdraw), reason+payment (TreasuryPayment), and token-key list (TreasuryGRC20TokensUpdate) are dropped from the executor description, leaving generic strings like `"A payment will be sent by the GovDAO treasury."` The same info still lives in the proposal Description, so it shows in the `## Description` block — but the `## Execution Details` block (the canonical "what will execute") is now generic.

## Critical (must fix)

- **[counter shows 3 for every proposal]** [`render.gno:122-123`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L122-L123) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L122-L123), [`render.gno:180-184`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L180-L184) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L180-L184), [`render.gno:221-229`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L221-L229) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L221-L229), [`types.gno:143-150`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/types.gno#L143-L150) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/types.gno#L143-L150) — `ps.YesVotes.Size()` is the tier-bucket count, not the voter count
  <details><summary>details</summary>

  **Shape:** the new tables claim a "Votes" column. The values come from `ps.YesVotes.Size()` / `ps.NoVotes.Size()` / `ps.AbstainVotes.Size()`.

  **Mechanism:** `YesVotes` is [`memberstore.MembersByTier`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L29-L36) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/memberstore/types.gno#L29-L36), which wraps an `*avl.Tree` keyed by tier names (T1/T2/T3) where each value is the inner `*avl.Tree` of actual voters. [`newEmptyVoteStore`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/types.gno#L66-L72) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/types.gno#L66-L72) seeds three tier buckets at creation, so `.Size() == 3` before any vote is cast, and stays `3` for every proposal regardless of how many members actually voted.

  **What you see:** the [filetest output](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_00_filetest.gno#L84-L86) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_00_filetest.gno#L84-L86) renders `| YES | 3 | 0.00% |` / `| NO | 3 | 0.00% |` / `| ABSTAIN | 3 | 0.00% |` even though the proposal in that test received zero votes. Same for [`govdao_execute_proposal_01_filetest.gno`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_01_filetest.gno#L123-L127) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_01_filetest.gno#L123-L127). The author committed these `3`s as expected output, so the green CI is hiding the bug.

  Fix: count actual voters by summing inner-tree sizes. Add a helper on `MembersByTier` (or inline it):
  ```gno
  func (mbt MembersByTier) TotalMembers() int {
      total := 0
      mbt.Iterate("", "", func(_ string, v interface{}) bool {
          if t, ok := v.(*avl.Tree); ok { total += t.Size() }
          return false
      })
      return total
  }
  ```
  Use it for the `Votes` column in `proposalStatus.String()`, `renderProposalListItem`, and `renderVotesForProposal`. Update the filetest goldens to reflect real counts.
  </details>

- **[`p.Title()` rendered unescaped into markdown tables and headings]** [`render.gno:104`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104), [`render.gno:173`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L173) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L173), [`render.gno:212`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L212) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L212) — escape removed silently
  <details><summary>details</summary>

  Master rendered titles via `md.EscapeText(p.Title())` in both the list-item and the proposal-page header. This PR drops the `md` import and inlines `p.Title()` raw into `### [Title](link)` and `# Proposal #N - Title`. The new layout adds risk on top of that: the proposal-page header is followed by a `| | |` markdown table — a `|` in a title would break the table parse on gnoweb; a stray `]` in a title would close the `[Title](url)` link of the list item early.

  Today's callers happen to be safe: `prop_requests.gno` hardcodes titles ("Change Law Proposal", "Member Withdrawal Proposal", …), and `gnops/valopers/proposal` builds titles from regex-validated monikers ([`valopers.gno:286-300`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno) only allows `[a-zA-Z0-9][\w -]{...}[a-zA-Z0-9]`). But `dao.NewProposalRequest` is public; any future caller passing a user-supplied title bypasses the renderer's only defense.

  Fix: re-add the `gno.land/p/moul/md` import and wrap every `p.Title()` site in `md.EscapeText(p.Title())`. Cheaper than fixing it after the next realm ships a free-form proposal title.
  </details>

## Warnings (should fix)

- **[executor metadata silently emptied for treasury/token-update proposals]** [`prop_requests.gno:163-167`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L163-L167) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L163-L167), [`prop_requests.gno:197-200`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L197-L200) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L197-L200) — `## Execution Details` block now generic
  <details><summary>details</summary>

  Master built executor descriptions with the actual payload: `Reason: ... / Payment: 42ugnot to g1...` for `NewTreasuryPaymentRequest`, and the new `bulletList` of token keys for `NewTreasuryGRC20TokensUpdate`. This PR replaces both with constant strings (`"A payment will be sent by the GovDAO treasury."`, `"The list of GRC20 tokens used by the treasury will be updated."`).

  The same information is duplicated into the proposal Description, so users still see it under `## Description`. But the `## Execution Details` fenced block ([`render.gno:133-144`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L133-L144) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L133-L144)) is supposed to surface "the exact payload that will run" — that signal is now lost. A voter who reads only the Execution Details (the most security-sensitive section) sees nothing actionable.

  Same shape for [`NewAddMemberRequest`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L86) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L86) (portfolio dropped) and [`NewWithdrawMemberRequest`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L113) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L113) (reason dropped). Fix: keep the structured payload in executor descriptions, or remove the `## Execution Details` block entirely and document that description is canonical — but don't silently de-fang it.
  </details>

- **[dead code: `tierColoredChip` has no callers]** [`memberstore.gno:176-181`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L176-L181) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L176-L181) — function defined but unused after PR
  <details><summary>details</summary>

  Pre-PR `rendermembers.gno` called `tierColoredChip(tnStr)` to prepend an svg chip to each tier cell. The new [`rendermembers.gno:53`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/memberstore/rendermembers.gno#L53) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/memberstore/rendermembers.gno#L53) is `tierCell := ufmt.Sprintf("%s", tn)` — chip gone, function orphaned. `grep -rn tierColoredChip examples/` finds only the definition.

  Also worth noting: dropping the chip from the members list is a real UX regression vs the PR's screenshots (which show colored tier badges). And `ufmt.Sprintf("%s", tn)` on an `interface{}` value containing a string is just an obfuscated `tn.(string)` — preserve the chip OR drop the Sprintf and use a direct type assert. Fix: re-introduce the chip in the table cell (it's purely SVG, no behavior change), or delete `tierColoredChip` and `tierColor` if the design intentionally dropped colored chips.
  </details>

- **[`getStatusBadge` and downstream cells deref `ps` without nil check]** [`render.gno:101`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L101) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L101), [`render.gno:114-118`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L114-L118) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L114-L118), [`render.gno:209`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L209) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L209) — panics on missing status
  <details><summary>details</summary>

  `renderProposalPage` does `ps := d.pss.GetStatus(...)` then immediately `getStatusBadge(ps)` (line 101) and `ps.DeniedReason` / `ps.TiersAllowedToVote` (lines 114, 118). [`pss.GetStatus`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/types.gno#L29-L46) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/types.gno#L29-L46) explicitly returns nil for missing IDs. Only `getPropStatus(ps)` (line 113) has a nil guard. `renderVotesForProposal` (line 199) does check for nil before using `ps` further, so the contract is inconsistent.

  Under current flow, `PostCreateProposal` always sets a status alongside the proposal, so this is latent. But the public `pss` indirection makes it cheap to break — anyone adding a proposal-creation path that skips `PostCreateProposal` (or hitting a future migration window where some proposals lack statuses) crashes the whole render. Fix: short-circuit at the top of `renderProposalPage` with the same `if ps == nil { return "..." }` pattern `renderVotesForProposal` already uses.
  </details>

- **[`countActiveProposals` iterates the full pss on every home render]** [`render.gno:265-275`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L265-L275) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L265-L275) — O(N) per page load
  <details><summary>details</summary>

  The home page now displays an "X active" counter, computed by `pss.Iterate("", "", ...)` over the entire proposals tree on every render. With a paginated `pssPager` (`pageSize=5`), the rest of the home is O(pageSize); this counter is O(N). Today's N is small, but the trajectory is "DAO grows linearly with time" — a counter the user can mostly read off the page (pagination footer + `totalProposals`) is not worth N tree walks.

  Fix: either drop the active counter (totalProposals - completed buckets via `seqid` end), or maintain `activeCount` incrementally on `PostCreateProposal` / `ExecuteProposal` state transitions. The latter is cleaner — one int field on `GovDAO`, two increments and one decrement.
  </details>

- **[`renderVotesForProposal` dead branch]** [`render.gno:234-239`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L234-L239) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L234-L239) — `votes == ""` is unreachable
  <details><summary>details</summary>

  `StringifyVotes` returns either `"No one voted yet."` or a non-empty buffer ([`types.gno:158-170`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/types.gno#L158-L170) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/types.gno#L158-L170)). The check `if votes == "" || votes == "No one voted yet."` has a dead first disjunct, and the string-comparison-as-sentinel pattern is brittle (any tweak to the message breaks the empty-state branding). Fix: return `("", true)` or expose a `HasVotes(ps) bool` helper, then condition the render on that.
  </details>

## Nits

- [`render.gno:74`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L74) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L74) — empty-state copy is now `_No proposals to display._` while everywhere else uses sentence-case unitalicised prose ("Voting in Progress", "Cast your vote:"). Pick one tone.
- [`render.gno:104`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104) — mixed format verbs in one Sprintf: `"# Proposal #%v - %s\n\n"`. `pid` is `int64`; use `%d` to match the surrounding code (e.g. line 174 already uses `%d`).
- [`render.gno:294-298`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L294-L298) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L294-L298) — `"> **Requirement:** You must have a registered namespace to cast a vote."` is a hard claim. Vote eligibility is by tier membership, not namespace registration. Re-check the wording — voters need a member entry, not a `r/sys/users` namespace.
- [`rendermembers.gno:53`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/memberstore/rendermembers.gno#L53) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/memberstore/rendermembers.gno#L53) — `ufmt.Sprintf("%s", tn)` on a single `interface{}` value is a no-op wrap; if you keep dropping the chip, write `tierCell, _ := tn.(string)`.
- [`memberstore.gno:185`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L185) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/memberstore/memberstore.gno#L185) — `md.H1("GovDAO Members")` then the home view writes `## Tier Summary` directly. On the `members` route the page ends up with `# GovDAO Members` then `## Members List` — readable. On the `home` route it's `# GovDAO Members` then `## Tier Summary` — also fine, but the page title says "Members" while the body leads with "Tier Summary". Consider renaming the H1 to "GovDAO Memberstore" or routing-aware.
- [`govdao_execute_proposal_01_filetest.gno:107-112`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_01_filetest.gno#L107-L112) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/filetests/govdao_execute_proposal_01_filetest.gno#L107-L112) — Details table shows YES/NO % rows on top of the Voting Statistics table further down. The same numbers appear twice on the page; pick one location.

## Missing Tests

- **[no test exercises the actual voter-count column]** [`govdao_test.gno`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L148-L149) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno#L148-L149) — assertions weakened, bug uncovered
  <details><summary>details</summary>

  Pre-PR `TestCreateProposalAndVote` checked the exact string `"NO PERCENT: 81.25%"`; post-PR the assertion is just `contains(dao.Render("0"), "81.25%")`. `81.25%` now appears in two places (Details table NO row, Voting Statistics NO row) and the bug-shaped value `3` in the Votes column is never asserted. Add a regression test that creates a proposal with K real voters and asserts the rendered Votes column shows K, not 3.
  </details>

- **[no test for unescaped title rendering]** [`govdao_test.gno`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/govdao_test.gno) — bypasses the now-removed `md.EscapeText`
  <details><summary>details</summary>

  Add a proposal with title containing `|`, `[`, `*`, and confirm the rendered list-item link and the Details table both still parse. If escaping is restored (Critical above) the test becomes the regression guard; if escaping is intentionally dropped, the test is documentation of the new contract.
  </details>

## Suggestions

- [`render.gno:104,212`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/render.gno#L104) — the heading `# Proposal #N - Title` is duplicated between `renderProposalPage` and `renderVotesForProposal`. Factor the header (title + nav links + badge) into a helper to keep them in sync; today they already disagree on the nav-link target ("View All Votes" vs "Back to Proposal").
- [`prop_requests.gno:113`](https://github.com/gnolang/gno/blob/5dec5ee43/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L113) · [↗](../../../../../.worktrees/gno-review-5051/examples/gno.land/r/gov/dao/v3/impl/prop_requests.gno#L113) — the withdrawal description bolds `**Reason:**` for non-T1 withdrawals where `reason` may be empty, producing `**Reason:** ` (trailing). Either elide the block when reason is empty, or require a reason everywhere.

## Questions for Author

- The PR description has six screenshots and no rationale — what's the motivation for dropping the executor metadata payload (treasury reason/payment, token keys, member portfolio)? If it's "duplicated with description", consider removing the `## Execution Details` block instead so the page has one source of truth.
- Was the colored tier chip in the members list dropped intentionally? The screenshots in the PR body include tier indicators on member rows; the code does not.
- Is "Eligible Tiers" in the Details table meant to communicate which tiers are *allowed* to vote, or which tiers *the proposal targets*? Today the column says T1, T2, T3 for everything because that's what `TiersAllowedToVote` carries, but readers may parse it as "this proposal targets all tiers" which isn't always true (e.g. T2-only filtered proposals).
