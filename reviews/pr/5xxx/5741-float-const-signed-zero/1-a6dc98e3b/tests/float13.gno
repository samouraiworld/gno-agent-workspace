/* Run: from a gno checkout:
gh pr checkout 5741 -R gnolang/gno && git checkout a6dc98e3b
curl -fsSL -o gnovm/tests/files/float13.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5741-float-const-signed-zero/1-a6dc98e3b/tests/float13.gno
go test -v -run 'TestFiles/float13.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/float13.gno
*/
// Mirror of float12 with a negative constant: float32(big) where big
// overflows float32 on the negative side. At a6dc98e3b the VM prints -Inf;
// Go and the go/types parity checker both reject it. Asserts the post-fix
// reject; fails at a6dc98e3b, passes once the narrowing check also rejects
// values below -MaxFloat32.

package main

const big float64 = -1e39

func main() {
	x := float32(big)
	println(x)
}

// Error:
// main/float13.gno:17:7-19: cannot convert constant of type Float64Kind to Float32Kind

// TypeCheckError:
// main/float13.gno:17:15: cannot convert big (constant -1e+39 of type float64) to type float32
