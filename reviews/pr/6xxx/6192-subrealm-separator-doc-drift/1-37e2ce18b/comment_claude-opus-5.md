# Review: [#6192](https://github.com/gnolang/gno/pull/6192)
Verdict: COMMENT, on one defect the branch documents rather than causes: the registry key `NewToken`'s godoc now points at is built from the registering frame's raw pkgpath, so one realm holds a separate `grc20reg` entry per `realm.Sub` identity and a sub-created token is refused by its own host frame. Every comment the branch rewrites matches the code at this sha.
Event: COMMENT
Model: claude-opus-5, quick review
Commit: 37e2ce18b
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6192 37e2ce18b`
Round: 1. One agent as finder, judge and writer over 19 changed lines, 6 candidates, the Warning run from scratch as two filetests and the other five settled by reading.

## Body
Comment-only: the five files carry no statement change.

## examples/gno.land/p/nt/grc20/v0/token.gno:33-34 [gh](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc20/v0/token.gno#L33-L34) · [↗](../../../../../.worktrees/gno-review-6192/examples/gno.land/p/nt/grc20/v0/token.gno#L33) · Warning
Related: `Register` derives its key from the registering frame's raw pkgpath while [`origRealm` strips the `realm.Sub` subpath](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc20/v0/token.gno#L60-L64), so a host frame cannot register what its own sub identity created.

<details><summary>repro</summary>

Both files go in `examples/gno.land/p/nt/grc20/v0/filetests/` and record what the registry does today. The first is the other direction of the same split: one realm registers `host#a.SREG`, `host#b.SREG` and `host.SREG` as three entries, so the one-token-per-realm-and-symbol relation `Register` documents holds only for a realm that never calls `realm.Sub`. The second is the host frame being refused.

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6192 -R gnolang/gno

cat > examples/gno.land/p/nt/grc20/v0/filetests/subreg_dup_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/demo/grc20subdup

package grc20subdup

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
)

func main(cur realm) {
	sa := cur.Sub("a")
	sb := cur.Sub("b")
	a, _ := grc20.NewToken("SubA", "SREG", 4, 0, sa)
	b, _ := grc20.NewToken("SubB", "SREG", 4, 1, sb)
	c, _ := grc20.NewToken("Host", "SREG", 4, 2, cur)
	println("key a:", grc20reg.Register(cross(sa), a, ""))
	println("key b:", grc20reg.Register(cross(sb), b, ""))
	println("key c:", grc20reg.Register(cross(cur), c, ""))
	println("lookup by (realm, symbol):", grc20reg.Get("gno.land/r/demo/grc20subdup.SREG") != nil)
	println("lookup of the sub entry:", grc20reg.Get("gno.land/r/demo/grc20subdup#a.SREG") != nil)
}

// Output:
// key a: gno.land/r/demo/grc20subdup#a.SREG
// key b: gno.land/r/demo/grc20subdup#b.SREG
// key c: gno.land/r/demo/grc20subdup.SREG
// lookup by (realm, symbol): true
// lookup of the sub entry: true
GNO

cat > examples/gno.land/p/nt/grc20/v0/filetests/subreg_host_filetest.gno <<'GNO'
// PKGPATH: gno.land/r/demo/grc20subhost

package grc20subhost

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
)

func main(cur realm) {
	tok, _ := grc20.NewToken("SubHost", "SHOS", 4, 0, cur.Sub("vault"))
	println("id:", tok.ID())
	println("key:", grc20reg.Register(cross(cur), tok, ""))
}

// Output:
// id: gno.land/r/demo/grc20subhost#vault.SHOS.0000000

// Error:
// grc20reg: token must be registered from its own realm
GNO

cd examples && gno test -v ./gno.land/p/nt/grc20/v0
```

```text
=== RUN   ./gno.land/p/nt/grc20/v0/subreg_dup_filetest.gno
--- PASS: ./gno.land/p/nt/grc20/v0/subreg_dup_filetest.gno (elapsed: 0.01s, gas: 1722572, storage: gno.land/r/demo/grc20subdup:+8188b gno.land/r/nt/grc20reg/v0:+5599b)
=== RUN   ./gno.land/p/nt/grc20/v0/subreg_host_filetest.gno
--- PASS: ./gno.land/p/nt/grc20/v0/subreg_host_filetest.gno (elapsed: 0.01s, gas: 928278)
ok      ./gno.land/p/nt/grc20/v0 	5.46s
```

`Register` takes `rlmPath` from `cur.Previous().PkgPath()`, which carries the `"#sub"` the VM synthesizes in [`Sub`](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/gnovm/pkg/gnolang/uverse.go#L1732), and keys with `fqname.Construct(rlmPath, symbol)`. Nothing strips the subpath on that path, while `guardHome` and `origRealm` both strip it, so the registry is the one consumer of `Token.ID()` that treats a sub identity as a separate realm. The prefix guard then compares that key against an id built from the creation frame, which is why the host and its sub disagree in both directions.

</details>

## SKIP examples/gno.land/p/nt/grc721/v0/token.gno:13 [gh](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc721/v0/token.gno#L13) · [↗](../../../../../.worktrees/gno-review-6192/examples/gno.land/p/nt/grc721/v0/token.gno#L13) · Nit
Nit: the exported godoc says the caller's `PkgPath` becomes `origRealm`, which [the construction comment 27 lines down](https://github.com/gnolang/gno/blob/37e2ce18bbf40a60d6b52456621c5b6c324c8d6e/examples/gno.land/p/nt/grc721/v0/token.gno#L40-L45) contradicts by cutting any `realm.Sub` subpath off first.

Skipped: a code comment's own wording, which changes no behaviour.
