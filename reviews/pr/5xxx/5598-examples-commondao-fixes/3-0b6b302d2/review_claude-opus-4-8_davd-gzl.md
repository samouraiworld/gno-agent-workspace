# PR #5598: fix(examples): fixes issues in `commondao` package

URL: https://github.com/gnolang/gno/pull/5598
Author: jeronimoalbi | Base: master | Files: 63 | +2440 -1458
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `0b6b302d2` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5598 0b6b302d2`

**TL;DR:** The PR fixes bugs and tightens the API of `commondao`, a building-block package for on-chain DAOs that handles members, proposals, voting, and execution. It rejects-then-archives losing proposals, recovers from panics in custom vote-counting, and reworks the read-only/immutable types.

**Verdict: REQUEST CHANGES** — the realm now compiles and CI is green (the two round-2 build-break Criticals are fixed), but the round-1/round-2 mixed-`memberCount` quorum-vs-majority bug is still live in all three canonical proposal definitions, and the new render path calls `dao.Tally` which flips an active proposal's displayed status to "passed"/"rejected" mid-voting.

## What changed since round 2 (`002214760` → `0b6b302d2`)

The PR head moved and PR content changed; patch-ids differ. The base also advanced (a `Merge branch 'master'` commit pulls hundreds of unrelated files: markdown-sanitize, crypto stdlibs, tm2 privval, launder filetests). Those are not PR content and are excluded. The PR's own delta since `002214760` is 11 files, all under the realm `r/nt/commondao/v0`; the package `p/nt/commondao/v0` is byte-identical to round 2.

Delta contents:
- `execute(realm) error` → `execute(string) error` in `proposal_members.gno`, `proposal_subdao.gno` (×2), the `z_6_a` filetest, and the two `*_test.gno` sites (`fn(cross(cur))` → `fn(cur.PkgPath())`). Fixes round-2 Critical "realm executor still has the old signature".
- `p.Definition().Title()/Body()` → `p.Title()/p.Body()` in `render.gno` and five `zp_*` filetests. Fixes round-2 Critical "`p.Definition()` does not exist".
- `render.gno` proposal-detail view now calls `dao.Tally(p.ID(), true)` instead of the manual `MustNewVotingContext` + `def.Tally(ctx)` dance, and prints the tally error. This is the render rewrite the round-2 Critical-fix sentence and the "Tally after deadline can re-flip status" Warning asked to be done carefully; it introduces the new display bug below.

Resolved since round 2: both build-break Criticals (realm compiles, `gno test ./gno.land/r/nt/commondao/v0` passes, CI `build` / `check` / `gno-checks` green). Carried forward: the mixed-`memberCount` Critical and all package-side Warnings/Nits/Missing-Tests/Suggestions, since `p/nt/commondao/v0` is unchanged this round. New: the render status-display bug.

## Summary

`commondao` is a `/p/` library that realms compose into governance: members live in a `MemberStorage` (with optional second-tier groupings), proposals carry a user-supplied `Definition` whose `Tally(ctx)` counts votes, and `Execute` runs the winning proposal. Round 3 migrates the in-tree realm `r/nt/commondao/v0` to the round-2 package API so the tree builds again. Two correctness problems survive into this round: the canonical proposal definitions feed two different member counts into one tally (all members for the quorum denominator, ungrouped-only for the super-majority threshold), so a grouped-member DAO can pass quorum yet always fail the majority; and the rewritten proposal renderer counts votes through `dao.Tally`, which writes the projected outcome back onto the live proposal's status, so an in-progress proposal renders as "Status: passed" while voting is still open.

## Glossary

- **Sealable** — `Seal() Sealable; IsSealed() bool`. Returns an immutable copy; further writes panic.
- **ExecFunc** — `func(pkgPath string) error`, returned by `Executable.Executor()`. As of this PR the executor receives only the caller package path, not realm authority, and runs in its own declaring-realm context.
- **member grouping** — second-tier storage attached to a `MemberStorage`; ungrouped `Size()`/`Has()` do not reach into groups.
- **VotingContext** — the sealed `(VotingRecord, Members)` snapshot passed to a `Definition.Tally`.

## Fix

Round 3 is a realm-only migration plus one render rewrite. Executors drop their `realm` parameter ([`proposal_members.gno:122`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/proposal_members.gno#L122) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/proposal_members.gno#L122)), the renderer reads proposal title/body straight off `Proposal` instead of the removed `Definition()` accessor, and the proposal-detail view replaces its hand-rolled tally with `dao.Tally(p.ID(), true)` ([`render.gno:415-427`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L427) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L427)). The load-bearing constraint missed: `CommonDAO.Tally` is a mutator (its own doc says it "updates proposal status"), so calling it from a renderer is only safe for chain state because render runs against a throwaway query store, and it is still wrong for the rendered output because the same render then reads back the mutated status.

## Critical (must fix)

- **[grouped-member DAO can pass quorum but always fail the vote, unchanged from rounds 1-2]** [`examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno:82-92`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L82-L92) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L82-L92) — quorum uses `GetTotalMemberStorageSize` (grouped + ungrouped) but the super-majority threshold uses `ctx.Members.Size()` (ungrouped only).
  <details><summary>details</summary>

  Same shape in all three canonical definitions: `proposal_members.gno:106-111` ([↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/proposal_members.gno#L106-L111)), `proposal_subdao.gno:82-92` (subDAO), and `proposal_subdao.gno:146-156` ([↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L146-L156)) (dissolve). Each computes `memberCount := GetTotalMemberStorageSize(ctx.Members)` for `IsQuorumReached`, then calls `SelectChoiceBySuperMajority(ctx.VotingRecord, ctx.Members.Size())`. `SelectChoiceBySuperMajority` returns `ChoiceNone, false` when its count is `< 3` ([`record.gno:172-173`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/record.gno#L172-L173) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L172-L173)) and otherwise needs `ceil(2*count/3)` YES votes ([`record.gno:177`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/record.gno#L177) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L177)). Scenario for the subDAO definition: a DAO with all members in groups (0 ungrouped, e.g. 10 grouped). `GetTotalMemberStorageSize = 10`, `QuorumFull` needs 100% so 10 YES votes pass the quorum gate; then `SelectChoiceBySuperMajority(record, 0)` hits the `< 3` early-return and yields `false`, so the proposal fails even with unanimous YES. The members-update definition partly shields itself with a `ctx.Members.Size() < 3` early YES-path at `proposal_members.gno:102`, but the two subDAO/dissolve definitions have no such guard, so the mismatch is fully exposed there. The realm itself only adds ungrouped members today, so this surfaces for downstream realms that compose grouping, but the bug lives in the canonical example definitions that downstream realms copy. Fix: pass the same count to both calls, `SelectChoiceBySuperMajority(ctx.VotingRecord, memberCount)`.
  </details>

## Warnings (should fix)

- **[active proposal renders as "passed"/"rejected" while voting is still open]** [`examples/gno.land/r/nt/commondao/v0/render.gno:415-429`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L429) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L429) — `dao.Tally(p.ID(), true)` mutates `p.status`, and the status line two lines down reads the mutated value.
  <details><summary>details</summary>

  `CommonDAO.Tally` is a mutator: `p.tally` assigns `p.status = StatusPassed` / `StatusRejected` ([`proposal.gno:291-295`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/proposal.gno#L291-L295) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L291-L295)), and the method's own doc says it "updates proposal status with the current outcome" ([`commondao.gno:270-271`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L270-L271) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L270-L271)). `render.gno:415` only enters the block when `p.Status() == StatusActive`, calls `dao.Tally(p.ID(), true)` at line 416, then `render.gno:429` builds `"Status: " + p.Status()` from the now-mutated proposal. So an active, mid-voting-period proposal that is currently ahead renders `Status: passed` (or `rejected`), contradicting the `Expected Outcome` line the same block just wrote and implying the vote is over. Chain state is unaffected: render runs on a throwaway query store (`newGnoTransactionStore`, "throwaway (never committed)", [`keeper.go:1325`](https://github.com/gnolang/gno/blob/0b6b302d2/gno.land/pkg/sdk/vm/keeper.go#L1325) · [↗](../../../../../.worktrees/gno-review-5598/gno.land/pkg/sdk/vm/keeper.go#L1325)), so the write is discarded after the query, which is why this is display-only and not a Critical. The round-2 review already warned this rewrite needed a non-mutating tally path before the renderer could adopt `dao.Tally`; it was adopted without one. Fix: add a read-only projection (e.g. `TallyPreview(proposalID) (ProposalStatus, error)` that returns the status without assigning it back) and use it here, or compute the projected outcome into a local and never let it reach the `Status:` line. Verified behaviorally (see [repro](comment_claude-opus-4-8.md)): before `dao.Tally`, `p.Status()` is `active`; after, `passed`.
  </details>

- **[stale public docs lie about live signature, unchanged from round 2]** [`examples/gno.land/p/nt/commondao/v0/README.md:128-136`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/README.md?plain=1#L128-L136) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/README.md#L128-L136) — README still documents `Executor() func(realm) error` and calls it "the crossing function returned by `Executor()`".
  <details><summary>details</summary>

  The live interface is `Executor() ExecFunc` with `ExecFunc func(pkgPath string) error`; the realm was migrated this round but the README at lines 131 and 135 was not. A user copying the README will write the old signature and hit the type error the in-tree realm just had fixed. Fix: change the documented type to `Executor() func(pkgPath string) error` and reword the "crossing function" sentence; the executor is no longer wrapped in `cross(rlm)` and runs in its declaring realm's context, which deserves an explicit line in the Security section.
  </details>

- **[double seal, unchanged from rounds 1-2]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:134-139`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L134-L139) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L134-L139) — `Seal()` already replaces `s.grouping` with a sealed copy at [`member_storage.gno:95-97`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L97) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L97); calling `.Seal()` again inside `Grouping()` allocates a fresh sealed copy per call.
  <details><summary>details</summary>

  Idempotent but costs an O(1) allocation per `Grouping()` access on a sealed storage, on the tally hot path (`GetTotalMemberStorageSize` walks `Grouping()`). Fix: when `s.sealed`, return `s.grouping` directly; it is already a sealed copy.
  </details>

