/* Run: from a gno checkout:
gh pr checkout 6028 -R gnolang/gno && git checkout 37182a315
curl -fsSL -o examples/gno.land/p/onbloc/identifier/across_blocks_test.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6028-registry-owned-id-generator/1-37182a315/tests/identifier_across_blocks_test.gno
cd examples && go run ../gnovm/cmd/gno test -v -run TestNextIDAcrossBlocks ./gno.land/p/onbloc/identifier
rm gno.land/p/onbloc/identifier/across_blocks_test.gno
*/

// NextID resets its per-block sequence when the chain height advances, so the
// (height, sequence) pair carrying uniqueness only ever changes on one axis in
// the shipped tests: they all run at a single height. This exercises the reset
// branch across three blocks and passes at 37182a315.
package identifier

import "testing"

func newBlockTestGenerator(_ int, rlm realm) (g *Generator) {
	func(cur realm) {
		g = NewGenerator(0, cur)
	}(cross(rlm))
	return
}

func TestNextIDAcrossBlocks(cur realm, t *testing.T) {
	g := newBlockTestGenerator(0, cur)

	seen := map[string]struct{}{}
	for block := 0; block < 3; block++ {
		for i := 0; i < 3; i++ {
			id := g.NextID()
			if _, dup := seen[id]; dup {
				t.Errorf("NextID repeated across blocks: " + id)
			}
			seen[id] = struct{}{}
		}
		// Advance the chain so the next NextID takes the sequence-reset branch.
		testing.SkipHeights(1)
	}

	if len(seen) != 9 {
		t.Errorf("expected 9 distinct ids across 3 blocks")
	}
}
