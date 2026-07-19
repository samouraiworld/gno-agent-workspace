# PR [#5958](https://github.com/gnolang/gno/pull/5958): privval: add GCP Secret Manager backed signer

URL: https://github.com/gnolang/gno/pull/5958
Author: ygd58 | Base: master | Files: 6 | +512 -12
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: f2e427a71 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5958 f2e427a71`

**TL;DR:** A validator normally reads its signing key from a file on disk. This PR adds an option to read that key from Google Cloud's Secret Manager instead, so the key lives in a managed store rather than on the machine. The key is still copied into the node's memory and all signing still happens inside the node.

**Verdict: REQUEST CHANGES** — the branch does not compile, and a mistyped configuration silently mints a new validator identity instead of failing (2 Critical, 4 Warnings, 2 Nits, 2 Missing tests, 1 Suggestion).

## Summary

The new [`gcpsecretmanager`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L1) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L1) package fetches the amino-JSON `FileKey` payload from a Secret Manager version at startup, parses it through the local signer's newly exported [`ParseFileKey`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/local/key.go#L92) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/local/key.go#L92), and holds the key in memory for the process lifetime. Signing is in-process, so the security posture matches the local-file signer rather than the tmkms listener; the package doc comment states this plainly.

Two blockers. The branch imports `cloud.google.com/go/secretmanager` without touching `go.mod` or `go.sum`, so `go build ./...` fails at this head. And a configuration block carrying only `project_id` or only `secret_id` passes validation, is reported as disabled by [`IsEnabled`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L48) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L48), and falls through to `LoadOrMakeLocalSigner`, which generates and persists a fresh validator key. An operator migrating a live validator to Secret Manager gets a silently different identity.

Three further defects sit in the `create_if_missing` path: it aborts on the exact "secret exists, no version yet" case its own helper documents, it writes version 1 while the config may read a pinned version, and the gRPC client it opens is never closed.

```
config block                     IsEnabled   ValidateBasic   signer chosen
-------------------------------  ---------   -------------   ----------------------
project_id="p" secret_id="s"     true        ok              gcpsecretmanager
project_id="p" secret_id=""      false       ok  <-- gap     local (mints new key)
project_id=""  secret_id="s"     false       ok  <-- gap     local (mints new key)
```

## Fix

`Config.ValidateBasic` at [`config.go:54-58`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) returns nil unconditionally, which is what lets both the half-filled block and the pinned-version-plus-`create_if_missing` combination through. Making it reject a block where exactly one of `ProjectID` and `SecretID` is set, and reject `CreateIfMissing` together with a numeric `Version`, closes both without touching the signer. The dependency blocker needs `go.mod` and `go.sum` regenerated; the resolved closure is 16 modules.

## Critical (must fix)

- **[branch does not compile]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go:7-8` — the GCP Secret Manager imports have no `go.mod` or `go.sum` entry, so `go build ./...` fails at this head.
  <details><summary>details</summary>

  [`client.go:7-8`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L7-L8) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L7-L8) imports `cloud.google.com/go/secretmanager/apiv1` and its `secretmanagerpb` subpackage, but the diff touches neither `go.mod` nor `go.sum`. The repository's own lint run uses `modules-download-mode: readonly` per [`.github/golangci.yml:6`](https://github.com/gnolang/gno/blob/f2e427a71/.github/golangci.yml#L6) · [↗](../../../../../.worktrees/gno-review-5958/.github/golangci.yml#L6), so it fails the same way. The test suites never ran on this PR: the only CI job that executed was the semantic-title check. Resolving the closure locally pulls in 16 modules, 2 direct and 14 indirect. Fix: commit the regenerated `go.mod` and `go.sum`. [repro](comment_claude-opus-4-8.md)
  </details>