- **[interface contract drift, unchanged from round 2]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:18-19`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L18-L19) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L18-L19) — `MemberStorage.Has` interface docstring says "checks if a member exists in the storage" but the concrete implementation only checks ungrouped members.
  <details><summary>details</summary>

  The concrete impl at [`member_storage.gno:111-115`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L115) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L115) carries the "Grouped members are not checked" caveat; the interface comment does not. Callers code against the interface: `CommonDAO.Vote` gates on `dao.Members().Has(member)` ([`commondao.gno:245`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245)), so a DAO whose members live only in groups silently rejects every legitimate vote. Fix: update the interface docstring to say ungrouped-only and point to `ExistsInMemberStorage`, and decide whether `Vote` should use `ExistsInMemberStorage`.
  </details>

- **[dead field, unchanged from round 2]** [`examples/gno.land/p/nt/commondao/v0/commondao.gno:45`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L45) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L45) — `disableCanonicalCheck bool` on `CommonDAO` is declared but never read or written.
  <details><summary>details</summary>

  Grepped the package: the only occurrence on `CommonDAO` is the declaration; the `member_grouping.gno` matches are a different, used field on `memberGrouping`. The field still serializes into realm state, pinning the schema; removing it later is a breaking storage migration. Fix: drop it now.
  </details>

- **[`GetTotalMemberStorageSize` called twice, unchanged from rounds 1-2]** [`examples/gno.land/p/nt/commondao/v0/exts/definition/definition.gno:122,128`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/exts/definition/definition.gno#L122-L128) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/exts/definition/definition.gno#L122-L128) — `TallyByAbsoluteMajority` traverses the storage twice for the same number.
  <details><summary>details</summary>

  Line 122 binds `memberCount := GetTotalMemberStorageSize(ctx.Members)`, line 128 binds `count := GetTotalMemberStorageSize(ctx.Members)`: same call, same value, two O(groups) walks. Fix: reuse `memberCount`.
  </details>

- **[`GetMemberGroups` O(1)→O(G) regression, unchanged from rounds 1-2]** [`examples/gno.land/p/nt/commondao/v0/member_grouping.gno:178-193`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L178-L193) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L178-L193) — iterates every group and probes `Members().Has(member)` per group.
  <details><summary>details</summary>

  Was a per-member reverse index; now O(G) per call. `boards2` exts/permissions calls this on every `HasPermission`, `SetUserRoles`, `RemoveUser`, `IterateUsers` ([`permissions.gno:95`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95)). Fix: keep a `member → []groupName` secondary index, or document the cost ceiling so realm authors bound role counts deliberately.
  </details>

- **[`GetMeta` panics on sealed non-Sealable meta, unchanged from rounds 1-2]** [`examples/gno.land/p/nt/commondao/v0/member_group.gno:122-134`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122-L134) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122-L134) — `GetMeta()` on a sealed group with non-`Sealable` metadata panics rather than returning nil.
  <details><summary>details</summary>

  `boards2` permissions store `boards.PermissionSet` via `SetMeta` ([`permissions.gno:85`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85)), a type that does not implement `Sealable`. A sealed `MemberGroup` whose `GetMeta()` is reached (e.g. through a sealed `VotingContext.Members.Grouping().Get(...)` in a custom `Tally`) panics. `GetMeta` returns a single `any` ([`member_group.gno:122`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122)), so the remedy is to return the unsealed metadata (or nil) instead of panicking, or to require `Sealable` meta at `SetMeta` time.
  </details>

- **[`GetTotalMemberStorageSize` double-counts multi-group members, unchanged from rounds 1-2]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:154-170`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L154-L170) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L154-L170) — sums each group's `Size()` plus ungrouped; an address in two groups is counted twice.
  <details><summary>details</summary>

  The docstring acknowledges it ("without considering potential duplicate members"); a TODO references a future unique iterator. The impact is an inflated quorum denominator when groups overlap (`singleUserRole == false` is the `boards2` default), biasing every quorum check toward rejection. Fix: ship `GetUniqueMemberStorageSize` (an `address → struct{}` AVL walked across groups), or rename to make the slot-count semantics obvious in the symbol.
  </details>

