// Equivalence proof for the hasRawGoGenerate simplification suggested in the
// review: the strings.SplitSeq form and the hand-rolled offset loop agree on
// every input, so the 17-line loop can be replaced by an 8-line one.
//
// Run: from a gno checkout:
//   gh pr checkout 6078 -R gnolang/gno && git checkout 5e2a310bb
//   curl -fsSL -o gnovm/pkg/gnolang/hasrawgogenerate_equiv_test.go \
//     https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6078-reject-tooling-directives/2-5e2a310bb/tests/hasrawgogenerate_equiv_test.go
//   go test -run 'TestHasRawGoGenerateEquivalence' ./gnovm/pkg/gnolang/
//   rm gnovm/pkg/gnolang/hasrawgogenerate_equiv_test.go
//
// Passes at 5e2a310bb over 200k generated bodies: the two forms never disagree.
package gnolang

import (
	"math/rand"
	"strings"
	"testing"
)

// splitSeqGoGenerate is the proposed replacement body.
func splitSeqGoGenerate(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "//go:generate ") ||
			strings.HasPrefix(line, "//go:generate\t") {
			return true
		}
	}
	return false
}

func TestHasRawGoGenerateEquivalence(t *testing.T) {
	t.Parallel()

	// The fragments that make the line-start rule hard: the prefix with each
	// accepted separator, near-misses, indentation, CR line endings, and the
	// no-final-newline case.
	frags := []string{
		"//go:generate ls", "//go:generate\tls", "//go:generate", "//go:generatex ls",
		" //go:generate ls", "\t//go:generate ls", "//go:generate ls\r",
		"package p", "", "\r", "/*", "*/", "var s = `", "`", "// ordinary",
	}
	seps := []string{"\n", "\r\n", ""}

	rng := rand.New(rand.NewSource(1))
	for i := range 200_000 {
		var b strings.Builder
		for range rng.Intn(6) {
			b.WriteString(frags[rng.Intn(len(frags))])
			b.WriteString(seps[rng.Intn(len(seps))])
		}
		body := b.String()
		if got, want := splitSeqGoGenerate(body), hasRawGoGenerate(body); got != want {
			t.Fatalf("case %d disagree on %q: splitSeq=%v offsetLoop=%v", i, body, got, want)
		}
	}
}
