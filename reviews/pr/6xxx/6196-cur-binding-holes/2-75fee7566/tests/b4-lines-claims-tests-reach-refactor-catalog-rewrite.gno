// Rewrite of gnovm/tests/stdlibs/testing/cur_subtest_test.gno (PR 6196, sha
// 6d88deba7) with the dead `if !cur.IsCurrent()` guard dropped: 25 lines to 22,
// 13 code lines to 10, same regression power. Replaces the file in place.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno.git && cd gno
//	git fetch origin pull/6196/head
//	git checkout 6d88deba71165b165641b65541259d00aba20377
//	cp <this file> gnovm/tests/stdlibs/testing/cur_subtest_test.gno
//	go test ./gnovm/pkg/gnolang/ -run 'TestStdlibs/test-testing'   # ok
//	# now revert the fix and confirm the rewrite still catches it:
//	sed -i 's/ \&\& !m.harnessSeedsCur()//' gnovm/pkg/gnolang/op_call.go
//	go test ./gnovm/pkg/gnolang/ -run 'TestStdlibs/test-testing'   # FAIL
//
// Observed at 6d88deba7: ok 2.889s, then
// "panic: crossing function testing_test func literal at cur_subtest_test.gno:16
// was entered without cross()" and FAIL.

package testing_test

import "testing"

// A crossing sub-test entered after the test crossed into a helper must keep
// working. t.Run seeds the sub-test with the TOP-LEVEL test's cur on purpose
// (testing.gno), while the live crossing frame is the helper's minted cur, so
// the VM's stale-capture check has to exempt the testing stdlib's own
// dispatch. Without that exemption this test panics and aborts the run.

// curSubtestTarget is entered no-cross with the sub-test's seeded cur: the
// value has to stay forwardable to a second crossing entry.
func curSubtestTarget(cur realm) {}

func curSubtestHelper(cur realm, t *testing.T) {
	t.Run("sub", func(cur realm, t *testing.T) {
		curSubtestTarget(cur)
	})
}

func TestCurSubtestAfterCross(cur realm, t *testing.T) {
	curSubtestHelper(cross(cur), t)
}