- **[wrong validator identity, silently]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go:48-58` — a config block with only `project_id` or only `secret_id` passes validation and falls through to the local signer, which mints and persists a new validator key.
  <details><summary>details</summary>

  [`IsEnabled`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L48-L50) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L48-L50) requires both fields, and [`ValidateBasic`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) never rejects a partial block, so [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L151-L157) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L151-L157) reaches `local.LoadOrMakeLocalSigner`, which calls [`GeneratePersistedFileKey`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/local/key.go#L141-L142) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/local/key.go#L141-L142) when the file is absent. The observed signer is `{Type: LocalSigner, Addr: g19e4ml9t...}` with a fresh key written to disk. The operator-facing path reproduces it: `gnoland config set consensus.priv_validator.gcp_secret_manager.project_id my-proj` is accepted and leaves `secret_id` empty. An operator moving a live validator to Secret Manager loses the identity they meant to keep, or keeps signing with a stale on-disk key they believed was retired. Fix: reject a block where exactly one of `ProjectID` and `SecretID` is set. [repro](comment_claude-opus-4-8.md)
  </details>

## Warnings (should fix)

- **[documented recovery path does not work]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:123-136` — `createAndStoreKey` always calls `CreateSecret`, so the "secret exists but has no versions yet" case that `isNotFound` documents aborts with `AlreadyExists`.
  <details><summary>details</summary>

  [`isNotFound`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L103-L109) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L103-L109) states that Secret Manager answers NotFound both for a missing secret and for an existing secret with no versions. In the second case [`createAndStoreKey`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L123-L136) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L123-L136) hits `AlreadyExists` on `CreateSecret` and returns `unable to create secret "s" in GCP Secret Manager`, with no way to recover short of deleting the secret. The same shape blocks a retry after a transient `AddSecretVersion` failure, which leaves a created-but-empty secret behind. Fix: treat `AlreadyExists` on `CreateSecret` as success and continue to `AddSecretVersion`. [repro](comment_claude-opus-4-8.md)
  </details>

- **[key written where it cannot be read back]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:138-145` — `create_if_missing` with a numeric `version` writes version 1 while the config reads the pinned version, so the node signs once with a key no restart can load.
  <details><summary>details</summary>

  [`versionName`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L62-L69) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L62-L69) resolves to `versions/5` when `Version` is `"5"`, but [`AddSecretVersion`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L138-L145) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L138-L145) can only create version 1. The first boot succeeds and signs blocks with the generated key; every restart then fails, because `versions/5` is still NotFound and `CreateSecret` now answers `AlreadyExists`. `ValidateBasic` accepts the combination. Fix: reject `create_if_missing` together with a version other than `latest`. [repro](comment_claude-opus-4-8.md)
  </details>

- **[leaked connection]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go:69-74` — the GCP client opened at construction is never closed, so its gRPC connection and background goroutines live for the process lifetime.
  <details><summary>details</summary>

  [`newClient`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L68-L75) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/client.go#L68-L75) builds a `*secretmanager.Client`, whose upstream doc comment says the user should invoke `Close` when the client is no longer required. The only `Close` in the package is [`Signer.Close`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L42-L44) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L42-L44), which returns nil and never sees the client. The key is read exactly once inside [`newSigner`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L79-L101) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/signer.go#L79-L101), so nothing needs the connection afterwards. Confirmed by source: `grep -rn '\.Close()'` over the package matches only the test's `signer.Close()`. Fix: close the client once the key is loaded.
  </details>

- **[stale operator guidance]** `docs/validators/tmkms.md:13-20` — the guide says gnoland supports three mutually exclusive privval setups and this PR adds a fourth without updating the table.
  <details><summary>details</summary>

  [`tmkms.md:13-14`](https://github.com/gnolang/gno/blob/f2e427a71/docs/validators/tmkms.md?plain=1#L13-L14) · [↗](../../../../../.worktrees/gno-review-5958/docs/validators/tmkms.md#L13-L14) reads "Gnoland supports three privval setups. Pick one — they're mutually exclusive", and the table below it enumerates local file, gnokms, and tmkms. The new mode belongs in that table, and its production-readiness cell matches the [local-file row](https://github.com/gnolang/gno/blob/f2e427a71/docs/validators/tmkms.md?plain=1#L18) · [↗](../../../../../.worktrees/gno-review-5958/docs/validators/tmkms.md#L18): the key sits in the gnoland process next to a network listener, and there is no signer-side double-sign protection. A reader who sees "cloud KMS" without that row will assume otherwise. Fix: add the row, with the local-file security posture stated.
  </details>

## Nits

- **[validation that validates nothing]** [`config.go:54-58`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L54-L58) — `ValidateBasic` returns nil unconditionally and its comment asserts there are no cross-field constraints; the two Criticals and the pinned-version Warning are all cross-field constraints. The comment is the thing to change first.

- **[comment hides the trap]** [`config.go:17`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L17) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/config.go#L17) — the `secret_id` TOML comment reads "If set (together with project_id), the local signer is disabled". The parenthetical is the whole contract, and it renders in every generated `config.toml`. Say that both are required and that setting one alone is an error.

## Missing Tests

- **[mock cannot express the failing case]** `tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_test.go:49-58` — `mockClient.CreateSecret` always succeeds and ignores whether the secret already exists, so no test in the file can reach either `create_if_missing` defect.
  <details><summary>details</summary>

  [`CreateSecret`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_test.go#L49-L58) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_test.go#L49-L58) returns a fresh `*secretmanagerpb.Secret` on every call and never consults `m.secrets`, so `TestNewSigner_MissingSecret_CreateIfMissing` passes on a code path that cannot fail. Tracking created secret names in the mock and returning `AlreadyExists` on a repeat call makes both Warnings reproducible. Ready-to-add cases: [`tests/gcpsecretmanager_review_test.go`](tests/gcpsecretmanager_review_test.go).
  </details>

- **[wiring is untested]** `tm2/pkg/bft/privval/config.go:110-126` — none of the new `PrivValidatorConfig` behavior is covered: not the nil guard, not `errMultipleSignerSourcesSet`, and not the fall-through to the local signer.
  <details><summary>details</summary>

  The GCP branches added to [`ValidateBasic`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L110-L126) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L110-L126), [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L151-L154) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L151-L154), and [`NewPrivValidatorFromConfig`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L173-L186) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L173-L186) have no test in `tm2/pkg/bft/privval`. The mutual-exclusion pair is the one a fourth signer PR will silently break. Ready-to-add case: [`tests/privval_config_review_test.go`](tests/privval_config_review_test.go).
  </details>

