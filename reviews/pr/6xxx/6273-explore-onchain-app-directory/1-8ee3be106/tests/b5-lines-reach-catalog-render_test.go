// gnolang/gno#6273 at 8ee3be106: render the real store home page (templates as
// shipped) to a static HTML file that links the committed, CI-checked build
// gno.land/pkg/gnoweb/public/main.css, for b5-lines-reach-catalog-css-probe.mjs.
//
// Fixture: four trusted apps (hero + a 3-card Spotlight), four community apps
// (cards carrying .community and .trust), and a quiet chain: height 1000, every
// pulse count 0, one milestone event.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//	cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b5_render_test.go
//	B5_OUT=/tmp/b5-home.html go test ./gno.land/pkg/gnoweb/feature/store -run TestB5RenderFixture -count=1
//	node b5-lines-reach-catalog-css-probe.mjs /tmp/b5-home.html   # see its header
package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestB5RenderFixture(t *testing.T) {
	out := os.Getenv("B5_OUT")
	if out == "" {
		t.Skip("B5_OUT not set")
	}
	ls := []string{
		listingJSON("alpha", "gnoland", kindApp, "Trusted One", `"stars":12`),
		listingJSON("bravo", "gnoland", kindApp, "Trusted Two", `"stars":5`),
		listingJSON("charlie", "gnoland", kindApp, "Trusted Three", ""),
		listingJSON("delta", "gnoland", kindApp, "Trusted Four", ""),
		listingJSON("pixels", "nym-acmex123", kindApp, "Pixels", ""),
		listingJSON("two", "nym-acmex123", kindApp, "Two", ""),
		listingJSON("three", "nym-acmex123", kindApp, "Three", ""),
		listingJSON("four", "nym-acmex123", kindApp, "Four", ""),
	}
	body := `{"version":1,"height":1000,"time":1759538400,` +
		`"pulse":{"listed_7d":0,"stars_7d":0,"updated_7d":0,"listed_30d":0,"stars_30d":0,"updated_30d":0},` +
		`"categories":[{"key":"defi","label":"DeFi","count":8}],` +
		`"shelves":[{"title":"New","pinned":true,"slugs":["alpha","bravo","charlie","delta","pixels","two","three","four"]}],` +
		`"activity":[{"kind":"milestone","slug":"alpha","stars":10}],` +
		`"listings":[` + strings.Join(ls, ",") + `]}`
	h, _ := newTestHandler(t, map[string]string{"api/v1/home": body})
	html := render(t, h, "")
	require.NotEmpty(t, html)
	css, err := filepath.Abs("../../public/main.css")
	require.NoError(t, err)
	doc := `<!doctype html><html lang="en"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<link rel="stylesheet" href="file://` + css + `"></head>` +
		`<body><main style="padding:16px">` + html + `</main></body></html>`
	require.NoError(t, os.WriteFile(out, []byte(doc), 0o644))
}
