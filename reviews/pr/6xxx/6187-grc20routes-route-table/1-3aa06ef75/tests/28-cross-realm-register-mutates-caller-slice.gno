// from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cd examples
// mkdir -p gno.land/r/demo/zsort
// cat > gno.land/r/demo/zsort/gnomod.toml <<'GNO'
// module = "gno.land/r/demo/zsort"
// gno = "0.9"
// GNO
// cat > gno.land/r/demo/zsort/zsort.gno <<'GNO'
// package zsort
//
// import (
// 	"gno.land/p/nt/grc20/v0"
// 	"gno.land/p/nt/seqid/v0"
//
// 	"gno.land/r/nt/grc20reg/v0"
// 	"gno.land/r/nt/grc20routes/v0"
// )
//
// var (
// 	tok *grc20.Token
// 	led *grc20.PrivateLedger
// 	id  seqid.ID
//
// 	myRoute grc20routes.Route
// )
//
// func init(cur realm) {
// 	tok, led = grc20.NewToken("Sort", "SRT", 6, id.Next(), cur)
// 	grc20reg.Register(cross(cur), tok, "")
// }
//
// // A realistic extension: Canonical() gives [approve, transfer,
// // transfer_from] (already Op-sorted), then .With appends two ops whose
// // names sort before some of the canonical three, landing Funcs out of Op
// // order: [approve, transfer, transfer_from, deposit, withdraw].
// func DeclareOutOfOrder(cur realm) string {
// 	myRoute = grc20routes.Canonical().With("deposit", "Deposit").With("withdraw", "Withdraw")
// 	return grc20routes.Register(cross(cur), "SRT", myRoute)
// }
//
// func MyRouteOps() string {
// 	s := ""
// 	for _, e := range myRoute.Funcs {
// 		s += e.Op + ","
// 	}
// 	return s
// }
// GNO
// cat > gno.land/r/demo/zsort/zsort_test.gno <<'GNO'
// package zsort
//
// import (
// 	"testing"
//
// 	"gno.land/r/nt/grc20routes/v0"
// )
//
// func TestSortMutatesCallerSlice(cur realm, t *testing.T) {
// 	built := grc20routes.Canonical().With("deposit", "Deposit").With("withdraw", "Withdraw")
// 	before := ""
// 	for _, e := range built.Funcs {
// 		before += e.Op + ","
// 	}
// 	key := DeclareOutOfOrder(cross(cur))
// 	t.Log("key:", key)
// 	after := MyRouteOps()
// 	t.Log("before:", before)
// 	t.Log("after: ", after)
// 	if after == before {
// 		t.Fatal("caller's own slice unchanged: no aliasing observed")
// 	}
// }
// GNO
// gno test -v ./gno.land/r/demo/zsort
//
// Candidate #28 (grc20routes.gno:295): CONFIRMED on both halves.
//
// 1. Coverage gap: `grep -rn Register --include=*.txtar --include=*_test.gno`
//    over the tree shows Register called only from grc20routes_test.gno
//    itself (19 hits, all with cur.Previous().PkgPath() == selfPath by
//    construction) and from nowhere in
//    gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar, which
//    exercises only the proven path.
//
// 2. The regression that gap hides: sortEntries (grc20routes.gno:638) sorts
//    r.Funcs in place, and Register stores &r by reference. A route built by
//    a realistic .With() chain in a foreign realm and hand it to Register,
//    the caller's own local Route.Funcs is reordered as a side effect:
//
//      before: approve,transfer,transfer_from,deposit,withdraw,
//      after:  approve,deposit,transfer,transfer_from,withdraw,
//
//    The registering realm's own copy of the Route it built is silently
//    mutated by Register — surprising for a call whose signature promises a
//    value parameter, and undetectable by any test the suite ships since no
//    test calls Register from outside the package.
