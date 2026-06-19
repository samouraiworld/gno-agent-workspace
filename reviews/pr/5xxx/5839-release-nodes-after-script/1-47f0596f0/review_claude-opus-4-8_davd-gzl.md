# PR #5839: test(gno.land): release in-memory nodes after each script to fix TestTestdata leak

URL: https://github.com/gnolang/gno/pull/5839
Author: thehowl | Base: master | Files: 1 | +6 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 47f0596f0 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5839 47f0596f0`

**TL;DR:** The integration test suite (`TestTestdata`) starts one throwaway blockchain node per test script. A bookkeeping map kept a reference to every node it ever started and never dropped them, so the memory of all already-finished nodes piled up until the run ran out of RAM. This adds one line to remove each node from the map when its script ends.

**Verdict: APPROVE** — one-line leak fix, mechanism and effect both reproduce; no concerns.

## Summary
`SetupGnolandTestscript` boots an in-memory node per [txtar](https://github.com/gnolang/gno/blob/47f0596f0/gno.land/pkg/integration/testdata) script and registers each in a process-lifetime `NodesManager` map ([`testscript_gnoland.go:341`](https://github.com/gnolang/gno/blob/47f0596f0/gno.land/pkg/integration/testscript_gnoland.go#L341) · [↗](../../../../../.worktrees/gno-review-5839/gno.land/pkg/integration/testscript_gnoland.go#L341)). The per-script teardown stopped the node but only removed it from the map on an explicit `gnoland stop`, which almost no script issues, so each finished node's in-memory store stayed reachable for the whole suite. With ~170 node-booting scripts that retention is the dominant heap cost: measured ~52 MB per retained node, climbing to ~9 GB live heap / ~12.3 GB `Sys` sequentially and OOM-killing high-core/low-RAM `-parallel` runs. The fix calls `nodesManager.Delete(sid)` in the teardown so the node and its store become collectable once the script ends.

## Glossary
- **txtar**: testscript-based integration test under `gno.land/pkg/integration/testdata/`.
- **stdlib**: gno standard libraries; each booted node loads its own copy into its store at genesis init.

## Fix
Before, the teardown `defer` ran `Get(sid)` then `Stop()` and returned, leaving the `*tNodeProcess` (which transitively pins the node's in-memory store) in the manager map for the rest of the run. After, `Delete(sid)` runs first ([`testscript_gnoland.go:196-200`](https://github.com/gnolang/gno/blob/47f0596f0/gno.land/pkg/integration/testscript_gnoland.go#L196-L200) · [↗](../../../../../.worktrees/gno-review-5839/gno.land/pkg/integration/testscript_gnoland.go#L196)), so once the closure returns nothing references the node and GC reclaims it. The load-bearing fact is that the map is the only suite-lifetime reference: drop it and the per-script store no longer accumulates. Placing `Delete` before `Stop` (the `gnoland stop` path does the reverse at [`testscript_gnoland.go:389-395`](https://github.com/gnolang/gno/blob/47f0596f0/gno.land/pkg/integration/testscript_gnoland.go#L389-L395) · [↗](../../../../../.worktrees/gno-review-5839/gno.land/pkg/integration/testscript_gnoland.go#L389)) is harmless: the local `n` keeps the node alive across the `Stop` call regardless.

## Benchmarks / Numbers
Full `TestTestdata` suite, `-parallel=1`, live heap sampled in the teardown every 15 scripts (`runtime.MemStats`); end-of-run figures after a forced 2×GC. Same box, fix reverted vs present.

| scripts torn down | retained nodes (no fix) | HeapInuse (no fix) | retained (fix) | HeapInuse (fix) |
|---|---|---|---|---|
| 15 | 15 | 1.27 GB | 0 | 0.75 GB |
| 60 | 60 | 4.69 GB | 0 | 0.59 GB |
| 90 | 90 | 6.04 GB | 0 | 0.78 GB |
| 150 | 150 | 9.01 GB | 0 | 1.06 GB |
| 165 | 165 | 9.15 GB | 0 | 0.93 GB |
| end (post-GC) | 169 | 6.07 GB HeapAlloc / 12.36 GB Sys | 0 | 0.40 GB HeapAlloc |

Retained-node count tracks teardown count 1:1 without the fix and stays 0 with it; incremental cost ~52 MB/node matches the comment's "~50 MB/script".

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
- **[leak can silently come back]** `gno.land/pkg/integration/testscript_gnoland.go:200` — no guard fails if the `Delete` is dropped again.
  <details><summary>details</summary>

  The suite stays green whether or not the node is released, so a future refactor can reintroduce the leak invisibly (CI only OOMs on high-core/low-RAM runners). A guard asserting the manager map drains to zero after a script would catch it, but it reads unexported state and the maintainer deliberately kept this minimal, so this is optional, not blocking. Confirmed behaviorally: with the line reverted, retained-node count climbs 1:1 with scripts (table above) while the suite still passes.
  </details>

## Suggestions
None.

## Open questions
- The retained store, not the global stdlib/typecheck cache, is what leaked: the typecheck cache is process-global and shared across in-memory nodes ([`testdata_test.go:29-32`](https://github.com/gnolang/gno/blob/47f0596f0/gno.land/pkg/integration/testdata_test.go#L29-L32) · [↗](../../../../../.worktrees/gno-review-5839/gno.land/pkg/integration/testdata_test.go#L29)), so the comment's "per-node stdlib cache" refers to each node's own store copy, which the measurement confirms at ~52 MB/node. Wording only, not posted.
