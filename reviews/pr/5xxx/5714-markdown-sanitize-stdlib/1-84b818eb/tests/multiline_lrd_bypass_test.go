// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5714 -R gnolang/gno && git checkout 84b818eb
curl -fsSL -o gnovm/stdlibs/chain/markdown/multiline_lrd_bypass_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5714-markdown-sanitize-stdlib/1-84b818eb/tests/multiline_lrd_bypass_test.go
go test -v -run TestMultilineLRDBypass ./gnovm/stdlibs/chain/markdown/
rm gnovm/stdlibs/chain/markdown/multiline_lrd_bypass_test.go
*/

package markdown

import "testing"

// EscapeBlockHazards' isLRDDefinition only recognises link-reference
// definitions whose URL is on the same line as `[label]:`. CommonMark
// §4.7 also allows the destination on the *next* line ("optional
// whitespace including up to one line ending"). Goldmark parses these
// the same way. The strip therefore misses the multi-line shape, and
// a user-supplied LRD survives sanitize.Block intact — letting user
// content define a target that realm-side `[label]` shortcut references
// resolve against.
//
// IS: output equals input — the LRD passes through.
// SHOULD: output is empty (or the `[` is escaped) so goldmark cannot
// resolve `[label]` against user-provided bytes.
func TestMultilineLRDBypass(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"label-only first line", "[foo]:\nhttp://attacker\n"},
		{"label + trailing space", "[foo]:  \n  http://attacker\n"},
		{"label + indented URL", "[foo]:\n  http://attacker\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := EscapeBlockHazards(c.in)
			if got == c.in {
				t.Errorf("bypass: input survived unchanged\n  in : %q\n  out: %q", c.in, got)
			}
		})
	}
}
