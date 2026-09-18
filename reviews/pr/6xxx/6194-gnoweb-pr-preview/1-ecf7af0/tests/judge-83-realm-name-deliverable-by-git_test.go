// Which hostile realm directory names survive `git diff --name-only`, the only
// source of gnopreview's changed-file list (.github/workflows/pr-preview.yml,
// "Collect the changed paths"). Measured at PR 6194 head ecf7af0f2: git quotes a
// name carrying `"`, so that one never reaches BuildPlan; a backtick passes
// through and breaks the comment's `%s` code spans. FAILS at this head.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
//	cp reviews/pr/6xxx/6194-gnoweb-pr-preview/1-ecf7af0/tests/judge-83-realm-name-deliverable-by-git_test.go misc/gnopreview/zz_judge83_test.go
//	cd misc/gnopreview && go test -run TestJudge83 -v .
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// judge83Repo writes one realm per name under examples/gno.land/r/ and returns
// the root plus the paths git itself reports as changed.
func judge83Repo(t *testing.T, names []string) (string, []string) {
	t.Helper()
	root := t.TempDir()
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
	for _, n := range names {
		dir := "examples/gno.land/r/" + n
		// A *valid* module line: module.CheckImportPath (gnovm/pkg/gnomod/file.go:122)
		// rejects a path with a backtick, so the payload rides the directory name,
		// which is the only thing gnopreview reads (plan.go:136-141).
		write(dir+"/gnomod.toml", "module = \"gno.land/r/ok\"\n")
		write(dir+"/lib.gno", "package p\n")
	}
	for _, a := range [][]string{{"init", "-q", "."}, {"add", "-A"}} {
		cmd := exec.Command("git", a...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v: %v %s", a, err, out)
		}
	}
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Skip(err)
	}
	var changed []string
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasSuffix(l, ".gno") {
			changed = append(changed, l)
		}
	}
	return root, changed
}

// A `"` in the directory name is quoted by git, so it never reaches BuildPlan.
func TestJudge83QuoteIsNotDeliverable(t *testing.T) {
	_, changed := judge83Repo(t, []string{`ev"il`})
	for _, c := range changed {
		if strings.HasPrefix(c, "examples/") {
			t.Errorf("git passed %q through unquoted", c)
		}
	}
	t.Logf("git reports %q — the leading quote is why BuildPlan drops it", changed)
}

// A backtick is not quoted by git, reaches ChangedRealms, and closes the code
// span comment.go writes around every realm path.
func TestJudge83BacktickBreaksTheCodeSpan(t *testing.T) {
	root, changed := judge83Repo(t, []string{"aaa", "zz`<img src=x onerror=1>`y"})
	plan, err := BuildPlan(root, changed, 25)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%q", plan.ChangedRealms)
	body := Comment(plan, "https://example.test/pr-1", "1")
	t.Logf("comment:\n%s", body)
	// IS:     the realm path lands raw inside the `%s` code span.
	if strings.Contains(body, "`<img src=x onerror=1>`") {
		t.Errorf("a realm directory name closed the comment's code span")
	}
	// SHOULD: escaped, so no realm name can introduce markup.
	// if strings.Contains(body, "\\`<img src=x onerror=1>\\`") { /* fixed */ }
}
