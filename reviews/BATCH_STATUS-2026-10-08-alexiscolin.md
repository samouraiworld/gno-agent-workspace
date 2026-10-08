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

Status 2026-10-08: all five rounds drafted, final-checked, pushed and posted.

Round 2 posted 2026-10-08 as an AI review (COMMENT): 6262 review 5458892490, 6297 review 5458894155, 6298 review 5458895890, 6299 review 5458897507.

## Round 2, the author's fixes

Scope: the commits pushed after round 1, each diffed against round 1's head merged locally with master `47d19f8a3`, so no master content enters the diff. The generated `public/main.css` is left out. #6293's head did not move, so it has no round 2.

| PR | Old head | New head | Diff base (local) | Shape | Projected | Round dir | Run |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [gnolang/gno#6262](https://github.com/gnolang/gno/pull/6262) | `c5ae4b1` | `8abf28b` | `d631c44` | solo | $4 | `6262-realm-package-breadcrumb-switch/2-8abf28b` | `wf_56ffe08f-732` |
| [gnolang/gno#6297](https://github.com/gnolang/gno/pull/6297) | `56ff177` | `6a68bc6` | `84df2c4` | solo wider, 2 finders | $12 | `6297-gnoweb-inline-icons/2-6a68bc6` | `wf_32f71aaa-ca5` |
| [gnolang/gno#6298](https://github.com/gnolang/gno/pull/6298) | `065ec36` | `a4be1f6` | `ab27ce5` | solo wider, 2 finders | $8 | `6298-gnoweb-markdown-button/2-a4be1f6` | `wf_64ea4525-00f` |
| [gnolang/gno#6299](https://github.com/gnolang/gno/pull/6299) | `1f9bf51` | `11a5d79` | `96fcf8a` (the author's own merge) | solo wider, 2 finders | $8 | `6299-gnoweb-frame-block/2-11a5d79` | `wf_32b81a43-c5b` |

Round inputs at the workspace's `.worktrees/<n>-round2/`.
