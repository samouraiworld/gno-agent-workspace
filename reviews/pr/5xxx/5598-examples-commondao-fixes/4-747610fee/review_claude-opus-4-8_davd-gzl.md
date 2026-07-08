# PR [#5598](https://github.com/gnolang/gno/pull/5598): fix(examples): fixes issues in `commondao` package

URL: https://github.com/gnolang/gno/pull/5598
Author: jeronimoalbi | Base: master | Files: 70 | +2534 -1457
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh) | Commit: `747610fee` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5598 747610fee`

**TL;DR:** The PR fixes bugs and tightens the API of `commondao`, a building-block package for on-chain DAOs that handles members, proposals, voting, and execution. It archives losing proposals, recovers from panics in custom vote-counting, and reworks the read-only/immutable types.

**Verdict: REQUEST CHANGES** — the mixed-`memberCount` quorum-vs-majority bug is still live in the canonical proposal definitions (now behaviorally reproduced: a grouped-member DAO passes quorum then always fails the vote), the render path still flips an active proposal's displayed status to "passed"/"rejected" mid-voting, and the live `p/nt/commondao/v0` package still carries the `Has`-docstring drift, the `GetMeta` sealed-panic, the dead `disableCanonicalCheck` field, and the double-counting quorum denominator.

## What changed since round 3 (`0b6b302d2` → `747610fee`)

Patch-ids differ, so this is a full round. The head moved for three reasons:

1. **Base advanced (not PR content).** Master merged [#5726](https://github.com/gnolang/gno/pull/5726), which moved the realm `r/nt/commondao/v0` and the package `p/nt/commondao/v0/exts/definition` into `examples/quarantined/`. All round-3 realm and definition-ext findings re-anchor to `examples/quarantined/...`. Quarantined packages are excluded from the test13 genesis, so the realm-specific findings (the Critical, the render Warning) are no longer deployed on-chain, but the definitions remain the canonical example downstream DAO authors copy.
2. **PR content: one source change.** `record.gno` switched `VotingRecord`'s `votes`/`count` from `bptree.BPTree` to `avl.Tree` ([`record.gno:7,40-41`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/record.gno#L7) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L7)). Verified: the full package unit suite and all 25 filetests pass, and the switch changed one carried finding's failure mode (negative-offset no longer panics, see below).
3. **PR content: two test-fix commits** ([`d13278bd7`](https://github.com/gnolang/gno/commit/d13278bd7), [`7d175e75b`](https://github.com/gnolang/gno/commit/7d175e75b), both by [@thehowl](https://github.com/thehowl)). They thread `cur realm` into the test functions, correct the `urequire`/`uassert` panic-assertion calls to pass `cur` (not `cross(cur)`), and swap `ErrorIs`→`ErrorContains` in the Tally test. Test-only; they let the package compile and pass under the current interrealm test API.

Resolved since round 3: the "panicking Tally recover" Missing Test — `TestCommonDAOTally` now carries three cases (panic with error, error string, unknown type at [`commondao_test.gno:374-392`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L374-L392) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L374-L392)), which is exactly the coverage that finding asked for. Dropped. The two round-2 build-break Criticals stayed resolved (CI green). Everything else carries forward, re-anchored to the current shas and paths.

## Summary

`commondao` is a `/p/` library that realms compose into governance: members live in a `MemberStorage` (with optional second-tier groupings), proposals carry a user-supplied `Definition` whose `Tally(ctx)` counts votes, and `Execute` runs the winning proposal. Two correctness problems survive into this round. The canonical proposal definitions feed two different member counts into one tally — all members for the quorum denominator (`GetTotalMemberStorageSize`), ungrouped-only for the super-majority threshold (`ctx.Members.Size()`) — so a grouped-member DAO passes quorum yet always fails the majority. And the proposal renderer counts votes through `dao.Tally`, a mutator that writes the projected outcome back onto the live proposal's status, so an in-progress proposal renders as `Status: passed` while voting is still open. Both now live in `examples/quarantined/` (not deployed), but the definitions are the reference implementation.

## Examples

Grouped-member DAO, all 10 members in groups (0 ungrouped), unanimous YES, subDAO definition ([`proposal_subdao.gno:83-88`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88)):

| Gate | Count fed | Result |
|------|-----------|--------|
| `IsQuorumReached(QuorumFull, memberCount)` | 10 (grouped+ungrouped) | passes |
| `SelectChoiceBySuperMajority(record, Members.Size())` | 0 (ungrouped only) | `<3` → false |
| `SelectChoiceBySuperMajority(record, memberCount)` (fixed) | 10 | true |

The vote passes quorum and is then silently rejected despite being unanimous.

## Glossary

- **Sealable** — `Seal() Sealable; IsSealed() bool`. Returns an immutable copy; further writes panic.
- **ExecFunc** — `func(pkgPath string) error`, returned by `Executable.Executor()`. As of this PR the executor receives only the caller package path, not realm authority, and runs in its own declaring-realm context.
- **member grouping** — second-tier storage attached to a `MemberStorage`; ungrouped `Size()`/`Has()` do not reach into groups.
- **VotingContext** — the sealed `(VotingRecord, Members)` snapshot passed to a `Definition.Tally`.
- **quarantined** — moved under `examples/quarantined/` by [#5726](https://github.com/gnolang/gno/pull/5726); compiled and tested but excluded from test13 genesis.

## Fix

Round 4 carries the same two behavioral defects into their new quarantined paths, plus the live `p/nt/commondao/v0` API issues. The canonical definitions pass `GetTotalMemberStorageSize(ctx.Members)` to `IsQuorumReached` but `ctx.Members.Size()` to `SelectChoiceBySuperMajority` ([`proposal_subdao.gno:83-88`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88)); the one-line fix is to pass `memberCount` to both. The renderer calls `dao.Tally(p.ID(), true)` inside the `StatusActive` block and then reads `p.Status()` two lines down ([`render.gno:415-429`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429)); the load-bearing constraint is that `CommonDAO.Tally` is a mutator, so a renderer needs a read-only projection that never writes the status back.

## Critical (must fix)

- **[grouped-member DAO can pass quorum but always fail the vote, unchanged from rounds 1-3]** [`examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:83-88`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83-L88) — quorum uses `GetTotalMemberStorageSize` (grouped + ungrouped) but the super-majority threshold uses `ctx.Members.Size()` (ungrouped only).
  <details><summary>details</summary>

  Same shape in all three canonical definitions: `proposal_members.gno:106-111` ([↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_members.gno#L106-L111)), `proposal_subdao.gno:83-88` (subDAO), and `proposal_subdao.gno:147-152` ([↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L147-L152)) (dissolve). Each computes `memberCount := GetTotalMemberStorageSize(ctx.Members)` for `IsQuorumReached`, then calls `SelectChoiceBySuperMajority(ctx.VotingRecord, ctx.Members.Size())`. `SelectChoiceBySuperMajority` returns `ChoiceNone, false` when its count is `< 3` ([`record.gno:172-173`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/record.gno#L172-L173) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L172-L173)) and otherwise needs `ceil(2*count/3)` YES votes ([`record.gno:177`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/record.gno#L177) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L177)). Reproduced on 747610fee (see [repro](comment_claude-opus-4-8.md)): a storage with 3 members all in one group (0 ungrouped) yields `GetTotalMemberStorageSize == 3` but `Size() == 0`; `IsQuorumReached(QuorumFull, 3, record)` is `true` while `SelectChoiceBySuperMajority(record, 0)` is `false` and `SelectChoiceBySuperMajority(record, 3)` is `true`. The members-update definition partly shields itself with a `ctx.Members.Size() < 3` early YES-path at [`proposal_members.gno:102`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_members.gno#L102) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_members.gno#L102), but the two subDAO/dissolve definitions have no such guard, so the mismatch is fully exposed there. The realm is now quarantined, so no deployed DAO hits this today, but the definitions are the canonical example downstream realms copy. Fix: pass the same count to both calls, `SelectChoiceBySuperMajority(ctx.VotingRecord, memberCount)`.
  </details>

## Warnings (should fix)

- **[active proposal renders as "passed"/"rejected" while voting is still open, unchanged from round 3]** [`examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:415-429`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429) — `dao.Tally(p.ID(), true)` mutates `p.status`, and the status line built two lines down reads the mutated value.
  <details><summary>details</summary>

  `CommonDAO.Tally` is a mutator: `p.tally` assigns `p.status = StatusPassed` / `StatusRejected` ([`proposal.gno:292-294`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/proposal.gno#L292-L294) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L292-L294)), and the method's own doc says it "updates proposal status with the current outcome" ([`commondao.gno:270-272`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L270-L272) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L270-L272)). `render.gno:415` only enters the block when `p.Status() == StatusActive`, calls `dao.Tally(p.ID(), true)` at line 416, then `render.gno:429` builds `"Status: " + p.Status()` from the now-mutated proposal. So an active, mid-voting-period proposal that is currently ahead renders `Status: passed`, contradicting the `Expected Outcome` line the same block just wrote. Chain state is unaffected: render runs on a throwaway query store, so the write is discarded after the query, which is why this is display-only and not a Critical. Reproduced on 747610fee (see [repro](comment_claude-opus-4-8.md)): before `dao.Tally`, `p.Status()` is `active`; after, `passed`. Fix: add a read-only projection (e.g. `TallyPreview(proposalID) (ProposalStatus, error)` that returns the status without assigning it back) and use it here, or compute the projected outcome into a local and never let it reach the `Status:` line.
  </details>

- **[stale public docs lie about live signature, unchanged from rounds 2-3]** [`examples/gno.land/p/nt/commondao/v0/README.md:130-135`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/README.md?plain=1#L130-L135) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/README.md#L130-L135) — README still documents `Executor() func(realm) error` and calls it "the crossing function returned by `Executor()`".
  <details><summary>details</summary>

  The live interface is `Executor() ExecFunc` with `ExecFunc func(pkgPath string) error`; the realm was migrated in an earlier round but the README at lines 131 and 135 was not. A user copying the README will write the old signature and hit a type error. Fix: change the documented type to `Executor() func(pkgPath string) error` and reword the "crossing function" sentence; the executor is no longer wrapped in `cross(rlm)` and runs in its declaring realm's context, which deserves an explicit line in the Security section.
  </details>

- **[double seal on the tally hot path, unchanged from rounds 1-3]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:134-139`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L134-L139) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L134-L139) — `Seal()` already replaces `s.grouping` with a sealed copy at [`member_storage.gno:95-96`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L96) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L96); calling `.Seal()` again inside `Grouping()` allocates a fresh sealed copy per call.
  <details><summary>details</summary>

  Idempotent but costs an O(1) allocation per `Grouping()` access on a sealed storage, on the tally hot path (`GetTotalMemberStorageSize` walks `Grouping()`). Fix: when `s.sealed`, return `s.grouping` directly; it is already a sealed copy.
  </details>

