// b2-reach: chain/reflect.objectInfo is priced flat (Base 1200, SizeFlat) in
// gnovm/stdlibs/native_gas.go, whose row comment says "nothing here scales with
// caller input". declaredTypeName builds dt.PkgPath + "." + dt.Name on every
// call, and the type name is an identifier the caller declares in its own
// package, bounded only by the tx size (MaxTxBytes 1_000_000 in
// gno.land/pkg/gnoland/node_inmemory.go). This benchmark feeds the same stamped
// object through the native with its declared type name at 8 B, 64 KiB and
// 512 KiB and reports ns/op against the flat 1200-gas price.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6214/head && git checkout --detach 58b2692067d6545410b99c71f33bdf506caadaa0
//	cp <this file> gnovm/cmd/calibrate/b2_reach_typename_bench_test.go
//	cd gnovm && go test ./cmd/calibrate -run '^$' -bench 'Reflect_ObjectInfo_(Stamped$|TypeName)' -benchtime 2000x -count 3
//	rm cmd/calibrate/b2_reach_typename_bench_test.go
//
// Reuses newReflectObjectInfoBench from the PR's own reflect_bench_test.go.
package calibrate

import (
	"strings"
	"testing"

	gno "github.com/gnolang/gno/gnovm/pkg/gnolang"
)

func benchObjectInfoTypeName(b *testing.B, n int) {
	h := newReflectObjectInfoBench(b, true)
	name := gno.Name(strings.Repeat("A", n))
	tv := h.m.Blocks[0].Values[0]
	tv.T = &gno.PointerType{Elt: &gno.DeclaredType{PkgPath: reflectBenchPkgPath, Name: name, Base: &gno.StructType{}}}
	h.m.Blocks[0].Values[0] = tv
	// sanity: the native must return the long name, so the path is the real one
	h.wrapper(h.m)
	rets := h.m.PopValues(6)
	if got := len(rets[3].GetString()); got != len(reflectBenchPkgPath)+1+n {
		b.Fatalf("typ len = %d, want %d", got, len(reflectBenchPkgPath)+1+n)
	}
	if !rets[4].GetBool() {
		b.Fatalf("object not stamped")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.call()
	}
}

func BenchmarkNative_Reflect_ObjectInfo_TypeName8B(b *testing.B)    { benchObjectInfoTypeName(b, 8) }
func BenchmarkNative_Reflect_ObjectInfo_TypeName64KiB(b *testing.B) { benchObjectInfoTypeName(b, 64<<10) }
func BenchmarkNative_Reflect_ObjectInfo_TypeName512KiB(b *testing.B) {
	benchObjectInfoTypeName(b, 512<<10)
}
