# Review: [#4885](https://github.com/gnolang/gno/pull/4885)
Posted: https://github.com/gnolang/gno/pull/4885#pullrequestreview-5040464036
Event: COMMENT

## Body
[AI review, not manually verified]

Status: APPROVE

Looks good.

<details><summary>reverts run against this head</summary>

Putting `StringValue.GetShallowSize()` back to header plus bytes moves `alloc_13a.gno`'s post-GC total from the asserted 8972 to 11076, one extra copy of the 1052-byte backing for `s1` and one for the package slot. Putting `convert.go`'s string arm back to `tv.SetString(gno.StringValue(arg))` makes `TestConvertArgToGno_StringArgIsCharged` report `"0" is not greater than "1000"`. Both fixtures pin the behaviour rather than a total that happens to match.
</details>

## gnovm/pkg/gnolang/realm.go:1895 [gh](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/realm.go#L1895) · [↗](../../../../../.worktrees/gno-review-4885/gnovm/pkg/gnolang/realm.go#L1895) [posted](https://github.com/gnolang/gno/pull/4885#discussion_r3871472083)
Suggestion: on the query path `m.Alloc` is not the allocator this line mints into, so [`GCVisitorFn`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/garbage_collector.go#L247) bills a 4096-byte string loaded from the store as its 48-byte header. Either mint into the allocator that recounts, or say in the ADR that the query machine's tally leaves stored strings out.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4885 -R gnolang/gno
cat > gnovm/pkg/gnolang/zz_qalloc_test.go <<'EOF'
package gnolang

import (
	"strings"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

// Mirrors withQueryEvalMachine: the machine gets its own allocator, the
// store keeps the one it was forked with.
func TestZZQueryStringRecount(t *testing.T) {
	tm2Store := dbadapter.StoreConstructor(memdb.NewMemDB(), storetypes.StoreOptions{})
	st := NewStore(nil, tm2Store, tm2Store)
	st.SetAllocator(NewAllocator(1 << 30))

	m := NewMachineWithOptions(MachineOptions{
		Store: st, Alloc: NewAllocator(1 << 30), SkipPackage: true,
	})
	defer m.Release()

	loaded := fillTypesOfValue(nil, st, StringValue(strings.Repeat("x", 4096)))
	var vc int64
	vis := GCVisitorFn(1, m.Alloc, &vc)
	m.Alloc.Reset()
	vis(loaded)
	_, recount := m.Alloc.Status()
	if recount == allocString {
		t.Fatalf("recount = %d, the header alone, for a live 4096-byte loaded string", recount)
	}
}
EOF
go test -count=1 -v -run TestZZQueryStringRecount ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/zz_qalloc_test.go
```

The collector bills 48 bytes for a live 4096-byte string that came from the store; the merge-base bills 4144.

```
=== RUN   TestZZQueryStringRecount
    zz_qalloc_test.go:31: recount = 48, the header alone, for a live 4096-byte loaded string
--- FAIL: TestZZQueryStringRecount (0.00s)
FAIL
FAIL	github.com/gnolang/gno/gnovm/pkg/gnolang	0.058s
```

[`withQueryEvalMachine`](https://github.com/gnolang/gno/blob/6c2839ca0/gno.land/pkg/sdk/vm/keeper.go#L1740) is the one caller where the two allocators differ: [`NewMachineWithOptions`](https://github.com/gnolang/gno/blob/6c2839ca0/gnovm/pkg/gnolang/machine.go#L189-L190) installs its allocator on the store only when the store has none, and the query's throwaway transaction store already carries one. The four transaction paths pass `gnostore.GetAllocator()` and see a single tally. The bytes are not lost: this branch charges them to the store's allocator at load, and that allocator has no collect function, so it hard-caps at `maxAllocTx`. The ADR's audit lists `fillTypesOfValue` among the entry points that make every Gno-visible string tracked, which is what this contradicts.
</details>
