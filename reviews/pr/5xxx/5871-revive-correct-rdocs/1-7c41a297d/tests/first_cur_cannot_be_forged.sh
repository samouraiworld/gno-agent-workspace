#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# Takes the `current-guard` "vulnerable" fixture merged by PR 5835 verbatim, deploys it
# as a real realm, and walks every route to its first `cur`. Nine routes cannot carry a
# non-current realm value that far; four arrive, and all four read IsCurrent() == true.
# Measured at 7c41a297d.
#
# Run: from a gno checkout:
#   gh pr checkout 5871 -R gnolang/gno && git checkout 7c41a297d
#   bash first_cur_cannot_be_forged.sh
set -u
cd "$(git rev-parse --show-toplevel)"
GNOROOT=$PWD; export GNOROOT
GNO=${GNO:-$(command -v gno)}
A=examples/gno.land/r/zzprobe/admin
B=examples/gno.land/r/zzprobe/attacker
trap 'rm -rf "$GNOROOT/examples/gno.land/r/zzprobe"' EXIT
rm -rf examples/gno.land/r/zzprobe
mkdir -p "$A" "$B"

printf 'module = "gno.land/r/zzprobe/admin"\ngno = "0.9"\n'    > "$A/gnomod.toml"
printf 'module = "gno.land/r/zzprobe/attacker"\ngno = "0.9"\n' > "$B/gnomod.toml"

# The merged fixture, unchanged, plus a probe reporting what a guard would have seen.
cp misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno "$A/"
cat >> "$A/admin.gno" <<'EOF'

func Probe(cur realm) bool { return cur.IsCurrent() }
EOF

# lint <n> <label> — the route is refused before it runs; print the compiler's own line.
lint() {
	printf '%-4s %-46s ' "$1" "$2"
	(cd examples && GNOROOT=$GNOROOT "$GNO" lint ./gno.land/r/zzprobe/...) 2>&1 |
		grep -m1 -oE '(not enough arguments[^,]*|cannot use .*|cross argument must be [^(]*|only .* are allowed as the first argument [a-z ]*)' |
		cut -c1-120
	echo
}

# exec <n> <label> <main body> — the route compiles; run it and print panic or probe.
exec_route() {
	cat > "$B/z_run_filetest.gno" <<EOF
// PKGPATH: gno.land/r/zzprobe/runx
package runx

import (
	"gno.land/r/zzprobe/admin"
	"gno.land/r/zzprobe/attacker"
)

var _ = admin.Owner
var _ = attacker.Nop

func main(cur realm) {
$3
}
EOF
	printf '%-4s %-46s ' "$1" "$2"
	(cd examples && GNOROOT=$GNOROOT "$GNO" test -v ./gno.land/r/zzprobe/attacker/) 2>&1 |
		grep -m1 -oE '(cannot persist realm value[^"]*|cross: rlm is not the current cur[^"]*|probe: (true|false))' |
		cut -c1-120
	echo
}

echo "== refused at type-check"
cat > "$B/a.gno" <<'EOF'
package attacker

import (
	"chain/runtime/unsafe"

	"gno.land/r/zzprobe/admin"
)

func Nop() {}

func Go(cur realm) { r := unsafe.CurrentRealm(); admin.TransferOwnership(cross(r), address("g1x")) }
EOF
lint 1 "cross(name) = unsafe.CurrentRealm()"

cat > "$B/a.gno" <<'EOF'
package attacker

import (
	"chain/runtime/unsafe"

	"gno.land/r/zzprobe/admin"
)

func Nop() {}

func Go(cur realm) { r := unsafe.PreviousRealm(); admin.TransferOwnership(cross(r), address("g1x")) }
EOF
lint 2 "cross(name) = unsafe.PreviousRealm()"

cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Render(path string) string { admin.TransferOwnership(address("g1x")); return "" }
EOF
lint 3 "no realm value in scope, from Render"

echo
echo "== refused at preprocess"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Go(cur realm) { admin.TransferOwnership(cross(cur.Previous()), address("g1x")) }
EOF
lint 4 "cross() on a derived expression"

cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Go(cur realm) { p := cur.Previous(); admin.TransferOwnership(p, address("g1x")) }
EOF
lint 5 "derived realm, no cross()"

echo
echo "== refused at runtime, inside cross()"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Go(cur realm) { p := cur.Previous(); admin.TransferOwnership(cross(p), address("g1x")) }
EOF
exec_route 6 "derived realm bound to a name" '	attacker.Go(cross(cur))'

cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Go(cur realm) { launder(0, cur.Previous()) }

func launder(_ int, rlm realm) { admin.TransferOwnership(cross(rlm), address("g1x")) }
EOF
exec_route 7 "laundered through a non-crossing helper" '	attacker.Go(cross(cur))'

echo
echo "== refused at finalize"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

var saved realm

func Nop() {}

func Stash(cur realm) { saved = cur }

func Go(cur realm) { println("probe:", admin.Probe(cross(saved))) }
EOF
exec_route 8 "package var holding a realm" '	attacker.Stash(cross(cur))
	attacker.Go(cross(cur))'

cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

var hook func() bool

func Nop() {}

func Stash(cur realm) { hook = func() bool { return admin.Probe(cross(cur)) } }

func Go(cur realm) { println("probe:", hook()) }
EOF
exec_route 9 "closure capturing cur, kept in state" '	attacker.Stash(cross(cur))
	attacker.Go(cross(cur))'

echo
echo "== arrives; what an IsCurrent() guard would have read"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Nop() {}

func Crossed(cur realm) bool { return admin.Probe(cross(cur)) }

func Named(cur realm) bool { r := cur; return admin.Probe(cross(r)) }
EOF
exec_route 10 "cross(cur) from a user transaction" '	println("probe:", admin.Probe(cross(cur)))'
exec_route 11 "cross(cur) realm to realm" '	println("probe:", attacker.Crossed(cross(cur)))'
exec_route 12 "cross(name) where name = cur" '	println("probe:", attacker.Named(cross(cur)))'

cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func init(cur realm) { println("probe:", admin.Probe(cross(cur))) }

func Nop() {}
EOF
exec_route 13 "init(cur realm)" '	attacker.Nop()'
