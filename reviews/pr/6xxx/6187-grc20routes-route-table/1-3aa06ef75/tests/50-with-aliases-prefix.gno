// Candidate #50 — Route.With copies Funcs but aliases Prefix.
//
// What it asserts: two Routes produced by calling .With() on the same base
// Route share one Prefix backing array, so writing through one's Prefix slot
// is visible from the other and from the base Route itself. Funcs does not
// have this problem: With appends into a fresh slice every call.
//
// Repro (read-only trace, from a plain clone):
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
//   sed -n '224,232p;263,280p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno
//
// Read output (the two relevant declarations):
//
//   type Route struct {
//   	Prefix []string
//   	Funcs []Entry
//   }
//   ...
//   func (r Route) With(op, fn string) Route {
//   	out := Route{Prefix: r.Prefix}                 // <-- shares r's backing array
//   	for _, e := range r.Funcs {
//   		if e.Op != op {
//   			out.Funcs = append(out.Funcs, e)       // fresh slice each call
//   		}
//   	}
//   	out.Funcs = append(out.Funcs, Entry{op, fn})
//   	return out
//   }
//
// `out.Prefix` is a plain assignment of `r.Prefix`, a slice header copy that
// keeps the same backing array. `out.Funcs`, by contrast, is built by append
// from a nil starting slice, so it is always a fresh array. The two exported
// fields of the same struct are not immutable the same way.
//
// A Prefix array is always len==1-or-0 today (Keyed sets one element, Canonical
// none) and With never appends to Prefix, so no in-package call ever grows or
// reallocates it — every Route derived by chained .With() calls off one Keyed()
// or Canonical() call keeps pointing at that one call's array. Register stores
// the *Route it is handed (grc20routes.gno ~290s, "declared.Set(key, &r)"-shaped),
// so a stored declaration and any sibling built from the same base share the
// array for the rest of the realm's life.
//
// Line counts: the buggy With is 8 source lines (267-274); the fixed version
// below is the same 8 lines, one changed.
//
// Fix — copy Prefix explicitly so out is independent of r:
//
//	func (r Route) With(op, fn string) Route {
//		out := Route{Prefix: append([]string(nil), r.Prefix...)}
//		for _, e := range r.Funcs {
//			if e.Op != op {
//				out.Funcs = append(out.Funcs, e)
//			}
//		}
//		out.Funcs = append(out.Funcs, Entry{op, fn})
//		return out
//	}
//
// Band: Nit. Nothing in the current package ever takes a mutable reference
// into a Route's Prefix and writes through it — Prefix is read-only after
// construction in every call site this diff ships — so the aliasing is latent:
// real for any future caller that keeps a []string alias to a Route's Prefix
// and mutates it (e.g. a caller building Prefix from a slice it still holds
// and later appends to it without a length check forcing a copy), not
// exploitable through anything exposed by this PR today.
package grc20routes_test
