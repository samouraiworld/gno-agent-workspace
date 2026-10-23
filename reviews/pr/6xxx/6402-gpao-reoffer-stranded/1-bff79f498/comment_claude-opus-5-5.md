# Review: [#6402](https://github.com/gnolang/gno/pull/6402)
Event: COMMENT
Verdict: REQUEST CHANGES. Two Warnings in the new reoffer pass block: the cap gives up without telling `/status`, and every pass restarts at the first sorted path; both are code this branch adds.
Model: claude-opus-5-5, standard review, solo shape: two finders, then one judge and writer
Commit: bff79f498d08059d599404035d2a46e54611ecd0
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6402 bff79f498`
Round: 1. 2 finders, no separate reflector (the judge answered the completeness question), 7 candidates, each Warning re-run from its finder's artifact by a judge that was not its finder, the Nit and the Suggestions judged by read; 0 refuted. Event is COMMENT because the round goes out as an AI review.

## Body
> AI review, claude-opus-5-5, standard review, reviewed at [bff79f4](https://github.com/gnolang/gno/pull/6402/commits/bff79f498d08059d599404035d2a46e54611ecd0), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6402-gpao-reoffer-stranded/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6402-gpao-reoffer-stranded/1-bff79f498/claims.md) · Status: REQUEST CHANGES

## contribs/gpao/reoffer.go:82 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L82) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L82) · Warning

`/status` keeps answering `pending` once this cap removes the submission for the rest of the run, while the sibling caps [`maxOverBudgetAttempts`](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/oracle.go#L560) and [`maxEnableAttempts`](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/oracle.go#L635) record `gave_up`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6402 -R gnolang/gno
cat > contribs/gpao/solofinder2_test.go <<'EOF'
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
EOF
(cd contribs/gpao && go test -count=1 -run 'TestSoloFinder2StatusAfterReofferCap' -v .)
rm contribs/gpao/solofinder2_test.go
```

The test fails because the board still reads `pending` once the cap has removed the submission:

```text
=== RUN   TestSoloFinder2StatusAfterReofferCap
    solofinder2_test.go:52: status after the reoffer cap: {Path:gno.land/r/test/slow Status:pending Reason:the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns Attempt:0}
        	Error:      	Should not be: "pending"
--- FAIL: TestSoloFinder2StatusAfterReofferCap (1.11s)
```

Both sibling caps are 3 and count the first look, so they record `gave_up` before `maxReoffers` is reached; this cap ends only the `errVerifyUnavailable` outcome, the prepare-budget overrun the ADR cites.

</details>

## contribs/gpao/reoffer.go:57 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L57) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L57) · Warning

This loop restarts at the first sorted path on every pass, so under steady block traffic a late path waits three passes behind each earlier path that gets offered.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6402 -R gnolang/gno
cat > contribs/gpao/solofinder2_test.go <<'EOF'
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
EOF
(cd contribs/gpao && go test -count=1 -run 'TestSoloFinder2EarlyPathDelaysLaterOnes' -v .)
rm contribs/gpao/solofinder2_test.go
```

The test fails because `gno.land/r/test/zzz` is first offered on pass 4, with one candidate arriving during each pass:

```text
=== RUN   TestSoloFinder2EarlyPathDelaysLaterOnes
    solofinder2_test.go:97: pass 1: early offers=1 tracked=true, late offers=0
    solofinder2_test.go:97: pass 2: early offers=2 tracked=true, late offers=0
    solofinder2_test.go:97: pass 3: early offers=3 tracked=true, late offers=0
    solofinder2_test.go:97: pass 4: early offers=0 tracked=false, late offers=1
        	Error:      	"4" is not less than or equal to "2"
--- FAIL: TestSoloFinder2EarlyPathDelaysLaterOnes (0.38s)
```

`track` resets `offers` when the same path comes back at a new height ([reoffer.go:40-41](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L40-L41)), so a path resubmitted every pass keeps each path sorted after it waiting for the whole run.

</details>

## contribs/gpao/reoffer.go:97 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L97) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L97) · Nit

Nit: This branch records `blockedOn` without updating the status reason, so `/status` keeps saying `the oracle could not run verification` and never names the blocking import.

## contribs/gpao/reoffer.go:153 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L153) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L153) · Suggestion

Suggestion: reject on an import the chain holds nowhere, as a finished first look does, since this treats it as parked and its importer waits for the run uncounted.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6402 -R gnolang/gno
cat > contribs/gpao/solofinder1_test.go <<'EOF'
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
EOF
(cd contribs/gpao && go test -count=1 -run 'TestSoloFinder1AbsentImport' -v .)
rm contribs/gpao/solofinder1_test.go
```

The test fails because five passes leave the submission pending and blocked on the absent path, while a fresh look at the same bytes rejects them:

```text
=== RUN   TestSoloFinder1AbsentImport
    solofinder1_test.go:47: after 5 passes: status=pending attempt=0 reason="the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns" stranded={height:2 offers:0 blockedOn:gno.land/p/test/neversubmitted}
    solofinder1_test.go:53: control, a fresh look at the same bytes: status=rejected reason="gno.land/r/test/wantsmissing/wantsmissing.gno:3:8: could not import gno.land/p/test/neversubmitted (unknown import path \"gno.land/p/test/neversubmitted\")"
        	Error:      	Not equal: 
        	            	expected: "rejected"
        	            	actual  : "pending"
--- FAIL: TestSoloFinder1AbsentImport (0.29s)
```

Every later pass returns at [reoffer.go:67](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L67) before `queryPkgMeta`, so `offers` never moves and neither the cap nor the height check removes the entry.

</details>

## contribs/gpao/reoffer.go:58 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L58) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L58) · Suggestion

Suggestion: Run the pass once the queue drains, because this return drops the whole tick whenever a candidate is queued and the next tick is a full `reofferInterval` later.

## SKIP contribs/gpao/reoffer.go:112 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L112) · [↗](../../../../../.worktrees/gno-review-6402/contribs/gpao/reoffer.go#L112) · Suggestion

Suggestion: `submissionAt` reads the block and its results again for each stranded submission it holds, so a per-pass cache of `addedPackages` keyed by height would read each block once.

Not posted: the reads are bounded per submission, two RPCs on the first pass and on each offer, at most `maxReoffers` + 1 in a run, on a 10-minute pass.
