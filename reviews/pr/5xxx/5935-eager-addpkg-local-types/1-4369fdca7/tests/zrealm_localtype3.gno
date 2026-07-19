/* Run: from a gno checkout:
gh pr checkout 5935 -R gnolang/gno && git checkout 4369fdca7
curl -fsSL -o gnovm/tests/files/zrealm_localtype3.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5935-eager-addpkg-local-types/1-4369fdca7/tests/zrealm_localtype3.gno
go test -tags debugAssert -v -run 'TestFiles/zrealm_localtype3.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zrealm_localtype3.gno
*/

// A file-level var initializer and an init function both hold a value of a
// function-local type before saveNewPackageValuesAndTypes finalizes the realm.
// Passes at 4369fdca7; moving saveFuncLocalTypes below the FinalizeRealmTransaction
// block panics "dangling function-local type ref ... in persisted value", which no
// committed test detects.

// PKGPATH: gno.land/r/test
package test

type T struct{ x int }

func (t T) Get() int { return t.x }

type I interface{ Get() int }

func mk() I {
	type S struct{ T }
	return S{T{3}}
}

// Held by a file-level var initializer, so the value exists before the realm
// finalization inside saveNewPackageValuesAndTypes.
var X = mk()

var Y I

// Held by init, which runs after saveNewPackageValuesAndTypes and is persisted
// by resavePackageValues.
func init() {
	type Q struct{ T }
	Y = Q{T{4}}
}

func main(cur realm) {
	println(X.Get(), Y.Get())
}

// Output:
// 3 4
