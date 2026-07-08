# Review: PR [#5756](https://github.com/gnolang/gno/pull/5756)
Event: REQUEST_CHANGES

## Body
Reproduced on f9121247a: after [`AcceptInvite`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) then [`ChangeMemberRole`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649) for one address, the board member list counts and renders that member twice and RemoveMember cannot clear it.

If dropping the old root-set de-duplication was deliberate, the [non-dedup contract](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L116-L119) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L116-L119) needs a decision on how counts and iteration should treat an address that is both a guest and a role-holder.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5756-add-memberstorage-subpackage/2-f9121247a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno:154-163 [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L154)
The clear-roles loop only walks the user's groups, so a user first added as a guest keeps their ungrouped entry when later given a role. UsersCount then counts that address twice, IterateUsers emits it twice, and RemoveUser removes only the group entry and leaves the ungrouped one, so HasUser stays true. Reachable from [`AcceptInvite`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public_invite.gno#L105) followed by [`ChangeMemberRole`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/r/gnoland/boards2/v1/public.gno#L649).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5756 -R gnolang/gno
cat > examples/gno.land/p/gnoland/boards/exts/permissions/zz_repro_test.gno <<'EOF'
package permissions

import (
	"testing"

	"gno.land/p/gnoland/boards"
)

func TestZZGuestPromoteDoubleCount(t *testing.T) {
	user := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	perms := New(WithSuperRole("owner"))
	perms.AddRole("admin", testPermA)

	perms.SetUserRoles(user)          // accepted as guest -> ungrouped set
	perms.SetUserRoles(user, "admin") // promoted -> ungrouped entry never cleared

	println("UsersCount =", perms.UsersCount(), "(want 1)")
	visits := 0
	perms.IterateUsers(0, 100, func(u boards.User) bool { visits++; return false })
	println("IterateUsers visits =", visits, "(want 1)")
	println("ghost ungrouped Has =", perms.members.Has(user), "(want false)")
}
EOF
cd examples && go run ../gnovm/cmd/gno test -v -run TestZZGuestPromoteDoubleCount ./gno.land/p/gnoland/boards/exts/permissions/
rm gno.land/p/gnoland/boards/exts/permissions/zz_repro_test.gno
```

```
UsersCount = 2 (want 1)
IterateUsers visits = 2 (want 1)
ghost ungrouped Has = true (want false)
```
</details>

## examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno:521-524 [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno#L521)
Missing test: no case covers a guest promoted to a role, the flow the double-count breaks. This iterate test caps count at the distinct-user count, so it stops before the duplicate occurrence and never surfaces it.

<details><summary>test cases</summary>

Add to [`permissions_test.gno`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno?plain=1) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions_test.gno). Fails red on f9121247a, passes once the ungrouped entry is cleared on promotion and removal.

```go
func TestSetUserRolesGuestThenPromote(t *testing.T) {
	user := address("g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")
	perms := New(WithSuperRole("owner"))
	perms.AddRole("admin", testPermA)

	perms.SetUserRoles(user)          // accepted as guest -> ungrouped set
	perms.SetUserRoles(user, "admin") // promoted -> admin group, ungrouped entry never cleared

	if got := perms.UsersCount(); got != 1 {
		t.Errorf("UsersCount = %d, want 1", got)
	}

	visits := 0
	perms.IterateUsers(0, 100, func(u boards.User) bool {
		visits++
		return false
	})
	if visits != 1 {
		t.Errorf("IterateUsers visits = %d, want 1", visits)
	}

	if !perms.RemoveUser(user) {
		t.Error("RemoveUser returned false for an existing user")
	}
	if perms.HasUser(user) {
		t.Error("HasUser still true after RemoveUser: ungrouped ghost survived")
	}
}
```
</details>

## examples/gno.land/p/gnoland/boards/exts/memberstorage/member_group.gno:29 [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_group.gno#L29)
`IsCanonical` breaks the sibling naming set by [`IsCanonicalMemberStorage`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L45) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_storage.gno#L45) and [`IsCanonicalMemberGrouping`](https://github.com/gnolang/gno/blob/f9121247a/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L43) [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L43). Rename to `IsCanonicalMemberGroup`.

## examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno:88-91 [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/memberstorage/member_grouping.gno#L88)
`Delete` discards `bptree.Remove`'s result and always returns nil, so deleting a group that does not exist is a silent success. `Add` errors on a duplicate, so the two are asymmetric.

## examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno:213-226 [↗](../../../../../.worktrees/gno-review-5756/examples/gno.land/p/gnoland/boards/exts/permissions/permissions.gno#L213)
`IterateUsers` now emits members in ungrouped-then-group-name order instead of address order, which the two changed filetest goldens already bake in. Any external consumer that assumed address-sorted output is silently affected.
