# Findings in posting order, from round assemble: 6 to post, 0 SKIP, 0 refuted kept out

## contribs/gpao/reoffer.go:57 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L57) · Warning
State: CONFIRMED, band: Warning, angle: reach
TL;DR: each pass starts again at the first sorted path and stops once a candidate is queued, so under steady block traffic a later path waits maxReoffers passes behind every earlier offerable one
Check: cp tests/solo-finder-2-reoffer-cap-and-order.go contribs/gpao/solofinder2_test.go; go test -count=1 -run TestSoloFinder2EarlyPathDelaysLaterOnes -v ./contribs/gpao
Details: reofferStranded walks slices.Sorted(maps.Keys(o.stranded)) from the start each pass (reoffer.go:57) and returns at :58 once a candidate is queued, which an offer lasting up to the prepare plus verify budget makes likely on a busy chain. With aaa and zzz stranded and one candidate arriving per pass, zzz is first offered on pass 4, 30 minutes past the one interval the ADR states. track resets offers when the same path returns at a new height (reoffer.go:40-41), so a path resubmitted each pass keeps everything sorted after it waiting. The merge base never offers either path again, so the branch improves on it; the defect is the order the new pass walks.
Evidence: pass 1: early offers=1 tracked=true, late offers=0 / pass 2: early offers=2, late offers=0 / pass 3: early offers=3, late offers=0 / pass 4: early offers=0 tracked=false, late offers=1; Error: "4" is not less than or equal to "2"; --- FAIL: TestSoloFinder2EarlyPathDelaysLaterOnes (0.38s)
Artifact: tests/solo-finder-2-reoffer-cap-and-order.go

## contribs/gpao/reoffer.go:82 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L82) · Warning
State: CONFIRMED, band: Warning, angle: removed
TL;DR: reaching maxReoffers deletes the submission from stranded with only a stderr line, and /status keeps answering pending for a path gpao will not look at again this run
Check: cp tests/solo-finder-2-reoffer-cap-and-order.go contribs/gpao/solofinder2_test.go; go test -count=1 -run TestSoloFinder2StatusAfterReofferCap -v ./contribs/gpao
Details: reoffer.go:82-87 deletes the entry and calls errf, with no o.status.record. The sibling caps record statusGaveUp at their limit (oracle.go:560 for overBudget, :635 for failedEnable). Both siblings are 3 and count the first look, so they give up before maxReoffers is reached; the reoffer cap therefore ends exactly the errVerifyUnavailable case, whose reason reads 'the oracle could not run verification'. The merge base has no cap and no retry, so it too leaves the board at pending; the branch adds a give-up that the board does not show.
Evidence: status after the reoffer cap: {Path:gno.land/r/test/slow Status:pending Reason:the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns Attempt:0}; Error: Should not be: "pending"; --- FAIL: TestSoloFinder2StatusAfterReofferCap (1.11s)
Artifact: tests/solo-finder-2-reoffer-cap-and-order.go
State: CONFIRMED, band: Warning, angle: lines
TL;DR: the same defect as 1001 through the finder-1 harness: after the cap the board reads pending, not gave_up
Check: go test -count=1 -v -run TestSoloFinder1CapStatus ./contribs/gpao with tests/solo-finder-1-reoffer-gaps_test.go copied in
Details: Same lines and the same one edit as 1001 (a status.record of statusGaveUp beside the delete at reoffer.go:83), so one section carries both.
Evidence: after the cap: status=pending attempt=0 reason="the oracle could not run verification: ...-prepare-budget of 1ns"; expected: "gave_up" actual: "pending"; --- FAIL: TestSoloFinder1CapStatus (0.26s)
Artifact: tests/solo-finder-1-reoffer-gaps_test.go

