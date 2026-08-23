// Accept/reject matrix for type-declaration shapes, run on the merge-base
// and on the PR head. A row that moves is a consensus divergence: an
// unupgraded node and an upgraded one disagree on whether the package is
// valid. Measured at 093c32be0: 9 rows move REJECT -> ACCEPT, 0 move
// ACCEPT -> REJECT. Results in accept_base.txt / accept_head.txt.
//
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/pkg/gnolang/zz_accept_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/consensus/accept_boundary_test.go
go test -count=1 -v -run TestZZAcceptBoundary ./gnovm/pkg/gnolang/ | grep ZZACCEPT > /tmp/head.txt
git checkout $(git merge-base origin/master HEAD) -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
go test -count=1 -v -run TestZZAcceptBoundary ./gnovm/pkg/gnolang/ | grep ZZACCEPT > /tmp/base.txt
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
diff /tmp/base.txt /tmp/head.txt
rm gnovm/pkg/gnolang/zz_accept_test.go
*/

package gnolang

import (
	"fmt"
	"testing"
)

// TestZZAcceptBoundary runs a battery of type-declaration shapes and
// prints ACCEPT or REJECT for each, so the two trees can be diffed for
// any program whose accept/reject verdict moved.
func TestZZAcceptBoundary(t *testing.T) {
	cases := []struct{ name, src string }{
		{"plain-struct", `type S struct{ A int }`},
		{"derived", `type S struct{ A int }
type D S`},
		{"self-recursive", `type N struct{ V int; Next *N }`},
		{"mutual-ptr", `type T1 struct{ Next *T2; Val int }
type T2 T1`},
		{"mutual-ptr-swapped", `type T2 T1
type T1 struct{ Next *T2; Val int }`},
		{"mutual-value-illegal", `type T1 struct{ Self T2; Val int }
type T2 T1`},
		{"mutual-slice", `type T1 []*T2
type T2 T1`},
		{"mutual-map", `type T1 map[string]*T2
type T2 T1`},
		{"mutual-array", `type T1 [2]*T2
type T2 T1`},
		{"mutual-func", `type T1 func(*T2) int
type T2 T1`},
		{"mutual-iface", `type T1 interface{ M() *T2 }
type T2 T1`},
		{"mutual-ptrbase", `type T0 struct{ X int }
type T1 *T0
type T2 T1`},
		{"chan-decl", `type C chan int`},
		{"chan-derived", `type C chan int
type D C`},
		{"alias", `type S struct{ A int }
type A = S`},
		{"alias-then-derived", `type S struct{ A int }
type A = S
type D A`},
		{"uverse-derived", `type E error`},
		{"prim-derived", `type I int
type J I`},
		{"self-cycle-illegal", `type T T`},
		{"two-cycle-illegal", `type A B
type B A`},
		{"embed-mutual", `type T1 struct{ Next *T2; Val int }
type T2 T1
type T3 struct{ T2 }`},
		{"chain3", `type T1 struct{ Next *T2; Val int }
type T2 T1
type T3 T2`},
		{"threeway", `type A struct{ B *B; V int }
type B struct{ C *C; V int }
type C A`},
		{"local-scope-mutual", `func F() int {
	type T1 struct{ Next *T2; Val int }
	type T2 T1
	var a T1
	a.Next = &T2{Val: 3}
	return a.Next.Val
}`},
	}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("ZZACCEPT\t%s\tREJECT\t%v\n", c.name, firstLine(fmt.Sprint(r)))
					return
				}
			}()
			m := NewMachine("testdata", nil)
			defer m.Release()
			nn := m.MustParseFile("testdata.gno", "package testdata\n"+c.src+"\n")
			m.RunFiles(nn)
			fmt.Printf("ZZACCEPT\t%s\tACCEPT\n", c.name)
		}()
	}
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			return s[:i]
		}
	}
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
