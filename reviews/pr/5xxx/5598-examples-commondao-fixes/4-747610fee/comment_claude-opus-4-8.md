# Review: PR [#5598](https://github.com/gnolang/gno/pull/5598)
Event: REQUEST_CHANGES

## Body
The realm and the definition extension moved under `examples/quarantined` this round, so the findings against them are example-only now; the ones in the live `p/nt/commondao/v0` package still ship. Repros run at 747610fee.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5598-examples-commondao-fixes/4-747610fee/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:83-88 [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L83)
The quorum check counts all members through [`GetTotalMemberStorageSize`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L157-L170) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L157-L170), but the super-majority threshold uses `ctx.Members.Size()`, which counts ungrouped members only. A DAO whose members all live in groups passes quorum and then always fails the vote, because [`SelectChoiceBySuperMajority`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/record.gno#L171-L181) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/record.gno#L171-L181) receives a count of 0 and returns false. Same mismatch in the dissolve definition at [`proposal_subdao.gno:147-152`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L147-L152) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L147-L152) and the members-update definition at [`proposal_members.gno:106-111`](https://github.com/gnolang/gno/blob/747610fee/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_members.gno#L106-L111) · [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_members.gno#L106-L111).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5598 -R gnolang/gno
cat > examples/gno.land/p/nt/commondao/v0/zz_quorum_majority_filetest.gno <<'EOF'
package main

import "gno.land/p/nt/commondao/v0"

// Mirrors the subDAO/dissolve definitions: quorum uses GetTotalMemberStorageSize
// (grouped+ungrouped), majority uses ctx.Members.Size() (ungrouped only).
func main() {
	addrs := []address{
		"g125t352u4pmdrr57emc4pe04y40sknr5ztng5mt",
		"g12chzmwxw8sezcxe9h2csp0tck76r4ptwdlyyqk",
		"g16jpf0puufcpcjkph5nxueec8etpcldz7zwgydq",
	}

	s := commondao.NewMemberStorageWithGrouping()
	g, _ := s.Grouping().Add("g")
	for _, a := range addrs {
		g.Members().Add(a)
	}

	r := &commondao.VotingRecord{}
	for _, a := range addrs {
		r.AddVote(commondao.Vote{Address: a, Choice: commondao.ChoiceYes})
	}

	total := commondao.GetTotalMemberStorageSize(s)
	println("total:", total, "ungrouped:", s.Size())
	println("quorum(total):", commondao.IsQuorumReached(commondao.QuorumFull, total, *r))
	_, okBug := commondao.SelectChoiceBySuperMajority(*r, s.Size())
	_, okFix := commondao.SelectChoiceBySuperMajority(*r, total)
	println("majority(Size()):", okBug, "majority(total):", okFix)
}

// Output:
// x
EOF
(cd examples && GNOROOT=$(cd .. && pwd) go run ../gnovm/cmd/gno test -v ./gno.land/p/nt/commondao/v0 2>&1 | grep -E 'total:|quorum|majority')
rm examples/gno.land/p/nt/commondao/v0/zz_quorum_majority_filetest.gno
```

```
total: 3 ungrouped: 0
quorum(total): true
majority(Size()): false majority(total): true
```
</details>

## examples/quarantined/gno.land/r/nt/commondao/v0/render.gno:415-429 [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/render.gno#L415)
`dao.Tally(p.ID(), true)` updates the proposal's status to passed or rejected, and the status line below reads that mutated value. An active proposal that is currently ahead renders `Status: passed` while voting is still open, contradicting the `Expected Outcome` line just above it. Chain state is safe because render runs on a throwaway query store, but the displayed status is wrong; use a read-only projection that does not write the status back.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5598 -R gnolang/gno
cat > examples/gno.land/p/nt/commondao/v0/zz_render_status_filetest.gno <<'EOF'
package main

import (
	"time"

	"gno.land/p/nt/commondao/v0"
)

type passDef struct{}

func (passDef) Title() string                               { return "T" }
func (passDef) Body() string                                { return "B" }
func (passDef) VotingPeriod() time.Duration                 { return time.Hour }
func (passDef) Validate() error                             { return nil }
func (passDef) Tally(commondao.VotingContext) (bool, error) { return true, nil }

func main() {
	const member = address("g1w4ek2u33ta047h6lta047h6lta047h6ldvdwpn")
	dao := commondao.New(commondao.WithMember(member))
	p, _ := dao.Propose(member, passDef{})

	println("before:" + string(p.Status()))
	dao.Tally(p.ID(), true) // render.gno:416 calls this on every detail render
	println("after:" + string(p.Status())) // render.gno:429 prints this as "Status:"
}

// Output:
// x
EOF
(cd examples && GNOROOT=$(cd .. && pwd) go run ../gnovm/cmd/gno test -v ./gno.land/p/nt/commondao/v0 2>&1 | grep -E 'before:|after:')
rm examples/gno.land/p/nt/commondao/v0/zz_render_status_filetest.gno
```

```
before:active
after:passed
```
</details>

## examples/gno.land/p/nt/commondao/v0/README.md:130-135 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/README.md#L130)
The README documents `Executor() func(realm) error` and the "crossing function returned by `Executor()`", but the live interface is `Executor() func(pkgPath string) error` and the executor no longer runs through `cross(rlm)`. A reader copying this writes the old signature and hits a type error.

## examples/gno.land/p/nt/commondao/v0/member_storage.gno:134-139 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L134)
On a sealed storage, `Seal()` already stored a sealed copy in `s.grouping` at [`member_storage.gno:95-96`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L96) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L95-L96), so re-sealing here allocates a fresh copy on every `Grouping()` call on the tally hot path. Return `s.grouping` directly when sealed.

