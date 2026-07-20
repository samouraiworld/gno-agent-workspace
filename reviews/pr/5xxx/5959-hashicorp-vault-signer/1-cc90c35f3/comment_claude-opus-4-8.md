# Review: PR [#5959](https://github.com/gnolang/gno/pull/5959)
Event: COMMENT

## Body
[AI bot - Automatic review]

Automated technical pass: does the code build, run, and behave as described. No design or scope judgement, and no merge verdict. Posted to give a human reviewer a head start.

Nothing in CI has compiled this branch: the build and test workflows never ran, blocked by the initial-approval gate. Everything below was checked locally at cc90c35f3.

The struct doc at [`tm2/pkg/bft/privval/config.go:23`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L23) still says at most one of `RemoteSigner` or `TmkmsListener` may be enabled, now that Vault is a third exclusive mode.

Repros run at cc90c35f3.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5959-hashicorp-vault-signer/1-cc90c35f3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## tm2/pkg/bft/privval/signer/vault/client.go:7 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/client.go#L7)
Critical: `github.com/hashicorp/vault/api` is imported here but absent from the [direct requires in `go.mod`](https://github.com/gnolang/gno/blob/cc90c35f3/go.mod#L5-L62), so the module does not build. `go get` resolves v1.23.0 and 16 `require` lines, after which `tm2/pkg/bft/privval/...` builds and tests pass.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5959 -R gnolang/gno
go build ./tm2/pkg/bft/privval/...
```

```
tm2/pkg/bft/privval/signer/vault/client.go:7:2: no required module provides package github.com/hashicorp/vault/api; to add it:
	go get github.com/hashicorp/vault/api
```
</details>

## tm2/pkg/bft/privval/signer/vault/config.go:16 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L16)
The `toml` tag puts the Vault token into `config.toml`, which the node [writes at mode 0644](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/config/toml.go#L61), while the validator key that token fetches is [written at 0600](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L65). An operator who fills the field trades a private key file for a world-readable credential that retrieves it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5959 -R gnolang/gno
go get github.com/hashicorp/vault/api
cat > tm2/pkg/bft/config/zz_token_test.go <<'EOF'
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultTokenOnDisk(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	cfg := DefaultConfig()
	cfg.Consensus.PrivValidator.Vault.Token = "hvs.SUPERSECRET"
	if err := WriteConfigFile(p, cfg); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(p)
	b, _ := os.ReadFile(p)
	t.Logf("config.toml mode = %v", st.Mode().Perm())
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, "hvs.SUPERSECRET") {
			t.Logf("in file: %s", line)
		}
	}
}
EOF
go test -v -run TestVaultTokenOnDisk ./tm2/pkg/bft/config/
rm tm2/pkg/bft/config/zz_token_test.go && git checkout go.mod go.sum
```

```
=== RUN   TestVaultTokenOnDisk
    zz_token_test.go:20: config.toml mode = -rw-r--r--
    zz_token_test.go:23: in file: token = "hvs.SUPERSECRET"
--- PASS: TestVaultTokenOnDisk (0.00s)
```
</details>

## tm2/pkg/bft/privval/signer/vault/signer.go:92-97 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/signer.go#L92-L97)
Vault returns a nil `Data` map when the latest version was deleted but not destroyed, so with `create_if_missing` the node generates a fresh key and writes it as a newer version. The deleted key survives in history but is no longer what the node reads. `vault kv undelete` no longer recovers the validator, and the node comes up under a pubkey the validator set does not know.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5959 -R gnolang/gno
go get github.com/hashicorp/vault/api
cat > tm2/pkg/bft/privval/signer/vault/zz_softdelete_test.go <<'EOF'
package vault

import (
	"context"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/local"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
)

// A KV v2 path whose latest version was deleted but not destroyed: Get
// returns a secret with a nil Data map and no error.
type softDeletedKV struct{ written map[string]any }

func (m *softDeletedKV) Get(context.Context, string) (*vaultapi.KVSecret, error) {
	return &vaultapi.KVSecret{Data: nil}, nil
}

func (m *softDeletedKV) Put(_ context.Context, _ string, data map[string]any, _ ...vaultapi.KVOption) (*vaultapi.KVSecret, error) {
	m.written = data
	return &vaultapi.KVSecret{Data: data}, nil
}

func TestSoftDeletedSecretMintsNewIdentity(t *testing.T) {
	recoverable := local.GenerateFileKey()
	_, err := amino.MarshalJSONIndent(recoverable, "", "  ")
	require.NoError(t, err)

	client := &softDeletedKV{}
	cfg := &Config{SecretPath: "gno/validator-key", CreateIfMissing: true}

	signer, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err)

	t.Logf("recoverable key in Vault: %s", recoverable.Address)
	t.Logf("key the node came up with: %s", signer.PubKey().Address())
	t.Logf("a new version was written: %v", client.written != nil)
}
EOF
go test -v -run TestSoftDeletedSecretMintsNewIdentity ./tm2/pkg/bft/privval/signer/vault/
rm tm2/pkg/bft/privval/signer/vault/zz_softdelete_test.go && git checkout go.mod go.sum
```

```
=== RUN   TestSoftDeletedSecretMintsNewIdentity
    zz_softdelete_test.go:37: recoverable key in Vault: g15g728a8dafsufgavppqjn398s6kv6q94rykl96
    zz_softdelete_test.go:38: key the node came up with: g1t5esm9vadg3sy478pk8tye77yuqepzwecsha94
    zz_softdelete_test.go:39: a new version was written: true
