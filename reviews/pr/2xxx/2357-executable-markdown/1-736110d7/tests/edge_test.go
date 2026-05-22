package doctest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestEmptyCodeBlock probes parser handling of empty fenced blocks.
func TestEmptyCodeBlock(t *testing.T) {
	input := "```gno\n```"
	blocks, err := GetCodeBlocks(input)
	assert.Nil(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, "", blocks[0].content)

	// What happens when we try to execute an empty block?
	_, execErr := ExecuteCodeBlock(blocks[0], DefaultRootDir())
	t.Logf("empty block exec err: %v", execErr)
}

// TestSharedGnoStoreLeak verifies that two `package main` blocks
// sharing the same gnoStore in ExecuteMatchingCodeBlock do not see
// each other's state. The first block defines a global; the second
// references an undefined identifier with the same name. If isolation
// is broken, the second block would resolve `leaked` and succeed.
func TestSharedGnoStoreLeak(t *testing.T) {
	content := `
` + "```gno" + `
// NAME: first
package main

var leaked = "captured"

func main() {
	println(leaked)
}
` + "```" + `

` + "```gno" + `
// NAME: second
package main

func main() {
	println(leaked) // should NOT resolve
}
` + "```" + `
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := ExecuteMatchingCodeBlock(ctx, content, "", DefaultRootDir())
	t.Logf("results: %v", results)
	t.Logf("err: %v", err)
	if err == nil {
		assert.Fail(t, "expected second block to fail with undefined identifier")
	}
}

// TestEmptyOutputDirective: the `// Output:` directive with no values
// should ideally assert "no output", but the current implementation
// treats it as no expectation at all. This pins the current behavior.
func TestEmptyOutputDirective(t *testing.T) {
	content := `
` + "```gno" + `
// NAME: prints_but_expects_nothing
package main

func main() {
	println("surprise!")
}

// Output:
` + "```" + `
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := ExecuteMatchingCodeBlock(ctx, content, "", DefaultRootDir())
	assert.Nil(t, err)
	t.Logf("results: %v", results)
	// Documents current behavior: empty Output directive does not
	// assert anything; the code still emits "surprise!" and the run
	// is considered successful.
	assert.Len(t, results, 1)
	assert.Contains(t, results[0], "surprise!")
}

// TestInvalidPatternSilentlySkips: matchPattern silently returns
// false on invalid regex, so a malformed -run pattern matches
// nothing. The user gets "No code blocks matched" with no hint that
// the regex was malformed.
func TestInvalidPatternSilentlySkips(t *testing.T) {
	content := `
` + "```gno" + `
// NAME: simple
package main

func main() { println("hi") }
` + "```" + `
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := ExecuteMatchingCodeBlock(ctx, content, "[unclosed", DefaultRootDir())
	assert.Nil(t, err)
	assert.Empty(t, results)
}

// TestRegexLiteralEscape: a user wanting to match the literal string
// "regex:foo" cannot, since the prefix is consumed.
func TestRegexLiteralEscapeUnreachable(t *testing.T) {
	_, err := compareResults("regex:foo", "regex:foo", "")
	// With the prefix stripped, "foo" is compiled as a regex and
	// matched against "regex:foo" — passes by accident. Documents the
	// limitation: there is no way to assert literal "regex:..." text.
	assert.Nil(t, err)
}

// TestErrorAndOutputBothSet: when a block has both Output: and
// Error: directives, compareResults silently prefers Output. The
// Error expectation is dropped.
func TestErrorAndOutputBothSetPrefersOutput(t *testing.T) {
	_, err := compareResults("hello", "hello", "should-not-be-checked")
	assert.Nil(t, err)
}

// TestIgnoredBlockOptionsConflict: a block declaring both IGNORE and
// SHOULD_PANIC: IGNORE wins (block is skipped). Document precedence.
func TestIgnoreBeatsShouldPanic(t *testing.T) {
	cb := codeBlock{
		content: `package main
func main() {
	panic("never runs")
}`,
		lang: "gno",
		options: ExecutionOptions{
			Ignore:      true,
			ShouldPanic: true,
		},
	}
	out, err := ExecuteCodeBlock(cb, DefaultRootDir())
	assert.Nil(t, err)
	assert.Equal(t, "IGNORED", strings.TrimSpace(out))
}
