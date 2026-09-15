// Asserts cut-release.sh's check_version_shape accepts exactly what
// gno.land/pkg/gnoland.parseReleaseVersion accepts, the claim at
// misc/release/cut-release.sh:112 and node_params.go:208.
// Measured at 607942b78: the shell regex accepts v1.02.0, v01.2.0 and v1.2.00;
// the Go parser rejects all three. Fails at the reviewed head.
//
/* Run: from a gno checkout:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78
curl -fsSL -o gno.land/pkg/gnoland/zz_shape_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/03-cut-release-version-shape_test.go
go test -v -run 'TestCutReleaseShapeMatchesParser' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/zz_shape_test.go
*/

package gnoland

import (
	"os"
	"os/exec"
	"regexp"
	"testing"
)

// shapeRE pulls the ERE out of check_version_shape's [[ ... =~ ... ]] test, so
// the test pins the script's own regex rather than a copy of it.
var shapeRE = regexp.MustCompile(`\$\{VERSION\}\s+=~\s+(\S+)\s+\]\]`)

func cutReleaseAccepts(t *testing.T, re, tag string) bool {
	t.Helper()
	// Same evaluation bash gives the line in the script: unquoted RHS variable
	// is read as an ERE.
	cmd := exec.Command("bash", "-c", `re="$1"; v="$2"; [[ $v =~ $re ]]`, "_", re, tag)
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		}
		t.Fatalf("bash: %v", err)
	}
	return true
}

func TestCutReleaseShapeMatchesParser(t *testing.T) {
	src, err := os.ReadFile("../../../misc/release/cut-release.sh")
	if err != nil {
		t.Fatalf("read cut-release.sh: %v", err)
	}
	m := shapeRE.FindSubmatch(src)
	if m == nil {
		t.Fatal("check_version_shape's regex not found in cut-release.sh")
	}
	re := string(m[1])
	t.Logf("check_version_shape regex: %s", re)

	// Tags a release engineer could plausibly type, plus the shapes the Go
	// table already pins.
	tags := []string{
		"v1.2.0", "v1.2.3", "v0.0.0", "v10.20.30", "v1.3.0-rc.1", "v1.2.0+deadbeef",
		"v1.02.0", "v01.2.0", "v1.2.00", // leading zeros: shell yes, Go no
		"v1.2", "v1.2.3.4", "1.2.0", "develop", "chain/mainnet", "v1.x.0",
	}
	for _, tag := range tags {
		shell := cutReleaseAccepts(t, re, tag)
		_, goOK := parseReleaseVersion(tag)
		// SHOULD: the gate the script advertises and the gate the node runs
		// agree on every tag. IS at 607942b78: the three leading-zero tags
		// pass the script and are refused by the node.
		if shell != goOK {
			t.Errorf("%q: cut-release.sh accepts=%v, parseReleaseVersion accepts=%v", tag, shell, goOK)
		}
	}
}

// TestLeadingZeroFloorLosesOrdering shows what the mismatch costs once the
// proposal is on chain: a floor the node cannot parse degrades to byte
// equality, so the next build off that line satisfies nothing.
func TestLeadingZeroFloorLosesOrdering(t *testing.T) {
	// Baseline: a parseable floor orders, so a hotfix clears it.
	if !meetsMinVersion("v1.2.1", "v1.2.0") {
		t.Error(`meetsMinVersion("v1.2.1", "v1.2.0") = false, want true`)
	}
	// SHOULD: the same pair one typo apart behaves the same way.
	// IS at 607942b78: false — the floor fell back to binaryVersion == minVersion.
	if !meetsMinVersion("v1.02.1", "v1.02.0") {
		t.Error(`meetsMinVersion("v1.02.1", "v1.02.0") = false, want true`)
	}
}
