#!/usr/bin/env bash
# The dot-import branch in tryPredefine is never reached: initStaticBlocks2 rejects
# every dot import first, so a sentinel in tryPredefine leaves import2.gno green,
# a sentinel in initStaticBlocks2 turns it red, and the base's preprocess.go
# (capitalised tryPredefine message) passes the new filetest unchanged.
#
# Repro from a plain clone of gnolang/gno:
#   git fetch origin pull/6108/head && git checkout --detach b512ada646e132543f32d48631c8e1bf40a4d1d9
#   bash solo-trypredefine-unreachable.sh
set -u
cd gnovm/pkg/gnolang
run() { go test . -run 'TestFiles/^import2.gno$' -count=1 2>&1 | grep -m2 'SENTINEL\|^ok\|^FAIL'; }
echo '== head'; run
echo '== sentinel in tryPredefine case "."'
sed -i '5606s/dot imports not allowed in gno/SENTINEL-tryPredefine/' preprocess.go; run; git checkout -- preprocess.go
echo '== sentinel in initStaticBlocks2'
sed -i '491s/dot imports not allowed in gno/SENTINEL-isb2/' preprocess.go; run; git checkout -- preprocess.go
echo '== merge-base preprocess.go, new filetest'
git checkout 26c0a7b32a1ec0c194cb97df893ef77778e1a3e5 -- preprocess.go; run; git checkout HEAD -- preprocess.go
# Observed at b512ada64:
#   == head                                  ok
#   == sentinel in tryPredefine case "."     ok
#   == sentinel in initStaticBlocks2         +main/import2.gno:3:8-19: SENTINEL-isb2 / FAIL
#   == merge-base preprocess.go              ok
