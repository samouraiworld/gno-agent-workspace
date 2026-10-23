// Judge artifact for candidates 2 and 1002, gnolang/gno#6401 at 5454f8165.
// Asserts the genesis import cache survives a tip one block behind the last
// read height, and a -start-height above the tip where nothing was read.
// Both fail at the reviewed head: the cache is emptied (kept 0).
package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJudgeTipLagKeepsGenesisImports(t *testing.T) {
	for name, tc := range map[string]struct {
		start int64
		tips  []int64
	}{
		"one-block lag after reading 7": {start: 1, tips: []int64{6, 7, 6, 8}},
		"start height above the tip":    {start: 1000, tips: []int64{500}},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			o := newStubOracle(&resetRPC{tips: tc.tips, cancel: cancel})
			o.cfg.pollInterval = time.Millisecond
			o.cfg.startHeight = tc.start
			o.imports = importCache{"gno.land/p/test/lib": chainPackage("gno.land/p/test/lib", "package lib\n")}

			require.NoError(t, o.run(ctx))
			assert.Len(t, o.genesisImports(), 1)
		})
	}
}
