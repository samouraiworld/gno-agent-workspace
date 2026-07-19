/* Run: from a gno checkout:
gh pr checkout 5958 -R gnolang/gno && git checkout f2e427a71
GOFLAGS=-mod=mod go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager
curl -fsSL -o tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_review_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5958-gcp-secret-manager-signer/1-f2e427a71/tests/gcpsecretmanager_review_test.go
go test -v -run 'TestSecretExistsWithoutVersion|TestCreateIfMissingWithPinnedVersion' ./tm2/pkg/bft/privval/signer/gcpsecretmanager/
rm tm2/pkg/bft/privval/signer/gcpsecretmanager/gcpsecretmanager_review_test.go
git checkout HEAD -- go.mod go.sum
*/

// createAndStoreKey treats CreateSecret as always-new, so it aborts whenever the
// secret already exists without a readable version, and it writes version 1 while
// versionName reads whatever Version pins. At f2e427a71 both tests fail; they pass
// once CreateSecret tolerates AlreadyExists and ValidateBasic rejects a pinned
// Version together with CreateIfMissing.

package gcpsecretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A secret that exists but carries no version answers AccessSecretVersion with
// NotFound, so createAndStoreKey runs and CreateSecret answers AlreadyExists.
func TestSecretExistsWithoutVersion(t *testing.T) {
	t.Parallel()

	client := newMockClient()
	client.createErr = status.Error(codes.AlreadyExists, "Secret already exists")

	cfg := &Config{ProjectID: "p", SecretID: "s", Version: "latest", CreateIfMissing: true}

	signer, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err, "an existing but empty secret should receive a first version")
	require.NotNil(t, signer)

	_, ok := client.secrets[cfg.versionName()]
	assert.True(t, ok, "the generated key should read back at the configured version")
}

// CreateIfMissing writes version 1 while versionName reads the pinned version.
func TestCreateIfMissingWithPinnedVersion(t *testing.T) {
	t.Parallel()

	client := newMockClient()
	cfg := &Config{ProjectID: "p", SecretID: "s", Version: "5", CreateIfMissing: true}

	require.Error(t, cfg.ValidateBasic(), "a pinned version cannot be satisfied by a fresh secret")

	first, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err)

	// Real GCP answers AlreadyExists once the secret has been created.
	client.createErr = status.Error(codes.AlreadyExists, "Secret already exists")

	second, err := newSigner(context.Background(), client, cfg)
	require.NoError(t, err, "a restart must find the key the first boot minted")
	assert.True(t, second.PubKey().Equals(first.PubKey()), "a restart must keep the same identity")
}
