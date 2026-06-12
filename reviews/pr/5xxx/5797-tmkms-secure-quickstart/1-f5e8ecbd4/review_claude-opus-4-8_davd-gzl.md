# PR #5797: docs(validators/tmkms): secure-setup quickstart for tmkms-backed validators

URL: https://github.com/gnolang/gno/pull/5797
Author: D4ryl00 | Base: feat/tmkms-compat/03-listener-integration | Files: 2 | +1096 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: f5e8ecbd4 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5797 f5e8ecbd4`

**TL;DR:** A new, ~1090-line operator guide (`docs/validators/tmkms-quickstart.md`) that walks you from an empty machine to a gnoland validator whose consensus key is held by tmkms, plus a 6-line cross-link from the existing `tmkms.md` reference. It is procedural and security-first: dedicated `gnoland` user, `0600` secrets, transport choice (Unix socket vs firewalled TCP), and three signer backends (YubiHSM recommended, softsign for labs, Ledger for advanced).

**Verdict: APPROVE** — every load-bearing command, flag, file format, and code-behavior claim I checked verifies against the base branch; no blocking issues. The CI `mod-tidy` red is unrelated (no go.mod/go.sum touched). Two Nits and a few Suggestions only.

## Summary
Docs-only PR stacked on the tmkms listener-integration branch (not master). It adds a hands-on quickstart that complements the architecture/threat-model reference already in `tmkms.md`. The guide bakes in the OS- and config-level hardening that closes the operational footguns from the tmkms 0.15.0 security review (world-readable secrets, secret-leaking convenience commands, missing permission checks, fail-open allowlist), and flags the one residual code risk (no `fsync` on the state file before signing). Every command block, every gnoland/gnogenesis flag, both Go helper tools, and the example pubkey→address pair check out against the code on the base branch.

## Verification

I ran these against the worktree at `f5e8ecbd4` (base branch `feat/tmkms-compat/03-listener-integration`):

| Doc claim | Source | Result |
|---|---|---|
| `nodeid-hex.go` reads `priv_key` as a bare base64 string | `node_key.json` is amino-marshaled; `priv_key` serializes as a plain base64 string (not `@type`/`value`) | matches — helper is correct |
| C.1 python reads `["priv_key"]["value"]` | `priv_validator_key.json` serializes `priv_key` as a nested `{@type,value}` object | matches — different shape from node_key.json, doc gets both right |
| peer ID = `SHA256(pubkey)[:20]` (hex) | `PubKeyEd25519.Address()` = `tmhash.SumTruncated(pubKey[:])` (SHA256 truncated to 20 bytes) | matches |
| `pkconv.go` imports + example output | ran the helper on the doc's example hex | prints exactly `g1qmptf8uxdg6l0rh07jwvur0kk8my9vrdf5qtp4` and the `gpub1pggj7ard9eg…` used throughout B.3/D.4/genesis |
| empty `allowed_kms_pubkeys` rejected at startup | `errEmptyTmkmsAllowedPubkeys` in `TmkmsListenerConfig.ValidateBasic` | matches |
| only `protocol_version = "v0.34"` accepted | `upstream.ProtocolVersion = "v0.34"` const; `errUnsupportedProtocolVersion` | matches |
| `validator add` checks pubkey hashes to address | `errPublicKeyAddressMismatch` in `validator_add.go` | matches |
| `validator add` flags `-pub-key -name -power -address -genesis-path` | flag registrations in `validator.go` / `validator_add.go` | matches |
| `gnoland start` flags `-lazy -skip-genesis-sig-verification -genesis -chainid -data-dir`; `config -config-path` | flag registrations in `start.go` / `config.go` | matches |
| `secrets init -data-dir <dir>/secrets` lands where `gnoland -data-dir <dir>` reads | `DefaultSecretsDir = "secrets"`, `DefaultConfigDir = "config"` | matches |
| `wait_for_connection_timeout` default 60s | `DefaultTmkmsListenerConfig` sets `60 * time.Second` | matches |
| `tmkms_integration` test at `./tm2/pkg/bft/privval/upstream/...` | `TestTmkmsIntegration_FullSigningFlow`, build tag `tmkms_integration` | matches |
| GNOROOT troubleshooting message | `ErrUnableToGuessGnoRoot` = "gno was unable to determine GNOROOT…" | matches |
| all internal `#anchor` links resolve | derived GitHub slugs from headings | all 12 resolve |

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
- `docs/validators/tmkms-quickstart.md:222-231` — A.3 presents the UDS `0600` socket as a hard auth boundary, but gnoland's chmod is best-effort: on a filesystem that doesn't honor chmod on a socket the call fails and gnoland only logs a warning, leaving default (potentially world-writable) perms. Harmless here because A.2/A.3 already lock the parent dir `/run/gnoland` to `700 gnoland`, which is the real guard; worth one half-sentence so a reader who relocates the socket doesn't lean on a perm that can silently not apply. Anchor: [`config.go:204-207`](https://github.com/gnolang/gno/blob/f5e8ecbd4/tm2/pkg/bft/privval/config.go#L204-L207) · [↗](../../../../../.worktrees/gno-review-5797/tm2/pkg/bft/privval/config.go#L204).
- `docs/validators/tmkms-quickstart.md:144-145` — the cleanup line `sudo rm /usr/local/bin/{nodeid-hex,tmkms-identity-keygen}` uses brace expansion, which is bash/zsh-only; a `sh`/dash reader pasting it deletes nothing or errors. Minor since the rest of the guide is `#!/bin/sh`-agnostic and this is an optional teardown step.

## Missing Tests
None — docs-only PR. The gnoland side of every documented path is already exercised by the repo's `tmkms_integration` test (softsign + Ledger), which the guide points to in Part E.

## Suggestions
- `docs/validators/tmkms-quickstart.md:438-457` — B.3's "joining an existing chain" branch tells the operator to "submit the `gpub1…` / `g1…` to the chain's validator-onboarding path" but doesn't name a concrete mechanism (governance proposal, faucet, onboarding realm). A reader on the production path is left without the next action. A one-line pointer to wherever gno.land documents validator onboarding would close the loop; if that doc doesn't exist yet, saying so explicitly is still better than an unqualified "submit it".
  <details><summary>details</summary>

  The softsign/Ledger lab paths are fully runnable end-to-end, but the recommended production path (YubiHSM, join existing chain) ends at "hand your pubkey to the onboarding path" with no link. Not a blocker because the genesis-bootstrap branch right below it is complete and the identity derivation is identical, but it's the one place the otherwise copy-paste guide goes abstract.
  </details>
- `docs/validators/tmkms-quickstart.md:475-493` — the `tmkms.toml` heredoc is written with `sudo tee … <<TOML` (unquoted delimiter) so `${CHAIN_ID}` / `${TMKMS_ADDR}` expand, which is intended. Worth a one-line note that the shell, not tmkms, does this expansion, so the file must be regenerated if `$CHAIN_ID` changes later — a re-runner who only `export`s a new value and restarts tmkms (without re-running the `tee`) gets a stale config and a confusing chain_id-mismatch stall (already a Part E troubleshooting row, but the cause is non-obvious).

## Open questions
- The guide is the operator-facing companion to the listener-integration feature branch it targets; once that branch merges to master the relative cross-links (`tmkms.md` ↔ `tmkms-quickstart.md`) and the Part E test path stay valid as-is. No action needed in this PR — noting only that the doc's correctness is pinned to that branch's API surface, which I verified against, not master.
