// b1-claims-r2-address-profile — PR 6206, head 876762bdf2ea6635b27e2b0a42f26f9fdc54af24.
//
// Claim under test (gno.land/pkg/gnoweb/handler_http.go:600-602):
//   "It serves a page only for an address, a namespace holding packages, or a
//    name r/sys/users resolves; anything else would be a fabricated profile."
//
// Rule 1 (handler_http.go:606-607) accepts any checksum-valid g1 address with
// no further check, so every address in the space — no account, no package, no
// registered name — still renders a 200 profile marked "index, follow".
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//   git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//   cp <this file> gno.land/pkg/gnoweb/zz_claims_probe_test.go
//   go test ./gno.land/pkg/gnoweb -run TestClaimProbeAddr -v
//
// Observed (go1.25.9, 6.5s, in-memory node from TestMain):
//   PROBE /u/g1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqluuxe status=200 gnome=true robots-index=true
//   PROBE /u/g1qqrsu9guyv4rzwplgex4gkmzd9c8wl59pcdqg0 status=200 gnome=true robots-index=true
//
// Same probe file, earlier run, for the PR-body claim that $-suffixed /u/ views
// already 404 (that claim holds):
//   PROBE /u/zzznotauser        status=404   PROBE /u/zzznotauser$source status=404
//   PROBE /u/zzznotauser$help   status=404   PROBE /u/zzznotauser$state  status=404
//   PROBE /u/moul001$source     status=404   PROBE /u/moul001            status=200

package gnoweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestClaimProbeAddr(t *testing.T) {
	logger := log.NewTestingLogger(t)
	cfg := NewDefaultAppConfig()
	cfg.NodeRemote = sharedNodeRemote(t)
	router, err := NewRouter(logger, cfg)
	require.NoError(t, err)

	zero := crypto.Address{}.String()
	var b [20]byte
	for i := range b {
		b[i] = byte(i * 7)
	}
	unused := crypto.Address(b).String()

	for _, p := range []string{"/u/" + zero, "/u/" + unused} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		body := rr.Body.String()
		t.Logf("PROBE %s status=%d gnome=%v robots-index=%v", p, rr.Code,
			strings.Contains(body, "Gnome"),
			strings.Contains(body, `content="index, follow"`))
	}
}
