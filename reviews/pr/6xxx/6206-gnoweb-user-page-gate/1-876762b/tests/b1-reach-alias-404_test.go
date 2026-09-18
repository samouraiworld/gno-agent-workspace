// b1-reach-alias-404_test.go — PR 6206, head 876762bdf2ea6635b27e2b0a42f26f9fdc54af24.
//
// Shows that gnoweb's new /u/<name> gate cannot tell a live alias of a
// registered account from a name that was never registered: both render 404.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/b1_reach_alias_404_test.go
//	go test ./gno.land/pkg/gnoweb -run TestB1ReachAliasIsRegisteredButGets404 -v
//
// Observed at head (go1.25.9):
//
//	b1_reach_alias_404_test.go:60: status=404   (never registered)
//	b1_reach_alias_404_test.go:60: status=404   (live alias of bob)
//	--- FAIL: TestB1ReachAliasIsRegisteredButGets404/live_alias_of_bob
//	    Error: Not equal: expected: 200  actual: 404
//
// Mechanism. handler_http.go:566-575 (userExists) splits the vm/qeval body on
// "\n" and reads only the LAST line, which is ResolveName's second result
// `isCurrent` -- examples/gno.land/r/sys/users/users.gno:17,
// `return data, name == data.username`. The first line, the *UserData pointer,
// is the only value that answers "is this name registered", and it is
// discarded. r/sys/users never drops an old key: UpdateName
// (store.gno:192-214) validates the new name and calls
// nameStore.Set(newName, u); nothing removes the old entry. The companion
// artifact b1-reach-alias-registered_test.gno runs that on the VM.
//
// This file uses the package's own stubClient/getUserPage helpers from
// gno.land/pkg/gnoweb/handler_http_test.go, so it must live in that directory.
package gnoweb_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
)

func TestB1ReachAliasIsRegisteredButGets404(t *testing.T) {
	t.Parallel()

	const (
		neverRegistered = "(nil *gno.land/r/sys/users.UserData)\n(false bool)"
		liveAlias       = `(&(struct{("g1manfred47kzduec920z88wfr64ylksmdcedlf5" .uverse.address),("bob" string),(false bool)} gno.land/r/sys/users.UserData) *gno.land/r/sys/users.UserData)
(false bool)`
	)

	for name, body := range map[string]string{
		"never registered":  neverRegistered,
		"live alias of bob": liveAlias,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rr := getUserPage(t, &stubClient{
				// no packages under @alice, so the gate falls through to the registry
				listPathsFunc: func(context.Context, string, int) ([]string, error) {
					return nil, nil
				},
				evalFunc: func(context.Context, string, string) ([]byte, error) {
					return []byte(body), nil
				},
				realmFunc: func(context.Context, string, string) ([]byte, error) {
					return nil, gnoweb.ErrClientPackageNotFound
				},
			}, "/u/alice")

			t.Logf("status=%d", rr.Code)
			// An alias is a registered name, permanently held by its owner and
			// still resolvable on chain; it should render like any user page.
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}
