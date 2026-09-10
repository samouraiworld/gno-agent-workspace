/* Run: from a gno checkout:
gh pr checkout 6137 -R gnolang/gno && git checkout 861df2c42
curl -fsSL -o gnovm/tests/files/defer_multivalue_arg_shapes.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6137-doopdefer-value-stack/1-861df2c42/tests/defer_multivalue_arg_shapes.gno
go test -v -run 'TestFiles/defer_multivalue_arg_shapes.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/defer_multivalue_arg_shapes.gno
*/

// Asserts the three `defer f(g())` spread shapes the PR's own filetest leaves
// out: a variadic callee, a bound method and a builtin. Each panics with
// "defer called a nil function" at merge base bc35e978a, and passes at 861df2c42.

package main

type counter struct{ n int }

func (c counter) add(a, b int) { println("method:", c.n+a+b) }

func two() (int, int) { return 7, 8 }

func variadic(a int, rest ...int) { println("variadic:", a, len(rest), rest[0]) }

func main() {
	defer println(two())           // runs 3rd: builtin callee
	defer counter{n: 1}.add(two()) // runs 2nd: bound method callee
	defer variadic(two())          // runs 1st: variadic callee
	println("body")
}

// Output:
// body
// variadic: 7 1 8
// method: 16
// 7 8
