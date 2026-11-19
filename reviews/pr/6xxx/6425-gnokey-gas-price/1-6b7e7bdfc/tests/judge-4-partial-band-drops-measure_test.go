// Judge repro, gnolang/gno#6425 at 6b7e7bdfc61f4db559693bfdc521da564f364f41, candidate 4.
//
// From a plain clone of gnolang/gno:
//
//	git fetch origin pull/6425/head && git checkout --detach 6b7e7bdfc61f4db559693bfdc521da564f364f41
//	cp <this file> gno.land/pkg/gnoweb/zz_judge4_partial_band_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestJudge4PartialBandDropsMeasure -v -count=1
//
// An indexer that answers every band truthfully: the package's only calls sit
// in the fourth band (height 9,500,000, head 10,000,000). A fast read walks
// four bands and measures them. A slow read (3.2 s a band, under the 4 s
// client timeout) fits three bands in gasUseTimeout, returns 0 rows with
// ErrPartial, and gas_use.go:122 stores the empty map for gasUseTTL.
package gnoweb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/indexer"
	lru "github.com/hashicorp/golang-lru/v2"
)

func TestJudge4PartialBandDropsMeasure(t *testing.T) {
	row := `{"hash":"h","success":true,"block_height":9500000,"gas_used":1400000,` +
		`"messages":[{"value":{"__typename":"MsgCall","pkg_path":"` + testPkg + `","func":"Deposit"}}]}`
	var slow atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "latestBlockHeight") {
			_, _ = io.WriteString(w, `{"data":{"latestBlockHeight":10000000}}`)
			return
		}
		if slow.Load() {
			time.Sleep(3200 * time.Millisecond)
		}
		if strings.Contains(string(body), "gt: 8830000") { // the fourth band only
			_, _ = io.WriteString(w, `{"data":{"getTransactions":[`+row+`]}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"getTransactions":[]}}`)
	}))
	defer srv.Close()

	idx := indexer.New(srv.URL, "")
	clock := &fakeClock{}
	c := &gasUseCache{logger: newDiscardLogger(), now: clock.now}
	c.recent = func(ctx context.Context, p string, n int) ([]indexer.Tx, error) {
		txs, err := idx.RecentByPackage(ctx, p, n)
		t.Logf("RecentByPackage: %d rows, err=%v", len(txs), err)
		return txs, err
	}
	c.entries, _ = lru.New[string, gasUseEntry](maxGasUsePkgs)
	ctx := context.Background()

	c.read(ctx, testPkg)
	first := c.get(ctx, testPkg)["Deposit"]
	t.Logf("fast read: Deposit = %+v", first)
	if first.Calls == 0 {
		t.Fatalf("fast read did not measure")
	}
	clock.add(gasUseTTL)
	slow.Store(true)
	c.read(ctx, testPkg)
	got := c.get(ctx, testPkg)["Deposit"]
	_, fresh := c.lookup(testPkg)
	t.Logf("slow read: Deposit = %+v, fresh=%v", got, fresh)
	if got != first {
		t.Fatalf("last measure dropped: got %+v, want %+v", got, first)
	}
}
