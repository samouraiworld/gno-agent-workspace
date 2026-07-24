# PR [#5996](https://github.com/gnolang/gno/pull/5996): fix(tm2/store): guard multiStore.lastCommitID against the query connection

URL: https://github.com/gnolang/gno/pull/5996
Author: davd-gzl | Base: master | Files: 3 | +81 -9
Reviewed by: davd-gzl | Model: claude-opus-4-8 (xhigh, deep, inline) | Commit: 4b521105b (latest)
Local worktree: `.worktrees/gno-split-infra` on `fix/tm2-store-lastcommitid-race`

**TL;DR:** A query arriving without an explicit height reads the store's last committed version and hash, and it does so on a mutex that is deliberately independent of the one the consensus goroutine holds while committing. So a query could read a `CommitID` mid-update and pair a fresh version number with the previous block's hash. This adds a read/write mutex around that one field so the pair is always read and written atomically.

**Verdict: APPROVE** — the race is real and reachable on the production wiring, the guard covers every access to the field, no lock-order inversion is introduced, and the regression test fails under `-race` on a reverted fix (0 Critical, 0 Warning).

## Summary

`multiStore.lastCommitID` is a `{Version, Hash}` pair written by the consensus goroutine in `Commit` and `LoadVersion`, and read by the query connection. The query connection runs on its own mutex: [`proxy/client.go:26`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/bft/proxy/client.go#L26) gives `NewReadOnlyABCIClient` a `queryMtx` documented as "independent mutex for the query connection", so `Query` and `Commit` are concurrent by design. When a query omits its height, both [`handleQueryStore`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/sdk/baseapp.go#L498) and `handleQueryCustom` at [`baseapp.go:525`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/sdk/baseapp.go#L525) inject `app.LastBlockHeight()`, which reads `cms.LastCommitID().Version`. Nothing serialised that read against `Commit`, so a reader could observe a torn pair: the new version next to the old hash, or a partially written struct.

The fix adds `lastCommitIDMu sync.RWMutex` and routes every read through `LastCommitID()` under `RLock` and every write through a new `setLastCommitID` under `Lock`.

## Glossary

- query connection: the ABCI connection serving read-only queries, run on `queryMtx` independent of the consensus/mempool mutex, so queries do not block block production.
- CommitID: the `{Version, Hash}` pair identifying the last committed block; the version is the chain height and the hash is the app-state commitment.
- torn read: reading a multi-field value while another goroutine writes it, so the fields observed never coexisted as a consistent state.

## Fix

[`store.go:53-64`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/store/rootmulti/store.go#L53-L64) adds the mutex beside the field with a comment naming the two goroutines and the query path. Every prior raw access is now funnelled through two accessors: `LastCommitID()` takes `RLock`, `setLastCommitID` takes `Lock`, and `Commit` reads the previous version once via `LastCommitID().Version` rather than touching the field directly. The load-bearing property is completeness: a partial guard on a race looks solved while still tearing. Grepping the field confirms the only two remaining raw references, `store.go:249` and `:254`, are the bodies of the two accessors themselves.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

None.

## Missing Tests

None. [`TestLastCommitIDConcurrentWithCommit`](https://github.com/gnolang/gno/blob/4b521105b/tm2/pkg/store/rootmulti/store_test.go) models the production wiring: a reader loop calling `LastCommitID()` concurrent with a committing writer, documented as `Run under -race`.

## Suggestions

- **[measure the commit-path cost]** the guard sits on `Commit`, which runs once per block. An `RWMutex` write lock around a struct assignment is nanoseconds and cannot plausibly show up against a block commit, but the PR asserts free without a number. A one-line benchmark would close that. Out of scope; not worth blocking.

## Verified

- The race is real and reachable, not theoretical. `proxy/client.go:26` hands the query connection an independent `queryMtx`, and both `handleQueryStore` and `handleQueryCustom` inject `app.LastBlockHeight()` when a query omits its height, which reads `LastCommitID().Version` concurrent with `Commit` writing it. The read is not otherwise serialised against the consensus mutex.
- The guard is complete. Grepping every reference to `lastCommitID` and any field it is copied from leaves exactly two raw accesses, both inside the accessors; every other read and write goes through `LastCommitID()`/`setLastCommitID`. `Commit` reads the prior version via `LastCommitID().Version` rather than the raw field.
- No lock-order inversion. `lastCommitIDMu` is a leaf: the accessors take it, do a struct copy or assignment, and release it, with no other lock acquired inside and no call back into store code, so it cannot nest against the store's other mutexes or bptree's `pruneMu`.
- The test fails without the fix, and does so under `-race` as documented. Reverting the two accessor bodies to raw field access and running `go test -race -run LastCommitID -count=1` reports `WARNING: DATA RACE` and `--- FAIL: TestLastCommitIDConcurrentWithCommit`, while the same test passes without `-race` (a data-race guard is not expected to fail on value alone). Restoring the accessors turns it green both ways.
- Green at 4b521105b: `go test -race ./tm2/pkg/store/...` passes across all nine store packages; `gofmt -l` clean on the three changed files. The earlier `main / lint` failure on this branch (`go fix` wanting `wg.Go`) was fixed in this same head.

## Open questions

None.