## Nits

- [`examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno:141`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L141) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L141) — typo `"dissolveed"`. Unchanged from rounds 1-2.
- [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:158`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L158) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L158) — typo `"interaface"`. Unchanged from rounds 1-2.
- [`examples/gno.land/p/nt/commondao/v0/commondao.gno:74`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L74) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L74) — typo `implementaions`; also `instace` at [`commondao.gno:57`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L57) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L57).
- [`examples/gno.land/p/nt/commondao/v0/exts/definition/options.gno:72`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/exts/definition/options.gno#L72) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/exts/definition/options.gno#L72) — typo `validaton`.
- [`examples/gno.land/p/nt/commondao/v0/proposal.gno:74`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/proposal.gno#L74) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L74) — typo `esentially`.
- [`examples/gno.land/p/nt/commondao/v0/commondao_options.gno:81`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao_options.gno#L81) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_options.gno#L81) — typo `custopm`.

## Missing Tests

- **[regression cover]** [`examples/gno.land/p/nt/commondao/v0/commondao_test.gno:353-362`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L353-L362) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L353-L362) — no test asserts the `Tally`-then-`Execute` archive path for a rejected proposal.
  <details><summary>details</summary>

  The `rejected` case verifies status `StatusRejected` and presence in `FinishedProposals`, covering the headline archive bug. But no case calls `dao.Tally(id, false)` to flip status, then `dao.Execute(id, cur)`, asserting the proposal lands in `finishedProposals` exactly once and not in `activeProposals`.
  </details>

- **[panicking Tally recover]** [`examples/gno.land/p/nt/commondao/v0/proposal.gno:267-297`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/proposal.gno#L267-L297) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L267-L297) — the recover-and-translate logic has three branches and zero table cases.
  <details><summary>details</summary>

  `tallyPanic` is a fixture field used only inside the `Tally` body; no top-level test sets it. The recover differentiates string / error / other into `errors.New(v)`, `err = v`, `ErrProposalDefinitionPanicked`. The PR's first headline fix is invisible to the suite. Add three rows: panic with a string, an error, and an int.
  </details>

- **[negative offset]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:195-243`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L195-L243) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L195-L243) — `IterateMemberStorage` does not guard `offset < 0`.
  <details><summary>details</summary>

  `if count <= 0` short-circuits but a negative offset reaches `IterateByOffset` → `bptree` and panics. Clamp at entry or panic with a clear message.
  </details>

