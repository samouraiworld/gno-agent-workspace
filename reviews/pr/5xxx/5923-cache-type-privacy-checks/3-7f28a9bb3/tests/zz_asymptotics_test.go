// Growth of assertTypeIsPublic when the fast path cannot short-circuit.
//
// The head's assertTypeIsPublic calls typeHasPrivateDep once per node it
// visits, and typeHasPrivateDep re-walks that node's whole reachable
// closure from scratch (computeTypeHasPrivateDep consults no cache for
// intermediate nodes). When the walk cannot be skipped and cannot panic
// early — i.e. the only private package reached is the saving realm's own
// — the head does N closure walks where the merge base did one.
//
// Run: from a local clone of gnolang/gno, in two worktrees, one at the
// merge base ddb752cacbfb49df327e4dbf9cc9ec748facd781 and one at the PR
// head, drop this file into gnovm/pkg/gnolang/ in both and run:
//
//	go test ./gnovm/pkg/gnolang/ -run TestWalkGrowth -v 2>&1 | grep N=
//
// The shape is a chain of N public declared types whose deepest field is
// a struct declared in the saving realm's own (private) package: every
// node's verdict is true, so nothing is skipped, and pkgPath == rlm.Path
// at the only private node, so nothing panics.
package gnolang

import (
	"testing"
	"time"
)

func buildChain(pubPath, privPath string, n int) Type {
	var cur Type = &StructType{
		PkgPath: privPath,
		Fields:  []FieldType{{Name: "Leaf", Type: IntType}},
	}
	for i := n - 1; i >= 0; i-- {
		d := &DeclaredType{PkgPath: pubPath, Name: Name("T" + itoaLocal(i))}
		d.Base = &StructType{PkgPath: pubPath, Fields: []FieldType{{Name: "Next", Type: cur}}}
		cur = d
	}
	return cur
}

func itoaLocal(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func TestWalkGrowth(t *testing.T) {
	const pub = "gno.land/p/chain"
	const priv = "gno.land/r/owner"

	newStore := func() Store {
		st := NewStore(nil, nil, nil)
		st.SetCachePackage(&PackageValue{PkgPath: pub, Private: false})
		st.SetCachePackage(&PackageValue{PkgPath: priv, Private: true})
		return st
	}

	for _, n := range []int{50, 100, 200, 400, 800} {
		rlm := NewRealm(priv) // saving realm owns the private package: no panic
		root := buildChain(pub, priv, n)
		// One untimed call so lazily-built TypeID strings are warm.
		rlm.assertTypeIsPublic(newStore(), root, map[TypeID]struct{}{})

		const reps = 20
		// A fresh store per call: this is the FIRST save of this type in
		// a process, the only state the merge base ever has.
		stores := make([]Store, reps)
		for i := range stores {
			stores[i] = newStore()
		}
		start := time.Now()
		for i := range reps {
			rlm.assertTypeIsPublic(stores[i], root, map[TypeID]struct{}{})
		}
		d := time.Since(start) / reps
		t.Logf("N=%4d  per-cold-call=%v", n, d)
	}
}
