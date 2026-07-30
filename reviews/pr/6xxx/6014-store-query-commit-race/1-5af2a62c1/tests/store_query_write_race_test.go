/* Run: from a gno checkout:
gh pr checkout 6014 -R gnolang/gno && git checkout 5af2a62c1
curl -fsSL -o tm2/pkg/sdk/store_query_write_race_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6014-store-query-commit-race/1-5af2a62c1/tests/store_query_write_race_test.go
CGO_ENABLED=1 go test -race -count=10 -run 'TestStoreQueryConcurrentWithCommitAndWrites' ./tm2/pkg/sdk/
rm tm2/pkg/sdk/store_query_write_race_test.go
*/

// A .store query reads the live iavl tree through rootmulti.multiStore.Query,
// while Commit rewrites nodes of that same tree in MultiWrite. At 5af2a62c1 the
// race detector reports iavl Node.clone against Node.getRightNode on the same
// node, at least once per batch of ten runs. Serving .store queries from the
// committed snapshot makes every run clean.
package sdk

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
	bft "github.com/gnolang/gno/tm2/pkg/bft/types"
)

func TestStoreQueryConcurrentWithCommitAndWrites(t *testing.T) {
	deliverKey := []byte("deliver-key")
	anteKey := []byte("ante-key")
	anteOpt := func(bapp *BaseApp) { bapp.SetAnteHandler(anteHandlerTxTest(t, mainKey, anteKey)) }
	routerOpt := func(bapp *BaseApp) {
		bapp.Router().AddRoute(routeMsgCounter, newMsgCounterHandler(t, mainKey, deliverKey))
	}

	app := setupBaseApp(t, anteOpt, routerOpt)
	app.InitChain(abci.RequestInitChain{ChainID: "test-chain"})

	app.BeginBlock(abci.RequestBeginBlock{
		Header: &bft.Header{ChainID: "test-chain", Height: 1},
	})
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	// Query the key the message handler writes, so the query walks the part of
	// the tree the next commit rewrites.
	query := abci.RequestQuery{Path: ".store/main/key", Data: deliverKey}

	ready := make(chan struct{})
	done := make(chan struct{})
	var queries atomic.Int64
	var wg sync.WaitGroup
	wg.Go(func() {
		app.Query(query)
		queries.Add(1)
		close(ready)
		for {
			select {
			case <-done:
				return
			default:
				app.Query(query)
				queries.Add(1)
			}
		}
	})

	<-ready
	// The ante handler and the message handler each expect the stored counter to
	// equal the one carried by the transaction, so it advances by one per block.
	counter := int64(0)
	for height := int64(2); height <= 20; height++ {
		app.BeginBlock(abci.RequestBeginBlock{
			Header: &bft.Header{ChainID: "test-chain", Height: height},
		})
		tx := newTxCounter(counter, counter)
		txBytes, err := amino.Marshal(tx)
		if err != nil {
			t.Fatal(err)
		}
		res := app.DeliverTx(abci.RequestDeliverTx{Tx: txBytes})
		if !res.IsOK() {
			t.Fatalf("DeliverTx at height %d: %v", height, res)
		}
		counter++
		app.EndBlock(abci.RequestEndBlock{})
		app.Commit()
	}
	close(done)
	wg.Wait()

	if queries.Load() < 2 {
		t.Fatalf("query goroutine ran %d times, expected the loop to iterate", queries.Load())
	}
}
