// Routes a real "<host>#<sub>.<SYM>" registry key through the public surface
// (PkgPath, JSON, Get), which no test in the package does: the suite's own
// TestDestinationStripsSubRealmPath calls the unexported hostOf with literals.
// Passes at 3aa06ef75; reddens the moment mustToken stops stripping, which the
// shipped suite and grc20routes_keyed_proof.txtar both survive green.
//
// # from a local clone of github.com/gnolang/gno:
//
//	git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//	cp 19-subrealm-route-host.gno \
//	  examples/gno.land/r/nt/grc20routes/v0/zz_subrealm_route_test.gno
//	cd examples
//	gno test -v -run 'TestSubRealmTokenRoutesToCallableHost' ./gno.land/r/nt/grc20routes/v0
//	# ok. Now drop the strip the PR body headlines:
//	sed -i 's|^\treturn hostOf(pkgPath)$|\treturn pkgPath|' \
//	  gno.land/r/nt/grc20routes/v0/grc20routes.gno
//	gno test -v -run 'TestSubRealmTokenRoutesToCallableHost' ./gno.land/r/nt/grc20routes/v0
//	# FAILS here, while the shipped suite (rm this file first) stays ok.
//	git checkout -- gno.land/r/nt/grc20routes/v0/grc20routes.gno
//	rm gno.land/r/nt/grc20routes/v0/zz_subrealm_route_test.gno

package grc20routes

import (
	"strings"
	"testing"

	"gno.land/p/nt/grc20/v0"
	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"

	"gno.land/r/nt/grc20reg/v0"
)

func TestSubRealmTokenRoutesToCallableHost(cur realm, t *testing.T) {
	// grc20reg keys on the raw cur.Previous().PkgPath(), so a token minted
	// and registered while the realm runs under a sub identity keys with
	// the "#sub" attached.
	sub := cur.Sub("admin")
	tok, led := grc20.NewToken("Test SUBH", "SUBH", 6, nextID.Next(), sub)
	ledgers["SUBH"] = led
	key := grc20reg.Register(cross(sub), tok, "")
	urequire.Equal(t, selfPath+"#admin.SUBH", key)

	// "<host>#<sub>" is not a MsgCall destination; the host is.
	uassert.Equal(t, selfPath, PkgPath(key))
	uassert.True(t, strings.Contains(JSON(key), `"pkg_path":"`+selfPath+`"`),
		"JSON pkg_path must name the callable host, got "+JSON(key))

	// provenKeyed is keyed per host, so one proof run by the host realm has
	// to cover the token it registered under a sub identity.
	prover := testutils.TestAddress("subprover")
	keyA := mint(0, cur, "SUBA")
	keyB := mint(0, cur, "SUBB")
	testing.SetOriginCaller(prover)
	BeginKeyedProof(cross(cur), keyA, keyB, 901, 902)
	approveAs("SUBA", prover, 901)
	approveAs("SUBB", prover, 902)
	FinishKeyedProof(cross(cur), keyA, keyB)

	_, source := Get(key)
	uassert.Equal(t, SourceProven, source,
		"a proof on the host realm must cover its sub-registered tokens")
}
