#!/bin/sh
# Does anything in the tree pin that the testing-stdlib exemption stops at the
# IMMEDIATE caller? Run from a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git checkout 6d88deba71165b165641b65541259d00aba20377
#   sh b1-tests-harness-narrowness.sh
#
# Observed at that sha: every line below prints ok. Widening the exemption to
# any ANCESTOR testing frame disables the stale-capture identity check for
# every crossing call made under `gno test`, and no test notices.
set -e
python3 - <<'PY'
p = 'gnovm/pkg/gnolang/op_call.go'
s = open(p).read()
old = '''	caller := m.PeekCallFrame(2) // 1 is the callee frame being installed
	return caller != nil && caller.Func != nil && caller.Func.PkgPath == TestingBasePkgPath'''
new = '''	for n := 2; ; n++ {
		caller := m.PeekCallFrame(n)
		if caller == nil {
			return false
		}
		if caller.Func != nil && caller.Func.PkgPath == TestingBasePkgPath {
			return true
		}
	}'''
assert old in s
open(p, 'w').write(s.replace(old, new))
PY
go test ./gnovm/pkg/gnolang/ -count=1 -run 'TestHarnessSeedsCur'                      # ok 0.009s
go test ./gnovm/pkg/gnolang/ -count=1 -run 'TestFiles/zrealm_cur_backstop.gno$'       # ok 0.036s
go test ./gnovm/pkg/gnolang/ -count=1 -run 'TestFiles/zrealm_cur_defer.gno$'          # ok 0.046s
go test ./gnovm/pkg/gnolang/ -count=1 -run 'TestFiles/zrealm_cur_method_backstop.gno$' # ok 0.039s
go test ./gnovm/pkg/gnolang/ -count=1 -run 'TestStdlibs/test-testing'                 # ok 2.468s
git checkout -- gnovm/pkg/gnolang/op_call.go
# Control, same sha: DELETING the exemption is caught.
#   sed -i 's/&& !m.harnessSeedsCur() //' gnovm/pkg/gnolang/op_call.go
#   go test ./gnovm/pkg/gnolang/ -run 'TestStdlibs/test-testing'
#   -> FAIL, TestCurSubtestAfterCross/sub panics on the stale-capture check
