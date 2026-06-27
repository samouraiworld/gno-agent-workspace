# PR #5117: chore: update multisig docs: default sorting, derivation guidance and examples

URL: https://github.com/gnolang/gno/pull/5117
Author: D4ryl00 | Base: master | Files: 1 | +91 -68
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: bff5572db (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5117 bff5572db`

Round 2. Head advanced beabe42 → bff5572db via a single master merge. The authored multisig prose is byte-identical to round 1; the only change inside the multisig section is one example line that now passes `-broadcast=false`, picked up from master's broadcast-default flip. Verdict unchanged from round 1.

**TL;DR:** Rewrites the `gnokey` multisig walkthrough in `docs/users/interact-with-gnokey.md`. The old text told users that all participants must list multisig members in the exact same order or signing breaks; that is false, because `gnokey add multisig` sorts members by address unless you opt out. The rewrite corrects this and reframes the rule as "same member set, same threshold".

**Verdict: APPROVE** — docs-only; the three load-bearing claims (default sort by derived address, pubkey-based signature matching, member-set invariant) re-verified against `tm2/pkg/crypto/keys/client/`. One stale term and a stale PR-body dependency line remain; neither blocks.

## Summary

`gnokey add multisig` defaults to `-nosort=false`, which sorts member pubkeys by derived address before building the multisig pubkey ([`tm2/pkg/crypto/keys/client/add_multisig.go:98-106`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/add_multisig.go#L98-L106) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L98-L106)), so CLI member order does not affect the resulting key. `gnokey multisign` matches each signature to its member via the pubkey embedded in the signature file ([`tm2/pkg/crypto/keys/client/multisign.go:137-143`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/multisign.go#L137-L143) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/multisign.go#L137-L143)), not by CLI position. The prior doc framed key ordering as "the single most important rule" and signature-slot ordering as "the second most important rule"; both were wrong. The rewrite drops the per-keybase ordering admonitions, explains the default address sort, reframes the invariant as same member set plus same threshold, adds a concept intro (`k-of-n`, member set, threshold), and recommends a dedicated signer key.

## What changed since round 1 (beabe42)

The PR's diff against its current merge-base is now confined to the multisig section. The other deltas visible in a raw `beabe42..HEAD` (broadcast default, `-chainid`, `int64`, the wugnot/valoper query examples) are master's own edits to the same file, merged in, not authored by this PR. Inside the authored section, the single substantive change is the example at [`docs/users/interact-with-gnokey.md:844`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L844) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L844): the unsigned-tx `maketx send` now carries `-broadcast=false`. Master flipped the `-broadcast` default to `true` ([`tm2/pkg/crypto/keys/client/maketx.go:96-101`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/maketx.go#L96-L101) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/maketx.go#L96-L101)), so this example needs the explicit opt-out to write an unsigned payload to a file. The line is correct and necessary on the new default.

Round 1 also flagged the in-page anchor `#creating-the-same-multisig-key-on-every-machine` as needing a render check before merge. Resolved: both the anchor and its `#generating-a-key-pair` sibling point at real headers ([`docs/users/interact-with-gnokey.md:736`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L736) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L736), [`docs/users/interact-with-gnokey.md:37`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L37) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L37)) whose Docusaurus slugs match the links, and the `docs` CI build passes.

Round 1's beginner-clarity Nit on the `-nosort` sentence is dropped: gfanton raised the same point ([#discussion_r2768667149](https://github.com/gnolang/gno/pull/5117#discussion_r2768667149)), the author accepted a rewrite, and the current line ([`docs/users/interact-with-gnokey.md:750-751`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L750-L751) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L750-L751)) reads clearly.

## Verification

- Sort default: [`tm2/pkg/crypto/keys/client/add_multisig.go:45-51`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/add_multisig.go#L45-L51) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L45-L51) — `NoSort` registers with default `false`, matching the doc's `-nosort=false`.
- Sort key: [`tm2/pkg/crypto/keys/client/add_multisig.go:98-106`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/add_multisig.go#L98-L106) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L98-L106) — sorts on `publicKeys[i].Address().Compare(...)`, matching "by their derived addresses".
- Signature matching: [`tm2/pkg/crypto/keys/client/multisign.go:137-143`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/multisign.go#L137-L143) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/multisign.go#L137-L143) — `AddSignatureFromPubKey(sig.Signature, sig.PubKey, multisigPub.PubKeys)` resolves the slot from the signature's embedded pubkey, so `--signature` order is irrelevant, matching "you can pass signature files in any order".
- `gnokey add bech32` subcommand exists: [`tm2/pkg/crypto/keys/client/add_bech32.go:27-29`](https://github.com/gnolang/gno/blob/bff5572db/tm2/pkg/crypto/keys/client/add_bech32.go#L27-L29) · [↗](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_bech32.go#L27-L29), matching the doc's `gnokey add bech32` instruction.

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`docs/users/interact-with-gnokey.md:695`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L695) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L695) — Section header says "multi-signature key" but the body says "multisig" 79 times; "multi-signature" appears only here. Pick one and use it consistently in the header. Related to davd-gzl's earlier thread ([#discussion_r2841431405](https://github.com/gnolang/gno/pull/5117#discussion_r2841431405)).

## Missing Tests

N/A — docs-only.

## Suggestions

- [`docs/users/interact-with-gnokey.md:896-897`](https://github.com/gnolang/gno/blob/bff5572db/docs/users/interact-with-gnokey.md?plain=1#L896-L897) · [↗](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L896-L897) — The "Charlie would do the same ... but a 2-of-3 only needs 2 signatures" parenthetical sits mid-flow. A one-line note above section 3 ("for a 2-of-3, any two of alice, bob, charlie suffice") would read cleaner. Optional.

## Open questions

- The PR body still ends with "depends on https://github.com/gnolang/gno/issues/5122" (open), but the final diff no longer uses `--derivation-path` / `--account --index` and instead links to the existing "Generating a key pair" section, so the dependency is stale. Not posted: a PR-description edit the author makes, not a code finding.

## Status

- [@davd-gzl approved](https://github.com/gnolang/gno/pull/5117#pullrequestreview-3938080089) on commit 53b21e9a (2026-03-12).
- [@jeronimoalbi approved](https://github.com/gnolang/gno/pull/5117#pullrequestreview-4119574533) on commit bff5572d (current HEAD, 2026-04-16).
- CI: build, e2e-test, docs, genproto, mod-tidy all pass. Only `Merge Requirements` fails, gated on the bot's manual `IGNORE` checkbox or tech-staff review, not a code problem ([bot summary](https://github.com/gnolang/gno/pull/5117#issuecomment-3842326019)).
