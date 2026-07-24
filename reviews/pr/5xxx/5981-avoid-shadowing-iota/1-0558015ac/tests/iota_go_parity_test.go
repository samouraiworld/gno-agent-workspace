/* Run: from a gno checkout:
gh pr checkout 5981 -R gnolang/gno && git checkout 0558015ac
curl -fsSL -o gnovm/pkg/gnolang/iota_go_parity_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5981-avoid-shadowing-iota/1-0558015ac/tests/iota_go_parity_test.go
go test -v -run 'TestIotaShadowingGoParity' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/iota_go_parity_test.go
*/

// Records what the Go compiler does with each iota binding form, so the gno
// side of the same table can be compared against it. The wantGo column is the
// assertion; the gno column is prose, produced by running the same source
// through StaticBlock.Reserve at 0558015ac.
// Go accepts every binding form here; gno rejects all of them except the
// three-clause for init, which the loopvar rename hides from Reserve.

package gnolang

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestIotaShadowingGoParity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		src    string
		wantGo bool   // Go compiler accepts the program
		gno    string // observed at 0558015ac
	}{
		{
			name: "short var define",
			src: `package main
func main() { iota := 5; println(iota) }`,
			wantGo: true,
			gno:    "rejected: builtin identifiers cannot be shadowed",
		},
		{
			name: "for init three clause",
			src: `package main
func main() { for iota := 0; iota < 2; iota++ { println(iota) } }`,
			wantGo: true,
			gno:    "accepted, prints 0 and 1",
		},
		{
			name: "range key",
			src: `package main
func main() { s := []int{1}; for iota := range s { println(iota) } }`,
			wantGo: true,
			gno:    "rejected: builtin identifiers cannot be shadowed",
		},
		{
			name: "parameter, referenced",
			src: `package main
func f(iota int) int { return iota }
func main() { println(f(3)) }`,
			wantGo: true,
			gno:    "rejected: builtin identifiers cannot be shadowed",
		},
		{
			name: "parameter, never referenced",
			src: `package main
func f(iota int) { println("hi") }
func main() { f(3) }`,
			wantGo: true,
			gno:    "rejected at 0558015ac, accepted on master",
		},
		{
			name: "named result, never referenced",
			src: `package main
func f() (iota int) { return 5 }
func main() { println(f()) }`,
			wantGo: true,
			gno:    "rejected at 0558015ac, accepted on master",
		},
		{
			name: "method receiver, never referenced",
			src: `package main
type T int
func (iota T) M() { println("m") }
func main() { var t T; t.M() }`,
			wantGo: true,
			gno:    "rejected at 0558015ac, accepted on master",
		},
		{
			name: "type switch guard",
			src: `package main
func main() { var x interface{} = 5; switch iota := x.(type) { case int: println(iota) } }`,
			wantGo: true,
			gno:    "rejected: builtin identifiers cannot be shadowed",
		},
		{
			name: "package level var",
			src: `package main
var iota = 5
func main() { println(iota) }`,
			wantGo: true,
			gno:    "rejected on master and at 0558015ac",
		},
		{
			name: "struct field",
			src: `package main
type T struct{ iota int }
func main() { t := T{iota: 9}; println(t.iota) }`,
			wantGo: true,
			gno:    "accepted on master and at 0558015ac",
		},
		{
			name: "nested const blocks",
			src: `package main
func main() {
	const ( A = iota; B )
	{
		const ( C = iota; D )
		println(A, B, C, D)
	}
}`,
			wantGo: true,
			gno:    "accepted, prints 0 1 0 1, same as Go",
		},
		{
			name: "package level var iota then const block using iota",
			src: `package main
var iota = 5
const ( A = iota; B )
func main() { println(A, B) }`,
			wantGo: false, // iota is a variable here, so not a constant
			gno:    "rejected at the var declaration, before the const block is reached",
		},
		{
			name: "parameter named len",
			src: `package main
func f(len int) int { return len }
func main() { println(f(3)) }`,
			wantGo: true,
			gno:    "accepted on master and at 0558015ac",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "main.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			conf := types.Config{Importer: importer.Default()}
			_, err = conf.Check("main", fset, []*ast.File{f}, nil)
			gotGo := err == nil
			if gotGo != tc.wantGo {
				t.Fatalf("Go accepts = %v, want %v (err: %v)", gotGo, tc.wantGo, err)
			}
			t.Logf("go accepts=%v; gno: %s", gotGo, tc.gno)
		})
	}
}
