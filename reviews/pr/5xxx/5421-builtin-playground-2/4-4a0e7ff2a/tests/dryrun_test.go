package playground

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlerPlaygroundDryRun tests the POST /_/api/dryrun handler directly.
func TestHandlerPlaygroundDryRun(t *testing.T) {
	t.Parallel()

	h := New(Deps{
		Client:  &stubClient{simulateResult: &abci.ResponseDeliverTx{ResponseBase: abci.ResponseBase{Data: []byte("mock run")}}},
		Logger:  discardLogger(),
		Domain:  "gno.land",
		Remote:  "http://localhost:26657",
		ChainId: "test",
	})

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantResult string
		wantError  string
	}{
		{
			name:       "valid dry run",
			body:       `{"pkg_path":"r/mock/path","script":"package main\n\nfunc main() {}\n","address":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"}`,
			wantStatus: http.StatusOK,
			wantResult: "mock run",
		},
		{
			name:       "key name rejected",
			body:       `{"pkg_path":"r/mock/path","script":"package main\n","address":"mykey"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "address must be a bech32 address",
		},
		{
			name:       "empty script rejected",
			body:       `{"pkg_path":"r/mock/path","script":"","address":"g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "script is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/_/api/dryrun", strings.NewReader(tc.body))
			rr := httptest.NewRecorder()
			h.DryRunHandler().ServeHTTP(rr, req)
			assert.Equal(t, tc.wantStatus, rr.Code)

			var resp dryRunResponse
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
			assert.Equal(t, tc.wantResult, resp.Result)
			assert.Contains(t, resp.Error, tc.wantError)
		})
	}
}
