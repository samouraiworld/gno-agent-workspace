# PR #5598: fix(examples): fixes issues in `commondao` package

**URL:** https://github.com/gnolang/gno/pull/5598
**Author:** jeronimoalbi | **Base:** master | **Files:** 42 | **+2266 -1122**
**Reviewed by:** davd-gzl | **Model:** claude-sonnet-4-6

## Summary

Fixes multiple issues in `commondao` (`examples/gno.land/p/nt/commondao/v0`). Core bug fixes: `Execute()` archives rejected proposals; `Proposal.Tally` privatised with panic recovery; new `CommonDAO.Tally()` for pre-tallying. API refactored from `Readonly*` to `Sealable` interface. `exts/storage` deleted; helpers promoted to core.

## Test Results
- **Existing tests:** UNABLE TO RUN (gno CLI environment mismatch)
- **Edge-case tests:** Skipped

## Critical (must fix)
- [ ] `r/nt/commondao/v0/proposal_members.gno:106-111` — Mixed memberCount: `IsQuorumReached` uses `GetTotalMemberStorageSize` (all members) but `SelectChoiceBySuperMajority` uses `ctx.Members.Size()` (ungrouped only). Same bug at `proposal_subdao.gno:83-88` and `:147-152`. Silently wrong with grouping storage.

## Warnings (should fix)
- [ ] `member_storage.gno:63-69` — Double-seal in `Grouping()`.
- [ ] `commondao.gno:278-284` — `Execute()` re-tallies even when `Tally()` pre-called.
- [ ] `member_grouping.gno:134-149` — `GetMemberGroups` O(1) to O(G*M) regression.
- [ ] `member_group.gno:97-108` — `GetMeta` panics when sealed and meta non-sealable.
- [ ] `member_storage.gno:127-128` — `GetTotalMemberStorageSize` double-counts grouped members.

## Nits
- [ ] `exts/definition/definition.gno:122,128` — `GetTotalMemberStorageSize` called twice.
- [ ] `proposal_subdao.gno:141` — Typo `"dissolveed"`.
- [ ] `member_storage.gno:128` — Typos `"interaface"`, `"though"`.

## Missing Tests
- [ ] Execute-after-Tally integration.
- [ ] `ErrTallyNotAllowed` on finished proposals.
- [ ] `GetMemberGroups` with member in multiple groups.
- [ ] `IterateMemberStorage` negative offset.

## Suggestions
- Revisit `Sealable` with generics when available.
- `GetMemberGroups` should accept `MemberGrouping` for composability.
- Document Tally/Execute asymmetry.

## Questions for Author
- Was removal of `ErrNoQuorum` intentional?
- Is `GetTotalMemberStorageSize` double-counting intentional?
- Migration path for deleted `exts/storage`?

## Verdict
REQUEST CHANGES — Mixed memberCount in quorum vs majority calculations is a latent correctness bug.
