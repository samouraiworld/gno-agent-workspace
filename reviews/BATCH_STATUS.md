# Deep-review batch — status (started 2026-07-16)

Scope: Jae's six most recent merges. User asked for the latest merge and confirmed "All 6 recent Jae merges" when the scope was ambiguous. Model claude-opus-4-8, reviewer davd-gzl. Deep mode on every PR (parallel lens agents, one critic round, claim-verification gate). Nothing posted.

All six are already merged. Each is reviewed at its PR head on its own merits; the merged status is stated in each round note and does not soften any verdict.

## Final set

| PR | Size | Head sha | Merged as | Round | Worktree | Review dir |
|----|------|----------|-----------|-------|----------|------------|
| [5890](https://github.com/gnolang/gno/pull/5890) | +2662/-232, 50f | `b940037d1` | `5b989cad5` | 2 (round 1 at `8a115c8ca`) | `.worktrees/gno-review-5890` | `reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/` |
| [5891](https://github.com/gnolang/gno/pull/5891) | +509/-24, 10f | `82e5cb868` | `af23ea2ae` | 2 (round 1 at `057894796`) | `.worktrees/gno-review-5891` | `reviews/pr/5xxx/5891-split-mempackage-prod-test/2-82e5cb868/` |
| [5892](https://github.com/gnolang/gno/pull/5892) | +242/-60, 32f | `03ab3eea2` | `412ab1962` | 2 (round 1 at `d2f3d1337`) | `.worktrees/gno-review-5892` | `reviews/pr/5xxx/5892-meter-preprocess-gas/2-03ab3eea2/` |
| [5893](https://github.com/gnolang/gno/pull/5893) | +117/-65, 9f | `7fc5ec06a` | `9bfc0a4bb` | 2 (round 1 at `131c5fccb`, APPROVE) | `.worktrees/gno-review-5893` | `reviews/pr/5xxx/5893-deterministic-typecheck-verdict/2-7fc5ec06a/` |
| [5937](https://github.com/gnolang/gno/pull/5937) | +1490/-295, 49f | `b79972d22` | `dc305b6d6` | 1 (new) | `.worktrees/gno-review-5937` | `reviews/pr/5xxx/5937-bptree-clean-tree-fast-index/1-b79972d22/` |
| [5938](https://github.com/gnolang/gno/pull/5938) | +426/-100, 20f | `27c5ece7e` | `1e2e00e2f` | 1 (new) | `.worktrees/gno-review-5938` | `reviews/pr/5xxx/5938-mount-bptree-fast-index/1-27c5ece7e/` |

## Dropped

None. The user named all six, so the head-unchanged, already-APPROVED, and patch-id-equal base-only drops were not applied. The patch-id gate still runs on 5890, 5891, 5892, and 5893, but only to characterize head movement in each round note; no round is reanchored.

## Head movement

5890, 5891, 5892, and 5893 all advanced past their round-1 shas, so each gets a full round 2 rather than a reanchor.

`7fc5ec06a` (5893) is a merge of master. `git show 7fc5ec06a --cc` prints zero hunks, so the merge authored no conflict-resolution content. Master now carries 5891 (`af23ea2ae`) and 5892 (`412ab1962`), so 5893's diff against master is finally its own nine files, and round 1's scope note about the stacked trio is obsolete.

## Dispatch

One `general-purpose` coordinator per PR, all in one message. Each runs deep mode and dispatches its own lens agents. The parent created every worktree and checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch switch. Subagents write `review_claude-opus-4-8_davd-gzl.md` and `comment_claude-opus-4-8.md`, and do not commit, push, regenerate indexes, or post.

## Results

All six returned. Five REQUEST CHANGES, one APPROVE. Two rounds overturn a round-1 APPROVE: 5891 and 5893.

| PR | Verdict | Headline |
|----|---------|----------|
| 5890 | REQUEST CHANGES | the `NewBanker` sub-token gate calls interpreted `chain.SplitPkgSubPath` instead of the native accessor the PR adds, so every `OriginSend` pays for a string split |
| 5891 | REQUEST CHANGES (overturns round 1) | `GetMemPackageAll` hands a raw path to `MPAnyAll.Decide`, which panics on `#`; the `#allbutprod` sibling is addressable through `vm/qfile` and `vm/qdoc` by any unauthenticated client |
| 5892 | APPROVE | no consensus defect; charge is exact and deterministic. Two Warnings: eleven unreachable nil-guard lines at `machine.go:330-340`, and dependency source billed at `ReadCostPerByte` 17 against the PR's 1250 |
| 5893 | REQUEST CHANGES (overturns round 1) | a `//go:build go1.N` line in a submitted file overrides the pinned `GoVersion`, so the accept/reject verdict is still a function of the validator's build toolchain |
| 5937 | REQUEST CHANGES | the ABCI query-height open can rebuild and rewrite the live fast index, because the `ImmutableDB` wrapper never reaches a store mounted with an explicit db; plus the unbounded version rescan |
| 5938 | REQUEST CHANGES | mounting bptree puts a full scan of every retained version on the RPC path (100.9ms at 100K versions vs IAVL's flat 14.1µs); SET-read gas pinned 30% under its own cited measurement |

Cross-PR confirmations and conflicts:

- 5937 and 5938 independently found the same `discoverVersions` rescan from opposite ends: 5937 from tm2 internals (`nodedb.go:473`), 5938 from the mount that exposes it on mainnet RPC (`app.go:106`). One root cause, one fix.
- Both flagged the same missing fingerprint guard at `generate.go:676`.
- They conflict once: absent-key GET pricing is a Warning in 5937 (`params.go:40`) and a deliberate, not-posted Open question in 5938. Unresolved; settle before either becomes a fix.

Parent verification of the two heaviest findings, run directly rather than taken from agent summaries:

- 5893's Critical reproduces at `7fc5ec06a`. `//go:build go1.22` plus `for range 10` type-checks clean under the go1.18 pin, and `//go:build go1.99` fails with `file requires newer Go version go1.99 (application built with go1.26)`, naming the building toolchain. Two validators on go1.25 and go1.26 disagree on state, not just on the results hash.
- 5937's immutable-write Warning holds: `MultiImmutableCacheWrapWithVersion` wraps the db and sets `Immutable=true`, but `constructStore:378-382` prefers `params.db` when non-nil and gno.land mounts `mainKey` with an explicit `cfg.DB`, so the read-only wrapper is dead. `ensureFastIndex` checks only `FastIndex`, never `opts.Immutable`.
- 5892's dead-code Warning holds: `IterMemPackage` has exactly one implementation, and it already skips nil at `store.go:1263-1269`, so `machine.go:330`'s guard cannot fire.

Two agents self-corrected during their own citation audits, which is worth recording: 5892 withdrew a 20.2 gas/byte figure that was a pre-5891 baseline and re-grounded the finding on `ReadCostPerByte`; 5891 fixed four bad anchors and retracted a round-1 Suggestion whose premise came from trusting a doc comment its own review proves false.

## State at pause (2026-07-17)

The review half is finished. All six PRs have `review_*.md` and `comment_*.md`; all six worktrees are clean; `build-indexes.sh` has been run.

Blocked on one thing: commit `31b4c448e` ("review: PRs 5760, 5890-5893, 5937, 5938; 5082 GC claim retracted") was authored outside this batch and already swept these six review dirs into itself, together with a 5760 round-3 review and a 5082 retraction this batch never produced or verified. It is local only; `main` is ahead of `origin/main` by 1. The user is cutting that commit themselves. Nothing here was pushed, and nothing was posted to GitHub.

Left staged on top of it, pending that cut: `index.html`, `reviews/BATCH_STATUS.md`, `reviews/README.md`, the 5082 round-2 file, 5891's and 5892's final review and comment (rewritten by resumed agents after `31b4c448e` captured an earlier state), and the 5938 review's ADR correction.

Note for whoever re-commits: `31b4c448e`'s message says 5892 prices dependency bytes "~62x under". The final review says ~70x against `ReadCostPerByte` 17, after its citation audit withdrew an earlier figure as a stale pre-5891 baseline. Do not carry the 62x forward.

## Next: fix PRs

The goal is a fix PR per finding, each in its own worktree, reviewed until clean. Nothing is branched or written yet. Per `skills/fix-issue.md`: worktree at `.worktrees/gno-fix-<slug>` off `origin/master`, fork remote `fork` = `davd-gzl/gno`, never push to `origin`.

Proposed grouping, not yet agreed with the user:

| Fix | Source | Shape |
|---|---|---|
| Clear `ast.File.GoVersion` in `GoParseMemPackage` | 5893 Critical | Consensus fork; highest value, self-contained. Parent-verified red at `7fc5ec06a`. Tests already written in 5893's `tests/`. |
| Immutable open: stop scanning, stop writing | 5937 + 5938 | One PR, two commits. `discoverVersions` seek first/last instead of full scan; `ensureFastIndex` early-return on `opts.Immutable` plus `constructStore` wrapping `params.db` when Immutable. Both parent-verified. |
| Delete unreachable nil guard `machine.go:330-340` | 5892 Warning | Trivial deletion; parent-verified unreachable. |
| `GetMemPackageAll` panics on `#` path | 5891 Warning | Check overlap first: open PR 5971 fixes it incidentally. |
| pb3 removal vs stored block results | 5893 Warning | Migration/compat; needs its own look. |
| 5890 banker + address fixes | 5890 | Four Warnings, likely one PR. |
| Depth-gas repins | 5938 + 5937 | Consensus genesis defaults, needs a fingerprint append. Wants Jae's buy-in; may be an issue, not a PR. |
| Fingerprint guard test | 5937 + 5938 | Test-only. |
| Doc/comment nits | all six | One batch PR. |

Unresolved before any gas fix: 5937 and 5938 disagree on absent-key GET pricing. 5937 calls it a Warning at `params.go:40`; 5938 deliberately left it an unposted Open question. Settle first.

## Resume

1. Confirm the user has cut `31b4c448e`, then re-commit the review artifacts.
2. Agree the fix grouping above with the user before writing code; seven-plus PRs is a lot of surface and wants cutting down.
3. Then per fix: worktree, implement, test, deep-review the fix itself until clean.

If a session dies mid-batch: check which review dirs hold both `review_*.md` and `comment_*.md`, re-dispatch only the incomplete ones. The review worktrees already exist at the shas in the table; do not re-create them.
