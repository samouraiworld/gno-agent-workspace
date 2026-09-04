// Measures what the numeric half of a realm ID counts. The same realm path,
// deployed from two sources that differ only by declarations the ID path never
// reads, gets two different first IDs, so a token's identifier is a function of
// its realm's whole source rather than of its mint order.
//
/* Run: from a gno checkout:
gh pr checkout 6101 -R gnolang/gno && git checkout 911e1a57a
curl -fsSL -o gno.land/pkg/sdk/vm/realmid_source_sensitivity_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6101-realm-scoped-token-ids/1-911e1a57a/tests/realmid_source_sensitivity_test.go
go test -v -run 'TestRealmIDTracksRealmSource' ./gno.land/pkg/sdk/vm/
rm gno.land/pkg/sdk/vm/realmid_source_sensitivity_test.go
*/
package vm

import (
	"testing"

	"github.com/gnolang/gno/gnovm/pkg/gnolang"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/stretchr/testify/require"
)

func TestRealmIDTracksRealmSource(t *testing.T) {
	const pkgPath = "gno.land/r/test/idsource"

	// The two sources mint the same first ID at the same point of init. They
	// differ only in declarations no ID call reads.
	const lean = `package idsource

import "chain/runtime"

var First string

func init(cur realm) {
	First = runtime.NewRealmID()
}

func ID(cur realm) string { return First }`

	const padded = `package idsource

import "chain/runtime"

type filler struct{ a, b, c string }

var (
	pad1 = &filler{"a", "b", "c"}
	pad2 = &filler{"d", "e", "f"}
	pad3 = []*filler{{"g", "h", "i"}}

	First string
)

func init(cur realm) {
	First = runtime.NewRealmID()
}

func ID(cur realm) string { return First }`

	deploy := func(t *testing.T, body string) string {
		t.Helper()
		env := setupTestEnv()
		addr := crypto.AddressFromPreimage([]byte("realm-id-source"))
		ctx := env.vmk.MakeGnoTransactionStore(env.ctx)
		acc := env.acck.NewAccountWithAddress(ctx, addr)
		env.acck.SetAccount(ctx, acc)
		require.NoError(t, env.bankk.SetCoins(ctx, addr, initialBalance))
		require.NoError(t, env.vmk.AddPackage(ctx, NewMsgAddPackage(addr, pkgPath, []*std.MemFile{
			{Name: "gnomod.toml", Body: gnolang.GenGnoModLatest(pkgPath)},
			{Name: "idsource.gno", Body: body},
		})))
		env.vmk.CommitGnoTransactionStore(ctx)
		callCtx := env.vmk.MakeGnoTransactionStore(env.ctx)
		res, err := env.vmk.Call(callCtx, NewMsgCall(addr, nil, pkgPath, "ID", nil))
		require.NoError(t, err)
		return res
	}

	leanID := deploy(t, lean)
	paddedID := deploy(t, padded)
	t.Log("lean  :", leanID)
	t.Log("padded:", paddedID)

	// IS:     the first ID a realm mints moves with unrelated declarations.
	require.NotEqual(t, leanID, paddedID)
	// SHOULD: the same realm path minting its first token gets the same ID.
	// require.Equal(t, leanID, paddedID)
}
