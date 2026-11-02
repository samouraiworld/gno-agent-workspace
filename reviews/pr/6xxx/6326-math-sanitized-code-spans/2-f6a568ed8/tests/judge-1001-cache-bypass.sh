#!/usr/bin/env bash
# Asserts the search cache, not the scanner, leaves "$b c⏎d$" as text in "$a \\(`\\) $b c⏎d$ `e".
# With findCloseCached bypassed the same input renders 2 <math>; with it, 1. Fails at f6a568ed8.
# Run from a gnolang/gno checkout at the reviewed head, with solo-finder-2-cached-open-run_test.go copied in as zz_cached_open_run_test.go.
set -e
cd gno.land/pkg/gnoweb/markdown
go test . -run TestZZCachedOpenRun -v -count=1 | grep 'alone:' || true
perl -0pi -e 's/(\t\treturn -1, 0, 0\n\t\}\n)/$1\treturn find(b, ticks, nil)\n/' ext_math.go
go test . -run TestZZCachedOpenRun -v -count=1 | grep 'alone:\|PASS\|FAIL' || true
git checkout -- ext_math.go
