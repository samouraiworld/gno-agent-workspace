// Repro for gnolang/gno#6231 at dd40ef3dc, gno.land/pkg/gnoweb/indexer/queries.go recent().
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/indexer/zz_window_repro_test.go
//	go test ./gno.land/pkg/gnoweb/indexer/ -run 'TestZZ' -v -count=1
//
// TestZZDeploysMissesAnythingPastTheFourthWindow: tip 2,000,000, one deploy at
// height 5. Deploys answers no rows and a nil error after four windows, so the
// page shows an empty "Deploy history" with no notice.
//
// TestZZThreeSlowScansOpenTheBreaker: three concurrent Deploys whose second
// window outlives the caller's deadline record three failures; the next,
// unrelated RecentByAddress gets ErrUnavailable with no request sent.
//
// TestZZImportersTextMatchesALongerPath: the like regex built from the chain
// path, evaluated the way tx-indexer evaluates `like` (regexp.MatchString),
// matches a body importing gno.land/r/demo/foobar and a sub-package gnomod.toml.
package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestZZDeploysMissesAnythingPastTheFourthWindow(t *testing.T) {
	var lowest = -1
	var bands int
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req gqlRequest
		_ = json.Unmarshal(body, &req)
		if strings.Contains(req.Query, "latestBlockHeight") {
			respond(w, `{"data":{"latestBlockHeight":2000000}}`)
			return
		}
		bands++
		b := parseBounds(t, req.Query)
		lowest = b[0]
		// The only deploy of the package sits at height 5.
		if b[0] < 5 && b[1] > 5 {
			respond(w, `{"data":{"getTransactions":[{"hash":"genesis-deploy","block_height":5}]}}`)
			return
		}
		respond(w, `{"data":{"getTransactions":[]}}`)
	})

	txs, err := c.Deploys(context.Background(), "gno.land/r/demo/boards", 20)
	t.Logf("bands queried = %d, lowest gt bound = %d, rows = %d, err = %v", bands, lowest, len(txs), err)
	if err == nil && len(txs) == 0 {
		t.Errorf("Deploys reported success with no rows; the deploy at height 5 was never asked for (lowest band gt=%d)", lowest)
	}
}

func TestZZThreeSlowScansOpenTheBreaker(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req gqlRequest
		_ = json.Unmarshal(body, &req)
		if strings.Contains(req.Query, "latestBlockHeight") {
			respond(w, `{"data":{"latestBlockHeight":2000000}}`)
			return
		}
		b := parseBounds(t, req.Query)
		if b[1]-b[0] > 2001 {
			// A wide window costs the indexer time; the narrow first one is cheap.
			select {
			case <-r.Context().Done():
			case <-time.After(3 * time.Second):
			}
			return
		}
		respond(w, `{"data":{"getTransactions":[]}}`)
	})

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			defer cancel()
			_, errs[i] = c.Deploys(ctx, "gno.land/r/demo/quiet", 20)
		}()
	}
	wg.Wait()
	t.Logf("three deploys scans: %v", errs)

	_, err := c.RecentByAddress(context.Background(), "g1someoneelse", 20)
	t.Logf("next, unrelated account: search: err = %v", err)
	if errors.Is(err, ErrUnavailable) {
		t.Errorf("three slow scans opened the shared breaker: an unrelated search got %v", err)
	}
}

func TestZZImportersTextMatchesALongerPath(t *testing.T) {
	q := captureTxQuery(t, func(c *Client) {
		_, _ = c.SourceContains(context.Background(), "gno.land/r/demo/foo", "", 20)
	})
	likes := likeValues(t, q)
	if len(likes) != 1 {
		t.Fatalf("like filters = %q", likes)
	}
	for _, body := range []string{
		"import \"gno.land/r/demo/foobar\"",
		"module = \"gno.land/r/demo/foo/sub\"",
	} {
		// tx-indexer: matched, err := regexp.MatchString(*f.Like, *val)
		matched, _ := regexp.MatchString(likes[0], body)
		t.Logf("like %q on %q: %v", likes[0], body, matched)
		if matched {
			t.Errorf("importers of gno.land/r/demo/foo matches a body that never names it: %q", body)
		}
	}
}
