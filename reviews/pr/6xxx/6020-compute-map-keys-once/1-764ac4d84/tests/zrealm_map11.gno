/* Run: from a gno checkout:
gh pr checkout 6020 -R gnolang/gno && git checkout 764ac4d84
curl -fsSL -o gnovm/tests/files/zrealm_map11.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6020-compute-map-keys-once/1-764ac4d84/tests/zrealm_map11.gno
go test -v -run 'TestFiles/zrealm_map11.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/zrealm_map11.gno
*/
// PKGPATH: gno.land/r/map_iface_key
package map_iface_key

// An interface-keyed map read back from the store: the lazy index build and
// every later probe have to agree on keeping the TypeID prefix, or int(1) and
// int64(1) merge into one entry during the build and every probe misses.
// Green at 764ac4d84. zrealm_map6.gno covers the same round trip for a
// concrete key type; no filetest covers it for an interface key type.

var m map[any]string

func init() {
	m = map[any]string{}
	m[int(1)] = "int"
	m[int64(1)] = "int64"
	m["k"] = "string"
}

func main(cur realm) {
	println(len(m))
	println(m[int(1)])
	println(m[int64(1)])
	println(m["k"])
	m[int32(1)] = "int32"
	println(len(m))
	delete(m, int64(1))
	_, ok := m[int64(1)]
	println(len(m), ok)
}

// Output:
// 3
// int
// int64
// string
// 4
// 3 false
