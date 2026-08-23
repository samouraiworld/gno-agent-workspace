// Two goroutines preprocess `type E error`, each with its own Machine and
// store. The uverse `error` *InterfaceType is a process-global singleton
// (uverse.go:483), and after this PR the TRANS_LEAVE TypeDecl handler
// writes it: dstT.Base and tmp2.Base are both baseOf(error), so
// fillTypeInPlace runs *dst = *dst on the singleton.
//
// Measured with go1.26.5:
//   merge-base 0397fc87f  2 races, both (*DeclaredType).TypeID() memoization
//   head       093c32be0  3 races, two naming fillTypeInPlace (types.go:1577)
//   head + `dstT.Base != tmp2.Base` guard   2 races, zero fillTypeInPlace frames
//
// Full reports in race_base.txt / race_head.txt / race_guard.txt. gno CI
// runs no -race, so no job catches this.
//
/* Run: from a gno checkout:
gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
curl -fsSL -o gnovm/pkg/gnolang/zz_uverserace_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5763-unsealed-declaredtype-mutual-recursion/3-093c32be0/tests/consensus/uverse_base_race_test.go
go test -race -count=1 -run TestZZUverseBaseRace ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'WARNING: DATA RACE'
git checkout $(git merge-base origin/master HEAD) -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
go test -race -count=1 -run TestZZUverseBaseRace ./gnovm/pkg/gnolang/ 2>&1 | grep -c 'WARNING: DATA RACE'
git checkout HEAD -- gnovm/pkg/gnolang/preprocess.go gnovm/pkg/gnolang/types.go
rm gnovm/pkg/gnolang/zz_uverserace_test.go
*/

package gnolang

import (
	"sync"
	"testing"
)

// TestZZUverseBaseRace preprocesses `type E error` in two goroutines, each
// with its own Machine and store. The uverse `error` *InterfaceType is a
// process-global singleton shared by both, and after PR 5763 the
// TRANS_LEAVE TypeDecl handler writes it (*dst = *dst, dst == src). Run
// under -race on the merge-base and on the head and compare.
func TestZZUverseBaseRace(t *testing.T) {
	// Force uverse init before the goroutines start, so the race (if any)
	// is the fill-site write and not lazy uverse construction.
	_ = UverseNode()
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { recover() }()
			m := NewMachine("testdata", nil)
			defer m.Release()
			nn := m.MustParseFile("testdata.gno", "package testdata\ntype E error\n")
			m.RunFiles(nn)
		}()
	}
	wg.Wait()
}
