# Candidate find-b1-3: docs/resources/gno-interrealm.md:932-935 — MsgCall example does not compile

The diff adds `import "chain/runtime/unsafe"` and swaps `runtime.*` for
`unsafe.*` in this MsgCall sample, but the block still passes the bare `cross`
builtin (`AnotherPublic(cross)`) and an undefined `cur` (`AnotherPublic(cur)`,
while the parameter is `_ realm`). `gno test` refuses both.

Head block (`gno-interrealm.md:925-936`):

```go
// PKGPATH: gno.land/r/test/test

import "chain/runtime/unsafe"

func Public(_ realm) {
    unsafe.PreviousRealm()
    unsafe.CurrentRealm()

    // Call a crossing function of same realm with crossing
    AnotherPublic(cross)

    // Call a crossing function of same realm without crossing
    AnotherPublic(cur)
}
```

## Check

```bash
# from a local clone of gnolang/gno, at the PR head 7f309e625:
git fetch origin pull/6230/head && git checkout FETCH_HEAD

cp -r examples /tmp/examples-finder && cd /tmp/examples-finder
mkdir -p gno.land/r/msgcallprobe
printf 'module = "gno.land/r/msgcallprobe"\ngno = "0.9"\n' > gno.land/r/msgcallprobe/gnomod.toml
cat > gno.land/r/msgcallprobe/probe.gno <<'EOF'
package msgcallprobe

import "chain/runtime/unsafe"

func Public(_ realm) {
	unsafe.PreviousRealm()
	unsafe.CurrentRealm()
	AnotherPublic(cross)
	AnotherPublic(cur)
}

func AnotherPublic(_ realm) {}
EOF
gno test ./gno.land/r/msgcallprobe
```

## Observed

```
gno.land/r/msgcallprobe/probe.gno:8:16: cannot use cross (value of type func(rlm realm) realm) as realm value in argument to AnotherPublic: func(rlm realm) realm does not implement gno0p9.realm (missing method Address) (code=gnoTypeCheckError)
gno.land/r/msgcallprobe/probe.gno:9:16: undefined: cur (code=gnoTypeCheckError)
FAIL
```

The same bare-`cross` shape sits at `gno-interrealm.md:227` (`SendMail(cross, ...)`)
and `gno-interrealm.md:972` (`realmA.PublicCrossing(cross)`).
