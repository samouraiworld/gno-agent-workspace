#!/usr/bin/env bash
# Repro (from a plain clone of gnolang/gno at 58b2692067d6545410b99c71f33bdf506caadaa0, run from the repo root):
# deletes the `case *gno.BoundMethodValue:` arm of ownObject and runs every test the PR adds for chain/reflect.
# Observed at that sha: 5 filetests and 8 unit tests still PASS with the arm gone.
set -euo pipefail
python3 - <<'PY'
p='gnovm/stdlibs/chain/reflect/reflect.go'
s=open(p).read()
i=s.index("\tcase *gno.BoundMethodValue:"); j=s.index("return cv",i)+len("return cv")
open(p,'w').write(s[:i]+s[j:])
PY
(cd gnovm/pkg/gnolang && go test -short -v -run 'TestFiles/^zrealm_reflect_(identity|objectid|duplicate_symbol|proposal_fund|foreign_finalize).gno$' . | grep -c -- '--- PASS: TestFiles/')
(cd gnovm/stdlibs/chain/reflect && go test -v . | grep -c -- '^--- PASS')
git checkout -- gnovm/stdlibs/chain/reflect/reflect.go
