# Review: PR [#5964](https://github.com/gnolang/gno/pull/5964)
Event: REQUEST_CHANGES

## Body
Booted gnoweb in strict mode at 37b883fca against a local node and loaded the run and playground pages in headless Chromium. Both editors draw with zero CSP violations. The same binary built at c355059a1 logs the blocked stylesheet on both.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5964-codemirror-styles-strict-csp/1-37b883fca/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/cmd/gnoweb/main.go:358-362 [↗](../../../../../.worktrees/gno-review-5964/gno.land/cmd/gnoweb/main.go#L358-L362)
A `-remote` value containing `%` reaches `connect-src` corrupted: the header string is interpolated here and then [formatted again per request](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main.go#L386), so `%25eth0` comes out as `%!e(MISSING)th0`. The browser then blocks the `abci_query` calls the run and playground pages make. IPv6 zone identifiers and percent-encoded path segments both hit this.

<details><summary>repro</summary>

Asserts the remote survives into `connect-src`. Fails on this branch, passes on the base commit.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5964 -R gnolang/gno
cat > gno.land/cmd/gnoweb/csp_pct_test.go <<'EOF'
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSPRemoteWithPercent(t *testing.T) {
	const remote = "http://[fe80::1%25eth0]:26657"
	h := SecureHeadersMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), true, remote)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "http://example.com", nil))
	csp := rec.Result().Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self' "+remote+"/abci_query") {
		t.Fatalf("remote mangled in CSP: %s", csp)
	}
}
EOF
go test -run TestCSPRemoteWithPercent ./gno.land/cmd/gnoweb/
rm gno.land/cmd/gnoweb/csp_pct_test.go
```

```
--- FAIL: TestCSPRemoteWithPercent (0.00s)
    csp_pct_test.go:17: remote mangled in CSP: default-src 'self'; script-src 'self' https://sa.gno.services;
    style-src 'self' 'nonce-0nlI9kE0bfz5fMqhzD1Rpw=='; img-src …; font-src 'self';
    connect-src 'self' http://[fe80::1%!e(MISSING)th0]:26657/abci_query; form-action 'self'
FAIL	github.com/gnolang/gno/gno.land/cmd/gnoweb	0.037s
```
</details>

## gno.land/pkg/gnoweb/handler_http.go:234 [↗](../../../../../.worktrees/gno-review-5964/gno.land/pkg/gnoweb/handler_http.go#L234)
Missing test: nothing asserts the nonce reaches the rendered page. [`TestIndexLayout_CSPNonce`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/pkg/gnoweb/components/layout_test.go#L422) sets `HeadData.CSPNonce` by hand and [`TestSecureHeadersMiddlewareNonceMatchesContext`](https://github.com/gnolang/gno/blob/37b883fca/gno.land/cmd/gnoweb/main_test.go#L131) stops at the request context, so deleting this line leaves `go test ./gno.land/pkg/gnoweb/... ./gno.land/cmd/gnoweb/...` green while the editor loses its styles again.

<details><summary>test cases</summary>

Passes at 37b883fca, fails once the assignment is removed. Drop into `gno.land/pkg/gnoweb/handler_csp_nonce_test.go`.

```go
func TestHTTPHandler_CSPNoncePropagation(t *testing.T) {
	t.Parallel()

	mockPackage := &gnoweb.MockPackage{
		Domain: "example.com",
		Path:   "/r/mock/path",
		Files: map[string]string{
			"render.gno": `package main; func Render(path string) string { return "one more time" }`,
		},
		Functions: []*doc.JSONFunc{
			{Name: "Render", Params: []*doc.JSONField{{Name: "path", Type: "string"}}, Results: []*doc.JSONField{{Name: "", Type: "string"}}},
		},
	}

	cases := []struct {
		name  string
		nonce string
		want  string // "" means the meta tag must be absent
	}{
		{
			name:  "context nonce reaches the head meta tag",
			nonce: "Ml2rzjv6QqQEexAw32Pbeg==",
			want:  `<meta name="csp-nonce" content="Ml2rzjv6QqQEexAw32Pbeg==" />`,
		},
		{
			name:  "no context nonce omits the meta tag",
			nonce: "",
			want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(&testingLogger{t}, &slog.HandlerOptions{}))
			handler, err := gnoweb.NewHTTPHandler(logger, newTestHandlerConfig(t, gnoweb.NewMockClient(mockPackage)))
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodGet, "/r/mock/path", nil)
			require.NoError(t, err)
			if tc.nonce != "" {
				req = req.WithContext(gnoweb.WithCSPNonce(req.Context(), tc.nonce))
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			require.Equal(t, http.StatusOK, rr.Code)

			if tc.want == "" {
				assert.NotContains(t, rr.Body.String(), `name="csp-nonce"`)
			} else {
				assert.Contains(t, rr.Body.String(), tc.want)
			}
		})
	}
}
```
</details>
