/* Run: from a gno checkout, at 7f28a9bb3 and at the merge base ddb752cac:
gh pr checkout 5923 -R gnolang/gno && git checkout 7f28a9bb3
curl -fsSL -o gnovm/pkg/gnolang/privatepath_cost_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/3-7f28a9bb3/tests/privatepath_cost_test.go
go test -run '^$' -bench BenchmarkPrivateChain -benchmem ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privatepath_cost_test.go
*/

// A realm using its own private types is exempt by pkgPath == rlm.Path, so
// its verdict is true and the fast path never fires. assertTypeIsPublic
// then recurses, and every node it reaches calls typeHasPrivateDep, which
// runs a fresh full walk from that node: depth d costs d walks instead of
// one. Uses only symbols that exist at both shas.

package gnolang

import (
	"strconv"
	"testing"
)

func buildPrivateChain(store Store, pkgPath string, depth int) Type {
	store.SetCachePackage(&PackageValue{PkgPath: pkgPath, Private: true})
	var t Type = &StructType{
		PkgPath: pkgPath,
		Fields:  []FieldType{{Name: "Leaf", Type: IntType}},
	}
	for i := range depth {
		t = &StructType{
			PkgPath: pkgPath,
			Fields:  []FieldType{{Name: Name("F" + strconv.Itoa(i)), Type: t}},
		}
	}
	return t
}

func benchPrivateChain(b *testing.B, depth int) {
	b.Helper()
	pkgPath := "gno.land/r/bench/priv"
	rlm := NewRealm(pkgPath)
	for b.Loop() {
		b.StopTimer()
		// Fresh store per iteration: the first save of this type in a
		// process, which is the cost the cache cannot remove.
		store := NewStore(nil, nil, nil)
		t := buildPrivateChain(store, pkgPath, depth)
		b.StartTimer()
		rlm.assertTypeIsPublic(store, t, map[TypeID]struct{}{})
	}
}

func BenchmarkPrivateChain8(b *testing.B)   { benchPrivateChain(b, 8) }
func BenchmarkPrivateChain32(b *testing.B)  { benchPrivateChain(b, 32) }
func BenchmarkPrivateChain128(b *testing.B) { benchPrivateChain(b, 128) }
