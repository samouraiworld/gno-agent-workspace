/* Run:
gh pr checkout 5890 -R gnolang/gno && git checkout b940037d1
curl -fsSL -o gnovm/tests/files/zrealm_sub_mirror_vector.gno https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5890-realm-sub-subrealm-identities/2-b940037d1/tests/zrealm_sub_mirror_vector.gno
go test ./gnovm/pkg/gnolang/ -run 'TestFiles/zrealm_sub_mirror_vector.gno$' -v
rm gnovm/tests/files/zrealm_sub_mirror_vector.gno
*/

// PKGPATH: gno.land/r/demo/submirror
package submirror

// Drives ONE input vector (TestIsValidSubpath's, verbatim) through BOTH
// grammar sides and asserts per-input agreement, so neither can drift alone.
// Passes at b940037d1; the shipped .gno reject set never tries "a#b", NUL,
// non-ASCII, RTL-override, ".", "a/./b" or edge punctuation, so a mirror that
// diverges on those is caught here and nowhere else.

import (
	"chain"
	"strings"
)

const host = "gno.land/r/demo/submirror"

// subAccepts reports whether the Go native cur.Sub admits sp. The `_ int`
// leading param keeps this non-crossing, so main's cur stays the topmost
// crossing cur and Sub's strict-HIV entry guard still passes.
func subAccepts(_ int, rlm realm, sp string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	rlm.Sub(sp)
	return true
}

// deriveAccepts reports whether the .gno mirror chain.DerivePkgSubAddr
// admits sp for the same host.
func deriveAccepts(sp string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	chain.DerivePkgSubAddr(host, sp)
	return true
}

func main(cur realm) {
	// Verbatim from TestIsValidSubpath (gnovm/pkg/gnolang/uverse_test.go),
	// which drives the Go side only.
	valid := []string{
		"a", "dao", "dao42", "dao/42", "v1.2", "role_admin",
		"a/b/c", "g1abc", "a-b", "x.y-z_w", "0", "1/2/3",
	}
	invalid := []string{
		"", "/", "//", "dao/", "/dao", "a//b", "Dao", "DAO",
		"a b", "a\tb", "a\nb", "a:b", "a#b", "a\x00b", "..", ".", "../x",
		"_x", "x_", "-x", "x-", ".x", "x.",
		"café", // non-ASCII (é)
		"a‮b",  // RTL override
		"a/",
		"a/./b",
	}

	// Reject side: a mirror looser than the native derives fundable
	// addresses cur.Sub can never mint against, stranding the funds.
	rejectOK := true
	for _, sp := range invalid {
		s := subAccepts(0, cur, sp)
		d := deriveAccepts(sp)
		if s || d {
			println("MUST REJECT but accepted -- Sub:", s, "Derive:", d, "subpath:", sp)
			rejectOK = false
		}
	}
	println("both reject all", len(invalid), "invalid subpaths:", rejectOK)

	// Accept side (the paired baseline invariant): a mirror stricter than
	// the native breaks off-chain derivation for subpaths Sub happily mints.
	acceptOK := true
	for _, sp := range valid {
		s := subAccepts(0, cur, sp)
		d := deriveAccepts(sp)
		if !s || !d {
			println("MUST ACCEPT but rejected -- Sub:", s, "Derive:", d, "subpath:", sp)
			acceptOK = false
		}
	}
	println("both accept all", len(valid), "valid subpaths:", acceptOK)

	// Total-length cap, driven through both sides at the boundary. The
	// native caps len(host+sep+subpath); the mirror open-codes the
	// separator width as +1 instead of len(subRealmSep), so the two agree
	// only while the separator stays one byte.
	okSub := strings.Repeat("a", 256-len(host)-1) // synthesized == 256
	tooLong := okSub + "a"                        // synthesized == 257
	println("both accept the 256-byte boundary:",
		subAccepts(0, cur, okSub) && deriveAccepts(okSub))
	println("both reject the 257-byte boundary:",
		!subAccepts(0, cur, tooLong) && !deriveAccepts(tooLong))

	// Agreement on the derived address itself, not merely on accept/reject.
	println("derived addresses agree:",
		cur.Sub("dao/42").Address() == chain.DerivePkgSubAddr(host, "dao/42"))
}

// Output:
// both reject all 27 invalid subpaths: true
// both accept all 12 valid subpaths: true
// both accept the 256-byte boundary: true
// both reject the 257-byte boundary: true
// derived addresses agree: true
