// Logs the upgrade status for a client sending its own target as Origin
// (websocket-client's default), under an empty list and a narrowed one.
// At b0476e4df and at dd05df75b: empty list 101, narrowed list 403.
// Run: overlay into tm2/pkg/bft/rpc/lib/server, go test -v -run TestJudgeOwnHostOrigin.
package rpcserver_test

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestJudgeOwnHostOrigin(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed []string
	}{
		{"empty list", []string{}},
		{"narrowed list", []string{"https://gno.land"}},
	} {
		s := newWSServerWithOrigins(tc.allowed)
		addr := s.Listener.Addr().String()
		c, resp, _ := websocket.DefaultDialer.Dial("ws://"+addr+"/websocket", http.Header{"Origin": {"http://" + addr}})
		require.NotNil(t, resp, tc.name)
		t.Logf("%s: Origin http://<own host> -> %d", tc.name, resp.StatusCode)
		if c != nil {
			c.Close()
		}
		s.Close()
	}
}
