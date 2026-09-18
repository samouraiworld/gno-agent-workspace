// Repro, from a plain clone:
//
//   git clone https://github.com/gnolang/gno && cd gno
//   git fetch origin ecf7af0f29abe4737a52803d672bc5a33c17cc60 && git checkout FETCH_HEAD
//   cp <this file> misc/gnopreview/zz_inject_test.go
//   cd misc/gnopreview && go test -run 'TestCommentEscapesRealmPath|TestIndexEscapesRealmPath' ./...
//
// Both tests FAIL at ecf7af0: a realm path taken from a directory name a fork
// pull request controls reaches _preview/comment.md and _preview/index.html
// verbatim. pr-preview-publish.yml:174-177 posts that file as the body of the
// github-actions[bot] comment (gh api -F body=@_preview/comment.md), and
// index.html is pushed to the previews Pages site.
//
// Observed comment.md line at ecf7af0:
//   - [`gno.land/r/x/ev`il<img src=x>`](https://.../pr-1/r/x/ev`il<img src=x>/)
// The backtick in the directory name closes the code span Comment() wraps the
// path in (comment.go:33), so <img src=x> renders as markup in the bot comment.
//
// Reaching comment.md needs no successful crawl of the hostile realm:
// Comment() lists every entry of plan.Realms and never consults the crawler,
// and gnomod.toml's module line (which gnodev reads) is independent of the
// directory name (which plan.go:137 reads). index.html additionally needs the
// page in c.pages, which the second test supplies directly.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// payload is a directory name a fork pull request may add under examples/.
// The backtick closes the code span Comment wraps every realm path in; what
// follows it lands in the comment body as markup.
const payload = "ev`il<img src=x>"

func hostileRepo(t *testing.T) (root, realm string) {
	t.Helper()
	root = t.TempDir()
	write := func(p, body string) {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("examples/gnowork.toml", "")
	dir := "examples/gno.land/r/x/" + payload
	realm = "gno.land/r/x/" + payload
	write(dir+"/gnomod.toml", "module = \""+realm+"\"\n")
	write(dir+"/lib.gno", "package evil\n")
	return root, realm
}

func TestCommentEscapesRealmPath(t *testing.T) {
	root, realm := hostileRepo(t)
	plan, err := BuildPlan(root, []string{"examples/gno.land/r/x/" + payload + "/lib.gno"}, 25)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(plan.Realms, realm) {
		t.Fatalf("realm not planned: %#v", plan.Realms)
	}
	body := Comment(plan, "https://gnolang.github.io/gno-previews/pr-1", "1")
	t.Logf("comment.md:\n%s", body)
	if strings.Contains(body, "[`gno.land/r/x/ev`il<img src=x>`](") {
		t.Errorf("realm path reaches comment.md unescaped: the code span closes early and <img src=x> is link text")
	}
}

func TestIndexEscapesRealmPath(t *testing.T) {
	root, realm := hostileRepo(t)
	plan, err := BuildPlan(root, []string{"examples/gno.land/r/x/" + payload + "/lib.gno"}, 25)
	if err != nil {
		t.Fatal(err)
	}
	c := &Crawler{pages: map[string]*page{urlOf(realm): {}}}
	html := Index(plan, c)
	if strings.Contains(html, "<code>gno.land/r/x/ev`il<img src=x></code>") {
		t.Errorf("realm path reaches index.html unescaped:\n%s", html)
	}
}