## examples/gno.land/p/nt/commondao/v0/member_storage.gno:18-19 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L18)
The interface docstring for `Has` says it checks whether a member exists in the storage, but the implementation at [`member_storage.gno:111-114`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L114) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L111-L114) only checks ungrouped members. `CommonDAO.Vote` gates on this at [`commondao.gno:245`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L245), so a grouped-only DAO silently rejects every legitimate vote. Match the docstring to the behavior and point to `ExistsInMemberStorage`.

## examples/gno.land/p/nt/commondao/v0/commondao.gno:45 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L45)
`disableCanonicalCheck` on `CommonDAO` is never read or written. It still serializes into realm state and pins the schema, so removing it later is a breaking migration. Drop it now.

## examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/definition.gno:122-128 [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/definition.gno#L122)
`memberCount` at line 122 and `count` at line 128 are the same `GetTotalMemberStorageSize(ctx.Members)` call, so the storage is traversed twice for one value. Reuse `memberCount`.

## examples/gno.land/p/nt/commondao/v0/member_grouping.gno:179-193 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_grouping.gno#L179)
`GetMemberGroups` iterates every group and probes `Members().Has(member)` per group, an O(groups) scan that was previously a reverse index. `boards2` permissions call it on every `HasPermission`, `SetUserRoles`, `RemoveUser`, and `IterateUsers` at [`permissions.gno:95`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L95). Keep a member-to-groups index or document the cost ceiling.

## examples/gno.land/p/nt/commondao/v0/member_group.gno:122-133 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_group.gno#L122)
`GetMeta()` on a sealed group panics when the stored metadata does not implement `Sealable`. `boards2` stores a `boards.PermissionSet` via `SetMeta` at [`permissions.gno:85`](https://github.com/gnolang/gno/blob/747610fee/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85) · [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L85), and `PermissionSet` implements only `Has` and `IsEmpty`, so a sealed group's `GetMeta` reached from a custom `Tally` panics. Return the unsealed metadata, or require `Sealable` meta at `SetMeta` time.

## examples/gno.land/p/nt/commondao/v0/member_storage.gno:157-170 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L157)
`GetTotalMemberStorageSize` sums each group's size plus the ungrouped count, so a member in two groups is counted twice. That inflates the quorum denominator when groups overlap, which is the `boards2` default, biasing every quorum check toward rejection. Add a unique-member count, or rename to make the slot-count semantics explicit.

## examples/gno.land/p/nt/commondao/v0/member_storage.gno:195-243 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L195)
`IterateMemberStorage` short-circuits on `count <= 0` but does not validate `offset < 0`. A negative offset is silently accepted and iterates from the start of the storage, so a paginating caller that passes a bad offset gets the first page instead of an error. Decide the contract, clamp to zero or panic, and add the case; the table has a negative-count case but no negative-offset case.

## examples/gno.land/p/nt/commondao/v0/commondao.gno:282-283 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L282)
`Tally` only looks at `activeProposals`, so executed, failed, and withdrawn proposals return `ErrProposalNotFound` before this `ErrTallyNotAllowed` guard can fire. Either consult `finishedProposals` so the branch is reachable, or remove it as dead code. Add a test for whichever you keep.

## examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno:141 [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/r/nt/commondao/v0/proposal_subdao.gno#L141)
Typo: "dissolveed".

## examples/gno.land/p/nt/commondao/v0/member_storage.gno:158 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/member_storage.gno#L158)
Typo: "interaface".

## examples/gno.land/p/nt/commondao/v0/commondao.gno:74 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao.gno#L74)
Typo: "implementaions". Also "instace" at line 57.

## examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/options.gno:72 [↗](../../../../../.worktrees/gno-review-5598/examples/quarantined/gno.land/p/nt/commondao/v0/exts/definition/options.gno#L72)
Typo: "validaton".

## examples/gno.land/p/nt/commondao/v0/proposal.gno:74 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/proposal.gno#L74)
Typo: "esentially".

## examples/gno.land/p/nt/commondao/v0/commondao_options.gno:81 [↗](../../../../../.worktrees/gno-review-5598/examples/gno.land/p/nt/commondao/v0/commondao_options.gno#L81)
Typo: "custopm".
