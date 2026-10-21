// Repro: a cgroup at its memory.max reads as a failed reading, so a ramped
// node budget never shrinks there, and every poll re-reads.
//
// PR: https://github.com/gnolang/gno/pull/6061 at 115d118c79c72256f69dcd5e419104f794316b8f
//
// clampToCgroup (tm2/pkg/testutils/parallel_linux.go:71-79) returns
// Available 0 once memory.current less inactive_file reaches memory.max: the
// "usage at the limit" case of TestClampToCgroup pins exactly that. ReadMemInfo
// then returns `mi, mi.Total > 0 && mi.Available > 0` (parallel_linux.go:62),
// i.e. ok == false, and resize (gno.land/pkg/integration/testscript_gnoland.go:215-217)
// returns before shrinking and before advancing lastCheck.
//
// readAtLimit below reproduces that return expression verbatim, so the budget
// sees what the real reader hands it at 1 MiB, 1 byte and 0 bytes free.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 115d118c79c72256f69dcd5e419104f794316b8f
//	cp <this file> gno.land/pkg/integration/zz_budget_at_cgroup_limit_test.go
//	go test ./gno.land/pkg/integration -run TestReproBudgetAtCgroupLimit -v -count=1
//
// Expected on a correct budget: limit falls to 4 in all three rows, one read.
// Observed: the 0-byte row keeps limit 6 and reads on every tryAcquire.
package integration

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/testutils"
)

func TestReproBudgetAtCgroupLimit(t *testing.T) {
	for _, avail := range []uint64{1 << 20, 1, 0} {
		reads := 0
		readAtLimit := func() (testutils.MemInfo, bool) {
			reads++
			mi := testutils.MemInfo{Total: 4 << 30, Available: avail}
			return mi, mi.Total > 0 && mi.Available > 0 // parallel_linux.go:62
		}
		b := &nodeBudget{
			min: 2, max: 8, limit: 6, running: 6,
			reserve:  1 << 30, // Total / nodeMemReserveDiv
			readMem:  readAtLimit,
			readHeld: func() uint64 { return 1 << 40 },
		}
		for range 3 { // three readings, one interval apart (elapse is in node_budget_test.go)
			elapse(b)
			b.tryAcquire()
		}
		readsBefore := reads
		for range 20 { // 20 polls inside one read interval
			b.tryAcquire()
		}
		t.Logf("available=%d bytes: limit=%d (started 6, min 2), reads during 20 polls inside the interval=%d",
			avail, b.limit, reads-readsBefore)
	}
}
