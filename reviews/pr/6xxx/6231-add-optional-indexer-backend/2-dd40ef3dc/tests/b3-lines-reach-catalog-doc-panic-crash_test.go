// Probe of Handler.doc (gno.land/pkg/gnoweb/feature/omnisearch/feature.go) at
// gnolang/gno#6231 head dd40ef3dc: a panic inside ClientAdapter.Doc runs in the
// singleflight.DoChan goroutine, which no recover reaches, so it ends the whole
// gnoweb process instead of failing one request. net/http recovers a panic on
// the request goroutine only; the control subtest shows that baseline.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6231/head && git checkout --detach dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp b3-lines-reach-catalog-doc-panic-crash_test.go gno.land/pkg/gnoweb/feature/omnisearch/zz_b3_docpanic_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestB3DocPanic' -count=1 -v
//
// Expected at dd40ef3dc: control PASS (server survives a request-goroutine
// panic), func_selector FAIL (child exits 2 with "panic: boom in Doc" from
// singleflight). A recover in the DoChan function turns it green.
package omnisearch

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
	"github.com/gnolang/gno/gnovm/pkg/doc"
)

type b3PanicDocClient struct{}

func (b3PanicDocClient) Realm(context.Context, string, string) ([]byte, error) { return nil, nil }
func (b3PanicDocClient) Doc(context.Context, string, int64) (*doc.JSONDocumentation, error) {
	panic("boom in Doc")
}
func (b3PanicDocClient) ListFiles(context.Context, string, int64) ([]string, error) { return nil, nil }

// b3Child serves the omnisearch handler behind a real net/http server, the
// way gnoweb does, sends one request and prints SURVIVED if the process lives.
func b3Child(t *testing.T, mode string) {
	h := New(Deps{Client: b3PanicDocClient{}, Directory: newDiscoveryDir(), Domain: "gno.land"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == "control" {
			panic("boom on the request goroutine")
		}
		u, err := weburl.ParseFromURL(r.URL)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		status, view := h.Handle(r.Context(), w, r, u)
		if view == nil {
			return
		}
		w.WriteHeader(status)
		_ = view.Render(w)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/r/alice/blog$search&q=func:Render")
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	// Give a goroutine panic time to land.
	time.Sleep(300 * time.Millisecond)
	os.Stdout.WriteString("SURVIVED\n")
}

func TestB3DocPanicChild(t *testing.T) {
	mode := os.Getenv("B3_CHILD")
	if mode == "" {
		t.Skip("child only")
	}
	b3Child(t, mode)
}

func TestB3DocPanic(t *testing.T) {
	for _, mode := range []string{"control", "func_selector"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestB3DocPanicChild$", "-test.v")
			cmd.Env = append(os.Environ(), "B3_CHILD="+mode)
			out, err := cmd.CombinedOutput()
			survived := strings.Contains(string(out), "SURVIVED")
			first := string(out)
			if i := strings.Index(first, "goroutine "); i > 0 {
				first = first[:i]
			}
			t.Logf("mode=%s exit=%v survived=%v\n%s", mode, err, survived, strings.TrimSpace(first))
			if !survived || err != nil {
				t.Errorf("gnoweb process died serving one search request (mode %s)", mode)
			}
		})
	}
}
