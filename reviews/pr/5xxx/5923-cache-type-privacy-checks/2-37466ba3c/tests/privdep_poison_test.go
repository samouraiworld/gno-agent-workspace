/* Run: from a gno checkout:
gh pr checkout 5923 -R gnolang/gno && git checkout 37466ba3c
curl -fsSL -o gnovm/pkg/gnolang/privdep_poison_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5923-cache-type-privacy-checks/2-37466ba3c/tests/privdep_poison_test.go
go test -v -run 'TestPrivacyVerdictFromDiscardedTransaction' ./gnovm/pkg/gnolang/
rm gnovm/pkg/gnolang/privdep_poison_test.go
*/

// typeHasPrivateDep writes into defaultStore.typePrivacyCache, which is shared
// by every BeginTransaction child and outlives the transaction that filled it.
// At 37466ba3c assertTypeIsPublic returns without panicking; at the merge base
// d14a03770 it panics with "cannot persist object of type defined in the
// private realm". Uses no symbol introduced by the PR, so it runs on both.

package gnolang

import (
	"io"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/db/memdb"
	"github.com/gnolang/gno/tm2/pkg/std"
	"github.com/gnolang/gno/tm2/pkg/store/dbadapter"
	storetypes "github.com/gnolang/gno/tm2/pkg/store/types"
)

const poisonPkgPath = "gno.land/r/x/secret"

func deployPoisonPkg(t *testing.T, tx TransactionStore, private bool) {
	t.Helper()
	mod := "module = \"" + poisonPkgPath + "\"\ngno = \"0.9\"\n"
	if private {
		mod += "private = true\n"
	}
	m := NewMachineWithOptions(MachineOptions{Store: tx, Output: io.Discard})
	m.RunMemPackage(&std.MemPackage{
		Type: MPUserProd, Name: "secret", Path: poisonPkgPath,
		Files: []*std.MemFile{
			{Name: "gnomod.toml", Body: mod},
			{Name: "secret.gno", Body: "package secret\n\ntype Token struct{ Amount int }\n\nvar Current = &Token{Amount: 1}\n"},
		},
	}, true)
}

func TestPrivacyVerdictFromDiscardedTransaction(t *testing.T) {
	db := memdb.NewMemDB()
	base := dbadapter.StoreConstructor(db, storetypes.StoreOptions{})
	st := NewStore(nil, base, base)

	// A transaction deploys the path without private=true, then is thrown
	// away: no tx.Write(), no cache-wrap Write(), nothing reaches the base
	// store. This is what simulate mode and a failed DeliverTx both do.
	w1 := base.CacheWrap()
	deployPoisonPkg(t, st.BeginTransaction(w1, w1, nil, nil), false)

	// The deployment that actually lands declares the package private.
	w2 := base.CacheWrap()
	tx2 := st.BeginTransaction(w2, w2, nil, nil)
	deployPoisonPkg(t, tx2, true)
	tx2.Write()
	w2.Write()

	if pv := tx2.GetPackage(poisonPkgPath, false); !pv.Private {
		t.Fatalf("precondition failed: package is not private")
	}
	dt := tx2.GetType(DeclaredTypeID(poisonPkgPath, Location{}, "Token")).(*DeclaredType)

	// Another realm persisting a value of that type must be rejected.
	rlm := NewRealm("gno.land/r/x/other")
	defer func() {
		if recover() == nil {
			t.Fatalf("assertTypeIsPublic accepted %s.Token from another realm", poisonPkgPath)
		}
	}()
	rlm.assertTypeIsPublic(tx2, dt, map[TypeID]struct{}{})
}
