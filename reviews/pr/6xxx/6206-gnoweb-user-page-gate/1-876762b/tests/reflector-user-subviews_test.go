package gnoweb

// Reflector probe for PR 6206. GetPackageView dispatches $help, $source,
// IsFile and IsDir before its `if gnourl.IsUser()` branch (handler_http.go
// 430-450), so every one of those forms of /u/<name> reaches the chain
// without passing the new gate. The ADR asserts they "already 404 on chain
// for a name that owns nothing" and runs nothing. This prints the status and
// whether the body carries the profile heading for each form, against the
// shared in-memory node app_test.go boots.
//
// Copy into gno.land/pkg/gnoweb/ and run:
//   go test ./gno.land/pkg/gnoweb/ -run TestReflectorUserSubviewsBypassGate -v

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestReflectorUserSubviewsBypassGate(t *testing.T) {
	remoteAddr := sharedNodeRemote(t)

	cfg := NewDefaultAppConfig()
	cfg.NodeRemote = remoteAddr
	maps.Copy(cfg.Aliases, map[string]AliasTarget{})

	router, err := NewRouter(log.NewTestingLogger(t), cfg)
	require.NoError(t, err)

	for _, p := range []string{
		"/u/zzznotauser",
		"/u/zzznotauser$source",
		"/u/zzznotauser$help",
		"/u/zzznotauser/",
		"/u/zzznotauser$state",
		"/u/mou1",
		"/u/mou1$source",
		"/u/docs",
		"/docs",
	} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		body := rr.Body.String()
		fmt.Printf("PROBE %-26s status=%d gnome=%v notfound=%v bytes=%d\n",
			p, rr.Code,
			strings.Contains(body, "Gnome"),
			strings.Contains(body, "user not found"),
			len(body))
	}
}
