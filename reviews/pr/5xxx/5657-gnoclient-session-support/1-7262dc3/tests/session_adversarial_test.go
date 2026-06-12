// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5657 -R gnolang/gno && git checkout 7262dc3c5
curl -fsSL -o gno.land/pkg/gnoclient/session_adversarial_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5657-gnoclient-session-support/1-7262dc3/tests/session_adversarial_test.go
go test -v -run 'TestSession(SignTxMultiSigner|SignerValidate)' ./gno.land/pkg/gnoclient/
rm gno.land/pkg/gnoclient/session_adversarial_test.go
*/
// SignTx's SessionAddr loop calls Signature.PubKey.Address() on every signature;
// SignerFromKeybase.Sign leaves non-signer slots with a nil PubKey, so a session
// signer plus a second tx signer panics. SignerFromKeybase.Validate signs a blank
// MsgCall whose Caller is the session address while Sign searches for the master
// address, so Validate always errors for session signers. Both tests assert the
// fixed behavior: they fail at the pinned hash and pass once the bugs are fixed.
package gnoclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/tm2/pkg/crypto"
)

// TestSessionSignTxMultiSigner: a session signer signing a tx that has a second
// signer must not panic; SignTx should set SessionAddr only on the signature it
// produced and leave the other signer's empty slot alone.
func TestSessionSignTxMultiSigner(t *testing.T) {
	masterSigner := newInMemorySigner(t, "tendermint_test")
	masterInfo, err := masterSigner.Info()
	require.NoError(t, err)

	signer := newInMemorySessionSigner(t, "tendermint_test", masterInfo.GetAddress())
	signerInfo, err := signer.Info()
	require.NoError(t, err)

	client := Client{Signer: signer} // no RPCClient: account and sequence numbers are explicit below

	otherAddr, err := crypto.AddressFromBech32("g14a0y9a64dugh3l7hneshdxr4w0rfkkww9ls35p")
	require.NoError(t, err)

	tx, err := NewCallTx(BaseTxCfg{
		GasFee:    ugnot.ValueString(2100000),
		GasWanted: 50000000,
	},
		vm.MsgCall{Caller: masterInfo.GetAddress(), PkgPath: "gno.land/r/demo/deep", Func: "Render", Args: []string{""}},
		vm.MsgCall{Caller: otherAddr, PkgPath: "gno.land/r/demo/deep", Func: "Render", Args: []string{""}},
	)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		signedTx, err := client.SignTx(*tx, 1, 1) // non-zero numbers skip the account query
		require.NoError(t, err)
		require.Len(t, signedTx.Signatures, 2)
		assert.Equal(t, signerInfo.GetAddress(), signedTx.Signatures[0].SessionAddr)
		assert.Nil(t, signedTx.Signatures[1].PubKey) // second signer still unsigned, untouched
	})
}

// TestSessionSignerValidate: a correctly configured session signer must pass
// its own Validate check.
func TestSessionSignerValidate(t *testing.T) {
	masterSigner := newInMemorySigner(t, "tendermint_test")
	masterInfo, err := masterSigner.Info()
	require.NoError(t, err)

	signer := newInMemorySessionSigner(t, "tendermint_test", masterInfo.GetAddress())
	assert.NoError(t, signer.Validate())
}
