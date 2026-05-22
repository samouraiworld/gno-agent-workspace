/* Run:

   # Drop the adversarial filetest into the seal package's filetests/ dir.
   cp reviews/pr/5xxx/5704-lint-filetest-isolation/1-4ff865d10/tests/z_seal_violation_mismatched_directive_filetest.gno \
      .worktrees/gno-review-5704/examples/gno.land/p/test/seal/filetests/

   # Build the PR's gno binary once.
   go build -C .worktrees/gno-review-5704 -o /tmp/gno-5704-bin ./gnovm/cmd/gno

   cd .worktrees/gno-review-5704

   # (1) gno lint — silently absorbs the panic, exits 0, prints nothing.
   #     This is the regression surface: the directive's mere presence
   #     turns off lint's signal for any preprocess panic in the file.
   GNOROOT=$PWD /tmp/gno-5704-bin lint -C examples ./gno.land/p/test/seal/...
   echo "lint exit: $?"   # expect: 0   (BAD — should flag the mismatch)

   # (2) gno test — correctly catches the mismatch and prints a
   #     clean Expected/Actual diff. Exit code is currently 0 (a
   #     separate bug; the FAIL line is the meaningful signal).
   GNOROOT=$PWD /tmp/gno-5704-bin test ./examples/gno.land/p/test/seal/ 2>&1 | head -10
   # expect:
   #   --- FAIL: ./gno.land/p/test/seal/z_seal_violation_mismatched_directive_filetest.gno
   #   Error diff:
   #   - integer divide by zero
   #   + ...foreignImpl does not implement ...Sealed (missing method isSealed)

   # (3) Control: strip the `// Error:` block and re-run lint. With
   #     no directive present, lint correctly reports the panic and
   #     exits 1 — proving the absorption above is keyed solely off
   #     the directive's presence, not its content.
   sed -i '/^\/\/ Error:/,$d' \
     examples/gno.land/p/test/seal/filetests/z_seal_violation_mismatched_directive_filetest.gno
   GNOROOT=$PWD /tmp/gno-5704-bin lint -C examples ./gno.land/p/test/seal/...
   echo "lint exit: $?"   # expect: 1 with the seal-violation panic printed

   # Cleanup.
   cd ..
   rm .worktrees/gno-review-5704/examples/gno.land/p/test/seal/filetests/z_seal_violation_mismatched_directive_filetest.gno

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
