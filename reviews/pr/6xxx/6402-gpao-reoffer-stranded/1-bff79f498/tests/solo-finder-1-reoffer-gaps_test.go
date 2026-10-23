// Repro for gnolang/gno#6402 at bff79f498d08059d599404035d2a46e54611ecd0.
//
// From a plain clone of gnolang/gno:
//
//	git fetch origin pull/6402/head && git checkout --detach bff79f498d08059d599404035d2a46e54611ecd0
//	cp <this file> contribs/gpao/zz_solofinder1_test.go
//	cd contribs/gpao && go test -count=1 -v -run 'SoloFinder1' .
//
// Two gaps in the reoffer pass (contribs/gpao/reoffer.go):
//
//   - AbsentImport: a submission whose first look could not finish (prepare
//     budget, as on onyx) and that imports a path submitted nowhere is never
//     verified again. firstImportNotLive counts "absent" as "not live", so the
//     pass parks it on blockedOn forever, while a fresh verification of the
//     same bytes rejects it at once.
//   - CapStatus: once maxReoffers is reached the pass forgets the submission,
//     but the status board still reports it pending, as it would one that is
//     going to be retried.
package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoloFinder1AbsentImport(t *testing.T) {
	chain := newInertChain(t)
	const path = "gno.land/r/test/wantsmissing"
	pkg := chainPackage(path, "package wantsmissing\n\nimport \"gno.land/p/test/neversubmitted\"\n\nfunc N(cur realm) int { return neversubmitted.Answer() }\n")
	chain.park(t, pkg)
	h := chain.submittedAt(t, path)

	o := chain.oracle(t)
	o.cfg.prepareBudget = time.Nanosecond // the first look cannot finish
	o.handleCandidate(t.Context(), candidate{mpkg: pkg, height: h})
	require.Equal(t, statusPending, o.status.get(path).Status, "premise: stranded")
	require.Contains(t, o.stranded, path, "premise: tracked")

	o.cfg.prepareBudget = defaultPrepareBudget // the oracle can finish now
	for i := 0; i < 5; i++ {
		o.reofferStranded(t.Context())
	}
	st := o.status.get(path)
	t.Logf("after 5 passes: status=%s attempt=%d reason=%q stranded=%+v",
		st.Status, st.Attempt, st.Reason, o.stranded[path])

	control := chain.oracle(t)
	control.handleCandidate(t.Context(), candidate{mpkg: pkg, height: h})
	cs := control.status.get(path)
	t.Logf("control, a fresh look at the same bytes: status=%s reason=%q", cs.Status, cs.Reason)
	require.Equal(t, statusRejected, cs.Status, "control: a look that finishes reaches a verdict")

	assert.Equal(t, statusRejected, st.Status,
		"README: one whose verification could not finish is verified again")
}

func TestSoloFinder1CapStatus(t *testing.T) {
	chain := newInertChain(t)
	const path = "gno.land/r/test/slow"
	pkg := chainPackage(path, "package slow\n\nfunc N(cur realm) int { return 1 }\n")
	chain.park(t, pkg)

	o := chain.oracle(t)
	o.cfg.prepareBudget = time.Nanosecond // every look ends unavailable
	o.handleCandidate(t.Context(), candidate{mpkg: pkg, height: chain.submittedAt(t, path)})
	for n := 0; n <= maxReoffers; n++ {
		o.reofferStranded(t.Context())
	}
	require.NotContains(t, o.stranded, path, "premise: no longer offered this run")

	st := o.status.get(path)
	t.Logf("after the cap: status=%s attempt=%d reason=%q", st.Status, st.Attempt, st.Reason)
	assert.Equal(t, statusGaveUp, st.Status,
		"the board should say it is no longer retried, as the overBudget and failedEnable caps do")
}
