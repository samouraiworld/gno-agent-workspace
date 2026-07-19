/* Run: from a gno checkout:
gh pr checkout 5958 -R gnolang/gno && git checkout f2e427a71
GOFLAGS=-mod=mod go get ./tm2/pkg/bft/privval/signer/gcpsecretmanager
curl -fsSL -o tm2/pkg/bft/privval/privval_config_review_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5958-gcp-secret-manager-signer/1-f2e427a71/tests/privval_config_review_test.go
go test -v -run 'TestHalfConfiguredGCPIsRejected' ./tm2/pkg/bft/privval/
rm tm2/pkg/bft/privval/privval_config_review_test.go
git checkout HEAD -- go.mod go.sum
*/

// IsEnabled requires both ProjectID and SecretID, and Config.ValidateBasic returns
// nil unconditionally, so a block carrying only one of them validates and
// NewSignerFromConfig falls through to LoadOrMakeLocalSigner. At f2e427a71 the test
// fails on both assertions and reports the freshly minted local signer address.

package privval

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/gcpsecretmanager"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHalfConfiguredGCPIsRejected(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	cfg := DefaultPrivValidatorConfig()
	cfg.RootDir = root
	cfg.GCPSecretManager = &gcpsecretmanager.Config{
		ProjectID: "my-project",
		SecretID:  "", // an operator typo, or a template variable that did not expand
		Version:   "latest",
	}

	assert.Error(t, cfg.ValidateBasic(), "a partially filled gcp block should be rejected")

	signer, err := NewSignerFromConfig(
		context.Background(), cfg, ed25519.GenPrivKey(), slog.New(slog.DiscardHandler),
	)
	require.NoError(t, err)

	keyPath := filepath.Join(root, "priv_validator_key.json")
	_, statErr := os.Stat(keyPath)
	assert.True(t, os.IsNotExist(statErr),
		"no fresh validator key should be minted on disk, got signer %s", signer)
}
