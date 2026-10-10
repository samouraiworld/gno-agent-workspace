// Round-3 probe for gnolang/gno#6229 at ea1c620f7: NewStaticAlias front
// matter. The first group re-runs the round-2 Warning (block scalars and
// wrapped plain values) and the round-2 quote Nit; the second group sends
// shapes the round-3 parser has not met.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6229/head && git checkout --detach ea1c620f7f5413f9a6db21883476c51b14ba8b9a
//	cp <this file> gno.land/pkg/gnoweb/zz_front_matter_r3_test.go
//	go test ./gno.land/pkg/gnoweb -run 'TestZZR3' -v
//
// Each case states what YAML (or the page author) means; a failing case is
// the defect, its logged values the observation.
package gnoweb_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/assert"
)

func TestZZR3FrontMatter(t *testing.T) {
	cases := []struct {
		name, content, title, description, body string
	}{
		// Round-2 Warning shapes: expected fixed at the head.
		{
			name:        "r2 folded block scalar",
			content:     "---\ntitle: About\ndescription: >\n  What gno.land is\n  and why it exists.\n---\nBody.\n",
			title:       "About",
			description: "What gno.land is and why it exists.",
			body:        "Body.\n",
		},
		{
			name:        "r2 literal block scalar",
			content:     "---\ndescription: |\n  What gno.land is.\n---\nBody.\n",
			description: "What gno.land is.",
			body:        "Body.\n",
		},
		{
			name:        "r2 multi-line plain scalar",
			content:     "---\ndescription: What gno.land is\n  and why it exists.\n---\nBody.\n",
			description: "What gno.land is and why it exists.",
			body:        "Body.\n",
		},
		// Round-2 Nit residue the author's reply did not name.
		{
			name:    "r2 single-quoted value with an escaped quote",
			content: "---\ntitle: 'It''s gno'\n---\nBody.\n",
			title:   "It's gno",
			body:    "Body.\n",
		},
		{
			name:    "r2 trailing comment",
			content: "---\ntitle: About # shipped with the node\n---\nBody.\n",
			title:   "About",
			body:    "Body.\n",
		},
		// New shapes.
		{
			name:    "quoted value on the line after its key",
			content: "---\ntitle:\n  \"Gno: a language\"\n---\nBody.\n",
			title:   "Gno: a language",
			body:    "Body.\n",
		},
		{
			name:    "lowercase colon prose between two thematic breaks is not front matter",
			content: "---\n\n# Welcome\n\nnote: the testnet resets every week.\n\n---\n\nMore.\n",
			body:    "---\n\n# Welcome\n\nnote: the testnet resets every week.\n\n---\n\nMore.\n",
		},
		{
			name:    "closing fence at end of file without a newline",
			content: "---\ntitle: About\n---",
			title:   "About",
			body:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gnoweb.NewStaticAlias(tc.content)
			t.Logf("title=%q description=%q body=%q", got.Title, got.Description, got.Value)
			assert.Equal(t, tc.title, got.Title)
			assert.Equal(t, tc.description, got.Description)
			assert.Equal(t, tc.body, got.Value)
		})
	}
}

// TestZZR3FrontMatterHead follows the round-2 folded description to the
// served head: the meta, og and twitter descriptions carry the joined text.
func TestZZR3FrontMatterHead(t *testing.T) {
	alias := gnoweb.NewStaticAlias("---\ntitle: About\ndescription: >\n  What gno.land is\n  and why it exists.\n---\n# About\n\nBody paragraph long enough to be a summary of its own.\n")
	handler := newMetadataHandler(t, "/r/gnoland/home", map[string]gnoweb.AliasTarget{"/about": alias})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/about", nil))
	head, _, _ := strings.Cut(rr.Body.String(), "</head>")
	for _, line := range strings.Split(head, "\n") {
		if strings.Contains(line, "description") {
			t.Log(strings.TrimSpace(line))
		}
	}
	assert.NotContains(t, head, `content="&gt;"`)
	assert.Contains(t, head, `<meta name="description" content="What gno.land is and why it exists." />`)
}

// TestZZR3LowercaseProseServed follows the lowercase colon prose case to the
// served page: the heading and the note between the two rules are gone from
// a 200 page that names itself index, follow. At the merge base main.go
// stored a static file verbatim (AliasTarget{Value: string(content)}), so
// the whole page rendered.
func TestZZR3LowercaseProseServed(t *testing.T) {
	alias := gnoweb.NewStaticAlias("---\n\n# Welcome\n\nnote: the testnet resets every week.\n\n---\n\nMore text that follows the second rule.\n")
	handler := newMetadataHandler(t, "/r/gnoland/home", map[string]gnoweb.AliasTarget{"/about": alias})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/about", nil))
	body := rr.Body.String()
	head, _, _ := strings.Cut(body, "</head>")
	for _, line := range strings.Split(head, "\n") {
		if strings.Contains(line, "<title>") || strings.Contains(line, `name="robots"`) {
			t.Log(strings.TrimSpace(line))
		}
	}
	t.Logf("status=%d welcome=%v note=%v more=%v", rr.Code,
		strings.Contains(body, "Welcome"), strings.Contains(body, "testnet resets"), strings.Contains(body, "More text"))
	assert.Contains(t, body, "Welcome")
	assert.Contains(t, body, "the testnet resets every week.")
}