## Suggestions

- **[four signers, four hardcoded branches]** [`config.go:151-157`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L151-L157) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L151-L157) — signer selection is an if-chain over concrete config fields, and the exclusion rule is written pairwise; the three sibling PRs each add another branch to the same lines.
  <details><summary>details</summary>

  [#5956](https://github.com/gnolang/gno/pull/5956), [#5959](https://github.com/gnolang/gno/pull/5959), and [#5980](https://github.com/gnolang/gno/pull/5980) all modify `tm2/pkg/bft/privval/config.go` and `tm2/pkg/bft/privval/signer/local/key.go` with the same shape: one config field, one `IsEnabled` branch, one clause bolted onto the exclusion condition, and the same `ParseFileKey` extraction. The pairwise condition at [`config.go:124-126`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/privval/config.go#L124-L126) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/privval/config.go#L124-L126) grows quadratically and the error text `only one of remote_signer, tmkms_listener, or gcp_secret_manager` goes stale in each sibling. A list of enabled mode names with a single "at most one" check would land all four without conflict. Separately, this PR alone adds 16 modules including `google.golang.org/api` and the OpenTelemetry gRPC and HTTP instrumentation; merging all four links four cloud SDKs into every `gnoland` binary regardless of the signer in use.
  </details>

## Verified

- The branch does not build at f2e427a71: `go build ./...` fails on the missing `go.sum` entries for `cloud.google.com/go/secretmanager/apiv1`. Resolving the closure with `go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager` adds 16 module lines, after which `go test ./tm2/pkg/bft/privval/...` is green across all 7 packages.
- A half-filled GCP block mints a validator key on disk. `NewSignerFromConfig` with `project_id` set and `secret_id` empty returned `{Type: LocalSigner, Addr: g19e4ml9tuldw3j4fjjd43c32mdagp3luwhgnydm}` and wrote `priv_validator_key.json` into an empty root. The operator-facing entry point agrees: `gnoland config set consensus.priv_validator.gcp_secret_manager.project_id my-proj` is accepted, and `config get` reports `secret_id: ""`.
- `create_if_missing` cannot recover an existing empty secret, and cannot survive a restart under a pinned version. With `CreateSecret` answering `AlreadyExists`, `newSigner` returns `unable to create secret "s" in GCP Secret Manager` in both cases.
- Existing `config.toml` files stay loadable: [`LoadConfigFile`](https://github.com/gnolang/gno/blob/f2e427a71/tm2/pkg/bft/config/toml.go#L18-L26) · [↗](../../../../../.worktrees/gno-review-5958/tm2/pkg/bft/config/toml.go#L18-L26) starts from `DefaultConfig()` and overlays the file, so an absent `[consensus.priv_validator.gcp_secret_manager]` section leaves the non-nil default in place and the new nil check never fires. `gnoland config init` emits the section with `create_if_missing = false` and `version = "latest"`.
- GCP call deadlines are bounded: `AccessSecretVersion` carries a default 60s `gax.WithTimeout` and retries only `Unavailable` and `ResourceExhausted`, so a `context.Background()` fetch at startup cannot hang forever.

## Open questions

- The signer holds `*local.FileKey` for the process lifetime with no zeroization, matching `LocalSigner`. Not posted: identical to the existing local signer, so it is a package-wide question rather than a defect this PR introduces.
- `version = "latest"` as the default means a key rotation in Secret Manager changes the validator's identity on the next restart. Not posted: an operator policy choice, and the field exists precisely so it can be pinned.
- Invariant catalog not walked: the diff is Go-only under `tm2/`, with no GnoVM, stdlib, or `.gno` change.
