# PR [#5980](https://github.com/gnolang/gno/pull/5980): privval: add YubiHSM2 backed signer

URL: https://github.com/gnolang/gno/pull/5980
Author: ygd58 | Base: master | Files: 6 | +425 -8
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: c9aaed5c8 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5980 c9aaed5c8`

**TL;DR:** A validator's signing key normally sits in a plaintext file on the node, or is held by a separate signing service. This PR adds one more option: keep the key inside a YubiHSM2 hardware security module and ask the device to sign each vote, so the key itself never reaches the machine.

**Verdict: REQUEST CHANGES** — the branch does not compile (the new dependency is missing from `go.mod`), the documented `connector_url` value is not the form the connector library accepts, and the HSM password lands in a world-readable `config.toml`; separately, the linked issue puts this behind gnokms rather than in the node's privval, which is a maintainer call (2 Critical, 5 Warnings, 1 Missing test, 4 Nits, 3 Suggestions).

## Summary

The node's [privval](https://github.com/samouraiworld/gno-agent-workspace/blob/main/docs/glossary.md) subsystem currently picks one of three signer modes from config: a local key file, a gnokms remote signer it dials, or a tmkms listener. This PR adds a fourth, `yubihsm`, which opens an authenticated SCP03 session to a YubiHSM2 through a `yubihsm-connector` HTTP service, caches the device's Ed25519 public key at startup, and forwards each set of sign-bytes to the device. The new mode is wrapped by the existing [sign state](https://github.com/samouraiworld/gno-agent-workspace/blob/main/docs/glossary.md) gate, so double-sign protection is unchanged. It also exports [`ParseFileKey`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L92) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L92) in the local signer, which nothing in this PR calls.

The branch as pushed does not build: `github.com/certusone/yubihsm-go` is imported at [`client.go:6-8`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/client.go#L6-L8) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/client.go#L6-L8) but absent from `go.mod` and `go.sum`. No build job ran on the PR, so CI is not showing it.

## Glossary

- privval: the validator's private-key signing subsystem (`tm2/pkg/bft/privval`), selecting one signer mode from config.
- sign state: the last-signed height/round/step record (`priv_validator_state.json`) consulted before every vote so a node refuses to sign twice at the same height and round.

## Fix

`PrivValidatorConfig` gains a `YubiHSM *yubihsm.Config` section at [`config.go:42`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L42) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L42), validated and mutually excluded against the two existing external-signer modes at [`config.go:110-126`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L110-L126) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L110-L126) and again at [`config.go:182-186`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L182-L186) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L182-L186). Enablement is keyed solely on a non-empty `connector_url`, matching how `remote_signer` and `tmkms_listener` already work. The signer itself splits construction in two so tests can drive it: [`newSession`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/client.go#L24) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/client.go#L24) builds the real session manager, and [`newSigner`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L93) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L93) takes the narrow `hsmAPI` interface, which the package tests satisfy with a mock.

## Critical (must fix)

- **[branch does not compile]** `tm2/pkg/bft/privval/signer/yubihsm/client.go:6-8` — the new dependency is imported but never added to `go.mod` or `go.sum`, so the whole module fails to build.
  <details><summary>details</summary>

  `go build ./tm2/pkg/bft/privval/...` on a clean checkout of c9aaed5c8 reports `no required module provides package github.com/certusone/yubihsm-go` for all three imported paths at [`client.go:6-8`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/client.go#L6-L8) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/client.go#L6-L8). Neither `go.mod` nor `go.sum` mentions `certusone`. The PR's checks are all green-or-skipped because the build and test workflows are gated behind maintainer approval and never ran. Adding it also pulls in `github.com/enceve/crypto` transitively, which is worth reviewing alongside it. Fix: run `go get github.com/certusone/yubihsm-go@v0.3.0` and commit both files.
  </details>

- **[documented setting cannot work]** `tm2/pkg/bft/privval/signer/yubihsm/config.go:9-12` — `connector_url` is documented as `http://127.0.0.1:12345`, but the connector library prepends its own scheme, so that value produces `http://http//127.0.0.1:12345/connector/api` and the node cannot start.
  <details><summary>details</summary>

  [`HTTPConnector.Request`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/connector/http.go#L39) builds its endpoint as `"http://" + c.URL + "/connector/api"`, so `NewHTTPConnector` at [`client.go:25`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/client.go#L25) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/client.go#L25) wants a bare `host:port`. The field doc at [`config.go:9-12`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/config.go#L9-L12) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L9-L12) and the `comment:` tag rendered into every generated `config.toml` both give a full URL instead, and [`ValidateBasic`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/config.go#L54-L68) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L54-L68) does not check the form, so the mistake only surfaces at node start. Booting `gnoland` with the documented value fails with `Post "http://http//127.0.0.1:12399/connector/api": dial tcp: lookup http: no such host`; with `127.0.0.1:12399` it reaches the port and reports `connection refused` (see [repro](comment_claude-opus-4-8.md)). The scheme being hardcoded also means `https://` is never reachable, so a remote connector cannot be addressed over TLS. Fix: document and validate a bare `host:port`, and reject a value containing `://` in `ValidateBasic`; the case table is in [`tests/connector_url_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5980-yubihsm2-backed-signer/1-c9aaed5c8/tests/connector_url_test.go) · [↗](tests/connector_url_test.go).
  </details>

## Warnings (should fix)

- **[secret stored where secrets never went before]** `tm2/pkg/bft/privval/signer/yubihsm/config.go:18-19` — the HSM authentication password is stored in `config.toml`, which is written mode 0644 and printed in full by `gnoland config get`, while the validator key file it replaces is 0600.
  <details><summary>details</summary>

  `Password` at [`config.go:18-19`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/config.go#L18-L19) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L18-L19) is the first and only credential in the node config tree; a grep for a `toml:"password"` tag across `tm2/`, `gno.land/` and `contribs/` returns this line alone. [`WriteConfigFile`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/config/toml.go#L61) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/config/toml.go#L61) writes `config.toml` at 0644, whereas the local signer's key file is written [at 0600](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L65) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L65). `gnoland config get consensus.priv_validator` dumps the whole section as JSON, password included, which puts it into shell history, logs, and support pastes. Anyone who can read `config.toml` and reach the connector can sign. Fix: source the password from a file path or an environment variable instead of storing the value in the config.
  </details>

- **[live device session leaks on a failed start]** `tm2/pkg/bft/privval/signer/yubihsm/signer.go:82-87` — when `newSigner` fails, the session opened just above it is never destroyed, leaving an authenticated session and its keepalive goroutine alive.
  <details><summary>details</summary>

  [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L82-L87) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L82-L87) calls `newSession`, then returns `newSigner(session, cfg.KeyID)` directly. `newSigner` has four failure exits ([`signer.go:96`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L96) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L96), [`101`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L101) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L101), [`106`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L106) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L106), [`110`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L110) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L110)), and on none of them does anything call `Destroy`. `NewSessionManager` starts a `pingRoutine` goroutine and holds one of the device's limited concurrent session slots until the process exits. A wrong `key_id` in config is enough to hit this, and an operator retrying the start burns another slot each time. Fix: destroy the session when `newSigner` returns an error.
  </details>

