// from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
// sed -n '295,300p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//
// Candidate #33 (grc20routes.gno:297): CONFIRMED behaviorally.
// convertArgToGno (gno.land/pkg/sdk/vm/convert.go) panics on any non-
// primitive/array/slice arg type with "unexpected type in contract arg", so
// MsgCall can never build a call to Register(cur realm, symbol string, r
// Route) — Route is a struct. Every reachable caller is therefore already a
// realm, and cur.Previous().PkgPath() is never "". Deleting the guard keeps
// the whole package suite green:
//
//   sed -i '297,299d' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//   cd examples && gno test ./gno.land/r/nt/grc20routes/v0
//   # ok      ./gno.land/r/nt/grc20routes/v0 	3.19s
//
// Line counts: grc20routes.gno is 740 lines at head, 737 after this removal.
// This file ships the rewrite of the guarded block; it is documentation only
// (not a runnable package member) since the removal itself is the artifact.

// --- before (grc20routes.gno:295-299) ---
// func Register(cur realm, symbol string, r Route) string {
// 	rlmPath := cur.Previous().PkgPath()
// 	if rlmPath == "" {
// 		panic("grc20routes: caller is not a realm")
// 	}

// --- after (3 lines removed, guard was unreachable) ---
// func Register(cur realm, symbol string, r Route) string {
// 	rlmPath := cur.Previous().PkgPath()
