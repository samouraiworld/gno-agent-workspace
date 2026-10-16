// Pins that NewHTMLRenderer's "ignoring invalid IPFS gateway" warning never
// echoes the gateway value. Passes at 36b475912; fails once render.go:98 logs
// "gateway", cfg.IPFSGateway, which every existing test in the package misses.
// Run: cp into gno.land/pkg/gnoweb/, go test ./gno.land/pkg/gnoweb -run TestNewHTMLRendererIPFSGatewayLog -count=1

package gnoweb

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTMLRendererIPFSGatewayLog(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	cfg := NewDefaultRenderConfig()
	cfg.IPFSGateway = "https://user:s3cret@gw.example"
	NewHTMLRenderer(slog.New(slog.NewTextHandler(&logs, nil)), cfg, nil)
	assert.Contains(t, logs.String(), "ignoring invalid IPFS gateway")
	assert.NotContains(t, logs.String(), "s3cret")
}
