# gpao: `reofferStranded` verifies again the packages it left pending
Written by claude-opus-5-5.
PR: [gnolang/gno#6402](https://github.com/gnolang/gno/pull/6402)

## TLDR

gpao is the program that checks newly uploaded packages and switches them on when they compile. When it could not finish checking a package, or the package needed another one that was not switched on yet, it left the package waiting and never came back to it unless someone uploaded it again or restarted gpao. This change makes gpao remember those waiting packages and look at them again every 10 minutes, so a package whose dependency goes live is checked and switched on without anyone stepping in.

## Pending outcomes

`handleCandidate` ends every look at a submission in one of a few states on the status board. Two of those states are final, the package switched on or refused for its code. Pending covers three cases where nobody has judged the bytes yet. `errVerifyUnavailable` means the oracle could not run verification, for example the import closure took longer than `--prepare-budget` to fetch. `errAwaitingDependency` means the package imports one that is parked, submitted but not enabled, so the type check cannot see it. A failed enable means the package passed but the `MsgEnablePackage` transaction did not land. The over-budget and failed-enable cases already had counters, `maxOverBudgetAttempts` and `maxEnableAttempts`, each 3, that turn pending into `gave_up` once they run out.

## Why a pending package was never reached again

The block reader in `run` reads each height once and only moves forward, and `processBlock` is the only thing that feeds the verifier. A package left pending was therefore never offered again: the retry was a resubmission or a restart with `--start-height` at or below its block. One stuck package also strands everything that imports it, since each importer is left pending on a parked import that never goes live. The PR's decision record gives the case that prompted it: two packages overran the prepare budget on a test network and eight importers stayed pending behind them.

## `stranded` and `track`

The oracle gains a map, `stranded`, from package path to a `strandedSubmission`: the submitting height, how many times it has been offered again, and `blockedOn`, the import that held it back on the last look. The bytes are not kept, so a submitter cannot grow gpao's memory with content; they are read back from the block instead.

`handleCandidate` now runs `defer o.track(path, c.height)` once it knows the bytes are new. `track` reads the status board after the look: pending keeps or refreshes the entry, and any other status deletes it. A submission that comes back at a new height starts again with zero offers.

```go
// after: contribs/gpao/reoffer.go
func (o *oracle) track(path string, height int64) {
	if o.status.get(path).Status != statusPending {
		delete(o.stranded, path)
		return
	}
	s := o.stranded[path]
	if s.height != height {
		s = strandedSubmission{height: height}
	}
	s.blockedOn = ""
	o.stranded[path] = s
}
```

## The reoffer pass

`run` starts a second ticker, `reofferInterval`, 10 minutes, and hands its channel to `runVerifier`, which now selects on it beside the candidate queue. A tick runs `reofferStranded` on the verifier goroutine itself, so the maps it touches stay single-owner and the block reader never waits on it.

`reofferStranded` walks the stranded paths in sorted order and stops as soon as the candidate queue holds anything, so new blocks go first. For each path, `reoffer`:

1. skips it cheaply if `blockedOn` is set and that import is still not live;
2. asks `vm/qpkgmeta_json` about the path itself, and forgets it if the chain no longer holds it inert at that height, because someone enabled it or a newer submission replaced it, which reaches the verifier from its own block;
3. forgets it once it has been offered `maxReoffers` times, 3;
4. reads its bytes back with `submissionAt`;
5. records `blockedOn` and stops if one of its chain imports is not live;
6. otherwise counts the offer and calls `handleCandidate` with the original height.

## Live imports and `isLive`

`firstImportNotLive` parses the package's production imports, the same set the verifier compiles, and returns the first user-library import that `isLive` says is not running on chain. `isLive` treats a genesis package the import cache already holds as live without asking, and otherwise asks `vm/qpkgmeta_json` once per pass, keeping the answer in a per-pass map. A node that does not answer counts as not live until the next pass. Only the package's own imports are asked about, since a live package's imports are live already.

## Reading the bytes back from the block

`processBlock` is split in two. `addedPackages` holds the old decoding: it fetches the block and its results, refuses a block whose results do not pair one for one with its transactions, and returns the package of every `MsgAddPackage` whose `DeliverTx` succeeded. `processBlock` enqueues them, and `submissionAt` takes the last one at the path, which is the one the chain stores when two land in the same block. The reoffered bytes therefore go through exactly the checks a first look does.

## Before and after

**Before:** a package left pending stayed pending until it was resubmitted or gpao restarted at or below its block; the comments in `handleCandidate` said so in each pending branch.

**After:** `track` remembers it and `reofferStranded` offers it again on a 10-minute tick once its imports are live, up to 3 times per submission per run; those comments now say "a later offer".

## What a user notices

- A package that waited on a parked import is enabled within about one reoffer tick after the import goes live, without a resubmission. This is per gpao process.
- A package whose verification could not finish is tried again, up to 3 more times per run.
- An operator sees `gpao: offering %q again, submitted at height %d` in the log for each retry.

## What it leaves open

- A restart forgets what was pending; `--start-height` at or below the oldest such block recovers it, as before.
- A package that overran the prepare budget meets the same budget again; only a larger `--prepare-budget` or faster queries help it.
- A redeploy parked over a live private realm is not offered again, because `vm/qpkgmeta_json` reports the live package for that path.

## Words used here

| Name | What it is |
| --- | --- |
| `stranded` | Map on the oracle from path to the submission it last left pending; touched only by the verifier goroutine. |
| `strandedSubmission` | Height, offer count and `blockedOn` for one stranded path; no bytes. |
| `track` | Deferred in `handleCandidate`; keeps a pending submission in `stranded` and deletes anything else. |
| `reofferStranded` | The pass run on each reoffer tick, in sorted path order, stopping when a candidate is queued. |
| `reofferInterval` | 10 minutes, the reoffer ticker's period. |
| `maxReoffers` | 3, how often one submission is offered again per run. |
| `addedPackages` | Decodes one block into the packages its successful `MsgAddPackage` transactions added. |
| `submissionAt` | The last package a block added at a path. |
| `firstImportNotLive` / `isLive` | The import check run before an offer, one `vm/qpkgmeta_json` query per import per pass. |
| parked | Submitted with `MsgAddPackage` under the `inert` policy and not yet enabled. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file.

### 1. `hot` · The pass and one offer, [`contribs/gpao/reoffer.go`](https://github.com/gnolang/gno/pull/6402/files#diff-4cb288da1e4a7c40e5ad010395a43dc6edaec29e17e90f075a52a4eea86e6ba9R55-R106)
Decides which stranded submission is offered again, which is forgotten, and when the pass yields to new blocks.
**If wrong:** a package is offered again forever, or never, or new blocks wait behind retries.
```go
// after
for _, path := range slices.Sorted(maps.Keys(o.stranded)) {
	if ctx.Err() != nil || len(o.candidates) > 0 {
		return
	}
	o.reoffer(ctx, path, live)
}
```

### 2. `hot` · Remembering a pending look, [`contribs/gpao/reoffer.go`](https://github.com/gnolang/gno/pull/6402/files#diff-4cb288da1e4a7c40e5ad010395a43dc6edaec29e17e90f075a52a4eea86e6ba9R34-R45)
`track` keeps pending submissions and drops every verdict.
**If wrong:** a verdict is offered again, or a pending one is lost.

### 3. `hot` · The import check, [`contribs/gpao/reoffer.go`](https://github.com/gnolang/gno/pull/6402/files#diff-4cb288da1e4a7c40e5ad010395a43dc6edaec29e17e90f075a52a4eea86e6ba9R124-R156)
`firstImportNotLive` and `isLive` decide whether an offer can reach a verdict.
**If wrong:** an importer is offered while its import is parked and comes back pending, spending an offer.

### 4. `hot` · The verifier's select and the deferred `track`, [`contribs/gpao/oracle.go`](https://github.com/gnolang/gno/pull/6402/files#diff-b2b053b200c5d256780b2bfcf7679b12d1e3570671892f502ffdbe6bf7be8557R470-R481)
Runs the pass on the verifier goroutine between candidates; `defer o.track` sits at line 538.
**If wrong:** the stranded map is touched from two goroutines, or the pass blocks the queue.

### 5. `warm` · The decision record, [`gno.land/adr/pr6402_gpao_reoffer_stranded.md`](https://github.com/gnolang/gno/pull/6402/files#diff-5651cf5a3cae4806e9ede412d0cdeab1b05b31f2a827f01e4d655ca660b3837bR1-R27)
States why the bytes are re-read rather than kept and why the chain's inert listing was not used.
**If wrong:** the reasoning a later change relies on no longer matches the code.

### 6. `cold` · The tests, [`contribs/gpao/reoffer_test.go`](https://github.com/gnolang/gno/pull/6402/files#diff-2f6ab2bfe1343aff187e9e90ecb279c4650e407f059bd0c7ea6b92f4447b9f73R1-R158)
Pin the blocked-import short cut, the genesis cache, the cap, forgetting an enabled path, and the tick.
**If wrong:** a regression in the pass goes unnoticed.

### 7. `cold` · Stranded packages, [`contribs/gpao/README.md`](https://github.com/gnolang/gno/pull/6402/files#diff-2b039c2695b0957e35b702da1981db3e47b58ea1e4b75ea271780b23265e72f8R192-R209)
The operator's account of the retry and of restarting.
**If wrong:** an operator restarts when the pass would have retried, or waits when it will not.

</details>
