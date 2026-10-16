#!/usr/bin/env bash
# gnolang/gno#6209: loop shapes that COMPILE on the merge base with a wrong
# binding (an outer plain variable of the same name), and run differently at
# head. The PR says the range / type-switch / range-assign cases are "valid Go
# rejected on master" and "No gas change"; with an outer plain variable they
# are accepted on master and bind that variable instead.
#
# Repro from a plain clone (each run compiles the gnolang test binary):
#   git clone https://github.com/gnolang/gno && cd gno
#   bash <this file> bbd9b2ffe87c9a295ccfd6dfb45224bb02c82d8a   # merge base: 4 of 4 FAIL with the outputs below
#   bash <this file> 1de7f587da469abad9ce4cf5fa945f3328aa0e40   # head: 4 of 4 PASS
#
# Measured 2026-10-16 (go test -short -run 'TestFiles/zz_sf1' ./gnovm/pkg/gnolang):
#   shape                         expected (go run)   base actual             head
#   zz_sf1_shadow_between         5 6                 0 1 2 3 4 5 6           PASS
#   zz_sf1_range_outer_plain      0 0 1               0..99 twice (200 lines) PASS
#   zz_sf1_typeswitch_outer_plain int 1               other outer             PASS
#   zz_sf1_range_assign_outer_plain 1 -1              0 1                     PASS
#
# gno.land never persists BlockNodes (gnovm/pkg/gnolang/store.go:162-163): every
# node boot re-preprocesses every stored package (gno.land/pkg/sdk/vm/keeper.go:203,
# PreprocessAllFilesAndSaveBlockNodes), so a realm deployed with one of these
# shapes switches binding at the first boot on the new binary, and a node still
# on the old binary computes the old result for the same transaction.
set -euo pipefail
sha=${1:?usage: $0 <sha>}
git checkout --quiet --detach "$sha"
d=gnovm/tests/files
cat > $d/zz_sf1_shadow_between.gno <<'EOF'
package main

func main() {
	for x := 0; x < 1; x++ {
		x := 5
		for x := x; x < 7; x++ {
			println(x)
		}
	}
}

// Output:
// 5
// 6
EOF
cat > $d/zz_sf1_range_outer_plain.gno <<'EOF'
package main

func main() {
	x := 100
	_ = x
	for x := 1; x < 3; x++ {
		for x := range make([]int, x) {
			println(x)
		}
	}
}

// Output:
// 0
// 0
// 1
EOF
cat > $d/zz_sf1_typeswitch_outer_plain.gno <<'EOF'
package main

func main() {
	var x interface{} = "outer"
	_ = x
	for x := interface{}(1); x != nil; x = nil {
		switch x := x.(type) {
		case int:
			println("int", x)
		default:
			println("other", x)
		}
	}
}

// Output:
// int 1
EOF
cat > $d/zz_sf1_range_assign_outer_plain.gno <<'EOF'
package main

func main() {
	k := -1
	for k := 0; k < 1; k++ {
		for k = range []int{7, 8} {
		}
		println(k)
	}
	println(k)
}

// Output:
// 1
// -1
EOF
trap 'rm -f $d/zz_sf1_*.gno' EXIT
(cd gnovm/pkg/gnolang && go test -short -v -count=1 -run 'TestFiles/zz_sf1' . 2>&1 | grep -E -- '--- (PASS|FAIL)|^ok|^FAIL|^\s+[-+@ ][^-+]' || true)
