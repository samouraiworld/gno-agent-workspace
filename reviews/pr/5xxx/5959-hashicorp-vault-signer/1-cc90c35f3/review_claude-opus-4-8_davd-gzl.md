# PR [#5959](https://github.com/gnolang/gno/pull/5959): privval: add HashiCorp Vault backed signer

URL: https://github.com/gnolang/gno/pull/5959
Author: ygd58 | Base: master | Files: 6 | +470 -8
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: cc90c35f3 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5959 cc90c35f3`

**TL;DR:** A validator node normally keeps its signing key in a file on disk. This PR lets the node read that key out of HashiCorp Vault at startup instead, and optionally generate one and store it there on first run. The key still lives in the node's memory and the node still signs locally; Vault only replaces the file as the place the key is kept.

**Verdict: REQUEST CHANGES** — the package does not compile because the Vault SDK is missing from `go.mod`, plus a Vault token that lands in a world-readable config file and an unconditional key write that can replace an existing validator identity (1 Critical, 3 Warnings, 1 Missing test, 3 Nits, 1 Suggestion).

## Summary

`vault.Signer` mirrors `local.LocalSigner` one-for-one: fetch a `FileKey` once at construction, keep it in memory, sign with `key.PrivKey.Sign`. The new `local.ParseFileKey` export is what makes the mirror possible, so both key sources share one encoding and one validation path. Wiring is a fourth branch in `PrivValidatorConfig`, ordered after the remote signer and before the local file.

The blocker is mechanical: `github.com/hashicorp/vault/api` is imported but never added to `go.mod`, so `go build ./tm2/pkg/bft/privval/...` fails. None of the repo's build or test jobs ran on this PR (the bot gates them behind an initial approval), so CI never surfaced it. Adding the module pulls in 16 new lines of `require`, 11 of them new `hashicorp/*` modules.

Beyond that, two problems are specific to using a network secret store as a key vault. The `token` field is TOML-serialized into `config.toml`, which the node writes at mode 0644, while the key that token unlocks is written at 0600 in local mode. And `createAndStoreKey` writes with a plain `Put`, no check-and-set, so a create can silently replace a validator identity that is already at that path.

## Fix

Before this PR, `LoadFileKey` unmarshalled and validated inline. It now delegates both to a new exported `ParseFileKey`, and the `vault` package calls that same function on the string pulled out of the KV v2 secret's `priv_validator_key_json` field. Selection happens in [`tm2/pkg/bft/privval/config.go:151-154`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L151-L154) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L151-L154), with `SecretPath != ""` as the single enable switch. The load-bearing constraint is that Vault is a key store, not a signing oracle: the private key leaves Vault and sits in process memory, so the trust model is the local signer's, not tmkms's.

## Critical (must fix)

- **[branch does not compile]** `tm2/pkg/bft/privval/signer/vault/client.go:7` — the Vault SDK is imported but absent from `go.mod`, so the whole module fails to build.
  <details><summary>details</summary>

  [`client.go:7`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/client.go#L7) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/client.go#L7) imports `github.com/hashicorp/vault/api`, and the diff touches neither `go.mod` nor `go.sum`. `go build ./tm2/pkg/bft/privval/...` at cc90c35f3 reports `no required module provides package github.com/hashicorp/vault/api`. The repo's build and test workflows never ran on this PR, so nothing caught it. Running `go get github.com/hashicorp/vault/api` resolves v1.23.0 and adds 16 `require` lines, after which every package under `tm2/pkg/bft/privval/` builds and its tests pass. Fix: commit the `go.mod` and `go.sum` updates. [repro](comment_claude-opus-4-8.md)
  </details>

## Warnings (should fix)

- **[key-equivalent credential on disk]** `tm2/pkg/bft/privval/signer/vault/config.go:16` — the Vault token is written into `config.toml` at mode 0644, while the key it unlocks is written at 0600.
  <details><summary>details</summary>

  [`config.go:16`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/config.go#L16) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L16) carries a `toml:"token"` tag, so `WriteConfigFile` serializes it. That file is written with [`osm.WriteFile(configFilePath, configRaw, 0o644)`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/config/toml.go#L61) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/config/toml.go#L61), whereas the local validator key is written with [`0o600`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L65) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/local/key.go#L65). A token with read access to the secret is equivalent to the key for confidentiality, so an operator who fills the field trades a 0600 key file for a world-readable credential that fetches it. Setting `Token = "hvs.SUPERSECRET"` and calling `WriteConfigFile` produces `token = "hvs.SUPERSECRET"` in a `-rw-r--r--` file. Fix: drop the field and rely on `VAULT_TOKEN` / `~/.vault-token`, which the doc comment already points at. [repro](comment_claude-opus-4-8.md)
  </details>

- **[silent validator identity replacement]** `tm2/pkg/bft/privval/signer/vault/signer.go:135` — the create path writes with a plain `Put`, so it overwrites a validator key already at that path instead of failing.
  <details><summary>details</summary>

  [`signer.go:135`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/signer.go#L135) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/signer.go#L135) calls `client.Put(ctx, cfg.SecretPath, data)` with no `KVOption`. `KVv2.Put` only [attaches an options block when one is passed](https://github.com/hashicorp/vault/blob/api/v1.23.0/api/kv_v2.go#L230-L237), so the write is unconditional and Vault appends a new version. Two nodes started with `create_if_missing` against the same path both read "not found", both write, and the loser signs with a key that is no longer in Vault. `WithCheckAndSet(0)` is exactly the guard: a [write carrying `cas=0` is allowed only when the key does not exist](https://github.com/hashicorp/vault/blob/api/v1.23.0/api/kv_v2.go#L95). The `kvAPI` interface at [`client.go:15`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/client.go#L15) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/client.go#L15) already carries the variadic option, it is just never used. Fix: make the create conditional on the path being empty. Test: [`vault_cas_test.go`](tests/vault_cas_test.go) fails at cc90c35f3 and passes with the option added. [repro](comment_claude-opus-4-8.md)
  </details>

- **[soft-deleted key becomes a new identity]** `tm2/pkg/bft/privval/signer/vault/signer.go:92-97` — a soft-deleted secret plus `create_if_missing` mints a fresh validator key over a recoverable one.
  <details><summary>details</summary>

  [`signer.go:92-97`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/signer.go#L92-L97) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/signer.go#L92-L97) treats a nil `Data` map as "not found". `KVv2.Get` [documents that shape for a deleted latest version](https://github.com/hashicorp/vault/blob/api/v1.23.0/api/kv_v2.go#L112-L114), so the read is correct, but the branch it feeds is not: with `create_if_missing` the node generates a new key and writes it as a newer version. The old key stays in Vault history yet is no longer what the node reads, so `vault kv undelete` no longer recovers the validator, and the node comes up under a pubkey the validator set does not know. Fix: a deleted version is a state an operator has to resolve, not one to overwrite. The check-and-set guard above also covers this, since `cas=0` rejects a path that already has versions.
  </details>

## Nits

- **[stale doc comment]** `tm2/pkg/bft/privval/config.go:23` — the struct doc still says "At most one of RemoteSigner or TmkmsListener may be enabled" after Vault became a third exclusive mode. [`config.go:23`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L23) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L23)

- **[misleading error text]** `tm2/pkg/bft/privval/signer/local/key.go:83` — key-validation failures are now reported as unmarshal failures, and the unmarshal message is doubled.
  <details><summary>details</summary>

  [`key.go:83`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L83) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/local/key.go#L83) wraps everything `ParseFileKey` returns, including the `validate()` errors that master returned bare. Loading a key whose address does not match its pubkey now yields `unable to unmarshal FileKey from <path>: address does not match public key`; a malformed file yields `unable to unmarshal FileKey from <path>: unable to unmarshal FileKey: invalid character 'n'`, since [`key.go:96`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L96) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/local/key.go#L96) adds the same prefix. Confirmed behaviorally against both file shapes. `errors.Is` still matches, so only the operator-facing text regressed. Fix: wrap the file path once, at the point that reads the file.
  </details>

- **[unused exported helper]** `tm2/pkg/bft/privval/signer/vault/config.go:47` — `TestConfig` has no caller in the tree. [`config.go:47`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/config.go#L47) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L47) returns `DefaultConfig()` verbatim, and `TestPrivValidatorConfig` reaches Vault through `DefaultPrivValidatorConfig` instead. Grepped `tm2/` for `vault.TestConfig`: no hits.

- **[secret layout undiscoverable]** `tm2/pkg/bft/privval/signer/vault/config.go:25` — nothing outside the source says the secret must carry a `priv_validator_key_json` field. The name is a private constant at [`signer.go:17`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/signer.go#L17) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/signer.go#L17), the `secret_path` comment at [`config.go:25`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/config.go#L25) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L25) does not mention it, and there is no `docs/validators/` page for this mode the way [`docs/validators/tmkms.md`](https://github.com/gnolang/gno/blob/cc90c35f3/docs/validators/tmkms.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-5959/docs/validators/tmkms.md#L1) covers the listener mode.

## Missing Tests

- **[untested config guards]** `tm2/pkg/bft/privval/config.go:110-126` — the two new validation errors have no test, though the errors they sit beside do.
  <details><summary>details</summary>

  [`config.go:110-126`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L110-L126) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L110-L126) adds `errNilVaultConfig` and `errMultipleSignerSourcesSet`, and [`config.go:184-185`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L184-L185) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L184-L185) repeats the second one in `NewPrivValidatorFromConfig`. `config_test.go` was not touched, yet it already covers the adjacent cases: [`errNilRemoteSignerConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config_test.go#L62) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config_test.go#L62) and [`errBothExternalSignersEnabled`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config_test.go#L272) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config_test.go#L272). Nothing exercises the vault branch of `NewSignerFromConfig` either, so a future edit to the if-ordering there would not be caught. Fix: mirror the existing cases for the vault pairs.
  </details>

## Suggestions

- **[four PRs, one unenforced switch]** `tm2/pkg/bft/privval/config.go:124-126` — each sibling signer PR adds its own pointer field and its own pairwise exclusion check, so any two of them merged together leave the cross-pair unguarded.
  <details><summary>details</summary>

  [`config.go:124-126`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L124-L126) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L124-L126) checks Vault against `RemoteSigner` and `TmkmsListener` only. [#5956](https://github.com/gnolang/gno/pull/5956), [#5958](https://github.com/gnolang/gno/pull/5958), and [#5980](https://github.com/gnolang/gno/pull/5980) each modify this same file with the same shape for AWS Secrets Manager, GCP Secret Manager, and YubiHSM2. Merge two of them and a config enabling both Vault and AWS passes `ValidateBasic`, then [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L140-L157) · [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L140-L157) silently picks whichever branch comes first. Each PR also declares its own `errMultipleSignerSourcesSet`, which collides outright. A single list of enabled backends, rejected when its length exceeds one, scales; a hand-written matrix does not.
  </details>

## Verified

- The branch does not build: `go build ./tm2/pkg/bft/privval/...` at cc90c35f3 fails on the missing `hashicorp/vault/api` module. Adding it with `go get` makes every package under `tm2/pkg/bft/privval/` build and its tests pass, so the missing `go.mod` entry is the only build defect.
- The Vault token reaches disk in the clear: writing a config with `Token` set produces `token = "hvs.SUPERSECRET"` inside a `-rw-r--r--` `config.toml`, against `0o600` for the local key file.
- The create path replaces an existing key: [`vault_cas_test.go`](tests/vault_cas_test.go) drives `createAndStoreKey` against a mock honouring Vault's `cas` semantics, fails at cc90c35f3, and passes once the write carries `WithCheckAndSet(0)`. The package's own tests stay green with the option added.
- `LoadFileKey` error text regressed: an address-mismatch key now reports `unable to unmarshal FileKey from <path>: address does not match public key`, and a malformed file doubles the unmarshal prefix.
- Green at cc90c35f3 with the module added: `tm2/pkg/bft/privval/...` (all packages), `gno.land/cmd/gnoland -run TestConfig`.

## Open questions

- The `check` job is red because the PR title `privval: ...` is not one of the allowed conventional-commit types. Not posted: contribution-policy compliance, visible in the CI log.
- `gnoland secrets get` and `secrets verify` still read the local `priv_validator_key.json` regardless of the vault mode, so an operator on Vault has no CLI to inspect the live key. Not posted: pre-existing gap shared with the remote-signer mode, out of scope here.
- `Signer.Close` does not zero the key material, matching `LocalSigner.Close`. Not posted: no delta against the mode this one mirrors.
- `vault.Config.ValidateBasic` always returns nil. Not posted: the comment explains why there are no cross-field constraints, and the shape keeps the interface uniform with the other signer configs.
