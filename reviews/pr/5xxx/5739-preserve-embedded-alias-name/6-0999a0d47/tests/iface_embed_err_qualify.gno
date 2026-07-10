/* Run: from a gno checkout:
gh pr checkout 5739 -R gnolang/gno && git checkout 0999a0d47
mkdir -p gnovm/tests/files/extern/secptr
curl -fsSL -o gnovm/tests/files/extern/secptr/secptr.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/6-0999a0d47/tests/secptr.gno
curl -fsSL -o gnovm/tests/files/iface_embed_err_qualify.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5739-preserve-embedded-alias-name/6-0999a0d47/tests/iface_embed_err_qualify.gno
go test -v -run 'TestFiles/iface_embed_err_qualify.gno$' ./gnovm/pkg/gnolang/
rm -r gnovm/tests/files/iface_embed_err_qualify.gno gnovm/tests/files/extern/secptr
*/

package main

import "filetests/extern/secptr"

// VerifyImplementedBy qualifies a stamped cross-package method in its
// missing-method error, but not in its pointer-receiver or wrong-type errors.
// At 0999a0d47 the panic reads "(method sec has pointer receiver)" while the
// interface printed beside it reads "filetests/extern/secptr.sec".
// Qualifying those two error sites the same way turns this green.

// T carries two distinct unexported sec methods: its own main.sec, and the
// stamped secptr.sec promoted from *PtrImpl. Only the secptr one needs a
// pointer receiver, so a T value fails the pointer-receiver check on it.
type T struct{ secptr.PtrImpl }

func (T) sec() int { return 1 }

func main() {
	var _ interface {
		secptr.Sec
		sec() int
	} = T{}
	println("unreachable")
}

// Error:
// main/iface_embed_err_qualify.gno:30:6-33:9: main.T does not implement interface {filetests/extern/secptr.sec func() int; sec func() int} (method filetests/extern/secptr.sec has pointer receiver)

// TypeCheckError:
// main/iface_embed_err_qualify.gno:33:6: cannot use T{} (value of struct type T) as interface{sec() int; secptr.Sec} value in variable declaration: T does not implement interface{sec() int; secptr.Sec} (method sec has pointer receiver)
