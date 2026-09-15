/* Run: from a gno checkout:
gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
curl -fsSL -o gno.land/pkg/gnoland/release_scripts_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/51-release-scripts-no-ci_test.go
go test -v -run 'TestReleaseScripts' ./gno.land/pkg/gnoland/
rm gno.land/pkg/gnoland/release_scripts_test.go
*/

// Asserts the two claims misc/release/cut-release.sh makes about itself: that
// its shape gate is "the shape parseReleaseVersion accepts" (cut-release.sh:111)
// and that --help prints its own header. Both are false at 607942b78, and no
// job on the pull request runs over either script: `grep -rnE
// 'shellcheck|shfmt|bash -n' .github/` returns one hit, a disable comment in
// _ci-go.yml, and ci-dir-misc.yml's fixed program matrix has no `release` row.

package gnoland

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const cutRelease = "../../../misc/release/cut-release.sh"

// scriptAcceptsShape runs cut-release.sh's own check_version_shape over one
// version, with the function lifted out of the script so main() does not run.
func scriptAcceptsShape(t *testing.T, version string) bool {
	t.Helper()
	src, err := os.ReadFile(cutRelease)
	if err != nil {
		t.Fatalf("read %s: %v", cutRelease, err)
	}
	var fn strings.Builder
	keep := false
	for _, line := range strings.Split(string(src), "\n") {
		switch {
		case strings.HasPrefix(line, "die() {"), strings.HasPrefix(line, "check_version_shape() {"):
			keep = true
		case strings.HasPrefix(line, "ok() "):
			fn.WriteString(line + "\n")
			continue
		}
		if keep {
			fn.WriteString(line + "\n")
		}
		if line == "}" {
			keep = false
		}
	}
	if !strings.Contains(fn.String(), "check_version_shape() {") {
		t.Fatalf("could not lift check_version_shape out of %s", cutRelease)
	}
	cmd := exec.Command("bash", "-c", fn.String()+"\nVERSION="+version+"\ncheck_version_shape")
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

func TestReleaseScripts(t *testing.T) {
	// The gate and the parser must agree: a tag the script clears but the node
	// rejects cannot be used as halt_min_version, which the script's own
	// comment calls "most of the point of tagging".
	t.Run("ShapeGateAgreesWithParser", func(t *testing.T) {
		for _, v := range []string{
			"v1.2.0", "v1.2.0-rc.1", // both accept
			"v1.02.0", "v01.2.0", "v1.2.00", // script only, at 607942b78
			"v1.2.0-", "chain/gnoland2.0", // parser only, at 607942b78
			"v1.2", "1.2.0", // neither
		} {
			_, parsed := parseReleaseVersion(v)
			got := scriptAcceptsShape(t, v)
			if got != parsed {
				// IS at 607942b78: the two disagree on five of nine shapes.
				t.Errorf("%-16s cut-release.sh accepts=%v, parseReleaseVersion ok=%v", v, got, parsed)
			}
			// SHOULD: got == parsed for every shape.
		}
	})

	// usage() reads a fixed line range out of its own header, so a line added
	// to the header past that range never reaches --help.
	t.Run("UsageCoversItsOwnHeader", func(t *testing.T) {
		src, err := os.ReadFile(cutRelease)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		lines := strings.Split(string(src), "\n")
		last := 0 // last line of the leading comment block, 1-indexed
		for i, line := range lines[1:] {
			if !strings.HasPrefix(line, "#") {
				break
			}
			last = i + 2
		}
		out, err := exec.Command("bash", filepath.Clean(cutRelease), "--help").Output()
		if err != nil {
			t.Fatalf("--help: %v", err)
		}
		want := strings.TrimPrefix(strings.TrimPrefix(lines[last-1], "#"), " ")
		if !strings.Contains(string(out), want) {
			// IS at 607942b78: header line 35, the --halt-height example, is
			// outside usage()'s `sed -n '2,34p'` range.
			t.Errorf("--help drops header line %d: %q", last, want)
		}
		// SHOULD: --help prints every line of the header block.
	})
}
