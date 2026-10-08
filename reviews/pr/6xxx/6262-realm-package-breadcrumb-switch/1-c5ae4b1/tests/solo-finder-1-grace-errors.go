// solo-finder-1: the 300 ms grace opens when the page collects, after its own
// queries, and a failed lookup is never cached, so a lookup that fails slower
// than the grace adds 300 ms and one ListPaths to every page view.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6262/head && git checkout c5ae4b13a8883303a382d6b1ae1c61b929fd048c
//	cp <this file> gno.land/pkg/gnoweb/solo_finder1_grace_test.go
//	cd gno.land/pkg/gnoweb && go test -count=1 -run TestSoloFinder1_GraceAndErrors -v .
//
// Observed at c5ae4b13a (go1.25.9), Realm answering at once, ListPaths 400 ms:
//
//	slow empty answer: request 0: 300ms, requests 1-3: 0s; ListPaths calls over 4 requests: 1
//	slow error:        requests 0-3: 300ms each;        ListPaths calls over 4 requests: 4
package gnoweb_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSoloFinder1_GraceAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{{"slow empty answer", nil}, {"slow error", errors.New("node busy")}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			client := &stubClient{
				realmFunc: func(context.Context, string, string) ([]byte, error) { return []byte("hello"), nil },
				listPathsFunc: func(ctx context.Context, _ string, _ int) ([]string, error) {
					calls.Add(1)
					select {
					case <-time.After(400 * time.Millisecond):
						return nil, tc.err
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
			}
			h := newCounterpartHandler(t, client)
			for i := range 4 {
				start := time.Now()
				serve(h, httptest.NewRequest(http.MethodGet, "/r/alice/golf/game", nil))
				t.Logf("request %d: %v", i, time.Since(start).Round(10*time.Millisecond))
				time.Sleep(500 * time.Millisecond) // let the lookup land
			}
			t.Logf("ListPaths calls over 4 requests: %d", calls.Load())
		})
	}
}
