/* Run: from a gno checkout:
gh pr checkout 5956 -R gnolang/gno && git checkout 1ad3009b9
go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/service/secretsmanager
curl -fsSL -o tm2/pkg/bft/privval/config_awssecretsmanager_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5956-aws-secrets-manager-signer/1-1ad3009b9/tests/config_awssecretsmanager_test.go
go test -v -run 'TestValidateBasicAWSSecretsManager|TestNewPrivValidatorFromConfigAWSSecretsManager' ./tm2/pkg/bft/privval/
rm tm2/pkg/bft/privval/config_awssecretsmanager_test.go && git checkout go.mod go.sum
*/

// The PR adds errNilAWSSecretsManagerCfg and errMultipleSignerSourcesSet to both
// ValidateBasic and NewPrivValidatorFromConfig, with no test in the privval package.
// Every earlier signer-exclusion rule has a matching subtest in config_test.go.

package privval

import (
	"encoding/hex"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/awssecretsmanager"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
