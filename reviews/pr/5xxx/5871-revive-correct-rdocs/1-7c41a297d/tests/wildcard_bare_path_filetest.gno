/* Run: from a gno checkout:
gh pr checkout 5871 -R gnolang/gno && git checkout 7c41a297d
curl -fsSL -o examples/gno.land/r/docs/routing/zz_wildcard_filetest.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5871-revive-correct-rdocs/1-7c41a297d/tests/wildcard_bare_path_filetest.gno
cd examples && gno test ./gno.land/r/docs/routing/
rm gno.land/r/docs/routing/zz_wildcard_filetest.gno
*/

// Asserts that /r/docs/routing:wildcard reaches wildcardHandler with an empty
// tail. It aborts instead, at mux.Router.Render indexing reqParts[1] on a
// one-segment path. Fails at 7c41a297d; passes once the realm also registers
// the bare "wildcard" pattern.

package main

import "gno.land/r/docs/routing"

func main() {
	println(routing.Render("wildcard"))
}

// IS at 7c41a297d — the reviewed head aborts before printing anything:
//
//	// Error:
//	// runtime error: slice index out of bounds: 1 (len=1)

// SHOULD, and does once router.HandleFunc("wildcard", wildcardHandler) is
// registered ahead of the "wildcard/*" pattern:

// Output:
// # Wildcard handler
//
// This page matched a wildcard route.
//
// **Wildcard path:** ``
//
// You can use this to match and handle arbitrarily nested paths.
//
// Try visiting:
// - [/r/docs/routing:wildcard/foo/bar](/r/docs/routing:wildcard/foo/bar)
// - [/r/docs/routing:wildcard/a/b/c](/r/docs/routing:wildcard/a/b/c)
//
// ### Wildcard segments:
// - Segment 1: ``
