# Candidate find-b1-2: docs/resources/gno-interrealm-v2.md:737 — `realmA.PublicCrossing(cross)` does not compile

The diff rewrites this MsgRun example (adds `import "chain/runtime/unsafe"`,
swaps `runtime.*` for `unsafe.*`) so the block reads as a runnable sample, but
the surviving call `realmA.PublicCrossing(cross)` passes the bare `cross`
builtin where a `realm` value is required. In current Gno the crossing call is
`PublicCrossing(cross(cur))` (2776 uses of `cross(cur)` in `examples/`, zero
non-comment uses of bare `(cross)`); the repo's own syntax is visible in
`examples/gno.land/r/demo/disperse/filetests/z_4_filetest.gno:21`. `gno test`
refuses the block.

Head block (`gno-interrealm-v2.md:729-738`):

```go
import (
    "chain/runtime/unsafe"
    "gno.land/r/realmA"
)

func main() {
    unsafe.PreviousRealm()
    unsafe.CurrentRealm()

    realmA.PublicNoncrossing()    // runs inside ephemeral, no boundary
    realmA.PublicCrossing(cross)  // crosses into realmA
}
```

## Check

```bash
# from a local clone of gnolang/gno, at the PR head 7f309e625:
git fetch origin pull/6230/head && git checkout FETCH_HEAD

# stage the example as a realm under examples/ with a stub target:
cp -r examples /tmp/examples-finder && cd /tmp/examples-finder
mkdir -p gno.land/r/realma gno.land/e/g1user/run
printf 'module = "gno.land/r/realma"\ngno = "0.9"\n' > gno.land/r/realma/gnomod.toml
cat > gno.land/r/realma/realma.gno <<'EOF'
package realma
func PublicNoncrossing() {}
func PublicCrossing(cur realm) {}
EOF
printf 'module = "gno.land/e/g1user/run"\ngno = "0.9"\n' > gno.land/e/g1user/run/gnomod.toml
cat > gno.land/e/g1user/run/run.gno <<'EOF'
package main

import (
	"chain/runtime/unsafe"

	"gno.land/r/realma"
)

func main() {
	unsafe.PreviousRealm()
	unsafe.CurrentRealm()
	realma.PublicNoncrossing()
	realma.PublicCrossing(cross)
}
EOF
gno test ./gno.land/r/realma
gno test ./gno.land/e/g1user/run
```

## Observed

```
ok      ./gno.land/r/realma
gno.land/e/g1user/run/run.gno:14:24: cannot use cross (value of type func(rlm realm) realm) as realma.realm value in argument to realma.PublicCrossing: func(rlm realm) realm does not implement gno0p9.realm (missing method Address) (code=gnoTypeCheckError)
FAIL    ./gno.land/e/g1user/run
```

The same shape appears in `gno-interrealm.md` at lines 227, 932 and 972.
