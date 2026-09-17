// Sentinel variant of gnovm/tests/stdlibs/testing/cur_subtest_test.gno at
// 6d88deba71165b165641b65541259d00aba20377. The shipped test's only assertion
// is `if !cur.IsCurrent() { panic(...) }`; the VM makes the entered cur the
// current realm on entry, so that condition is false by construction and the
// panic is unreachable. Flipping the condition fires it, which is the proof.
//
// Repro from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin pull/6196/head:pr6196 && git checkout 6d88deba71165b165641b65541259d00aba20377
//   cp <this file> gnovm/tests/stdlibs/testing/cur_subtest_test.gno
//   go test ./gnovm/pkg/gnolang/ -run 'TestStdlibs/test-testing'
//
// Observed at head, with this file in place:
//   panic: SENTINEL: IsCurrent() was true
//   --- FAIL: TestCurSubtestAfterCross/sub (0.00s)
//
// Observed at head, unmodified: PASS. The two runs together show the shipped
// assertion cannot fail, whatever cur the harness seeded. What the test does
// pin is the entry not aborting: dropping `!m.harnessSeedsCur()` from
// gnovm/pkg/gnolang/op_call.go:513 turns it red with
//   panic: crossing function testing_test func literal at cur_subtest_test.gno:18
//   was entered without cross(), ...
// so the test covers the abort and not the identity the carve-out preserves.
// An assertion that discriminates reads the value back, e.g. comparing
// cur.Previous() against the realm the helper crossed from.

package testing_test

import "testing"

func curSubtestTarget(cur realm) {
	if cur.IsCurrent() {
		panic("SENTINEL: IsCurrent() was true")
	}
}

func curSubtestHelper(cur realm, t *testing.T) {
	t.Run("sub", func(cur realm, t *testing.T) {
		curSubtestTarget(cur)
	})
}

func TestCurSubtestAfterCross(cur realm, t *testing.T) {
	curSubtestHelper(cross(cur), t)
}
