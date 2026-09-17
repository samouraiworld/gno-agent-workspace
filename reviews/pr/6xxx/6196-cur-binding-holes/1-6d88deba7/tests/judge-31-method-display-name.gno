/* Run: from a gno checkout:
gh pr checkout 6196 -R gnolang/gno && git checkout 6d88deba7
curl -fsSL -o gnovm/tests/files/judge31_methodname.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6196-cur-binding-holes/1-6d88deba7/tests/judge-31-method-display-name.gno
go test -v -run 'TestFiles/judge31_methodname.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/judge31_methodname.gno
*/

// PKGPATH: gno.land/r/demo/curambig
package curambig

// Asserts what the identity panic names when two types in one realm carry a
// crossing method of the same name. funcDisplayName's named arm is
// PkgPath + "." + Name (op_call.go:568-570), so the message says `.target`
// for both; the gno stacktrace beside it names the receiver and the line.
// Passes at the reviewed head.

type A struct{}
type B struct{}

func (a *A) target(cur realm) { println("A:", cur.PkgPath()) }
func (b *B) target(cur realm) { println("B:", cur.PkgPath()) }

func (b *B) exec(cur realm, cb func()) { cb() }

func main(cur realm) {
	b := &B{}
	cb := func() { b.target(cur) }
	b.exec(cross(cur), cb) // cb carries main's cur into B.target's frame
}

// IS:     the name has no receiver, and the stacktrace supplies `b.target` at line 29
// SHOULD: unchanged — a receiver-bearing arm copied from debugger.go:809 renders
//         gno.land/r/demo/curambig.(*gno.land/r/demo/curambig.B).target, a doubled path

// Error:
// crossing function gno.land/r/demo/curambig.target was entered without cross(), so it takes its caller's identity — but the value passed is not the caller's own cur (a stale capture from another frame, a sibling frame, or a rebound cur). Pass the caller's own `cur` unchanged, or use cross(cur) to enter as this realm