## contribs/gpao/reoffer.go:133 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L133) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: isLive answers false for an import submitted nowhere as for a parked one, so a stranded submission importing an absent path waits on it for the whole run and never gets the rejection a finished look gives
Check: go test -count=1 -v -run TestSoloFinder1AbsentImport ./contribs/gpao with tests/solo-finder-1-reoffer-gaps_test.go copied in as solofinder1_test.go, in a scratch worktree at bff79f498
Details: isLive returns err == nil && meta.Status == vm.PackageStatusLive (reoffer.go:153), so an absent import and a parked one both read as not live and firstImportNotLive (reoffer.go:133) returns the absent path as blockedOn. Every later pass returns at reoffer.go:67 before queryPkgMeta, offers never moves, so neither the cap at :82 nor the meta check at :78 ever removes it, and the board keeps the first look's 'the oracle could not run verification'. The trigger is a first look ending errVerifyUnavailable, a prepare-budget overrun or a node fault, on a package with an absent import; a look that finishes rejects the same bytes with 'unknown import path'. The merge base has no reoffer pass and leaves the same submission pending for the run too, so the outcome predates the branch; what the branch adds is the README's 'one whose verification could not finish is verified again', which this case does not meet.
Evidence: after 5 passes: status=pending attempt=0 reason="the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns" stranded={height:2 offers:0 blockedOn:gno.land/p/test/neversubmitted}; control: status=rejected reason="...could not import gno.land/p/test/neversubmitted (unknown import path ...)"; expected "rejected" actual "pending"; --- FAIL: TestSoloFinder1AbsentImport (0.29s). The PR's five reoffer tests pass in the same run.
Artifact: tests/solo-finder-1-reoffer-gaps_test.go

## contribs/gpao/reoffer.go:97 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L97) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: a pass that finds the import holding a submission back stores it in blockedOn and leaves the board's reason at the first look's 'the oracle could not run verification'
Check: the TestSoloFinder1AbsentImport output: stranded={... blockedOn:...} beside the first look's reason; read reoffer.go:97-100
Details: reoffer.go:97-100 sets s.blockedOn and returns with no o.status.record, so the submitter polling /status is never told which import holds the package, which the errAwaitingDependency branch does name on a first look.
Evidence: stranded={height:2 offers:0 blockedOn:gno.land/p/test/neversubmitted} with reason="the oracle could not run verification: verifier unavailable: ..." (rerun of TestSoloFinder1AbsentImport)

## contribs/gpao/reoffer.go:58 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L58) · Suggestion
State: CONFIRMED, band: Suggestion, angle: claims
TL;DR: a reoffer tick that lands while the queue holds a candidate runs no path at all, and nothing runs the pass when the queue drains, so the next try is a full reofferInterval later
Check: read runVerifier (oracle.go select on o.candidates and reoffer) and reofferStranded line 58
Details: runVerifier takes the tick in its select (oracle.go diff: case <-reoffer: o.reofferStranded(ctx)); reofferStranded returns at reoffer.go:58 'if ctx.Err() != nil || len(o.candidates) > 0' before its first path. A time.Ticker channel holds one tick, so the tick is spent and the next comes 10 minutes later; during a --start-height catch-up the queue is rarely empty at a tick. A pass held pending until the queue drains keeps 'new blocks go first' and loses no tick. Read-settled, not run.
Evidence: reoffer.go:58 `if ctx.Err() != nil || len(o.candidates) > 0 {` / `return`; oracle.go runVerifier `case <-reoffer: o.reofferStranded(ctx)`; reofferInterval = 10 * time.Minute (reoffer.go:22)

## contribs/gpao/reoffer.go:112 [gh](https://github.com/gnolang/gno/blob/bff79f498d08059d599404035d2a46e54611ecd0/contribs/gpao/reoffer.go#L112) · Suggestion
State: CONFIRMED, band: Suggestion, angle: reach
TL;DR: submissionAt reads Block and BlockResults again for each stranded submission, so N submissions from one block cost N block reads on the pass after they strand and on each offer
Check: read reoffer.go reoffer and submissionAt: addedPackages(ctx, s.height) per path, no per-pass map keyed by height
Details: submissionAt (reoffer.go:111-122) calls addedPackages per path; reofferStranded keeps a live map for meta queries and nothing for blocks. The finder's 'on every pass' overstates it: a blocked entry returns at :67 before the block read once blockedOn is set, and an unavailable one is read at most once per offer, maxReoffers in all. So the cost is two RPCs per submission per offer on a 10-minute pass, bounded per submission.
Evidence: reoffer.go:112 `added, err := o.addedPackages(ctx, height)` inside submissionAt, called from reoffer at :88 once per path; :67 `if s.blockedOn != "" && !o.isLive(ctx, s.blockedOn, live) { return }` precedes it
