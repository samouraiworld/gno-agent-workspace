// Fixture package for iface_embed_err_qualify.gno.
// Install at gnovm/tests/files/extern/secptr/secptr.gno.
// The stock ifaceext fixture only has a value receiver, so it cannot reach
// VerifyImplementedBy's pointer-receiver error path.
package secptr

// Sec is sealed: sec is unexported, so only secptr can supply it.
type Sec interface{ sec() int }

// PtrImpl satisfies Sec only through a pointer.
type PtrImpl struct{}

func (*PtrImpl) sec() int { return 9 }
