// b7-catalog-r2-waitready-realm-probe.go — gnolang/gno PR 6194, head ecf7af0f2.
//
// Two checks from the invariant-catalog walk over misc/gnopreview/plan.go:
//
//   TestWaitReadyProbeIsOneRealm  gnoweb answers 200 on "/" for the whole run,
//     yet render()'s readiness gate is waitReady(c.Base, urlOf(plan.Realms[0]))
//     (main.go:131), so one realm that does not answer 200 burns cfg.timeout and
//     aborts the entire preview with "gnoweb not ready after <timeout>".
//
//   TestBuildPlanDeterministic  catalog class "Determinism": BuildPlan ranges
//     Go maps in LoadPkgs, byDir and dependents; five runs must marshal
//     identically. They do — the class is clean, and the log line records which
//     realm the readiness probe actually lands on.
//
// Repro from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//   cp <this file> misc/gnopreview/zz_b7_catalog_test.go
//   cd misc/gnopreview && go test -run 'TestWaitReadyProbeIsOneRealm|TestBuildPlanDeterministic' -v .
//
// Observed (go1.25.9):
//
//   === RUN   TestWaitReadyProbeIsOneRealm
//       probe=/r/demo/broken err=gnoweb not ready after 3s (server healthy, 4 requests served)
//   --- PASS: TestWaitReadyProbeIsOneRealm (3.00s)
//   === RUN   TestBuildPlanDeterministic
//       seed=examples/gno.land/p/nt/avl/pager/v0/pager.gno realms=8 dropped=0 changed_realms=0 gnoweb=true
//       Realms[0]=gno.land/r/demo/defi/grc20factory probe=/r/demo/defi/grc20factory
//   --- PASS: TestBuildPlanDeterministic (0.04s)
//
// Realms[0] is neither a changed realm nor one of the four curated
// gnowebSeedRealms: it is whatever sorts first in the affected set.

package main

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 1. The readiness gate is bound to plan.Realms[0]: one realm that does not
// answer 200 aborts the whole render, even while gnoweb itself is up.
func TestWaitReadyProbeIsOneRealm(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/r/demo/broken" {
			http.Error(w, "runtime error: index out of range", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gnoweb is up"))
	}))
	defer srv.Close()
	died := make(chan error, 1)

	// gnoweb answers 200 on its own root the whole time.
	if err := waitReady(srv.URL, "/", 3*time.Second, died); err != nil {
		t.Fatalf("root probe: %v", err)
	}
	// But render() probes urlOf(plan.Realms[0]) instead, and that realm 500s.
	err := waitReady(srv.URL, urlOf("gno.land/r/demo/broken"), 3*time.Second, died)
	if err == nil {
		t.Fatal("expected the realm-bound probe to fail")
	}
	t.Logf("probe=%s err=%v (server healthy, %d requests served)", urlOf("gno.land/r/demo/broken"), err, hits)
}

// 2. Catalog, determinism: BuildPlan's output must not depend on Go map
// iteration order.
func TestBuildPlanDeterministic(t *testing.T) {
	root := "../.."
	var seed string
	err := filepath.WalkDir(filepath.Join(root, "examples/gno.land/p/nt/avl"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || seed != "" {
			return err
		}
		if strings.HasSuffix(p, ".gno") && !strings.HasSuffix(p, "_test.gno") && !strings.HasSuffix(p, "_filetest.gno") {
			seed = filepath.ToSlash(strings.TrimPrefix(p, root+"/"))
		}
		return nil
	})
	if err != nil || seed == "" {
		t.Skipf("no seed file found: %v", err)
	}
	changed := []string{seed, "gno.land/pkg/gnoweb/app.go"}
	var first string
	for i := 0; i < 5; i++ {
		p, err := BuildPlan(root, changed, 25)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(p)
		if i == 0 {
			first = string(b)
			t.Logf("seed=%s realms=%d dropped=%d changed_realms=%d gnoweb=%v",
				seed, len(p.Realms), p.Dropped, len(p.ChangedRealms), p.Gnoweb)
			if len(p.Realms) > 0 {
				t.Logf("Realms[0]=%s probe=%s", p.Realms[0], urlOf(p.Realms[0]))
			}
			continue
		}
		if string(b) != first {
			t.Fatalf("run %d differs from run 0", i)
		}
	}
}
