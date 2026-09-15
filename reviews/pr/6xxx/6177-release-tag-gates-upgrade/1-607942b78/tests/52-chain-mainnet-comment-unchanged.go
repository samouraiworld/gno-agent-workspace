// Candidate #52: gno.land/pkg/gnoland/node_params_version_test.go:80-83's own
// comment says "the regression this shape exists for" is a minVersion of
// "chain/mainnet" degrading to byte equality, and that "both directions are
// now ordered". "chain/mainnet" is not one of the two shapes parseReleaseVersion
// accepts (only "vMAJOR.MINOR.PATCH" and "chain/gnolandMAJOR.MINOR"), so
// meetsMinVersion("<anything>", "chain/mainnet") still falls through to
// node_params.go's byte-equality fallback, unchanged from the merge base.
//
// Run from a plain clone:
//   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
//   curl -fsSL -o /tmp/chain_mainnet_probe_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6177-release-tag-gates-upgrade/1-607942b78/tests/52-chain-mainnet-comment-unchanged.go
//   cp /tmp/chain_mainnet_probe_test.go gno.land/pkg/gnoland/chain_mainnet_probe_test.go
//   go test -v -run 'TestChainMainnetFallbackUnchanged' ./gno.land/pkg/gnoland/
//   rm gno.land/pkg/gnoland/chain_mainnet_probe_test.go
//
// Then repeat against the merge base (1fc4c140ec4064e8d66cb9828ddea280a723cca6)
// by copying the same probe next to node_params.go there (meetsMinVersion and
// its fallback are unexported but identical in shape at both revisions):
//   git checkout 1fc4c140ec4064e8d66cb9828ddea280a723cca6
//   <copy probe as above, run the same -run, observe the same two results>
//
// Both revisions print the same pair: byte-equality fallback engaged, not the
// ordered comparison the comment credits this diff with adding for this case.
package gnoland

import "testing"

func TestChainMainnetFallbackUnchanged(t *testing.T) {
	// IS: chain/mainnet still falls back to byte equality, both at head and
	// at the merge base — the comment's "both directions are now ordered"
	// claim does not cover this row.
	if got := meetsMinVersion("v1.3.0", "chain/mainnet"); got != false {
		t.Fatalf("meetsMinVersion(%q, %q) = %v, want false (byte-equality fallback, unordered)", "v1.3.0", "chain/mainnet", got)
	}
	if got := meetsMinVersion("chain/mainnet", "chain/mainnet"); got != true {
		t.Fatalf("meetsMinVersion(%q, %q) = %v, want true (byte-equality fallback)", "chain/mainnet", "chain/mainnet", got)
	}
	// SHOULD (if the comment's claim were true): meetsMinVersion would parse
	// "chain/mainnet" as an ordered version and compare it against v1.3.0
	// rather than falling back to `binaryVersion == minVersion`. It does not:
	// parseReleaseVersion only accepts "vMAJOR.MINOR.PATCH" and
	// "chain/gnolandMAJOR.MINOR", never "chain/mainnet".
}
