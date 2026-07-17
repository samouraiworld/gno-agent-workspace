# Review: PR [#5938](https://github.com/gnolang/gno/pull/5938)
Event: REQUEST_CHANGES

## Body
Reproduced on 27c5ece7e: with only the store constructor differing, the per-query store open stays flat at ~14µs on IAVL from 1K to 100K retained versions and reaches 100.9ms on bptree at 100K.

- The squash-merged commit message says SET/WRITE became estimator-driven at Fixed=0 with Min floors, but [`NewParams`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L82-L84) pins Fixed = Min = 200/540 and [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/cache/store.go#L105-L108) returns the Fixed value on any non-zero pin, so `git log` tells a future maintainer that consensus gas self-corrects with state growth when only governance moves it.
- The same message credits `TestDefaultParams` with pinning that zero Fixed values are representable, which [the test](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params_test.go#L291-L311) never exercises, and signs off with `ADR: gno.land/adr/prxxxx_mount_bptree_store.md`, which landed as [`pr5938_mount_bptree_store.md`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/adr/pr5938_mount_bptree_store.md).

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5938-mount-bptree-fast-index/1-27c5ece7e/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/gnoland/app.go:106 [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L106)
Critical: every custom ABCI query re-opens the store at a height through [`MultiImmutableCacheWrapWithVersion`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/sdk/baseapp.go#L505) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/sdk/baseapp.go#L505), and the bptree store's immutable load calls [`st.mtree.Load()`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/bptree/store.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/bptree/store.go#L187) first, which runs [`discoverVersions`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/nodedb.go#L473-L495) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/nodedb.go#L473) over every retained root record, twice, where [IAVL's immutable load](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/iavl/store.go#L177-L184) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/iavl/store.go#L177) went straight to one root fetch. Measured, the per-query open grows ~480x between 1K and 100K retained versions while IAVL stays flat, and [`PruneSyncable`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/types/options.go#L42) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/types/options.go#L42) retains 705,600 by [default](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app.go#L63) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L63), reached in about eight days at one-second blocks and unbounded on archive nodes. It holds the mutex [`QuerySync`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bft/abci/client/local_client.go#L176-L182) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bft/abci/client/local_client.go#L176) shares with `CommitSync`, so this wants a fix before a chain launches from the mount.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5938 -R gnolang/gno

cat > tm2/pkg/store/rootmulti/zz_scan_test.go <<'EOF'
package rootmulti

import (
	"fmt"
	"testing"
	"time"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"
	"github.com/gnolang/gno/tm2/pkg/store/iavl"
	"github.com/gnolang/gno/tm2/pkg/store/types"
)

func TestZZScan(t *testing.T) {
	for _, b := range []struct {
		name string
		ctor types.CommitStoreConstructor
	}{
		{"iavl", iavl.StoreConstructor},
		{"bptree+fastindex", storebptree.FastStoreConstructor},
	} {
		for _, versions := range []int{1000, 20000, 100000} {
			db := memdb.NewMemDB()
			ms := NewMultiStore(db)
			key := types.NewStoreKey("main")
			ms.MountStoreWithDB(key, b.ctor, db)
			if err := ms.LoadLatestVersion(); err != nil {
				t.Fatal(err)
			}
			for i := range versions {
				ms.GetStore(key).Set(nil, []byte(fmt.Sprintf("k%06d", i)), []byte("v"))
				ms.Commit()
			}
			last := ms.LastCommitID().Version
			start := time.Now()
			for range 10 {
				// exactly what baseapp's handleQueryCustom does per query
				if _, err := ms.MultiImmutableCacheWrapWithVersion(last); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("%-18s retained=%6d  per-query store open = %v", b.name, versions, time.Since(start)/10)
		}
	}
}
EOF

go test -v -run TestZZScan -timeout 1800s ./tm2/pkg/store/rootmulti/ 2>&1 | grep "per-query"
rm tm2/pkg/store/rootmulti/zz_scan_test.go
```

```
    zz_scan_test.go:44: iavl               retained=  1000  per-query store open = 18.954µs
    zz_scan_test.go:44: iavl               retained= 20000  per-query store open = 14.508µs
    zz_scan_test.go:44: iavl               retained=100000  per-query store open = 14.097µs
    zz_scan_test.go:44: bptree+fastindex   retained=  1000  per-query store open = 209.086µs
    zz_scan_test.go:44: bptree+fastindex   retained= 20000  per-query store open = 17.582257ms
    zz_scan_test.go:44: bptree+fastindex   retained=100000  per-query store open = 100.863395ms
```
</details>

## gno.land/pkg/sdk/vm/params.go:41 [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L41)
This comment newly calls 2.0 measured, but [`PERFORMANCE.md`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L64) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L64), cited five lines up as the provenance, measures SET reads at 2.86 at the same ~101M calibration point and records the correction as ["modeled 2.0 → measured 2.86"](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L101-L102) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L101), having stated that the [measured number wins](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L34-L37) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L34) where the two disagree. The mount moves this pin from 17x under to 1.43x under, so it is the label that is wrong rather than the direction, but at `ReadCostFlat` 59,000 the 0.86 gap still undercharges every SET by ~50,700 gas on a genesis default.

## contribs/gnogenesis/internal/fork/generate.go:453-457 [↗](../../../../../.worktrees/gno-review-5938/contribs/gnogenesis/internal/fork/generate.go#L453)
The rewrite mutates the forked chain's consensus gas params and prints nothing, so an operator has no signal that the depth params in their output genesis differ from their input. The fingerprint match infers "untuned" from an exact value match, so the case where that inference is wrong is also the case that leaves no trace.

## contribs/gnogenesis/internal/fork/generate.go:676-681 [↗](../../../../../.worktrees/gno-review-5938/contribs/gnogenesis/internal/fork/generate.go#L676)
Missing test: nothing reddens when the vm depth defaults change without a matching era fingerprint being appended here. The rule lives only in this comment and its mirror at [`params.go:38-39`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/params.go#L38-L39) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/params.go#L38), and the era this PR is repairing is the one that had no guard either.

<details><summary>test cases</summary>

Drops into `contribs/gnogenesis/internal/fork/`. A two-step ratchet: the live defaults must equal the last history entry, and every superseded entry must be an `untunedDepthFingerprints` era. Green at 27c5ece7e; changing a depth default reddens step 1, and appending the new era to satisfy it then reddens step 2 until the superseded set is fingerprinted. Full file, including a third test keeping the current defaults out of the fingerprint list: [`depth_fingerprint_decay_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5938-mount-bptree-fast-index/1-27c5ece7e/tests/depth_fingerprint_decay_test.go) · [↗](tests/depth_fingerprint_decay_test.go).

```go
// depthDefaultsHistory is every depth/iteration gas default set vm.DefaultParams()
// has ever returned, OLDEST FIRST. Append-only. The LAST entry must always equal
// the live defaults; every earlier entry must be an untunedDepthFingerprints era.
var depthDefaultsHistory = []vm.Params{
	// Era 0 — pre-#5415 (gas-storage refactor): the fields did not exist and
	// deserialize to zero.
	{},
	// Era 1 — post-#5415, pre-bptree-mount: IAVL-era untuned defaults.
	{
		MinGetReadDepth100: 300, MinSetReadDepth100: 200, MinWriteDepth100: 440,
		FixedGetReadDepth100: 300, FixedSetReadDepth100: 200, FixedWriteDepth100: 440,
		IterNextCostFlat: 1_000,
	},
	// Era 2 — bptree mount + fast index. CURRENT.
	{
		MinGetReadDepth100: 100, MinSetReadDepth100: 200, MinWriteDepth100: 540,
		FixedGetReadDepth100: 100, FixedSetReadDepth100: 200, FixedWriteDepth100: 540,
		IterNextCostFlat: 1_000,
	},
}

// Step 1: changing a depth default reddens this until the new set is APPENDED.
func TestDepthDefaultsHistoryTracksLiveDefaults(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, depthDefaultsHistory)
	current := depthDefaultsHistory[len(depthDefaultsHistory)-1]

	assert.True(t, depthParamsMatch(vm.DefaultParams(), current),
		"vm.DefaultParams() depth/iteration fields no longer match the last entry of "+
			"depthDefaultsHistory.\n  live:   %s\n  pinned: %s\n"+
			"If the change is intentional: APPEND the new default set as a new era at the "+
			"end of depthDefaultsHistory (do not edit the existing last entry).",
		fmtDepth(vm.DefaultParams()), fmtDepth(current))
}

// Step 2: every superseded set shipped on some chain, so a genesis exported from
// one carries it verbatim and buildHardforkGenesis must recognize it as untuned.
func TestDepthDefaultsSupersededAreFingerprinted(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, depthDefaultsHistory)
	superseded := depthDefaultsHistory[:len(depthDefaultsHistory)-1]

	for i, era := range superseded {
		found := false
		for _, fp := range untunedDepthFingerprints {
			if depthParamsMatch(era, fp) {
				found = true
				break
			}
		}
		assert.Truef(t, found,
			"era %d of depthDefaultsHistory (%s) is a superseded vm default set but is NOT in "+
				"untunedDepthFingerprints (generate.go).\nA genesis exported from a chain running "+
				"those defaults would fork under the CURRENT store/pricing without being repriced.",
			i, fmtDepth(era))
	}
}
```
</details>

## gno.land/pkg/gnoland/app_test.go:1517 [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app_test.go#L1517)
The depth pins are calibrated for bptree specifically, but this is the only thing that reddens if [the app's mount](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/gnoland/app.go#L106) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/gnoland/app.go#L106) goes back to IAVL, and only by accident: it fails on a DB-format collision naming no cause, because its own multistore mounts bptree over a DB the app filled with IAVL data. The four rebaselined gas goldens and [`TestAppHashCrossrealm38`](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L74) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L74) stay green, since [`effectiveSetReadDepth100`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/store/cache/store.go#L105-L108) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/store/cache/store.go#L105) short-circuits on the Fixed pin and never reads the tree. A chain that took that revert would charge the pinned 2.0 for IAVL SET reads [measured at 34.1](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/PERFORMANCE.md?plain=1#L64) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/PERFORMANCE.md#L64).

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5938 -R gnolang/gno

# overlay that reverts only the mount, leaving the working tree untouched
mkdir -p /tmp/ov
sed -e 's#storebptree "github.com/gnolang/gno/tm2/pkg/store/bptree"#"github.com/gnolang/gno/tm2/pkg/store/iavl"#' \
    -e 's#baseApp.MountStoreWithDB(mainKey, storebptree.FastStoreConstructor, cfg.DB)#baseApp.MountStoreWithDB(mainKey, iavl.StoreConstructor, cfg.DB)#' \
    gno.land/pkg/gnoland/app.go > /tmp/ov/app.go
printf '{"Replace":{"%s/gno.land/pkg/gnoland/app.go":"/tmp/ov/app.go"}}\n' "$(pwd)" > /tmp/ov/overlay.json

echo "== gas goldens =="; go test -overlay=/tmp/ov/overlay.json ./gno.land/pkg/integration/ -run 'TestTestdata/(restart_gas|gnokey_gasfee|simulate_gas|gc)$' 2>&1 | tail -1
echo "== apphash golden =="; go test -overlay=/tmp/ov/overlay.json ./gno.land/pkg/sdk/vm/ -run 'TestAppHashCrossrealm38' 2>&1 | tail -1
echo "== gnoland =="; go test -overlay=/tmp/ov/overlay.json ./gno.land/pkg/gnoland/ 2>&1 | grep -E '^(--- FAIL|ok|FAIL)' | head -3
rm -rf /tmp/ov
```

```
== gas goldens ==
ok  	github.com/gnolang/gno/gno.land/pkg/integration	43.890s
== apphash golden ==
ok  	github.com/gnolang/gno/gno.land/pkg/sdk/vm	41.746s
== gnoland ==
--- FAIL: TestPruneStrategyNothing (12.56s)
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoland	46.783s
```
</details>

## gno.land/pkg/integration/testdata/addpkg_outofgas.txtar:10-12 [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L10)
Nit: neither case fails "early in store access", and the [63000 case](https://github.com/gnolang/gno/blob/27c5ece7e/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L22) · [↗](../../../../../.worktrees/gno-review-5938/gno.land/pkg/integration/testdata/addpkg_outofgas.txtar#L22) is not "slightly later" than the 60000 one. Both report the same `GAS USED: 2847971` and are rejected by gnokey's simulate check before the meter they name is ever applied, so the two differ only in the number printed in the error.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5938 -R gnolang/gno
go test ./gno.land/pkg/integration/ -run 'TestTestdata/addpkg_outofgas' -v 2>&1 | grep -E "GAS WANTED|GAS USED|deliver transaction failed"
```

```
        GAS WANTED: 60000
        GAS USED:   2847971
            0  gno/tm2/pkg/errors/errors.go:103 - deliver transaction failed: log:gas used (2847971) exceeds tx's gas wanted (60000) during operation: simulation
        GAS WANTED: 63000
        GAS USED:   2847971
            0  gno/tm2/pkg/errors/errors.go:103 - deliver transaction failed: log:gas used (2847971) exceeds tx's gas wanted (63000) during operation: simulation
```
</details>

## tm2/pkg/bptree/benchmarks/BENCHMARKS.md:3-8 [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L3)
Nit: this says the ~101M run has since been measured, but [`## TODO — the 100M run (the one that matters)`](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L247) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L247) and ["Confirm against the 100M run."](https://github.com/gnolang/gno/blob/27c5ece7e/tm2/pkg/bptree/benchmarks/BENCHMARKS.md?plain=1#L264) · [↗](../../../../../.worktrees/gno-review-5938/tm2/pkg/bptree/benchmarks/BENCHMARKS.md#L264) are still in the file.