--- PASS: TestSoftDeletedSecretMintsNewIdentity (0.00s)
```
</details>

## tm2/pkg/bft/privval/signer/vault/signer.go:135 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/signer.go#L135)
`Put` carries no `KVOption`, so [no options block reaches Vault](https://github.com/hashicorp/vault/blob/api/v1.23.0/api/kv_v2.go#L230-L237) and the write is unconditional. Two nodes started with `create_if_missing` against the same path both read "not found", both write, and the loser signs with a key that is no longer in Vault. A [`cas=0` write is allowed only when the key does not exist](https://github.com/hashicorp/vault/blob/api/v1.23.0/api/kv_v2.go#L95), and the [`kvAPI` interface](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/client.go#L15) already carries the option.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5959 -R gnolang/gno
go get github.com/hashicorp/vault/api
cat > tm2/pkg/bft/privval/signer/vault/zz_cas_test.go <<'EOF'
package vault

import (
	"context"
	"errors"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/local"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCheckAndSetFailed = errors.New("check-and-set parameter did not match the current version")

// casKV honours the cas option the real KV v2 engine honours.
type casKV struct{ secrets map[string]map[string]any }

func (m *casKV) Get(_ context.Context, secretPath string) (*vaultapi.KVSecret, error) {
	data, ok := m.secrets[secretPath]
	if !ok {
		return nil, vaultapi.ErrSecretNotFound
	}
	return &vaultapi.KVSecret{Data: data}, nil
}

func (m *casKV) Put(_ context.Context, secretPath string, data map[string]any, opts ...vaultapi.KVOption) (*vaultapi.KVSecret, error) {
	_, exists := m.secrets[secretPath]
	for _, opt := range opts {
		if k, v := opt(); k == vaultapi.KVOptionCheckAndSet && v == 0 && exists {
			return nil, errCheckAndSetFailed
		}
	}
	m.secrets[secretPath] = data
	return &vaultapi.KVSecret{Data: data}, nil
}

func TestCreateAndStoreKeyDoesNotOverwrite(t *testing.T) {
	const secretPath = "gno/validator-key"

	// Another node already seeded the validator key at this path.
	existing := local.GenerateFileKey()
	jsonBytes, err := amino.MarshalJSONIndent(existing, "", "  ")
	require.NoError(t, err)

	client := &casKV{secrets: map[string]map[string]any{
		secretPath: {dataFieldName: string(jsonBytes)},
	}}
	cfg := &Config{SecretPath: secretPath, CreateIfMissing: true}

	signer, err := createAndStoreKey(context.Background(), client, cfg)
	require.Error(t, err, "creating a key must not succeed when the path already holds one")
	assert.Nil(t, signer)

	stored, err := local.ParseFileKey([]byte(client.secrets[secretPath][dataFieldName].(string)))
	require.NoError(t, err)
	assert.True(t, stored.PubKey.Equals(existing.PubKey), "the pre-existing validator key was replaced")
}
EOF
go test -v -run TestCreateAndStoreKeyDoesNotOverwrite ./tm2/pkg/bft/privval/signer/vault/
rm tm2/pkg/bft/privval/signer/vault/zz_cas_test.go && git checkout go.mod go.sum
```

