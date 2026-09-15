// Run: from a gno checkout:
// gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
// curl -fsSL -o /tmp/07-trailing-separator_test.go \
//   https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/07-trailing-separator.go
// cp /tmp/07-trailing-separator_test.go gno.land/pkg/gnoland/07-trailing-separator_test.go
// go test -v -run 'TestParseReleaseVersionTrailingSeparator' ./gno.land/pkg/gnoland/
// rm gno.land/pkg/gnoland/07-trailing-separator_test.go
//
// Asserts that a tag ending in "-" or "+" is rejected by parseReleaseVersion,
// mirroring the malformed-input rows already in TestParseReleaseVersion
// (node_params_version_test.go:50). Both fail at 607942b78: strings.Cut treats
// a trailing separator as an empty pre-release/build component and accepts the
// tag as the final release, so "v1.2.0-" parses equal to "v1.2.0".

package gnoland

import "testing"

func TestParseReleaseVersionTrailingSeparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{"trailing dash", "v1.2.0-"},
		{"trailing plus", "v1.2.0+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// IS:     bug — trailing separator silently accepted as the release.
			got, ok := parseReleaseVersion(tt.in)
			if ok {
				t.Errorf("parseReleaseVersion(%q) = %+v, true; want ok=false", tt.in, got)
			}

			// SHOULD: rejected, same as the other malformed-input rows.
			// require.False(t, ok, "parse ok for %q", tt.in)
		})
	}
}
