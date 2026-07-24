# Batch status — review all (started 2026-07-24)

Model claude-opus-4-8, reviewer davd-gzl. Normal (non-deep) mode.

## External-contribution safety gate

Run before any review work. Five of the seven PRs come from `FIRST_TIME_CONTRIBUTOR` accounts.
Static danger check over the raw diffs, nothing executed.

| PR | Author | Files | Result |
|---|---|---|---|
| [6003](https://github.com/gnolang/gno/pull/6003) | ygd58 | 1 test file | clear |
| [5998](https://github.com/gnolang/gno/pull/5998) | ygd58 | 2 `.gno` | clear |
| [5986](https://github.com/gnolang/gno/pull/5986) | ygd58 | 2 `.go` | clear |
| [5985](https://github.com/gnolang/gno/pull/5985) | ygd58 | 6 (4 new) | clear |
| [5983](https://github.com/gnolang/gno/pull/5983) | zeycan1 | 2 new `.gno` | clear |

Checks:

- No `.github/workflows`, Makefile, `go.mod`, `go.sum`, `package.json`, Dockerfile, or `.sh`
  touched. Zero build, CI, or dependency surface across all five.
- No `os/exec`, `net/http`, `net.Dial`, `syscall`, `go:generate`, `go:embed`, Go `unsafe`,
  base64 or hex decode, environment or credential reads, and no filesystem writes.
- Trojan Source scan: every added line is pure ASCII in all five diffs. No bidirectional
  overrides, no zero-width characters, no homoglyphs.
- The `unsafe` import in 5983 is `chain/runtime/unsafe`, the gno stdlib package behind
  `PreviousRealm()`, not Go's `unsafe`.
- The `go/token` imports in 5985 are the Go parser, used to rewrite string variables in
  memory. Nothing is written back to disk.

Non-malicious risks carried into the reviews: 5985 rewrites source before parsing, 5983 moves
ugnot through a per-token vault banker, 6003 unskips a 1 GB benchmark.

## Dropped

| Reason | PRs |
|---|---|
| dependabot | 6005, 5992, 5990, 5989 |
| WIP-titled | 5922, 5263, 5223, 4949 |
| authored by reviewer (davd-gzl) | 5993, 5950, 5936, 5934 |

## Final set (7)

All seven are first rounds. No head-unchanged, already-APPROVED, or patch-id gate applied.

| PR | Head sha | Author | Worktree | Review dir | Mode |
|---|---|---|---|---|---|
| [6003](https://github.com/gnolang/gno/pull/6003) | `32ca59929` | ygd58 | `.worktrees/gno-review-6003` | `reviews/pr/6xxx/6003-*/1-32ca59929/` | bot |
| [6002](https://github.com/gnolang/gno/pull/6002) | `6886aa7bc` | aeddi | `.worktrees/gno-review-6002` | `reviews/pr/6xxx/6002-*/1-6886aa7bc/` | normal |
| [5998](https://github.com/gnolang/gno/pull/5998) | `cf75d982a` | ygd58 | `.worktrees/gno-review-5998` | `reviews/pr/5xxx/5998-*/1-cf75d982a/` | bot |
| [5986](https://github.com/gnolang/gno/pull/5986) | `223aea42e` | ygd58 | `.worktrees/gno-review-5986` | `reviews/pr/5xxx/5986-*/1-223aea42e/` | bot |
| [5985](https://github.com/gnolang/gno/pull/5985) | `639f73fb3` | ygd58 | `.worktrees/gno-review-5985` | `reviews/pr/5xxx/5985-*/1-639f73fb3/` | bot |
| [5983](https://github.com/gnolang/gno/pull/5983) | `7d9a11104` | zeycan1 | `.worktrees/gno-review-5983` | `reviews/pr/5xxx/5983-*/1-7d9a11104/` | bot |
| [5981](https://github.com/gnolang/gno/pull/5981) | `0558015ac` | Villaquiranm | `.worktrees/gno-review-5981` | `reviews/pr/5xxx/5981-*/1-0558015ac/` | normal |

Bot mode: `Event: COMMENT` regardless of verdict, body opens `[AI bot - Automatic review]`. The
review file keeps its real verdict.

## Dispatch

One `general-purpose` agent per PR, all in one message. The parent created every worktree and
checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch
switch. Subagents write `review_claude-opus-4-8_davd-gzl.md` and `comment_claude-opus-4-8.md`,
and do not commit, push, regenerate indexes, or post.

## Progress

All seven returned; every PR has both a `review_` and a `comment_` file.

| PR | Verdict | Posted event | Findings |
|---|---|---|---|
| [6003](https://github.com/gnolang/gno/pull/6003) | NEEDS DISCUSSION | COMMENT (bot) | 1 Warning, 1 Missing test, 1 Nit, 1 Suggestion |
| [6002](https://github.com/gnolang/gno/pull/6002) | APPROVE | not posted | 1 Warning, 1 Missing test, 1 Nit, 1 Suggestion |
| [5998](https://github.com/gnolang/gno/pull/5998) | NEEDS DISCUSSION | COMMENT (bot) | 1 Warning, 1 Missing test, 2 Nits, 1 Suggestion |
| [5986](https://github.com/gnolang/gno/pull/5986) | APPROVE | COMMENT (bot) | 1 Missing test, 1 Nit |
| [5985](https://github.com/gnolang/gno/pull/5985) | REQUEST CHANGES | COMMENT (bot) | 3 Warnings, 1 Missing test, 2 Nits |
| [5983](https://github.com/gnolang/gno/pull/5983) | REQUEST CHANGES | COMMENT (bot) | 3 Warnings, 1 Missing test, 4 Nits, 2 Suggestions |
| [5981](https://github.com/gnolang/gno/pull/5981) | REQUEST CHANGES | not posted | 2 Warnings, 1 Missing test, 2 Nits, 1 Suggestion |

## Cross-PR notes

**CI never ran on the external PRs.** 6003, 5998, 5986, 5985 and 5983 all sit behind the
first-time-contributor approval gate, so only the bot and labelling jobs executed. The local
runs in each review are the only test signal on those five diffs.

**Seeded angles that died under measurement.** Several hypotheses handed to the agents did not
survive a real run and were recorded as Verified rather than as findings: 5985's `go/parser`
mismatch has no trigger (1401 examples, 253 stdlibs and 2503 filetests parse clean, the only 9
failures being deliberately-invalid fixtures) and re-rendering preserves every comment and
filetest directive; 5983's missing amount, recipient and balance validation in `Withdraw` is
caught by the banker on every path except zero. In both cases the strongest finding was one
nobody predicted: 5985's line-shift under `printer.Fprint`, and 5983's `selfPath` versus
`cur.Sub` path divergence.

## Posted

All five bot-mode drafts are up. 6002 and 5981 stay local drafts.

| PR | Review | Inline comments |
|---|---|---|
| [6003](https://github.com/gnolang/gno/pull/6003) | [pullrequestreview-4771979843](https://github.com/gnolang/gno/pull/6003#pullrequestreview-4771979843) | 2 |
| [5998](https://github.com/gnolang/gno/pull/5998) | [pullrequestreview-4771981362](https://github.com/gnolang/gno/pull/5998#pullrequestreview-4771981362) | 3 |
| [5986](https://github.com/gnolang/gno/pull/5986) | [pullrequestreview-4771982861](https://github.com/gnolang/gno/pull/5986#pullrequestreview-4771982861) | 2 |
| [5985](https://github.com/gnolang/gno/pull/5985) | [pullrequestreview-4771984589](https://github.com/gnolang/gno/pull/5985#pullrequestreview-4771984589) | 4 |
| [5983](https://github.com/gnolang/gno/pull/5983) | [pullrequestreview-4771987027](https://github.com/gnolang/gno/pull/5983#pullrequestreview-4771987027) | 9 |

## Still open

- [6002](https://github.com/gnolang/gno/pull/6002) verdict is APPROVE and needs human
  confirmation before it can be posted with `--approve`.
- [5981](https://github.com/gnolang/gno/pull/5981) draft is REQUEST_CHANGES, awaiting a read.
- `reviews/pr/5xxx/5994-bench-caller-controlled-gas-inputs/` and
  `reviews/pr/5xxx/5996-guard-lastcommitid-query/` were already untracked before this batch and
  are left uncommitted.
