# Claims: gnolang/gno#6402 https://github.com/gnolang/gno/pull/6402 round 1, bff79f498, claude-opus-5-5, solo review

Round shape: solo round, 2 finders and one judge and writer, every Critical and Warning run, the rest read

## Candidates

| # | State | Band | file:line | Check | Observed | Artifact | Tier | Mechanism |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | CONFIRMED | Suggestion | contribs/gpao/reoffer.go:153 | go test -count=1 -v -run TestSoloFinder1AbsentImport ./contribs/gpao with tests/solo-finder-1-reoffer-gaps_test.go copied in as solofinder1_test.go, in a scratch worktree at bff79f498 | after 5 passes: status=pending attempt=0 reason="the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns" stranded={height:2 offers:0 blockedOn:gno.land/p/test/neversubmitted}; control: status=rejected reason="...could not import gno.land/p/test/neversubmitted (unknown import path ...)"; expected "rejected" actual "pending"; --- FAIL: TestSoloFinder1AbsentImport (0.29s). The PR's five reoffer tests pass in the same run. | tests/solo-finder-1-reoffer-gaps_test.go | hot | absent-import-read-as-parked |
| 2 | CONFIRMED | Warning | contribs/gpao/reoffer.go:82 | go test -count=1 -v -run TestSoloFinder1CapStatus ./contribs/gpao with tests/solo-finder-1-reoffer-gaps_test.go copied in | after the cap: status=pending attempt=0 reason="the oracle could not run verification: ...-prepare-budget of 1ns"; expected: "gave_up" actual: "pending"; --- FAIL: TestSoloFinder1CapStatus (0.26s) | tests/solo-finder-1-reoffer-gaps_test.go | hot | give-up-not-on-status-board |
| 3 | CONFIRMED | Suggestion | contribs/gpao/reoffer.go:58 | read runVerifier (oracle.go select on o.candidates and reoffer) and reofferStranded line 58 | reoffer.go:58 `if ctx.Err() != nil \|\| len(o.candidates) > 0 {` / `return`; oracle.go runVerifier `case <-reoffer: o.reofferStranded(ctx)`; reofferInterval = 10 * time.Minute (reoffer.go:22) |  | hot |  |
| 4 | CONFIRMED | Nit | contribs/gpao/reoffer.go:97 | the TestSoloFinder1AbsentImport output: stranded={... blockedOn:...} beside the first look's reason; read reoffer.go:97-100 | stranded={height:2 offers:0 blockedOn:gno.land/p/test/neversubmitted} with reason="the oracle could not run verification: verifier unavailable: ..." (rerun of TestSoloFinder1AbsentImport) |  | hot |  |
| 1001 | CONFIRMED | Warning | contribs/gpao/reoffer.go:82 | cp tests/solo-finder-2-reoffer-cap-and-order.go contribs/gpao/solofinder2_test.go; go test -count=1 -run TestSoloFinder2StatusAfterReofferCap -v ./contribs/gpao | status after the reoffer cap: {Path:gno.land/r/test/slow Status:pending Reason:the oracle could not run verification: verifier unavailable: the verifier did not have its sources within the -prepare-budget of 1ns Attempt:0}; Error: Should not be: "pending"; --- FAIL: TestSoloFinder2StatusAfterReofferCap (1.11s) | tests/solo-finder-2-reoffer-cap-and-order.go | hot | give-up-not-on-status-board |
| 1002 | CONFIRMED | Warning | contribs/gpao/reoffer.go:57 | cp tests/solo-finder-2-reoffer-cap-and-order.go contribs/gpao/solofinder2_test.go; go test -count=1 -run TestSoloFinder2EarlyPathDelaysLaterOnes -v ./contribs/gpao | pass 1: early offers=1 tracked=true, late offers=0 / pass 2: early offers=2, late offers=0 / pass 3: early offers=3, late offers=0 / pass 4: early offers=0 tracked=false, late offers=1; Error: "4" is not less than or equal to "2"; --- FAIL: TestSoloFinder2EarlyPathDelaysLaterOnes (0.38s) | tests/solo-finder-2-reoffer-cap-and-order.go | hot | reoffer-pass-restarts-at-first-path |
| 1003 | CONFIRMED | Suggestion | contribs/gpao/reoffer.go:112 | read reoffer.go reoffer and submissionAt: addedPackages(ctx, s.height) per path, no per-pass map keyed by height | reoffer.go:112 `added, err := o.addedPackages(ctx, height)` inside submissionAt, called from reoffer at :88 once per path; :67 `if s.blockedOn != "" && !o.isLive(ctx, s.blockedOn, live) { return }` precedes it |  | hot |  |

Hit rate per tier, from the rows above: hot 7/7 confirmed over 2 files, warm 0/0 confirmed over 1 files, cold 0/0 confirmed over 2 files.

### Settled by the finder, no verifier

