# Review: [#6514](https://github.com/gnolang/gno/pull/6514)
Posted: https://github.com/gnolang/gno/pull/6514#pullrequestreview-6080528244
Event: APPROVE
Verdict: APPROVE, the default moves to `info` with a test that fails on `debug`; the one open point is an optional README wording Nit.
Model: claude-opus-5-5, standard review
Commit: 1a97619ca (latest)
Overview: [overview](../overview.md)
Open the code: `git -C gno worktree add ../.worktrees/gno-review-6514 1a97619ca`
Round: 1. One finder and one judge, 13 candidates: 9 settled by the finder's own read, 4 from the judge's completeness pass, 1 kept.

## Body

## gno.land/cmd/gnoland/README.md:64-66 [gh](https://github.com/gnolang/gno/blob/1a97619ca57fc8cd9a76b195f6da1e9d728e51b4/gno.land/cmd/gnoland/README.md#L64-L66) · [↗](../../../../../.worktrees/gno-review-6514/gno.land/cmd/gnoland/README.md#L64) · Nit [posted](https://github.com/gnolang/gno/pull/6514#discussion_r4713407322)
Nit: the callout names no release for "that change", so an operator reading it cannot tell whether the binary they run still defaults to `debug`.

**Fix:** point at the binary's own help, for instance "pass `-log-level info` explicitly if `gnoland start -h` shows `default \"debug\"`".
