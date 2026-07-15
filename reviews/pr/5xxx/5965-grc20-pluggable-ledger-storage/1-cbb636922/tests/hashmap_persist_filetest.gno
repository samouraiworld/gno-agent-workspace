/* Run: from a gno checkout:
gh pr checkout 5965 -R gnolang/gno && git checkout cbb636922
mkdir -p examples/gno.land/p/nt/hashmap/v0/filetests
curl -fsSL -o examples/gno.land/p/nt/hashmap/v0/filetests/hashmap_persist_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5965-grc20-pluggable-ledger-storage/1-cbb636922/tests/hashmap_persist_filetest.gno
(cd examples/gno.land/p/nt/hashmap/v0 && gno test -v .)
rm -r examples/gno.land/p/nt/hashmap/v0/filetests
*/
// Exercises the serialize-to-store path no committed hashmap test touches: a
// Map written in init() must reload correctly in main() across the persist
// boundary. Passes at cbb636922 (storage line confirms the realm was persisted);
// guards the single-object-persistence property against a future refactor.

// PKGPATH: gno.land/r/demo/hmtest
package hmtest

import "gno.land/p/nt/hashmap/v0"

var m *hashmap.Map

func init() {
	m = hashmap.New()
	m.Set("alice", int64(1000))
	m.Set("bob", int64(500))
}

func main(cur realm) {
	println("alice:", m.Get("alice").(int64))
	println("bob:", m.Get("bob").(int64))
	println("size:", m.Size())
	m.Set("alice", int64(1234))
	m.Remove("bob")
	println("alice-after:", m.Get("alice").(int64))
	println("has-bob-after:", m.Has("bob"))
	println("size-after:", m.Size())
}

// Output:
// alice: 1000
// bob: 500
// size: 2
// alice-after: 1234
// has-bob-after: false
// size-after: 1
