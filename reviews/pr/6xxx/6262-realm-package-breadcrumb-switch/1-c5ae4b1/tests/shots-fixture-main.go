// Fixture server for gnolang/gno#6262 screenshots: real gnoweb templates and
// CSS over a MockClient, no node. /_open/<path> serves <path> with the
// kind-switch popover opened by an injected showPopover(), for browsers driven
// without clicks (firefox --screenshot); /_probe/<path> also writes the menu's
// box into the page. NOANCHOR=1 serves main.css with every anchor( renamed
// xanchor(, so @supports not (top: anchor(bottom)) holds.
//
// From a checkout of c5ae4b13a, with this file at .shots/main.go:
//
//	go build -o .shots/srv ./.shots/main.go
//	.shots/srv 127.0.0.1:8799 & NOANCHOR=1 .shots/srv 127.0.0.1:8798 &
//	./scripts/capture-web.mjs --url http://127.0.0.1:8799 --script shots-walk-fallback.mjs ...
//	firefox --headless --screenshot out.png http://127.0.0.1:8798/_open/r/alice/golf/game
package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/gnovm/pkg/doc"
)

const domain = "gno.land"

type fixture struct {
	*gnoweb.MockClient
	md map[string]string
}

func (f *fixture) Realm(ctx context.Context, path, args string) ([]byte, error) {
	if s, ok := f.md[path]; ok {
		return []byte(s), nil
	}
	return f.MockClient.Realm(ctx, path, args)
}

// ListPaths answers the counterpart lookup, which asks with the domain prefixed.
func (f *fixture) ListPaths(ctx context.Context, prefix string, limit int) ([]string, error) {
	return f.MockClient.ListPaths(ctx, strings.TrimPrefix(prefix, domain), limit)
}

var renderFuncs = []*doc.JSONFunc{{
	Name:    "Render",
	Params:  []*doc.JSONField{{Name: "path", Type: "string"}},
	Results: []*doc.JSONField{{Type: "string"}},
}}

func realm(p string) *gnoweb.MockPackage {
	name := p[strings.LastIndex(p, "/")+1:]
	return &gnoweb.MockPackage{Path: p, Domain: domain, Files: map[string]string{name + ".gno": "package " + name}, Functions: renderFuncs}
}

func pure(p string) *gnoweb.MockPackage {
	name := p[strings.LastIndex(p, "/")+1:]
	return &gnoweb.MockPackage{Path: p, Domain: domain, Files: map[string]string{name + ".gno": "package " + name}}
}

const inject = `<script>addEventListener("load",()=>document.getElementById("kind-switch-menu").showPopover())</script></body>`

// probe opens the menu and writes its box, the button's and the fallback's
// state into a fixed block at the bottom of the page.
const probe = `<script>addEventListener("load",()=>{const m=document.getElementById("kind-switch-menu");m.showPopover();{const r=m.getBoundingClientRect(),b=document.querySelector(".kind-switch").getBoundingClientRect(),c=getComputedStyle(m),o=document.createElement("pre");o.id="probe-out";o.style.cssText="position:fixed;bottom:0;left:0;right:0;margin:0;padding:12px;background:#ffd;font:16px monospace;z-index:9";o.textContent=navigator.userAgent.match(/(Firefox|Chrome)\/[0-9.]+/)[0]+"\nCSS.supports(top: anchor(bottom)): "+CSS.supports("top: anchor(bottom)")+"\nmenu top/left px: "+Math.round(r.top)+"/"+Math.round(r.left)+"\nbutton bottom/left px: "+Math.round(b.bottom)+"/"+Math.round(b.left)+"\ncomputed top/left: "+c.top+"/"+c.left;document.body.append(o)}})</script></body>`

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cli := &fixture{
		MockClient: gnoweb.NewMockClient(
			realm("/r/tests/vm"),
			realm("/r/tests/vm/crossrealm"),
			realm("/r/tests/vm/subtests"),
			pure("/p/tests/vm/crossrealm"),
			realm("/r/alice/golf/game"),
			pure("/p/alice/golf/course"),
			pure("/p/alice/golf/physics"),
		),
		md: map[string]string{
			"/r/tests/vm":            "# tests/vm\n\nThis is the Render output of the realm at `/r/tests/vm` itself.\n",
			"/r/tests/vm/crossrealm": "# crossrealm\n\nRender output of `/r/tests/vm/crossrealm`.\n",
			"/r/tests/vm/subtests":   "# subtests\n\nRender output of `/r/tests/vm/subtests`.\n",
			"/r/alice/golf/game":     "# Golf game\n\nRender output of `/r/alice/golf/game`.\n\nSome text so the page has a body under the header.\n",
		},
	}
	renderer := gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), cli)
	h, err := gnoweb.NewHTTPHandler(logger, &gnoweb.HTTPHandlerConfig{
		ClientAdapter: cli,
		Meta:          gnoweb.StaticMetadata{Domain: domain, AssetsPath: "/public/", ChromaPath: "/public/_chroma/style.css", ChainId: "dev"},
		Renderer:      renderer,
		Aliases:       map[string]gnoweb.AliasTarget{},
		Timeout:       5e9,
	})
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/public/_chroma/style.css", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		renderer.WriteChromaCSS(w)
	}))
	assets := http.StripPrefix("/public/", gnoweb.AssetHandler())
	if os.Getenv("NOANCHOR") != "" {
		// Rename every anchor( in main.css so the anchor() declarations fail to
		// parse and @supports not (top: anchor(bottom)) holds, as in a browser
		// without anchor positioning.
		inner := assets
		assets = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/main.css") {
				inner.ServeHTTP(w, r)
				return
			}
			rec := httptest.NewRecorder()
			inner.ServeHTTP(rec, r)
			w.Header().Set("Content-Type", "text/css")
			w.WriteHeader(rec.Code)
			w.Write(bytes.ReplaceAll(rec.Body.Bytes(), []byte("anchor("), []byte("xanchor(")))
		})
	}
	mux.Handle("/public/", assets)
	for prefix, script := range map[string]string{"/_open": inject, "/_probe": probe} {
		mux.Handle(prefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r2 := r.Clone(r.Context())
			r2.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			r2.RequestURI = r2.URL.RequestURI()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r2)
			body := bytes.Replace(rec.Body.Bytes(), []byte("</body>"), []byte(script), 1)
			for k, v := range rec.Header() {
				w.Header()[k] = v
			}
			w.Header().Del("Content-Length")
			w.WriteHeader(rec.Code)
			w.Write(body)
		}))
	}
	mux.Handle("/", h)
	addr := os.Args[1]
	logger.Info("listening", "addr", addr)
	panic(http.ListenAndServe(addr, mux))
}
