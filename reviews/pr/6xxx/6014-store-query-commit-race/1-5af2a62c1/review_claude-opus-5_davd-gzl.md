# PR [#6014](https://github.com/gnolang/gno/pull/6014): fix(tm2): prevent a race between store queries and commits

URL: https://github.com/gnolang/gno/pull/6014
Author: notJoon | Base: master | Files: 3 | +63 -10
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 5af2a62c1 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-6014 5af2a62c1`

**TL;DR:** A gno.land node answers RPC queries on one connection while it commits blocks on another. A query that does not name a block height asks the node "what is the latest height?", and the commit path was updating that number without any synchronization, so the two threads touched the same memory at the same time. This PR publishes the number through an atomic pointer so the read is well-defined.

**Verdict: NEEDS DISCUSSION** — the fix is correct and the race is real, but the identical change is already inside [PR #6018](https://github.com/gnolang/gno/pull/6018), so the merge order is a maintainer call; separately the new guard cannot run under either of this repo's test entry points, and its comment claims a safety property that the same query path still violates (2 Warnings, 1 Nit, 1 Suggestion).

## Verify first

- [`tm2/pkg/store/rootmulti/store.go:236-241`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L236-L241) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L236-L241) — every read of the commit ID must now go through this accessor. Confirm with `grep -n lastCommitID tm2/pkg/store/rootmulti/store.go`: the only direct touches left should be the declaration, three `Store` calls in `LoadVersion` and `Commit`, and this one `Load`.
- [`tm2/pkg/store/rootmulti/store.go:257-263`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L257-L263) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L257-L263) — `Commit` reads the version and writes it back in two separate atomic operations, which is only safe with a single writer. Confirm `Commit` is reached only from [`baseapp.go:1001`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L1001) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L1001) on the consensus connection; a second writer would skip or duplicate a version.
- [`tm2/pkg/sdk/baseapp_test.go:1544-1588`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp_test.go#L1544-L1588) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp_test.go#L1544-L1588) — nothing else proves the fix, so run it yourself against the reverted diff: `git stash push tm2/pkg/store/rootmulti/store.go && CGO_ENABLED=1 go test -race -run TestStoreQueryConcurrentWithCommit ./tm2/pkg/sdk/`. The race must appear, and disappear once unstashed.

## Summary

The ABCI proxy hands the query connection its own mutex, [`queryMtx`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/bft/proxy/client.go#L25-L26) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/bft/proxy/client.go#L25-L26), separate from the one consensus and mempool share, so a query executes while a block commits. A `.store` query with no height gets one injected at [`baseapp.go:498`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L498) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L498) from [`LastBlockHeight`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L166-L168) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L166-L168), which read the plain `multiStore.lastCommitID` struct that `Commit` assigned with no synchronization. The same field also feeds [`Info`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L323-L327) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L323-L327), which reads `Version` and `Hash` as a pair and could therefore have observed a version from one block beside a hash from another. The PR turns the field into an [`atomic.Pointer[types.CommitID]`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L54) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L54) and publishes a fully built value in one store, so a reader sees either the whole old commit ID or the whole new one.

## Diagram

```
   consensus conn (mtx)                        query conn (queryMtx)
   BaseApp.Commit  baseapp.go:1001             BaseApp.Query  baseapp.go:445
        │                                            │
        │                                     handleQueryStore
        │                                       │            │
   multiStore.Commit                            │            │
     ├─ MultiWrite ──► iavl MutableTree ◄───────┼── (2) multiStore.Query
     │   node.clone()      (live, shared)       │       tree.GetVersioned
     │                                          │       ── STILL RACING ──
     │                                          │
     └─ Store(&commitID) ─► lastCommitID ◄── (1) LastBlockHeight
                            atomic.Pointer       ── FIXED BY THIS PR ──