```
=== RUN   TestCreateAndStoreKeyDoesNotOverwrite
    zz_cas_test.go:60:
        	Error:      	An error is expected but got nil.
        	Messages:   	creating a key must not succeed when the path already holds one
--- FAIL: TestCreateAndStoreKeyDoesNotOverwrite (0.00s)
FAIL	github.com/gnolang/gno/tm2/pkg/bft/privval/signer/vault	0.007s
```
</details>

## tm2/pkg/bft/privval/config.go:110-126 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L110-L126)
Missing test: neither `errNilVaultConfig` nor `errMultipleSignerSourcesSet` is exercised, and the vault branch of [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L151-L154) is unreached by any test. `config_test.go` already covers the two neighbouring cases, [`errNilRemoteSignerConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config_test.go#L62) and [`errBothExternalSignersEnabled`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config_test.go#L272).

<details><summary>test cases</summary>

```go
t.Run("nil vault config rejected", func(t *testing.T) {
	t.Parallel()

	cfg := DefaultPrivValidatorConfig()
	cfg.Vault = nil

	assert.ErrorIs(t, cfg.ValidateBasic(), errNilVaultConfig)
})

t.Run("vault and remote signer both configured rejected", func(t *testing.T) {
	t.Parallel()

	cfg := DefaultPrivValidatorConfig()
	cfg.Vault.SecretPath = "gno/validator-key"
	cfg.RemoteSigner.ServerAddress = "unix:///tmp/x.sock"

	assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)

	_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
	assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
})

t.Run("vault and tmkms listener both configured rejected", func(t *testing.T) {
	t.Parallel()

	cfg := DefaultPrivValidatorConfig()
	cfg.Vault.SecretPath = "gno/validator-key"
	cfg.TmkmsListener.ListenAddr = "tcp://127.0.0.1:0"
	cfg.TmkmsListener.ChainID = "test"

	_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
	assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
})
```
</details>

## tm2/pkg/bft/privval/signer/local/key.go:83 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/local/key.go#L83)
Nit: a malformed file now carries the path prefix twice, since [line 96](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L96) adds its own copy. The wrap also catches the [`validate`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/local/key.go#L100-L102) errors that used to come back bare, so an address mismatch reports `unable to unmarshal FileKey from <path>: address does not match public key`. Wrap the file path once, where the file is read.

## tm2/pkg/bft/privval/signer/vault/config.go:25 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L25)
Nit: an operator seeding the secret by hand has no way to learn it must land under the `priv_validator_key_json` field, a private constant at [`signer.go:17`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/signer/vault/signer.go#L17). No `docs/validators/` page covers this mode, unlike [`tmkms.md`](https://github.com/gnolang/gno/blob/cc90c35f3/docs/validators/tmkms.md?plain=1#L1) for the listener.

## tm2/pkg/bft/privval/signer/vault/config.go:47 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/signer/vault/config.go#L47)
Nit: `TestConfig` returns `DefaultConfig()` verbatim and has no caller in the tree, since [`TestPrivValidatorConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L65-L67) reaches Vault through `DefaultPrivValidatorConfig`.

## tm2/pkg/bft/privval/config.go:124-126 [↗](../../../../../.worktrees/gno-review-5959/tm2/pkg/bft/privval/config.go#L124-L126)
Suggestion: this checks Vault against `RemoteSigner` and `TmkmsListener` only, and [#5956](https://github.com/gnolang/gno/pull/5956), [#5958](https://github.com/gnolang/gno/pull/5958), and [#5980](https://github.com/gnolang/gno/pull/5980) each add the same shape for their own backend. Merge two of them and a config enabling both Vault and AWS passes validation, [`NewSignerFromConfig`](https://github.com/gnolang/gno/blob/cc90c35f3/tm2/pkg/bft/privval/config.go#L140-L157) takes whichever branch comes first, and the duplicate `errMultipleSignerSourcesSet` declarations collide. A single list of enabled backends, rejected when longer than one, would scale across the four.
