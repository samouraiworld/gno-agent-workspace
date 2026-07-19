/* Run: from a gno checkout:
gh pr checkout 5964 -R gnolang/gno && git checkout 37b883fca
curl -fsSL -o gno.land/pkg/gnoweb/handler_csp_nonce_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5964-gnoweb-csp-nonce/1-37b883fca/tests/handler_csp_nonce_test.go
go test -v -run 'TestHTTPHandler_CSPNoncePropagation' ./gno.land/pkg/gnoweb/
rm gno.land/pkg/gnoweb/handler_csp_nonce_test.go
*/

// The nonce reaches the page through a single assignment in HTTPHandler.Get:
// HeadData.CSPNonce = CSPNonceFromContext(r.Context()). At 37b883fca deleting
// that line leaves every gno.land/pkg/gnoweb and gno.land/cmd/gnoweb test green,
// so the editor silently loses its styles again. This test fails without the line.

package gnoweb_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/gnovm/pkg/doc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
