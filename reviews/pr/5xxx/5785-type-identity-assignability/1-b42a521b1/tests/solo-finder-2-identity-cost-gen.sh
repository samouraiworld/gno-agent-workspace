#!/usr/bin/env bash
# solo-finder-2: generates the gno packages for the end-to-end assertion-cost run.
# Usage: bash solo-finder-2-identity-cost-gen.sh <outdir> <outer> <inner>
#        cd <outdir>/sfcost && gno test -v -print-runtime-metrics .
# outer=1 inner=1 is the control; outer=20 inner=100 and outer=100 inner=100
# gave the timings in solo-finder-2-identity-cost_test.go at b42a521b1.
set -euo pipefail
out=$1/sfcost; mkdir -p "$out"; in=""; f=""
for j in $(seq 0 $(($3-1))); do in="$in G$j int;"; done
for i in $(seq 0 $(($2-1))); do f="$f F$i struct{$in};"; done
printf 'module = "gno.land/p/sf2/sfcost"\ngno = "0.9"\n' > "$out/gnomod.toml"
cat > "$out/a.gno" <<GNO
package sfcost

type I interface{ M(s struct{$f}) }

type T struct{}

func (T) M(s struct{$f}) {}

func Loop(n int) int {
	var x any = T{}
	c := 0
	for i := 0; i < n; i++ {
		if _, ok := x.(I); ok {
			c++
		}
	}
	return c
}
GNO
cat > "$out/a_test.gno" <<'GNO'
package sfcost

import "testing"

func TestLoop(t *testing.T) {
	if Loop(20000) != 20000 {
		t.Fatal("bad")
	}
}
GNO
