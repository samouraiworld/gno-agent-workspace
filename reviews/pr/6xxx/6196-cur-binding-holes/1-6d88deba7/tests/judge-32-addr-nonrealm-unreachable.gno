// A non-realm binding spelled `cur` inside a crossing function's own block
// does not type-check, at the reviewed head or at the merge base: the shape
// the address-of arm of candidate 32 needs cannot exist in compilable source.
//
// Measured 2026-09-18, same output at both shas:
//   zrealm_cur_nonrealm_probe.gno:5:6: cur redeclared in this block;
//   zrealm_cur_nonrealm_probe.gno:4:19: other declaration of cur
//
// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   cp <this file> gnovm/tests/files/zrealm_cur_nonrealm_probe.gno
//   git checkout 6d88deba71165b165641b65541259d00aba20377
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nonrealm_probe.gno$'
//   git checkout b7845fb2da9a21bfb895e1c008f016776f8342d0
//   go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_cur_nonrealm_probe.gno$'
//   rm gnovm/tests/files/zrealm_cur_nonrealm_probe.gno
//
// A crossing function's first realm parameter must be spelled `cur`
// (preprocess.go:2828), the one carve-out being the synthesized `.arg` name a
// blank takes, and the type checker names that blank parameter `cur` too, so a
// body-level `var cur <non-realm>` collides with it either way. The address-of
// rule at preprocess.go:2583 therefore never gets a non-realm value to refuse
// in source that compiles.

// PKGPATH: gno.land/r/demo/curnonrealm
package curnonrealm

func main(_ realm) {
	var cur int = 42
	println(cur)
}

// Output:
// 42
