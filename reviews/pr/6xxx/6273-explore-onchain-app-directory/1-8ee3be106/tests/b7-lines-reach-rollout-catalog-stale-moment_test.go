// gnoweb half of b7-lines-reach-rollout-catalog-stale-moment_test.gno.
//
//   cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b7_moment_test.go
//   go test ./gno.land/pkg/gnoweb/feature/store/ -run TestB7MomentHasNoAge -v
//
// Expected at 8ee3be106: pulse: Block 6307200 · "Valopers just joined"
package store

import "testing"
// The realm serves its activity with no height (TestB7StaleSeedMoment on the
// realm side): gnoweb phrases the newest event as just happened, beside the
// chain's current block.
func TestB7MomentHasNoAge(t *testing.T) {
	v := listing{Slug: "valopers", Kind: kindApp, Title: "Valopers", webPath: "/r/gnops/valopers"}
	res := &homeResponse{Activity: []activity{{Kind: "listed", Slug: "valopers"}}}
	res.Height = 365 * blocksPerDay
	p := newPulse(res.chainClock, res.Pulse, latestMoment(res.Activity, indexListings([]listing{v})))
	t.Logf("pulse: Block %d · %q", p.Height, p.Moment.Text)
}