- **[interface contract drift, unchanged from rounds 2-3]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:18-19`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L18-L19) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L18-L19) — `MemberStorage.Has` interface docstring says "checks if a member exists in the storage" but the concrete implementation only checks ungrouped members.
  <details><summary>details</summary>

  The concrete impl at [`member_storage.gno:111-114`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L114) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L114) carries the "Grouped members are not checked" caveat; the interface comment does not. Callers code against the interface: `CommonDAO.Vote` gates on `dao.Members().Has(member)` ([`commondao.gno:245`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245)), so a DAO whose members live only in groups silently rejects every legitimate vote. Fix: update the interface docstring to say ungrouped-only and point to `ExistsInMemberStorage`, and decide whether `Vote` should use `ExistsInMemberStorage`.
  </details>

- **[dead field pins realm-state schema, unchanged from rounds 2-3]** [`examples/gno.land/p/nt/commondao/v0/commondao.gno:45`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L45) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L45) — `disableCanonicalCheck bool` on `CommonDAO` is declared but never read or written.
  <details><summary>details</summary>

  Grepped the package: the only occurrence on `CommonDAO` is the declaration; the `member_grouping.gno` matches are a different, used field on `memberGrouping`. The field still serializes into realm state, pinning the schema; removing it later is a breaking storage migration. Fix: drop it now.
  </details>

- **[`GetTotalMemberStorageSize` called twice, unchanged from rounds 1-3]** [`examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/definition.gno:122-128`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/definition.gno#L122-L128) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/definition.gno#L122-L128) — `TallyByAbsoluteMajority` traverses the storage twice for the same number.
  <details><summary>details</summary>

  Line 122 binds `memberCount := GetTotalMemberStorageSize(ctx.Members)`, line 128 binds `count := GetTotalMemberStorageSize(ctx.Members)`: same call, same value, two O(groups) walks. Unlike the realm definitions above, this one feeds the total count to both the quorum check and the majority, so it is not affected by the mixed-count Critical; the only defect is the redundant traversal. Fix: reuse `memberCount`.
  </details>

- **[`GetMemberGroups` O(1)→O(G) regression, unchanged from rounds 1-3]** [`examples/gno.land/p/nt/commondao/v0/member_grouping.gno:179-193`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L179-L193) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L179-L193) — iterates every group and probes `Members().Has(member)` per group.
  <details><summary>details</summary>

  Was a per-member reverse index; now O(G) per call. `boards2` exts/permissions calls this on every `HasPermission`, `SetUserRoles`, `RemoveUser`, `IterateUsers` ([`permissions.gno:95`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95)). Fix: keep a `member → []groupName` secondary index, or document the cost ceiling so realm authors bound role counts deliberately.
  </details>

- **[`GetMeta` panics on sealed non-Sealable meta, unchanged from rounds 1-3]** [`examples/gno.land/p/nt/commondao/v0/member_group.gno:122-133`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122-L133) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122-L133) — `GetMeta()` on a sealed group with non-`Sealable` metadata panics rather than returning nil.
  <details><summary>details</summary>

  `boards2` permissions store `boards.PermissionSet` via `SetMeta` ([`permissions.gno:85`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85)); `PermissionSet` implements only `Has` and `IsEmpty` ([`permission_set.gno:36,47`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/gnoland/boards/permission_set.gno#L36) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/permission_set.gno#L36)), so it is not `Sealable`. A sealed `MemberGroup` whose `GetMeta()` is reached (e.g. through a sealed `VotingContext.Members.Grouping().Get(...)` in a custom `Tally`) hits the `panic` at [`member_group.gno:131`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_group.gno#L131) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_group.gno#L131). `GetMeta` returns a single `any`, so the remedy is to return the unsealed metadata (or nil) instead of panicking, or to require `Sealable` meta at `SetMeta` time.
  </details>

- **[`GetTotalMemberStorageSize` double-counts multi-group members, unchanged from rounds 1-3]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:157-170`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L157-L170) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L157-L170) — sums each group's `Size()` plus ungrouped; an address in two groups is counted twice.
  <details><summary>details</summary>

  The docstring acknowledges it ("without considering potential duplicate members"); a TODO references a future unique iterator. The impact is an inflated quorum denominator when groups overlap (`singleUserRole == false` is the `boards2` default), biasing every quorum check toward rejection. Fix: ship `GetUniqueMemberStorageSize` (an `address → struct{}` AVL walked across groups), or rename to make the slot-count semantics obvious in the symbol.
  </details>

