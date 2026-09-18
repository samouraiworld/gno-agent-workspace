#!/usr/bin/env bash
# Does `make fix` reach misc/gnopreview, the module CI runs `go fix` against?
#
# misc/gnopreview/go.mod makes misc/gnopreview its own module, and
# .github/workflows/ci-dir-misc.yml adds it to the matrix that calls
# _ci-go.yml, whose lint job runs
#
#     GOTOOLCHAIN=go1.26.1 CGO_ENABLED=1 go fix -omitzero=false -diff ./...
#
# in the module and, on any output, prints
#
#     ::error::'go fix' would modify files. Run 'make fix' from the repo root
#     and commit the result.
#
# The root Makefile's `fix` recipe covers the root module plus a hardcoded list
# of nested ones, with the comment "The misc/ entries must track the
# ci-dir-misc.yml matrix". misc/gnopreview/ is not in that list, and a nested
# module is outside the root `go fix ./...` pattern, so the command CI names
# cannot fix the module CI checks.
#
# Repro from a plain clone:
#
#   git clone https://github.com/gnolang/gno.git
#   cd gno
#   git fetch origin pull/6194/head
#   git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash reviews/.../b6-lines-reach-catalog-make-fix-skips-gnopreview.sh
#
# Needs network the first time: GOTOOLCHAIN=go1.26.1 downloads that toolchain.
# Exits 1 while the gap is open, 0 once misc/gnopreview/ joins the Makefile
# loop. Run from the repository root.

set -u

fail=0
sentinel=misc/gnopreview/zz_sentinel.go
cleanup() { rm -f "$sentinel"; }
trap cleanup EXIT

[ -f go.mod ] && [ -d misc/gnopreview ] || { echo "run from the repo root"; exit 2; }

echo "== 1. CI runs go fix against misc/gnopreview =="
if grep -q '^ *- gnopreview$' .github/workflows/ci-dir-misc.yml; then
  echo "   ci-dir-misc.yml matrix: gnopreview listed"
else
  echo "   ci-dir-misc.yml matrix: gnopreview NOT listed (premise gone)"; exit 2
fi

echo "== 2. the root fix target's module list =="
sed -n '/^	@for d in /p' Makefile
if grep -q 'misc/gnopreview/' Makefile; then
  echo "   misc/gnopreview/ IS listed -- gap closed"
else
  echo "   misc/gnopreview/ is NOT listed"; fail=1
fi

echo "== 3. the root module cannot see the nested package =="
n=$(go list ./misc/... 2>/dev/null | grep -c gnopreview)
echo "   go list ./misc/... packages under gnopreview: $n"
[ "$n" -eq 0 ] || echo "   (nested module visible -- root ./... would cover it)"

cat > "$sentinel" <<'EOF'
package main

func sentinelAny(v interface{}) interface{} { return v }

func sentinelLoop(n int) int {
	t := 0
	for i := 0; i < n; i++ {
		t += i
	}
	return t
}
EOF

echo "== 4. what \`make fix\` would rewrite (root pass, scoped to ./misc/...) =="
root_hits=$(GOTOOLCHAIN=go1.26.1 CGO_ENABLED=1 go fix -omitzero=false -diff ./misc/... 2>&1 | grep -c sentinel)
echo "   sentinel lines in the root pass: $root_hits"

echo "== 5. what CI runs, in the module =="
ci_out=$(cd misc/gnopreview && GOTOOLCHAIN=go1.26.1 CGO_ENABLED=1 go fix -omitzero=false -diff ./... 2>&1)
ci_hits=$(printf '%s\n' "$ci_out" | grep -c sentinel)
printf '%s\n' "$ci_out" | sed -n '1,12p' | sed 's/^/   /'
echo "   sentinel lines in the CI pass: $ci_hits"

echo
if [ "$root_hits" -eq 0 ] && [ "$ci_hits" -gt 0 ]; then
  echo "GAP: CI's go fix check fails on misc/gnopreview and \`make fix\` leaves it untouched;"
  echo "     the error message sends the contributor to a target that does nothing for it."
  fail=1
else
  echo "no gap: both passes agree on misc/gnopreview"
fi
exit $fail
