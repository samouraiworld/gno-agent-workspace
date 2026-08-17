# PR [#6068](https://github.com/gnolang/gno/pull/6068): fix(gov/dao): allowlist lockdown, proposal-page escaping, and executor disclosure

URL: https://github.com/gnolang/gno/pull/6068
Author: jaekwon | Base: master | Files: 26 | +1911 -48
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 304f09a7a (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6068 304f09a7a`

## Overview

`r/gov/dao` decides who may replace the governance implementation, take the mutable member
store and move treasury funds by matching the caller's realm path against a list called
`allowedDAOs`. An empty list matches everyone, which is the bootstrap state a genesis script
needs before any DAO exists to authorise it. Nothing made that state one-way, so a proposal
carrying an empty list put the chain back into it. The branch ignores an empty list on write
and rejects blank or space-padded entries, making the move from open to locked permanent.

Three smaller things travel with it. Text that a proposal's executor controls, the denial
reason and the realm the executor says it came from, was written into the proposal page as raw
markdown and with no length bound, so it could forge page structure or price the page out of
the query gas cap. Both are now cut to a fixed size and then escaped, in that order. The line
naming the executor's origin realm used to print only when the executor also carried a
description, which the branch counts 16 production call sites leaving empty, so voters saw
nothing; it now prints on its own. And two lookups that assumed a proposal always has a voting
status now say so instead of dereferencing nil.

**Verdict: REQUEST CHANGES** — the lockdown and the clamps do what the description says, and
the new disclosure line is the one thing on the page an executor can switch off, leaving its
own raw text as the only provenance a voter sees (1 warning, 2 nits, 1 suggestion).

## Verify first

- [`examples/gno.land/r/gov/dao/v3/impl/render.gno:140`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140) — the `cr != ""` guard decides whether the page carries a provenance line at all. Render a proposal whose executor returns `""` from `CreationRealm()` and count `Executor created in:` on the page.
- [`examples/gno.land/r/gov/dao/proxy.gno:164`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/proxy.gno#L164) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/proxy.gno#L164) — `len(r.AllowedDAOs) != 0` is now the only thing standing between a passed proposal and the fail-open state. Confirm no other assignment to `allowedDAOs` exists: `grep -rn 'allowedDAOs =' examples/gno.land/r/gov/dao/`.

## Summary

The allowlist guard is the load-bearing change. Every entry is validated before anything is
assigned, so a rejected request leaves the previous list untouched, and `UpdateImpl` is the
only writer of `allowedDAOs`. The denial reason and the creation realm come out escaped
under every payload the branch's filetest sends and under the ones added here. What the
branch does not close is the gap its own third commit opens: [`render.gno:140`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/render.gno#L140)
prints the provenance line only when the escaped value is non-empty, and the same executor
that chooses that value also writes [`p.ExecutorString()`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/render.gno#L123) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/render.gno#L123)
into the page unescaped. Returning `""` from one and the disclosure line from the other leaves
the page with a single `Executor created in:` line that the executor wrote.

## Benchmarks / Numbers

One proposal-page render, gas as reported by `gno test -v`, from the repro blocks in
[`comment_claude-opus-5.md`](comment_claude-opus-5.md) and the filetests in [`tests/`](tests/). `maxGasQuery` is 3,000,000,000 at [`gno.land/pkg/sdk/vm/keeper.go:52`](https://github.com/gnolang/gno/blob/304f09a7a/gno.land/pkg/sdk/vm/keeper.go#L52) · [↗](../../../../../.worktrees/gno-review-6068/gno.land/pkg/sdk/vm/keeper.go#L52).

| Executor `String()` body | Gas | Against the cap |
|---|---|---|
| returns immediately | 11,995,967 | 0.4% |
| loops 2,000,000 times, returns 4 bytes | 3,895,973,145 | over |
| same, if the value were computed once | 1,953,984,556 | 65% |

| Proposal description | Gas |
|---|---|
| 40 bytes | 10,931,522 |
| 1,000,000 bytes | 25,115,675 |

## Warnings (should fix)

- **[disclosure the subject can switch off]** `examples/gno.land/r/gov/dao/v3/impl/render.gno:140-141` — an executor returning `""` from `CreationRealm()` suppresses the realm's own `Executor created in:` line, so its unescaped `String()` output supplies the only one, naming any realm it likes.
  <details><summary>details</summary>

  The commit adds the line so that a voter always learns which realm's code runs. The guard
  reads the value the executor supplies, so the executor decides whether the line appears.
  `p.ExecutorString()` is emitted unescaped one block above, which the ADR records as a
  deliberate cost decision, so the executor can write a well-formed `Executor created in:`
  line naming any realm it likes. Combining the two leaves exactly one such line on the page
  and the realm did not write it. Counted on the page rather than inferred: 1 line here, 2 at
  the merge base 33763e826, where the realm's own line printed whatever the executor returned
  and its emptiness was the tell, so the same filetest passes there and fails here. [`types.gno:237`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/types.gno#L237) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/types.gno#L237)
  carries the same expression, so a fix lands twice.

  The ADR notes that the new filetest does not exercise this path;
  [`tests/executor_disclosure_spoof_filetest.gno`](tests/executor_disclosure_spoof_filetest.gno)
  is that case, asserting the fixed count of 2 and failing with 1 at this head.
  Fix: emit the label whatever the executor returns, so the page always carries a line the
  realm wrote.
  </details>

## Nits

- **[comment states a mechanism the code does not have]** `examples/gno.land/r/gov/dao/proxy.gno:189-194` — this panic crosses a realm boundary and aborts the transaction, so it never reaches `DeniedReason`, which only the error return of `ExecuteProposal` writes.
  <details><summary>details</summary>

  `status.DeniedReason` is written only on the error return of
  [`ExecuteProposal`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L173) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/govdao.gno#L173).
  A panic inside the executor crosses a realm boundary, which the interrealm spec says
  [aborts the program](https://github.com/gnolang/gno/blob/304f09a7a/docs/resources/gno-interrealm.md?plain=1#L827-L831) · [↗](../../../../../.worktrees/gno-review-6068/docs/resources/gno-interrealm.md#L827),
  so the transaction reverts and nothing is stored. Run at this head, an executor panicking
  with `boom-from-executor` surfaces as an abort through
  `dao.ExecuteProposal` and leaves the proposal open. The storage worry the comment raises is
  also already answered by the same commit, which bounds every stored reason to
  `maxRenderedReason`. Fix: name the entry in the message, or drop the sentence.
  </details>

- **[ADR names a type this branch deleted]** `gno.land/adr/pr6068_govdao_allowlist_and_disclosure.md:15` — the trust-boundary table still lists `SafeExecutor.Execute` at `r/gov/dao/types.gno:221`, which this branch deletes, and lines 184 and 591 reason from it as live code.
  <details><summary>details</summary>

  The last [commit](https://github.com/gnolang/gno/commit/304f09a7a18e3ee8cbc2b2f40830015f15708b9f) drops `SafeExecutor` and `NewSafeExecutor` from `r/gov/dao/types.gno`.
  Three ADR passages survive it: the table row above, which makes the count of privileged
  operations five where four remain, line 184 on why `""` grants nothing today, and line 591
  pairing it with `UpdateImpl` on the `IsCurrent()` question. A reader checking the boundary
  list against the tree finds the first entry missing and has no way to tell which of the
  three is stale.
  </details>

## Suggestions

- **[attacker's work paid for twice]** `examples/gno.land/r/gov/dao/v3/impl/render.gno:123` — `p.ExecutorString()` is evaluated twice per page render, so a third-party executor's method body runs twice on the path the change hardens.
  <details><summary>details</summary>

  This predates the branch, and the branch rewrites the format call, so the value fits in a
  local in the same edit. It matters here because the file the branch adds exists to keep
  attacker-controlled work on the render path inside the query cap, and this is the part of
  that work no clamp reaches: `clamp.gno` says so itself. Measured on the real render path, an
  executor whose `String()` loops two million times and returns four bytes costs 3,895,973,145
  gas against 11,995,967 for one that returns immediately. Half of the difference is the
  second dispatch, so the same payload would cost 1,953,984,556 with the value computed once,
  which is inside the cap rather than past it.
  [`types.gno:227-232`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/types.gno#L227-L232) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/types.gno#L227)
  has the same pair.
  </details>

## Verified

- The allowlist guard rejects the whole request before any assignment, so a bad entry cannot
  half-apply. Read at [`proxy.gno:164-210`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/proxy.gno#L164-L210) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/proxy.gno#L164) and confirmed by the branch's own
  `TestUpdateImplRejectsPaddedAllowedDAOEntry`, which asserts the previous list survives.
- The proposal description stays unclamped and unescaped, which the change leaves alone, and
  the render cost does not scale into the cap: 1,000,000 bytes render in 25,115,675 gas
  against 10,931,522 for 40 bytes, about 14 gas per byte, so the render costs under 1% of
  the cap. Artifacts in [`tests/description_1mb_filetest.gno`](tests/description_1mb_filetest.gno)
  and [`tests/description_40b_filetest.gno`](tests/description_40b_filetest.gno).
- [`tests/executor_disclosure_spoof_filetest.gno`](tests/executor_disclosure_spoof_filetest.gno)
  passes unchanged at the merge base 33763e826 and fails at this head, so the count moved from
  2 to 1 on this branch.
- `Executor.String()` dispatch count per proposal-page render is 2, counted in
  [`tests/executor_string_double_dispatch_filetest.gno`](tests/executor_string_double_dispatch_filetest.gno).
- An executor panic aborts the transaction rather than being recorded, run at this head
  through `dao.ExecuteProposal`.

## Open questions

- `StringifyProposal` has no production caller, by the branch's own account in
  [`stringify_proposal_00_filetest.gno`](https://github.com/gnolang/gno/blob/304f09a7a/examples/gno.land/r/gov/dao/v3/impl/filetests/stringify_proposal_00_filetest.gno#L18-L23) · [↗](../../../../../.worktrees/gno-review-6068/examples/gno.land/r/gov/dao/v3/impl/filetests/stringify_proposal_00_filetest.gno#L18),
  yet it is exported and now carries a second copy of the disclosure expression that a fix has
  to reach. Not posted because folding the two is a design call the warning already forces.
- The `IsCurrent()` guards added to `treasury.Send` and `treasury.SetTokenKeys` are dead under
  the crossing-function rule, which the commit message says outright, and `UpdateImpl` does not
  get one for the reason the ADR gives. Not posted: the ADR settles it and the reasoning is
  consistent.
