// judge 1 and 3: what counterpartTarget returns on a twinless walk reaching a
// package directory, with direct children (1) and on a listing capped at
// maxCounterpartPaths (3). Prints the results; run at the head and the base.
package gnoweb

import (
	"fmt"
	"testing"
)

func TestJudgeCounterpartTarget(t *testing.T) {
	tgt, n := counterpartTarget("/r/tests/vm/foo", "/r/tests/vm",
		[]string{"/r/tests/vm", "/r/tests/vm/crossrealm", "/r/tests/vm/subtests"})
	fmt.Printf("c1 twinless, dir has direct children: target=%q n=%d label=%q\n", tgt, n, counterpartLink(tgt, "/r/tests/vm", n).Label)

	paths := []string{"/p/alice/golf"}
	for i := 0; i < maxCounterpartPaths-1; i++ {
		paths = append(paths, fmt.Sprintf("/p/alice/golf/a%02d", i))
	}
	tgt, n = counterpartTarget("/p/alice/golf/zz", "/p/alice/golf", paths)
	fmt.Printf("c3 capped listing (%d paths), twin past the cap: target=%q n=%d label=%q\n", len(paths), tgt, n, counterpartLink(tgt, "/p/alice/golf", n).Label)
}
