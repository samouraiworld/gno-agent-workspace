/* Run: from a gno checkout:
gh pr checkout 5871 -R gnolang/gno && git checkout d8aa8541a
curl -fsSL -o examples/gno.land/r/docs/complexargs/z_sanitize_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5871-revive-correct-rdocs/2-d8aa8541a/tests/z_sanitize_filetest.gno
cd examples && go run ../gnovm/cmd/gno test ./gno.land/r/docs/complexargs/
rm gno.land/r/docs/complexargs/z_sanitize_filetest.gno
*/

// Asserts the code span survives a backtick in the value. Passes at d8aa8541a,
// fails on the ufmt.Sprintf plus InlineText line it replaced.

// PKGPATH: gno.land/r/docs/complexargs/z_sanitize_filetest

package z_sanitize_filetest

import (
	"strings"

	"gno.land/r/docs/complexargs"
)

// SetMyObject is unguarded on purpose, so any account sets Name through
// MsgRun. A backtick in the name must not close the code span early.
func main(cur realm) {
	obj := complexargs.NewCustomType("x`y", []int{1})
	complexargs.SetMyObject(cross(cur), obj)
	out := complexargs.Render("")
	println(out[strings.LastIndex(out, "Value of myObject: "):])
}

// Output:
// Value of myObject: ``CustomType{Name: x`y, Numbers: 1}``
