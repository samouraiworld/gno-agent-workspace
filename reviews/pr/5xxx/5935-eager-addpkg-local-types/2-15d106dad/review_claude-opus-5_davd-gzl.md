# PR [#5935](https://github.com/gnolang/gno/pull/5935): fix(gnovm): persist function-local declared types eagerly at addpkg (alt to save-time walk)

URL: https://github.com/gnolang/gno/pull/5935
Author: ltzmaxwell | Base: master | Files: 10 | +978 -4
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 15d106dad (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5935 15d106dad`

**TL;DR:** A type declared inside a function body was never written to the chain's type store, so storing a value of it saved a pointer to a record that does not exist and every read after a restart crashed. This PR writes all such types once, at upload time. Since round 1 the author added the addpkg-time test the earlier round asked for and turned three silent skips into panics; the migration gap and the compiled-out guard are unchanged.

**Verdict: NEEDS DISCUSSION** — the enumeration is complete and the restart tests reproduce, but the branch still drops the save-time walk without a migration, and the guard meant to catch a missed route is compiled out everywhere CI runs (1 Warning, 1 Missing test now closed).

## What moved since 1-4369fdca7

The head advanced by a master merge plus two commits of the PR's own.

| Commit | Effect |
|---|---|
| [`87d1f229d`](https://github.com/gnolang/gno/commit/87d1f229dd0d0723625a5e9a54aced8df953eb9c) | the `pv.Block`, `GetSource` and nil-fileset paths in [`saveFuncLocalTypes`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/pkg/gnolang/machine.go#L931-L941) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L931) panic and abort the tx instead of returning and skipping persistence |
| [`15d106dad`](https://github.com/gnolang/gno/commit/15d106dad97b5143eaef6e7b11976879d990e01c) | adds [`zrealm_localtype3.gno`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/tests/files/zrealm_localtype3.gno) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/tests/files/zrealm_localtype3.gno) and a `zlti` realm in the restart txtar; refreshes the ADR |

| Round 1 finding | State at 15d106dad |
|---|---|
| `machine.go:917` — no migration for packages already in the store | open, moved to the Body |
| `store.go:618` — the `debugAssert` guard never runs | open, re-anchored to `store.go:656` |
| `machine.go:870-874` — no addpkg-time var-initializer test | fixed by `zrealm_localtype3.gno` |

## The missing test the author landed

Round 1 proposed a filetest holding a local-typed value in a file-level var initializer, so that the ordering constraint stated in the comment above [`m.saveFuncLocalTypes(pv)`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/pkg/gnolang/machine.go#L884-L888) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L884) is pinned by something. `zrealm_localtype3.gno` covers both addpkg-time routes the proposal named: `X` through the var initializer, saved by `saveNewPackageValuesAndTypes`, and `Y` through `init`, saved by `resavePackageValues`. The commit message records that reordering the save after finalization fails it under `-tags debugAssert`, which is the property the proposal was written to hold. Closed, nothing carried.

## Warnings (should fix)

- **[the fix reaches only packages uploaded after it merges]** unanchored, in the Body — no path re-runs the enumeration for a stored package, and the ADR's own "fresh chain" escape is retired two sentences after it is offered.
  <details><summary>details</summary>

  [`m.saveFuncLocalTypes(pv)`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/pkg/gnolang/machine.go#L888) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L888) runs on the addpkg-save path and nowhere else, so a package already in the store never gets a `/t/` record and a value of its local types persisted after the upgrade still writes a ref nothing can resolve. [#5894](https://github.com/gnolang/gno/pull/5894)'s save-time walk healed those on the package's next save; this branch removes it. The ADR states the consequence in the author's own words and offers two ways out, [genesis or a one-shot migration](https://github.com/gnolang/gno/blob/15d106dad/gnovm/adr/pr5935_local_type_persist.md?plain=1#L98-L101) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/adr/pr5935_local_type_persist.md#L98), then removes the first: [#5737](https://github.com/gnolang/gno/pull/5737) is on master, so [chain state built on post-#5737 code before this fix can already carry dangling method-value refs](https://github.com/gnolang/gno/blob/15d106dad/gnovm/adr/pr5935_local_type_persist.md?plain=1#L102-L105) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/adr/pr5935_local_type_persist.md#L102). Unanchored because the action is a deployment decision or a new migration, and no line the diff changes carries either. Fix: ship the migration, or keep the save-time walk as a transitional backstop until one exists.
  </details>

- **[the only guard against a missed route is compiled out]** `gnovm/pkg/gnolang/store.go:656` — [`assertNoDanglingLocalTypeRef`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/pkg/gnolang/store.go#L656) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/store.go#L656) sits behind `debugAssert`, which no CI job sets.
  <details><summary>details</summary>

  The assert is what turns a missed enumeration route into a test failure rather than a corrupt realm discovered after a restart. `grep -rn debugAssert .github/ gnovm/Makefile Makefile` at this head returns one hit outside the sources, the [`test.debugAssert` target](https://github.com/gnolang/gno/blob/15d106dad/gnovm/Makefile#L116-L119) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/Makefile#L116), and no workflow invokes it. `87d1f229d`'s commit message records the author running the debugAssert filetests by hand, which is the same evidence that nothing runs them automatically. The four `zrealm_localtype*` filetests under the tag are seconds of CI. Fix: add them to a workflow so the guard is load-bearing.
  </details>

## Missing Tests

None. Round 1's finding is closed by `zrealm_localtype3.gno`, and the txtar now chains a `zlti` realm covering the initializer route across a node restart.

## Verified

- Re-read `git diff 4369fdca7..15d106dad --stat`: the PR's own files move in two commits, `87d1f229d` and `15d106dad`; everything else in the range is the master merge.
- `zrealm_localtype3.gno` at this head declares `S` inside `mk()` and `U` inside `init`, assigns both to `any`-typed package vars, and asserts `9 4`. Both are the routes the round-1 proposal named.
- `saveFuncLocalTypes` is still reached from one call site, [`machine.go:888`](https://github.com/gnolang/gno/blob/15d106dad/gnovm/pkg/gnolang/machine.go#L888) · [↗](../../../../../.worktrees/gno-review-5935/gnovm/pkg/gnolang/machine.go#L888), inside the addpkg save. Nothing in the branch calls it for a package read back from the store.
- Both anchors are added lines: `git diff origin/master...HEAD` carries `+	m.saveFuncLocalTypes(pv)` and `+		ds.assertNoDanglingLocalTypeRef(o2)`.
- The ADR's alternatives section now lists predefine-time collection ([#6084](https://github.com/gnolang/gno/pull/6084)) as a maintained side-by-side alternative, so the two approaches are being carried in parallel rather than one superseding the other.

## Open questions

- The three new panics in `saveFuncLocalTypes` abort and roll back the tx. They are reachable only if the package block is not a live `*Block` sourced from a `*PackageNode` at addpkg save, which the surrounding code establishes. Not posted: the commit message states the suites that were run against them, and nothing in the branch shows a path that violates them.
- Whether this or [#6084](https://github.com/gnolang/gno/pull/6084) should land is a maintainer call between a language fact and a bookkeeping invariant. Not posted: both are open and the ADR states the trade-off correctly.
