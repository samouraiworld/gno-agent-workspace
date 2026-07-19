# PR [#5956](https://github.com/gnolang/gno/pull/5956): privval: add AWS Secrets Manager backed signer

URL: https://github.com/gnolang/gno/pull/5956
Author: ygd58 | Base: master | Files: 6 | +442 -12
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 1ad3009b9 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5956 1ad3009b9`

**TL;DR:** A gno validator normally keeps its signing key in a file on the machine it runs on. This PR adds an option to keep that key in AWS Secrets Manager instead: the node downloads the key once at startup and signs with it in memory, exactly as it would with the file.

**Verdict: REQUEST CHANGES** — the AWS SDK imports were never added to `go.mod`, so the repository does not build at this commit; on top of that, `create_if_missing` writes the validator key under the account-wide default KMS key and cannot work with the ARN form the config documents (1 Critical, 3 Warnings, 2 Missing tests, 3 Nits, 2 Suggestions).

## Summary

The new package [`tm2/pkg/bft/privval/signer/awssecretsmanager`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L1) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L1) implements `types.Signer` by fetching a `priv_validator_key.json` payload from a Secrets Manager secret and keeping the parsed key in memory. It reuses the local signer's encoding through a new exported [`local.ParseFileKey`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/local/key.go#L92) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/local/key.go#L92), and plugs into privval as a fourth signer mode with a [three-way mutual-exclusion check](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L124-L126) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L124-L126). The mode is off by default: an empty `secret_id` disables it.

The AWS path goes through the same wrapper as the local and gnokms modes, so it is [wrapped by the sign state gate](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L206-L210) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L206-L210). It also composes with `gnoland start -lazy`, which derives a genesis validator set from [whatever signer the config selects](https://github.com/gnolang/gno/blob/1ad3009b9/gno.land/cmd/gnoland/start.go#L242) · [↗](../../../../../.worktrees/gno-review-5956/gno.land/cmd/gnoland/start.go#L242), so a lazy node comes up with the AWS-held pubkey rather than silently falling back to a local key.

## Glossary

- privval: the validator's private-key signing subsystem (`tm2/pkg/bft/privval`), one signer mode selected from config.
- sign state: the local `priv_validator_state.json` height/round/step record consulted before every signature, which only serializes signing within one node.

## Critical (must fix)

- **[repository does not build]** `tm2/pkg/bft/privval/signer/awssecretsmanager/client.go:7-8` — the AWS SDK imports are never added to `go.mod` or `go.sum`, so `go build ./...` fails at the root module.
  <details><summary>details</summary>

  The new package imports [`aws-sdk-go-v2/config` and `aws-sdk-go-v2/service/secretsmanager`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/client.go#L7-L8) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/client.go#L7-L8), plus `.../secretsmanager/types` in [signer.go](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L8-L9) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L8-L9). None of the three appears in the diff's module files, and gno is a single-module repo, so every build target fails, not just the new package. `go get` on the two roots pulls 19 modules, and after that the whole diff builds and every test in `tm2/pkg/bft/privval/...` is green, so this is purely a missing dependency record. The PR's own test jobs never caught it because a first-time fork contributor's workflows sit pending maintainer approval. Fix: add the AWS SDK requirements to `go.mod` and `go.sum`.
  </details>

## Warnings (should fix)

- **[new validator key readable by the whole AWS account]** `tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go:131-133` — `create_if_missing` stores the key with no `KmsKeyId`, so it lands under the `aws/secretsmanager` managed key that every user and role in the account can use.
  <details><summary>details</summary>

  [`createAndStoreKey` passes only `Name` and `SecretString`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L133) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L133). The AWS API reference for [CreateSecret](https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_CreateSecret.html) states that without `KmsKeyId` Secrets Manager uses `aws/secretsmanager`, and that "All users and roles in the AWS account automatically have access to use `aws/secretsmanager`". The encryption boundary then collapses to the `secretsmanager:GetSecretValue` resource policy alone, with no second, independently-scoped KMS key policy in front of a validator signing key. [`Config`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25) exposes no way to name a customer-managed key. Fix: add a `kms_key_id` option and pass it through to `CreateSecret`.
  </details>

- **[documented ARN form cannot be created]** `tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:11` — `secret_id` is documented as an ARN or a name, but with `create_if_missing` the same string is passed as `CreateSecret`'s `Name`, which rejects the colons an ARN contains.
  <details><summary>details</summary>

  The [`secret_id` comment](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L11) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L11) says "ARN or name", which holds for `GetSecretValue` but not for creation: [`createAndStoreKey` sets `Name: &secretID`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L132) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L132), and the [CreateSecret reference](https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_CreateSecret.html) limits `Name` to ASCII letters, numbers and `/_+=.@-`. An operator who points `secret_id` at an ARN, turns on `create_if_missing`, and hits a missing secret gets an opaque `InvalidParameterException` at node startup. [`Config.ValidateBasic`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L49-L53) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L49-L53) returns nil unconditionally, so nothing catches it during config load. Confirmed with [`tests/config_arn_create_test.go`](tests/config_arn_create_test.go), which fails at this commit. Fix: reject the ARN-plus-`create_if_missing` combination during config validation.
  </details>

- **[shared key store, per-node double-sign guard]** `tm2/pkg/bft/privval/config.go:151-154` — moving the key into a shared cloud store makes it trivially fetchable from several hosts, while the sign state that prevents double-signing stays a local file on each of them.
  <details><summary>details</summary>

  The AWS branch returns a plain signer that [`NewPrivValidatorFromConfig` wraps with a `FileState` at `config.SignStatePath()`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L206-L210) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L206-L210), a path under the node's own `RootDir`. The gnokms and tmkms modes do not have this shape: the key never leaves the signer process, so the signer is itself the serialization point. The local file mode has it only in the sense that copying the key file is a deliberate act. Here two nodes configured with the same `secret_id` both come up as the same validator, each with its own `priv_validator_state.json`, and equivocate at the first contested height. Nothing in the [config field comment](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L38-L42) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L38-L42) or the [signer doc comment](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L16-L22) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L16-L22) mentions it. Fix: say in the config comment that the operator must guarantee exactly one node holds the secret at a time.
  </details>

## Missing Tests

- **[new rejection rules are unexercised]** `tm2/pkg/bft/privval/config.go:110-126` — the PR adds `errNilAWSSecretsManagerCfg` and `errMultipleSignerSourcesSet` to both `ValidateBasic` and `NewPrivValidatorFromConfig`, with no test in the privval package.
  <details><summary>details</summary>

  [`config_test.go`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config_test.go#L23) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config_test.go#L23) has a matching subtest for every earlier rule of this kind: [nil remote signer config](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config_test.go#L56-L63) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config_test.go#L56-L63) and [both external signers enabled](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config_test.go#L262-L273) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config_test.go#L262-L273). The AWS equivalents have none, so a later edit that drops the exclusion term silently reintroduces a config in which two signer modes are set and the first one in source order silently wins. Ready-to-add cases in [`tests/config_awssecretsmanager_test.go`](tests/config_awssecretsmanager_test.go), green at this commit once the AWS modules exist.
  </details>

- **[binary secret path never runs]** `tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go:107-116` — `secretPayload`'s `SecretBinary` fallback and its `errEmptySecretValue` return are unreachable in the test suite.
  <details><summary>details</summary>

  [`secretPayload`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L107-L116) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L107-L116) has three outcomes, and the [mock only ever returns `SecretString`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L48) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L48), so two of them never execute. A secret created outside gno through the console's binary upload path takes the untested branch on every startup. Two table rows over `secretPayload` cover it: one `GetSecretValueOutput` with only `SecretBinary` set, one with neither field set asserting `errEmptySecretValue`.
  </details>

## Nits

- **[unreachable helper]** `tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:37-40` — [`TestConfig`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L37-L40) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L37-L40) has no callers anywhere in the tree. The pattern it copies, [`TestRemoteSignerClientConfig`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/remote/client/config.go#L46) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/remote/client/config.go#L46), is exercised by [its own config test](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/remote/client/config_test.go#L28) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/remote/client/config_test.go#L28).

- **[unused mock hook]** `tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go:25-26` — [`createErr`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L25-L26) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L25-L26) is declared and honored by the mock but never set by any test, so the `CreateSecret` failure path in [`createAndStoreKey`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L136) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L136) never runs.

- **[misleading error text]** `tm2/pkg/bft/privval/signer/local/key.go:81-84` — the `ParseFileKey` split makes [`LoadFileKey`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/local/key.go#L81-L84) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/local/key.go#L81-L84) wrap two different classes of failure under one "unable to unmarshal" prefix.
  <details><summary>details</summary>

  `ParseFileKey` already wraps the amino failure with ["unable to unmarshal FileKey"](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/local/key.go#L96) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/local/key.go#L96), so a malformed file now reports `unable to unmarshal FileKey from /path/bad.json: unable to unmarshal FileKey: invalid character 'n'`. The validation failures matter more: at the merge base `LoadFileKey` returned them bare, and a key whose address does not match its pubkey now reports `unable to unmarshal FileKey from /path/mismatch.json: address does not match public key`, naming unmarshalling for a file that unmarshalled fine. `errors.Is` still matches, so [`key_test.go:140`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/local/key_test.go#L140) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/local/key_test.go#L140) stays green. Both strings observed at 1ad3009b9 against the same inputs at HEAD~1.
  </details>

## Suggestions

- **[undocumented operator surface]** `tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:6-25` — the tmkms mode ships an operator page at [`docs/validators/tmkms.md`](https://github.com/gnolang/gno/blob/1ad3009b9/docs/validators/tmkms.md?plain=1#L1) · [↗](../../../../../.worktrees/gno-review-5956/docs/validators/tmkms.md#L1); this mode has no equivalent.
  <details><summary>details</summary>

  Everything an operator needs sits in [TOML comments](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25): nothing states the required IAM actions (`secretsmanager:GetSecretValue`, plus `secretsmanager:CreateSecret` when `create_if_missing` is on), the expected secret payload shape, or the single-node constraint from the double-sign Warning above. A short `docs/validators/aws-secrets-manager.md` alongside the tmkms page would carry all three.
  </details>

- **[fourth copy of the same wiring]** `tm2/pkg/bft/privval/config.go:120-126` — three sibling PRs add the same shape, and every one of them edits the same two files and the same mutual-exclusion expression.
  <details><summary>details</summary>

  [#5980](https://github.com/gnolang/gno/pull/5980) (YubiHSM2), [#5959](https://github.com/gnolang/gno/pull/5959) (HashiCorp Vault) and [#5958](https://github.com/gnolang/gno/pull/5958) (GCP Secret Manager) each add a `client.go` / `config.go` / `signer.go` triple, a `PrivValidatorConfig` field, a nil check, and another term to the [exclusion condition](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L120-L126) · [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L120-L126), whose error string names the modes one by one. Three of the four also extract `ParseFileKey` from `local/key.go`. Merged as they stand they conflict pairwise, and the exclusion check becomes a six-way boolean. A small registry keyed on an "at most one enabled" interface would collapse the check to a count and let each backend land independently. This is a cross-PR call for a maintainer, not a change to make inside this PR alone.
  </details>

## Verified

- The repository does not build at 1ad3009b9: `go build ./...` in the PR worktree stops on the three unresolved `aws-sdk-go-v2` import paths. Adding the two module roots with `go get` resolves 19 modules, after which `go test ./tm2/pkg/bft/privval/...` is green across all seven packages including the new one.
- The mutual-exclusion rules the PR adds do hold once tested: [`tests/config_awssecretsmanager_test.go`](tests/config_awssecretsmanager_test.go) passes on all six cases, so the finding is missing coverage rather than a broken rule.
- The ARN plus `create_if_missing` combination is accepted by config validation today: [`tests/config_arn_create_test.go`](tests/config_arn_create_test.go) fails at this commit on the ARN case and passes the two supported ones.
- `LoadFileKey`'s validation-failure message changed against the merge base. Feeding a key file whose address does not match its pubkey returns `address does not match public key` at HEAD~1 and `unable to unmarshal FileKey from <path>: address does not match public key` at 1ad3009b9.
- Malformed secret payloads do not leak key material into error text: a secret whose `priv_key.value` is a recognisable string but whose shape is wrong produces `invalid validator key in secret "k": unable to unmarshal FileKey: decodeReflectJSONArray: byte-length mismatch, got 16 want 64`, with no fragment of the payload in it.

## Open questions

- The failing `check` job is the conventional-commit title lint rejecting the `privval:` scope-as-type. Contribution-convention only, no code impact, so it is not posted.
- `create_if_missing` has no `ResourceExistsException` handling, so two nodes racing to seed the same secret leave the loser unable to start. Startup-only failure with a clear AWS error, and the flag is off by default, so not worth a comment.
- The key is fetched once and cached for process lifetime, which the signer doc comment states plainly, so a rotated secret needs a restart. Correct for a validator identity key; noting it only because a reader may expect Secrets Manager rotation to apply.
