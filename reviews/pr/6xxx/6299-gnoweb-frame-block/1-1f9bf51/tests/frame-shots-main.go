package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

// Realm markdown per path: the repro's two failing cases, and each with its
// HTML block removed.
var pages = map[string]string{
	"/r/demo/div":        "<gno-frame>\n## Apps\n<gno-columns>\n<div></div>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n",
	"/r/demo/nodiv":      "<gno-frame>\n## Apps\n<gno-columns>\n<gno-frame>\ncard\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\n",
	"/r/demo/comment":    "<gno-frame>\nintro\n<!--\ndraft\n<gno-columns>\nhidden\n-->\n</gno-frame>\n",
	"/r/demo/nocomment":  "<gno-frame>\nintro\n</gno-frame>\n",
	"/r/demo/outcomment": "intro\n<!--\ndraft\n<gno-columns>\nhidden\n-->\n",
}

type fixture struct{ *gnoweb.MockClient }

func (f *fixture) Realm(_ context.Context, path, _ string) ([]byte, error) {
	if b, ok := pages[path]; ok {
		return []byte(b), nil
	}
	return nil, gnoweb.ErrClientPackageNotFound
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	var pkgs []*gnoweb.MockPackage
	for p := range pages {
		pkgs = append(pkgs, &gnoweb.MockPackage{Path: p, Domain: "gno.land", Files: map[string]string{"x.gno": "package x"}})
	}
	cli := &fixture{gnoweb.NewMockClient(pkgs...)}
	renderer := gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), cli)
	h, err := gnoweb.NewHTTPHandler(logger, &gnoweb.HTTPHandlerConfig{
		ClientAdapter: cli,
		Meta:          gnoweb.StaticMetadata{Domain: "gno.land", AssetsPath: "/public/", ChromaPath: "/public/_chroma/style.css", AssetsVersion: gnoweb.AssetsVersion()},
		Renderer:      renderer,
		Aliases:       map[string]gnoweb.AliasTarget{},
		Timeout:       5e9,
	})
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/public/_chroma/style.css", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_ = renderer.WriteChromaCSS(w)
	}))
	mux.Handle("/public/", http.StripPrefix("/public/", gnoweb.AssetHandler()))
	mux.Handle("/", h)
	panic(http.ListenAndServe("127.0.0.1:8811", mux))
}
