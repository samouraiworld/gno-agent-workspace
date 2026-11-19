package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/gnolang/gno/gno.land/pkg/gnoweb/components"
	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func toc(items []*components.TocItem, out *[]string) {
	for _, it := range items {
		*out = append(*out, it.Title)
		toc(it.Items, out)
	}
}

func main() {
	r := gnoweb.NewHTMLRenderer(slog.New(slog.DiscardHandler), gnoweb.NewDefaultRenderConfig(), nil)
	u, _ := weburl.Parse("/")
	for _, f := range os.Args[1:] {
		src, _ := os.ReadFile(f)
		var buf bytes.Buffer
		meta, err := r.RenderRealm(&buf, u, src, gnoweb.RealmRenderContext{})
		var t []string
		toc(meta.Toc.Items, &t)
		h := buf.String()
		fmt.Printf("== %s err=%v html=%dB\n  toc(%d): %s\n  boards-heading=%v  read-blog-button=%v  last-heading=%q\n", f, err, len(h), len(t), strings.Join(t, " | "),
			strings.Contains(h, "Boards: on-chain forum</h2>"), strings.Contains(h, "Read the blog"), t[len(t)-1])
	}
}
