# Candidate find-b1-1: docs/resources/effective-gno.md:748 — `mySafeObject` is undefined

The PR rewrites the registration example to thread `cur` through
`NewSafeStruct(cur)` but leaves the very next line calling
`otherrealm.Register(mySafeObject)`; the local is named `mySafeObj`. Code copied
from the block fails to type-check, which is the exact class of defect the PR is
supposed to remove ("Code copied from those docs would not compile").

Block as it stands at 7f309e625 (effective-gno.md:743-749):

```go
import "gno.land/r/otherrealm"

func init(cur realm) {
	mySafeObj := NewSafeStruct(cur)
	otherrealm.Register(mySafeObject)
}
```

## Check (run in the head worktree)

`gno test` on the extracted package; `gno lint` prints type errors but always
exits 0, so read `gno test`'s status and output.

```bash
# from a local clone of gnolang/gno, at the PR head:
git fetch origin pull/6230/head && git checkout FETCH_HEAD

mkdir -p /tmp/docprobe/gno.land/r/docprobe
printf 'module = "gno.land/r/docprobe"\ngno = "0.9"\n' \
  > /tmp/docprobe/gno.land/r/docprobe/gnomod.toml
cat > /tmp/docprobe/gno.land/r/docprobe/gnomod.toml <<'EOF'
module = "gno.land/r/docprobe"
gno = "0.9"
EOF
cat > /tmp/docprobe/gno.land/r/docprobe/probe.gno <<'EOF'
package docprobe

type MySafeStruct struct {
	counter int
	admin   address
}

func NewSafeStruct(cur realm) *MySafeStruct {
	caller := cur.Previous().Address()
	return &MySafeStruct{counter: 0, admin: caller}
}

func init(cur realm) {
	mySafeObj := NewSafeStruct(cur)
	otherrealm.Register(mySafeObject)
}
EOF
# (the module is copied under examples/ so the gno module resolver finds it;
#  the run used a copy of the repo's examples/ tree)
cd examples && gno test ./gno.land/r/docprobe
```

## Observed

```
gno.land/r/docprobe/probe.gno:17:22: undefined: mySafeObject (code=gnoTypeCheckError)
gno.land/r/docprobe/probe.gno:16:2: declared and not used: mySafeObj (code=gnoTypeCheckError)
FAIL
```

Same failure reproduced from the exact snippet extracted at
`docs/resources/effective-gno.md:748` (probe run at head 7f309e625).
