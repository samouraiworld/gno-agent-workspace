// Asserts SetApprovalForAll(op, false) for a never-approved operator leaves the
// operatorApprovals tree empty, the way Approve's revoke and setBalance(0) do.
// Fails at f151f475d: the tree holds one node, so it grows per operator.
//
// Run: from a gno checkout:
//   gh pr checkout 6072 -R gnolang/gno && git checkout f151f475d
//   cp <this-file> examples/gno.land/p/demo/tokens/grc721/zz_sgd_false_store_test.gno
//   ( cd examples && go run ../gnovm/cmd/gno test -v -run TestSGDFalseDoesNotStore ./gno.land/p/demo/tokens/grc721/ )
//   rm examples/gno.land/p/demo/tokens/grc721/zz_sgd_false_store_test.gno
package grc721

import (
	"testing"

	"gno.land/p/nt/uassert/v0"
)

func TestSGDFalseDoesNotStore(cur realm, t *testing.T) {
	_, led := newTestToken("Foo", "FOO", 0, cur)
	uassert.NoError(t, led.SetApprovalForAll(alice, bob, false))
	uassert.Equal(t, 0, led.operatorApprovals.Size()) // IS 1 at head; SHOULD be 0
}
