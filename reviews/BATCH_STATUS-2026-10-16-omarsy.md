# Batch: omarsy's unreviewed gno pull requests, 2026-10-16

Scope: every open `gnolang/gno` pull request by `omarsy` with no round under
`reviews/pr/` and no review by `davd-gzl` on GitHub, drafts included at the
user's request. Each round posts with `post as an AI` once it finishes.
Checkout synced at `origin/main`, 0 behind.

| PR | Draft | Head | Shape | Measured | Status | Review dir |
| --- | --- | --- | --- | --- | --- | --- |
| [6272](https://github.com/gnolang/gno/pull/6272) | no | `03363f296` | solo, class simple | $2.91 + $3.92 | round 1 stopped at the judge; round 2 [posted](https://github.com/gnolang/gno/pull/6272#pullrequestreview-5548547488) | `reviews/pr/6xxx/6272-getnamesource-fileset-decls/2-03363f296/` |
| [6270](https://github.com/gnolang/gno/pull/6270) | no | `fe1c9f046` | pipeline | $39.28 | [posted](https://github.com/gnolang/gno/pull/6270#pullrequestreview-5547698095) | `reviews/pr/6xxx/6270-gnoweb-ipfs-gateway/1-fe1c9f046/` |
| [6209](https://github.com/gnolang/gno/pull/6209) | yes | `1de7f587d` | solo, 2 finders | $9.50 | [posted](https://github.com/gnolang/gno/pull/6209#pullrequestreview-5548097705) | `reviews/pr/6xxx/6209-loopvar-rename-clause/1-1de7f587d/` |
| [5977](https://github.com/gnolang/gno/pull/5977) | yes | `0089e5d95` | solo, 2 finders | $7.13 | [posted](https://github.com/gnolang/gno/pull/5977#pullrequestreview-5547557958) | `reviews/pr/5xxx/5977-tryevalstatic-error-result/1-0089e5d95/` |
| [5884](https://github.com/gnolang/gno/pull/5884) | yes | `fb16708ac` | solo, 2 finders | $6.10 | [posted](https://github.com/gnolang/gno/pull/5884#pullrequestreview-5548671499) | — |
| [5820](https://github.com/gnolang/gno/pull/5820) | yes | `8995e9424` | solo, 2 finders | $6.38 | [posted](https://github.com/gnolang/gno/pull/5820#pullrequestreview-5547554754) | `reviews/pr/5xxx/5820-promoted-field-depth-shadowing/1-8995e9424/` |
| [5785](https://github.com/gnolang/gno/pull/5785) | yes | `b42a521b1` | solo, 2 finders | $11.26 | [posted](https://github.com/gnolang/gno/pull/5785#pullrequestreview-5547651848) | `reviews/pr/5xxx/5785-type-identity-assignability/1-b42a521b1/` |

Dropped, already reviewed (a round here or a `davd-gzl` review on GitHub):
6271, 6137, 6081, 6078, 5967, 5965, 5960, 5901, 5689, 5478, 5382.

Worktrees: `.worktrees/gno-review-<n>` and `-base` in this checkout. Setup
inputs: `.worktrees/review-<n>/setup/` at the workspace root.

Resume: relaunch `scripts/workflows/review-pipeline.js` with
`{"args_file": "<workspace>/.worktrees/review-<n>/setup/args.js"}` for any row
with no draft.
