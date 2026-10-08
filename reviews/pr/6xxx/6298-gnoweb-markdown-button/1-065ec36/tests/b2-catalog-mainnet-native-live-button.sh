#!/usr/bin/env bash
# gnolang/gno#6298 at 065ec369b: head gnoweb (button extension on) rendering
# content sanitized by the chain/markdown native that mainnet runs today.
#
# mainnet's native is byte-identical to the merge base's:
#   git diff --quiet chain/mainnet b0656512d -- gnovm/stdlibs/chain/markdown/   # exit 0
# The native only changes with a chain upgrade; gnoweb ships on its own deploy.
#
# Repro from a plain clone (Go 1.25.9):
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6298/head && git checkout --detach 065ec369b321c207815f82b0c1dbf2b0f9f10b01
#   git fetch origin tag chain/mainnet
#   bash <this file> .
#
# Expect: pass 1 (head native) green; pass 2 (mainnet native under head gnoweb)
# regenerates the sanitize goldens' output.html with <a ... class="gno-button ...">
# anchors where the head native renders &lt;gno-button as text.
set -u
T=${1:-.}
cd "$T"
pkg=./gno.land/pkg/gnoweb/markdown
run='TestSanitizeIntegration/.*gno-button'
echo "== pass 1: head native =="
GNOROOT=$PWD go test "$pkg" -run "$run" -count=1 2>&1 | tail -3
echo "== pass 2: chain/mainnet native, head gnoweb, goldens regenerated =="
git show chain/mainnet:gnovm/stdlibs/chain/markdown/markdown.go > gnovm/stdlibs/chain/markdown/markdown.go
GNOROOT=$PWD go test "$pkg" -run "$run" -count=1 -update-golden-tests 2>&1 | tail -1
echo "-- goldens whose output.html now carries a live button anchor --"
grep -l '<a [^>]*class="gno-button' gno.land/pkg/gnoweb/markdown/golden/sanitize/*gno-button*.txtar | xargs -n1 basename
echo "count: $(grep -l '<a [^>]*class="gno-button' gno.land/pkg/gnoweb/markdown/golden/sanitize/*gno-button*.txtar | wc -l) of $(ls gno.land/pkg/gnoweb/markdown/golden/sanitize/*gno-button*.txtar | wc -l)"
echo "-- one regenerated golden --"
sed -n '/-- output.html --/,$p' gno.land/pkg/gnoweb/markdown/golden/sanitize/block-gno-button-midline-escaped.txtar
git checkout -- gnovm/stdlibs/chain/markdown/markdown.go gno.land/pkg/gnoweb/markdown/golden/sanitize
