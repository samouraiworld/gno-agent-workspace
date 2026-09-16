// from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// cd examples
// cat > gno.land/r/nt/grc20routes/v0/zzz_filetest.gno <<'GNO'
// package main
//
// import (
// 	"gno.land/r/nt/grc20routes/v0"
// )
//
// func main() {
// 	println(grc20routes.Render("gno.land/r/demo/nope.NOPE"))
// }
// GNO
// gno test -v ./gno.land/r/nt/grc20routes/v0
//
// Candidate #24 (grc20routes.gno:652): CONFIRMED behaviorally. grep -c Render
// grc20routes_test.gno is 0 — no test drives Render, renderToken, opTail or
// renderEntry (~90 lines, grc20routes.gno:652-740) — and Render on any
// unregistered token key panics rather than rendering a not-found page:
//
//   panic: grc20routes: unknown token: gno.land/r/demo/nope.NOPE
//   mustToken<VPBlock(3,38)>(tokenKey<VPBlock(1,0)>)
//       gno.land/r/nt/grc20routes/v0/grc20routes.gno:516
//   Get<VPBlock(3,33)>(tokenKey<VPBlock(1,0)>)
//       gno.land/r/nt/grc20routes/v0/grc20routes.gno:437
//   renderToken<VPBlock(4,47)>(path<VPBlock(2,0)>)
//       gno.land/r/nt/grc20routes/v0/grc20routes.gno:700
//   ref(gno.land/r/nt/grc20routes/v0).Render(gno.land/r/demo/nope.NOPE)
//       gno.land/r/nt/grc20routes/v0/grc20routes.gno:654
//
// A gnoweb request against a stale or mistyped link (e.g. after a token
// realm's redeploy, or a copy-pasted path with a typo) aborts the whole VM
// call instead of rendering "not found", and no test or txtar step catches
// it because none queries Render at all.
