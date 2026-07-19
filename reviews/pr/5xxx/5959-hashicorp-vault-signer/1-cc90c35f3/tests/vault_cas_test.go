/* Run: from a gno checkout:
gh pr checkout 5959 -R gnolang/gno && git checkout cc90c35f3
go get github.com/hashicorp/vault/api
curl -fsSL -o tm2/pkg/bft/privval/signer/vault/vault_cas_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5959-hashicorp-vault-signer/1-cc90c35f3/tests/vault_cas_test.go
go test -v -run 'TestCreateAndStoreKey_DoesNotOverwriteExistingSecret' ./tm2/pkg/bft/privval/signer/vault/
rm tm2/pkg/bft/privval/signer/vault/vault_cas_test.go && git checkout go.mod go.sum
*/

// createAndStoreKey calls Put with no KVOption, so Vault writes a new version
// unconditionally. casKV below enforces Vault's own check-and-set semantics:
// a write carrying cas=0 is rejected when the path already holds a version.
// At cc90c35f3 the test fails, the pre-existing validator key is replaced. With
// a cas=0 option on the write it passes, the existing key survives.

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

// errCheckAndSetFailed mirrors the 400 Vault returns when a cas guard loses.
var errCheckAndSetFailed = errors.New("check-and-set parameter did not match the current version")

// casKV is a mock kvAPI honouring the cas option the real KV v2 engine honours.
type casKV struct {
	secrets map[string]map[string]any
}

func (m *casKV) Get(_ context.Context, secretPath string) (*vaultapi.KVSecret, error) {
	data, ok := m.secrets[secretPath]
	if !ok {
		return nil, vaultapi.ErrSecretNotFound
	}

	return &vaultapi.KVSecret{Data: data}, nil
}

func (m *casKV) Put(
	_ context.Context,
	secretPath string,
	data map[string]any,
	opts ...vaultapi.KVOption,
) (*vaultapi.KVSecret, error) {
	_, exists := m.secrets[secretPath]

	for _, opt := range opts {
		k, v := opt()
		if k == vaultapi.KVOptionCheckAndSet && v == 0 && exists {
			return nil, errCheckAndSetFailed
		}
	}

	m.secrets[secretPath] = data

	return &vaultapi.KVSecret{Data: data}, nil
}

func TestCreateAndStoreKey_DoesNotOverwriteExistingSecret(t *testing.T) {
	t.Parallel()

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