## Nits

- [`examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:141`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L141) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L141) — typo `"dissolveed"`. Unchanged from rounds 1-3.
- [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:158`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L158) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L158) — typo `"interaface"`. Unchanged from rounds 1-3.
- [`examples/gno.land/p/nt/commondao/v0/commondao.gno:74`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L74) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L74) — typo `implementaions`; also `instace` at [`commondao.gno:57`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L57) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L57).
- [`examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/options.gno:72`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/options.gno#L72) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/options.gno#L72) — typo `validaton`.
- [`examples/gno.land/p/nt/commondao/v0/proposal.gno:74`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/proposal.gno#L74) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L74) — typo `esentially`.
- [`examples/gno.land/p/nt/commondao/v0/commondao_options.gno:81`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao_options.gno#L81) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_options.gno#L81) — typo `custopm`.

## Missing Tests

- **[unvalidated negative offset, failure mode changed by the avl switch]** [`examples/gno.land/p/nt/commondao/v0/member_storage.gno:195-243`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L195-L243) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L195-L243) — `IterateMemberStorage` guards `count <= 0` but not `offset < 0`.
  <details><summary>details</summary>

  Round 3 flagged this as a panic (bptree rejected the negative offset). After the `avl.Tree` switch the failure mode changed: a negative offset no longer panics, it is silently accepted and the walk returns members from the start of the storage. Verified on 747610fee: `IterateMemberStorage(storage, -1, 10, fn)` and `IterateMemberStorage(storage, -2, 10, fn)` both visit every member with no panic. A paginating caller that passes a bad negative offset gets the first page instead of an error. The existing table has a `count negative` case at [`member_storage_test.gno:583`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage_test.gno#L583) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage_test.gno#L583) but none for a negative offset. Decide the contract (clamp to 0, or panic with a clear message) and add the case.
  </details>

- **[Tally not-allowed states]** [`examples/gno.land/p/nt/commondao/v0/commondao.gno:282-283`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L282-L283) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L282-L283) — the `ErrTallyNotAllowed` branch is untested and may be unreachable.
  <details><summary>details</summary>

  Public `Tally` only looks at `activeProposals` ([`commondao.gno:277`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L277) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L277)), so executed/failed/withdrawn proposals aren't found and return `ErrProposalNotFound` at line 279 before the status guard at 282-283 can fire. Either it should also consult `finishedProposals` (then the branch is reachable and needs a test), or the guard is dead code. Pick one and test it. No test references `ErrTallyNotAllowed`.
  </details>

- **[`MemberGrouping.Add` non-canonical storage factory]** [`examples/gno.land/p/nt/commondao/v0/member_grouping.gno:119-140`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L119-L140) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L119-L140) — the canonical-check panic on the safe path is not tested.
  <details><summary>details</summary>

  `TestMemberGroupingAdd` covers the sealed-panic path but not the `assertGroupingMemberStorageIsCanonical` panic at [`member_grouping.gno:128`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L128) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L128). Configure a `NewMemberStorageWithGrouping` with a `WithMemberStorageFactory` returning a mock, call `.Add("g")`, expect the panic.
  </details>

- **[render status display]** [`examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:415-429`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415-L429) — no realm filetest renders an active proposal's detail page and asserts the `Status:` line.
  <details><summary>details</summary>

  No realm filetest exercises `renderProposal` with votes cast on an active proposal, so the status-display Warning above is invisible to CI. A filetest that creates a DAO, proposes, votes YES to a passing majority, and renders the proposal detail page would catch the `Status: passed` regression and protect the fix.
  </details>

- **[rejected proposal archived exactly once via the public-Tally path]** [`examples/gno.land/p/nt/commondao/v0/commondao_test.gno:449-457`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L449-L457) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_test.gno#L449-L457) — the headline archive path is covered, the Tally-then-Execute sequence is not.
  <details><summary>details</summary>

  The `rejected` case in `TestCommonDAOExecute` proposes then `Execute`s directly, asserting `StatusRejected` and presence in `FinishedProposals`, which covers the headline "rejected proposals were not finished" fix. No case first calls `dao.Tally(id, false)` to flip status to `StatusRejected`, then `dao.Execute(id, cur)` — the path the extended execution guard newly allows ([`commondao.gno:320-324`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L320-L324) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L320-L324)) — asserting the proposal lands in `finishedProposals` exactly once and not in `activeProposals`.
  </details>

## Suggestions

- [`examples/gno.land/p/nt/commondao/v0/sealable.gno:1-13`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/sealable.gno#L1-L13) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/sealable.gno#L1-L13) — add a block comment explaining the "value-receiver + `return &s`" seal idiom; every implementor reverse-engineers it from `member_storage.gno`.
- [`examples/gno.land/p/nt/commondao/v0/commondao.gno:310-368`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L310-L368) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L310-L368) — `Execute` is a long function mixing the status guard, validation, tally, the rejected early-return, and execution; extract the validate-tally-execute block so the top-level reads in a few lines.
- [`examples/gno.land/p/nt/commondao/v0/README.md`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/README.md) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/README.md) — add a "Migration from previous version" subsection: `Readonly*` → `Sealable`, `Tally` → `tally`, `Executor() func(realm)` → `Executor() func(string)`, `exts/storage` deleted.
- [`examples/gno.land/p/nt/commondao/v0/proposal_storage.gno:9-27`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/proposal_storage.gno#L9-L27) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal_storage.gno#L9-L27) — `ProposalStorage` does not embed `Sealable` though the member types do; decide intentionally and document.

## Open questions

- `record.gno` switched `VotingRecord` from `bptree.BPTree` to `avl.Tree` this round. Both are deterministic ordered trees and the full suite plus filetests pass, so behavior is preserved; the switch also silently changed the negative-offset failure mode (panic → lenient), captured in the Missing Tests. Worth a word from the author on whether the switch was deliberate or a rebase artifact. Not posted: no defect, and the negative-offset Missing Test already carries the concrete consequence.
- The render rewrite picked `dao.Tally` over a read-only path. Whether a `TallyPreview` belongs in this PR or a follow-up is the author's call; not posted separately since the render Warning already names the fix.
- Why is `Tally` callable on `StatusPassed`/`StatusRejected` (re-tally allowed) rather than locking status once set? Deferred: no caller depends on the answer in this PR; the render Warning is the concrete consequence.
