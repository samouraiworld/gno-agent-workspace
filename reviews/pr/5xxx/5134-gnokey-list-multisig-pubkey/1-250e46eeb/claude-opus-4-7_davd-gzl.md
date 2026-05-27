# PR #5134: feat: improve gnokey list multisig pubkey

URL: https://github.com/gnolang/gno/pull/5134
Author: D4ryl00 | Base: master | Files: 3 | +154 -13
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5134 250e46eeb` (then `gh -R gnolang/gno pr checkout 5134` inside it)

**Verdict: APPROVE** — small, focused UX fix; one minor inconsistency between `gnokey list` and `gnokey add` output formats worth fixing in the same PR, and a flag-naming nit.

## Summary

Fixes [#5125](https://github.com/gnolang/gno/issues/5125): `gnokey list` printed a multisig key's members as a comma-joined array, leaving no way to copy the multisig pubkey itself (needed for `gnokey add bech32`-style imports on another machine). After: multi-type entries print the bech32-encoded multisig pubkey by default; pass `-multisig-members` to fall back to the member-list view, one key per line. Side change: reorders the per-entry fields to `addr / path / pub` so the long bech32 lands at the end of the line, and tightens `PubKeyMultisigThreshold.String()` to multi-line with an explicit empty-case guard.

## Glossary

- `PubKeyToBech32`: encodes any `crypto.PubKey`'s amino-marshalled bytes with the `gpub` HRP — works uniformly for secp256k1, ed25519, and multisig ([`tm2/pkg/crypto/bech32.go:28-34`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/bech32.go#L28-L34) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/bech32.go#L28-L34)).
- `TypeMulti`: keybase tag (`"multi"`) for keys created via `kb.CreateMulti(...)` ([`tm2/pkg/crypto/keys/types.go:72,79`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/types.go#L72-L79) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/types.go#L72-L79)).
- `ListCfg`: new flag-bearing struct for `gnokey list`; mirrors `AddCfg{RootCfg *BaseCfg, ...}` already used elsewhere in this package.

## Fix

Before, `printInfos` hard-coded `keypub` (whose `String()` for a multisig dumped `[member1, member2, ...]`); the address was the only multisig identifier you could share, but you cannot import a key from an address. After, when `keytype == TypeMulti && !showMultisigMembers`, the displayed pubkey is replaced with `crypto.PubKeyToBech32(keypub)`, producing a `gpub1...` string that round-trips through `gnokey add bech32`. For non-multi keys nothing changes semantically: `secp256k1.PubKey.String()` and `ed25519.PubKey.String()` already return `PubKeyToBech32(pubKey)` ([`secp256k1.go:137-139`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/secp256k1/secp256k1.go#L137-L139) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/secp256k1/secp256k1.go#L137-L139), [`ed25519.go:138-140`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/ed25519/ed25519.go#L138-L140) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/ed25519/ed25519.go#L138-L140)) so the branch is a no-op for them.

The `PubKeyMultisigThreshold.String()` reshape ([`threshold_pubkey.go:35-49`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/multisig/threshold_pubkey.go#L35-L49) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/multisig/threshold_pubkey.go#L35-L49)) is what gives `-multisig-members` its readable "one key per line" output: `printInfos` falls back to `keypub.String()` for that path. The empty-case guard short-circuits to `"[]"`; the constructor panics on `len(pubkeys) < k <= 0` ([`threshold_pubkey.go:20-32`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/multisig/threshold_pubkey.go#L20-L32) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/multisig/threshold_pubkey.go#L20-L32)) so an empty `PubKeys` slice is only reachable through direct struct literal construction or amino-decode of malformed bytes — defensive, not load-bearing.

## Critical (must fix)

None.

## Warnings (should fix)

- **[inconsistent list/add output format]** [`tm2/pkg/crypto/keys/client/add.go:294`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/client/add.go#L294) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/client/add.go#L294) — `gnokey add` still prints `addr: %v pub: %v, path: %v`; `gnokey list` now prints `addr: %v path: %v pub: %v`. Same entity, two formats.
  <details><summary>details</summary>

  Both `printInfos` (list) and `printNewInfo` (add) describe a `keys.Info` record. The PR rearranges the list format to put `pub:` last (sensible — bech32 is long and wraps in narrow terminals) but leaves `printNewInfo` untouched, so creating then listing the same key now shows two different field orders. Anyone scripting against either output (existing automation, docs/examples) gets bitten by drift; the user-facing inconsistency is the bigger cost. Fix: update `printNewInfo` to the same `addr / path / pub` order (and drop the stray comma between `pub` and `path`) in the same PR — it is a one-line change and keeps the two display paths in lockstep.
  </details>

- **[flag name not future-proof]** [`tm2/pkg/crypto/keys/client/list.go:36-43`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/client/list.go#L36-L43) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/client/list.go#L36-L43) — `-multisig-members` is a behaviour-modifier targeting one key type; consider `-show-multisig-members` or `-multisig-verbose` for clarity, and document that the flag is a no-op for non-multi entries.
  <details><summary>details</summary>

  This is a Warning, not a Nit, because the flag name is an API the moment it ships — renaming later breaks user shell history and scripts. `-multisig-members` reads like a positional input selector ("list only multisig-members") rather than an output toggle. The existing flag style in `gnokey` is verb-prefixed where ambiguous (`-insecure-password-stdin`, `-no-backup`, `-multisig-threshold`). Fix: rename to `-show-multisig-members` (or `-multisig-verbose`) before merge; cheap now, expensive later. If the maintainers prefer the short form, at minimum extend the help text to `"show multisig member public keys instead of the multisig bech32 public key (no effect on non-multisig keys)"` so users do not wonder why it appears global.
  </details>

## Nits

- [`tm2/pkg/crypto/multisig/threshold_pubkey.go:36-38`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/multisig/threshold_pubkey.go#L36-L38) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/multisig/threshold_pubkey.go#L36-L38) — empty-case guard is unreachable via `NewPubKeyMultisigThreshold` (constructor panics on `k <= 0` and `len(pubkeys) < k`); only reachable via direct struct literal or amino-decoded malformed bytes. Worth a one-line code comment so future readers do not delete it thinking it is dead code.
- [`tm2/pkg/crypto/keys/client/list_test.go:155`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/client/list_test.go#L155) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/client/list_test.go#L155) — `assert.Contains(t, output, "\n  "+pk.String()+"\n")` is brittle to leading-/trailing-whitespace tweaks in `PubKeyMultisigThreshold.String()`. The intent is "each member key appears on its own indented line"; a regex per member, or splitting output into lines and asserting membership, would survive harmless formatting changes.

## Missing Tests

- **[behaviour parity across key types]** [`tm2/pkg/crypto/keys/client/list_test.go:61-159`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/client/list_test.go#L61-L159) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/client/list_test.go#L61-L159) — both new tests only exercise multisig entries; neither asserts that the flag is a no-op for `secp256k1`/`ed25519`/offline keys in the same keybase.
  <details><summary>details</summary>

  A mixed keybase (local + multi + offline) would catch regressions where someone refactors the type-switch and accidentally bech32-encodes a non-multi entry's pubkey twice, or where `-multisig-members` starts affecting non-multi rows. One small additional test that creates one of each type and asserts: (a) default — multi entry shows `gpub1...` once, locals show their normal `gpub1...`; (b) with flag — locals identical, multi shows member list. Adversarial test not written; the gap is shape-correctness, not security.
  </details>

- **[amino-roundtrip empty multisig]** [`tm2/pkg/crypto/multisig/threshold_pubkey_test.go:190-198`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/multisig/threshold_pubkey_test.go#L190-L198) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/multisig/threshold_pubkey_test.go#L190-L198) — existing test constructs an empty `PubKeyMultisigThreshold` via struct literal; no test covers what `String()` returns for a `PubKeyMultisigThreshold` decoded from a malformed amino payload with `k > 0` but zero pubkeys. Same code path, but documents the defensive guard's purpose. Low priority.

## Suggestions

- [`tm2/pkg/crypto/keys/client/list.go:63-78`](https://github.com/gnolang/gno/blob/250e46eeb/tm2/pkg/crypto/keys/client/list.go#L63-L78) · [↗](../../../../../.worktrees/gno-review-5134/tm2/pkg/crypto/keys/client/list.go#L63-L78) — once `printNewInfo` is aligned with this format, factor the field-formatting into a tiny helper (`formatInfoLine(info, prefix, showMembers) string`) so the two callers cannot drift again.
  <details><summary>details</summary>

  Single source of truth for the per-entry line. Out of scope for this PR if rushed, but the inconsistency that already exists is what motivates the warning above — and a helper would have prevented it. Leave a TODO if not done here.
  </details>

## Questions for Author

- Why reorder `addr / pub / path` → `addr / path / pub`? The PR description shows the new order without rationale. Long-bech32-last is a sensible reason (terminal wrapping), but if the rationale is different the commit message should say so — and either way, the same order should propagate to `printNewInfo`.
- Was `-multisig-members` considered as `-show-multisig-members`? The latter scans as an output toggle; the former scans as a filter ("list multisig members"). Naming is cheap to change pre-merge, expensive post-merge.
