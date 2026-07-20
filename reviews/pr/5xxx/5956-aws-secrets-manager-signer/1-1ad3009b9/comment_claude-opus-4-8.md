# Review: PR [#5956](https://github.com/gnolang/gno/pull/5956)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

- Three sibling PRs land the same shape as this one. [#5980](https://github.com/gnolang/gno/pull/5980), [#5959](https://github.com/gnolang/gno/pull/5959) and [#5958](https://github.com/gnolang/gno/pull/5958) each add a signer triple, a `PrivValidatorConfig` field, and another term to the mutual-exclusion expression, and three of the four also extract `ParseFileKey` from `local/key.go`. A small registry keyed on an "at most one enabled" interface would let the four merge independently instead of conflicting pairwise, and it should be settled before any of them lands.

Repros run at 1ad3009b9.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5956-aws-secrets-manager-signer/1-1ad3009b9/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/privval/signer/awssecretsmanager/client.go:7-8 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/client.go#L7-L8)
Critical: the three `aws-sdk-go-v2` import paths are missing from `go.mod` and `go.sum`. gno is a single module, so `go build ./...` fails on every target, not just this package. Adding the two module roots makes everything under `tm2/pkg/bft/privval` build and pass.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5956 -R gnolang/gno
go build ./... ; echo "exit=$?"
```

```
tm2/pkg/bft/privval/signer/awssecretsmanager/client.go:7:2: no required module provides package github.com/aws/aws-sdk-go-v2/config; to add it:
	go get github.com/aws/aws-sdk-go-v2/config
tm2/pkg/bft/privval/signer/awssecretsmanager/client.go:8:2: no required module provides package github.com/aws/aws-sdk-go-v2/service/secretsmanager; to add it:
	go get github.com/aws/aws-sdk-go-v2/service/secretsmanager
tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go:9:2: no required module provides package github.com/aws/aws-sdk-go-v2/service/secretsmanager/types; to add it:
	go get github.com/aws/aws-sdk-go-v2/service/secretsmanager/types
exit=1
```
</details>

## tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go:131-133 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L133)
`CreateSecret` is called without `KmsKeyId`, so a key minted by `create_if_missing` is encrypted under the `aws/secretsmanager` default key, which [every user and role in the account can use](https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_CreateSecret.html). Only the secret's own resource policy then guards the validator key, and [`Config`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25) offers no way to name a customer-managed key.

## tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:11 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L11)
`secret_id` is documented as an ARN or a name, but [`createAndStoreKey` passes it as `CreateSecret`'s `Name`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L132), and [`Name` allows only letters, numbers and `/_+=.@-`](https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_CreateSecret.html), so an ARN's colons are rejected. An ARN plus `create_if_missing` therefore fails at node startup with an opaque `InvalidParameterException`, and [`ValidateBasic`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L49-L53) returns nil unconditionally so config load never catches it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5956 -R gnolang/gno
go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/service/secretsmanager
cat > tm2/pkg/bft/privval/signer/awssecretsmanager/zz_arn_test.go <<'EOF'
package awssecretsmanager

import "testing"

func TestARNWithCreateIfMissing(t *testing.T) {
	const arn = "arn:aws:secretsmanager:eu-west-1:123456789012:secret:validator-key-a1b2c3"
	if err := (&Config{SecretID: arn, CreateIfMissing: true}).ValidateBasic(); err == nil {
		t.Errorf("ValidateBasic accepted an ARN secret_id with create_if_missing")
	}
}
EOF
go test -run TestARNWithCreateIfMissing ./tm2/pkg/bft/privval/signer/awssecretsmanager/
rm tm2/pkg/bft/privval/signer/awssecretsmanager/zz_arn_test.go && git checkout go.mod go.sum
```

```
--- FAIL: TestARNWithCreateIfMissing (0.00s)
    zz_arn_test.go:8: ValidateBasic accepted an ARN secret_id with create_if_missing
FAIL
FAIL	github.com/gnolang/gno/tm2/pkg/bft/privval/signer/awssecretsmanager	0.012s
FAIL
```
</details>

## tm2/pkg/bft/privval/config.go:38-42 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L38-L42)
Any host with the IAM role can fetch the key, while [the sign state guarding against double-signing is a file under the node's own `RootDir`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config.go#L206-L210). Two nodes pointed at the same `secret_id` come up as the same validator with independent state files and equivocate at the first contested height. gnokms and tmkms avoid this because the key never leaves the signer process, so document here that exactly one node may hold the secret.

## tm2/pkg/bft/privval/config.go:110-126 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/config.go#L110-L126)
Missing test: nothing in the privval package exercises `errNilAWSSecretsManagerCfg` or `errMultipleSignerSourcesSet`, in either `ValidateBasic` or `NewPrivValidatorFromConfig`, though every earlier rule of this kind has a subtest: [nil remote signer config](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config_test.go#L56-L63) and [both external signers enabled](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/config_test.go#L262-L273). A later edit dropping the exclusion term would go unnoticed, and the first mode in source order would silently win.

<details><summary>test cases</summary>

```go
func TestValidateBasicAWSSecretsManager(t *testing.T) {
	t.Parallel()

	t.Run("aws secrets manager config is nil", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.AWSSecretsManager = nil

		assert.ErrorIs(t, cfg.ValidateBasic(), errNilAWSSecretsManagerCfg)
	})

	t.Run("aws secrets manager with remote signer rejected", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.AWSSecretsManager = &awssecretsmanager.Config{SecretID: "validator-key"}
		cfg.RemoteSigner.ServerAddress = "unix:///tmp/remote_signer.sock"

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("aws secrets manager with tmkms listener rejected", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.AWSSecretsManager = &awssecretsmanager.Config{SecretID: "validator-key"}
		cfg.RemoteSigner.ServerAddress = ""
		cfg.TmkmsListener.ListenAddr = "tcp://127.0.0.1:0"
		cfg.TmkmsListener.ChainID = "test-chain"
		// The allowlist is validated before the exclusion check, so it must be
		// a well-formed hex ed25519 pubkey for the exclusion error to surface.
		pub := ed25519.GenPrivKey().PubKey().(ed25519.PubKeyEd25519)
		cfg.TmkmsListener.AllowedKMSPubKeys = []string{hex.EncodeToString(pub[:])}

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("aws secrets manager alone accepted", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.AWSSecretsManager = &awssecretsmanager.Config{SecretID: "validator-key"}
		cfg.RemoteSigner.ServerAddress = ""

		assert.NoError(t, cfg.ValidateBasic())
	})
}

func TestNewPrivValidatorFromConfigAWSSecretsManager(t *testing.T) {
	t.Parallel()

	privKey := ed25519.GenPrivKey()
	logger := log.NewNoopLogger()

	t.Run("aws secrets manager with remote signer rejected", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.RootDir = t.TempDir()
		cfg.AWSSecretsManager = &awssecretsmanager.Config{SecretID: "validator-key"}
		cfg.RemoteSigner.ServerAddress = "unix:///tmp/remote_signer.sock"

		privVal, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		require.Nil(t, privVal)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})

	t.Run("aws secrets manager with tmkms listener rejected", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.RootDir = t.TempDir()
		cfg.AWSSecretsManager = &awssecretsmanager.Config{SecretID: "validator-key"}
		cfg.RemoteSigner.ServerAddress = ""
		cfg.TmkmsListener.ListenAddr = "tcp://127.0.0.1:0"
		cfg.TmkmsListener.ChainID = "test-chain"

		privVal, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		require.Nil(t, privVal)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})
}
```
</details>

## tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go:107-116 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L107-L116)
Missing test: [the mock only ever returns `SecretString`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L48), so `secretPayload`'s `SecretBinary` fallback and its `errEmptySecretValue` return never run. A secret created outside gno through the console's binary upload takes the untested branch on every startup.

<details><summary>test cases</summary>

```go
func TestSecretPayload(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"priv_key":null}`)

	t.Run("secret string", func(t *testing.T) {
		t.Parallel()

		value := string(raw)
		got, err := secretPayload(&secretsmanager.GetSecretValueOutput{SecretString: &value})
		require.NoError(t, err)
		assert.Equal(t, raw, got)
	})

	t.Run("secret binary fallback", func(t *testing.T) {
		t.Parallel()

		got, err := secretPayload(&secretsmanager.GetSecretValueOutput{SecretBinary: raw})
		require.NoError(t, err)
		assert.Equal(t, raw, got)
	})

	t.Run("neither field set", func(t *testing.T) {
		t.Parallel()

		got, err := secretPayload(&secretsmanager.GetSecretValueOutput{})
		require.ErrorIs(t, err, errEmptySecretValue)
		assert.Nil(t, got)
	})
}
```
</details>

## tm2/pkg/bft/privval/signer/local/key.go:81-84 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/local/key.go#L81-L84)
Nit: validation failures now arrive wrapped as `unable to unmarshal FileKey from <path>: address does not match public key`, naming unmarshalling for a file that unmarshalled fine. Unmarshal failures get the prefix twice, since [`ParseFileKey` already adds one](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/local/key.go#L96).

## tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:37-40 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L37-L40)
Nit: `TestConfig` has no callers anywhere in the tree, unlike the [`TestRemoteSignerClientConfig`](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/remote/client/config.go#L46) pattern it copies, which [its own config test](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/remote/client/config_test.go#L28) exercises.

## tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go:25-26 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/awssecretsmanager_test.go#L25-L26)
Nit: `createErr` is honored by the mock but never set by any test, so the [`CreateSecret` failure path](https://github.com/gnolang/gno/blob/1ad3009b9/tm2/pkg/bft/privval/signer/awssecretsmanager/signer.go#L131-L136) never runs.

## tm2/pkg/bft/privval/signer/awssecretsmanager/config.go:6-25 [↗](../../../../../.worktrees/gno-review-5956/tm2/pkg/bft/privval/signer/awssecretsmanager/config.go#L6-L25)
Suggestion: this mode ships no operator docs, so the required IAM actions, the expected secret payload shape and the single-node constraint live only in TOML comments. A short `docs/validators/aws-secrets-manager.md` alongside [the tmkms page](https://github.com/gnolang/gno/blob/1ad3009b9/docs/validators/tmkms.md?plain=1#L1) would carry all three.
