// Repro, gnolang/gno#6273 at 8ee3be106: titleKey (listing.gno) folds only
// ASCII, Cyrillic and Greek look-alikes, and validText refuses a foreign
// script only next to Latin letters. Two shapes therefore list beside the
// title they copy, each accepted by upsert with reason "":
//   - a title written wholly in Lisu or Cherokee, scripts whose capitals are
//     drawn as Latin capitals (ADR T6's own "ACME" example);
//   - a non-letter look-alike inside a Latin word: a Myanmar or Devanagari
//     digit zero for "o", U+2223 DIVIDES for the "|" titleKey folds to "l".
//     Digits and symbols never set validText's foreign flag.
//
// From a plain clone of gnolang/gno:
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp b10-lines-reach-catalog-titlekey-confusables_test.gno \
//      examples/gno.land/r/gnoland/store/v0/zz_b10_confusables_test.gno
//   cd examples/gno.land/r/gnoland/store/v0 && gno test . -run TestB10Confusable -v
//
// Expect on the head: FAIL, every look-alike lists ("got reason \"\"").
package store

import (
	"chain/runtime"
	"testing"
)

func TestB10ConfusableTitlesList(cur realm, t *testing.T) {
	h := runtime.ChainHeight()
	list := func(path, slug, title string) string {
		l := newListing(path, slug, title, "A tagline", "utilities", 4, kindApp)
		l.Confirmed = true
		return upsert(l, h)
	}
	if r := list("gno.land/r/latin/acme", "latin-acme", "ACME"); r != "" {
		t.Fatalf("setup: %s", r)
	}
	cases := []struct{ label, original, twin string }{
		{"all Lisu", "ACME", "ꓮꓚꓟꓰ"},       // ꓮꓚꓟꓰ
		{"all Cherokee", "ACME", "ᎪᏟᎷᎬ"},   // ᎪᏟᎷᎬ
		{"myanmar zero", "Boards", "B၀ards"},             // seed title
		{"devanagari zero", "Gno.land Blog", "Gn०.land Blog"}, // seed title
		{"divides as l", "Gno.land Blog", "Gno.∣and Blog"},   // seed title
	}
	for i, tc := range cases {
		path := "gno.land/r/twin" + string(rune('a'+i)) + "/app"
		got := list(path, "twin-"+string(rune('a'+i)), tc.twin)
		t.Logf("%-16s validText=%v titleKey(%q)=%q titleKey(%q)=%q reason=%q",
			tc.label, validText(tc.twin, maxTitle), tc.original, titleKey(tc.original), tc.twin, titleKey(tc.twin), got)
		if got != reasonTitleTaken {
			t.Errorf("%s: %q lists beside %q: got reason %q, want %q", tc.label, tc.twin, tc.original, got, reasonTitleTaken)
		}
	}
}
