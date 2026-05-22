// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5704 -R gnolang/gno
# (save this file at examples/gno.land/p/test/seal/filetests/z_seal_violation_mismatched_directive_filetest.gno)
DEST=examples/gno.land/p/test/seal/filetests/z_seal_violation_mismatched_directive_filetest.gno

# (1) lint silently absorbs the (unrelated) preprocess panic because a directive is present.
go run ./gnovm/cmd/gno lint -C examples ./gno.land/p/test/seal/... ; echo "lint exit: $?"   # expect: 0 (BAD)

# (2) gno test catches the directive/panic mismatch.
go run ./gnovm/cmd/gno test ./examples/gno.land/p/test/seal/ 2>&1 | head -10
# expect: FAIL with Error diff between "integer divide by zero" and the seal-violation panic

# (3) Control: strip the `// Error:` block; lint then exits 1 with the panic printed, proving (1) is keyed off directive presence.
sed -i '/^\/\/ Error:/,$d' "$DEST"
go run ./gnovm/cmd/gno lint -C examples ./gno.land/p/test/seal/... ; echo "lint exit: $?"   # expect: 1

rm "$DEST"
*/
// PKGPATH: gno.land/r/test/sealviolation_mismatch
package sealviolation_mismatch

// Adversarial filetest demonstrating PR #5704's silent-absorption gap.
//
// The body triggers a real preprocess panic (sealed-interface
// violation: "missing method isSealed"). The `// Error:` directive
// claims a totally unrelated runtime panic ("integer divide by zero").
//
// Observed:
//   gno test  → FAIL with a clean Expected/Actual diff (directive
//               mismatch caught). This is the correct behavior.
//   gno lint  → exit 0, no output. The mismatch is silently
//               absorbed because lint's `expectsErr` check only
//               asks "is any Error directive present?", not "does
//               the panic message match the directive?".
//
// Removing the `// Error:` block below restores lint's signal
// (exits 1, prints the seal-violation panic), proving the
// absorption is keyed off the directive's mere presence.

import (
	"gno.land/p/test/seal"
)

type foreignImpl struct{}

func (f *foreignImpl) Hello() string { return "evil" }
func (f *foreignImpl) isSealed()     {}

func main() {
	var s seal.Sealed = &foreignImpl{} // preprocess panics here.
	println(s.Hello())
}

// Error:
// integer divide by zero