- **[Tally not-allowed states]** [`examples/gno.land/p/nt/commondao/v0/commondao.gno:282-284`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L282-L284) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L282-L284) — the `ErrTallyNotAllowed` branch is untested and may be unreachable.
  <details><summary>details</summary>

  Public `Tally` only looks at `activeProposals`, so executed/failed/withdrawn proposals aren't found and return `ErrProposalNotFound` before the status guard at lines 282-284 can fire. Either it should also consult `finishedProposals` (then the branch is reachable and needs a test), or the guard is dead code. Pick one and test it.
  </details>

- **[`MemberGrouping.Add` non-canonical storage factory]** [`examples/gno.land/p/nt/commondao/v0/member_grouping.gno:119-140`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L119-L140) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L119-L140) — the canonical-check panic is not tested on the safe path.
  <details><summary>details</summary>

  Configure a `NewMemberGrouping` with a `WithMemberStorageFactory` returning a mock, call `.Add("g")`, expect panic. The unsafe path is covered by filetests; the safe-path negative test is missing.
  </details>

- **[render status display]** [`examples/gno.land/r/nt/commondao/v0/render.gno:415-429`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L429) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/r/nt/commondao/v0/render.gno#L415-L429) — no test renders an active proposal's detail page and asserts the `Status:` line.
  <details><summary>details</summary>

  No realm filetest exercises `renderProposal` with votes cast on an active proposal, so the status-display Warning above is invisible to CI. A filetest that creates a DAO, proposes, votes YES to a passing majority, and renders the proposal detail page would catch the `Status: passed` regression (and would protect the fix).
  </details>

