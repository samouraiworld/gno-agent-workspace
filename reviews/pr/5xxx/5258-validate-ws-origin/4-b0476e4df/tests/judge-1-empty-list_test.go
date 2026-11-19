// Asserts a node with cors_allowed_origins = [] keeps /websocket same-origin:
// a cross-site Origin gets 403, the node's own host and no Origin get 101.
// Passes at b0476e4df; fails once startRPC maps an empty list to ["*"].
// Run: overlay into tm2/pkg/bft/node, go test -run TestJudgeNodeRPCWebsocketOriginEmptyList.
package node

import (
	"net/http"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfg "github.com/gnolang/gno/tm2/pkg/bft/config"
	"github.com/gnolang/gno/tm2/pkg/events"
	"github.com/gnolang/gno/tm2/pkg/log"
)

func TestJudgeNodeRPCWebsocketOriginEmptyList(t *testing.T) {
	config, genesisFile := cfg.ResetTestRoot("node_rpc_ws_origin_empty_test")
	defer os.RemoveAll(config.RootDir)
	config.RPC.ListenAddress = "tcp://127.0.0.1:0"
	config.RPC.CORSAllowedOrigins = []string{}

	n, err := DefaultNewNode(config, genesisFile, events.NewEventSwitch(), log.NewNoopLogger())
	require.NoError(t, err)
	require.NoError(t, n.Start())
	defer n.Stop()

	addr := n.rpcListeners[0].Addr().String()
	for origin, status := range map[string]int{
		"https://evil.test": http.StatusForbidden,
		"http://" + addr:    http.StatusSwitchingProtocols,
		"":                  http.StatusSwitchingProtocols,
	} {
		header := http.Header{}
		if origin != "" {
			header.Set("Origin", origin)
		}
		c, resp, _ := websocket.DefaultDialer.Dial("ws://"+addr+"/websocket", header)
		require.NotNil(t, resp, "origin %q", origin)
		assert.Equal(t, status, resp.StatusCode, "origin %q", origin)
		if c != nil {
			c.Close()
		}
	}
}
