/* Run: from a gno checkout:
gh pr checkout 5871 -R gnolang/gno && git checkout 7c41a297d
curl -fsSL -o examples/gno.land/r/docs/minisocial/v2/zz_window_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5871-revive-correct-rdocs/1-7c41a297d/tests/minisocial_v2_update_window_test.gno
cd examples && gno test -v -run TestZZUpdateWindow ./gno.land/r/docs/minisocial/v2/
rm gno.land/r/docs/minisocial/v2/zz_window_test.gno
*/

// Asserts the ten-minute edit window closes. SkipHeights advances block time
// five seconds a height, so 240 is twenty minutes. Passes at 7c41a297d; put
// master's `post.updatedAt.After(post.createdAt.Add(time.Minute * 10))` back
// in posts.gno and it fails, because a post that was never edited carries
// updatedAt == createdAt and passes the check however old it is.

package minisocial

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
)

func TestZZUpdateWindowClosesAfterTenMinutes(cur realm, t *testing.T) {
	resetPostsForTest()

	alice := testutils.TestAddress("alice")
	testing.SetRealm(testing.NewUserRealm(alice))
	uassert.NoError(t, CreatePost(cross(cur), "original"))
	id := postID.String()

	testing.SkipHeights(240)

	testing.SetRealm(testing.NewUserRealm(alice))
	err := UpdatePost(cross(cur), id, "late edit")
	uassert.ErrorIs(t, err, ErrUpdateWindowExpired, "edit twenty minutes later must be refused")
}
