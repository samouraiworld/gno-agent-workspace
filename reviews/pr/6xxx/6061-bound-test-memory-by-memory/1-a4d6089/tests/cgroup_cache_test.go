/* Run: from a gno checkout, inside a container with a cgroup v2 memory limit:
gh pr checkout 6061 -R gnolang/gno && git checkout a4d6089
curl -fsSL -o tm2/pkg/testutils/cgroup_cache_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6061-bound-test-memory-by-memory/1-a4d6089/tests/cgroup_cache_test.go
go test -v -run TestCgroupCacheUnderReportsAvailable ./tm2/pkg/testutils/
rm tm2/pkg/testutils/cgroup_cache_test.go
*/

// Asserts ReadMemInfo does not count the page cache charged to the cgroup as
// available. memory.current includes reclaimable file cache and slab, so
// limit-current reads as pressure that reclaim would clear on demand.
// This reads live figures rather than a fixture: it fails at a4d6089 once the
// reclaimable share of memory.current is large, and the logged lines are worth
// reading whatever the result. A `go build ./...` beforehand warms the cache
// enough on a container of a few GiB.

package testutils

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func cgStat(t *testing.T, key string) uint64 {
	t.Helper()
	bz, err := os.ReadFile("/sys/fs/cgroup/memory.stat")
	if err != nil {
		t.Skip("no cgroup v2 memory.stat")
	}
	for line := range strings.Lines(string(bz)) {
		k, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || k != key {
			continue
		}
		v, err := strconv.ParseUint(strings.TrimSpace(rest), 10, 64)
		if err != nil {
			t.Fatalf("parse %s: %v", key, err)
		}
		return v
	}
	t.Fatalf("%s absent from memory.stat", key)
	return 0
}

func gib(v uint64) float64 { return float64(v) / (1 << 30) }

func TestCgroupCacheUnderReportsAvailable(t *testing.T) {
	limit, used, ok := cgroupMemory()
	if !ok {
		t.Skip("no cgroup v2 memory limit in force")
	}
	// Reclaimable without evicting anything a process is using.
	reclaimable := cgStat(t, "inactive_file") + cgStat(t, "slab_reclaimable")

	mi, ok := ReadMemInfo()
	if !ok {
		t.Fatal("ReadMemInfo failed")
	}

	var reported uint64
	if limit > used {
		reported = limit - used
	}
	// What the reading should be once reclaimable pages are not counted as used.
	corrected := reported + reclaimable

	t.Logf("cgroup limit=%.2fGiB current=%.2fGiB inactive_file+slab_reclaimable=%.2fGiB",
		gib(limit), gib(used), gib(reclaimable))
	t.Logf("ReadMemInfo Available=%.2fGiB, cache-corrected=%.2fGiB", gib(mi.Available), gib(corrected))

	// The budget in gno.land/pkg/integration shrinks below Total/4.
	reserve := mi.Total / 4
	t.Logf("reserve=Total/4=%.2fGiB shrinks=%v (corrected would shrink=%v)",
		gib(reserve), mi.Available < reserve, corrected < reserve)

	if mi.Available < corrected/2 {
		t.Fatalf("Available %.2fGiB is under half the cache-corrected %.2fGiB: reclaimable cache reads as pressure",
			gib(mi.Available), gib(corrected))
	}
}
