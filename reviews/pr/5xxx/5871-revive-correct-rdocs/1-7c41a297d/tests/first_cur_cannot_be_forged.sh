#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# Takes the `current-guard` "vulnerable" fixture merged by PR 5835 verbatim, deploys
# it as a real realm, and tries every route to reaching it with a first `cur` whose
# IsCurrent() is false. Two routes are refused by the preprocessor, two by cross()
# at runtime. Measured at 7c41a297d.
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
mkdir -p "$A" "$B"

printf 'module = "gno.land/r/zzprobe/admin"\ngno = "0.9"\n'    > "$A/gnomod.toml"
printf 'module = "gno.land/r/zzprobe/attacker"\ngno = "0.9"\n' > "$B/gnomod.toml"

# The merged fixture, unchanged: examples are at misc/audit-pattern-harness/
# fixtures/current-guard/vulnerable, plus a probe reporting what a guard would see.
cat > "$A/admin.gno" <<'EOF'
package admin

var owner = address("g1qz8e0fz3y0pl9y4dq9d7c5dwnyu6qf04hs7z0a")

func TransferOwnership(cur realm, next address) {
	if cur.Previous().Address() != owner {
		panic("owner only")
	}
	owner = next
}

func Owner() address { return owner }

func Probe(cur realm) bool { return cur.IsCurrent() }
EOF

echo "### A. cross() on a derived realm expression"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Attack(cur realm, next address) {
	admin.TransferOwnership(cross(cur.Previous()), next)
}
EOF
(cd examples && "$GNO" lint ./gno.land/r/zzprobe/...) 2>&1 | sed 's/^.*gno:/  a.gno:/' | head -2

echo "### B. no cross() at all"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Attack(cur realm, next address) {
	p := cur.Previous()
	admin.TransferOwnership(p, next)
}
EOF
(cd examples && "$GNO" lint ./gno.land/r/zzprobe/...) 2>&1 | sed 's/^.*gno:/  a.gno:/' | head -2

echo "### C. bind the derived realm to a name, then cross() it"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Attack(cur realm, next address) {
	p := cur.Previous()
	admin.TransferOwnership(cross(p), next)
}
EOF
cat > "$B/z_run_filetest.gno" <<'EOF'
// PKGPATH: gno.land/r/zzprobe/runx
package runx

import (
	"gno.land/r/zzprobe/admin"
	"gno.land/r/zzprobe/attacker"
)

func main(cur realm) {
	attacker.Attack(cross(cur), address("g1evilevilevilevilevilevilevilevilevil0"))
	println("owner:", admin.Owner().String())
}
EOF
(cd examples && "$GNO" test ./gno.land/r/zzprobe/attacker/) 2>&1 | grep -m1 -E 'panic|ok ' | sed 's/^/  /'

echo "### D. launder the derived realm through a non-crossing helper"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func Attack(cur realm, next address) {
	launder(0, cur.Previous(), next)
}

func launder(_ int, rlm realm, next address) {
	admin.TransferOwnership(cross(rlm), next)
}
EOF
(cd examples && "$GNO" test ./gno.land/r/zzprobe/attacker/) 2>&1 | grep -m1 -E 'panic|ok ' | sed 's/^/  /'

echo "### E. what an IsCurrent() guard would have seen on the paths that do arrive"
cat > "$B/a.gno" <<'EOF'
package attacker

import "gno.land/r/zzprobe/admin"

func ProbeCrossed(cur realm) bool { return admin.Probe(cross(cur)) }
EOF
cat > "$B/z_run_filetest.gno" <<'EOF'
// PKGPATH: gno.land/r/zzprobe/runp
package runp

import (
	"gno.land/r/zzprobe/admin"
	"gno.land/r/zzprobe/attacker"
)

func main(cur realm) {
	println("  user tx -> realm  :", admin.Probe(cross(cur)))
	println("  realm   -> realm  :", attacker.ProbeCrossed(cross(cur)))
}

// Output:
//   user tx -> realm  : true
//   realm   -> realm  : true
EOF
(cd examples && "$GNO" test -v ./gno.land/r/zzprobe/attacker/) 2>&1 | grep -E 'user tx|realm   ->|PASS|FAIL' | sed 's/^/  /'
