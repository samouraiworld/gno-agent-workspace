// Repro, gnolang/gno#6425 at 6b7e7bdfc61f4db559693bfdc521da564f364f41, finder b2 lines/reach/catalog.
//
// From a plain clone of gnolang/gno:
//
//	git fetch origin pull/6425/head && git checkout --detach 6b7e7bdfc61f4db559693bfdc521da564f364f41
//	cp <this file> gno.land/pkg/gnoweb/zz_b2_partial_drops_measure_test.go
//	go test ./gno.land/pkg/gnoweb/ -run TestB2PartialEmptyDropsMeasure -v -count=1
//
// Claim: gas_use.go:125-127 says a failed read keeps the last measure, since
// "one timeout would otherwise send the fast-mode gas back to TxGasWanted". A
// slow indexer whose bands each answer under the client's 4 s defaultTimeout
// (3.2 s here) spends the read's 10 s budget in three bands; indexer.recent
// then breaks its walk on minWindowBudget and returns zero rows with
// ErrPartial. read() takes ErrPartial as a success (gas_use.go:122) and
// stores an empty map for gasUseTTL, so the last measure is dropped for
// 5 minutes. Expect FAIL on the last assertion.
//
// Observed at 6b7e7bdfc (go1.27.3):
//
//	RecentByPackage: 12 rows, err=searched the most recent blocks only, took 0s
//	first read: Deposit = {Max:1400000 Calls:12}
//	RecentByPackage: 0 rows, err=searched the most recent blocks only, took 9.6s
//	after a slow read: Deposit = {Max:0 Calls:0}, entry fresh=true, expires in 5m0s, uses=map[]
//	--- FAIL: TestB2PartialEmptyDropsMeasure (9.60s)
//
// A band slower than 4 s fails as an error instead and the measure is kept
// (tried: 4.6 s bands, PASS), and a chain short enough for three bands to
// reach block 0 ends the walk with err=nil, which is a real "no calls".
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

func TestB2PartialEmptyDropsMeasure(t *testing.T) {
	row := func(h, gas int) string {
		return `{"hash":"h","success":true,"block_height":` + itoa(h) + `,"gas_used":` + itoa(gas) +
			`,"messages":[{"value":{"__typename":"MsgCall","pkg_path":"` + testPkg + `","func":"Deposit"}}]}`
	}
	rows := strings.Join([]string{row(9999990, 1_400_000), row(9999980, 1_380_000), row(9999970, 1_360_000)}, ",")
	var slow atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "latestBlockHeight") {
			_, _ = io.WriteString(w, `{"data":{"latestBlockHeight":10000000}}`)
			return
		}
		if slow.Load() {
			time.Sleep(3200 * time.Millisecond) // a loaded indexer: each band answers in 3.2 s, under the client timeout, empty
			_, _ = io.WriteString(w, `{"data":{"getTransactions":[]}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"getTransactions":[`+rows+`]}}`)
	}))
	defer srv.Close()

	idx := indexer.New(srv.URL, "")
	clock := &fakeClock{}
	c := &gasUseCache{logger: newDiscardLogger(), now: clock.now}
	c.recent = func(ctx context.Context, p string, n int) ([]indexer.Tx, error) {
		start := time.Now()
		txs, err := idx.RecentByPackage(ctx, p, n)
		t.Logf("RecentByPackage: %d rows, err=%v, took %s", len(txs), err, time.Since(start).Round(100*time.Millisecond))
		return txs, err
	}
	c.entries, _ = lru.New[string, gasUseEntry](maxGasUsePkgs)
	ctx := context.Background()

	c.read(ctx, testPkg)
	first := c.get(ctx, testPkg)["Deposit"]
	t.Logf("first read: Deposit = %+v", first)
	if first.Calls == 0 {
		t.Fatalf("first read did not measure: %+v", first)
	}

	clock.add(gasUseTTL)
	slow.Store(true)
	c.read(ctx, testPkg)
	got := c.get(ctx, testPkg)["Deposit"]
	e, fresh := c.lookup(testPkg)
	t.Logf("after a slow read: Deposit = %+v, entry fresh=%v, expires in %s, uses=%v", got, fresh, e.expires.Sub(clock.now()), e.uses)
	if got != first {
		t.Fatalf("measure dropped by a read that ran out of budget: got %+v, want the last measure %+v", got, first)
	}
}

func itoa(n int) string {
	var b []byte
	if n == 0 {
		return "0"
	}
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
