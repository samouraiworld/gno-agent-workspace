// Probe of Handler.Handle's JSON budget (jsonTimeout = 3s, handler.go) against
// indexer.Client's breaker at gnolang/gno#6231 head dd40ef3dc. The omnibar's
// own 3s deadline expires before the indexer client's 4s timeout, and the
// client records that caller deadline as an indexer failure. Three omnibar
// lookups at once against a healthy indexer answering in 3.3s open the breaker, and
// the results page, whose 6s budget would have waited for the answer, then
// reports "indexer unavailable" for 30s.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6231/head && git checkout --detach dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp b3-lines-reach-catalog-omnibar-trips-breaker_test.go gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_breaker_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3OmnibarTripsBreaker' -count=1 -v
//
// Expected at dd40ef3dc: FAIL, the page group reads "indexer unavailable" and
// the indexer saw 3 queries, not 4. Takes about 4s.
package omnisearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/indexer"
)

func TestB3OmnibarTripsBreaker(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Query string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(req.Query, "latestBlockHeight") {
			_, _ = w.Write([]byte(`{"data":{"latestBlockHeight":7}}`))
			return
		}
		hits.Add(1)
		time.Sleep(3300 * time.Millisecond) // healthy, under the client's own 4s timeout
		_, _ = w.Write([]byte(`{"data":{"getTransactions":[]}}`))
	}))
	defer srv.Close()

	h := newHandlerWithDir(t, newDiscoveryClient(), newDiscoveryDir(), indexer.New(srv.URL, ""))
	q := url.QueryEscape("account:g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5")

	// Three visitors typing at once. Run one at a time, each search is
	// followed by indexerStatus's tip fetch, whose success resets the
	// failure count: that sequential shape does not trip the breaker.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.Handle(context.Background(), w, httptest.NewRequest(http.MethodGet, "/", nil),
				parseURL(t, "/$search&q="+q+"&json"))
			var resp jsonResponse
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if len(resp.Groups) > 0 {
				t.Logf("omnibar %d: group error %q", i+1, resp.Groups[0].Error)
			}
		}()
	}
	wg.Wait()

	w := httptest.NewRecorder()
	_, view := h.Handle(context.Background(), w, httptest.NewRequest(http.MethodGet, "/", nil),
		parseURL(t, "/$search&q="+q))
	var body strings.Builder
	if view != nil {
		_ = view.Render(&body)
	}
	unavailable := strings.Contains(body.String(), "indexer unavailable")
	t.Logf("results page: indexer queries seen=%d, says unavailable=%v", hits.Load(), unavailable)
	if unavailable {
		t.Errorf("results page reports the indexer unavailable after 3 omnibar lookups hit their own 3s deadline; the indexer answers every query in 3.3s")
	}
}
