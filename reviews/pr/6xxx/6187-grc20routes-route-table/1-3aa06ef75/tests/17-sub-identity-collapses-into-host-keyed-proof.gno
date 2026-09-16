// mustToken (grc20routes.gno:514-520) strips a token key's "#subpath" via
// hostOf before BeginKeyedProof's "both tokens must live in the same realm"
// check at :361. grc20reg.Register keys on the raw, un-stripped
// cur.Previous().PkgPath() (grc20reg.gno:39-40), so a token registered from a
// realm's own top-level frame and a token registered from that same realm's
// cur.Sub("x") pass the same-realm check as one realm. Proving the pair keyed
// then marks provenKeyed[host] — via Get's fold at :442-445 — for EVERY
// token hosted by that path, including the host's own top-level token, which
// this run never demonstrates routes by a leading symbol argument at all.
// Measured at head 3aa06ef75: KeyHost resolves source="proven",
// prefix=["HOST"] after a proof whose only Approve calls ran through the
// sub-identity's ledger, never the host's own entry point.
//
// # from a local clone of github.com/gnolang/gno:
//   git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//   mkdir -p examples/gno.land/r/demo/zsub6
//   printf 'module = "gno.land/r/demo/zsub6"\n\ngno = "0.9"\n' \
//     > examples/gno.land/r/demo/zsub6/gnomod.toml
//   cp 17-sub-identity-collapses-into-host-keyed-proof.gno \
//      examples/gno.land/r/demo/zsub6/main_filetest.gno
//   (cd examples && go run ../gnovm/cmd/gno test -v ./gno.land/r/demo/zsub6)
//   rm -rf examples/gno.land/r/demo/zsub6

// PKGPATH: gno.land/r/demo/zsub6
package zsub6

import (
	"gno.land/p/nt/grc20/v0"
	"gno.land/r/nt/grc20reg/v0"
	"gno.land/r/nt/grc20routes/v0"
)

var (
	hostLedger, subLedger     *grc20.PrivateLedger
	keyHost, keySub           string
)

func init(cur realm) {
	// KeyHost: minted and registered from this realm's own top-level frame.
	// It exposes no keyed Approve here — it stands in for a realm whose real
	// entry point takes no leading symbol at all.
	host, hl := grc20.NewToken("Host", "HOST", 4, 0, cur)
	hostLedger = hl
	keyHost = grc20reg.Register(cross(cur), host, "")

	// KeySub: minted and registered from this same realm's cur.Sub("x").
	sub := cur.Sub("x")
	subTok, sl := grc20.NewToken("Sub", "SUBT", 4, 1, sub)
	subLedger = sl
	keySub = grc20reg.Register(cross(sub), subTok, "")
}

func main(cur realm) {
	probe := grc20routes.ProbeAddress()
	prover := cur.Address() // BeginKeyedProof keys on cur.Previous().Address()

	// The two-key proof runs entirely through the sub identity's own ledger
	// object and this realm's, never through a keyed entry point on the host
	// token itself.
	grc20routes.BeginKeyedProof(cross(cur), keyHost, keySub, 111, 222)
	must(hostLedger.Approve(prover, probe, 111))
	must(subLedger.Approve(prover, probe, 222))
	grc20routes.FinishKeyedProof(cross(cur), keyHost, keySub)

	println("keyed(host)      :", grc20routes.IsKeyedRealm("gno.land/r/demo/zsub6"))
	rHost, sourceHost := grc20routes.Get(keyHost)
	println("route(host token):", sourceHost, rHost.Prefix)

	// The pair. Active: what the reviewed head does.
	if sourceHost == "proven" {
		println("HEAD: the host's own token is marked proven from a proof the host itself never satisfied alone")
	}
	// Post-fix, uncomment once the same-realm check distinguishes sub
	// identities from their host, or the proof requires both approvals from
	// the SAME identity:
	// if sourceHost == "proven" {
	// 	panic("host token should not inherit a proof its own identity did not join")
	// }
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// Output:
// keyed(host)      : true
// route(host token): proven slice[("HOST" string)]
// HEAD: the host's own token is marked proven from a proof the host itself never satisfied alone
