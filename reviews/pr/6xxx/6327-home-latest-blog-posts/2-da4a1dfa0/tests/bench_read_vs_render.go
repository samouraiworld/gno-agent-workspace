package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func main() {
	path := os.Args[1]
	const n = 2000
	r := gnoweb.NewHTMLRenderer(slog.New(slog.DiscardHandler), gnoweb.NewDefaultRenderConfig(), nil)
	u, _ := weburl.Parse("/")

	start := time.Now()
	for i := 0; i < n; i++ {
		if _, err := os.Stat(path); err != nil {
			panic(err)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		_ = gnoweb.NewStaticAlias(string(b))
	}
	read := time.Since(start) / n

	src, _ := os.ReadFile(path)
	start = time.Now()
	for i := 0; i < n; i++ {
		var buf bytes.Buffer
		if _, err := r.RenderRealm(&buf, u, src, gnoweb.RealmRenderContext{}); err != nil {
			panic(err)
		}
	}
	render := time.Since(start) / n
	fmt.Printf("stat+read+NewStaticAlias: %v per request\nRenderRealm:              %v per request\nratio: %.0fx\n", read, render, float64(render)/float64(read))
}
