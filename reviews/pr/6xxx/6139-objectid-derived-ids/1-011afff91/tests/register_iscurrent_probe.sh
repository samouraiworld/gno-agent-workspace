#!/usr/bin/env bash
# Probe, from a plain clone of github.com/gnolang/gno:
#
#	gh pr checkout 6139 -R gnolang/gno
#	bash register_iscurrent_probe.sh
#
# Inserts the IsCurrent() guard asked for on grc20reg.Register at the top of
# Register, then tries every route that could reach Register with a realm that
# is not the current cur. The guard never fires: the ordinary call reports
# IsCurrent() true and each hostile route is stopped before the body runs.
# Restores grc20reg.gno on exit.
set -u

ROOT=$(git rev-parse --show-toplevel)
REG=$ROOT/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno
P=$ROOT/examples/gno.land/r/demo/regprobe
trap 'git -C "$ROOT" checkout -- "$REG"; rm -rf "$P"' EXIT

python3 - "$REG" <<'PY'
import sys, pathlib
p = pathlib.Path(sys.argv[1])
s = p.read_text()
old = 'func Register(cur realm, token *grc20.Token, slug string) string {\n\tif token == nil {'
new = ('func Register(cur realm, token *grc20.Token, slug string) string {\n'
       '\tif !cur.IsCurrent() {\n\t\tpanic("grc20reg: PROBE-GUARD-FIRED")\n\t}\n\n'
       '\tif token == nil {')
assert old in s, 'Register signature moved'
p.write_text(s.replace(old, new))
PY

mkdir -p "$P"
printf 'module = "gno.land/r/demo/regprobe"\ngno = "0.9"\n' > "$P/gnomod.toml"

head='// PKGPATH: gno.land/r/demo/regprobe
package regprobe

import (
	"gno.land/p/demo/tokens/grc20"
	"gno.land/r/demo/defi/grc20reg"
)
'

# The ordinary call the doc comment prescribes.
cat > /tmp/probe_a.gno <<EOF
$head
func main(cur realm) {
	token, _ := grc20.NewToken("Probe", "PRB", 4, cur)
	println("key:", grc20reg.Register(cross(cur), token, "probe"))
	println("cur.IsCurrent():", cur.IsCurrent())
}
EOF

# A captured realm handed straight to a crossing function.
cat > /tmp/probe_b.gno <<EOF
$head
func main(cur realm) {
	token, _ := grc20.NewToken("Probe", "PRB", 4, cur)
	captured := cur
	println(grc20reg.Register(captured, token, "probe"))
}
EOF

# A realm parked in a package var, so it outlives its frame.
cat > /tmp/probe_c.gno <<EOF
$head
var held realm

func main(cur realm) {
	token, _ := grc20.NewToken("Probe", "PRB", 4, cur)
	held = cur
	println(grc20reg.Register(cross(held), token, "probe"))
}
EOF

# cross() over a realm value from an adjacent frame.
cat > /tmp/probe_d.gno <<EOF
$head
func main(cur realm) {
	token, _ := grc20.NewToken("Probe", "PRB", 4, cur)
	stale := cur.Previous()
	println(grc20reg.Register(cross(stale), token, "probe"))
}
EOF

# A non-crossing call into another realm's crossing function.
cat > /tmp/probe_e.gno <<EOF
$head
func main(cur realm) {
	token, _ := grc20.NewToken("Probe", "PRB", 4, cur)
	println(grc20reg.Register(cur, token, "probe"))
}
EOF

for n in a b c d e; do
	cp /tmp/probe_$n.gno "$P/probe_${n}_filetest.gno"
	printf '%s ' "$n:"
	(cd "$P" && GNOROOT=$ROOT go run ../../../../../gnovm/cmd/gno test -v . 2>&1) |
		grep -oE 'unexpected panic: .*|cur\.IsCurrent\(\): .*|key: .*' | awk '!seen[$0]++'
	rm "$P/probe_${n}_filetest.gno"
done
rm -f /tmp/probe_[a-e].gno
