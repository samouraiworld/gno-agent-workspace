// Equivalence harness for the DeriveObjectIDCryptoAddr rewrite in
// refactor_pass.patch. From a plain clone of github.com/gnolang/gno:
//
//	gh pr checkout 6139 -R gnolang/gno
//	git apply refactor_pass.patch
//	cp refactor_derive_equiv_test.go gnovm/pkg/gnolang/zz_derive_equiv_test.go
//	go test ./gnovm/pkg/gnolang/ -run TestZZDeriveEquiv -count=1 -v
//	rm gnovm/pkg/gnolang/zz_derive_equiv_test.go
//	git checkout -- gnovm/pkg/gnolang/misc.go
//
// Prints the shipped form beside the three-guard form over the four cells of
// (PkgID zero?, NewTime zero?). Both panic on the same three and answer the same
// address on the fourth, which is the golden vector for foo20 at NewTime 1.

package gnolang

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
)

func TestZZDeriveEquiv(t *testing.T) {
	pid := PkgIDFromPkgPath("gno.land/r/demo/foo20")
	var zero PkgID
	for _, c := range []struct {
		name string
		oid  ObjectID
	}{
		{"both zero", ObjectID{}},
		{"pkgID zero, newTime 7", ObjectID{PkgID: zero, NewTime: 7}},
		{"pkgID set, newTime 0", ObjectID{PkgID: pid}},
		{"both set (newTime 1)", ObjectID{PkgID: pid, NewTime: 1}},
	} {
		fmt.Printf("%-22s old=%-42s new=%s\n", c.name,
			zzRun(zzThreeGuardForm, c.oid), zzRun(DeriveObjectIDCryptoAddr, c.oid))
	}
}

func zzRun(f func(ObjectID) crypto.Address, oid ObjectID) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("panic(%v)", r)
		}
	}()
	return f(oid).String()
}

// zzThreeGuardForm is the shape at the reviewed head.
func zzThreeGuardForm(objectID ObjectID) crypto.Address {
	if objectID.IsZero() {
		panic("objectID cannot be zero")
	}
	if objectID.PkgID.IsZero() {
		panic("pkgID cannot be zero")
	}
	if objectID.NewTime == 0 {
		panic("newTime cannot be zero")
	}
	return crypto.AddressFromPreimage([]byte("objectid:" + objectID.PkgID.String() + ":" + strconv.FormatUint(objectID.NewTime, 10)))
}
