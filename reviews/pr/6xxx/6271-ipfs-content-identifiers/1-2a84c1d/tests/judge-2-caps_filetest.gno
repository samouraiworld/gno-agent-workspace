// Judge filetest for candidate 2 of gnolang/gno PR 6271 at 2a84c1dcd: a Gas
// golden pinning the len(s) == 46 cap in Parse and the maxCIDLen truncation in
// DecodeFirst. Gas 760183 at the head, 37477045 with the cap relaxed to
// len(s) >= 2, 1027783 without the truncation. Copy into filetests/, gno test.
package main

import (
	"strings"

	cid "gno.land/p/omarsy/cid/v0"
)

func main() {
	_, err := cid.Parse("Qm" + strings.Repeat("z", 510))
	println(err)
	_, n, _ := cid.DecodeFirst(make([]byte, 1<<16))
	println(n)
}

// Output:
// cid: invalid multibase string
// 0

// Gas:
// 760183
