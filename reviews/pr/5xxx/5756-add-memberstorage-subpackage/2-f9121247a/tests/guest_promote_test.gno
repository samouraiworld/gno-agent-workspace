/* Run: from a gno checkout:
gh pr checkout 5756 -R gnolang/gno && git checkout f9121247a
curl -fsSL -o examples/gno.land/p/gnoland/boards/exts/permissions/guest_promote_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5756-add-memberstorage-subpackage/2-f9121247a/tests/guest_promote_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestSetUserRolesGuestThenPromote ./gno.land/p/gnoland/boards/exts/permissions/
rm gno.land/p/gnoland/boards/exts/permissions/guest_promote_test.gno
*/
// SetUserRoles(user) with no roles puts the user in the ungrouped set; a later
// SetUserRoles(user, "admin") never clears that ungrouped entry, so the user
// occupies both the ungrouped set and the admin group.
// At f9121247a this fails: UsersCount is 2, IterateUsers emits the user twice,
// and RemoveUser leaves the ungrouped ghost behind. It passes once the ungrouped
// entry is cleared on promotion and on removal.
package permissions

import (
	"testing"

	"gno.land/p/gnoland/boards"
)

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
