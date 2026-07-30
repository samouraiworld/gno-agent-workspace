# Review: PR [#6014](https://github.com/gnolang/gno/pull/6014)
Event: COMMENT

## Body
Reverting only [`store.go`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go) to the merge base makes the new test report the race, and 5af2a62c1 runs clean.

[PR #6018](https://github.com/gnolang/gno/pull/6018) carries the same change to the same lines of that file and of [`store_test.go`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store_test.go#L471), so whichever lands second conflicts. Which one should carry it?

The red `docs` check is a dead link in [`gnoland-networks.md`](https://github.com/gnolang/gno/blob/5af2a62c1/docs/resources/gnoland-networks.md?plain=1#L9), not a code problem.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6014-store-query-commit-race/1-5af2a62c1/review_claude-opus-5_davd-gzl.md [↗](review_claude-opus-5_davd-gzl.md)

## tm2/pkg/sdk/baseapp_test.go:1542-1543 [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp_test.go#L1542-L1543)
A latest-height store query does not in fact run safely alongside `Commit`. `.store` queries walk the [live tree](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/iavl/store.go#L300-L316), and pinning the version does not isolate the reader from nodes the commit [mutates in place](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/iavl/node.go#L314-L315). This test passes only because its 19 blocks carry no transaction, so the walk never reaches a rewritten node; the collision predates this PR and reproduces at the merge base.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6014 -R gnolang/gno
cat > tm2/pkg/sdk/zz_write_race_test.go <<'EOF'
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
	app.BeginBlock(abci.RequestBeginBlock{Header: &bft.Header{ChainID: "test-chain", Height: 1}})
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
	// Both handlers expect the stored counter to equal the one the transaction
	// carries, so it advances by one per block.
	counter := int64(0)
	for height := int64(2); height <= 20; height++ {
		app.BeginBlock(abci.RequestBeginBlock{Header: &bft.Header{ChainID: "test-chain", Height: height}})
		txBytes, err := amino.Marshal(newTxCounter(counter, counter))
		if err != nil {
			t.Fatal(err)
		}
		if res := app.DeliverTx(abci.RequestDeliverTx{Tx: txBytes}); !res.IsOK() {
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
EOF
CGO_ENABLED=1 go test -race -count=10 -run TestStoreQueryConcurrentWithCommitAndWrites ./tm2/pkg/sdk/
rm tm2/pkg/sdk/zz_write_race_test.go
```

```
==================
WARNING: DATA RACE
Read at 0x00c0005b61d0 by goroutine 12:
  github.com/gnolang/gno/tm2/pkg/iavl.(*Node).getRightNode()
      tm2/pkg/iavl/node.go:677 +0x3c
  github.com/gnolang/gno/tm2/pkg/iavl.(*MutableTree).GetVersioned()
      tm2/pkg/iavl/mutable_tree.go:686 +0x1d2
  github.com/gnolang/gno/tm2/pkg/store/iavl.(*Store).Query()
      tm2/pkg/store/iavl/store.go:316 +0x221
# …
Previous write at 0x00c0005b61d0 by goroutine 11:
  github.com/gnolang/gno/tm2/pkg/iavl.(*Node).clone()
      tm2/pkg/iavl/node.go:315 +0x1b2
  github.com/gnolang/gno/tm2/pkg/iavl.(*MutableTree).Set()
      tm2/pkg/iavl/mutable_tree.go:170 +0x78
  github.com/gnolang/gno/tm2/pkg/store/cache.(*cacheStore).writeLocked()
      tm2/pkg/store/cache/store.go:301 +0xb54
# …
==================
    testing.go:1617: race detected during execution of test
--- FAIL: TestStoreQueryConcurrentWithCommitAndWrites (0.04s)
FAIL
```
</details>

## tm2/pkg/sdk/baseapp_test.go:1544 [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp_test.go#L1544)
Nothing automated passes `-race`: the tm2 job runs [plain `go test`](https://github.com/gnolang/gno/blob/5af2a62c1/.github/workflows/_ci-go.yml#L124), and `make test` exports [`CGO_ENABLED=0`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/Makefile#L16-L17), under which `go test -race` refuses to run. Without the flag the only assertion left is [`queries.Load() > 1`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp_test.go#L1587), which the loop satisfies by construction. Restoring the plain struct field would keep every check green.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6014 -R gnolang/gno
CGO_ENABLED=0 go test -race -count=1 -run TestStoreQueryConcurrentWithCommit ./tm2/pkg/sdk/
grep -rn -- '-race' .github/workflows/ | wc -l
```

```
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
0
```
</details>

## tm2/pkg/store/rootmulti/store.go:54 [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L54)
Nit: this field is the only one in the struct with an unstated concurrency contract; [`querySnapshot`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L67-L69) and [`snapshotMu`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L79-L97) both name their writers and readers. With no automated `-race` run, a comment here is the only guard against a future cleanup silently reverting it to a plain struct.

## tm2/pkg/store/rootmulti/store.go:236-241 [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L236-L241)
Suggestion: the returned `Hash` still points at the one slice every reader shares, and nothing says it must not be written to after publication. Both publishers build a fresh slice today, so this is latent.
