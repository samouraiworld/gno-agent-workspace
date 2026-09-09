// Repro, from a plain clone of github.com/gnolang/gno:
//
//	cp objectid_bench_test.go gnovm/cmd/calibrate/
//	cd gnovm/cmd/calibrate
//	go test -bench 'BenchmarkNative_ChainRuntime_ObjectID|BenchmarkNative_Chain_PackageAddress_20' \
//	    -benchtime=200ms -count=3 -run '^$' .
//
// Drives chain/runtime.objectID through the same dispatcher harness every
// other row of calibratedNativeGas was fitted on, so its ns/op is directly
// comparable to the Base values in gnovm/stdlibs/native_gas.go. The
// chain.packageAddress bench beside it is the closest calibrated neighbour:
// same truncated-SHA256-plus-bech32 derivation, priced at Base 552.
package calibrate

import (
	"math"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

// benchObjectIDDispatch builds a finalized struct object under a realm and
// calls the generated chain/runtime.objectID wrapper on it.
func benchObjectIDDispatch(b *testing.B, pkgPath string, newTime uint64) {
	b.Helper()

	alloc := gno.NewAllocator(math.MaxInt64)
	object := alloc.NewStruct(nil, nil)
	object.SetPkgID(gno.PkgIDFromPkgPath(pkgPath))
	object.SetNewTime(newTime)

	m := &gno.Machine{Alloc: gno.NewAllocator(math.MaxInt64), Stage: gno.StageRun}
	m.Blocks = []*gno.Block{{Values: []gno.TypedValue{{V: object}}}}

	h := &dispatchHarness{m: m, wrapper: resolveWrapper(b, "chain/runtime", "objectID"), nReturns: 1}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.call()
	}
}

// A realm path of typical length, at the first and at the largest realm tick.
// Neither is a table input: objectID is priced flat.
func BenchmarkNative_ChainRuntime_ObjectID(b *testing.B) {
	benchObjectIDDispatch(b, "gno.land/r/demo/foo20", 1)
}

func BenchmarkNative_ChainRuntime_ObjectID_MaxTime(b *testing.B) {
	benchObjectIDDispatch(b, "gno.land/r/demo/foo20", ^uint64(0))
}

// chain.packageAddress on a 21-character path, the length of the realm path
// above, for a side-by-side against a row that was fitted.
func BenchmarkNative_Chain_PackageAddress_21(b *testing.B) {
	benchChainPackageAddress(b, 21)
}