| Angle | file:line | Suspected | Settled by |
| --- | --- | --- | --- |
| lines | contribs/gpao/reoffer.go:111 | the pass might verify bytes other than those parked at the recorded height | submissionAt reuses addedPackages (DeliverTx IsOK filter, block order) and takes the last add at the path; keeper_inert.go:565 'Height is the block the package was submitted in', and reoffer.go:78 forgets the entry once meta.Height differs, so a later replacement is never verified under the old height |
| catalog | contribs/gpao/reoffer.go:56 | o.stranded and the new ticker path racing the block reader | go test -race -count=1 -run 'Stranded\|Reoffer\|CachedGenesis' ./contribs/gpao -> ok github.com/gnolang/gno/contribs/gpao 17.319s, no DATA RACE |
| reach | contribs/gpao/reoffer.go:67 | The blockedOn check runs before the meta check, so an entry whose blocking import never goes live is kept even after the chain moves its path on | Deliberate and pinned: reoffer_test.go TestStrandedImporterWaitsWhileItsImportIsParked asserts metaQueries == 1, "a pass asks about the parked import alone"; a replacement reaches track from its own block and resets the entry |
| removed | contribs/gpao/README.md:168 | Sibling sentences still saying nothing re-offers a pending package, or naming a restart or resubmission as the only retry | grep -rn -i 're-offer\\|resubmi\\|restart\\|start-height\\|nothing re\\|no-op\\|will be retried' contribs/gpao (non-test): remaining hits are oracle.go:122 (cap per resubmission, still true), verifyone.go:319 (about rejections), oracle.go:599/609 and README:315 (spend limit, which track does not keep), README:203-204 (the new restart caveat); onyx and mainnet deployment READMEs carry no such sentence |
| removed | contribs/gpao/oracle.go:385 | processBlock split into addedPackages plus enqueue changes what a failed block or a cancelled enqueue does | Read the diff: BlockResults failure and the pairing check still return before anything is queued, undecodable txs are still skipped, and enqueue's error still returns from processBlock; callers oracle.go:370 and processblock_test.go:104,124,133 are unchanged |

## Completeness

- Thin angles: none of the seven candidates came from the tests angle; the PR's five reoffer tests pass at the head, and the finders' artifacts fail against code the diff adds, so no added test is credited for a case it does not pin.
- Removed-sentence sweep: the diff deletes every "nothing re-offers" and "restart or resubmit" sentence it rewrites; `grep -niE 'resubmi|nothing re-?offers|restart with|start-height at or below'` over `contribs/gpao/*.go` and `README.md` at the head returns `README.md:204` (the new Stranded packages text) and `oracle.go:122-123,526`, whose `maxOverBudgetAttempts` and `seen` rationale still holds. No stale sibling.
- Sibling caps: `maxOverBudgetAttempts = 3` (`oracle.go:131`) and `maxEnableAttempts = 3` (`oracle.go:144`) record `statusGaveUp` at `oracle.go:560` and `:635` before `maxReoffers` is reached, so the reoffer cap ends only the `errVerifyUnavailable` case; folded into 1001 and 2, one section at `reoffer.go:82`.
- Catalog: the diff touches `contribs/gpao` and `gno.land/adr` only, a tooling change, which the project rule exempts from the invariant catalog; no class applies.
- Goroutine ownership: `stranded`, `track`, `reofferStranded` and `isLive` run only on the verifier goroutine through the `runVerifier` select (`oracle.go:470-481`), like `seen` and `overBudget`.
- One edit per section: 1001 and 2 close with one `status.record(path, statusGaveUp, ...)` beside the delete at `reoffer.go:83`. 1002 (`:57`) and 3 (`:58`) are separate edits, a resume cursor and a pass re-armed when the queue drains.
- 1003 ships SKIP: the block reads are bounded per submission, the first pass plus one per offer, since `reoffer.go:67` returns before `submissionAt` once `blockedOn` is set.
- Candidate 1 predates the branch in outcome: the merge base has no reoffer pass (`grep -c reoffer` over its `contribs/gpao/*.go` returns 0 in every file) and leaves the same submission pending for the run; the branch's README line "one whose verification could not finish is verified again" is what the case fails.
- PR author is `gfanton`, not the user, so a verdict is in order; the draft's Event is COMMENT for `post as an AI`.

## Retro

- Band moved by the parent: row 1 from Warning to Suggestion. The merge base leaves the same submission pending for the run, and treating an absent import as one that may still be enabled is a design call, so the rejection is offered as a choice rather than claimed as the defect.

- What failed: nothing died, and no agent left the session's model, per `./scripts/review-retro.py --model claude-opus-5-5` exiting 0. The parent moved the absent-import Warning from `reoffer.go:133` to `:153`, the line the fix edits, and rewrote two sentences prose-check flagged.
- What worked: both finders reached the cap-status defect independently, rows 2 and 1001, and every one of 7 candidates held; the three Warnings each ship a test that fails at the head.
- Hit rate per tier: hot 7 of 7 confirmed over 2 files, warm and cold none over 1 and 2; no cold file carried a candidate, so the weights stand untested.
- Upgrade: the same `--brief` placement as round 1 of 6400, named to the user rather than applied.
- Cost: 3 agents, 60 turns, 129k output, 6.8M cache read, about $7.29.
