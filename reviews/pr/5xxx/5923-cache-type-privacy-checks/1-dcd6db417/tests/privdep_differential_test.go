/* Run: from a gno checkout:
gh pr checkout 5923 -R gnolang/gno && git checkout dcd6db417
curl -fsSL -o gnovm/pkg/gnolang/privdep_differential_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/1-dcd6db417/tests/privdep_differential_test.go
go test -v -run 'TestPrivateDepDifferential' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privdep_differential_test.go
*/

// Random type graphs over public and private packages, queried repeatedly
// against a shared set of type objects so the memo accumulates. Green at
// dcd6db417: no walk order commits an answer the cache-free reference
// disagrees with.
package gnolang

import (
	"math/rand"
	"strconv"
	"testing"
)

// Differential check: typeHasPrivateDep, run repeatedly over a shared set
// of type objects so the memo accumulates, must always agree with a
// cache-free reachability walk over the same graph.
func TestPrivateDepDifferential(t *testing.T) {
	for seed := range 5000 {
		rng := rand.New(rand.NewSource(int64(seed)))
		store := NewStore(nil, nil, nil)

		const numPkgs = 4
		privatePkg := map[string]bool{}
		pkgs := make([]string, numPkgs)
		for i := range pkgs {
			pkgs[i] = "gno.land/r/p" + strconv.Itoa(i)
			priv := rng.Intn(3) == 0
			privatePkg[pkgs[i]] = priv
			store.SetCachePackage(&PackageValue{PkgPath: pkgs[i], Private: priv})
		}

		// Random declared types, each with a struct base whose fields point
		// at other declared types (cycles included) or scalars.
		n := 3 + rng.Intn(6)
		dts := make([]*DeclaredType, n)
		for i := range dts {
			dts[i] = &DeclaredType{PkgPath: pkgs[rng.Intn(numPkgs)], Name: Name("T" + strconv.Itoa(i))}
		}
		for i, dt := range dts {
			nf := 1 + rng.Intn(4)
			fields := make([]FieldType, nf)
			for f := range fields {
				var ft Type
				switch rng.Intn(4) {
				case 0:
					ft = IntType
				case 1:
					ft = &PointerType{Elt: dts[rng.Intn(n)]}
				case 2:
					ft = &SliceType{Elt: dts[rng.Intn(n)]}
				default:
					ft = dts[(i+1+rng.Intn(n-1))%n]
				}
				fields[f] = FieldType{Name: Name("F" + strconv.Itoa(f)), Type: ft}
			}
			dt.Base = &StructType{PkgPath: dt.PkgPath, Fields: fields}
			if rng.Intn(2) == 0 {
				mft := &FuncType{
					Params:  []FieldType{{Name: "recv", Type: dt}, {Name: "a", Type: dts[rng.Intn(n)]}},
					Results: []FieldType{{Name: "", Type: &PointerType{Elt: dts[rng.Intn(n)]}}},
				}
				dt.Methods = []TypedValue{{T: mft, V: &FuncValue{Type: mft, Name: "M"}}}
			}
		}

		// Query every type several times, in shuffled order, so cached
		// answers from earlier queries feed later ones.
		for round := range 4 {
			order := rng.Perm(n)
			for _, i := range order {
				got := typeHasPrivateDep(store, dts[i])
				want := reachesPrivate(dts[i], privatePkg, map[Type]bool{})
				if got != want {
					t.Fatalf("seed %d round %d type %s: typeHasPrivateDep=%v want %v",
						seed, round, dts[i].Name, got, want)
				}
			}
		}
	}
}

// Cache-free reference: does any node reachable from t sit in a private
// package?
func reachesPrivate(t Type, private map[string]bool, seen map[Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	pkgPath, children := typePkgPathAndChildren(t)
	if pkgPath != "" && private[pkgPath] {
		return true
	}
	for _, c := range children {
		if reachesPrivate(c, private, seen) {
			return true
		}
	}
	return false
}
