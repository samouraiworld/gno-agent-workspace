# Review: PR #5117
Event: APPROVE

## Body
Verified on bff5572db: the default sort by derived address, the pubkey-based signature matching, and the `-broadcast=false` example all match `tm2/pkg/crypto/keys/client/` and master's new `-broadcast=true` default.

The PR body still says it depends on [#5122](https://github.com/gnolang/gno/issues/5122), but the merged diff no longer uses `--derivation-path` and links to the existing "Generating a key pair" section instead, so that dependency line is stale.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5117-multisig-docs-default-sorting/2-bff5572db/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## docs/users/interact-with-gnokey.md:896-897 [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L896-L897)
The "Charlie would do the same ... but a 2-of-3 only needs 2 signatures" parenthetical interrupts the signing flow. A one-line note above section 3 saying any two of alice, bob, charlie suffice would carry it better.
