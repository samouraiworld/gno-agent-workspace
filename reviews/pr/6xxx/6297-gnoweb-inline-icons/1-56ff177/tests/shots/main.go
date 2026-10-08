package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
)

type fixture struct {
	*gnoweb.MockClient
	pages map[string][]byte
}

func (f *fixture) Realm(ctx context.Context, path, args string) ([]byte, error) {
	if b, ok := f.pages[path]; ok {
		return b, nil
	}
	return f.MockClient.Realm(ctx, path, args)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	md, err := os.ReadFile(".shots/icons.md")
	if err != nil {
		panic(err)
	}
	toc, err := os.ReadFile(".shots/toc.md")
	if err != nil {
		panic(err)
	}
	cli := &fixture{
		MockClient: gnoweb.NewMockClient(&gnoweb.MockPackage{
			Path: "/r/demo/icons", Domain: "gno.land",
			Files: map[string]string{"icons.gno": "package icons\n\nfunc Render(string) string { return \"\" }\n"},
		}, &gnoweb.MockPackage{
			Path: "/r/demo/toc", Domain: "gno.land",
			Files: map[string]string{"toc.gno": "package toc\n\nfunc Render(string) string { return \"\" }\n"},
		}),
		pages: map[string][]byte{"/r/demo/icons": md, "/r/demo/toc": toc},
	}
	renderer := gnoweb.NewHTMLRenderer(logger, gnoweb.NewDefaultRenderConfig(), cli)
	meta := gnoweb.StaticMetadata{Domain: "gno.land", AssetsPath: "/public/", ChromaPath: "/public/_chroma/style.css"}
	h, err := gnoweb.NewHTTPHandler(logger, &gnoweb.HTTPHandlerConfig{
		ClientAdapter: cli, Meta: meta, Renderer: renderer,
		Aliases: map[string]gnoweb.AliasTarget{}, Timeout: 5e9,
	})
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.Handle("/public/", http.StripPrefix("/public/", gnoweb.AssetHandler()))
	mux.Handle("/shots/", http.StripPrefix("/shots/", http.FileServer(http.Dir(".shots/static"))))
	panic(http.ListenAndServe("127.0.0.1:"+os.Args[1], mux))
}
