// IsKeyedRealm (grc20routes.gno:456) looks its argument up in provenKeyed raw,
// while every writer and every other reader host-strips through hostOf, so the
// pkgpath half of a "<host>#<sub>.<SYM>" registry key answers false while Get
// on that same key answers SourceProven. Measured at head 3aa06ef75: Get =
// "proven", IsKeyedRealm("gno.land/r/nt/grc20routes/v0#vault") = false.
// The post-fix assertion is the commented uassert.True below.
//
// # from a local clone of github.com/gnolang/gno:
//
//	git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//	cp 35-iskeyedrealm-ignores-subpath.gno \
//	  examples/gno.land/r/nt/grc20routes/v0/zz_iskeyedrealm_sub_test.gno
//	cd examples
//	gno test -v -run 'TestIsKeyedRealmIgnoresSubPath' ./gno.land/r/nt/grc20routes/v0
//	# then the fix, and the same run goes red on the active line:
//	#   sed -i 's/return provenKeyed.Has(pkgPath) }/return provenKeyed.Has(hostOf(pkgPath)) }/' \
//	#     gno.land/r/nt/grc20routes/v0/grc20routes.gno
//	rm gno.land/r/nt/grc20routes/v0/zz_iskeyedrealm_sub_test.gno

package grc20routes

import (
	"testing"

	"gno.land/p/nt/fqname/v0"
	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/seqid/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"

	"gno.land/r/nt/grc20reg/v0"
)

var (
	subNextID  seqid.ID
	subLedgers = map[string]*grc20.PrivateLedger{}
)

// mintUnderSub is mint, run under this realm's own cur.Sub("vault") identity.
// grc20reg keys on the raw cur.Previous().PkgPath(), so the key carries "#vault".
func mintUnderSub(_ int, rlm realm, symbol string) string {
	sub := rlm.Sub("vault")
	tok, led := grc20.NewToken("Sub "+symbol, symbol, 6, subNextID.Next(), sub)
	subLedgers[symbol] = led
	return grc20reg.Register(cross(sub), tok, "")
}

func approveUnderSub(symbol string, owner address, amount int64) {
	if err := subLedgers[symbol].Approve(owner, ProbeAddress(), amount); err != nil {
		panic(err)
	}
}

func TestIsKeyedRealmIgnoresSubPath(cur realm, t *testing.T) {
	keyA := mintUnderSub(0, cur, "SUBA")
	keyB := mintUnderSub(0, cur, "SUBB")
	urequire.Equal(t, selfPath+"#vault.SUBA", keyA, "grc20reg keys on the raw sub path")

	// A real keyed proof over the pair, exactly as TestKeyedProofCoversEveryTokenOnTheRealm.
	prover := testutils.TestAddress("subprover")
	testing.SetOriginCaller(prover)
	BeginKeyedProof(cross(cur), keyA, keyB, 777, 888)
	approveUnderSub("SUBA", prover, 777)
	approveUnderSub("SUBB", prover, 888)
	FinishKeyedProof(cross(cur), keyA, keyB)

	// The pkgpath half of the registry key, which is what a consumer holding the
	// key has before it asks this realm anything.
	rawPath, _ := fqname.Parse(keyA)
	urequire.Equal(t, selfPath+"#vault", rawPath)

	_, source := Get(keyA)
	uassert.Equal(t, SourceProven, source, "Get host-strips through mustToken")
	uassert.Equal(t, selfPath, PkgPath(keyA), "PkgPath host-strips too")

	// The pair. Both lines ask the same question about the same token.
	uassert.False(t, IsKeyedRealm(rawPath), "at head 3aa06ef75: raw lookup, disagrees with Get")
	// uassert.True(t, IsKeyedRealm(rawPath), "post-fix: IsKeyedRealm host-strips like every other reader")
}
