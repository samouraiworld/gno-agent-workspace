#!/usr/bin/env bash
# Mutation harness for misc/gnopreview/plan_test.go at gnolang/gno PR 6194, head ecf7af0.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno
#   cd gno && git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash <this file>            # run from the repo root, go1.25.x on PATH
#
# Each mutation removes one line or guard that plan_test.go / crawl_test.go are
# supposed to pin. GREEN means `go test ./misc/gnopreview/...` still passes with
# the guard gone: the line is unpinned. The last block is the control, a
# mutation that must go RED, so a GREEN row is not just a broken harness.
#
# Measured 2026-09-19 with go1.25.9:
#   M1 shotGrid row wrap            GREEN
#   M2 sort.Strings(realms) re-sort GREEN
#   M3 every base=="" guard         GREEN
#   M4 LoadPkgs package detection   GREEN
#   M5 gnoImports parse recovery    GREEN
#   M6 tabs() help link             GREEN
#   M8 max-realms=0 (no cap)        GREEN
#   control renderRelevant _test    RED   (harness works)

set -u
cd "$(git rev-parse --show-toplevel)/misc/gnopreview" || exit 1

restore() { git checkout -- . ; }

run() {
  name="$1"; shift
  "$@"
  if go test ./... >/tmp/gnopreview-mut.txt 2>&1; then
    echo "$name: GREEN (mutation survives)"
  else
    echo "$name: RED"
    head -3 /tmp/gnopreview-mut.txt
  fi
  restore
}

run "M1 shotGrid row wrap i%2"          sed -i 's|if i > 0 && i%2 == 0 {|if i > 0 \&\& i%2 == 99 {|' comment.go
run "M2 final sort.Strings(realms)"     sed -i '260s|^\tsort.Strings(realms)$|\t_ = realms|' plan.go
run "M3 every base==\"\" guard"         sed -i 's|base == ""|base == "\\x00"|g' comment.go
run "M4 LoadPkgs !hasMod||no .gno"      sed -i 's#if !hasMod || len(files) == 0 {#if !hasMod \&\& false {#' plan.go
run "M5 gnoImports parse recovery"      sed -i 's|^\t\t\tcontinue // a PR that does not compile.*|\t\t\treturn nil|' plan.go
run "M6 tabs() help link"               sed -i 's| · \[help\](%s%s/_t/help/)||; s|, base, u, base, u)|, base, u)|' comment.go
run "M8 maxRealms>0 no-cap sentinel"    sed -i 's#if maxRealms > 0 \&\& len(realms) > maxRealms {#if maxRealms >= 0 \&\& len(realms) > maxRealms {#' plan.go

echo "--- control: this one must go RED"
run "C1 renderRelevant test-file skip"  sed -i 's|case strings.HasSuffix(base, "_test.gno"), strings.HasSuffix(base, "_filetest.gno"):|case false:|' plan.go
