// Repro for gnolang/gno#6231 at dd40ef3dc,
// gno.land/pkg/gnoweb/feature/omnisearch/resolve_indexer.go resolveContent / resolveImporters.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6231/head && git checkout dd40ef3dc9b92a3df888bb9e70dc21871dcc192a
//	cp <this file> gno.land/pkg/gnoweb/feature/omnisearch/zz_multimsg_repro_test.go
//	go test ./gno.land/pkg/gnoweb/feature/omnisearch/ -run 'TestZZ' -v -count=1
//
// tx-indexer returns a transaction when ANY of its messages matches the
// MsgAddPackage filter (FilterTransaction.Eval, "elemMatchMessages"), and
// returns all of its messages. The tx below deploys gno.land/r/alice/a (the
// match) and, in the same tx, calls gno.land/r/bob/b. Both resolvers list
// every message's path, so r/bob/b is shown as "contains avl.Tree" and as an
// importer of gno.land/r/demo/foo.
package omnisearch

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/indexer"
)

const zzMultiMsgTx = `[{"hash":"h1","block_height":7,"success":true,"messages":[
 {"value":{"__typename":"MsgAddPackage","creator":"g1alice","package":{"path":"gno.land/r/alice/a"}}},
 {"value":{"__typename":"MsgCall","caller":"g1alice","pkg_path":"gno.land/r/bob/b","func":"Register"}}
]}]`

func zzTxs(t *testing.T) []indexer.Tx {
	t.Helper()
	var txs []indexer.Tx
	if err := json.Unmarshal([]byte(zzMultiMsgTx), &txs); err != nil {
		t.Fatal(err)
	}
	return txs
}

func TestZZContentListsTheCalledRealm(t *testing.T) {
	h := newHandler(t, &mockClient{}, &mockIndexer{txs: zzTxs(t)})
	groups, _ := h.Search(context.Background(), mustQuery(t, h, "content:avl.Tree", ""))
	for _, g := range groups {
		for _, r := range g.Results {
			t.Logf("content result: %q detail=%q tags=%v", r.Title, r.Detail, r.Tags)
			if r.Title == "gno.land/r/bob/b" {
				t.Errorf("content:avl.Tree lists %q, a realm the tx only called", r.Title)
			}
		}
	}
}

func TestZZImportersListsTheCalledRealm(t *testing.T) {
	h := newHandler(t, &mockClient{}, &mockIndexer{txs: zzTxs(t)})
	groups, _ := h.Search(context.Background(), mustQuery(t, h, "importers", "/r/demo/foo"))
	for _, g := range groups {
		for _, r := range g.Results {
			t.Logf("importers result: %q detail=%q tags=%v", r.Title, r.Detail, r.Tags)
			if r.Title == "gno.land/r/bob/b" {
				t.Errorf("importers lists %q (%s), a realm the tx only called", r.Title, r.Detail)
			}
		}
	}
}
