# Review: PR [#5835](https://github.com/gnolang/gno/pull/5835)
Event: COMMENT

## Body
The precision and documentation feedback from the previous review is addressed on 3700f767f. The two false positives are fixed with tests, the rules and report output are documented, and the fixtures now carry vulnerable/fixed comments.

The open item is the framing. The docs call the harness executable and enforceable, but it does not run in CI, its contract asserts a loose per-rule substring rather than the expected file and line, and no shipped command scans unfamiliar realm code. Either wire it in as a genuinely enforced, agent-runnable contract, or soften the framing to a manual audit aid. The inline comments from the previous review still stand.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/4-3700f767f/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP misc/audit-pattern-harness/internal/auditpattern/run.go:471-478 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L471)
Posted round 3 (https://github.com/gnolang/gno/pull/5835#discussion_r3488799157); still open on 3700f767f, not reposting.

On code that is not already gofmt-clean, the reported `file:line` and text come from the gofmt-reformatted buffer, not the on-disk file, so a hit can point at the wrong line.

## SKIP misc/audit-pattern-harness/internal/auditpattern/run.go:413-417 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L413)
Posted round 3 (https://github.com/gnolang/gno/pull/5835#discussion_r3488799161); still open, not reposting.

`render_map_iteration` matches `range `+name as a substring with no right word boundary, so a map named `scores` flags `range scoresList`, an unrelated slice.

## SKIP examples/gno.land/r/docs/security_patterns/security_patterns.gno:33-41 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L33)
Posted round 3 (https://github.com/gnolang/gno/pull/5835#discussion_r3488799158); still open, not reposting.

The patterns are implicit: `Render` prints only the admin, message, and path, and only `assertAdmin` has a comment. Explain each pattern and why it is safer.

## SKIP examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38)
Posted round 3 (https://github.com/gnolang/gno/pull/5835#discussion_r3488799159); still open, not reposting.

A backtick in `path` closes the manual code span, and `path` reaches `Render` as arbitrary bytes. `md.InlineCode(path)` is the safe primitive.

## SKIP misc/audit-pattern-harness/internal/auditpattern/run_test.go:246-303 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L246)
SKIP'd round 3; author-acknowledged follow-up (https://github.com/gnolang/gno/pull/5835#discussion_r3475424437). Keeping it skipped.

The contract ties each vulnerable hit to a loose substring like `*` or `.Previous()`, not to a file and line, so a rule rewritten to a coincidental marker-bearing line still passes while it stops detecting its vulnerability.

## SKIP AGENTS.md:98 [↗](../../../../../.worktrees/gno-review-5835/AGENTS.md#L98)
SKIP'd round 3. Keeping it skipped.

No shipped command runs a rule against arbitrary realm code, though this line asks agents to. The only command runs the bundled fixtures, and the directory-scanning function `RunRule` is internal.

## SKIP .github/workflows/ci-dir-misc.yml:24 [↗](../../../../../.worktrees/gno-review-5835/.github/workflows/ci-dir-misc.yml#L24)
Pruned from the round-3 comment. Keeping it out.

The harness is absent from the fixed misc matrix, so its tests and the agent-contract guarantee never run in CI even as the docs call it executable.
