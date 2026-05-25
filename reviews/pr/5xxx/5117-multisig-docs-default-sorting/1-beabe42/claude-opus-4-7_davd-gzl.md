# PR #5117: chore: update multisig docs: default sorting, derivation guidance and examples

URL: https://github.com/gnolang/gno/pull/5117
Author: D4ryl00 | Base: master | Files: 1 | +91 -68
Reviewed by: davd-gzl | Model: claude-opus-4-7

Verdict: APPROVE — docs-only rewrite of the multisig walkthrough; technical claims (default pubkey sort, pubkey-based signature matching) verified against `tm2/pkg/crypto/keys/client/`; the earlier dependency on the broken `--derivation-path` flag (issue #5122) was dropped from the final diff. Two prior approvals already on file ([@davd-gzl on 53b21e9a](https://github.com/gnolang/gno/pull/5117#pullrequestreview-2967773636), [@jeronimoalbi on beabe422](https://github.com/gnolang/gno/pull/5117#pullrequestreview-3055835653)).

## Summary

The previous version of `docs/users/interact-with-gnokey.md` told users that `gnokey add multisig` requires a strict, identical member ordering across all participating keybases and that `gnokey multisign` signature slots must match that order — both wrong. `gnokey add multisig` defaults to `-nosort=false`, which sorts member pubkeys by derived address before constructing the multisig pubkey ([`tm2/pkg/crypto/keys/client/add_multisig.go:100-104`](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L100-L104)), and `gnokey multisign` matches each signature to its corresponding member via `AddSignatureFromPubKey(...)` ([`tm2/pkg/crypto/keys/client/multisign.go:138-144`](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/multisign.go#L138-L144)), not by CLI position. The rewrite corrects both claims, adds a short concept intro (`k-of-n`, member set, threshold), and recommends a dedicated signer key for operational hygiene.

## Fix

Before: docs framed key ordering as the "single most important rule," with a canonical `alice / multisig-bob / multisig-charlie` order repeated in three keybase setups, and signature-slot ordering as the "second most important rule." After: docs explain the default sort by derived address, drop the per-keybase ordering admonitions, and reframe the invariant as "same member set + same threshold" (which is what actually matters at the protocol level). The diagram and the `gnokey multisign` section are updated consistently. An earlier draft included `gnokey add --recover --account 0 --index 1` examples that hit the bug tracked in [#5122](https://github.com/gnolang/gno/issues/5122); those were removed and replaced with a generic pointer to the existing [Generating a key pair](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L44) section.

## Verification

- Sort default: [`tm2/pkg/crypto/keys/client/add_multisig.go:46-51`](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L46-L51) — `NoSort` flag defaults to `false`, matches doc claim `-nosort=false`.
- Sort key: [`tm2/pkg/crypto/keys/client/add_multisig.go:100-104`](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/add_multisig.go#L100-L104) — `publicKeys[i].Address().Compare(...)`, matches doc claim "sorts the member pubkeys ... by their derived addresses".
- Signature matching: [`tm2/pkg/crypto/keys/client/multisign.go:138-144`](../../../../../.worktrees/gno-review-5117/tm2/pkg/crypto/keys/client/multisign.go#L138-L144) — `AddSignatureFromPubKey(sig.Signature, sig.PubKey, multisigPub.PubKeys)` resolves the slot from the embedded `sig.PubKey`, so CLI `--signature` order is irrelevant. Matches doc claim "you can pass signature files in any order".

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`docs/users/interact-with-gnokey.md:681`](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L681) — Section title mixes `k-of-n multi-signature key` with `multisig` throughout the body; pick one term and stay with it. Carried over from previous comment thread ([#discussion_r2841431405](https://github.com/gnolang/gno/pull/5117#discussion_r2841431405)).
- [`docs/users/interact-with-gnokey.md:887`](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L887) — In-page anchor `[Creating the same multisig key on every machine](#creating-the-same-multisig-key-on-every-machine)` depends on Docusaurus auto-anchor slugging; worth a one-line render check before merge (preview env or `npm run build` in `docs/`).
- [`docs/users/interact-with-gnokey.md:736`](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L736) — "If you set `-nosort`" is correct (Go flag implicit-true), but a beginner reader may not know that. Optional: "If you pass `-nosort` (which sets it to `true`), ...".

## Missing Tests

N/A — docs-only.

## Suggestions

- [`docs/users/interact-with-gnokey.md:881-883`](../../../../../.worktrees/gno-review-5117/docs/users/interact-with-gnokey.md#L881-L883) — The 2-of-3 "Charlie optionally signs" parenthetical reads slightly off in flow. Consider promoting it to a one-line note above section 3 ("For a 2-of-3, any two of {alice, bob, charlie} suffice"). Optional.

## Questions for Author

- The PR body still says "depends on https://github.com/gnolang/gno/issues/5122", but the final diff no longer uses `--derivation-path` / `--account --index`. Worth removing the depends-on line from the PR description so reviewers don't think this is still blocked.

## Status

- [@davd-gzl approved](https://github.com/gnolang/gno/pull/5117#pullrequestreview-2967773636) on commit `53b21e9a` (2026-03-12).
- [@jeronimoalbi approved](https://github.com/gnolang/gno/pull/5117#pullrequestreview-3055835653) on commit `beabe422` (HEAD, 2026-04-16).
- Only outstanding CI item is the `Merge Requirements` bot, which fails because the bot wants either tech-staff review or the manual `IGNORE` checkbox set; the underlying `User davd-gzl already reviewed PR 5117 with state APPROVED` condition is already green ([bot summary](https://github.com/gnolang/gno/pull/5117#issuecomment-3842326019)).
- All other checks (build, e2e-test, docs, genproto, mod-tidy, etc.) pass.
