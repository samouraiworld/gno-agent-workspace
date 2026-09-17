// Probe: the `if !cur.IsCurrent()` guard added by PR 6196 in
// gnovm/tests/stdlibs/testing/cur_subtest_test.gno can never be false, so the
// committed test asserts nothing there. The guard is flipped here: the panic
// firing proves `cur.IsCurrent()` is true by construction on a crossing
// function's own first parameter (gnovm/adr/interrealm_v2.md).
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno.git && cd gno
//	git fetch origin pull/6196/head
//	git checkout 6d88deba71165b165641b65541259d00aba20377
//	cp <this file> gnovm/tests/stdlibs/testing/zzprobe_iscurrent_test.gno
//	go test ./gnovm/pkg/gnolang/ -run 'TestStdlibs/test-testing' -v
//
// Observed at 6d88deba7:
//
//	=== RUN   TestZZProbeIsCurrentVacuous/sub
//	panic: IS-CURRENT-VACUOUS: the guard in cur_subtest_test.gno can never fire
//	--- FAIL: TestZZProbeIsCurrentVacuous (0.00s)

package testing_test

import "testing"

func zzprobeTarget(cur realm) {
	if cur.IsCurrent() {
		panic("IS-CURRENT-VACUOUS: the guard in cur_subtest_test.gno can never fire")
	}
}

func zzprobeHelper(cur realm, t *testing.T) {
	t.Run("sub", func(cur realm, t *testing.T) {
		zzprobeTarget(cur)
	})
}

func TestZZProbeIsCurrentVacuous(cur realm, t *testing.T) {
	zzprobeHelper(cross(cur), t)
}
