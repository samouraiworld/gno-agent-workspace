// Equivalence proof for the getRetry rewrite proposed on gnolang/gno#6194
// (misc/gnopreview/crawl.go:412, 15 lines -> 11 with named results).
//
// It carries the ORIGINAL getRetry body as origGetRetry and asserts the
// package's own getRetry agrees with it on body, code, err and on the number of
// requests actually made, over the four paths the function has: an immediate
// 200, one 429 then 200, three 429s, and a transport error. The 429 paths sleep
// 2s and 4s, so the file runs in ~16s; the package's own suite never enters
// them, which is why vet plus identical return expressions is not enough here.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_judge51_test.go
//	cd misc/gnopreview && go test -run TestJudge51GetRetryEquivalent -v .   # green: baseline
//	# then apply the rewrite (b2-refactor-crawl.patch) and re-run the same line.
//
// Mutation that must turn it red (proves the harness is not vacuous): change
// the rewrite's `break` to `return`-after-`continue`, or drop one retry.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// origGetRetry is crawl.go's getRetry verbatim at ecf7af0f2, before the rewrite.
func origGetRetry(c *Crawler, p string) (string, int, error) {
	var body string
	var code int
	var err error
	for attempt := range 3 {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		body, code, err = c.get(p)
		if err != nil || code != http.StatusTooManyRequests {
			return body, code, err
		}
	}
	return body, code, err
}

func TestJudge51GetRetryEquivalent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		codes []int // status per request; the last one repeats
	}{
		{"200 first try", []int{200}},
		{"429 then 200", []int{429, 200}},
		{"429 throughout", []int{429}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// One server per implementation, so the request counters and the
			// per-attempt status sequence are not shared.
			serve := func(hits *int) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					i := *hits
					*hits++
					if i >= len(tc.codes) {
						i = len(tc.codes) - 1
					}
					w.WriteHeader(tc.codes[i])
					w.Write([]byte("body"))
				}))
			}
			var newHits, oldHits int
			sNew, sOld := serve(&newHits), serve(&oldHits)
			defer sNew.Close()
			defer sOld.Close()

			gotBody, gotCode, gotErr := (&Crawler{Base: sNew.URL}).getRetry("/r/x/y")
			wantBody, wantCode, wantErr := origGetRetry(&Crawler{Base: sOld.URL}, "/r/x/y")
			if gotBody != wantBody || gotCode != wantCode || (gotErr == nil) != (wantErr == nil) {
				t.Errorf("getRetry = (%q, %d, %v); original = (%q, %d, %v)",
					gotBody, gotCode, gotErr, wantBody, wantCode, wantErr)
			}
			if newHits != oldHits {
				t.Errorf("getRetry made %d request(s); original made %d", newHits, oldHits)
			}
			t.Logf("code=%d requests=%d", gotCode, newHits)
		})
	}

	// Transport error: the server is closed before the call, so c.get fails and
	// both forms must give up after one attempt.
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := s.URL
	s.Close()
	gotBody, gotCode, gotErr := (&Crawler{Base: url}).getRetry("/r/x/y")
	wantBody, wantCode, wantErr := origGetRetry(&Crawler{Base: url}, "/r/x/y")
	if gotBody != wantBody || gotCode != wantCode || (gotErr == nil) != (wantErr == nil) {
		t.Errorf("transport error: getRetry = (%q, %d, %v); original = (%q, %d, %v)",
			gotBody, gotCode, gotErr, wantBody, wantCode, wantErr)
	}
	if gotErr == nil {
		t.Fatal("want a transport error from the closed server")
	}
}
