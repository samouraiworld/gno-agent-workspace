// Probes of gno.land/pkg/gnoweb/indexer/client.go at gnolang/gno#6231 head dd40ef3dc.
//
// Repro from a plain clone of gnolang/gno:
//
//	git fetch origin pull/6231/head && git checkout --detach dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp b5-lines-reach-catalog-breaker_test.go gno.land/pkg/gnoweb/indexer/b5_breaker_probe_test.go
//	go test ./gno.land/pkg/gnoweb/indexer/ -run 'TestB5' -count=1 -v
//
// Each test fails (t.Errorf) when the defect it names is present, and logs what
// the client returned either way.
package indexer

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The JSON path gives a request 3 s (omnisearch jsonTimeout), below the
// client's own 4 s timeout, so a query the indexer answers in 3.5 s fails on
// gnoweb's own deadline. Query records that as an indexer failure. Scaled
// down here: the caller gives up at 50 ms, the indexer answers in 300 ms.
func TestB5CallerDeadlineOpensBreaker(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		respond(w, `{"data":{"latestBlockHeight":5}}`)
	})

	var out struct {
		H int `json:"latestBlockHeight"`
	}
	for i := range breakerThreshold {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		err := c.Query(ctx, `{ latestBlockHeight }`, &out)
		cancel()
		t.Logf("caller deadline %d: %v", i+1, err)
	}

	err := c.Query(context.Background(), `{ latestBlockHeight }`, &out)
	t.Logf("next caller, no deadline: err=%v height=%d", err, out.H)
	if errors.Is(err, ErrUnavailable) {
		t.Errorf("an indexer that answers every query is skipped for %s after %d callers' own deadlines expired",
			breakerCooldown, breakerThreshold)
	}
}

// ErrTooLarge (the indexer's element cap) is exempt from the breaker; the
// client's own 8 MiB cap, ErrResponseTooLarge, is not. Both are a property of
// the result set the query asked for, not of the indexer's health.
func TestB5OversizedResponseOpensBreaker(t *testing.T) {
	big := `{"data":{"getTransactions":[{"hash":"` + strings.Repeat("a", maxResponseSize) + `"}]}}`
	small := `{"data":{"getTransactions":[{"hash":"abc"}]}}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "small") {
			respond(w, small)
			return
		}
		respond(w, big)
	})

	var out struct {
		Txs []Tx `json:"getTransactions"`
	}
	for i := range breakerThreshold {
		err := c.Query(context.Background(), `{ getTransactions { hash } }`, &out)
		t.Logf("oversized result %d: %v", i+1, err)
	}

	// A different, small query from another reader.
	c.url += "?small"
	err := c.Query(context.Background(), `{ getTransactions { hash } }`, &out)
	t.Logf("next caller, small result: err=%v", err)
	if errors.Is(err, ErrUnavailable) {
		t.Errorf("%d oversized result sets open the breaker for every query for %s", breakerThreshold, breakerCooldown)
	}
}

// A 200 JSON body carrying neither "data" nor "errors" (here the JSON-RPC
// error object a JSON-RPC endpoint answers a request with no method) is taken
// as a successful, empty answer.
func TestB5EnvelopeWithoutDataReportsSuccess(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		respond(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"invalid request"}}`)
	})

	h, errTip := c.LatestBlockHeight(context.Background())
	t.Logf("LatestBlockHeight = %d, %v", h, errTip)
	_, errTx := c.TxByHash(context.Background(), "deadbeef")
	t.Logf("TxByHash = %v", errTx)

	if errTip == nil {
		t.Errorf("LatestBlockHeight = %d, nil on a body with neither data nor errors", h)
	}
	if errors.Is(errTx, ErrNotFound) {
		t.Errorf("TxByHash reports the transaction absent (%v) on a body with neither data nor errors", errTx)
	}
}
