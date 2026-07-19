/* Run: from a gno checkout:
gh pr checkout 5980 -R gnolang/gno && git checkout c9aaed5c8
go get github.com/certusone/yubihsm-go@v0.3.0
curl -fsSL -o tm2/pkg/bft/privval/privval_config_yubihsm_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5980-yubihsm2-backed-signer/1-c9aaed5c8/tests/privval_config_yubihsm_test.go
go test -v -run 'TestPrivValidatorConfig_YubiHSM' ./tm2/pkg/bft/privval/
rm tm2/pkg/bft/privval/privval_config_yubihsm_test.go
git checkout go.mod go.sum
*/

// The nil-config and mutual-exclusion rules the yubihsm section adds to
// PrivValidatorConfig.ValidateBasic and NewPrivValidatorFromConfig have no
// coverage at c9aaed5c8; the equivalent remote-signer and tmkms rules do
// (errNilRemoteSignerConfig, errBothExternalSignersEnabled in config_test.go).
// These cases pass at c9aaed5c8 and pin the behavior against later edits.

package privval

import (
	"log/slog"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/bft/privval/signer/yubihsm"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enabledYubiHSMConfig() *yubihsm.Config {
	return &yubihsm.Config{
		ConnectorURL: "127.0.0.1:12345",
		AuthKeyID:    1,
		KeyID:        2,
	}
}

func TestPrivValidatorConfig_YubiHSMValidateBasic(t *testing.T) {
	t.Parallel()

	t.Run("nil yubihsm config", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = nil

		assert.ErrorIs(t, cfg.ValidateBasic(), errNilYubiHSMConfig)
	})

	t.Run("yubihsm with remote signer", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.RemoteSigner.ServerAddress = "tcp://127.0.0.1:26659"

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm with tmkms listener", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.TmkmsListener.ListenAddr = "unix:///tmp/tmkms-yubihsm-test.sock"
		cfg.TmkmsListener.ChainID = "test-chain"

		assert.ErrorIs(t, cfg.ValidateBasic(), errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm alone", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()

		require.NoError(t, cfg.ValidateBasic())
	})
}

func TestPrivValidatorConfig_YubiHSMNewPrivValidator(t *testing.T) {
	t.Parallel()

	privKey := ed25519.GenPrivKey()
	logger := slog.New(slog.DiscardHandler)

	t.Run("yubihsm with remote signer", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.RemoteSigner.ServerAddress = "tcp://127.0.0.1:26659"

		_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})

	t.Run("yubihsm with tmkms listener", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultPrivValidatorConfig()
		cfg.YubiHSM = enabledYubiHSMConfig()
		cfg.TmkmsListener.ListenAddr = "unix:///tmp/tmkms-yubihsm-test2.sock"
		cfg.TmkmsListener.ChainID = "test-chain"

		_, err := NewPrivValidatorFromConfig(cfg, privKey, logger)
		assert.ErrorIs(t, err, errMultipleSignerSourcesSet)
	})
}