```

Edge (1) is the whole diff. Edge (2) is untouched: `.store` queries read the live tree, unlike custom queries, which go through the committed snapshot at [`baseapp.go:533`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L533) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L533).

## Fix

Before, `lastCommitID` was a `types.CommitID` value assigned in three places and read directly in three more. After, it is an atomic pointer written only by [`LoadVersion`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L225-L226) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L225-L226) and [`Commit`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L295-L299) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L295-L299), and read only through [`LastCommitID`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L236-L241) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L236-L241). The load-bearing constraint is that both publishers build the value first and store the address once, so no reader can observe a half-updated pair; the version arithmetic inside `Commit` stays a plain read-then-write, which holds only because the consensus connection is the sole writer. The last direct field read in the package's own tests moves to the accessor at [`store_test.go:471`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store_test.go#L471) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store_test.go#L471).

## Critical (must fix)

None.

## Warnings (should fix)

- **[the guard cannot run under either of this repo's test entry points]** `tm2/pkg/sdk/baseapp_test.go:1544` — the new test only detects anything under `-race`, which the tm2 CI job never passes and `make test` cannot pass, so reverting the atomic pointer leaves every check green.
  <details><summary>details</summary>

  The tm2 job runs [`go test -covermode=set -timeout 30m ...`](https://github.com/gnolang/gno/blob/5af2a62c1/.github/workflows/_ci-go.yml#L124) · [↗](../../../../../.worktrees/gno-review-6014/.github/workflows/_ci-go.yml#L124) with no `-race`, and no workflow in `.github/workflows/` passes it anywhere. `make test` is worse than merely missing the flag: [`tm2/Makefile:16-17`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/Makefile#L16-L17) · [↗](../../../../../.worktrees/gno-review-6014/tm2/Makefile#L16-L17) exports `CGO_ENABLED=0`, and `go test -race` refuses outright under it with `-race requires cgo; enable cgo by setting CGO_ENABLED=1`, so a contributor following the comment has to override the module's own default as well as add the flag. Without `-race` the test's single assertion is [`require.Greater(t, queries.Load(), int64(1))`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp_test.go#L1587) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp_test.go#L1587), which the loop satisfies by construction; measured 620 to 1,214 iterations per run. Fix: make the guard something an automated run executes, or say in the comment that it is a manual check.
  </details>

- **[a store query still races a commit, so the comment reads as false reassurance]** `tm2/pkg/sdk/baseapp_test.go:1542-1543` — the comment states that a latest-height store query can run concurrently with `Commit`, but `.store` queries walk the live tree, and adding one write per block makes `-race` report the collision on this very commit.
  <details><summary>details</summary>

  `handleQueryStore` sends the query to the live multistore at [`baseapp.go:506`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L506) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L506), which routes to the mounted store at [`store.go:462`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L462) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L462) and lands in [`tree.GetVersioned`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/iavl/store.go#L316) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/iavl/store.go#L316). The tree it walks is the same `MutableTree` the commit writes through [`Store`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/iavl/store.go#L205-L207) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/iavl/store.go#L205-L207), and that type documents itself as [not safe for concurrent use](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/iavl/mutable_tree.go#L25-L26) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/iavl/mutable_tree.go#L25-L26). The version-pinned read is not a shield: `GetImmutable` at [`mutable_tree.go:682`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/iavl/mutable_tree.go#L682) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/iavl/mutable_tree.go#L682) hands back a tree over shared node objects, and `clone` clears the cached child pointers on the node it copies at [`node.go:314-315`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/iavl/node.go#L314-L315) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/iavl/node.go#L314-L315) while the reader reads one of them at [`node.go:677-678`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/iavl/node.go#L677-L678) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/iavl/node.go#L677-L678).

  The new test misses this only because it never writes: its query targets the literal key `"key"` at [`baseapp_test.go:1554-1557`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp_test.go#L1554-L1557) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp_test.go#L1554-L1557) and its 19 blocks carry no transaction, so `main` stays empty and the walk never reaches a node the commit rewrites. [`store_query_write_race_test.go`](tests/store_query_write_race_test.go) is the same test plus one `DeliverTx` per block; it reports the collision on 5af2a62c1 and would run clean if `.store` queries were served from the committed snapshot. The collision predates this PR and reproduces identically at the merge-base b4d044acc, so it is not a regression; it is in scope because this test's comment is now the codebase's statement about the property. Fix: narrow the comment to the commit-ID publication it actually guards, or make the test write to the store so it covers what it says.
  </details>

## Nits

- **[the only field in the struct whose concurrency contract is unstated]** `tm2/pkg/store/rootmulti/store.go:54` — the field changes to an atomic pointer with no comment, so the next reader sees a heavier type and no reason for it.
  <details><summary>details</summary>

  Its two neighbours both carry theirs: [`querySnapshot:67-69`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L67-L69) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L67-L69) names its writer and its readers in two lines, and [`snapshotMu:79-97`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L79-L97) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L79-L97) draws the interleaving it exists to prevent. Since nothing automated runs `-race`, that comment is the only thing standing between a future cleanup and a silent revert to a plain struct. Fix: name the writers and the concurrent reader on the field.
  </details>

## Missing Tests

None.

## Suggestions

- **[the published hash is one shared slice, not a copy]** `tm2/pkg/store/rootmulti/store.go:236-241` — `LastCommitID` returns a struct copy whose `Hash` still points at the single slice every reader shares, and nothing records that it must not be written to after publication.
  <details><summary>details</summary>

  Both publishers happen to build a fresh slice, [`commitInfo.Hash()`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go#L577-L580) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go#L577-L580) via `merkle.SimpleHashFromMap`, and no current caller mutates what it gets back, so this is latent only. The struct-copy return makes the sharing easy to miss, which is exactly the shape that survives a review: a caller that trims or reuses the returned `Hash` in place would corrupt what every concurrent reader sees, and no `-race` run would catch a write that is properly ordered against itself.
  </details>

## Verified

- Reverting only the store change reproduces the exact race the PR describes: with [`store.go`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store.go) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store.go) restored to the merge-base b4d044acc, `CGO_ENABLED=1 go test -race -run TestStoreQueryConcurrentWithCommit ./tm2/pkg/sdk/` reports a write in `multiStore.Commit` against a read in `multiStore.LastCommitID` reached through `BaseApp.LastBlockHeight` and `handleQueryStore`, and the same command at 5af2a62c1 is clean. One race is reported, matching the one the PR claims to fix and no other.
- The `.store` query path still collides with the commit path on the PR head. With one `DeliverTx` added per block, `-race` reports `iavl.Node.clone` in `MultiWrite` against `iavl.Node.getRightNode` in `tree.GetVersioned`, at least once in every `-count=10` batch run and once in a `-count=3` batch. The same collision appears at the merge-base under `GORACE=halt_on_error=0`, alongside the commit-ID race, so it predates the diff.
- `go test -race` cannot run at all under the tm2 module's own default: `CGO_ENABLED=0 go test -race ./tm2/pkg/sdk/` prints `-race requires cgo; enable cgo by setting CGO_ENABLED=1` and runs nothing.
- Line-for-line duplication with [PR #6018](https://github.com/gnolang/gno/pull/6018) confirmed from its own diff at head 5ceafd2c5: it introduces the same `atomic.Pointer[types.CommitID]` field, the same nil-guarded `LastCommitID`, the same `LoadVersion` and `Commit` publication sites, and the same [`store_test.go:471`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/store/rootmulti/store_test.go#L471) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/store/rootmulti/store_test.go#L471) field-to-accessor edit, wrapped in a `setLastCommitID` helper. Its two `.go` hunks overlap this PR's, so whichever lands second conflicts.
- `go test ./tm2/pkg/sdk/ ./tm2/pkg/store/rootmulti/` green at 5af2a62c1, plain and under `-race`; `gofmt -l` and `go vet` clean on all three changed files.

## Open questions

- `.store` queries read the live tree while custom queries read the committed snapshot, an asymmetry [PR #6018](https://github.com/gnolang/gno/pull/6018) closes by routing `.store` through `QueryImmutable`. Not posted separately: the decision belongs to that PR, and the part that touches this diff is already the second Warning.
- Under aggressive pruning, `handleQueryStore` can inject a height at [`baseapp.go:498`](https://github.com/gnolang/gno/blob/5af2a62c1/tm2/pkg/sdk/baseapp.go#L498) · [↗](../../../../../.worktrees/gno-review-6014/tm2/pkg/sdk/baseapp.go#L498) that a concurrent commit prunes before the substore is reached, yielding an empty result rather than an error. Pre-existing and untouched here; `TestSimulateConcurrentWithCommit` already documents the same window as acceptable behaviour for its own path.
- The new test omits `t.Parallel()` while both neighbouring concurrency tests use it. Defensible for a busy-spin test that would otherwise compete with the rest of the package for CPU, so not raised.
- This PR predates [PR #6018](https://github.com/gnolang/gno/pull/6018) by a day, so the duplicated hunks arrived in the larger PR second. Not posted: merge order is a maintainer decision and the Body already asks for it.