- **[node can sign with the wrong key and say nothing]** `tm2/pkg/bft/privval/signer/yubihsm/config.go:41-44` — a `yubihsm` section with the auth key, key id and password filled in but `connector_url` still empty is treated as disabled, and the node starts on the local key file with no warning.
  <details><summary>details</summary>

  [`IsEnabled`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/config.go#L41-L44) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L41-L44) keys only on `ConnectorURL`, and [`ValidateBasic`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/config.go#L54-L57) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/config.go#L54-L57) returns early for a disabled config, so a half-filled section is accepted. The natural way to reach that state is `gnoland config set`: setting `connector_url` first is rejected with `yubihsm signer: auth_key_id cannot be zero when enabled`, so the value is never written, while the three follow-up sets succeed. The operator ends up with `auth_key_id`, `key_id` and `password` populated, `connector_url` empty, and a node that silently falls through to `LoadOrMakeLocalSigner` at [`config.go:157`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L157) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L157). Fix: reject a partially-filled disabled section, or log at startup when a `yubihsm` section carries values but is not enabled.
  </details>

- **[archived crypto in the signing path]** `tm2/pkg/bft/privval/signer/yubihsm/client.go:6-8` — `yubihsm-go` had its last release in January 2023 against Go 1.14, and it pulls in `github.com/enceve/crypto` from 2016 for the CMAC that authenticates every command sent to the device.
  <details><summary>details</summary>

  `go list -m -versions github.com/certusone/yubihsm-go` shows `v0.3.0` as the newest, published 2023-01-11, with `go 1.14` in its `go.mod`. Its only non-stdlib crypto dependency is [`github.com/enceve/crypto/cmac`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/securechannel/channel.go#L12), pinned to a 2016 commit and used at [`channel.go:318`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/securechannel/channel.go#L318) to MAC every SCP03 message. Adding it puts unmaintained third-party crypto on the path that every validator vote crosses. The linked issue [#3236](https://github.com/gnolang/gno/issues/3236) already flags the library's dormancy as an open question rather than a settled choice. Fix: state in the PR whether vendoring, forking, or replacing the SCP03 layer is the intended answer, so maintainers can decide before the dependency lands.
  </details>

- **[feature may belong in a different component]** `tm2/pkg/bft/privval/config.go:38-42` — the linked issue and its master issue place the HSM behind a gnokms backend, whereas this wires the device into the node's own privval.
  <details><summary>details</summary>

  Issue [#3236](https://github.com/gnolang/gno/issues/3236) is titled "Add YubiHSM2 device support as a TM2 remote signer" and sits under [#3230](https://github.com/gnolang/gno/issues/3230), whose stated goal is that validator secrets "no longer need to be duplicated over machines" and live "completely decoupled from the TM2 blockchain client". The [gnokms README](https://github.com/gnolang/gno/blob/c9aaed5c8/contribs/gnokms/README.md?plain=1#L5) · [↗](../../../../../.worktrees/gno-review-5980/contribs/gnokms/README.md#L5) says it "aims to provide several backends, including a local gnokey instance, a remote HSM, or a cloud-based KMS service", and its flowchart carries an explicit "HSM backend" box. The section added at [`config.go:38-42`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L38-L42) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L38-L42) instead keeps the connector address and the HSM password on the validator machine. Both placements are defensible; the choice is not stated in the PR. Fix: say why the node-side placement was chosen over a gnokms backend, since it changes where the credential lives.
  </details>

## Missing Tests

- **[new rules can silently regress]** `tm2/pkg/bft/privval/config.go:110-126` — the `yubihsm` nil-config and mutual-exclusion branches have no test, though the matching `remote_signer` and `tmkms_listener` rules do.
  <details><summary>details</summary>

  `config_test.go` covers [`errNilRemoteSignerConfig`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config_test.go#L62) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config_test.go#L62) and [`errBothExternalSignersEnabled`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config_test.go#L272) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config_test.go#L272), but nothing exercises [`errNilYubiHSMConfig`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L111-L113) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L111-L113) or either [`errMultipleSignerSourcesSet`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/config.go#L124-L125) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/config.go#L124-L125) branch. The new package's own tests stop at the `yubihsm` package boundary and never reach `PrivValidatorConfig`. The mutual-exclusion check is the only thing standing between an operator and two live signer sources. Fix: add the four `ValidateBasic` cases and the two `NewPrivValidatorFromConfig` cases; they are written and green at c9aaed5c8 in [`tests/privval_config_yubihsm_test.go`](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5980-yubihsm2-backed-signer/1-c9aaed5c8/tests/privval_config_yubihsm_test.go) · [↗](tests/privval_config_yubihsm_test.go).
  </details>

## Nits

- **[comment describes code that isn't there]** `tm2/pkg/bft/privval/signer/yubihsm/signer.go:15` — "Unlike the file/AWS/GCP/Vault-backed signers" names three signers the repository does not have; the existing modes are the local file, the gnokms remote client, and the tmkms listener.
- **[stale invariant in a doc comment]** `tm2/pkg/bft/privval/config.go:23` — the struct comment still says "At most one of RemoteSigner or TmkmsListener may be enabled" after the PR made it three mutually exclusive modes.
- **[error text now says the wrong thing]** `tm2/pkg/bft/privval/signer/local/key.go:83` — `LoadFileKey` wraps everything `ParseFileKey` returns as "unable to unmarshal", so a malformed file now reports `unable to unmarshal FileKey from <path>: unable to unmarshal FileKey: EOF` and a validation failure is labelled an unmarshal failure. Confirmed behaviorally by loading a truncated key file.
- **[operators have nothing to follow]** `docs/validators/tmkms.md:1` — the tmkms mode ships an operator guide; this one has no equivalent, and the device-side setup (auth key, `sign-eddsa` capability, connector service) is nowhere written down.

## Suggestions

- **[wrong key type gives a confusing error]** `tm2/pkg/bft/privval/signer/yubihsm/signer.go:104-111` — `GetPubKeyResponse` carries an `Algorithm` field that is discarded, so pointing `key_id` at a non-Ed25519 object reports "unexpected public key length" rather than naming the real problem.
  <details><summary>details</summary>

  [`parseGetPubKeyResponse`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/commands/response.go#L345-L353) splits the payload into `Algorithm` and `KeyData`, and the signer at [`signer.go:104-111`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L104-L111) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L104-L111) reads only the length of the latter. The length check does catch the realistic wrong-key cases, so this is diagnostics rather than safety.
  </details>

- **[unchecked device output]** `tm2/pkg/bft/privval/signer/yubihsm/signer.go:50` — `Sign` returns whatever bytes the device sent without checking the Ed25519 signature length, so a malformed response surfaces later as a rejected vote instead of a signer error.
  <details><summary>details</summary>

  [`parseSignDataEddsaResponse`](https://github.com/certusone/yubihsm-go/blob/v0.3.0/commands/response.go#L279-L283) copies the payload straight into `Signature` with no length check of its own, and [`signer.go:50`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L50) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/yubihsm/signer.go#L50) passes it through. The public key gets a length check at construction; the signature deserves the same.
  </details>

- **[new exported API with no caller]** `tm2/pkg/bft/privval/signer/local/key.go:89-92` — `ParseFileKey` is exported for "alternative key sources (e.g. a secrets manager)", and nothing in this PR or the tree calls it.
  <details><summary>details</summary>

  The only caller of [`ParseFileKey`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L92) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L92) is [`LoadFileKey`](https://github.com/gnolang/gno/blob/c9aaed5c8/tm2/pkg/bft/privval/signer/local/key.go#L81) · [↗](../../../../../.worktrees/gno-review-5980/tm2/pkg/bft/privval/signer/local/key.go#L81) in the same file; the yubihsm signer never reads a `FileKey`. Splitting the function is fine, exporting it for a hypothetical consumer widens the package API for nothing.
  </details>

## Verified

- Booting `gnoland` from this worktree with `connector_url = "http://127.0.0.1:12399"`, the value the field doc gives, fails with `Post "http://http//127.0.0.1:12399/connector/api": dial tcp: lookup http: no such host`; changing it to `127.0.0.1:12399` reaches the port and fails with `connection refused`. Confirms the connector library prepends its own scheme.
- `gnoland config get consensus.priv_validator` on a config with the yubihsm password set prints `"password": "sup3rs3cret"` in the JSON dump, and the generated `config.toml` is mode 0644.
- `gnoland config set consensus.priv_validator.yubihsm.connector_url <addr>` on a fresh config is rejected with `unable to validate config, yubihsm signer: auth_key_id cannot be zero when enabled` and the value is not written; the same four sets in reverse order all succeed.
- `go build ./tm2/pkg/bft/privval/...` fails at c9aaed5c8 on the unmodified tree; after `go get github.com/certusone/yubihsm-go@v0.3.0` it builds, and `go test ./tm2/pkg/bft/privval/... ./tm2/pkg/bft/config/... ./gno.land/cmd/gnoland/...` is green.
- [`tests/privval_config_yubihsm_test.go`](tests/privval_config_yubihsm_test.go) passes at c9aaed5c8; [`tests/connector_url_test.go`](tests/connector_url_test.go) fails at c9aaed5c8 on both scheme-prefixed cases and passes on both bare `host:port` cases.

## Existing threads

None. No review comments and no human issue comments on the PR.

## Invariant catalog

Not walked: the diff is Go-only node infrastructure under `tm2/pkg/bft/privval`, with no GnoVM, stdlib, or `.gno` change, so the catalog's gating condition does not apply. The two classes that reach this code were checked anyway. Global mutable state and concurrency: the signer holds no package-level mutable state, and `SessionManager.SendEncryptedCommand` serializes on its own mutex, so concurrent `Sign` calls are safe. Error and panic handling: every error path returns rather than panics, with the one swallowed case being the leaked session covered as a Warning above.

## Open questions

- The `privval` glossary entry lists three signer modes and says "every mode except tmkms is wrapped by the sign state gate"; a merge here makes it four. Not posted, our glossary, not the PR's.
- `TestConfig()` in the yubihsm package is unused, but `config.TestConfig()` and `TestPrivValidatorConfig()` set the same precedent, so it is consistent rather than dead. Not posted, no change needed.
- CI's `check` job is red only because the PR title uses `privval:`, which is not a conventional-commit type. Not posted, contribution-policy compliance is out of scope for findings.
- The lazy-init guard at [`start.go:231-233`](https://github.com/gnolang/gno/blob/c9aaed5c8/gno.land/cmd/gnoland/start.go#L231-L233) · [↗](../../../../../.worktrees/gno-review-5980/gno.land/cmd/gnoland/start.go#L231-L233) rejects tmkms because the validator pubkey is not locally available. The yubihsm path does not need the same guard: it fetches the real device pubkey at construction, so lazy genesis seeds correctly. Not posted, nothing to change.
