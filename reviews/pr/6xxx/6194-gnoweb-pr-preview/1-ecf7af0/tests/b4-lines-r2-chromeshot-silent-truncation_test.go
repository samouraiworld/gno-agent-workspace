package main

// b4-lines-r2-chromeshot-silent-truncation
//
// chromeShot (misc/gnopreview/shots.go:166) accepts any non-empty PNG as a
// successful screenshot. Two pages that never rendered still pass it:
//
//   1. a page whose content appears after the 4 s --virtual-time-budget
//      (shots.go:180) is captured in its pre-render state;
//   2. a URL the local file server answers 404 for is captured as a
//      screenshot of "404 page not found".
//
// Both return nil from chromeShot, so the image is published as the
// before/after of a changed realm with nothing on stderr.
//
// Repro from a plain clone:
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//   cp <this file> misc/gnopreview/b4_test.go
//   cd misc/gnopreview && go test -run TestB4ChromeShotSilentTruncation -v ./...
// Needs a Chrome/Chromium on PATH; measured with Google Chrome 152.0.7977.75, go1.25.9.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func b4write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func b4sum(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:6])
}

func TestB4ChromeShotSilentTruncation(t *testing.T) {
	bin := findChrome("")
	if bin == "" {
		t.Skip("no Chrome/Chromium on PATH")
	}
	dir := t.TempDir()
	b4write(t, dir, "rendered.html", `<!doctype html><body style="background:#ff0000;margin:0"></body>`)
	b4write(t, dir, "blank.html", `<!doctype html><body style="background:#ffffff;margin:0"></body>`)
	b4write(t, dir, "slow.html", `<!doctype html><body style="background:#ffffff;margin:0"><script>setTimeout(function(){document.body.style.background="#ff0000"},10000)</script></body>`)

	srv, origin, err := serve(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	shot := func(name, url string) string {
		dst := filepath.Join(dir, name+".png")
		if err := chromeShot(bin, url, dst); err != nil {
			t.Fatalf("%s: chromeShot reported an error: %v", name, err)
		}
		return b4sum(t, dst)
	}

	rendered := shot("rendered", origin+"/rendered.html")
	blank := shot("blank", origin+"/blank.html")
	slow := shot("slow", origin+"/slow.html")
	missing := shot("missing", origin+"/r/gone/index.html") // 404 from the file server

	t.Logf("rendered=%s blank=%s slow=%s missing=%s", rendered, blank, slow, missing)

	if slow != blank {
		t.Fatalf("slow page rendered inside the budget (slow=%s blank=%s rendered=%s)", slow, blank, rendered)
	}
	t.Log("slow page: chromeShot returned nil and the PNG is the unrendered page")

	fi, err := os.Stat(filepath.Join(dir, "missing.png"))
	if err != nil || fi.Size() == 0 {
		t.Fatalf("404 page wrote no image: %v", err)
	}
	t.Logf("404 URL: chromeShot returned nil and wrote %d bytes", fi.Size())
}
