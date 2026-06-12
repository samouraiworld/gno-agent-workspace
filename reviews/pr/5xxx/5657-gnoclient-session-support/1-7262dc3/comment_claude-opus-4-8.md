# Review: PR #5657
Event: REQUEST_CHANGES

## Body
Two defects on session paths the tests don't cover, reproduced on 7262dc3c5. The single-signer flow is correct and matches gnokey.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5657-gnoclient-session-support/1-7262dc3/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/gnoclient/client_txs.go:435 [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/client_txs.go#L435)
When the signer has a master set, SignTx calls `.PubKey.Address()` on every signature slot, but slots for other signers are still nil, so a session-signed tx with a second signer panics with a nil-pointer dereference. A single-signer tx has one slot and is fine. Fix: skip slots with a nil PubKey, or match the slot by the master signer address before reading `.Address()`.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5657 -R gnolang/gno
cat > gno.land/pkg/gnoclient/session_repro_test.go <<'EOF'
package gnoclient

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/gno.land/pkg/gnoland/ugnot"
	"github.com/gnolang/gno/gno.land/pkg/sdk/vm"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/keys"
)

func sessionSigner(t *testing.T, master crypto.Address) *SignerFromKeybase {
	t.Helper()
	kb := keys.NewInMemory()
	_, err := kb.CreateAccount("user1",
		"mention vintage immense fix clerk state magnet embrace meadow buzz captain bar mystery decade mammal rib chunk upset finish athlete maple undo space palace",
		"", "", 0, 0)
	require.NoError(t, err)
	return &SignerFromKeybase{Keybase: kb, Account: "user1", ChainID: "tendermint_test", Master: master}
}

func TestReproMultiSignerPanic(t *testing.T) {
	master, err := SignerFromBip39("source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast", "tendermint_test", "", 0, 0)
	require.NoError(t, err)
	masterInfo, err := master.Info()
	require.NoError(t, err)

	signer := sessionSigner(t, masterInfo.GetAddress())
	client := Client{Signer: signer} // no RPCClient; explicit account/sequence below

	other, err := crypto.AddressFromBech32("g14a0y9a64dugh3l7hneshdxr4w0rfkkww9ls35p")
	require.NoError(t, err)

	tx, err := NewCallTx(BaseTxCfg{GasFee: ugnot.ValueString(2100000), GasWanted: 50000000},
		vm.MsgCall{Caller: masterInfo.GetAddress(), PkgPath: "gno.land/r/demo/deep", Func: "Render", Args: []string{""}},
		vm.MsgCall{Caller: other, PkgPath: "gno.land/r/demo/deep", Func: "Render", Args: []string{""}},
	)
	require.NoError(t, err)

	require.NotPanics(t, func() { client.SignTx(*tx, 1, 1) })
}
EOF
go test -run TestReproMultiSignerPanic ./gno.land/pkg/gnoclient/
rm gno.land/pkg/gnoclient/session_repro_test.go
```

```
--- FAIL: TestReproMultiSignerPanic (0.03s)
        	Panic value:	runtime error: invalid memory address or nil pointer dereference
        	.../gno.land/pkg/gnoclient/client_txs.go:435 +0x4b9
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoclient	0.089s
```
</details>

*(AI Agent)*

## gno.land/pkg/gnoclient/signer.go:48-49 [↗](../../../../../.worktrees/gno-review-5657/gno.land/pkg/gnoclient/signer.go#L48)
`Validate` signs a probe MsgCall whose Caller is the session address, but when a master is set `Sign` searches the signer set for the master address, which isn't there, so `Validate` always returns "not in signer set" for a session signer. Fix: set the probe MsgCall Caller to the master address when Master is non-zero.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5657 -R gnolang/gno
cat > gno.land/pkg/gnoclient/session_repro_test.go <<'EOF'
package gnoclient

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/keys"
)

func TestReproSessionValidate(t *testing.T) {
	master, err := SignerFromBip39("source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast", "tendermint_test", "", 0, 0)
	require.NoError(t, err)
	masterInfo, err := master.Info()
	require.NoError(t, err)

	kb := keys.NewInMemory()
	_, err = kb.CreateAccount("user1",
		"mention vintage immense fix clerk state magnet embrace meadow buzz captain bar mystery decade mammal rib chunk upset finish athlete maple undo space palace",
		"", "", 0, 0)
	require.NoError(t, err)
	signer := &SignerFromKeybase{Keybase: kb, Account: "user1", ChainID: "tendermint_test", Master: masterInfo.GetAddress()}
	_ = crypto.Address{}

	require.NoError(t, signer.Validate())
}
EOF
go test -run TestReproSessionValidate ./gno.land/pkg/gnoclient/
rm gno.land/pkg/gnoclient/session_repro_test.go
```

```
--- FAIL: TestReproSessionValidate (0.01s)
        	Received unexpected error:
        	address g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5 (user1) not in signer set
FAIL	github.com/gnolang/gno/gno.land/pkg/gnoclient	0.037s
```
</details>

*(AI Agent)*
</content>
