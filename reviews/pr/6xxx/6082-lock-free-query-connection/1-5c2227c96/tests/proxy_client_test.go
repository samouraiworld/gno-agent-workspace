package proxy

/* Run: from a gno checkout:
gh pr checkout 6082 -R gnolang/gno && git checkout 5c2227c96
curl -fsSL -o tm2/pkg/bft/proxy/client_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6082-lock-free-query-connection/1-5c2227c96/tests/proxy_client_test.go
go test -race -count=1 ./tm2/pkg/bft/proxy/
rm tm2/pkg/bft/proxy/client_test.go
*/

// The locking contract of the three ABCI connections, pinned structurally: the
// read-only connection must admit every caller at once, the mutating one must
// admit exactly one, and the two must not share a lock. Reintroducing any lock
// on the query connection deadlocks the first test rather than slowing it, so
// the property cannot regress quietly. 1.4s under -race, no DB and no VM.

import (
	"sync"
	"testing"
	"time"

	abci "github.com/gnolang/gno/tm2/pkg/bft/abci/types"
)

// gateApp reports entry into Query/CheckTx and blocks there until released, so
// a test can prove how many calls the client lets in at once without timing
// heuristics.
type gateApp struct {
	abci.BaseApplication
	entered chan struct{}
	release chan struct{}
}

func (g *gateApp) Query(abci.RequestQuery) abci.ResponseQuery {
	g.entered <- struct{}{}
	<-g.release
	return abci.ResponseQuery{}
}

func (g *gateApp) CheckTx(abci.RequestCheckTx) abci.ResponseCheckTx {
	g.entered <- struct{}{}
	<-g.release
	return abci.ResponseCheckTx{}
}

// The read-only connection must admit every caller at once. Reintroducing any
// per-call lock on it deadlocks this test rather than slowing it down, so the
// invariant cannot regress silently.
func TestReadOnlyABCIClient_AdmitsConcurrentQueries(t *testing.T) {
	t.Parallel()

	const n = 8
	app := &gateApp{entered: make(chan struct{}, n), release: make(chan struct{})}
	cli, err := NewLocalClientCreator(app).NewReadOnlyABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			if _, err := cli.QuerySync(abci.RequestQuery{Path: ".app/version"}); err != nil {
				t.Error(err)
			}
		})
	}

	for i := range n {
		select {
		case <-app.entered:
		case <-time.After(30 * time.Second):
			t.Fatalf("only %d of %d queries entered Application.Query: the query "+
				"connection is serialising again", i, n)
		}
	}
	close(app.release)
	wg.Wait()
}

// The mutating connection must still serialise: one caller inside the
// application, the rest waiting.
func TestABCIClient_SerialisesMutatingCalls(t *testing.T) {
	t.Parallel()

	app := &gateApp{entered: make(chan struct{}, 2), release: make(chan struct{})}
	cli, err := NewLocalClientCreator(app).NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := cli.CheckTxSync(abci.RequestCheckTx{}); err != nil {
				t.Error(err)
			}
		})
	}

	<-app.entered // first caller is inside
	select {
	case <-app.entered:
		t.Fatal("both CheckTx calls entered the application at once: the mutating " +
			"connection is no longer serialised")
	case <-time.After(200 * time.Millisecond):
	}
	close(app.release)
	wg.Wait()
}

// The consensus and mempool connections share one lock; the query connection
// shares none with them.
func TestReadOnlyClient_DoesNotContendWithConsensus(t *testing.T) {
	t.Parallel()

	app := &gateApp{entered: make(chan struct{}, 2), release: make(chan struct{})}
	creator := NewLocalClientCreator(app)
	cons, err := creator.NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}
	query, err := creator.NewReadOnlyABCIClient()
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if _, err := cons.CheckTxSync(abci.RequestCheckTx{}); err != nil {
			t.Error(err)
		}
	})
	<-app.entered // consensus is inside and holding its mutex

	wg.Go(func() {
		if _, err := query.QuerySync(abci.RequestQuery{}); err != nil {
			t.Error(err)
		}
	})
	select {
	case <-app.entered:
	case <-time.After(30 * time.Second):
		t.Fatal("query blocked behind the consensus connection")
	}
	close(app.release)
	wg.Wait()
}

// NewLocalClient's nil guard: an untyped nil gets a working mutex.
func TestNewLocalClient_NilLockerGetsAMutex(t *testing.T) {
	t.Parallel()

	cli, err := (&localClientCreator{app: abci.NewBaseApplication()}).NewABCIClient()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.QuerySync(abci.RequestQuery{}); err != nil {
		t.Fatal(err)
	}
}
