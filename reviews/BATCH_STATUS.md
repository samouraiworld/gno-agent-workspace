# Batch status — review all (started 2026-07-19)

Scope confirmed by user: every open, non-draft gnolang/gno PR absent from `reviews/pr/`, minus the reviewer's own PRs. Model claude-opus-4-8, reviewer davd-gzl. Normal (non-deep) mode. Nothing posted.

## Dropped

| Reason | PRs |
|---|---|
| WIP-titled | 5922, 5263, 5223, 4949 |
| dependabot | 5968, 5953, 5952 |
| authored by reviewer (davd-gzl) | 5978, 5950, 5936, 5934 |

## Final set (16)

| PR | Head sha | Author | Worktree | Review dir |
|---|---|---|---|---|
| [5980](https://github.com/gnolang/gno/pull/5980) | `c9aaed5c8` | ygd58 | `.worktrees/gno-review-5980` | `reviews/pr/5xxx/5980-*/1-c9aaed5c8/` |
| [5976](https://github.com/gnolang/gno/pull/5976) | `0080497c9` | zardozmonopoly | `.worktrees/gno-review-5976` | `reviews/pr/5xxx/5976-*/1-0080497c9/` |
| [5975](https://github.com/gnolang/gno/pull/5975) | `e3b7a7934` | zardozmonopoly | `.worktrees/gno-review-5975` | `reviews/pr/5xxx/5975-*/1-e3b7a7934/` |
| [5970](https://github.com/gnolang/gno/pull/5970) | `f620d1c5c` | D4ryl00 | `.worktrees/gno-review-5970` | `reviews/pr/5xxx/5970-*/1-f620d1c5c/` |
| [5969](https://github.com/gnolang/gno/pull/5969) | `7e0728bd5` | Romainua | `.worktrees/gno-review-5969` | `reviews/pr/5xxx/5969-*/1-7e0728bd5/` |
| [5964](https://github.com/gnolang/gno/pull/5964) | `37b883fca` | jefft0 | `.worktrees/gno-review-5964` | `reviews/pr/5xxx/5964-*/1-37b883fca/` |
| [5959](https://github.com/gnolang/gno/pull/5959) | `cc90c35f3` | ygd58 | `.worktrees/gno-review-5959` | `reviews/pr/5xxx/5959-*/1-cc90c35f3/` |
| [5958](https://github.com/gnolang/gno/pull/5958) | `f2e427a71` | ygd58 | `.worktrees/gno-review-5958` | `reviews/pr/5xxx/5958-*/1-f2e427a71/` |
| [5956](https://github.com/gnolang/gno/pull/5956) | `1ad3009b9` | ygd58 | `.worktrees/gno-review-5956` | `reviews/pr/5xxx/5956-*/1-1ad3009b9/` |
| [5951](https://github.com/gnolang/gno/pull/5951) | `9208bed41` | zardozmonopoly | `.worktrees/gno-review-5951` | `reviews/pr/5xxx/5951-*/1-9208bed41/` |
| [5946](https://github.com/gnolang/gno/pull/5946) | `037a90410` | zardozmonopoly | `.worktrees/gno-review-5946` | `reviews/pr/5xxx/5946-*/1-037a90410/` |
| [5945](https://github.com/gnolang/gno/pull/5945) | `fc4052651` | aeddi | `.worktrees/gno-review-5945` | `reviews/pr/5xxx/5945-*/1-fc4052651/` |
| [5943](https://github.com/gnolang/gno/pull/5943) | `05e124d67` | zeycan1 | `.worktrees/gno-review-5943` | `reviews/pr/5xxx/5943-*/1-05e124d67/` |
| [5941](https://github.com/gnolang/gno/pull/5941) | `d415ef332` | coinsspor | `.worktrees/gno-review-5941` | `reviews/pr/5xxx/5941-*/1-d415ef332/` |
| [5935](https://github.com/gnolang/gno/pull/5935) | `4369fdca7` | ltzmaxwell | `.worktrees/gno-review-5935` | `reviews/pr/5xxx/5935-*/1-4369fdca7/` |
| [5923](https://github.com/gnolang/gno/pull/5923) | `dcd6db417` | Villaquiranm | `.worktrees/gno-review-5923` | `reviews/pr/5xxx/5923-*/1-dcd6db417/` |

All 16 are first rounds; no head-unchanged, already-APPROVED, or patch-id gate applied.

## Dispatch

One `general-purpose` agent per PR, all in one message. The parent created every worktree and checked out every PR head; subagents never run `worktree add`, `gh pr checkout`, or any branch switch. Subagents write `review_claude-opus-4-8_davd-gzl.md` and `comment_claude-opus-4-8.md`, and do not commit, push, regenerate indexes, or post.

## Progress

All 16 returned; every PR has both a `review_` and a `comment_` file.

| PR | Verdict | Findings |
|---|---|---|
| [5980](https://github.com/gnolang/gno/pull/5980) | REQUEST CHANGES | 2 Critical, 5 Warnings, 1 Missing test, 4 Nits, 3 Suggestions |
| [5976](https://github.com/gnolang/gno/pull/5976) | REQUEST CHANGES | 1 Critical, 1 Warning, 3 Nits, 2 Missing tests, 2 Suggestions |
| [5975](https://github.com/gnolang/gno/pull/5975) | REQUEST CHANGES | 2 Critical, 1 Warning, 3 Nits, 1 Missing test, 1 Suggestion |
| [5970](https://github.com/gnolang/gno/pull/5970) | REQUEST CHANGES | 2 Warnings, 2 Nits, 2 Suggestions |
| [5969](https://github.com/gnolang/gno/pull/5969) | REQUEST CHANGES | 2 Warnings, 2 Missing tests, 1 Nit |
| [5964](https://github.com/gnolang/gno/pull/5964) | REQUEST CHANGES | 1 Warning, 1 Missing test |
| [5959](https://github.com/gnolang/gno/pull/5959) | REQUEST CHANGES | 1 Critical, 3 Warnings, 1 Missing test, 4 Nits, 1 Suggestion |
| [5958](https://github.com/gnolang/gno/pull/5958) | REQUEST CHANGES | 2 Critical, 4 Warnings, 2 Nits, 2 Missing tests, 1 Suggestion |
| [5956](https://github.com/gnolang/gno/pull/5956) | REQUEST CHANGES | 1 Critical, 3 Warnings, 2 Missing tests, 3 Nits, 2 Suggestions |
| [5951](https://github.com/gnolang/gno/pull/5951) | REQUEST CHANGES | 1 Critical, 3 Warnings, 1 Missing test, 4 Nits, 4 Suggestions |
| [5946](https://github.com/gnolang/gno/pull/5946) | REQUEST CHANGES | 3 Critical, 1 Warning, 3 Nits, 1 Missing test, 3 Suggestions |
| [5945](https://github.com/gnolang/gno/pull/5945) | REQUEST CHANGES | 1 Critical, 2 Warnings, 3 Nits, 4 Suggestions |
| [5943](https://github.com/gnolang/gno/pull/5943) | REQUEST CHANGES | 1 Warning, 1 Suggestion |
| [5941](https://github.com/gnolang/gno/pull/5941) | APPROVE | 1 Nit (not posted) |
| [5935](https://github.com/gnolang/gno/pull/5935) | NEEDS DISCUSSION | 2 Warnings, 1 Missing test, 1 Suggestion |
| [5923](https://github.com/gnolang/gno/pull/5923) | REQUEST CHANGES | 3 Warnings, 1 Missing test, 1 Nit, 1 Suggestion |

## Cross-PR findings

**privval set (5980, 5959, 5958, 5956).** All four import a cloud or HSM SDK without adding it to `go.mod`/`go.sum`, so each branch fails `go build ./...` across the whole module. CI hid it on all four: the fork contributors' build and test workflows sit behind the initial-approval bot and never ran, leaving only the conventional-commit title check visible. All four also add a backend field to `PrivValidatorConfig` with its own pairwise exclusion term and its own `errMultipleSignerSourcesSet`, so they conflict pairwise and merging any two leaves the cross-pair unguarded while the duplicate error variables collide. Merging all four links four cloud SDKs into every `gnoland` binary.

**zardozmonopoly examples (5975, 5946).** The same unguarded `a*b` before division appears in both, wrapping `int64` at pool sizes a demo would plausibly hit.

## Decisions needed

- [5943](https://github.com/gnolang/gno/pull/5943): davd-gzl already submitted an APPROVED review on this exact commit (`pullrequestreview-4683747087`, 2026-07-13). Posting REQUEST_CHANGES supersedes it.
- [5969](https://github.com/gnolang/gno/pull/5969): borderline. Both Warnings are pre-existing and outside linked issue 5957, which the PR fully closes; REQUEST_CHANGES rests mainly on the untested assignability change. COMMENT is defensible.
- [5970](https://github.com/gnolang/gno/pull/5970): verdict rests entirely on two documentation defects. COMMENT is defensible.
- [5958](https://github.com/gnolang/gno/pull/5958): the config-fallback finding is filed Critical because the failure is silent and yields a wrong validator identity. Downgrades cleanly to Warning without changing the verdict.
- [5956](https://github.com/gnolang/gno/pull/5956): the failing `check` job is the conventional-commit title lint rejecting `privval:`. Left out of the draft as contribution-convention.
- [5941](https://github.com/gnolang/gno/pull/5941): APPROVE needs explicit human confirmation before posting.

## Notes

- `.worktrees/gno-review-5945` holds 210 MB of gitignored build output under `misc/deployments/topaz.gno.land/work/` from the genesis reproduction run. Left in place for re-verification.

## Resume / finalize

1. Re-dispatch any PR whose review file is missing, per `skills/review.md` *Parallel dispatch*.
2. `./scripts/build-indexes.sh`
3. `git add reviews/ docs/glossary.md && git commit -m "review: batch of 16 open PRs" && git push`
4. Post nothing until the user says `post` per draft.
