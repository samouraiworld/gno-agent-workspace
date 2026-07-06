# Review: PR [#5873](https://github.com/gnolang/gno/pull/5873)
Event: REQUEST_CHANGES

## Body
The reference is accurate: every flag and query endpoint I checked matches `tm2/pkg/crypto/keys/client` and `gno.land/pkg/sdk/vm/handler.go`. Running `make generate` in `docs/` regenerates `misc/docs/sidebar.json` and clears the red `docs` check; the two stale nav slugs and the red check share that root cause. I applied all the fixes below locally and re-ran the `docs` gate: `make generate -B` swaps the two dead slugs and is idempotent, `make lint` reports no issues. The linter does not check cross-file heading anchors, so I resolved the repointed link separately; `./gnokey-reference.md#airgapped-signing` lands on the `Airgapped signing` heading.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5873-rewrite-gnokey-guide-reference/1-66552ff7a/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/resources/gnodev-reference.md:118-119 [↗](../../../../../.worktrees/gno-review-5873/docs/resources/gnodev-reference.md#L118)
This links to `../users/interact-with-gnokey.md#making-an-airgapped-transaction`, a file this PR deleted. The airgapped flow now lives in `gnokey-reference.md`; point it at `./gnokey-reference.md#airgapped-signing`.

## docs/builders/getting-started.md:275 [↗](../../../../../.worktrees/gno-review-5873/docs/builders/getting-started.md#L275)
The testnet URL cell `` `https://``rpc.<testN>...` `` breaks the anti-autolink split. The first single-backtick span closes early and the following double backtick opens an unclosed run, so the backticks render literally instead of forming one inline URL. Restore the zero-width-space split, or use two separate single-backtick spans.

## docs/users/using-gnokey.md:141 [↗](../../../../../.worktrees/gno-review-5873/docs/users/using-gnokey.md#L141)
"Calling `Deposit` on the `wugnot` realm to wrap `1000ugnot`." has no main verb. Fold it into the surrounding sentence or make it a full one.

## docs/builders/getting-started.md:369 [↗](../../../../../.worktrees/gno-review-5873/docs/builders/getting-started.md#L369)
The link text reads "`addpkg` in Interact with gnokey" but the target is now the page titled "gnokey command reference". The same stale label sits over reference-page targets in query-state-api.md:6 and :210, rpc-clients.md:29, and gno-packages.md:49. Links resolve; only the label is stale.
