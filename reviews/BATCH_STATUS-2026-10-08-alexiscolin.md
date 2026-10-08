# Batch status: alexiscolin's review requests, 2026-10-08

Scope: every pull request by `alexiscolin` on `gnolang/gno` that requested a review from `davd-gzl` on 2026-10-08, read from the notifications API and each pull request's timeline. Confirmed by the user before dispatch. Word: `review`.

| PR | Head | Base | Shape | Projected | Review dir | Run |
| --- | --- | --- | --- | --- | --- | --- |
| [gnolang/gno#6262](https://github.com/gnolang/gno/pull/6262) | `c5ae4b1` | `41841e9` | solo wider, 2 finders | $14 | `reviews/pr/6xxx/6262-realm-package-breadcrumb-switch/1-c5ae4b1` | `wf_32ab428e-bdc` |
| [gnolang/gno#6293](https://github.com/gnolang/gno/pull/6293) | `c2e4e02` | `b065651` | solo | $6 | `reviews/pr/6xxx/6293-mainnet-home-visitor-routing/1-c2e4e02` | `wf_e45825ff-95a` |
| [gnolang/gno#6297](https://github.com/gnolang/gno/pull/6297) | `56ff177` | `41841e9` | pipeline | $19 | `reviews/pr/6xxx/6297-gnoweb-inline-icons/1-56ff177` | `wf_7144bca8-998` |
| [gnolang/gno#6298](https://github.com/gnolang/gno/pull/6298) | `065ec36` | `b065651` | pipeline, own run (changes a VM native) | $17 | `reviews/pr/6xxx/6298-gnoweb-markdown-button/1-065ec36` | `wf_ca668205-a53` |
| [gnolang/gno#6299](https://github.com/gnolang/gno/pull/6299) | `1f9bf51` | `41841e9` | pipeline | $19 | `reviews/pr/6xxx/6299-gnoweb-frame-block/1-1f9bf51` | `wf_6b8ea3f0-10c` |

Dropped, merged before the batch: [gnolang/gno#6294](https://github.com/gnolang/gno/pull/6294), [gnolang/gno#6295](https://github.com/gnolang/gno/pull/6295), [gnolang/gno#6296](https://github.com/gnolang/gno/pull/6296).

Coupling: 6297, 6298 and 6299 each carry `gno.land/pkg/gnoweb/markdown/utils.go` (`scanGnoTag`), stated byte-for-byte identical; a finding on it is reconciled across all three drafts before the commit.

Resume: worktrees at `.worktrees/gno-review-<n>` and `-base`; round inputs at the workspace's `.worktrees/<n>-round/`; resume a run with its run id and the same `args_file`.

Status 2026-10-08: all five rounds drafted, final-checked and pushed; each waits on `post as an AI`.
