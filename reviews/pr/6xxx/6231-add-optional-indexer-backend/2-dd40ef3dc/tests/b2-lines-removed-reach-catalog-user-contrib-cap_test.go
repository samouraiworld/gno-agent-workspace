// Repro, from a plain clone of gnolang/gno at dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
// (pathsCaller is the stub the PR adds to client_test.go, so this runs at head only):
//
//	cp b2-lines-removed-reach-catalog-user-contrib-cap_test.go gno.land/pkg/gnoweb/zz_user_contrib_cap_test.go
//	go test ./gno.land/pkg/gnoweb -run TestUserContributionsListingNotLowered -v -count=1
//
// Before the PR, rpcClient.ListPaths dropped its limit and sent "vm/qpaths", so
// the node's pathsLimit default of 1000 governed every caller. The PR forwards
// it, which raises search to 10000 but lowers buildContributions
// (/u/<user>, limit MaxUserContributions = 200) to 200 paths per namespace,
// with no truncated flag on the user page, whose PackageCount and RealmCount
// are then computed over the capped list.
package gnoweb

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/rpc/client"
)

func TestUserContributionsListingNotLowered(t *testing.T) {
	caller := &pathsCaller{listing: "gno.land/r/alice/a"}
	c := NewRPCClientAdapter(newDiscardLogger(), client.NewRPCClient(caller), "gno.land", 0)

	// The exact call buildContributions makes for each namespace.
	if _, err := c.ListPaths(context.Background(), "@alice", MaxUserContributions); err != nil {
		t.Fatal(err)
	}
	t.Logf("buildContributions query path: %q", caller.gotPath)

	limit := 1000 // node default when no ?limit= is sent (vm pathsLimit)
	if _, q, ok := strings.Cut(caller.gotPath, "?"); ok {
		v, _ := url.ParseQuery(q)
		limit, _ = strconv.Atoi(v.Get("limit"))
	}
	if limit < 1000 {
		t.Fatalf("user page now asks the node for %d paths per namespace, below the 1000 it received before the PR", limit)
	}
}
