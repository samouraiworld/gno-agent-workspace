package gnolang

import "testing"

var sinkS string

func BenchmarkDerivePkgBech32Addr(b *testing.B) {
	for b.Loop() {
		sinkS = string(DerivePkgBech32Addr("gno.land/r/demo/objectid"))
	}
}

func BenchmarkObjectIDDerivePath(b *testing.B) {
	oid := ObjectID{PkgID: PkgIDFromPkgPath("gno.land/r/demo/objectid"), NewTime: 7}
	for b.Loop() {
		sinkS = oid.DerivePath()
	}
}
