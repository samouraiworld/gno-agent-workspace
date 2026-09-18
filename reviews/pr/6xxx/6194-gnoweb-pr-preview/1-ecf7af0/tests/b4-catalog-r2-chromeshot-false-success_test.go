// chromeShot (misc/gnopreview/shots.go:184-190) reports success on two runs
// that produced no usable screenshot: a URL the preview server answers 404 on,
// and a browser invocation that wrote nothing over a file an earlier run left
// behind. Both make the pull-request comment publish an image that is not the
// page it is captioned with, and neither prints a line.
//
// Repro from a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp <this file> misc/gnopreview/zz_falsesuccess_test.go
//	cd misc/gnopreview && go test -run TestChromeShotFalseSuccess -v ./...
//
// Observed at ecf7af0f2 with Google Chrome 152.0.7977.75:
//
//	missing_page: chromeShot returned <nil> and wrote a 6743-byte PNG of
//	              "404 page not found"
//	stale_file:   chromeShot returned <nil> over a 22-byte file the browser
//	              never touched

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChromeShotFalseSuccess(t *testing.T) {
	bin := findChrome("")
	if bin == "" {
		t.Skip("no Chrome/Chromium on PATH")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv, origin, err := serve(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	t.Run("missing_page", func(t *testing.T) {
		dst := filepath.Join(dir, "missing.png")
		err := chromeShot(bin, origin+"/r/x/leaf/index.html", dst)
		size := int64(-1)
		if fi, serr := os.Stat(dst); serr == nil {
			size = fi.Size()
		}
		if err == nil {
			t.Fatalf("chromeShot on a URL the server answers 404 on returned nil and wrote %d bytes; "+
				"the caller publishes that error page as the realm's screenshot", size)
		}
	})

	t.Run("stale_file", func(t *testing.T) {
		dst := filepath.Join(dir, "stale.png")
		if err := os.WriteFile(dst, []byte("not a png, but nonzero"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A browser that exits 0 and writes nothing; chromeShot only asks
		// whether a non-empty file sits at the destination afterwards.
		if err := chromeShot("/bin/true", origin+"/index.html", dst); err == nil {
			t.Fatal("chromeShot returned nil after the browser wrote nothing, because a file from an earlier run was still at the destination")
		}
	})
}
