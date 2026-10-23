// Repro for gnolang/gno#6402 at bff79f498d08059d599404035d2a46e54611ecd0,
// review round 1, solo-finder-2.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6402/head && git checkout --detach bff79f498d08059d599404035d2a46e54611ecd0
//	cp <this file> contribs/gpao/solofinder2_test.go
//	cd contribs/gpao && go test -count=1 -run 'TestSoloFinder2' -v .
//
// TestSoloFinder2StatusAfterReofferCap: once maxReoffers is spent the
// submission leaves `stranded` and nothing offers it again this run, yet the
// status board still reports it `pending`, which the README defines as "will be
// retried". The sibling caps (maxOverBudgetAttempts, maxEnableAttempts) record
// `gave_up` at the same point.
//
// TestSoloFinder2EarlyPathDelaysLaterOnes: every pass restarts at the first
// path in sorted order and stops as soon as a candidate is queued, so a path
// sorted first and offered on every pass holds back every path after it until
// it spends its maxReoffers offers.
package main

import (
	"bytes"
	"sync/atomic"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoloFinder2StatusAfterReofferCap(t *testing.T) {
	chain := newInertChain(t)
	const path = "gno.land/r/test/slow"
	pkg := chainPackage(path, "package slow\n\nfunc N(cur realm) int { return 1 }\n")
	chain.park(t, pkg)

	o := chain.oracle(t)
	o.cfg.prepareBudget = time.Nanosecond // every verification ends unavailable
	o.handleCandidate(t.Context(), candidate{mpkg: pkg, height: chain.submittedAt(t, path)})
	require.Equal(t, statusPending, o.status.get(path).Status, "premise: the submission is stranded")

	for n := 0; n <= maxReoffers; n++ {
		o.reofferStranded(t.Context())
	}
	require.NotContains(t, o.stranded, path, "premise: gpao stopped offering it this run")

	st := o.status.get(path)
	t.Logf("status after the reoffer cap: %+v", st)
	assert.NotEqual(t, statusPending, st.Status,
		"nothing retries it any more, but the board says pending (will be retried)")
}

func TestSoloFinder2EarlyPathDelaysLaterOnes(t *testing.T) {
	chain := newInertChain(t)
	early := chainPackage("gno.land/r/test/aaa", "package aaa\n\nfunc N(cur realm) int { return 1 }\n")
	late := chainPackage("gno.land/r/test/zzz", "package zzz\n\nfunc N(cur realm) int { return 1 }\n")
	chain.park(t, early)
	chain.park(t, late)
	earlyAt := chain.submittedAt(t, early.Path)
	lateAt := chain.submittedAt(t, late.Path)

	// A new block's candidate arrives while the pass is reading a submission's
	// block, as one does on a chain that keeps receiving packages.
	var o *oracle
	var arm atomic.Bool
	proxy := httptest.NewServer(rewriting(t, chain.remote, func(body []byte) []byte {
		if arm.Load() && bytes.Contains(body, []byte(`"method":"block"`)) {
			select {
			case o.candidates <- candidate{mpkg: early, height: earlyAt}:
			default:
			}
		}
		return body
	}))
	t.Cleanup(proxy.Close)
	chain.remote = proxy.URL

	o = chain.oracle(t)
	o.cfg.prepareBudget = time.Nanosecond
	o.handleCandidate(t.Context(), candidate{mpkg: early, height: earlyAt})
	o.handleCandidate(t.Context(), candidate{mpkg: late, height: lateAt})
	require.Contains(t, o.stranded, early.Path, "premise")
	require.Contains(t, o.stranded, late.Path, "premise")

	arm.Store(true)
	lateFirstOffered := 0
	for pass := 1; pass <= maxReoffers+1; pass++ {
		o.reofferStranded(t.Context())
		for len(o.candidates) > 0 { // the verifier handles the new block's candidate
			<-o.candidates
		}
		_, earlyTracked := o.stranded[early.Path]
		t.Logf("pass %d: early offers=%d tracked=%v, late offers=%d",
			pass, o.stranded[early.Path].offers, earlyTracked, o.stranded[late.Path].offers)
		if lateFirstOffered == 0 && o.stranded[late.Path].offers > 0 {
			lateFirstOffered = pass
		}
	}
	assert.LessOrEqual(t, lateFirstOffered, 2,
		"a pass resuming where the last one stopped reaches the later path by pass 2")
}
