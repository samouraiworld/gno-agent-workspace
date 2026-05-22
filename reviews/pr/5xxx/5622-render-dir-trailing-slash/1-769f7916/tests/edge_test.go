package gnoweb

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Edge cases beyond what the PR's TestCanonicalPathURL covers.

func TestCanonicalPathURL_ArgsAndWebQuery(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/r/demo/foo/:bob$source", nil)
	assert.Equal(t, "/r/demo/foo:bob$source", canonicalPathURL(req))
}

func TestCanonicalPathURL_DoubleTrailingSlash(t *testing.T) {
	t.Parallel()

	// Only one slash is trimmed per redirect; second one would trigger another.
	req := httptest.NewRequest("GET", "/r/demo/foo//", nil)
	assert.Equal(t, "/r/demo/foo/", canonicalPathURL(req))
}

func TestCanonicalPathURL_DollarBeforeColon(t *testing.T) {
	t.Parallel()

	// `$` and `:` can both appear, and Cut on `:` runs first.
	// Pre-trim raw path: `/r/foo/$bar:baz` (no trailing slash; redirect path
	// wouldn't fire here, but we still want canonicalPathURL not to corrupt it).
	req := httptest.NewRequest("GET", "/r/foo$bar:baz", nil)
	got := canonicalPathURL(req)
	// Cut on `:` → prefix="/r/foo$bar", suffix="baz".
	// TrimSuffix("/") → no-op. canonical = "/r/foo$bar:baz".
	assert.Equal(t, "/r/foo$bar:baz", got)
}

func TestCanonicalPathURL_EmptyAfterDelimiter(t *testing.T) {
	t.Parallel()

	// Edge: trailing `:` or `$` with empty suffix.
	req := httptest.NewRequest("GET", "/r/foo/:", nil)
	// Cut on `:` → prefix="/r/foo/", suffix="", found=true.
	// suffix=="" branch skips the `:` reattachment.
	// → "/r/foo"
	assert.Equal(t, "/r/foo", canonicalPathURL(req))
}
