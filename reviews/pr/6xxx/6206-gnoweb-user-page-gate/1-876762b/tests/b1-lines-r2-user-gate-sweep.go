package gnoweb

// PR 6206, bundle gno.land/pkg/gnoweb, angle "lines".
//
// What the new /u/<name> gate in GetUserView (handler_http.go:603) actually
// serves, swept over the whole case space rather than the four stub cases the
// diff adds. Four questions:
//
//   1. reserved namespaces: do @std / @stdlibs, which vm/qpaths special-cases
//      onto the stdlib key space (keeper.go:1699), slip through the gate?
//   2. does any namespace that holds packages lose its page?
//   3. does any name registered in genesis lose its page?
//   4. which namespaces render as people ("Gnome <name>") though nobody
//      registered them?
//
// Repro from a plain clone (go1.25.9, the version go.mod pins):
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//   git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
//   cp b1-lines-r2-user-gate-sweep.go gno.land/pkg/gnoweb/zz_gate_sweep_test.go
//   go test ./gno.land/pkg/gnoweb/ -run TestUserGateSweep -v 2>&1 | grep '^RESULT'
//
// For the merge-base column, repeat with `git checkout 7916d1dd65f326efe46b5ad105412ce028768c4d`.
//
// Measured at 876762bdf (head):
//
//   RESULT reserved   /u/std=404 /u/stdlibs=404
//   RESULT trailing   /u/moul001=200 /u/moul001/=404 /u/g1manfred…dlf5=200 /u/g1manfred…dlf5/=404
//   RESULT namespaces=20 lost=0
//   RESULT genesisusers=6 lost=0
//   RESULT gnomes-without-registration: aeddi archive demo devrels docs gnoland
//          gnops gov jaekwon jefft0 jeronimoalbi leon mason moul nt onbloc
//          samcrew sunspirit sys tests
//
// Measured at 7916d1dd6 (merge base), same file:
//
//   RESULT trailing   /u/moul001=200 /u/moul001/=404 /u/g1manfred…dlf5=200 /u/g1manfred…dlf5/=404
//   /u/zzznotauser=200        <- the only status the diff moves: 200 -> 404
//
// So: the trailing-slash 404 is a router-level behaviour that predates the
// diff, the stdlib special case does not leak (its paths fail weburl.Parse and
// the name then falls through to ResolveName, which says no), every namespace
// with packages and every genesis-registered name keeps its page, and the
// reserved namespaces sys/demo/gnoland/tests are served as user profiles.
//
// Fifth check, a mutation rather than a run, on the head worktree:
//
//   sed -i 's/^const maxUsernameLen = 64$/const maxUsernameLen = 32/' \
//       gno.land/pkg/gnoweb/handler_http.go
//   go test ./gno.land/pkg/gnoweb/ -count=1
//   # ok  github.com/gnolang/gno/gno.land/pkg/gnoweb  7.200s
//
//   sed -i 's/^const maxUsernameLen = 64$/const maxUsernameLen = 8/' ...
//   go test ./gno.land/pkg/gnoweb/ -count=1
//   # --- FAIL: TestRoutes/test_route_/u/zoo_ma123
//
// The cap is pinned from below at 9 (the length of "zoo_ma123") by the new
// app_test route and not at all from above: any value in [9,64] keeps the
// package green, so the gate's 64 and the registry's 64
// (examples/gno.land/r/sys/users/errors.gno:19) can drift apart unnoticed.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	"github.com/gnolang/gno/tm2/pkg/log"
)

// the six RegisterUser calls in gno.land/genesis/genesis_txs.jsonl, which
// integration.LoadDefaultGenesisTXsFile feeds to the shared test node:
//
//	grep -o '"func":"RegisterUser","args":\["[^"]*"' gno.land/genesis/genesis_txs.jsonl
var genesisUsers = []string{
	"gfanton123", "zoo_ma123", "moul001",
	"piupiu123", "anarcher123", "ideamour123",
}

const manfred = "g1manfred47kzduec920z88wfr64ylksmdcedlf5"

func TestUserGateSweep(t *testing.T) {
	cfg := NewDefaultAppConfig()
	cfg.NodeRemote = sharedNodeRemote(t)

	router, err := NewRouter(log.NewTestingLogger(t), cfg)
	if err != nil {
		t.Fatal(err)
	}

	get := func(path string) (int, string) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		return rr.Code, rr.Body.String()
	}

	// 1. reserved namespaces vm/qpaths special-cases
	stdCode, _ := get("/u/std")
	stdlibsCode, _ := get("/u/stdlibs")
	fmt.Printf("RESULT reserved   /u/std=%d /u/stdlibs=%d\n", stdCode, stdlibsCode)

	// 2. trailing slash, head and merge base alike
	var parts []string
	for _, p := range []string{"/u/moul001", "/u/moul001/", "/u/" + manfred, "/u/" + manfred + "/"} {
		code, _ := get(p)
		parts = append(parts, fmt.Sprintf("%s=%d", p, code))
	}
	fmt.Printf("RESULT trailing   %s\n", strings.Join(parts, " "))

	// 3. every namespace the examples tree deploys
	names := examplesNamespaces(t)
	var lost, gnomes []string
	for _, n := range names {
		code, body := get("/u/" + n)
		if code != http.StatusOK {
			lost = append(lost, fmt.Sprintf("%s(%d)", n, code))
			continue
		}
		if strings.Contains(body, "Gnome "+n) && !slicesContains(genesisUsers, n) {
			gnomes = append(gnomes, n)
		}
	}
	fmt.Printf("RESULT namespaces=%d lost=%d %s\n", len(names), len(lost), strings.Join(lost, " "))

	// 4. every name genesis registers
	var lostUsers []string
	for _, n := range genesisUsers {
		if code, _ := get("/u/" + n); code != http.StatusOK {
			lostUsers = append(lostUsers, fmt.Sprintf("%s(%d)", n, code))
		}
	}
	fmt.Printf("RESULT genesisusers=%d lost=%d %s\n",
		len(genesisUsers), len(lostUsers), strings.Join(lostUsers, " "))

	fmt.Printf("RESULT gnomes-without-registration: %s\n", strings.Join(gnomes, " "))
}

func examplesNamespaces(t *testing.T) []string {
	t.Helper()

	seen := map[string]bool{}
	for _, kind := range []string{"r", "p"} {
		ents, err := os.ReadDir(filepath.Join(gnoenv.RootDir(), "examples", "gno.land", kind))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range ents {
			if e.IsDir() {
				seen[e.Name()] = true
			}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func slicesContains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
