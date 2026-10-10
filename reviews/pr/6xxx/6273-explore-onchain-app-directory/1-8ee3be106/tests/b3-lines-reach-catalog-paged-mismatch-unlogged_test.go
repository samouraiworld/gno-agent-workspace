// Repro, from a plain clone of gnolang/gno at 8ee3be106cd932e3156197c4ce3a080520493eb9:
//
//	git clone https://github.com/gnolang/gno && cd gno && git checkout 8ee3be106cd932e3156197c4ce3a080520493eb9
//	cp <this file> gno.land/pkg/gnoweb/feature/store/zz_b3_paged_unlogged_test.go
//	go test -count=1 -run TestB3PagedMismatchIsNotLogged -v ./gno.land/pkg/gnoweb/feature/store/
//
// Claim: paged() builds the "store: <endpoint> answered ..." error after
// decode() returned nil, so decode's deferred Warn never sees it, and
// servePage drops it ("an error is logged once by decode"). A realm whose
// paged answer does not echo the key, page or page count makes every such
// page fall back to the realm's markdown with no log line at all. The second
// target is the control: a malformed body on the same path shape is logged.
package store

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb/weburl"
)

func TestB3PagedMismatchIsNotLogged(t *testing.T) {
	var logs bytes.Buffer
	home := `{"version":1,"categories":[{"key":"defi","label":"DeFi","count":30}],"shelves":[],"listings":[]}`
	c := &fakeClient{responses: map[string]string{
		"api/v1/home": home,
		// The realm answers page 1 of another category: paged() refuses it.
		"api/v1/category/defi/1": `{"version":1,"category":{"key":"games","label":"Games","count":30},"page":1,"pages":2,"listings":[]}`,
		// Control: a truncated body, refused inside decode().
		"api/v1/category/defi/2": `{"version":1,"category":`,
	}}
	h := New(Deps{
		Client:    c,
		RealmPath: "/r/gnoland/store",
		Domain:    "gno.land",
		Trusted:   func(string) bool { return false },
		Logger:    slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	for _, target := range []string{":c/defi", ":c/defi?page=2"} {
		u, err := weburl.Parse("/r/gnoland/store" + target)
		if err != nil {
			t.Fatal(err)
		}
		view, m := h.View(context.Background(), u)
		t.Logf("%s: view nil=%v status=%d", target, view == nil, m.Status)
	}
	out := logs.String()
	t.Logf("log output:\n%s", out)
	if !strings.Contains(out, "category/defi/2") {
		t.Fatalf("control: the decode failure was not logged either")
	}
	if !strings.Contains(out, "answered") {
		t.Errorf("echo mismatch on category/defi/1 left no log line: the page fell back to markdown silently")
	}
}
