/* Run: from a gno checkout:
gh pr checkout 5980 -R gnolang/gno && git checkout c9aaed5c8
go get github.com/certusone/yubihsm-go@v0.3.0
curl -fsSL -o tm2/pkg/bft/privval/signer/yubihsm/connector_url_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5980-yubihsm2-backed-signer/1-c9aaed5c8/tests/connector_url_test.go
go test -v -run 'TestConfig_ConnectorURLForm' ./tm2/pkg/bft/privval/signer/yubihsm/
rm tm2/pkg/bft/privval/signer/yubihsm/connector_url_test.go
git checkout go.mod go.sum
*/

// connector.NewHTTPConnector hardcodes "http://"+URL+"/connector/api", so
// ConnectorURL must be a bare host:port. At c9aaed5c8 the field doc and the
// toml comment give "http://127.0.0.1:12345" and ValidateBasic accepts it,
// which builds "http://http//127.0.0.1:12345/connector/api" at session open.
// A fixed ValidateBasic rejects any scheme-prefixed value at config load.

package yubihsm

import "testing"

func TestConfig_ConnectorURLForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		connectorURL string
		wantErr      bool
	}{
		{"bare host and port", "127.0.0.1:12345", false},
		{"hostname and port", "yubihsm-connector.internal:12345", false},
		{"http scheme prefix", "http://127.0.0.1:12345", true},
		{"https scheme prefix", "https://127.0.0.1:12345", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{ConnectorURL: tt.connectorURL, AuthKeyID: 1, KeyID: 2}

			err := cfg.ValidateBasic()
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateBasic accepted %q; the connector prepends http:// itself", tt.connectorURL)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateBasic rejected %q: %v", tt.connectorURL, err)
			}
		})
	}
}
