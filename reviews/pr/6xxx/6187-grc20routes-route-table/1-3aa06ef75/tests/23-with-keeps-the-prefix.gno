// Pins the Prefix carry-over in Route.With (grc20routes.gno:267) for the
// documented keyed builder, Keyed("FOO").With("deposit", "Deposit"). Green at
// head; the PR's own suite is also green with line 267 mutated to
// `out := Route{}`, and this test is what reddens it.
//
// # from a local clone of github.com/gnolang/gno:
//   git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//   go build -o /tmp/gno ./gnovm/cmd/gno
//   P=examples/gno.land/r/nt/grc20routes/v0
//   cp <this file> $P/with_prefix_test.gno
//   (cd examples && /tmp/gno test ./gno.land/r/nt/grc20routes/v0)   # ok
//   sed -i '267s/.*/\tout := Route{}/' $P/grc20routes.gno           # drop the carry-over
//   (cd examples && /tmp/gno test ./gno.land/r/nt/grc20routes/v0)   # FAIL, and ok again with this file removed
//   git checkout $P/grc20routes.gno && rm $P/with_prefix_test.gno

package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

// A keyed realm that exposes more than the GRC20 three builds its route with
// Keyed then With. Every With must carry the leading symbol argument over.
func TestWithKeepsThePrefix(cur realm, t *testing.T) {
	r := Keyed("WPX").With("deposit", "Deposit").With("withdraw", "Withdraw")

	urequire.Equal(t, 1, len(r.Prefix), "With drops the prefix Keyed set")
	uassert.Equal(t, "WPX", r.Prefix[0])
	uassert.Equal(t, "Deposit", r.Func("deposit"))
	uassert.Equal(t, "Approve", r.Func(OpApprove), "the standard three survive With")

	// Through the store, which is the copy a wallet actually reads.
	key := mint(0, cur, "WPX")
	Register(cross(cur), "WPX", r)

	got, _ := Get(key)
	urequire.Equal(t, 1, len(got.Prefix), "the stored route lost the prefix")
	uassert.Equal(t, "WPX", got.Prefix[0])
	uassert.True(t, strings.Contains(JSON(key), `"prefix":["WPX"]`),
		"a wallet builds the leading approve argument from this prefix")
}