## Suggestions

- [`examples/gno.land/p/nt/commondao/v0/sealable.gno:1-13`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/sealable.gno#L1-L13) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/sealable.gno#L1-L13) — add a block comment explaining the "value-receiver + `return &s`" seal idiom; every implementor reverse-engineers it from `member_storage.gno`.
- [`examples/gno.land/p/nt/commondao/v0/commondao.gno:310-368`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/commondao.gno#L310-L368) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L310-L368) — `Execute` is a 60-line function with three "IMPORTANT" comments; extract the validate-then-tally block so the top-level reads in a few lines.
- [`examples/gno.land/p/nt/commondao/v0/README.md`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/README.md) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/README.md) — add a "Migration from previous version" subsection: `Readonly*` → `Sealable`, `Tally` → `tally`, `Executor() func(realm)` → `Executor() func(string)`, `exts/storage` deleted.
- [`examples/gno.land/p/nt/commondao/v0/proposal_storage.gno:9-27`](https://github.com/gnolang/gno/blob/0b6b302d2/examples/gno.land/p/nt/commondao/v0/proposal_storage.gno#L9-L27) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal_storage.gno#L9-L27) — `ProposalStorage` does not embed `Sealable` though the member types do; decide intentionally and document.

## Open questions

- The render rewrite picked `dao.Tally` over a read-only path the round-2 review flagged as a prerequisite. Worth confirming with the author whether a `TallyPreview` belongs in this PR or a follow-up; not posted as a separate question since the render Warning already names the fix.
- Why is `Tally` callable on `StatusPassed`/`StatusRejected` (re-tally allowed) rather than locking status once set? Deferred: no caller depends on the answer in this PR; the render Warning is the concrete consequence.
