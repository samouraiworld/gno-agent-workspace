/* Run: from a gno checkout:
gh pr checkout 6020 -R gnolang/gno && git checkout 764ac4d84
curl -fsSL -o gnovm/tests/files/map55.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6020-compute-map-keys-once/1-764ac4d84/tests/map55.gno
go test -v -run 'TestFiles/map55.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/map55.gno
*/
package main

import "math"

// The stored key is replaced by the last one assigned, so the second write
// leaves a negative zero behind where the first left a positive one. A
// negative zero has to come from math.Copysign: the constant -0.0 is +0,
// as gnovm/tests/files/float9.gno documents.
// Green at 764ac4d84; deleting `mli.Key = key` in GetPointerForKey turns it
// red while map51.gno stays green.

func main() {
	negZero := math.Copysign(0, -1)
	m := map[float64]string{}
	m[0] = "positive"
	m[negZero] = "negative"
	for k, v := range m {
		println(len(m), math.Signbit(k), v)
	}
}

// Output:
// 1 true negative
