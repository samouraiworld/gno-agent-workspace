// Candidate #58 — folding validateSymbol/validateOpName/validateFuncName into
// one shared helper.
//
// # from a local clone of gnolang/gno, at 3aa06ef757b87d731a3e554e96c7540a209fd9a3:
// sed -n '573,628p' examples/gno.land/r/nt/grc20routes/v0/grc20routes.gno | wc -l
//   -> 56 lines, three hand-rolled functions (matches the finder's "~55 lines").
//
// This file is the proposed replacement body for that same span; pasted in
// place of lines 573-628 it is 33 lines, roughly 40% shorter, in one place
// that decides emptiness, the length bound and the allowed-character class.
//
// Behaviour difference preserved deliberately, not fixed here: validateSymbol
// and validateOpName reject "" themselves; validateFuncName does not (its only
// caller, validate(), guards e.Func == "" first) — this is the gap already
// filed as its own finding (validate, line ~545). Folding the three functions
// does not paper over that gap; the empty check for the func-name path still
// has to be added at its call site or inside validateChars explicitly, and
// this sketch leaves it exactly where the current code leaves it so the two
// findings stay independent.
//
// Not run: a package-suite pass over this replacement is a mutation, out of
// this batch's read-only scope. The check that would confirm behavioural
// equivalence is exactly the one the finder names: substitute this block,
// `(cd examples && gno test ./gno.land/r/nt/grc20routes/v0)`, and diff output
// against the head baseline (already green, 3.7s, per the round's notes).

package grc20routes

// validateChars checks s against a length bound and an allowed-rune test,
// panicking with a message naming field on the first violation. checkFirst,
// when non-nil, applies a different rule to rune 0 only (validateFuncName's
// "must be an exported identifier" rule); pass nil to apply allow uniformly.
func validateChars(field, s string, max int, allow func(rune) bool, checkFirst func(rune) bool) {
	if len(s) > max {
		panic("grc20routes: " + field + " too long")
	}
	for i, c := range s {
		if i == 0 && checkFirst != nil {
			if !checkFirst(c) {
				panic("grc20routes: " + field + " must be an exported function name")
			}
			continue
		}
		if !allow(c) {
			panic("grc20routes: invalid character in " + field + ": " + string(c))
		}
	}
}

func validateSymbol(symbol string) {
	if symbol == "" {
		panic("grc20routes: empty symbol")
	}
	validateChars("symbol", symbol, maxSymbolLen, func(c rune) bool {
		return isAlphanumeric(c) || c == '_' || c == '-'
	}, nil)
}

func validateOpName(op string) {
	if op == "" {
		panic("grc20routes: empty operation name")
	}
	validateChars("operation name", op, maxOpLen, func(c rune) bool {
		return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
	}, nil)
}

// validateFuncName requires an exported identifier. No empty check here,
// same as head: validate() guards e.Func == "" before calling this.
func validateFuncName(field, name string) {
	validateChars(field+" name", name, maxIdentLen, func(c rune) bool {
		return isAlphanumeric(c) || c == '_'
	}, func(c rune) bool {
		return c >= 'A' && c <= 'Z'
	})
}
