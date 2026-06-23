# Review: PR #4555
Event: COMMENT

## Body
This can be closed. The stacktrace change is already on master: `tm2/pkg/errors/errors.go` and `errors_test.go` are byte-identical to this branch, landed via #4553.

Beyond the merged change, the branch only adds three unrelated files absent from master: `misc/jaekwon/tictac.md`, `misc/jaekwon/gnoland-whitepaper.md`, and a 984 KB binary `misc/jaekwon/Gyroscopic effect reduces lift force -- more efficient rockets! Also, TicTac UFO! _ r_UFOscience.pdf`. Its HEAD trails master by 405 commits, so it cannot be rebased to a clean diff.

Separately, three properties of the merged shortening are worth a follow-up against master, not this PR: the project-prefix match has no path-separator boundary, so a sibling dir like `gno-simple-e2e` next to build root `gno` renders `gno//<full-abs-path>` and leaks the path; the `gno/` rewrite depends on `$GOMOD`, which a deployed binary lacks, so project paths are not shortened on a normal node; and the new test resets `buildDir`/`buildDirOnce`, race-free only because it omits `t.Parallel()`.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4555-shorter-stacktraces/1-beada2b29/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

Repros run at beada2b29.
