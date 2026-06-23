/* Run: from a gno checkout:
gh pr checkout 4555 -R gnolang/gno && git checkout beada2b29
curl -fsSL -o tm2/pkg/errors/zz_strip_build_dir_prefix_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/4xxx/4555-shorter-stacktraces/1-beada2b29/tests/strip_build_dir_prefix_test.go
go test -run 'TestStripBuildDir_PrefixBoundary' ./tm2/pkg/errors/
rm tm2/pkg/errors/zz_strip_build_dir_prefix_test.go
*/

// stripBuildDir uses strings.HasPrefix(path, buildDir) with no path-separator
// boundary, so a sibling directory whose name merely starts with the build-dir
// name (build dir "gno", sibling "gno-simple-e2e" — moul's own dir in the PR
// description) matches and yields a "gno//<full-abs-path>" string that still
// leaks the absolute path. A boundary-aware prefix check (buildDir + "/") fixes it.
package errors

import (
	"os"
	"sync"
	"testing"
)

func TestStripBuildDir_PrefixBoundary(t *testing.T) {
	origGOMOD := os.Getenv("GOMOD")
	defer func() {
		os.Setenv("GOMOD", origGOMOD)
		buildDir = ""
		buildDirOnce = sync.Once{}
	}()

	buildDir = ""
	buildDirOnce = sync.Once{}
	// build dir resolves to /home/u/gno (parent of the GOMOD file)
	os.Setenv("GOMOD", "/home/u/gno/go.mod")

	// A sibling dir sharing the "gno" prefix but NOT under the build root.
	in := "/home/u/gno-simple-e2e/tm2/pkg/std/errors.go"
	got := stripBuildDir(in)

	// SHOULD: not under the build root, so left untouched (no "gno/" rewrite).
	want := "/home/u/gno-simple-e2e/tm2/pkg/std/errors.go"
	if got != want {
		t.Errorf("stripBuildDir(%q) = %q, want %q", in, got, want)
	}
}
