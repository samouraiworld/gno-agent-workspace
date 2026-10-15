#!/usr/bin/env bash
# b2-reach: chain/reflect.objectInfo is priced flat (native_gas.go: Base 1200,
# SizeFlat; row comment "nothing here scales with caller input"), yet
# declaredTypeName concatenates the caller's own type name on every call.
# Two realm filetests differ only in the declared type's name, 8 bytes vs
# 60000 bytes (the filetest directive scanner caps a line near 64 KiB; the chain
# itself caps only by MaxTxBytes, 1_000_000). Each is run with the loop at 0 and
# at 20000 iterations of reflect.Of(p); the per-iteration gas and wall time are
# the deltas. The Gas directive is left at 1 so the failure prints actual gas.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6214/head && git checkout --detach 58b2692067d6545410b99c71f33bdf506caadaa0
#   bash <this file>        # from the clone root; writes 4 filetests, runs, removes them
#
# Measured 2026-10-07 (AMD 7840HS, noisy box), gas exact, wall over two runs:
#   short: 12,495 gas/iter, 9.1-16.5 us/iter  (~0.7-1.3 ns/gas), three runs
#   60 KB: 22,235 gas/iter, 50.4-79.7 us/iter  (~2.3-3.6 ns/gas), three runs
#   marginal: +9,740 gas (allocation gas for the string) buys +59-63 us, ~6 ns/gas
set -euo pipefail
F=$PWD/gnovm/tests/files
gen() {
  local name="T$(head -c "$1" /dev/zero | tr '\0' 'A')" iters=$2 out=$3
  cat >"$out" <<EOF
// PKGPATH: gno.land/r/demo/b2reach
package b2reach

import "chain/reflect"

type $name struct{ x int }

var p = &$name{}

func main(cur realm) {
	n := 0
	for i := 0; i < $iters; i++ {
		o, ok := reflect.Of(p)
		if ok {
			n += len(o.Type())
		}
	}
	println(n)
}

// Output:
// 0

// Gas:
// 1
EOF
}
gen 7 0 $F/zrealm_b2reach_short0.gno
gen 7 20000 $F/zrealm_b2reach_short20000.gno
gen 59999 0 $F/zrealm_b2reach_big0.gno
gen 59999 20000 $F/zrealm_b2reach_big20000.gno
trap 'rm -f $F/zrealm_b2reach_*.gno' EXIT
(cd gnovm/pkg/gnolang && go test -c -o /tmp/b2reach-gnolang.test .)
cd gnovm/pkg/gnolang
for f in short0 short20000 big0 big20000; do
  s=$(date +%s%N)
  gas=$(/tmp/b2reach-gnolang.test -test.run "TestFiles/zrealm_b2reach_$f.gno" -test.short -test.v 2>&1 |
    grep -A6 'Gas diff' | grep '^\s*+[0-9]' | tr -d ' +' || true)
  e=$(date +%s%N)
  echo "$f gas=$gas wall_ms=$(((e - s) / 1000000))"
done
