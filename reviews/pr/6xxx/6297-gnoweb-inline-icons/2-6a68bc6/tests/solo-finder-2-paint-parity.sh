#!/usr/bin/env bash
# solo-finder-2: does every regenerated outline glyph (icons_gen.go) paint as
# its source symbol (icons/drawn.svg, icons/vendored.svg)? Renders both with
# resvg and counts differing pixels with ImageMagick, independent of
# TestIconTableMatchesSource's own inheritance model.
#
# Repro from a plain clone of gnolang/gno (needs resvg and magick on PATH):
#
#   git fetch origin pull/6297/head && git checkout --detach 6a68bc69fc638c389d2308743e2fb1d11cc326bc
#   cp solo-finder-2-paint-parity.go gno.land/pkg/gnoweb/markdown/zz_paintdump_test.go
#   mkdir -p /tmp/paint/head /tmp/paint/base
#   (cd gno.land/pkg/gnoweb/markdown && PAINT_DUMP_DIR=/tmp/paint/head go test -count=1 -run TestZZPaintDump -v .)
#   # the round-1 table, sources unchanged: swap icons_gen.go in, dump, swap back
#   cp gno.land/pkg/gnoweb/markdown/icons_gen.go /tmp/paint/icons_gen.head.go
#   git show 84df2c459:gno.land/pkg/gnoweb/markdown/icons_gen.go > gno.land/pkg/gnoweb/markdown/icons_gen.go
#   (cd gno.land/pkg/gnoweb/markdown && PAINT_DUMP_DIR=/tmp/paint/base go test -count=1 -run TestZZPaintDump -v .)
#   cp /tmp/paint/icons_gen.head.go gno.land/pkg/gnoweb/markdown/icons_gen.go
#   bash solo-finder-2-paint-parity.sh 96 /tmp/paint/head; bash solo-finder-2-paint-parity.sh 21 /tmp/paint/head
#   bash solo-finder-2-paint-parity.sh 96 /tmp/paint/base
#
# Measured 2026-10-08:
#   head 6a68bc69f:  "TOTAL 434 compared, 0 differ at 96px", "TOTAL 434 compared, 0 differ at 21px"
#   round-1 table (84df2c459 icons_gen.go): "TOTAL 434 compared, 58 differ at 96px"
#     (warning-circle, warning-hex, warning-triangle among them)
#
# Usage: solo-finder-2-paint-parity.sh <size> <dumpdir>; prints "<name> <AE>"
# for each pair that differs, then a total.
set -u
size=$1 dir=$2
diff=0 total=0
for src in "$dir"/*.src.svg; do
	name=$(basename "$src" .src.svg)
	gen="$dir/$name.gen.svg"
	resvg -w "$size" -h "$size" "$src" "$dir/$name.src.$size.png" 2>/dev/null || { echo "$name render-src-failed"; continue; }
	resvg -w "$size" -h "$size" "$gen" "$dir/$name.gen.$size.png" 2>/dev/null || { echo "$name render-gen-failed"; continue; }
	ae=$(magick compare -metric AE "$dir/$name.src.$size.png" "$dir/$name.gen.$size.png" null: 2>&1 | awk '{print $1}')
	total=$((total + 1))
	if [ "$ae" != "0" ]; then
		echo "$name $ae"
		diff=$((diff + 1))
	fi
done
echo "TOTAL $total compared, $diff differ at ${size}px"
