/* Run: from a gno checkout:
gh pr checkout 5707 -R gnolang/gno && git checkout c7b8227f2
# (save this file at gnovm/tests/files/var_initorder_methodexpr_xpkg.gno)
go test -v -run 'TestFiles/var_initorder_methodexpr_xpkg.gno$' ./gnovm/pkg/gnolang/
*/

// run

// Method expression on a type from an imported package.
// The new VPField branch must NOT register a fake dep "T.GetB" in main —
// the cross-package guard (dt.PkgPath != pn.PkgPath) drops it.
// If broken, resolveDeclDep panics trying to find type T in package main.
package main

import "filetests/extern/initorder_xpkgmethod"

// Method expression on imported type, called with explicit receiver.
var dummy = initorder_xpkgmethod.T.GetB(initorder_xpkgmethod.T{})

func main() {
	println(dummy)
}

// Output:
// 42
