package store

// b1-claims: ADR-004 T6 says titles are unique on a folded key where
// "Cyrillic and Greek letters that look Latin -> that letter", and that the
// realm refuses "mathematical alphanumerics". Each check below asserts one
// of those claims; a failure is the claim not holding at head 8ee3be106.
//
// Repro from a plain clone of gnolang/gno:
//   git fetch origin pull/6273/head && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//   cp b1-claims-greek-capital-fold.gno examples/gno.land/r/gnoland/store/v0/b1_claims_fold_test.gno
//   cd examples/gno.land/r/gnoland/store/v0 && gno test -v -run TestB1 .

import (
	"testing"
)

// All-Greek capitals drawn exactly like the Latin word on the left.
var greekTwins = [][2]string{
	{"TOKEN", "ΤΟΚΕΝ"}, // Tau Omicron Kappa Epsilon Nu
	{"ZONE", "ΖΟΝΕ"},        // Zeta Omicron Nu Epsilon
	{"MINT", "ΜΙΝΤ"},        // Mu Iota Nu Tau
	{"KEY", "ΚΕΥ"},               // Kappa Epsilon Upsilon
	{"BOX", "ΒΟΧ"},               // control: every letter folds right
}

func TestB1GreekCapitalsFold(t *testing.T) {
	for _, p := range greekTwins {
		if !validText(p[1], maxTitle) {
			t.Logf("%s: refused by validText, no impersonation", p[0])
			continue
		}
		if titleKey(p[0]) != titleKey(p[1]) {
			t.Errorf("%s: Greek twin accepted with a distinct key: %q vs %q", p[0], titleKey(p[0]), titleKey(p[1]))
		}
	}
}

func TestB1GreekTwinRegisters(cur realm, t *testing.T) {
	registerFrom(cur, "gno.land/r/acme/token", "acme-token", "TOKEN")
	registerFrom(cur, "gno.land/r/mallory/token", "mallory-token", "ΤΟΚΕΝ")
	if lookup(&listings, "mallory-token") != nil {
		t.Errorf("a second app took a title drawn as TOKEN: %q", lookup(&listings, "mallory-token").Title)
	}
}

// U+2102..U+2149 hold the double-struck, script and fraktur letters the
// Mathematical Alphanumeric Symbols block (U+1D400..) leaves as holes.
func TestB1LetterlikeMathLetters(t *testing.T) {
	for _, s := range []string{
		"ℤℴℴℳ", // script/double-struck "Zoom"
		"ℂℍℕ",       // double-struck "CHN"
	} {
		if validText(s, maxTitle) {
			t.Errorf("%q accepted though T6 says mathematical alphanumerics are refused", s)
		}
	}
	// control: the block isMimic covers is refused
	if validText("\U0001D400\U0001D401", maxTitle) {
		t.Errorf("control failed: U+1D400 accepted")
	}
}
