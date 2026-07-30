# Batch status — review all (started 2026-07-29)

Model claude-opus-5, reviewer davd-gzl. Normal (non-deep) mode.

Synced head `0d08897` (`samouraiworld/main`) before building the set. The same query run against a
checkout 457 commits behind returned 37 PRs, 28 of them already reviewed here.

gno master at dispatch: `d1a33f57`.

## External-contribution safety gate

Not applicable. All four PRs come from `MEMBER` accounts (jaekwon, notJoon, Villaquiranm); no
`FIRST_TIME_CONTRIBUTOR` in the set.

## Dropped

| Reason | PRs |
|---|---|
| already reviewed (present in `reviews/pr/`) | 121 of the 139 open non-draft PRs |
| dependabot | 6008, 6005, 5992, 5990, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by reviewer (davd-gzl) | 6006, 5993, 5950, 5936, 5934 |

## Final set (4)

All four are first rounds. No head-unchanged, already-APPROVED, or patch-id gate applied.

| PR | Head sha | Author | Size | Worktree | Review dir | Mode |
|---|---|---|---|---|---|---|
| [6018](https://github.com/gnolang/gno/pull/6018) | `5ceafd2c5` | jaekwon | +2200-155, 26f | `.worktrees/gno-review-6018` | `reviews/pr/6xxx/6018-*/1-5ceafd2c5/` | normal |
| [6014](https://github.com/gnolang/gno/pull/6014) | `5af2a62c1` | notJoon | +63-10, 3f | `.worktrees/gno-review-6014` | `reviews/pr/6xxx/6014-*/1-5af2a62c1/` | normal |
| [6012](https://github.com/gnolang/gno/pull/6012) | `bb9f82f69` | jaekwon | +9806-7000, 100f | `.worktrees/gno-review-6012` | `reviews/pr/6xxx/6012-*/1-bb9f82f69/` | normal |
| [5991](https://github.com/gnolang/gno/pull/5991) | `65ae435d4` | Villaquiranm | +52-5, 2f | `.worktrees/gno-review-5991` | `reviews/pr/5xxx/5991-*/1-65ae435d4/` | normal |

6018 and 6014 both touch tm2 store query/commit isolation and are likely to overlap; 6014 is the
narrow fix, 6018 the broad one. Read the pair together when synthesizing.

## Dispatch

One `general-purpose` agent per PR, all in one message. The parent created every worktree and
checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch
switch. Subagents write `review_claude-opus-5_davd-gzl.md` and `comment_claude-opus-5.md`, and do
not commit, push, regenerate indexes, or post.

## Progress

Dispatched; awaiting agent returns.

| PR | Verdict | Findings |
|---|---|---|
| 6018 | NEEDS DISCUSSION | 2 Warnings, 1 Missing test, 4 Nits, 2 Suggestions |
| 6014 | NEEDS DISCUSSION | 2 Warnings, 1 Nit, 1 Suggestion |

6018 notes: no defect in the four layers; each was revert-proofed against a named guard in the
suite. CI green, and the five new concurrency tests are clean under `-race`, which CI does not run.
Two author decisions: the boot refusal prints its only recovery instruction as the Go constant name
`PrefixMeta"fastidx"`, never its value, and omits the `s/_/` store prefix; and `.store` queries now
rebuild an immutable multistore per request, a full scan of the retained-root keyspace, measured
7.4 µs → 0.50 ms at 1,000 retained versions and 19 µs → 57.6 ms at 60,000 against gno.land's default
retention of 705,600, on a query connection serialized by one mutex. Package test time for
`tm2/pkg/store/rootmulti` goes 0.17 s → 305.6 s, 203 s of it in the 30-seed fuzz.

Cross-PR, 6018 vs 6014: commit `09853f109` makes the identical `atomic.Pointer[types.CommitID]`
change, conflicting textually in `rootmulti/store.go` and on one `store_test.go` line, so whichever
merges second needs a resolution. 6014's only piece not carried over is its
`TestStoreQueryConcurrentWithCommit` guard in `baseapp_test.go`, and 6018's own full-app test covers
that race under `-race` (revert-proofed), so it is redundant rather than a gap. Merge order is a
maintainer decision; both drafts raise it without asserting one.

Skill fix during the run: `skills/writing-style.md` forbade any severity word at the first position
of an inline comment while `skills/review.md:429` requires the `Critical:` / `Nit:` / `Suggestion:` /
`Missing test:` prefix (Warning takes none). Resolved in favour of `review.md`.
| 6012 | REQUEST CHANGES | 2 Critical, 5 Warnings, 4 Missing tests, 6 Nits, 4 Suggestions |
| 5991 | APPROVE | 1 Nit, 1 Missing test, 1 Suggestion |

6012 notes: treasury plumbing holds up — `NewBanker(RealmSend, sub)` bounds an executor to one
address, the derived treasury address agrees with the minted sub-identity, the raw-text render marker
cannot be smuggled, and the tally rule is property-tested. All suites green at `bb9f82f69` (24 + 13
test functions, 111 filetests, 5 txtar). Two Criticals, each shipped as a red filetest: `New` checks
the DAO-creation invite against `unsafe.OriginCaller()` but seats `cur.Previous().Address()`, so any
realm an invited user calls burns the invite and takes the only council seat permanently (the
`origin-caller-auth` audit pattern; `Invite` two functions above does it correctly); and the
dissolution sweep sends to `parent.Address()` without checking whether that parent is dissolved,
stranding 500ugnot in a dissolved DAO in the state `render.gno:86` itself labels unrecoverable. Lead
Warning: the abstain-shrunk tally denominator lets cast order decide the outcome, so two DAOs with
the same council and the same final intentions come out `passed` and `dismissed`. The red `gno2go`
check is this PR's own bug, `init(cur realm)` with `cur` unused, the only such case among 360
occurrences in `examples/`.

6012 head advanced `bb9f82f69` → `497724fe1` mid-review. Verified delta: one deleted `.gitignore`
line (`config/`), no code, no anchor affected. Kept as round 1, stale +1, rather than cutting a new
round. `overview.html` written (tally calculator mirror checked against all 14 rows of the branch's
own `TestTallyDefault`). The PR description documents `Options`, `UpdateOptions` and
`AllowExecution`, none of which exist in the code; raised as an unanchored Body item.

5991 notes: the memoized TypeID is a pure function of `PkgPath`, `ParentLoc` and `Name`, all written
once together in `types.go:1485-1491`; the other producers build fresh values with a zero memo, so
the deleted comparison could not fire. Confirmed by reinstating the recompute as a counter: over 24
million cached hits, zero mismatches. Gas unaffected. Win measured against merge base `d1a33f574`:
3 allocations and 64 bytes per cached call package-level, 13 and 277 function-level, both to zero.
APPROVE needs human confirmation before it can be posted with `--approve`.

6014 notes: fix proven by reverting `rootmulti/store.go` to merge-base `b4d044acc` (race reproduces,
clean on `5af2a62c1`). Duplicated line-for-line by 6018, which came a day later, so landing 6014
first shrinks 6018; merge order posed as a Body question rather than CLOSE. Its regression guard
cannot fire in CI: no workflow passes `-race` and `tm2/Makefile:16-17` sets `CGO_ENABLED=0`. A
separate pre-existing `Node.clone`/`Node.getRightNode` race on `.store` queries is scoped to the
test's misleading comment, since 6018 fixes it by routing through `QueryImmutable`.

Environment note: no Go toolchain was present in the review environment; go1.25.9 was installed
outside the tree to run the suites.

## Finalize

1. Parent commits once: `review: batch of 4 open PRs (6018, 6014, 6012, 5991)`.
2. Push to `samouraiworld`. The `davd-gzl` fork's `main` still points at the pre-sync `abaef76`
   and needs a manual force-push to match.
3. Nothing reaches GitHub without the literal `post`.

## Carried from the 2026-07-24 batch

- [6002](https://github.com/gnolang/gno/pull/6002) draft verdict is APPROVE and still needs human
  confirmation before posting with `--approve`.
- [5981](https://github.com/gnolang/gno/pull/5981) draft is REQUEST_CHANGES, posted since as part
  of `0d08897`.
