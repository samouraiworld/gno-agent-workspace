// TestHTTPHandler_GetUserView_RegisteredWithoutPackages pins nothing.
//
// gnolang/gno PR 6206, head 876762bdf2ea6635b27e2b0a42f26f9fdc54af24.
// Finding: gno.land/pkg/gnoweb/handler_http_test.go:1993 (the author's test).
//
// The author's test is the only one that covers the admit half of the new
// registry gate: a name with no packages that r/sys/users says is taken must
// still get a page. It stubs evalFunc to return `(true bool)` and puts its two
// real assertions (the pkgPath and the expression sent to vm/qeval) *inside*
// that stub, with no flag saying the stub ran. Delete the whole gate from
// GetUserView and the handler renders the same 200 page with "Gnome alice" on
// it, so the test stays green and both inner assertions are never reached.
//
// The integration routes added to app_test.go do not cover it either:
// /u/moul001 and /u/zoo_ma123 are registered in gno.land/genesis/genesis_txs.jsonl
// and own no package, and both still return 200 with the gate deleted. Only
// /u/zzznotauser (the reject half) goes red.
//
// This file is the missing test: the same case with an evalCalled flag.
// It passes on the head tree and fails the moment the gate goes away.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno
//	cd gno
//	git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//	cp <this file> gno.land/pkg/gnoweb/zz_gate_probe_test.go
//
//	# 1. head tree: the author's test and this one both pass
//	go test ./gno.land/pkg/gnoweb/ -run RegisteredWithoutPackages -count=1 -v
//	#   --- PASS: TestHTTPHandler_GetUserView_RegisteredWithoutPackages_QueriesRegistry
//	#   --- PASS: TestHTTPHandler_GetUserView_RegisteredWithoutPackages
//
//	# 2. delete the registry gate (handler_http.go:618-627) and rerun
//	sed -i '618,627d' gno.land/pkg/gnoweb/handler_http.go
//	go test ./gno.land/pkg/gnoweb/ -run RegisteredWithoutPackages -count=1 -v
//	#   --- PASS: TestHTTPHandler_GetUserView_RegisteredWithoutPackages
//	#   --- FAIL: TestHTTPHandler_GetUserView_RegisteredWithoutPackages_QueriesRegistry
//
//	# 3. same mutation, the integration routes: the admit cases survive it
//	go test ./gno.land/pkg/gnoweb/ -run TestRoutes -count=1 -v
//	#   --- PASS: TestRoutes/test_route_/u/moul001
//	#   --- PASS: TestRoutes/test_route_/u/zoo_ma123
//	#   --- FAIL: TestRoutes/test_route_/u/zzznotauser
//
//	git checkout -- gno.land/pkg/gnoweb/handler_http.go
//
// Toolchain: go1.25.9, the version go.mod pins.

package gnoweb_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
)

// stubClient, getUserPage and resolveNamePayload come from handler_http_test.go
// in this same package.
func TestHTTPHandler_GetUserView_RegisteredWithoutPackages_QueriesRegistry(t *testing.T) {
	t.Parallel()

	evalCalled := false
	client := &stubClient{
		listPathsFunc: func(context.Context, string, int) ([]string, error) { return nil, nil },
		evalFunc: func(_ context.Context, pkgPath, expr string) ([]byte, error) {
			evalCalled = true
			assert.Equal(t, "/r/sys/users", pkgPath)
			assert.Equal(t, `ResolveName("alice")`, expr)
			return resolveNamePayload(true), nil
		},
		realmFunc: func(context.Context, string, string) ([]byte, error) {
			return nil, gnoweb.ErrClientPackageNotFound
		},
	}

	rr := getUserPage(t, client, "/u/alice")

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Gnome alice")
	assert.True(t, evalCalled,
		"the page must be served because the registry said the name is taken, not by default")
}
