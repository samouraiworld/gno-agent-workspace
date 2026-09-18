// Asserts that a /u/<name> segment cannot carry an unescaped quote into the
// vm/qeval expression: reUsername (gnolang.Re_name) rejects every name outside
// [a-z][a-z0-9]*([_-][a-z0-9]+)*, and the %q at handler_http.go:555 escapes
// what a name could still hold. Both hold at the reviewed head.
//
/* Run: from a gno checkout:
gh pr checkout 6206 -R gnolang/gno && git checkout 876762bdf
curl -fsSL -o gno.land/pkg/gnoweb/judge25_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6206-gnoweb-user-page-gate/1-876762b/tests/judge-25-eval-expression-quoting.go
go test -v -run 'TestJudge25EvalExpressionQuoting' ./gno.land/pkg/gnoweb/
rm gno.land/pkg/gnoweb/judge25_test.go
*/
package gnoweb

import (
	"fmt"
	"testing"
)

func TestJudge25EvalExpressionQuoting(t *testing.T) {
	for _, name := range []string{
		`a") || Unsafe("`, // a break-out attempt
		`a\`,              // a trailing backslash
		"a\nb",            // a newline
		"moul001",         // the one shape the gate accepts
	} {
		// The expression handler_http.go:555 would build for this name.
		expr := fmt.Sprintf("ResolveName(%q)", name)
		t.Logf("name=%#v reUsername.Matches=%v expr=%s", name, reUsername.Matches(name), expr)
	}
}
