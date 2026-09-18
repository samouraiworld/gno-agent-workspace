# PR [#6194](https://github.com/gnolang/gno/pull/6194): feat(ci): publish a static gnoweb preview for every pull request

Verdict: REQUEST CHANGES: the publishing job picks the pull request to write to from a branch name the fork controls and commits fork-authored bytes onto the shared `gnolang.github.io` origin, and the preview it publishes advertises realms the crawl never captured.
Event: REQUEST_CHANGES
Model: claude-opus-5, standard review, high effort
Commit: ecf7af0f29abe4737a52803d672bc5a33c17cc60
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6194 ecf7af0`
Overview: [overview](../overview.md)
Open the code: [`misc/gnopreview`](https://github.com/gnolang/gno/tree/ecf7af0/misc/gnopreview)
Round: 1. 40 finders, one reflector, 255 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 14 refuted, none of them above Nit.

The run's four consumers each decide on their own which realms the snapshot holds, the [written tree](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L326), the [landing page](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L166), the [sticky comment](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L17) and the [size-capped publish](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L127-L132), and no value carried between them records what the render dropped.

## .github/workflows/pr-preview-publish.yml:77 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L77) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview-publish.yml#L77) · Warning
`head=$HEAD_OWNER:$HEAD_BRANCH` pastes the fork's branch name into the query unencoded, and the branch name is what the fork picks:

- `fix/loader&context-from-patterns` passes `git check-ref-format --branch` and truncates the filter at the `&`, so the lookup matches nothing and [line 81](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L81) prints `closed while rendering; nothing to publish` for a pull request that is open.
- `zzz&head=<owner>%3A<branch>` appends a second `head=`, which the REST API honours over the first, so `$prs` holds another pull request's number and head sha: `pr-<victim>/` is republished from a run whose base the fork chose and the victim's sticky comment is PATCHed with links into `pr-<attacker>/`. The [head-sha equality check](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L90-L93) holds the rendered bytes to that commit and does not hold the write to the right pull request.

Resolving by sha with `repos/$REPO/commits/$HEAD_SHA/pulls` takes the branch out of the query.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno (gh authenticated; every call is read-only):
gh pr checkout 6194 -R gnolang/gno

# 1. a branch name carrying & and %3A is a pushable ref
for b in 'zzz&head=gnolang%3Amaster' 'zzz&state=all' 'feat/a+b'; do
  git check-ref-format --branch "$b" >/dev/null 2>&1 && echo "  VALID   $b" || echo "  INVALID $b"
done

# 2. which occurrence of a repeated query parameter the API honours
gh api "repos/gnolang/gno/pulls?state=open&state=closed&per_page=3" --jq '[.[].state]'

# 3. the same override applied to head=, as the branch name would
victim=$(gh api "repos/gnolang/gno/pulls?state=open&per_page=1" --jq '.[] | .head.label')
vowner=${victim%%:*}; vref=${victim#*:}
gh api "repos/gnolang/gno/pulls?state=open&per_page=100&head=nobody-xyz:zzz" --jq 'length'
gh api "repos/gnolang/gno/pulls?state=open&per_page=100&head=nobody-xyz:zzz&head=$vowner%3A$vref" \
  --jq '.[] | "\(.number) \(.head.sha)"'
```

Step 1 marks every name VALID, step 2 answers with the closed states, and the two calls in step 3 answer `0` and then an unrelated pull request's number and head sha: that last value is what `$prs` would hold.

</details>

## .github/workflows/pr-preview-publish.yml:140 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L140) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview-publish.yml#L140) · Warning
`cp -r ../_preview/. "pr-$PR/"` commits bytes the pull request chose, since the render job [builds and runs the fork's own copy of `misc/gnopreview`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L76-L82) and the only filters ahead of the copy are the [`.git` prune and the 40 MiB `du -sm`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L127-L132), so a fork publishes arbitrary HTML and script on `gnolang.github.io`, one browser origin with the [godoc mirror](https://gnolang.github.io/gno/) served from that host. A hostname of its own for the previews site puts the snapshots in a separate origin.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno

# 1. the render job builds and runs the pull request's own gnopreview
sed -n '14,15p;74,78p;96,102p' .github/workflows/pr-preview.yml

# 2. everything the privileged job does to those bytes before committing them
sed -n '127,132p;138,141p' .github/workflows/pr-preview-publish.yml

# 3. one host, two Pages sites
for u in https://gnolang.github.io/gno-previews/ https://gnolang.github.io/gno/; do
  printf '  %-44s HTTP %s\n' "$u" "$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$u")"
done

# 4. the neighbour on that origin runs script against document.cookie
curl -s --max-time 15 https://gnolang.github.io/gno/ | grep -n -m1 -B2 'document.cookie'
```

Step 3 answers 200 for both URLs and step 4 prints the mirror's inline `document.cookie` read, so a page under `gno-previews/pr-<N>/` can fetch that site, frame and read it, and touch that origin's cookies and storage. The [header's own answer](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L14-L17) is about hostnames mistaken for production, which settles phishing and leaves script isolation open.

</details>

## misc/gnopreview/comment.go:24 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L24) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L24) · Warning
An empty plan returns an empty body and no publish path then removes what the last push left, since the render, the screenshots and the upload are [gated on the plan](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L85) and the upsert step [exits 0 on a missing body](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L166), so a push that reverts the last realm change leaves screenshots and links of a head the branch no longer carries until the pull request closes.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Asserts an empty plan still produces a marked body, which is the only thing
// the publish job's upsert step can replace the previous comment with.
// Fails at this head: Comment returns "".
package main

import (
	"strings"
	"testing"
)

func TestCatalogEmptyPlanCanClearTheComment(t *testing.T) {
	got := Comment(&Plan{}, "https://example.test/pr-1", "1")
	if !strings.Contains(got, CommentMarker) {
		t.Errorf("Comment(empty plan) = %q; a body carrying %q is the only way to replace the previous push comment",
			got, CommentMarker)
	}
}
EOF
cd misc/gnopreview && go test -run TestCatalogEmptyPlanCanClearTheComment -v .
rm zz_run_test.go
```

The body is empty, so the upsert step has nothing to PATCH the standing comment with.

```
=== RUN   TestCatalogEmptyPlanCanClearTheComment
    zz_run_test.go:32: Comment(empty plan) = ""; a body carrying "<!-- gnoweb-pr-preview -->" is the only way to replace the previous push comment
--- FAIL: TestCatalogEmptyPlanCanClearTheComment (0.00s)
FAIL
FAIL	github.com/gnolang/gno/misc/gnopreview	0.002s
```

</details>

## SKIP misc/gnopreview/comment.go:25 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L25) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L25) · Suggestion
Returning the empty string here is what makes [`render`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L111-L114) write no `comment.md`, and the publish job reads a missing body as nothing to do.

SKIP: the same empty-plan body as the posted Warning above.

## misc/gnopreview/comment.go:33 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L33) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L33) · Warning
A realm path is [the directory name the pull request chose](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L136-L141) rather than the module line, and it lands raw inside this code span, so a directory named with a backtick closes the span and introduces markup into a body [the repository token posts as github-actions[bot]](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L174-L177).

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Which hostile realm directory names survive `git diff --name-only`, the only
// source of the changed-file list. git quotes a name carrying a double quote,
// so that one never reaches BuildPlan; a backtick passes through and closes the
// comment's code span. Fails at this head.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestJudge83BacktickBreaksTheCodeSpan(t *testing.T) {
	root, changed := judge83Repo(t, []string{"aaa", "zz`<img src=x onerror=1>`y"})
	plan, err := BuildPlan(root, changed, 25)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%q", plan.ChangedRealms)
	body := Comment(plan, "https://example.test/pr-1", "1")
	t.Logf("comment:\n%s", body)
	if strings.Contains(body, "`<img src=x onerror=1>`") {
		t.Errorf("a realm directory name closed the comment's code span")
	}
}
EOF
cd misc/gnopreview && go test -run TestJudge83Backtick -v .
rm zz_run_test.go
```

The directory name lands inside the span, which ends where the name's first backtick does.

```
=== RUN   TestJudge83BacktickBreaksTheCodeSpan
    zz_run_test.go:87: ChangedRealms=["gno.land/r/aaa" "gno.land/r/zz`<img src=x onerror=1>`y"]
    zz_run_test.go:89: comment:
        <!-- gnoweb-pr-preview -->
        ### 🖼️ gnoweb preview

        **Changed realms (2)**

        - [`gno.land/r/aaa`](https://example.test/pr-1/r/aaa/) · [source](https://example.test/pr-1/r/aaa/_t/source/) · [help](https://example.test/pr-1/r/aaa/_t/help/)
        - [`gno.land/r/zz`<img src=x onerror=1>`y`](https://example.test/pr-1/r/zz`<img src=x onerror=1>`y/) · [source](https://example.test/pr-1/r/zz`<img src=x onerror=1>`y/_t/source/)
    zz_run_test.go:92: a realm directory name closed the comment's code span
--- FAIL: TestJudge83BacktickBreaksTheCodeSpan (0.01s)
FAIL
```

</details>

## SKIP misc/gnopreview/comment.go:118 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L118) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L118) · Suggestion
The realm path is written into this `alt` attribute and into [the index page's rows](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L173-L176) with no escaping, and a quote in the name ends the attribute.

SKIP: the attribute sites need a captured page, which the run did not reach, and `html.EscapeString` here is the same edit as the posted section on the code span.

## misc/gnopreview/comment.go:62 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L62) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L62) · Warning
This loop runs over the uncapped `ChangedRealms` while [the cap truncates the rendered list with no floor at their count](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L259), so a gnomod.toml sweep over the 58 realm directories in `examples/gno.land/r` links 33 realms to pages the render never wrote.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Asserts every changed realm the comment links was rendered. A gnomod.toml
// bump over every realm in examples/ at the default cap of 25 drops 33 changed
// realms from the render while the comment links all 58. Fails at this head.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func judge67Sweep(t *testing.T) []string {
	t.Helper()
	var changed []string
	err := filepath.WalkDir("../../examples/gno.land/r", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "gnomod.toml" {
			changed = append(changed, "examples/"+filepath.ToSlash(strings.TrimPrefix(p, "../../examples/")))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestJudge67CapDropsChangedRealmsTheCommentStillLinks(t *testing.T) {
	changed := judge67Sweep(t)
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("changed files=%d ChangedRealms=%d Realms=%d Dropped=%d cap=%d",
		len(changed), len(plan.ChangedRealms), len(plan.Realms), plan.Dropped, defaultMaxRealms)

	base := "https://example.test/pr-1"
	body := Comment(plan, base, "1")
	var linkedButUnrendered []string
	for _, r := range plan.ChangedRealms {
		if contains(plan.Realms, r) {
			continue
		}
		if strings.Contains(body, base+urlOf(r)+"/") {
			linkedButUnrendered = append(linkedButUnrendered, r)
		}
	}
	if len(linkedButUnrendered) > 0 {
		t.Errorf("%d changed realm(s) linked but never rendered, e.g. %s%s/ for %s",
			len(linkedButUnrendered), base, urlOf(linkedButUnrendered[0]), linkedButUnrendered[0])
	}
}
EOF
cd misc/gnopreview && go test -run TestJudge67CapDrops -v .
rm zz_run_test.go
```

Every one of the 33 realms the cap dropped is still linked, and each link is a 404 on the published site.

```
=== RUN   TestJudge67CapDropsChangedRealmsTheCommentStillLinks
    zz_run_test.go:50: changed files=58 ChangedRealms=58 Realms=25 Dropped=33 cap=25
    zz_run_test.go:67: 33 changed realm(s) linked but never rendered, e.g. https://example.test/pr-1/r/gov/dao/treasury/test/v0/ for gno.land/r/gov/dao/treasury/test/v0
--- FAIL: TestJudge67CapDropsChangedRealmsTheCommentStillLinks (0.01s)
FAIL
```

</details>

## misc/gnopreview/comment.go:69 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L69) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L69) · Warning
[`isSeed`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L157-L162) answers on membership in the gnoweb seed list rather than on why the realm is in the plan, so a pull request touching gnoweb and a package leaves `gno.land/r/gnoland/home` rendered, listed on the index page, and named nowhere in the comment, with [the screenshot grid](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L147-L152) empty because a changed realm is present.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Asserts a gnoweb seed realm the diff reaches through a changed package is
// named in the comment. On the real examples/ tree, a diff touching gnoweb,
// p/moul/dynreplacer/v0 and one unrelated realm renders r/gnoland/home and
// never links it. Fails at this head.
package main

import (
	"strings"
	"testing"
)

func TestJudge66SeedRealmAffectedByAChangedPackageIsUnlisted(t *testing.T) {
	changed := []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/p/moul/dynreplacer/v0/gnomod.toml",
		"examples/gno.land/r/demo/counter/gnomod.toml",
	}
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	const seed = "gno.land/r/gnoland/home"
	if !contains(plan.Realms, seed) {
		t.Fatalf("fixture stale: %s is not rendered; Realms=%v", seed, plan.Realms)
	}
	t.Logf("Mode=%s Gnoweb=%v ChangedRealms=%v ChangedPkgs=%v Realms=%v",
		plan.Mode(), plan.Gnoweb, plan.ChangedRealms, plan.ChangedPkgs, plan.Realms)

	body := Comment(plan, "https://example.test/pr-1", "1")
	if !strings.Contains(body, seed) {
		t.Errorf("%s is rendered but never named in the comment\n---\n%s", seed, body)
	}
}
EOF
cd misc/gnopreview && go test -run TestJudge66 -v .
rm zz_run_test.go
```

The plan renders five realms and the comment names one.

```
=== RUN   TestJudge66SeedRealmAffectedByAChangedPackageIsUnlisted
    zz_run_test.go:38: Mode=both Gnoweb=true ChangedRealms=[gno.land/r/demo/counter] ChangedPkgs=[gno.land/p/moul/dynreplacer/v0] Realms=[gno.land/r/demo/counter gno.land/r/docs/security_patterns gno.land/r/gnoland/blog gno.land/r/gnoland/boards2/v0 gno.land/r/gnoland/home]
    zz_run_test.go:44: gno.land/r/gnoland/home is rendered but never named in the comment
        ---
        <!-- gnoweb-pr-preview -->
        ### 🖼️ gnoweb preview

        This PR changes **gnoweb** and realm sources. **[Open the preview homepage](https://example.test/pr-1/)**

        **Changed realms (1)**

        - [`gno.land/r/demo/counter`](https://example.test/pr-1/r/demo/counter/) · [source](https://example.test/pr-1/r/demo/counter/_t/source/) · [help](https://example.test/pr-1/r/demo/counter/_t/help/)
--- FAIL: TestJudge66SeedRealmAffectedByAChangedPackageIsUnlisted (0.01s)
FAIL
```

</details>

## misc/gnopreview/comment.go:86 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L86) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L86) · Warning
This note tells the reader the changed realms are always kept while [the truncation has no floor at their count](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L259), so on a sweep of the 58 realm directories under `examples/gno.land/r` it stands over 33 changed realms the cap dropped, and it is the only line separating a missing realm from a rendered one.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Asserts the cap note matches what the cap did. A gnomod.toml bump over every
// realm in examples/ at the default cap of 25 drops 33 of the 58 changed realms
// while the note says they are always kept. Fails at this head.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func judge67Sweep(t *testing.T) []string {
	t.Helper()
	var changed []string
	err := filepath.WalkDir("../../examples/gno.land/r", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "gnomod.toml" {
			changed = append(changed, "examples/"+filepath.ToSlash(strings.TrimPrefix(p, "../../examples/")))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestJudge67CapNoteContradictsTheDrop(t *testing.T) {
	changed := judge67Sweep(t)
	plan, err := BuildPlan("../..", changed, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	body := Comment(plan, "https://example.test/pr-1", "1")
	const note = "the changed realms are always kept"
	dropped := 0
	for _, r := range plan.ChangedRealms {
		if !contains(plan.Realms, r) {
			dropped++
		}
	}
	t.Logf("changed realms dropped by the cap=%d Dropped=%d", dropped, plan.Dropped)
	if dropped > 0 && strings.Contains(body, note) {
		t.Errorf("cap dropped %d of %d changed realms, yet the comment says %q",
			dropped, len(plan.ChangedRealms), note)
	}
}
EOF
cd misc/gnopreview && go test -run TestJudge67CapNote -v .
rm zz_run_test.go
```

The note holds in no run where the cap bites.

```
=== RUN   TestJudge67CapNoteContradictsTheDrop
    zz_run_test.go:87: changed realms dropped by the cap=33 Dropped=33
    zz_run_test.go:90: cap dropped 33 of 58 changed realms, yet the comment says "the changed realms are always kept"
--- FAIL: TestJudge67CapNoteContradictsTheDrop (0.01s)
FAIL
```

</details>

## misc/gnopreview/comment.go:106 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L106) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L106) · Warning
The [empty-`Before` branch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L106-L114) prints one unlabelled screenshot, so a [merge-base render that failed](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L231-L280) reads exactly like a realm that has no baseline to compare against.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_baseline_test.go <<'EOF'
package main

import (
	"strings"
	"testing"
)

// The state main.go leaves behind when renderBase returns nil: the realm exists
// at the merge base, so newRealms does not mark it New, and no before shot was
// taken. ScreenshotPairs still appends the pair, Before == "" and New == false.
func TestB5BaselineLossIsUnannounced(t *testing.T) {
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/x/leaf"},
		Pairs: []ShotPair{
			{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/"},
		},
	}
	got := Comment(p, "https://example.test/pr-1", "1")
	for _, unwanted := range []string{"before", "merge base", "baseline", "compare", "unavailable"} {
		if strings.Contains(strings.ToLower(got), unwanted) {
			t.Errorf("comment mentions %q, so the reader is told: %s", unwanted, got)
		}
	}
}

// The genuinely-new realm renders the same markup plus one <sub> note. Strip the
// note and the two bodies are identical, so the note is the only signal a reader
// has, and the failed-baseline case does not carry it.
func TestB5NewRealmAndFailedBaselineAreTheSameMarkup(t *testing.T) {
	mk := func(isNew bool) string {
		p := &Plan{
			ChangedRealms: []string{"gno.land/r/x/leaf"},
			Realms:        []string{"gno.land/r/x/leaf"},
			Pairs: []ShotPair{
				{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/", New: isNew},
			},
		}
		return Comment(p, "https://example.test/pr-1", "1")
	}
	newRealm, failedBase := mk(true), mk(false)
	const note = "\n\n<sub>New in this PR — nothing to compare against.</sub>"
	if strings.ReplaceAll(newRealm, note, "") != failedBase {
		t.Fatalf("markup differs beyond the note:\nnew:\n%s\nfailed:\n%s", newRealm, failedBase)
	}
}
EOF
cd misc/gnopreview && go test -run 'B5Baseline|B5NewRealm' -v .
rm zz_baseline_test.go
```

Both cases pass, which is the finding: the body a reviewer reads is the same markup either way, and the only sentence distinguishing them is the one `New` earns.

```text
--- PASS: TestB5BaselineLossIsUnannounced
--- PASS: TestB5NewRealmAndFailedBaselineAreTheSameMarkup
```

`ShotPair` states this invariant for the `New` flag and keeps none for the silent case: ["claiming a realm is new when we simply did not look would be a lie in the comment"](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L38-L43). `renderBase` returns a nil crawler on five paths, an empty `-base-root`, a `startGnodev` error, a `waitReady` timeout, a `base.Run` error and a `base.Write` error, each reporting to stderr alone.

</details>

## misc/gnopreview/comment.go:17 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L17) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L17) · Warning
The crawl [skips any page that answers non-200 and still returns nil](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129-L137), so a realm that never rendered keeps its render, source and help links in the posted comment, while [the index page](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L168-L171) beside it drops that realm.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_run_test.go <<'EOF'
// Asserts the sticky comment links only realms the crawl captured.
// gnoweb answers 404 for one realm, Run skips it and returns nil, and the
// comment links it three times. A PASS is the finding.
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentLinksRealmsTheCrawlNeverCaptured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, "/r/x/broken") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `<html><head></head><body>ok</body></html>`)
	}))
	defer srv.Close()

	plan := &Plan{
		ChangedRealms: []string{"gno.land/r/x/ok", "gno.land/r/x/broken"},
		Realms:        []string{"gno.land/r/x/ok", "gno.land/r/x/broken"},
		Dirs:          []string{"examples/gno.land/r/x/ok", "examples/gno.land/r/x/broken"},
	}
	c := &Crawler{Base: srv.URL, Realms: plan.Realms, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.pages["/r/x/broken"]; ok {
		t.Fatal("the broken realm was captured; the fixture is wrong")
	}

	const base = "https://gnolang.github.io/gno-previews/pr-1"
	index := Index(plan, c)
	comment := Comment(plan, base, "1")

	if strings.Contains(index, "r/x/broken") {
		t.Error("Index listed the realm that was never captured")
	}
	t.Log("index.html omits the realm the crawl never captured")

	for _, want := range []string{
		base + "/r/x/broken/",
		base + "/r/x/broken/_t/source/",
		base + "/r/x/broken/_t/help/",
	} {
		if !strings.Contains(comment, want) {
			t.Errorf("comment is missing %q", want)
			continue
		}
		t.Logf("sticky comment links %s - nothing was written there", want)
	}
	if strings.Contains(comment, "not rendered") || strings.Contains(comment, "failed") {
		t.Error("the comment did warn about the realm; the finding does not hold")
	}
}
EOF
cd misc/gnopreview && go test -run TestCommentLinksRealmsTheCrawlNeverCaptured -v .
rm zz_run_test.go
```

The run is green, and each assertion it passes is a link the comment publishes to a directory the snapshot does not contain.

```
=== RUN   TestCommentLinksRealmsTheCrawlNeverCaptured
  ✓ /r/x/ok
  ! /r/x/broken: HTTP 404
  ! /r/x/broken$source: HTTP 404
  ! /r/x/broken$help: HTTP 404
  ✓ /r/x
    zz_run_test.go:66: index.html omits the realm the crawl never captured
    zz_run_test.go:77: sticky comment links https://gnolang.github.io/gno-previews/pr-1/r/x/broken/ - nothing was written there
    zz_run_test.go:77: sticky comment links https://gnolang.github.io/gno-previews/pr-1/r/x/broken/_t/source/ - nothing was written there
    zz_run_test.go:77: sticky comment links https://gnolang.github.io/gno-previews/pr-1/r/x/broken/_t/help/ - nothing was written there
--- PASS: TestCommentLinksRealmsTheCrawlNeverCaptured (0.00s)
ok  	github.com/gnolang/gno/misc/gnopreview	0.004s
```

</details>

## misc/gnopreview/crawl.go:40 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L40) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L40) · Warning
[`attrRe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L40) matches `href` and `src` only, so every `$help` page publishes the crawl-time `http://127.0.0.1:<port>` origin in the [params form's `action`](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/components/views/action.html#L104) and the [copy-link button](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/components/views/action.html#L84), both built from [the request origin](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/handler_http.go#L390). Stripping that origin before [`mapURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L381) and widening the pattern to the attributes gnoweb builds from it keeps the published link inside the preview.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_help_origin_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// action.html builds both attributes from data.Origin, the scheme+host the
// crawler itself dialled. Neither attribute is href or src.
const helpBody = `<html><head><title>x</title></head><body>
<a href="/r/x/y$source">source</a>
<button data-controller="copy" data-copy-text-value="%[1]s/r/x/y$help&func=Foo">anchor</button>
<form class="params" method="GET" action="%[1]s/r/x/y$help&amp;func=Foo"></form>
</body></html>`

func TestSnapshotKeepsCrawlTimeOrigin(t *testing.T) {
	var origin string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/r/x/y$help":
			fmt.Fprintf(w, helpBody, origin)
		default:
			fmt.Fprint(w, `<html><head></head><body><a href="/r/x/y$help">help</a></body></html>`)
		}
	}))
	defer srv.Close()
	origin = srv.URL

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := t.TempDir()
	if err := c.Write(out, ""); err != nil {
		t.Fatalf("Write: %v", err)
	}
	f, ok := c.FileOf("/r/x/y$help")
	if !ok {
		t.Fatal("the $help seed was not captured")
	}
	b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(f)))
	if err != nil {
		t.Fatal(err)
	}
	if host := strings.TrimPrefix(srv.URL, "http://"); strings.Contains(string(b), host) {
		t.Errorf("published snapshot still points at the crawl-time gnodev %s:\n%s", host, string(b))
	}
}
EOF
(cd misc/gnopreview && go test -run TestSnapshotKeepsCrawlTimeOrigin -v ./...)
rm -f misc/gnopreview/zz_help_origin_test.go
```

The written page keeps the crawler's own host, which is the URL the Link button hands a reviewer:

```
--- FAIL: TestSnapshotKeepsCrawlTimeOrigin (0.01s)
    zz_help_origin_test.go:54: published snapshot still points at the crawl-time gnodev 127.0.0.1:43095:
        # …
        <button data-controller="copy" data-copy-text-value="http://127.0.0.1:43095/r/x/y$help&func=Foo">anchor</button>
        <form class="params" method="GET" action="http://127.0.0.1:43095/r/x/y$help&amp;func=Foo"></form>
        # … the sibling href on the same page became ../../../../../r/x/y/_t/source/
FAIL
```

</details>

## misc/gnopreview/crawl.go:126-128 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L126-L128) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L126) · Warning
Reaching the cap returns an error that [`render()` propagates before `c.Write`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L134-L138), so every page already captured is thrown away and a pull request whose realms render more than `-max-pages` pages gets a red render job and no preview, where the [README calls the 400-page default a backstop](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L118). Stopping the dequeue at the cap and carrying the truncation into the comment, as [`plan.Dropped`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258) already carries a realm left out, keeps the snapshot.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_cap_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPageCapDiscardsTheCrawl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/r/x/y", "/r/x/z":
			fmt.Fprint(w, `<!doctype html><html><head></head><body>ok</body></html>`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := &Crawler{
		Base:     srv.URL,
		Realms:   []string{"gno.land/r/x/y", "gno.land/r/x/z"},
		Live:     "https://gno.land",
		MaxPages: 2,
	}
	if err := c.Run(); err != nil {
		t.Fatalf("the cap aborted the crawl, and render() drops the %d captured page(s)", len(c.pages))
	}
}
EOF
cd misc/gnopreview && go test -run TestPageCapDiscardsTheCrawl -v .
cd ../.. && rm misc/gnopreview/zz_cap_test.go
```

The failure is the finding: two pages are captured and none of them reaches the artifact.

```text
--- FAIL: TestPageCapDiscardsTheCrawl
    zz_cap_test.go:29: the cap aborted the crawl, and render() drops the 2 captured page(s)
FAIL
```

</details>

## misc/gnopreview/crawl.go:134-136 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L134-L136) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L134) · Warning
A realm page answering anything but 200 leaves one stderr line and never enters `c.pages`, `Run` [returns nil regardless](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L153), and [`mapURL`'s last branch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L405) then rewrites every link to that realm as `c.Live`, so a reviewer opening a realm the branch broke reads the deployed one and calls the change fine. Collecting the non-200 URLs and naming them in the [index](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L170) and the comment reports the loss.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_drop_test.go <<'EOF'
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailedPageFallsBackToLive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/r/demo/a":
			io.WriteString(w, `<html><head></head><body><a href="/r/demo/b">b</a></body></html>`)
		case "/r/demo/b": // what gnoweb serves when the realm's Render panics
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/demo/a", "gno.land/r/demo/b"},
		MaxPages:   50,
		Live:       "https://gno.land",
		RenderOnly: true,
	}
	if err := c.Run(); err != nil {
		t.Fatalf("Run reported the broken realm: %v", err)
	}
	if body := c.rewrite(c.pages["/r/demo/a"]); strings.Contains(body, `href="https://gno.land/r/demo/b"`) {
		t.Fatalf("Run() == nil with %d of 2 realms captured, and the link reads %q",
			len(c.pages), `href="https://gno.land/r/demo/b"`)
	}
}
EOF
cd misc/gnopreview && go test -run TestFailedPageFallsBackToLive -v .
cd ../.. && rm misc/gnopreview/zz_drop_test.go
```

The failure is the finding: the crawl reports success and the preview page points at production.

```text
--- FAIL: TestFailedPageFallsBackToLive
    zz_drop_test.go:35: Run() == nil with 1 of 2 realms captured, and the link reads "href=\"https://gno.land/r/demo/b\""
FAIL
```

</details>

## misc/gnopreview/crawl.go:153 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L153) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L153) · Warning
`return nil` does not depend on how many pages were captured and [`render()` gates `Write`, `Index` and `Comment` on nothing](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L134-L158), so a gnodev that answers the readiness probe and then fails every fetch publishes a preview whose [index header counts `len(p.Realms)`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L195) as rendered above an empty list, with the [gnodev log already deleted](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L103). A floor on the captured count, and an index counting the realms present in `c.pages`, make an empty crawl visible.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_empty_test.go <<'EOF'
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmptyCrawlStillPublishes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatalf("Run reported the empty crawl: %v", err)
	}
	plan := &Plan{
		Realms:        []string{"gno.land/r/x/y"},
		ChangedRealms: []string{"gno.land/r/x/y"},
		Dirs:          []string{"examples/gno.land/r/x/y"},
	}
	idx := Index(plan, c)
	cm := Comment(plan, "https://example.github.io/preview/6194", "6194")
	if !strings.Contains(idx, "<li>") && cm != "" {
		t.Fatalf("Run() == nil with %d page(s); the index lists no realm and comment.md is %d bytes",
			len(c.pages), len(cm))
	}
}
EOF
cd misc/gnopreview && go test -run TestEmptyCrawlStillPublishes -v .
cd ../.. && rm misc/gnopreview/zz_empty_test.go
```

The failure is the finding: nothing was captured, and the index and the sticky comment are produced anyway.

```text
--- FAIL: TestEmptyCrawlStillPublishes
    zz_empty_test.go:28: Run() == nil with 0 page(s); the index lists no realm and comment.md is 412 bytes
FAIL
```

</details>

## SKIP misc/gnopreview/crawl.go:126 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L126) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L126) · Warning
The cap check returns rather than breaking, so the pages already captured are discarded and a wide pull request loses the whole preview where [`plan.Dropped`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258) meets the same overflow by truncating and [naming what was left out](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L85-L87).

Skipped: the same edit as the section on `misc/gnopreview/crawl.go:126-128`, which carries the repro.

## misc/gnopreview/crawl.go:486-491 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L486-L491) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L486) · Missing test
Missing test: `urlToFile` slugs the render arguments and the tab query and `path.Join`s the base unchanged, and the four inputs [`TestURLToFileIsSafe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L243-L258) walks all put their dots after `:` or `$`, so the one position with no slug is the one the `No output may escape its realm directory` assertion never reaches.

<details>
<summary>test cases</summary>

```go
// A base carrying ".." has no slug, so it is the one position the loop above
// never exercises. path.Join folds it and the output leaves the tree writePages
// joins onto.
for _, u := range []string{"/../../etc/passwd", "/r/x/y/../../../../etc/passwd"} {
	got := urlToFile(u)
	if strings.HasPrefix(got, "../") || strings.Contains(got, "/../") {
		t.Errorf("urlToFile(%q) = %q — escaped the output tree", u, got)
	}
}
```

</details>

## SKIP misc/gnopreview/crawl.go:354 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L354) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L354) · Warning
[`rewrite`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L354) walks only the `href` and `src` attributes, so the `$help` page [seeded for every realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L92) ships with the loopback origin in its form `action` and its copy-link button, against [README's claim that every absolute URL is rewritten](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L99).

Skipped: the same edit at [`attrRe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L40) closes it, and that section carries the finding.

## misc/gnopreview/crawl.go:402 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L402) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L402) · Warning
The `c.pages` lookup keys on the exact string and [`canonicalURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L287-L298) leaves a trailing slash alone, so the `Browse` link gnoweb emits as [`href="{{ .Link }}/"`](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/components/views/directory.html#L28) misses the directory page [`Seeds` captured without the slash](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L95-L97) and sends the reviewer to gno.land, where a realm the branch adds does not exist. Normalising the slash in `canonicalURL` makes both spellings one key, and it retires the [`_dir` branch of `urlToFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L496-L498) with the [test case pinning it](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L61), which no crawled URL can reach while [`inScope` refuses a trailing-slash base](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L191-L193).

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_dir_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCapturedDirectoryIsLinkedOffsite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head></head><body><a href="/r/gnoland/">Browse</a></body></html>`)
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/gnoland/home"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.pages["/r/gnoland"]; !ok {
		t.Fatalf("directory seed not captured; pages=%v", c.order)
	}
	if got := c.mapURL("/r/gnoland/", "../../"); strings.HasPrefix(got, "https://gno.land") {
		t.Fatalf("the directory page is in the snapshot and mapURL(%q) = %q", "/r/gnoland/", got)
	}
}
EOF
cd misc/gnopreview && go test -run TestCapturedDirectoryIsLinkedOffsite -v .
cd ../.. && rm misc/gnopreview/zz_dir_test.go
```

The failure is the finding: the page is captured and the link to it leaves the snapshot.

```text
--- FAIL: TestCapturedDirectoryIsLinkedOffsite
    zz_dir_test.go:25: the directory page is in the snapshot and mapURL("/r/gnoland/") = "https://gno.land/r/gnoland/"
FAIL
```

</details>

## misc/gnopreview/crawl.go:405 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L405) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L405) · Warning
`return c.Live + target + frag` catches every in-scope page missing from `c.pages`, a wider set than the links that leave the preview the [README describes](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L99):

- a page [dropped on a non-200 or a transport error](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L130-L137), so the pull request's own preview URL serves the deployed realm.
- a `$source&file=<name>` page outside `plan.ChangedFiles`, which [`wantFile` refuses on `0 < 0`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L231) for every pull request that leaves gnoweb alone, since [`fileBudget()` returns 0 there](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L217-L221), so the source tab of a realm the branch adds 404s on gno.land.

Rewriting an uncaptured in-scope target to a generated stub keeps production content out of the preview.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_fallback_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDroppedPageLinksToProduction(t *testing.T) {
	render := `<!doctype html><html><head></head><body>` +
		`<a href="/r/x/y$source">source</a>` +
		`<a href="/r/x/y$source&amp;file=a.gno">a.gno</a>` +
		`</body></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/r/x/y":
			fmt.Fprint(w, render)
		case "/r/x/y$source":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			fmt.Fprint(w, `<!doctype html><html><head></head><body>ok</body></html>`)
		}
	}))
	defer srv.Close()

	c := &Crawler{Base: srv.URL, Realms: []string{"gno.land/r/x/y"}, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	got := c.rewrite(c.pages["/r/x/y"])
	if strings.Contains(got, `href="https://gno.land/r/x/y$source"`) &&
		strings.Contains(got, `href="https://gno.land/r/x/y$source&amp;file=a.gno"`) {
		t.Fatal("the lost page and the budget-refused file page both link to production")
	}
}
EOF
cd misc/gnopreview && go test -run TestDroppedPageLinksToProduction -v .
cd ../.. && rm misc/gnopreview/zz_fallback_test.go
```

The failure is the finding: both links in the snapshot point at gno.land.

```text
--- FAIL: TestDroppedPageLinksToProduction
    zz_fallback_test.go:35: the lost page and the budget-refused file page both link to production
FAIL
```

</details>

## misc/gnopreview/go.mod:1 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/go.mod#L1) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/go.mod#L1) · Warning
This `module` line puts `gnopreview` outside the root module, where [the root `fix` target's loop](https://github.com/gnolang/gno/blob/ecf7af0/Makefile#L101) does not list it while [the `ci-dir-misc` matrix](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/ci-dir-misc.yml#L42) runs the `go fix` check over it, so [the instruction that check prints](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/_ci-go.yml#L67) reaches no file here.

## SKIP misc/gnopreview/main.go:126 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L126) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L126) · Warning
The `-max-pages` help text calls the value a cap on crawled pages, and overflowing it deletes the snapshot rather than truncating it: the crawl queue grows with whatever the changed realm links to, so the realms with the most to show are the ones that lose their preview.

SKIP: one edit in [`Crawler.Run`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L126-L128) settles this and the section on `main.go:134`, so it posts once.

## SKIP misc/gnopreview/main.go:111 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L111) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L111) · Suggestion
Suggestion: this guard asks whether anything is worth previewing while the code under it depends on the realm list being non-empty, and the two answers differ exactly on a gnoweb plan with no realms.

SKIP: one edit settles this and the section on `main.go:131`, so it posts once.

## misc/gnopreview/main.go:134 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L134) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L134) · Warning
Reaching the page cap makes [`Crawler.Run`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L127) return an error instead of stopping at the cap, so `render` throws away the pages already captured and [the artifact upload never runs](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L117-L118), while the realm cap beside it [drops the overflow and keeps going](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L257-L258).

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cd misc/gnopreview
cat > zz_pagecap_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Every /r/... path answers with a page linking to all six realms, so a crawl
// started on realm 0 reaches the others and the queue outlives the cap.
func serveRealms(t *testing.T, n int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b strings.Builder
		b.WriteString("<html><head></head><body>")
		for i := range n {
			fmt.Fprintf(&b, `<a href="/r/x/r%d">r%d</a>`, i, i)
		}
		b.WriteString("</body></html>")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(b.String()))
	})
	return httptest.NewServer(mux)
}

func TestPageCapTruncates(t *testing.T) {
	const n = 6
	srv := serveRealms(t, n)
	defer srv.Close()

	realms := make([]string, n)
	for i := range n {
		realms[i] = fmt.Sprintf("gno.land/r/x/r%d", i)
	}

	uncapped := &Crawler{Base: srv.URL, Realms: realms, MaxPages: 0, Live: "https://gno.land"}
	if err := uncapped.Run(); err != nil {
		t.Fatalf("MaxPages=0: %v", err)
	}
	t.Logf("no cap: %d pages captured", len(uncapped.pages))

	capped := &Crawler{Base: srv.URL, Realms: realms, MaxPages: 3, Live: "https://gno.land"}
	if err := capped.Run(); err != nil {
		t.Errorf("MaxPages=3: want the crawl stopped at the cap, got error %q and the %d captured pages dropped", err, len(capped.pages))
	}
}
EOF
go test -count=1 -run TestPageCapTruncates -v .
rm zz_pagecap_test.go
```

The crawl that would have kept 19 pages keeps none: the cap turns into an error and the three pages already fetched go with it.

```text
=== RUN   TestPageCapTruncates
    zz_pagecap_test.go:41: no cap: 19 pages captured
    zz_pagecap_test.go:45: MaxPages=3: want the crawl stopped at the cap, got error "page cap 3 reached (queue still had 31)" and the 3 captured pages dropped
--- FAIL: TestPageCapTruncates (0.00s)
FAIL
FAIL	github.com/gnolang/gno/misc/gnopreview	0.005s
```

The default ceiling is 400 pages, and a changed realm's per-file source pages are [exempt from the per-file budget](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L227-L230), so a wide `examples/` change has no page ceiling under it.

</details>

## misc/gnopreview/main.go:216 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L216) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L216) · Suggestion
Suggestion: `fileBudget` has [one call site](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L129) and returns the zero value of the field it feeds in the other branch, so `if plan.Gnoweb { c.FileBudget = GnowebFileBudget }` written under the `Crawler` literal carries the same behaviour and the same comment in five fewer lines.

## SKIP misc/gnopreview/main.go:156 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L156) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L156) · Suggestion
Suggestion: the comment written here reserves a slot for the four-view chrome sample in the gnoweb-and-realms case, and the plan handed to it carries no shots for that case.

SKIP: one edit settles this and the section on `main.go:147`, so it posts once.

## misc/gnopreview/main.go:199 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L199) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L199) · Warning
The exit channel is read only inside [`waitReady`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L597-L601), so a gnodev that dies during the crawl leaves [`Crawler.Run`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L130-L132) printing a line per refused fetch and returning nil, and the job publishes a snapshot missing most of the realms its comment links.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_node_death_test.go <<'EOF'
// Asserts Crawler.Run reports a node that dies mid-crawl.
// The server answers the first seed, then closes its listener.
// Fails at this head: Run returns nil with a partial capture.
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNodeDeathMidCrawlIsSilent(t *testing.T) {
	var (
		srv  *httptest.Server
		once sync.Once
		gone = make(chan struct{})
	)
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		io.WriteString(w, "<html><body>rendered</body></html>")
		once.Do(func() {
			_ = srv.Listener.Close()
			close(gone)
		})
	}))
	defer func() { recover() }()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/demo/a", "gno.land/r/demo/b", "gno.land/r/demo/c"},
		RenderOnly: true,
		MaxPages:   3,
		Live:       "https://gno.land",
	}
	seeds := len(c.Seeds())
	err := c.Run()
	<-gone
	t.Logf("seeds=%d captured=%d err=%v", seeds, len(c.pages), err)
	if err != nil {
		t.Fatalf("node death surfaced as an error: %v", err)
	}
	if len(c.pages) >= seeds {
		t.Fatalf("every seed was captured (%d of %d)", len(c.pages), seeds)
	}
}
EOF
cd misc/gnopreview && go test -count=1 -run TestNodeDeathMidCrawl -v ./...
rm zz_node_death_test.go
```

The assertion that the dead node surfaces as an error never fires: one realm of three is captured and `Run` hands back nil, which is the state `render` continues from into `Write`, `Index` and `comment.md`.

```
=== RUN   TestNodeDeathMidCrawlIsSilent
  ✓ /r/demo/a
  ! /r/demo/b: Get "http://127.0.0.1:32857/r/demo/b": dial tcp 127.0.0.1:32857: connect: connection refused
  ! /r/demo/c: Get "http://127.0.0.1:32857/r/demo/c": dial tcp 127.0.0.1:32857: connect: connection refused
    zz_node_death_test.go:40: seeds=3 captured=1 err=<nil>
--- PASS: TestNodeDeathMidCrawlIsSilent (0.00s)
PASS
ok  	github.com/gnolang/gno/misc/gnopreview	0.004s
```

[`Index`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L166-L172) lists only the realms that were captured, while the sticky comment links every realm in the plan, so the two disagree with nothing in the job saying which pages are missing.

</details>

## misc/gnopreview/plan.go:256-258 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L256) · Warning
[`ChangedRealms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L229-L235) is filled before the cap truncates `realms`, so a pull request changing more than [25](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L29) realms gets a comment [linking every changed realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L60-L64) including the ones the snapshot never rendered, beneath [the note that the changed realms are always kept](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L86).

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_cap_test.go <<'EOF'
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The cap is set to 1 and all three realms are changed by the pull request.
func TestCapDropsChangedRealms(t *testing.T) {
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
	all := []string{"gno.land/r/x/a", "gno.land/r/x/b", "gno.land/r/x/c"}
	var changed []string
	for _, r := range all {
		write("examples/"+r+"/gnomod.toml", "module = \""+r+"\"\n")
		write("examples/"+r+"/lib.gno", "package x\n")
		changed = append(changed, "examples/"+r+"/lib.gno")
	}
	plan, err := BuildPlan(root, changed, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChangedRealms=%v Realms=%v Dropped=%d", plan.ChangedRealms, plan.Realms, plan.Dropped)
	body := Comment(plan, "https://preview.test/pr-1", "1")
	for _, r := range plan.ChangedRealms {
		rendered := false
		for _, got := range plan.Realms {
			rendered = rendered || got == r
		}
		linked := strings.Contains(body, "https://preview.test/pr-1"+urlOf(r)+"/")
		t.Logf("changed realm %-16s rendered=%-5v linked in comment=%v", r, rendered, linked)
		if !rendered && linked {
			t.Errorf("comment links %s, which the cap dropped", r)
		}
	}
	if strings.Contains(body, "the changed realms are always kept") {
		t.Errorf("comment says the changed realms are always kept, and %d were dropped", plan.Dropped)
	}
}
EOF
cd misc/gnopreview && go test -run TestCapDropsChangedRealms -v .
rm -f zz_cap_test.go
```

Two of the three changed realms are absent from `Realms`, so `gnodev` never serves them, and the generated comment links both of them and prints the always-kept note:

```text
=== RUN   TestCapDropsChangedRealms
    zz_cap_test.go:34: ChangedRealms=[gno.land/r/x/a gno.land/r/x/b gno.land/r/x/c] Realms=[gno.land/r/x/a] Dropped=2
    zz_cap_test.go:42: changed realm gno.land/r/x/a   rendered=true  linked in comment=true
    zz_cap_test.go:42: changed realm gno.land/r/x/b   rendered=false linked in comment=true
    zz_cap_test.go:44: comment links gno.land/r/x/b, which the cap dropped
    zz_cap_test.go:42: changed realm gno.land/r/x/c   rendered=false linked in comment=true
    zz_cap_test.go:44: comment links gno.land/r/x/c, which the cap dropped
    zz_cap_test.go:48: comment says the changed realms are always kept, and 2 were dropped
--- FAIL: TestCapDropsChangedRealms (0.00s)
FAIL
FAIL	github.com/gnolang/gno/misc/gnopreview	0.003s
```

The stable sort at [plan.go:253-255](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L253-L255) orders changed realms first, which holds the property only while the changed realms number at most `maxRealms`.

</details>

## SKIP misc/gnopreview/plan.go:258 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L258) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L258) · Warning
`realms = realms[:maxRealms]` is the only place the cap lands, and [`Plan.Dirs`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L271-L272) is built from the truncated list, so `gnodev` serves only the survivors while the comment links the rest.

Skipped: the same edit as the section on [plan.go:256-258](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258) resolves it.

## SKIP misc/gnopreview/plan.go:273 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L273) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L273) · Suggestion
Suggestion: the seed realms overshooting the cap are crawled against the shared page budget and named nowhere in the comment, since [comment.go:69](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L69) filters a seed out of the indirect list, so the operator gets more realms loaded than the cap asked for and no output showing them.

Skipped: it resolves in the same edit as the section at plan.go:263-268, which carries the overrun.

## misc/gnopreview/plan_test.go:209-212 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L209-L212) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L209) · Warning
Test: an assertion that a comment carries no screenshot grid passes whatever [`Comment`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L46-L54) writes when its fixture leaves `Plan.Shots` empty, since [`shotGrid`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L130-L133) returns the empty string for an empty slice.

- [`plan_test.go:210`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L210): the realm-only fixture carries no `Shots`, so `<table>` cannot appear.
- [`plan_test.go:257`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L257): the before/after fixture carries `Pairs` and no `Shots`, so the gnoweb sample cannot appear.

<details>
<summary>test cases</summary>

Both fixtures stay green with one `Shot` added, and both go red once `shotGrid` runs outside the `if p.Gnoweb` guard of the default branch.

```go
// TestCommentRealms
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		ChangedPkgs:   []string{"gno.land/p/x/base/v0"},
		Realms:        []string{"gno.land/r/x/leaf", "gno.land/r/x/other"},
		Dropped:       3,
		Shots:         []Shot{{File: "_shots/home.png", Label: "Home"}},
	}

// TestCommentBeforeAfter: the same field, and a check on the sample itself
		Shots: []Shot{{File: "_shots/home.png", Label: "Home"}},

	if strings.Contains(got, "_shots/home.png") {
		t.Errorf("realm change should not carry the gnoweb sample:\n%s", got)
	}
```

</details>

## SKIP misc/gnopreview/shots.go:166 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L166) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L166) · Warning
A realm page the pull request changed is photographed with its markdown image references intact, so one external host that accepts the connection and never answers holds the browser open for as long as the job lives.

Skipped: the same edit as the deadline finding on `misc/gnopreview/shots.go:171` closes it, and the reach it adds is carried in that section's collapsed block.

## misc/gnopreview/shots.go:171 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L171) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L171) · Warning
[`chromeShot`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L171-L183) gives the browser command no context and blocks in [`CombinedOutput`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L184), so a headless Chrome that never exits pins [the render step](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L91-L103) until the runner's own default timeout, which no `timeout-minutes` in these workflows shortens.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_shot_deadline_test.go <<'EOF'
// Asserts chromeShot is bounded by a deadline.
// The stub browser starts and never exits.
// Fails at this head: the call is still blocked after 6s.
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChromeShotHasNoDeadline(t *testing.T) {
	dir := t.TempDir()
	stub := filepath.Join(dir, "hung-chrome")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- chromeShot(stub, "http://127.0.0.1:1/never", filepath.Join(dir, "out.png"))
	}()
	select {
	case err := <-done:
		t.Fatalf("chromeShot returned on a hung browser: %v", err)
	case <-time.After(6 * time.Second):
		t.Log("chromeShot still blocked after 6s: no deadline bounds exec.Command")
	}
}
EOF
cd misc/gnopreview && go test -count=1 -run TestChromeShotHasNoDeadline -v ./...
rm zz_shot_deadline_test.go
```

The branch asserting a return never runs: the call is still blocked when the test gives up, so the only thing that ends a wedged browser is the process being killed from outside.

```
=== RUN   TestChromeShotHasNoDeadline
    zz_shot_deadline_test.go:27: chromeShot still blocked after 6s: no deadline bounds exec.Command
--- PASS: TestChromeShotHasNoDeadline (6.00s)
PASS
ok  	github.com/gnolang/gno/misc/gnopreview	6.003s
```

`--virtual-time-budget=4000` stops the page's virtual clock and not the process. The page being photographed is realm HTML from the pull request, and an external image reference in a realm's markdown survives into the captured page, so a single host that accepts the connection and never answers is enough to reach this state on a fork's branch. `cfg.timeout` reaches [`waitReady`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L131) alone and bounds gnodev readiness, not the browser.

</details>

## SKIP misc/gnopreview/shots.go:189 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L189) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L189) · Warning
The only post-run check on the browser is [`os.Stat` on the written file](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187-L189), which is reached only once the process exits, so nothing bounds a browser that starts and never exits.

Skipped: the same edit as the deadline finding on `misc/gnopreview/shots.go:171` closes it, and two sections resolving in one edit are one finding.

## misc/gnopreview/comment.go:30 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L30) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L30) · Missing test
Missing test: no case passes an empty `baseURL`, the [value `-base-url` defaults to](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L71) and the one every local run takes through [`link`'s empty-base branch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L30-L32), so rewriting that branch to return the empty string deletes every realm bullet with the suite green.

<details>
<summary>test cases</summary>

```go
func TestCommentWithoutBaseURL(t *testing.T) {
	t.Run("gnoweb mode still lists every realm", func(t *testing.T) {
		p := &Plan{Gnoweb: true, Realms: gnowebSeedRealms,
			Shots: []Shot{{File: "home.png", Label: "home"}}}
		got := Comment(p, "", "7")
		// The godoc promises the comment "still lists what was rendered".
		for _, r := range gnowebSeedRealms {
			if !strings.Contains(got, "- `"+r+"`\n") {
				t.Errorf("realm %q missing from the no-base comment:\n%s", r, got)
			}
		}
		// Nothing may emit a root-relative src/href: GitHub resolves one against
		// github.com, and the body is posted by a privileged job.
		for _, frag := range []string{`src="/`, `href="/`, "](/"} {
			if strings.Contains(got, frag) {
				t.Errorf("no-base comment carries a root-relative link %q:\n%s", frag, got)
			}
		}
		// shotGrid has no URL to embed against, so it contributes nothing.
		if strings.Contains(got, "<img") {
			t.Errorf("no-base comment embeds an image:\n%s", got)
		}
	})

	t.Run("both mode keeps its headings", func(t *testing.T) {
		p := &Plan{Gnoweb: true,
			ChangedRealms: []string{"gno.land/r/demo/foo"},
			Realms:        []string{"gno.land/r/demo/foo", "gno.land/r/demo/bar"},
			ChangedPkgs:   []string{"gno.land/p/demo/x"},
			Pairs:         []ShotPair{{Realm: "gno.land/r/demo/foo", URL: "u", After: "a.png", New: true}}}
		got := Comment(p, "", "7")
		for _, want := range []string{"**Changed realms (1)**", "- `gno.land/r/demo/foo`\n", "- `gno.land/r/demo/bar`\n"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "<img") || strings.Contains(got, `href="/`) {
			t.Errorf("no-base comment carries an image or a root-relative link:\n%s", got)
		}
	})
}
```

The four `base == ""` decisions sit at [30](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L30), [100](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L100), [131](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L131) and [150](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L150); `go test -coverprofile` puts three of those blocks at zero hits.

</details>

## misc/gnopreview/comment.go:47 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L47) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L47) · Missing test
Missing test: no fixture sets `Gnoweb` together with a changed realm, so the [gnoweb-plus-realms branch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L47-L54) and [`isSeed`'s membership check](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L161) are executed by nothing.

<details>
<summary>test cases</summary>

```go
// Mode()=="both": gnoweb plus realm sources. No test in the package reaches this
// branch, so nothing pins the sentence, the homepage link or the seed filter.
func TestCommentBothMode(t *testing.T) {
	p := &Plan{
		Gnoweb:        true,
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/gnoland/home", "gno.land/r/x/leaf"},
		Pairs:         []ShotPair{{Realm: "gno.land/r/x/leaf", After: "_shots/a.png", URL: "r/x/leaf/"}},
	}
	got := Comment(p, "https://example.test/pr-3", "3")
	for _, want := range []string{
		"changes **gnoweb** and realm sources",
		"**[Open the preview homepage](https://example.test/pr-3/)**",
		"**Changed realms (1)**",
		"_shots/a.png",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("both-mode comment missing %q\n---\n%s", want, got)
		}
	}
	// gno.land/r/gnoland/home is a gnoweb seed, so isSeed drops it from the
	// indirect list; pin that, and pin that no empty table is emitted.
	if strings.Contains(got, "gno.land/r/gnoland/home") {
		t.Errorf("seed realm listed as indirect:\n%s", got)
	}
	if strings.Contains(got, "<table><tr></tr></table>") {
		t.Errorf("empty grid emitted:\n%s", got)
	}
}
```

The single `Gnoweb` fixture sets neither `ChangedRealms` nor `ChangedPkgs`, so [`Mode()`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L94-L102) answers `"gnoweb"` for it and the `"both"` arm stays at zero coverage.

</details>

## misc/gnopreview/comment.go:166 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L166) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L166) · Missing test
Missing test: no test calls `Index`, so the landing page [the comment sends every reviewer to](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L40) ships with its header count, its skip of uncaptured realms and its row hrefs unpinned.

<details>
<summary>test cases</summary>

```go
func TestIndexCountsWhatItLists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/r/x/broken") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `<html><head></head><body>ok</body></html>`)
	}))
	defer srv.Close()

	plan := &Plan{Realms: []string{"gno.land/r/x/ok", "gno.land/r/x/broken"}}
	c := &Crawler{Base: srv.URL, Realms: plan.Realms, Live: "https://gno.land"}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	idx := Index(plan, c)
	if got := strings.Count(idx, "<li>"); got != 1 {
		t.Fatalf("rows = %d, want 1", got)
	}
	if !strings.Contains(idx, "1 realm(s) rendered") {
		t.Errorf("the header counts the plan, not the rows it wrote")
	}
	if !strings.Contains(idx, `<a href="r/x/ok/">`) {
		t.Errorf("row href is not the directory the snapshot wrote:\n%s", idx)
	}
}
```

At this head the href and the row count hold and the header assertion fails.

```
=== RUN   TestIndexCountsWhatItLists
  ! /r/x/broken: HTTP 404
    zz_idx_test.go:31: the header counts the plan, not the rows it wrote
--- FAIL: TestIndexCountsWhatItLists (0.00s)
FAIL
```

</details>

## misc/gnopreview/crawl.go:109 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L109) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L109) · Missing test
Missing test: no test calls `Run`, so nothing pins the choice to [abort at the cap rather than truncate](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L126-L128), the order [`chargeFile` runs in relative to the fetch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L138), or the fact that a budget-refused page's links never reach the queue.

<details>
<summary>test cases</summary>

```go
func TestRunPageCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/r/x/y", "/r/x/z":
			fmt.Fprint(w, `<!doctype html><html><head></head><body>ok</body></html>`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	new := func(max int) *Crawler {
		return &Crawler{
			Base:     srv.URL,
			Realms:   []string{"gno.land/r/x/y", "gno.land/r/x/z"},
			Live:     "https://gno.land",
			MaxPages: max,
		}
	}

	uncapped := new(0)
	if err := uncapped.Run(); err != nil {
		t.Fatalf("Run() = %v; want nil", err)
	}
	if len(uncapped.pages) != 2 {
		t.Errorf("captured %d page(s); want 2", len(uncapped.pages))
	}

	// The cap keeps the pages it captured and reports the truncation.
	capped := new(1)
	if err := capped.Run(); err != nil {
		t.Errorf("Run() = %v; want nil with the crawl truncated", err)
	}
	if len(capped.pages) != 1 {
		t.Errorf("captured %d page(s); want 1", len(capped.pages))
	}
}
```

</details>

## misc/gnopreview/crawl.go:326 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L326) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L326) · Missing test
Missing test: no test calls `Write` or `writePages`, so nothing pins the [`c.Prefix` in the file path](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L142) that keeps [the before pass, written into the same output directory](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L270-L276), from overwriting the after pass, which would make every after screenshot a photograph of the merge base.

<details>
<summary>test cases</summary>

```go
func TestWritePagesHonoursPrefix(t *testing.T) {
	for _, tc := range []struct{ prefix, want string }{
		{"", "r/x/y/index.html"},
		{"_before", "_before/r/x/y/index.html"},
	} {
		dir := t.TempDir()
		u := "/r/x/y"
		c := &Crawler{Live: "https://gno.land", Prefix: tc.prefix}
		c.pages = map[string]*page{u: {
			URL:  u,
			File: path.Join(tc.prefix, urlToFile(u)),
			Body: "<html><head></head><body>ok</body></html>",
		}}
		c.order = []string{u}
		if err := c.writePages(dir); err != nil {
			t.Fatalf("writePages(%q) = %v", tc.prefix, err)
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(tc.want))); err != nil {
			t.Errorf("prefix %q: %v", tc.prefix, err)
		}
	}
}
```

</details>

## misc/gnopreview/crawl.go:370 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L370) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L370) · Missing test
Missing test: nothing ties [`robotsRe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L39) to [gnoweb's own robots tag](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/components/layouts/head.html#L38), so reordering that tag's attributes drops `setNoindex` to its `<head>` arm and every preview page carries `index, follow` and `noindex, nofollow` at once. [`TestRewriteAddsNoindex`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L178) re-types the tag as a literal, and the separate `go.mod` blocks an import but not a disk read.

<details>
<summary>test cases</summary>

```go
// The tag robotsRe has to match is gnoweb's own, not the copy re-typed in
// TestRewriteAddsNoindex. misc/gnopreview is a separate module, so read the
// template off disk rather than importing it.
func TestRobotsReMatchesGnowebHead(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../gno.land/pkg/gnoweb/components/layouts/head.html")
	if err != nil {
		t.Fatal(err)
	}
	if !robotsRe.MatchString(string(b)) {
		t.Error("robotsRe no longer matches head.html: setNoindex would leave gnoweb's robots meta in place and add a second, contradicting one")
	}
}
```

</details>

## misc/gnopreview/crawl.go:547 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L547) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L547) · Missing test
Missing test: no test calls `copyTree`, so neither the [`up` computed per asset from its own depth](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L569-L576) nor the [zero-rewrite warning](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L314-L316) is exercised, and that quantity is not the page-side one [`TestMapURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L122) covers, so a regression leaves every dynamic import in `js/index.js` resolving to nothing and the preview looking right with its controls dead.

<details>
<summary>test cases</summary>

```go
func TestCopyTreeRelativizesPublicRefs(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "js"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(rel, body string) {
		if err := os.WriteFile(filepath.Join(src, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("js/index.js", `import("/public/js/controller-a.js")`)
	write("index.js", `import("/public/js/controller-a.js")`)
	write("js/controller-a.js", "")

	n, err := copyTree(src, filepath.Join(dst, "public"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("rewrote %d reference(s); want 2", n)
	}
	// Each rewritten specifier resolves from the asset's own directory.
	for rel, want := range map[string]string{
		"public/js/index.js": "public/js/controller-a.js",
		"public/index.js":    "public/js/controller-a.js",
	} {
		b, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		spec := strings.TrimSuffix(strings.TrimPrefix(string(b), `import("`), `")`)
		got := path.Join(path.Dir(rel), spec)
		if got != want {
			t.Errorf("%s imports %q, resolving to %q; want %q", rel, spec, got, want)
		}
	}
}
```

</details>

## misc/gnopreview/crawl_test.go:129 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L129) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L129) · Missing test
Missing test: this hand-written `up` is what [`TestMapURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L122) passes in, and [`TestRewriteAddsNoindex`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L167) feeds bodies carrying no `href` or `src`, so [`rewrite`'s own depth derivation](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L350-L351), the value every published link and asset path resolves from, sits inside no assertion.

<details>
<summary>test cases</summary>

Assert the resolved target rather than the string: `path.Join(path.Dir(p.File), rewritten)` must equal the captured file's own directory, for a shallow page and for a `/_t/` page.

```go
func TestRewriteResolvesFromThePageDir(t *testing.T) {
	t.Parallel()
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{
		"/r/a/b": {File: "r/a/b/index.html"},
	}}
	for _, tc := range []struct{ file, href, resolves string }{
		{"r/x/y/index.html", "/r/a/b", "r/a/b"},
		{"r/x/y/index.html", "/public/css/main.css", "public/css/main.css"},
		{"r/x/y/_t/source/index.html", "/r/a/b", "r/a/b"},
		{"r/x/y/_t/source/index.html", "/public/js/controller-abc.js", "public/js/controller-abc.js"},
	} {
		body := `<!doctype html><html><head></head><body><a href="` + tc.href + `">x</a></body></html>`
		got := c.rewrite(&page{File: tc.file, Body: body})
		i := strings.Index(got, `href="`)
		if i < 0 {
			t.Fatalf("%s: no href survived the rewrite: %s", tc.file, got)
		}
		rel := got[i+len(`href="`):]
		rel = rel[:strings.Index(rel, `"`)]
		if resolved := path.Join(path.Dir(tc.file), rel); resolved != tc.resolves {
			t.Errorf("page %s: %q rewritten to %q, which resolves to %q; want %q",
				tc.file, tc.href, rel, resolved, tc.resolves)
		}
	}
}
```

The case these cover: with `depth` replaced by `depth+1` the package reports `ok`, while a page at `r/x/y/index.html` points `/r/a/b` at `../../../../r/a/b/`, one directory above the preview root, so every link and asset in the published tree 404s.

</details>

## misc/gnopreview/crawl_test.go:178 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L178) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L178) · Missing test
Missing test: [`robotsRe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L39) matches the tag only with `name` first and double-quoted, and this fixture copies that one spelling by hand from [gnoweb's layout](https://github.com/gnolang/gno/blob/ecf7af0/gno.land/pkg/gnoweb/components/layouts/head.html#L38), so reordering the two attributes there leaves the preview carrying `noindex, nofollow` beside gnoweb's `index, follow` with no test red.

<details>
<summary>test cases</summary>

Drive the assertion from shapes the layout could take, and match the tag on `<meta` plus a robots name attribute in any position and quote style.

```go
func TestNoindexSurvivesATagReshape(t *testing.T) {
	t.Parallel()
	anyRobots := regexp.MustCompile(`(?is)<meta[^>]*robots[^>]*>`)
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, tag := range []string{
		`<meta name="robots" content="index, follow" />`,
		`<meta content="index, follow" name="robots" />`,
		`<meta name='robots' content='index, follow' />`,
	} {
		got := c.rewrite(&page{File: "r/x/index.html", Body: `<html><head>` + tag + `<title>x</title></head></html>`})
		if n := len(anyRobots.FindAllString(got, -1)); n != 1 {
			t.Errorf("%s: %d robots metas, want 1", tag, n)
		}
		if strings.Contains(got, "index, follow") {
			t.Errorf("%s: gnoweb's index,follow survived", tag)
		}
	}
}
```

The first row passes. The second and third each leave 2 robots metas with `index, follow` intact, which is the outcome [the `setNoindex` comment](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L365-L369) rules out.

</details>

## misc/gnopreview/crawl_test.go:181-192 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L181-L192) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L181) · Missing test
Missing test: these four assertions check presence, a count of one, a single robots meta and the absence of `index, follow`, all of which [the prepend fallback](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L377) satisfies on its own, so [the `<head>` insertion](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L374-L376) can die and every page ships the tag ahead of `<!doctype html>`, in quirks mode.

<details>
<summary>test cases</summary>

Assert where the tag landed: between `<head` and `</head>`, with no byte before the doctype.

```go
func TestNoindexLandsInsideHead(t *testing.T) {
	t.Parallel()
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, tc := range []struct{ name, body string }{
		{"normal head", `<!doctype html><html><head><title>x</title></head><body>hi</body></html>`},
		{"head with attrs", `<html><head lang="en"><title>x</title></head></html>`},
	} {
		got := c.rewrite(&page{File: "r/x/index.html", Body: tc.body})
		at := strings.Index(got, noindexTag)
		lower := strings.ToLower(got)
		open, closed := strings.Index(lower, "<head"), strings.Index(lower, "</head>")
		if at < open || at > closed {
			t.Errorf("%s: robots meta at %d is outside <head> (%d..%d)", tc.name, at, open, closed)
		}
		// Anything before the doctype drops the page into quirks mode, so the
		// preview boxes differently than gno.land itself.
		if d := strings.Index(lower, "<!doctype"); d > 0 {
			t.Errorf("%s: %d bytes precede the doctype: %.60s", tc.name, d, got)
		}
	}
}
```

The case these cover: with the `<head>` branch disabled the package reports `ok`, and both rows above then fail on the placement.

</details>

## misc/gnopreview/crawl_test.go:247 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L247) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L247) · Missing test
Missing test: every input here puts its dots after `:` or `$`, the two positions [`slug`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L520) neutralises, so no case reaches the base that [`urlToFile` joins verbatim](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L491-L495) and the stated guarantee rests on [`inScope`'s exact realm match](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L195-L200) instead.

<details>
<summary>test cases</summary>

Extend the loop with a base-position case.

```go
	// The base is joined as given, so the escape guarantee has to be asserted
	// on it too, not only on the slugged halves.
	for _, u := range []string{"/r/x/y/../../../../etc/passwd", "/r/x/y/../z"} {
		if got := urlToFile(u); strings.HasPrefix(got, "../") {
			t.Errorf("urlToFile(%q) = %q: escaped the output tree", u, got)
		}
	}
```

The second row passes, at `r/x/z/index.html`. The first returns `../etc/passwd/index.html`.

</details>

## misc/gnopreview/plan.go:271-276 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L271-L276) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L271) · Missing test
Missing test: `Dirs` and `ChangedFiles` are the two `BuildPlan` outputs gnodev and the crawler consume and no assertion reads either, while [main.go:239-241](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L239-L241) pairs `plan.Dirs[i]` with `plan.Realms[i]` on an alignment held up only by this loop sitting below the seed append, so moving or extending either block stats another realm's directory and renders the wrong package under the right name with the suite green.

<details>
<summary>test cases</summary>

```go
func TestBuildPlanDirsPairWithRealms(t *testing.T) {
	t.Parallel()
	root := fakeRepo(t)
	got, err := BuildPlan(root, []string{
		"gno.land/pkg/gnoweb/app.go",
		"examples/gno.land/r/x/leaf/lib.gno",
	}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Dirs) != len(got.Realms) {
		t.Fatalf("Dirs = %v; want one per realm %v", got.Dirs, got.Realms)
	}
	for i, r := range got.Realms {
		if want := examplesRel + "/" + r; got.Dirs[i] != want {
			t.Errorf("Dirs[%d] = %q; want %q", i, got.Dirs[i], want)
		}
	}
	if want := []string{"lib.gno"}; !reflect.DeepEqual(got.ChangedFiles["gno.land/r/x/leaf"], want) {
		t.Errorf("ChangedFiles[leaf] = %v; want %v", got.ChangedFiles["gno.land/r/x/leaf"], want)
	}
}
```

</details>

## misc/gnopreview/plan_test.go:29-30 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L29-L30) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L29-L30) · Missing test
Missing test: every fixture goes through this helper, which writes the manifest and the source together, so [`plan.go:133`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L133), the check keeping a test-only or intermediate directory from registering as a previewable realm, can be deleted with the suite green.

<details><summary>test cases</summary>

```go
	// a directory with a manifest and only test sources, and one with a source
	// and no manifest: neither is a package
	write("examples/gno.land/r/x/testonly/gnomod.toml", "module = \"gno.land/r/x/testonly\"\n")
	write("examples/gno.land/r/x/testonly/lib_test.gno", "package testonly\n")
	write("examples/gno.land/r/x/nomod/lib.gno", "package nomod\n")
```

```go
		{
			name:       "a directory with only test sources is not a package",
			changed:    []string{"examples/gno.land/r/x/testonly/gnomod.toml"},
			wantRealms: []string{},
			wantMode:   "none",
		},
		{
			name:       "a directory without gnomod.toml is not a package",
			changed:    []string{"examples/gno.land/r/x/nomod/lib.gno"},
			wantRealms: []string{},
			wantMode:   "none",
		},
```

</details>

## misc/gnopreview/plan_test.go:33-40 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L33-L40) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L33-L40) · Missing test
Missing test: every fixture body parses, so the recovery at [`plan.go:179`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L179), where a source that fails to parse contributes no imports and the realms depending on it fall out of the plan, runs in no case.

<details><summary>test cases</summary>

```go
	// a package whose lib.gno stops mid-import: its own imports are lost, the
	// sibling file's survive
	write("examples/gno.land/p/x/kept/v0/gnomod.toml", "module = \"gno.land/p/x/kept/v0\"\n")
	write("examples/gno.land/p/x/kept/v0/lib.gno", "package kept\n")
	write("examples/gno.land/p/x/lost/v0/gnomod.toml", "module = \"gno.land/p/x/lost/v0\"\n")
	write("examples/gno.land/p/x/lost/v0/lib.gno", "package lost\n")
	write("examples/gno.land/p/x/broken/v0/gnomod.toml", "module = \"gno.land/p/x/broken/v0\"\n")
	write("examples/gno.land/p/x/broken/v0/lib.gno", "package broken\nimport (\n\t\"gno.land/p/x/lost/v0\"\n")
	write("examples/gno.land/p/x/broken/v0/more.gno", "package broken\nimport \"gno.land/p/x/kept/v0\"\n")
	pkg("examples/gno.land/r/x/usesbroken", "gno.land/r/x/usesbroken",
		"package usesbroken\nimport \"gno.land/p/x/broken/v0\"\n")
```

```go
		{
			name:       "a package that does not parse still resolves its dependents",
			changed:    []string{"examples/gno.land/p/x/kept/v0/lib.gno"},
			wantRealms: []string{"gno.land/r/x/usesbroken"},
			wantMode:   "realms",
		},
		{
			name:       "an import inside an unparseable file is dropped",
			changed:    []string{"examples/gno.land/p/x/lost/v0/lib.gno"},
			wantRealms: []string{},
			wantMode:   "none",
		},
```

</details>

## misc/gnopreview/plan_test.go:45-47 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L45-L47) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L45) · Missing test
Missing test: no case changes a file inside the `ignore = true` package this fixture writes, so `&& !p.Ignore` at [`plan.go:220`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) can be dropped with every case green, and the ignored realm then enters `ChangedRealms` and gets a comment link at [`comment.go:62-63`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L62-L63) to a page gnodev never renders.

<details>
<summary>test cases</summary>

The indirect route is already covered by [`plan.go:241`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L241), so the case has to change the ignored package itself.

```go
func TestIgnoredRealmChangedDirectly(t *testing.T) {
	t.Parallel()
	got, err := BuildPlan(fakeRepo(t), []string{"examples/gno.land/r/x/ignored/lib.gno"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ChangedRealms) != 0 {
		t.Errorf("ChangedRealms = %v; want none for an ignored realm", got.ChangedRealms)
	}
	if len(got.Realms) != 0 {
		t.Errorf("Realms = %v; want none for an ignored realm", got.Realms)
	}
}
```

</details>

## misc/gnopreview/plan_test.go:80 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L80) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L80) · Missing test
Missing test: neither realm here is a changed package and the only other case producing two realms truncates to one, so [`plan.go:260`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L260), the sort restoring alphabetical order after the changed-first pass at [`plan.go:253-255`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L253-L255), never has anything to restore, and `Realms`, `Dirs`, `ChangedFiles` and the index page are all built from that slice.

<details><summary>test cases</summary>

```go
		{
			name:       "an uncapped run reports realms alphabetically",
			changed:    []string{"examples/gno.land/p/x/base/v0/lib.gno", "examples/gno.land/r/x/other/lib.gno"},
			wantRealms: []string{"gno.land/r/x/leaf", "gno.land/r/x/other"},
			wantMode:   "realms",
		},
```

</details>

## misc/gnopreview/plan_test.go:107-113 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L107-L113) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L107) · Missing test
Missing test: [`fakeRepo`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L14-L51) writes none of the four [`gnowebSeedRealms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L37-L42), so `pkgs[r] != nil` at [`plan.go:265`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L265) is false in every case and the injection that gives a gnoweb-only pull request something to render is asserted by nothing.

<details>
<summary>test cases</summary>

One seed realm in the fixture, then the two gnoweb cases carry it. The suite is green with both edits applied.

```go
// fakeRepo, beside the other packages
	pkg("examples/gno.land/r/gnoland/home", "gno.land/r/gnoland/home", "package home\n")

		{
			name:       "gnoweb alone",
			changed:    []string{"gno.land/pkg/gnoweb/app.go"},
			wantGnoweb: true,
			wantRealms: []string{"gno.land/r/gnoland/home"},
			wantMode:   "gnoweb",
		},
		{
			name:       "gnoweb and a realm",
			changed:    []string{"gno.land/pkg/gnoweb/public/main.css", "examples/gno.land/r/x/leaf/lib.gno"},
			wantGnoweb: true,
			wantRealms: []string{"gno.land/r/gnoland/home", "gno.land/r/x/leaf"},
			wantMode:   "both",
		},
```

</details>

## SKIP misc/gnopreview/plan_test.go:120 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L120) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L120) · Missing test
Missing test: neither gnoweb case has a seed realm to find, so [`plan.go:263-270`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L263-L270) appends nothing in either, and a gnoweb-only pull request reaching gnodev with an empty realm list would leave the suite green.

Skipped: the same fixture edit as the section at `misc/gnopreview/plan_test.go:107-113`.

## misc/gnopreview/plan_test.go:122 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L122) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L122) · Missing test
Missing test: this case has one directly changed realm under a cap of one, so no case puts more changed realms than the cap, where the truncation at [`plan.go:256-258`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258) drops one that [`comment.go:62-63`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L62-L63) still links and [`comment.go:86`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L86) calls always kept.

<details>
<summary>test cases</summary>

```go
func TestCapKeepsEveryChangedRealm(t *testing.T) {
	t.Parallel()
	changed := []string{
		"examples/gno.land/r/x/leaf/lib.gno",
		"examples/gno.land/r/x/lonely/lib.gno",
		"examples/gno.land/r/x/other/lib.gno",
	}
	got, err := BuildPlan(fakeRepo(t), changed, 2)
	if err != nil {
		t.Fatal(err)
	}
	rendered := map[string]bool{}
	for _, r := range got.Realms {
		rendered[r] = true
	}
	for _, r := range got.ChangedRealms {
		if !rendered[r] {
			t.Errorf("%s is listed as a changed realm and was never rendered", r)
		}
	}
}
```

Added to the file as it stands, the case fails, which is the gap it covers:

```text
--- FAIL: TestCapKeepsEveryChangedRealm (0.00s)
    gno.land/r/x/other is listed as a changed realm and was never rendered
    ChangedRealms=[gno.land/r/x/leaf gno.land/r/x/lonely gno.land/r/x/other] Realms=[gno.land/r/x/leaf gno.land/r/x/lonely] Dropped=1
```

</details>

## misc/gnopreview/plan_test.go:132-136 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L132-L136) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L132-L136) · Missing test
Missing test: this rewrites a zero `maxRealms` into `defaultMaxRealms`, so the value [`main.go:73`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L73) documents as no cap never reaches `BuildPlan`, and widening [`plan.go:256`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256) to `maxRealms >= 0`, which turns every uncapped run into a preview of zero realms, keeps the suite green.

<details><summary>test cases</summary>

```go
		maxRealms  int // 0 takes the default; -1 passes 0 through, the documented no cap
```

```go
			limit := tc.maxRealms
			switch limit {
			case 0:
				limit = defaultMaxRealms
			case -1:
				limit = 0
			}
```

```go
		{
			name:       "no cap keeps every affected realm",
			changed:    []string{"examples/gno.land/p/x/base/v0/lib.gno"},
			maxRealms:  -1,
			wantRealms: []string{"gno.land/r/x/leaf", "gno.land/r/x/other"},
			wantMode:   "realms",
		},
```

</details>

## misc/gnopreview/plan_test.go:140-154 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L140-L154) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L140) · Missing test
Missing test: the subtest checks `Gnoweb`, `Realms`, `Mode`, `Dropped` and `Empty`, and never `Dirs` or `ChangedFiles` at [`plan.go:271-275`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L271-L275) or `ChangedPkgs` at [`plan.go:233`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L233), so the fields gnodev is started on and the crawler picks its pages from can come back empty with every case green.

<details>
<summary>test cases</summary>

```go
func TestPlanCarriesChangedFilesAndDirs(t *testing.T) {
	t.Parallel()
	root := fakeRepo(t)
	got, err := BuildPlan(root, []string{"examples/gno.land/r/x/leaf/lib.gno"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"examples/gno.land/r/x/leaf"}; !reflect.DeepEqual(got.Dirs, want) {
		t.Errorf("Dirs = %v; want %v", got.Dirs, want)
	}
	if want := map[string][]string{"gno.land/r/x/leaf": {"lib.gno"}}; !reflect.DeepEqual(got.ChangedFiles, want) {
		t.Errorf("ChangedFiles = %v; want %v", got.ChangedFiles, want)
	}
	got, err = BuildPlan(root, []string{"examples/gno.land/p/x/base/v0/lib.gno"}, defaultMaxRealms)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"gno.land/p/x/base/v0"}; !reflect.DeepEqual(got.ChangedPkgs, want) {
		t.Errorf("ChangedPkgs = %v; want %v", got.ChangedPkgs, want)
	}
}
```

</details>

## SKIP misc/gnopreview/plan_test.go:132-136 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L132-L136) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L132-L136) · Missing test
Missing test: [`plan_test.go:136`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L136) is the only `BuildPlan` call site in the package and the two lines above it make its limit argument always positive, so no case distinguishes `maxRealms > 0` at [`plan.go:256`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256) from `maxRealms >= 0`.

Skipped: the same absence as the posted `misc/gnopreview/plan_test.go:132-136` section, settled by the same fixture, so it posts once.

## misc/gnopreview/plan_test.go:194 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L194) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L194) · Missing test
Missing test: every `Comment` call in the suite passes a base URL, so the comment rendered when [`-base-url`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L71) keeps its empty default, which [`Makefile:15-16`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/Makefile#L15-L16) does, is asserted by nothing: no screenshots from [`comment.go:131`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L131), no tabs from [`comment.go:150`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L150), and a bare backticked path where each realm link would be.

<details><summary>test cases</summary>

```go
func TestCommentWithoutBaseURL(t *testing.T) {
	t.Parallel()
	p := &Plan{
		Gnoweb:        true,
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/x/leaf"},
		Shots:         []Shot{{File: "_shots/home.png", Label: "Home"}},
	}
	got := Comment(p, "", "7")
	for _, unwanted := range []string{"<table>", "_t/source/", "_t/help/", "<img"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("comment with no base URL carries %q\n---\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "gno.land/r/x/leaf") {
		t.Errorf("comment with no base URL must still name the changed realm\n---\n%s", got)
	}
}
```

</details>

## misc/gnopreview/plan_test.go:199 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L199) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L199) · Missing test
Missing test: this want list carries the source tab and not the help tab, so the second half of [`comment.go:154`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L154), the shortcut to the page listing a realm's exported functions, can be deleted with the suite green.

<details><summary>test cases</summary>

```go
		"https://example.test/pr-7/r/x/leaf/_t/help/",
```

</details>

## misc/gnopreview/plan_test.go:217-221 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L217-L221) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L217) · Missing test
Missing test: this is the only fixture with `Gnoweb` true and it has no changed realm, so [`isSeed`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L157-L162) never answers true at [`comment.go:69`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L69), and without that filter a comment covering gnoweb and a realm lists the fixed sample realms as affected by the change.

<details>
<summary>test cases</summary>

```go
func TestCommentBothModeDropsSeedRealms(t *testing.T) {
	t.Parallel()
	p := &Plan{
		Gnoweb:        true,
		ChangedRealms: []string{"gno.land/r/x/leaf"},
		Realms:        []string{"gno.land/r/gnoland/home", "gno.land/r/x/leaf"},
	}
	got := Comment(p, "https://example.test/pr-3", "3")
	if strings.Contains(got, "Realms affected through a changed package") {
		t.Errorf("a seed realm is listed as affected by the change:\n%s", got)
	}
}
```

With `isSeed` out of the filter, the case reports the section a reviewer would read as a consequence of the diff:

```text
**Realms affected through a changed package (1)**

- [`gno.land/r/gnoland/home`](https://example.test/pr-3/r/gnoland/home/) · [source](https://example.test/pr-3/r/gnoland/home/_t/source/) · [help](https://example.test/pr-3/r/gnoland/home/_t/help/)
```

</details>

## misc/gnopreview/plan_test.go:220 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L220) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L220) · Missing test
Missing test: this fixture holds one image, so the row break at [`comment.go:137`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L137) fires in no case, while a gnoweb change photographs the four views of [`shots.go:107-112`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L107-L112) and would render them as a single four-cell row.

<details><summary>test cases</summary>

```go
		Shots: []Shot{
			{File: "_shots/home.png", Label: "Home"},
			{File: "_shots/source.png", Label: "Source"},
			{File: "_shots/help.png", Label: "Help"},
			{File: "_shots/docs.png", Label: "Docs"},
		},
```

```go
	if n := strings.Count(got, "</tr><tr>"); n != 1 {
		t.Errorf("four screenshots want two rows; got %d row break(s)\n---\n%s", n, got)
	}
	if strings.Index(got, `alt="Source"`) > strings.Index(got, "</tr><tr>") {
		t.Errorf("the row break must fall after the second screenshot\n---\n%s", got)
	}
```

</details>

## SKIP misc/gnopreview/plan_test.go:54 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L54) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L54) · Missing test
Missing test: every path the table takes into the `ignore = true` package is indirect, through `p/x/base`, where [`plan.go:241`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L241) already drops it, so the guard on the directly changed file at [`plan.go:220`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) is held up by nothing.

Skipped: the same case as the section at `misc/gnopreview/plan_test.go:45-47`, which the author applies once.

## misc/gnopreview/shots.go:50 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L50) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L50) · Missing test
Missing test: no test file covers `ScreenshotPairs`, so nothing pins the [`maxPairs` break](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L68), the [image name derived from the realm path](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L76) that keys every `-after.png`, or [`Before` staying empty when the merge-base tree has no such realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L88-L96), and a regression in that name derivation overwrites one realm's shot with another's while the comment keeps showing both headings.

<details>
<summary>test cases</summary>

```go
// misc/gnopreview/shots_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeChrome writes a non-empty file wherever --screenshot= points, which is
// all chromeShot checks.
func fakeChrome(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakechrome")
	script := "#!/bin/sh\n" +
		"for a in \"$@\"; do case \"$a\" in --screenshot=*) out=\"${a#--screenshot=}\" ;; esac; done\n" +
		"printf 'PNG' > \"$out\"\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func crawlerOf(files map[string]string) *Crawler {
	c := &Crawler{pages: map[string]*page{}}
	for u, f := range files {
		c.pages[u] = &page{URL: u, File: f}
	}
	return c
}

func TestScreenshotPairsCap(t *testing.T) {
	head := crawlerOf(map[string]string{
		"/r/x/a": "r/x/a/index.html",
		"/r/x/b": "r/x/b/index.html",
		"/r/x/c": "r/x/c/index.html",
	})
	pairs := ScreenshotPairs(t.TempDir(), head, nil,
		[]string{"gno.land/r/x/a", "gno.land/r/x/b", "gno.land/r/x/c"},
		nil, fakeChrome(t))
	if len(pairs) != maxPairs {
		t.Fatalf("pairs = %d, want %d", len(pairs), maxPairs)
	}
}

func TestScreenshotPairsBeforeAbsentFromBase(t *testing.T) {
	head := crawlerOf(map[string]string{"/r/x/a": "r/x/a/index.html"})
	base := crawlerOf(map[string]string{"/r/x/other": "r/x/other/index.html"})
	pairs := ScreenshotPairs(t.TempDir(), head, base,
		[]string{"gno.land/r/x/a"}, nil, fakeChrome(t))
	if len(pairs) != 1 || pairs[0].Before != "" || pairs[0].New {
		t.Fatalf("pair = %+v, want Before empty and New false", pairs[0])
	}
}

func TestScreenshotPairsNamesStayDistinctPastMaxSlugLen(t *testing.T) {
	long := strings.Repeat("a", maxSlugLen)
	head := crawlerOf(map[string]string{
		"/r/x/" + long + "1": "r/x/one/index.html",
		"/r/x/" + long + "2": "r/x/two/index.html",
	})
	pairs := ScreenshotPairs(t.TempDir(), head, nil,
		[]string{"gno.land/r/x/" + long + "1", "gno.land/r/x/" + long + "2"},
		nil, fakeChrome(t))
	if len(pairs) != 2 || pairs[0].After == pairs[1].After {
		t.Fatalf("after shots collide: %+v", pairs)
	}
}
```

</details>

## .github/workflows/pr-preview-cleanup.yml:12-14 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-cleanup.yml#L12-L14) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview-cleanup.yml#L12-L14) · Nit
Nit: this trigger carries no `paths:` filter, so every closed pull request in the repository runs the job and checks out the previews repository with `PREVIEWS_DEPLOY_KEY` on disk, almost always to find no `pr-<N>` directory at [line 56](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-cleanup.yml#L56). Giving it the filter [`pr-preview.yml:17-26`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L17-L26) carries, or reading the previews repository over the API before the deploy-key checkout, keeps the credential out of the runs that have nothing to delete.

## SKIP .github/workflows/pr-preview-cleanup.yml:21-22 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-cleanup.yml#L21-L22) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview-cleanup.yml#L21-L22) · Nit
Nit: this key is built from `github.event.pull_request.head.repo.full_name`, which is null once the fork is deleted, and a deleted fork closes the pull request, so the group renders as `pr-preview-publish--<ref>` and stops matching [`pr-preview-publish.yml:32-33`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L32-L33), leaving an in-flight publish free to re-add `pr-<N>` after the cleanup removed it. Keying both workflows on the pull request number, which neither payload leaves null, serializes the pair in every case.

Skipped: the null `head.repo` on a deleted fork was not observed on a closed pull request here, so the mismatch stands on the payload shape alone.

## .github/workflows/pr-preview.yml:17-26 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L17-L26) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview.yml#L17-L26) · Nit
Nit: a push touching none of these paths starts no render and no publish, so the comment from the last matching push stands with its links serving pages built from an earlier head, under the closing line at [`comment.go:90`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L90) promising a rebuild on every push. Naming the head sha the snapshot was built from in the comment body makes a stale snapshot visible without reconstructing the push history.

## .github/workflows/pr-preview.yml:49-52 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L49-L52) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview.yml#L49-L52) · Nit
Nit: this checkout passes no `ref:`, so on a `pull_request` event it lands on `refs/pull/<N>/merge` and the after pane renders head merged into the current base tip, while the before pane is the merge-base worktree from [line 70](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L70). Passing `ref: ${{ github.event.pull_request.head.sha }}` keeps the pair differing by the branch's own realm edit alone, which is what [the header comment](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L66-L69) and [`main.go:222-228`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L222-L228) both promise, and makes the rendered tree the sha the comment is about.

## .github/workflows/pr-preview.yml:78 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L78) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview.yml#L78) · Nit
Nit: this pipeline's exit status is `tee`'s and the step sets no `pipefail`, so a failing `gnopreview` leaves `plan.json` empty and the step green, the `jq` gate below it reads nothing and leaves `skip` unset, and the job builds gnodev and renders before meeting the same failure. `set -o pipefail` in the step, or a `defaults.run.shell: bash` block for the workflow, makes the plan step the red one.

## .gitignore:13-14 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.gitignore#L13-L14) · [↗](../../../../../.worktrees/gno-review-6194/.gitignore#L13-L14) · Nit
Nit: these two entries are one tool's leftovers in the root ignore file, and `_preview/` is unanchored, so it hides a `_preview/` directory at any depth in the repository. A `misc/gnopreview/.gitignore` of `/gnopreview`, `/_preview/` and `/_preview.changed` covers all three leftovers, the last of them written by [`Makefile:14`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/Makefile#L14) and matched by nothing today, and follows the per-tool file six other `misc/` tools carry, [`misc/gnoe2e/.gitignore`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnoe2e/.gitignore#L1-L2) among them.

## SKIP .gitignore:14 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.gitignore#L14) · [↗](../../../../../.worktrees/gno-review-6194/.gitignore#L14) · Nit
Nit: `_preview/` matches the render output and not `_preview.changed`, the sibling leftover [`Makefile:14`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/Makefile#L14) writes beside it, so that file shows as untracked after every local run and a `git add -A` commits the developer's own changed-file list.

Skipped: the per-tool ignore file asked for on `.gitignore:13-14` already carries `/_preview.changed`, so one edit settles both.

## misc/gnopreview/Makefile:20 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/Makefile#L20) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/Makefile#L20) · Nit
Nit: The `$(GNODEV)` target has no prerequisites, so `make run` reuses whatever binary already sits at the fixed `/tmp/gnodev` path and renders the checked-out branch through a gnodev built from another one.

<details>
<summary>mechanism</summary>

Make treats a file target with no prerequisites as up to date whenever the file exists, so the `go build -o $(GNODEV)` recipe runs once per machine and never again. `GNODEV=/tmp/gnodev-probe make -n run` after `touch /tmp/gnodev-probe` prints the render and serve lines with no gnodev build in the recipe. The path is also shared, so on a multi-user box the first user's binary is what every later `make run` executes. Listing the gnodev sources as prerequisites, or building into `$(OUT)/gnodev`, rebuilds the binary when the tree it came from moves. CI is unaffected: the workflow builds gnodev itself on every run.

</details>

## misc/gnopreview/comment.go:12 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L12) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L12) · Nit
Nit: [`CommentMarker`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L10-L12) is not what makes the comment sticky, since the publishing job matches [its own copy of the literal](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L169) and nothing ties the two, so editing the const posts a fresh comment on every push with no test reddening.

## misc/gnopreview/comment.go:19 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L19) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L19) · Nit
Nit: the marker and the heading go into the builder before the [`p.Empty()` bail](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L24) throws both away, and [`render` returns on an empty plan](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L111-L113) before `Comment` is ever reached.

## misc/gnopreview/comment.go:44 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L44) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L44) · Nit
Nit: the gnoweb branch's bullet writes `link(urlOf(r), r)` without the [source and help shortcuts the other two bullets carry](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63), though the crawler [seeds and captures both pages](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L88-L92) in exactly that mode.

## misc/gnopreview/comment.go:48 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L48) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L48) · Nit
Nit: the sentence tells the reviewer that realm sources changed on any [`Mode()` of `"both"`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L98-L99), which a gnoweb change beside one `p/` package reaches with `ChangedRealms` empty and [no changed-realm section printed](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L60).

## misc/gnopreview/comment.go:52 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L52) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L52) · Nit
Nit: this lone `\n` is the only separator left once `base` is empty and the [homepage link supplying the second one](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L49-L51) is skipped, so markdown folds the intro and the next heading into one paragraph.

## misc/gnopreview/comment.go:55 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L55) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L55) · Nit
Nit: `pairGrid` renders whatever survived [the cap of two pairs](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L31) and the realms whose after shot failed, announcing neither, while the same comment does announce [its other cap](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L85-L86).

## misc/gnopreview/comment.go:63 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L63) · Nit
Nit: this bullet links every entry of `p.ChangedRealms` with no capture check, so a realm the crawl [dropped on a transport error or a non-200](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129-L137) reaches the reviewer as a link to a 404 that [`Index` would have skipped](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L170).

## misc/gnopreview/comment.go:75 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L75) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L75) · Nit
Nit: the heading names a changed package though `indirect` also collects realms pulled in by a changed realm, and the [`_changed:` line that would name the package](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L76-L78) is dropped in exactly that case.

## misc/gnopreview/comment.go:75 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L75) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L75) · Nit
Nit: this heading also counts a realm whose only changed import is another realm, since [the dependency walk seeds from every changed package](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L240) and [a changed realm is one of those](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L221), and it then names no package at all because [`ChangedPkgs`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L76-L78) is empty in exactly that case.

## SKIP misc/gnopreview/comment.go:63 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L63) · Warning
This line writes a link for every changed realm with no lookup against the captured pages, so a realm whose every page answered 500 is advertised in a comment posted on a green job.

SKIP: one edit, passing the `Crawler` into `Comment` and gating each link on it, closes this and the posted section on [`comment.go:17`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L17) together; the measurement stays in `claims.md`.

## misc/gnopreview/comment.go:90 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L90) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L90) · Nit
Nit: the snapshot is rebuilt only when a push touches [the paths the render workflow filters on](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L18-L27), so a later commit under `gnovm/` leaves a stale preview under this promise, and nothing in the body names the commit that was rendered.

## misc/gnopreview/comment.go:170 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L170) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L170) · Nit
Nit: this guard looks up the render page alone while the row below writes [three hrefs](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L173-L176), and the crawl [drops `$source` or `$help` on their own transport error or non-200](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129-L137).

## misc/gnopreview/comment.go:174 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L174) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L174) · Nit
Nit: the row is written only when [the render page was captured](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L170) while the source and help links on it are written unconditionally, and both tabs are [seeds of their own](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L91-L92) that the crawl [logs and skips on a fetch error or a non-200](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L130-L137), so either link can point at a page the snapshot does not hold.

## misc/gnopreview/comment.go:191 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L191) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L191) · Nit
Nit: this count reads `len(p.Realms)` while [the rows](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L168-L171) drop every realm missing from the captured pages, so the landing page reads "25 realm(s) rendered" over 23 links and names neither of the two that are gone.

## misc/gnopreview/comment.go:191 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L191) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L191) · Nit
Nit: this page says links outside the preview do not work, while an uncaptured in-site path is [rewritten to the live origin](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L405), [`https://gno.land` by default](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L28), which is what the [comment footer says](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L88) one screen away.

## misc/gnopreview/comment.go:195 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L195) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L195) · Nit
Nit: the header counts `len(p.Realms)` while the [rows are filtered by `c.pages`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L168-L172), so the landing page claims more realms than it lists and names none of the missing ones.

## misc/gnopreview/crawl.go:38 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L38) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L38) · Nit
Nit: `<head[^>]*>` matches `<header class="b-header">` too, so on a body served without a document head, which is [the case the fallback exists for](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L374-L376), the noindex tag is planted inside the header element and the snapshot stays indexable while [the test counting the tag](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L175) still passes.

## misc/gnopreview/crawl.go:95 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L95) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L95) · Suggestion
Suggestion: this walk terminates only because it stops before [`path.Dir("/")`](https://pkg.go.dev/path#Dir), which returns `/` forever, so loosening the bound to `>= 1` spins and [`TestSeedsSkipSingleSegmentDirs`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L293) reports the test binary's ten-minute panic in place of its own message. Bound the loop on `d != "/"` as well, so a loosened depth guard fails with the assertion.

<details>
<summary>observed</summary>

With `strings.Count(d, "/") >= 1` the run never returns, and the failure surfaces as a timeout on a job that names neither the seed set nor the guard.

</details>

## misc/gnopreview/crawl.go:138 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L138) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L138) · Suggestion
Suggestion: scope is tested at enqueue while the budget is charged here, after [the fetch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129), so every per-file source page past the budget is rendered and syntax-highlighted by gnodev and then thrown away, on every gnoweb change. Re-test the budget at dequeue, or charge at enqueue and release the slot when the fetch fails.

<details>
<summary>observed</summary>

A realm's `$source` overview links one page per file it lists, and [`inScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L227-L232) reads the budget without spending it, so all of them queue while the budget is still whole. [`TestFileBudgetChargedOnCaptureNotDiscovery`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L304) pins that ordering, which is why the wasted renders read as deliberate.

</details>

## misc/gnopreview/crawl.go:138-141 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L138-L141) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L138) · Nit
Nit: `chargeFile` runs after `getRetry` returned the body, and it is the only writer of the counter [`wantFile` gates on](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L231), so on a gnoweb pull request every per-file link of a `$source` page is rendered by gnodev and then thrown away, around 23 chroma-highlighted pages per crawl across the four seed realms at `FileBudget` 2. Charging the budget in `inScope` at enqueue time, and letting `chargeFile` refund a link that 404s, bounds the work rather than the captures.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6194 -R gnolang/gno
cat > misc/gnopreview/zz_budget_test.go <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOverBudgetFilePagesAreFetchedThenDropped(t *testing.T) {
	const files = 20
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RequestURI() == "/r/x/y$source" {
			var b strings.Builder
			for i := range files {
				fmt.Fprintf(&b, `<a href="/r/x/y$source&amp;file=f%d.gno">f%d</a>`, i, i)
			}
			fmt.Fprint(w, "<html><head></head><body>"+b.String()+"</body></html>")
			return
		}
		fmt.Fprint(w, `<html><head></head><body>ok</body></html>`)
	}))
	defer srv.Close()

	c := &Crawler{
		Base:       srv.URL,
		Realms:     []string{"gno.land/r/x/y"},
		Live:       "https://gno.land",
		FileBudget: GnowebFileBudget,
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if int(hits.Load()) > len(c.pages)+GnowebFileBudget {
		t.Fatalf("%d fetch(es) for %d captured page(s) at FileBudget %d",
			hits.Load(), len(c.pages), GnowebFileBudget)
	}
}
EOF
cd misc/gnopreview && go test -run TestOverBudgetFilePagesAreFetchedThenDropped -v .
cd ../.. && rm misc/gnopreview/zz_budget_test.go
```

The failure is the finding: gnodev renders every file page before the budget drops all but two.

```text
--- FAIL: TestOverBudgetFilePagesAreFetchedThenDropped
    zz_budget_test.go:40: 24 fetch(es) for 5 captured page(s) at FileBudget 2
FAIL
```

</details>

## misc/gnopreview/crawl.go:139 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L139) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L139) · Nit
Nit: only the budget drop prints to stdout, while [a transport error](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L131) and [a non-200](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L135) print to stderr, so the page losses read out of the error stream are missing exactly the ones looked for when a changed file is absent from the preview.

## misc/gnopreview/crawl.go:185 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L185) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L185) · Nit
Nit: the guard rejects a path carrying render arguments and a tab query together, so a subpath render such as `/r/x/y:p/about` is captured while its own source tab is not, and [`mapURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L405) sends that tab to the deployed realm's source rather than the branch's. Mapping an arguments-plus-query URL onto the already-captured argument-free tab keeps the link inside the snapshot.

## misc/gnopreview/crawl.go:191-193 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L191-L193) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L191) · Nit
Nit: this early return decides nothing, because [`urlOf`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L470-L476) never yields a path ending in `/` and [the realm match](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L195-L200) below refuses such a base anyway. Drop the three lines and move their comment onto that match, or keep them and say they state intent.

<details>
<summary>observed</summary>

Deleting the guard leaves the package green, [`crawl_test.go:86`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L86) included: that row is satisfied by the realm match. `Realms` is either `gnowebSeedRealms` or a `Pkg.Path` from [the package scan](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L228-L262), so no `urlOf(r)` can end in a slash.

</details>

## SKIP misc/gnopreview/crawl.go:243 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L243) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L243) · Nit
Nit: [`wantFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L231) tests the budget for an empty `file=` value while [`chargeFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L243) returns before charging one, so such a page is kept without spending a slot.

Skipped: no page gnoweb serves links `file=` with an empty value, so reaching the asymmetry needs a realm's own `Render` output to emit that href.

## misc/gnopreview/crawl.go:318 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L318) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L318) · Nit
Nit: the `_chroma/style.css` fetch drops both a transport error and a non-200 status, so a snapshot whose `$source` pages lost their syntax highlighting says nothing on any stream, unlike [the assets check that warns when no absolute `/public/` reference was found](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L314-L316). One stderr line naming the status is what tells a reviewer reading a rendering difference that the snapshot is short an asset.

## misc/gnopreview/crawl.go:361 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L361) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L361) · Nit
Nit: this replacement rebuilds `url(...)` bare while [the pattern matched an optional quote](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L41), so a rewritten path carrying a space or a paren becomes a token the browser rejects and the declaration is dropped with no error.

```suggestion
		return `url("` + up + strings.TrimPrefix(sub[1], "/") + `")`
```

## misc/gnopreview/crawl.go:382 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L382) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L382) · Suggestion
Suggestion: `strings.HasPrefix("", "/")` is false, so the second clause already returns the empty href untouched and the `raw == ""` arm decides nothing while suggesting the empty href needed one.

```suggestion
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
```

<details>
<summary>coverage</summary>

[`TestMapURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L122) carries the `{"", ""}` case, so the empty href stays pinned either way. The file is 611 lines before and after: what the edit buys is a guard of two conditions instead of three.

</details>

## misc/gnopreview/crawl.go:412-426 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L412-L426) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L412) · Suggestion
Suggestion: the three declarations exist only to carry the loop's last assignment past its scope, which named results do, and the same three expressions are returned twice, so the body is 15 lines for 11 with the same values and the same request count on every path.

```suggestion
func (c *Crawler) getRetry(p string) (body string, code int, err error) {
	for attempt := range 3 {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		if body, code, err = c.get(p); err != nil || code != http.StatusTooManyRequests {
			break
		}
	}
	return body, code, err
}
```

## misc/gnopreview/crawl.go:450-466 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L450-L466) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L450) · Suggestion
Suggestion: the `seen` map, the parallel slice and the trailing `sort.Strings` are what [`sortedKeys`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L317) returns, and the `p == ""` arm cannot fire behind the [`/` prefix filter](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L454), which leaves `base` at least `"/"` for any `$` or `:` split, so the body is 19 lines for 12.

```suggestion
	seen := map[string]bool{}
	for _, m := range attrRe.FindAllStringSubmatch(body, -1) {
		raw := html.UnescapeString(m[2])
		if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/public/") {
			continue
		}
		p, _, _ := strings.Cut(raw, "#")
		seen[canonicalURL(p)] = true
	}
	return sortedKeys(seen)
```

<details>
<summary>the one delta</summary>

`sortedKeys` allocates with `make([]string, 0, len(m))`, so a page with no in-site link yields an empty slice where the current body yields nil. The only consumer is the [range at crawl.go:147](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L147), where the two are indistinguishable, and [`TestLinks`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L117) compares against a two-element want.

</details>

## SKIP misc/gnopreview/crawl.go:490 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L490) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L490) · Nit
Nit: the `listing` flag is only ever false in the binary, so the `_dir` output path exists for URLs the crawl cannot reach, and a reader of its test row takes gnoweb's directory listings to be inside the snapshot.

Skipped: the same deletion is asked for at [crawl.go:496](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L496), which names the test rows it touches.

## misc/gnopreview/crawl.go:496 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L496) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L496) · Nit
Nit: no captured URL ends in a slash, since [`Seeds`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L85-L99) builds every entry from `urlOf` and `path.Dir` and [`inScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L191-L193) rejects a trailing-slash base, so the `_dir` branch never runs and [`TestURLToFile`'s `{"/r/", "r/_dir/index.html"}` row](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L61) pins a shape the binary cannot produce. Deleting the branch, the `listing` flag above it and that row leaves [`TestCrawlerInScope`'s `{"/r/gnoland/home/", false}`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L86) as the statement that listings are left to the live site.

## misc/gnopreview/crawl.go:599 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L599) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L599) · Nit
Nit: [`cmd.Wait()`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L202) sends nil down `died` when gnodev exits 0, a rejected flag or a usage print, and `%w` renders that nil as `%!w(<nil>)` where the reason belongs.

## misc/gnopreview/crawl_test.go:51-52 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L51-L52) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L51) · Nit
Test: no row here checks the characters this header names, and [`urlToFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L491-L495) slugs only the render arguments and the tab query, so `urlToFile("/r/x&y")` returns `r/x&y/index.html`. Assert `strings.ContainsAny(got, "$:&") == false` over the table rather than stating it in prose.

<details>
<summary>observed</summary>

Nothing reaches that shape today: [`inScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L195-L200) requires the base to equal a selected realm's path. What is missing is the assertion, not a live escape.

</details>

## misc/gnopreview/crawl_test.go:62 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L62) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L62) · Nit
Nit: `Seeds` stops above `/r` at [`crawl.go:95`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L95) and `inScope` refuses a trailing slash at [`crawl.go:191`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L191), so `/` is never queued at [`crawl.go:148`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L148) and the `_root` branch at [`crawl.go:492-494`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L492-L494) that this case pins writes no file. Removing both touches this case and those three lines of `urlToFile`.

<details>
<summary>what still covers the snapshot root</summary>

`mapURL` answers `/` from its own branch at [`crawl.go:399-401`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L399-L401), ahead of the `pages` lookup, so the root link in a rewritten page does not depend on `_root` either way. [`crawl_test.go:140`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L140) is the case that pins it.

</details>

## misc/gnopreview/crawl_test.go:181-192 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L181-L192) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L181) · Missing test
Missing test: these four assertions count the noindex tag and the robots metas, so nothing pins what [`attrRe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L40) does to `href` and `src`, and a snapshot page keeping the crawl origin or an absolute `/r/` path passes.

<details>
<summary>test cases</summary>

```go
func TestRewriteRelativizesLinks(t *testing.T) {
	t.Parallel()
	c := &Crawler{Base: "http://127.0.0.1:8899", Live: "https://gno.land", pages: map[string]*page{
		"/r/x/y": {File: "r/x/y/index.html"},
	}}
	got := c.rewrite(&page{File: "r/x/index.html", Body: `<a href="/r/x/y">y</a><img src="/public/logo.png">`})
	if strings.Contains(got, c.Base) || strings.Contains(got, `href="/r/x/y"`) {
		t.Errorf("an absolute link survived the rewrite: %s", got)
	}
	if !strings.Contains(got, `href="../../r/x/y/"`) || !strings.Contains(got, `src="../../public/logo.png"`) {
		t.Errorf("a rewritten link lost its target: %s", got)
	}
}
```

The body it asserts on:

```text
<meta name="robots" content="noindex, nofollow"><a href="../../r/x/y/">y</a><img src="../../public/logo.png">
```

</details>

## misc/gnopreview/crawl_test.go:124-128 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L124-L128) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L124) · Nit
Test: [`slug`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L520-L533) appends a digest to any query the sanitiser rewrites, so `urlToFile` maps the third URL here to `r/x/y/_t/file-a.gno-source-adc2ecdc/index.html` and the `File` beside it is a path no crawl writes, which is the form [`crawl_test.go:58`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L58) asserts for the same shape.

<details>
<summary>test cases</summary>

Deriving each `File` from `urlToFile` keeps the fixture on the layout the writer produces. It needs `path` in the import list.

```go
	c := &Crawler{Live: "https://gno.land", pages: map[string]*page{}}
	for _, u := range []string{"/r/x/y", "/r/x/y$source", "/r/x/y$file=a.gno&source"} {
		c.pages[u] = &page{File: urlToFile(u)}
	}
	const up = "../../../" // a page at r/x/y/index.html
	dir := func(u string) string { return up + path.Dir(urlToFile(u)) + "/" }
	for _, tc := range []struct{ in, want string }{
		{"/r/x/y", dir("/r/x/y")},
		{"/r/x/y$source", dir("/r/x/y$source")},
		{"/r/x/y$source&file=a.gno", dir("/r/x/y$file=a.gno&source")},
```

</details>

## SKIP misc/gnopreview/crawl_test.go:136 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L136) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L136) · Nit
Test: this expectation reads as a rewritten link resolving onto a directory the writer created, and `mapURL` only takes `path.Dir` of the fixture value at [`crawl.go:403`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L403), so the digest [`slug`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L520-L533) puts in the real path is invisible here.

Skipped: the same fixture edit as the section at `misc/gnopreview/crawl_test.go:124-128`.

## misc/gnopreview/crawl_test.go:196 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L196) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L196) · Nit
Test: this test drives [`inScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L177) and [`chargeFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L235) rather than the [`wantFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L227) it is named for, and [`chargeFile`'s changed-set exemption](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L250-L252) is asserted nowhere, so losing those three lines would drop the `$source` page of every file a realms-only pull request touched, where [`fileBudget`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L216-L221) returns 0.

<details>
<summary>test cases</summary>

Call `wantFile` for the listed and unlisted pair, and charge more changed files than any budget allows.

```go
func TestWantFileDirect(t *testing.T) {
	t.Parallel()
	c := &Crawler{
		Realms:       []string{"gno.land/r/x/touched", "gno.land/r/x/untouched"},
		ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno", "b.gno", "c.gno"}},
		FileBudget:   0, // fileBudget(plan) for a realms-only PR
	}
	for _, tc := range []struct {
		realm, name string
		want        bool
	}{
		{"gno.land/r/x/touched", "a.gno", true},      // listed: the changed set wins
		{"gno.land/r/x/touched", "other.gno", false}, // listed realm, unlisted file
		{"gno.land/r/x/untouched", "a.gno", false},   // unlisted realm, zero budget
	} {
		if got := c.wantFile(tc.realm, tc.name); got != tc.want {
			t.Errorf("wantFile(%q, %q) = %v; want %v", tc.realm, tc.name, got, tc.want)
		}
	}
}

func TestChangedFilesEscapeTheBudget(t *testing.T) {
	t.Parallel()
	c := &Crawler{
		Realms:       []string{"gno.land/r/x/touched"},
		ChangedFiles: map[string][]string{"gno.land/r/x/touched": {"a.gno", "b.gno", "c.gno"}},
		FileBudget:   0,
	}
	for _, f := range []string{"a.gno", "b.gno", "c.gno"} {
		u := "/r/x/touched$source&file=" + f
		if !c.chargeFile(u) {
			t.Errorf("chargeFile(%q) refused a changed file under a zero budget", u)
		}
	}
}
```

Both pass at head, 40 lines added and none removed. With the exemption deleted the shipped suite reports `ok` and `TestChangedFilesEscapeTheBudget` fails.

</details>

## misc/gnopreview/crawl_test.go:219 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L219) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L219) · Nit
Test: this loop and both `FileBudget` fixtures take their bound from `GnowebFileBudget` itself, so the value at [`crawl.go:218`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L218) can be raised to any number with no assertion going red, on the cap that decides how many per-file pages a wide preview carries.

<details>
<summary>test cases</summary>

```go
func TestGnowebFileBudget(t *testing.T) {
	t.Parallel()
	if GnowebFileBudget != 2 {
		t.Errorf("GnowebFileBudget = %d; want 2", GnowebFileBudget)
	}
}
```

</details>

## misc/gnopreview/crawl_test.go:282-291 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L282-L291) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L282) · Nit
Test: adding `strings.TrimSuffix(p, "/")` to [`splitURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L275) reddens [`TestSplitURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L17), [`TestCanonicalURL`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L32), [`TestURLToFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L49), [`TestCrawlerInScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L70) and [`TestURLToFileIsSafe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L264) on their own rows, so these ten lines catch nothing those five miss. Move the 62,174 against 48,642 measurement onto [`TestSplitURL`'s `/r/x/y/` row](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L17), which already carries the comment, and delete the function.

```suggestion
```

<details>
<summary>observed</summary>

Deleting the function leaves the package green, 10 lines to 0 with no coverage lost. The `_dir` and `_root` branches of [`urlToFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L490-L498) stay: they are what keeps `/r/x/y` and `/r/x/y/` on separate files, which is the collision [`TestURLToFileIsSafe`'s `seen` map](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L261-L271) catches.

</details>

## misc/gnopreview/main.go:60 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L60) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L60) · Nit
Nit: the first argument becomes the command name before [the flag set](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L63) is built, so `gnopreview --help` prints no usage and instead blocks reading stdin before reporting `unknown command "--help"`.

## SKIP misc/gnopreview/main.go:150 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L150) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L150) · Nit
Nit: `plan.Shots` is assigned in this arm alone, so [the `shotGrid` call in the gnoweb-and-realms branch of the comment](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L53) receives an empty slice on every input that reaches it and renders nothing.

SKIP: filling `Shots` whenever `plan.Gnoweb` holds settles this and the section on `main.go:147`, so it posts once.

## misc/gnopreview/main.go:96 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L96) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L96) · Nit
Nit: the switch deciding whether the command exists runs after [`readLines`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L87) and [`BuildPlan`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L91), so a mistyped subcommand blocks on stdin, which [`-changed` defaults to](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L66), and walks every package under `examples/` before the name is rejected.

## SKIP misc/gnopreview/main.go:91 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L91) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L91) · Nit
Nit: `BuildPlan` walks `examples/gno.land` and parses every non-test `.gno` file before the subcommand name is checked, so a typo pays for the whole walk and reports whatever that walk hits instead of the usage line.

SKIP: moving the switch above [`findRoot`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L81) settles this and the section on `main.go:102`, so it posts once.

## misc/gnopreview/main.go:105 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L105) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L105) · Nit
Missing test: nothing runs `render`, `startGnodev`, `renderBase`, `waitReady`, `findRoot` or `readLines`, since [`crawl_test.go`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go) and [`plan_test.go`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go) name no symbol from this file, and both render steps are gated on [`steps.plan.outputs.skip != 'true'`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L85), which a diff confined to `misc/gnopreview/` never clears.

<details>
<summary>test cases</summary>

The two helpers that need no node run today and pin the flag contract; the first reddens at this head, which is the `-root` finding on `misc/gnopreview/main.go:287`.

```go
func TestFindRootRejectsATreeWithoutExamples(t *testing.T) {
	if got, err := findRoot(t.TempDir()); err == nil {
		t.Fatalf("findRoot accepted %q, a directory with no examples/gnowork.toml", got)
	}
}

func TestReadLinesDropsBlankAndPaddedLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "changed.txt")
	if err := os.WriteFile(p, []byte("a.gno\n\n  b.gno  \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readLines(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "a.gno" || got[1] != "b.gno" {
		t.Fatalf("readLines = %q", got)
	}
}
```

```
=== RUN   TestFindRootRejectsATreeWithoutExamples
    zz_helpers_test.go:11: findRoot accepted "/tmp/TestFindRootRejectsATreeWithoutExamples4122605679/001", a directory with no examples/gnowork.toml
--- FAIL: TestFindRootRejectsATreeWithoutExamples (0.00s)
=== RUN   TestReadLinesDropsBlankAndPaddedLines
--- PASS: TestReadLinesDropsBlankAndPaddedLines (0.00s)
FAIL
```

The tree `render` writes needs a stub gnoweb on loopback and a stub `-gnodev` binary to assert without a node.

</details>

## misc/gnopreview/main.go:106 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L106) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L106) · Nit
Nit: the output directory is created and never cleared, so [the local flow the README documents](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L51-L56) serves the previous run's pages once the plan shrinks, and [`Index`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L166-L172) lists only what the current run captured.

## misc/gnopreview/main.go:190 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L190) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L190) · Nit
Nit: `Setpgid: true` puts gnodev in its own process group and the module installs no signal handler, so a Ctrl-C reaches gnopreview alone, [`stop()`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L205-L210) never runs, and gnodev survives holding both the web port and the RPC port.

## misc/gnopreview/main.go:176 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L176) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L176) · Nit
Nit: nothing range-checks [`-port`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L69) between the flag and the two ports derived from it.

- `-port 56000` asks gnodev to bind 66000 on the RPC listener, outside the range a port number can hold, and the run fails on a number nobody typed.
- `-port 0` makes gnodev bind a random port while [the crawler dials `http://127.0.0.1:0`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L124), so the run costs the whole `-timeout` and ends on `gnoweb not ready`, which names neither the flag nor the port it produced.

## misc/gnopreview/main.go:177 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L177) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L177) · Nit
Nit: `-home` is built from `cfg.out`, which stays relative, and gnodev's local command [chdirs into `-C`](https://github.com/gnolang/gno/blob/ecf7af0/contribs/gnodev/command_local.go#L86-L91) before reading it, so the per-port home never resolves under the output directory and [the workflow's `rm -rf _preview/.gnodev-*`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L103) matches nothing.

## misc/gnopreview/main.go:241 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L241) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L241) · Suggestion
Suggestion: a directory counts as a package only with [`gnomod.toml` and a non-test `.gno` file in it](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L133) while this check stats the directory alone, so a pull request adding the missing `gnomod.toml` to one of the 12 such directories under `examples/gno.land/r` loses its [`New in this PR` note](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L109) and hands the base pass a directory holding no package.

## SKIP misc/gnopreview/main.go:251 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L251) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L251) · Nit
The base pass derives its own port from the same unchecked flag, so `-port 0` and `-port 56000` fail here for the same reason the head pass does.

Skipped: one range check after the flags are parsed closes this and the finding on `misc/gnopreview/main.go:176` together.

## misc/gnopreview/main.go:287 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L287) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L287) · Nit
Nit: an explicit directory is returned as [`filepath.Abs(dir)`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L288) with no check for `examples/gnowork.toml`, so a wrong `-root` skips [the one message naming the flag](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L300) and surfaces later as a bare ENOENT from the package walk.

## misc/gnopreview/shots.go:68 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L68) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L68) · Nit
Nit: the two slots are spent in the alphabetical order [`ChangedRealms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L229-L231) was built in, so realms the pull request added, which carry [no before half](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L106-L109), can take both from a modified realm, even though [`newRealms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L148) already says which realms have a merge-base rendering.

## misc/gnopreview/main.go:310 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L310) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L310) · Suggestion
Suggestion: reading `/dev/stdin` reopens the descriptor through a device node, so a stdin that cannot be reopened, a socket or a container with an unpopulated `/dev`, aborts the run naming a path the caller never passed, where `io.ReadAll(os.Stdin)` reads what the process already holds.

## misc/gnopreview/main.go:334 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L334) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L334) · Nit
Nit: `envOr` restates [`cmp.Or`](https://pkg.go.dev/cmp#Or) for its [one call site](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L68), which reads at the same width as `cmp.Or(os.Getenv("GNODEV"), "gnodev")`.

## misc/gnopreview/plan.go:117 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L117) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L117) · Nit
Nit: `os.ReadDir(p)` re-lists the directory `filepath.WalkDir` read to reach this callback, so `examples/gno.land` is enumerated end to end twice on every plan build; keying the callback's own entries by `filepath.Dir(p)` as the walk delivers them drops the second pass and the `hasMod` bookkeeping with it.

## SKIP misc/gnopreview/plan.go:91 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L91) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L91) · Suggestion
Suggestion: `Empty` returns false for `Gnoweb` true with no realms, which is the one state the readiness probe in `render` cannot handle.

SKIP: one edit settles this and the section on `main.go:131`, so it posts once.

## misc/gnopreview/plan.go:140 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L140) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L140) · Suggestion
Suggestion: `strings.Contains` matches an `/r/` segment anywhere in the directory, so a package at `examples/gno.land/p/<x>/r/<y>` is previewed as a realm at a `/p/<x>/r/<y>` URL gnoweb does not serve, against [the field's own `lives under gno.land/r/`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L57).

## misc/gnopreview/plan.go:142 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L142) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L142) · Nit
Nit: `pkg.Draft` is parsed here for every package and read nowhere else in the program, so either it goes with its parse branch in [modFlags](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L160-L166), or a `draft = true` realm is being previewed against the intent and wants the filter [plan.go:220](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) applies to `Ignore`.

## SKIP misc/gnopreview/plan.go:156 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L156) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L156) · Nit
Nit: an unreadable `gnomod.toml` collapses into `false, false`, so an `ignore = true` package would be previewed; a fresh CI checkout produces no such file, which leaves the loss unreproduced.

Skipped: the same defect posts at plan.go:156-158, and this read could not settle the reachability it rests on.

## misc/gnopreview/plan.go:156-158 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L156-L158) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L156) · Nit
Nit: a `gnomod.toml` that cannot be read reports the package as neither draft nor ignored with nothing on stderr, so an `ignore = true` package in a damaged tree passes the filters at [plan.go:220](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) and [plan.go:241](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L241) and is booted into gnodev; returning the error lets [LoadPkgs](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L142) fail loudly instead of guessing.

## misc/gnopreview/plan.go:212-216 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L212-L216) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L212) · Nit
Refactor: [`hasPrefixAny`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L326-L333) is the predicate this loop spells out by hand, and [`BuildPlan` calls it directly](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L244) for `indirectExclude`: 5 lines to 3.

```suggestion
		if hasPrefixAny(f, gnowebPaths) {
			plan.Gnoweb = true
		}
```

## SKIP misc/gnopreview/plan.go:220 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L220) · Nit
Nit: a realm the pull request deletes leaves no trace in `ChangedRealms`, `Realms` or `Dropped`, because the package lookup runs against the head tree where that realm no longer exists.

Skipped: the same defect and the same edit as the posted `plan.go:220` section, which carries the measurement.

## SKIP misc/gnopreview/plan.go:263 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L263) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L263) · Nit
Nit: the seed append runs past both the cap and the `Ignore` filter, so a gnoweb PR renders up to four realms over the documented bound and an ignored seed among them.

Skipped: the two halves post at plan.go:263-268 and plan.go:265.

## SKIP misc/gnopreview/plan.go:257 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L257) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L257) · Nit
Nit: `Dropped` is computed before the seed realms are appended, so a seed the cap cut and the seed loop put back is counted as not rendered while it sits in the preview.

Skipped: it resolves in the same edit as the section at plan.go:263-268, which carries it.

## SKIP misc/gnopreview/plan.go:264 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L264) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L264) · Nit
Nit: with 25 changed realms sorting before the seeds, the cap keeps 25 and the seed loop appends all four, so `len(plan.Realms)` is 29 against a flag that names 25.

Skipped: it resolves in the same edit as the section at plan.go:263-268.

## SKIP misc/gnopreview/plan.go:266 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L266) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L266) · Nit
Nit: any diff under `gno.land/pkg/gnoweb/`, `gno.land/cmd/gnoweb/` or `contribs/gnodev/` sets the flag this append is gated on, so `-max-realms 1` renders up to five realms and reports `Dropped` as 0.

Skipped: it resolves in the same edit as the section at plan.go:263-268.

## misc/gnopreview/plan_test.go:40 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L40) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L40) · Nit
Test: `lonely` is written here and named by no case, and [`examples/gnowork.toml`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L26) is read by nothing the package tests, since `LoadPkgs` walks `examples/gno.land` and the one reader of that file is [`findRoot`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L286), which takes no test call. Drop both lines, or add the orphan-realm case the `lonely` fixture implies.

## misc/gnopreview/plan_test.go:84-86 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L84-L86) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L84-L86) · Nit
Test: changing `p/x/mid/v0` yields dependents `mid` and `leaf` alone, and the fixture realm imports `p/x/base/v0`, so it is never a candidate and [`plan.go:244-246`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L244-L246), the skip this case is named for, does not run. Repointing [the fixture's import](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L42-L43) at `gno.land/p/x/mid/v0` puts it in range and makes the name true.

## misc/gnopreview/plan_test.go:83-88 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L83-L88) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L83) · Nit
Test: the fixture realm at [`plan_test.go:42-43`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L42-L43) imports `gno.land/p/x/base/v0` while this case changes `p/x/mid/v0`, so the fixture is never among the dependents and the [`indirectExclude`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L48) skip at [`plan.go:244`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L244) is reached only by the case above it.

<details>
<summary>test cases</summary>

One word in the fixture puts the case on the rule it is named for. The suite is green with the edit, and the case goes red when `indirectExclude` is emptied.

```go
	pkg("examples/gno.land/r/tests/vm/fixture", "gno.land/r/tests/vm/fixture",
		"package fixture\nimport \"gno.land/p/x/mid/v0\"\n")
```

With the fixture as it stands, emptying `indirectExclude` leaves that case passing:

```text
--- FAIL: TestBuildPlan/transitive_dependency_pulls_both_dependents (0.00s)
--- PASS: TestBuildPlan/tests_fixtures_are_not_pulled_in_indirectly (0.00s)
```

</details>

## misc/gnopreview/plan_test.go:95-100 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L95-L100) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L95) · Nit
Test: [`LoadPkgs`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L107-L110) walks `examples/gno.land` alone, so a quarantined package is absent from the directory map whatever the guard does and this case exits at the same `continue` as `nothing relevant`, leaving the prefix check at [`plan.go:217`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L217) held up by nothing.

<details>
<summary>what would make it a case</summary>

Either drop it, or have `LoadPkgs` walk `examples` whole and pin there that a package under `examples/quarantined` is not loaded. Deleting the `strings.HasPrefix(f, pkgRoot+"/")` half of the guard leaves `go test -run TestBuildPlan .` green as the file stands.

</details>

## misc/gnopreview/plan_test.go:143-144 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L143-L144) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L143-L144) · Nit
Test: `reflect.DeepEqual` here separates a nil `Realms` from an empty one, which [`plan.go:72`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L72) marshals into the plan JSON as `null` against `[]`, and `%v` prints both sides as `[]`.

```suggestion
			if !reflect.DeepEqual(got.Realms, tc.wantRealms) {
				t.Errorf("Realms = %#v; want %#v", got.Realms, tc.wantRealms)
```

<details><summary>what the two verbs print</summary>

With `sortedKeys` returning nil for an empty map, the four cases wanting `[]string{}` fail. As shipped:

```
    plan_test.go:144: Realms = []; want []
```

With `%#v`:

```
    plan_test.go:144: Realms = []string(nil); want []string{}
```

</details>

## misc/gnopreview/plan_test.go:178 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L178) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L178) · Nit
Test: the five tests from this line to the end of the file exercise `Comment`, `pairGrid`, `shotGrid`, `tabs` and `isSeed`, every one of them defined in [`comment.go`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L17), which has no test file beside it while `crawl.go` has one.

<details>
<summary>what it costs</summary>

101 of the file's 278 lines cover a different source file, so a reader looking for the comment renderer's coverage opens `comment_test.go`, finds nothing, and does not audit what is there.

</details>

## misc/gnopreview/shots.go:73 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L73) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L73) · Nit
Nit: the `!ok` guard drops a realm [the crawl skipped](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129-L141) out of the before/after grid with nothing on stderr, where [the screenshot failure below](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L80) names its realm.

## misc/gnopreview/shots.go:79 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L79) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L79) · Nit
Nit: the destination path handed to `chromeShot` keeps whatever [a failed run left at it](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L184-L189), and `_shots/` is uploaded whole as the published snapshot.

## misc/gnopreview/shots.go:94 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L94) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L94) · Nit
Nit: a failed before capture leaves `Before` empty and `New` false, and [`pairGrid` attaches its caption only when `New` is set](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L106-L113), so the comment shows the realm as one uncaptioned image with nothing saying the merge-base render was not obtained.

## misc/gnopreview/shots.go:99 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L99) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L99) · Nit
Nit: the run log prints `(before/after)` for every appended pair, [including the ones whose before half was skipped or failed](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L88-L96), so it reports a comparison the comment does not carry.

## misc/gnopreview/shots.go:122 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L122) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L122) · Nit
Nit: `base` is the loopback origin `serve` returns here and the merge-base [`*Crawler`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L50) seventy lines up, which [the sibling pass spells `origin`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L56).

## SKIP misc/gnopreview/shots.go:153 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L153) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L153) · Nit
Nit: `serve` builds an `*http.Server` that escapes nowhere, so `http.Server.Close`, the call that closes tracked connections, never runs and Chrome's keep-alive connection lives until the process exits.

Skipped: the same edit as the section on line 163, which carries this finding.

## SKIP misc/gnopreview/shots.go:163 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L163) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L163) · Nit
Nit: `go srv.Serve(ln)` discards the error and the returned `io.Closer` is the listener, so a `Serve` failing after a successful bind is silent and every screenshot reports Chrome's own connection error instead.

Skipped: the same edit as the posted section on line 163, which carries this finding.

## SKIP misc/gnopreview/shots.go:174 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L174) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L174) · Suggestion
Suggestion: the neighbouring `--virtual-time-budget` carries the reason it is there and `--no-sandbox` carries none, while the pages Chrome loads are gnoweb's render of realm source taken from the pull request.

Skipped: the same edit as the posted section on line 174, which carries this finding.

## SKIP misc/gnopreview/shots.go:177 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L177) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L177) · Suggestion
Suggestion: `--window-size=1280,860` with `--screenshot` and no full-page option frames whatever that window shows, so a change below the fold can leave both images identical while [the captions](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L117-L121) present them as the comparison.

Skipped: whether headless Chrome clips at the window height was not settled here, and the after image still links to the full page.

## SKIP misc/gnopreview/shots.go:208 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L208) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L208) · Nit
Nit: the stat arm checks only `IsDir`, so a half-downloaded browser or a `-chrome` flag pointing at the wrong file passes and `findChrome` answers non-empty; `fi.Mode()&0o111 != 0` is the whole fix.

Skipped: the same edit as the posted section on line 208, which carries this finding.

## misc/gnopreview/shots.go:187 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L187) · Nit
Nit: the `os.Stat` error is bound and never read, so an unwritable path or a missing `_shots` directory is reported as `chrome wrote no image` and the real errno never reaches the log.

## misc/gnopreview/shots.go:196 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L196) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L196) · Nit
Nit: an explicit `-chrome` value heads the same candidate list as the auto-detected names, and [the loop](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L201-L211) returns the first entry that resolves, so a path that does not resolve is dropped without a word and the screenshots come from whatever browser is on `PATH`.

## misc/gnopreview/shots.go:208 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L208) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L208) · Nit
Nit: the `os.Stat` arm accepts any non-directory path while [`exec.LookPath` above it](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L205) already resolves a candidate carrying a path separator whenever it is runnable, so the only path this arm adds is one that cannot be executed, and the preview then ships imageless with one exec failure per page in place of [the single no-browser line](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L53).

## misc/gnopreview/shots.go:196-200 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L196-L200) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L196) · Nit
Nit: `-chrome` and `$CHROME` sit in the same candidate list as the five fallbacks, and [the loop walks past any candidate resolving to neither a PATH entry nor a file](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L201-L211), so a typo or a path absent from the runner image photographs with whatever other browser is installed and prints nothing.

## SKIP .github/workflows/pr-preview-publish.yml:119-126 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L119-L126) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview-publish.yml#L119) · Suggestion
Suggestion: `MAX_PREVIEW_MIB` at 40 lets 26 previews fill a 1 GB site, against the description's figure of about 78 live previews, and at the 8.8 MiB worst case the comment quotes those 78 come to 686 MiB, so neither bound is what the per-preview cap enforces: the scheduled eviction in the previews repository at 700 MiB oldest-first is what holds the quota.

Skipped: the claim is about the comment's own wording and changes no behaviour, so it stays in `claims.md`.

## .github/workflows/pr-preview.yml:64 [gh](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L64) · [↗](../../../../../.worktrees/gno-review-6194/.github/workflows/pr-preview.yml#L64) · Suggestion
Suggestion: `git diff --name-only` C-quotes any path outside printable ASCII, and [`readLines`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L306-L324) only trims whitespace, so the leading quote fails both prefix tests at [`plan.go:212-217`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L212-L217) and that path's realm drops out of the plan. No file under `examples/` carries such a byte today.

```suggestion
          git -c core.quotePath=false diff --name-only "$base" "$HEAD_SHA" > changed.txt
```

## SKIP misc/gnopreview/comment.go:17 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L17) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L17) · Suggestion
`Comment` builds its links from the plan while [`Index`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L166-L171) builds its rows from the crawler, so the two disagree about which realms the snapshot holds, and [a draft realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L142) is filtered by neither.

SKIP: the same missing `Crawler` parameter as the posted Warning on this line.

## misc/gnopreview/comment.go:18-26 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L18-L26) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L18) · Suggestion
Suggestion: the marker and the heading are written into `b` above the guard that throws the builder away, and the marker is what [the publishing job matches to update the sticky comment](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L169-L172), so a later edit returning `b.String()` from the guard would post a marker-only body.

```suggestion
	// Nothing to preview means no comment: Comment is not called in that case,
	// and an empty string here keeps a stray call from posting a useless one.
	if p.Empty() {
		return ""
	}

	var b strings.Builder
	b.WriteString(CommentMarker + "\n")
	b.WriteString("### 🖼️ gnoweb preview\n\n")
```

## misc/gnopreview/comment.go:29-34 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L29-L34) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L29) · Suggestion
Suggestion: `link` takes a URL path and a label that all three call sites derive from the same realm, [44](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L44), [63](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63) and [80](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L80), so the signature admits a mismatched pair no caller writes; `link(pkgPath string)` calling `urlOf` inside is the same six lines, renders the same bytes, and reads `link(r)` at each site.

## misc/gnopreview/comment.go:36-46 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L36-L46) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L36) · Suggestion
Suggestion: [`Mode()`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L94-L104) returns four strings and [the `Empty()` guard](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L24-L26) removes `none`, so this default is reached by `both` and `realms` alone and its first statement re-reads `p.Gnoweb` to tell them apart, which is one fact read through two sources; the two-way branch says it in a line, 196 lines to 195.

```suggestion
	if p.Mode() == "gnoweb" {
		b.WriteString("This PR changes **gnoweb itself**, so the preview is a sample of pages rendered with it:\n\n")
		if base != "" {
			b.WriteString(fmt.Sprintf("**[Open the preview homepage](%s/)**\n\n", base))
		}
		b.WriteString(shotGrid(p.Shots, base))
		for _, r := range p.Realms {
			b.WriteString("- " + link(urlOf(r), r) + "\n")
		}
	} else {
```

## misc/gnopreview/comment.go:53 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L53) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L53) · Suggestion
Suggestion: `shotGrid(p.Shots, base)` returns the empty string whenever a realm changed, since [`Shots` is assigned only in the `else if` arm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L147-L151), so the sentence above it promises a gnoweb sample the comment cannot carry. Take the sample as well when `plan.Gnoweb` holds, or list the [seed realms](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L157-L162) under the blurb.

## misc/gnopreview/comment.go:56-69 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L56-L69) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L56) · Suggestion
Suggestion: the `direct` map is built to answer one membership test one loop later, and [`contains`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L335-L337) says the same thing in the line that asks, 196 lines to 192.

```suggestion
		if len(p.ChangedRealms) > 0 {
			b.WriteString(fmt.Sprintf("**Changed realms (%d)**\n\n", len(p.ChangedRealms)))
			for _, r := range p.ChangedRealms {
				b.WriteString("- " + link(urlOf(r), r) + tabs(base, r) + "\n")
			}
			b.WriteString("\n")
		}
		var indirect []string
		for _, r := range p.Realms {
			if !contains(p.ChangedRealms, r) && !isSeed(r, p) {
```

## misc/gnopreview/comment.go:74 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L74) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L74) · Suggestion
Suggestion: `sort.Strings(indirect)` cannot reorder its input, a forward filter over the [already-sorted `p.Realms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L260). Dropping it drops `comment.go`'s only use of the `sort` import, 196 lines to 194.

<details>
<summary>patch</summary>

```diff
--- a/misc/gnopreview/comment.go
+++ b/misc/gnopreview/comment.go
@@ -3,7 +3,6 @@ package main
 import (
 	"fmt"
 	"path"
-	"sort"
 	"strings"
 )
 
@@ -71,7 +70,6 @@ func Comment(p *Plan, baseURL, pr string) string {
 			}
 		}
 		if len(indirect) > 0 {
-			sort.Strings(indirect)
 			b.WriteString(fmt.Sprintf("**Realms affected through a changed package (%d)**\n\n", len(indirect)))
 			if len(p.ChangedPkgs) > 0 {
 				b.WriteString("_changed: " + "`" + strings.Join(p.ChangedPkgs, "`, `") + "`_\n\n")
```

Applied at this head: `gofmt -l .` silent, `go vet ./...` silent, `go test ./...` ok, output identical.

</details>

## misc/gnopreview/comment.go:117 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L117) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L117) · Suggestion
Suggestion: the before cell emits a bare `<img>` while [the after cell wraps its image in the page link](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L120-L122), so the `_before/` tree the [README offers for clicking through](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L73-L74) is reachable from nothing the tool writes. Carry the before page on [`ShotPair`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L34-L44) and wrap this cell in it.

## misc/gnopreview/comment.go:121 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L121) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L121) · Suggestion
Suggestion: every `<img src>` this cell writes points into the snapshot the [cleanup job removes on close](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-cleanup.yml#L60), so a closed pull request keeps a permanent comment of broken image boxes. Patching the sticky comment down to a one-line note needs `issues: write` on that job, which carries [`permissions: {}`](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-cleanup.yml#L33) and makes no API call.

## misc/gnopreview/comment.go:149 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L149) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L149) · Suggestion
Suggestion: `tabs` recomputes the `urlOf(pkgPath)` [each bullet already passed to `link`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63), and the bullet itself is spelled out at three call sites, one of them [without the shortcuts](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L44). One `realmItem(base, pkgPath)` carries all three, 196 lines to 191.

<details>
<summary>patch</summary>

Not output-neutral: the gnoweb branch gains the source and help shortcuts, whose `_t/source/` and `_t/help/` pages the crawler already writes.

```diff
--- a/misc/gnopreview/comment.go
+++ b/misc/gnopreview/comment.go
@@ -26,12 +26,6 @@ func Comment(p *Plan, baseURL, pr string) string {
 	}
 
 	base := strings.TrimSuffix(baseURL, "/")
-	link := func(urlPath, label string) string {
-		if base == "" {
-			return "`" + label + "`"
-		}
-		return fmt.Sprintf("[`%s`](%s%s/)", label, base, urlPath)
-	}
 
 	switch p.Mode() {
 	case "gnoweb":
@@ -41,7 +35,7 @@ func Comment(p *Plan, baseURL, pr string) string {
 		}
 		b.WriteString(shotGrid(p.Shots, base))
 		for _, r := range p.Realms {
-			b.WriteString("- " + link(urlOf(r), r) + "\n")
+			b.WriteString(realmItem(base, r))
 		}
 	default:
 		if p.Gnoweb {
@@ -60,7 +54,7 @@ func Comment(p *Plan, baseURL, pr string) string {
 		if len(p.ChangedRealms) > 0 {
 			b.WriteString(fmt.Sprintf("**Changed realms (%d)**\n\n", len(p.ChangedRealms)))
 			for _, r := range p.ChangedRealms {
-				b.WriteString("- " + link(urlOf(r), r) + tabs(base, r) + "\n")
+				b.WriteString(realmItem(base, r))
 			}
 			b.WriteString("\n")
 		}
@@ -77,7 +71,7 @@ func Comment(p *Plan, baseURL, pr string) string {
 				b.WriteString("_changed: " + "`" + strings.Join(p.ChangedPkgs, "`, `") + "`_\n\n")
 			}
 			for _, r := range indirect {
-				b.WriteString("- " + link(urlOf(r), r) + tabs(base, r) + "\n")
+				b.WriteString(realmItem(base, r))
 			}
 		}
 	}
@@ -145,13 +139,14 @@ func shotGrid(shots []Shot, base string) string {
 	return b.String()
 }
 
-// tabs adds the source/help shortcuts next to a realm link.
-func tabs(base, pkgPath string) string {
+// realmItem renders one realm bullet: the render link and its source and help
+// shortcuts, or the bare package path when there is nowhere to link to.
+func realmItem(base, pkgPath string) string {
 	if base == "" {
-		return ""
+		return "- `" + pkgPath + "`\n"
 	}
 	u := urlOf(pkgPath)
-	return fmt.Sprintf(" · [source](%s%s/_t/source/) · [help](%s%s/_t/help/)", base, u, base, u)
+	return fmt.Sprintf("- [`%s`](%s%s/) · [source](%s%s/_t/source/) · [help](%s%s/_t/help/)\n", pkgPath, base, u, base, u, base, u)
 }
```

Applied at this head: `gofmt -l .` silent, `go vet ./...` silent, `go test ./...` ok.

</details>

## misc/gnopreview/comment.go:154-155 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L154-L155) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L154) · Suggestion
Suggestion: the `_t/source/` and `_t/help/` suffixes are concatenated here and again in [`Index`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L174), four hand-spelled copies of the layout [`urlToFile` decides](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L503), so changing the scheme there leaves every source and help link pointing at a directory the crawler no longer writes.

```suggestion
	return fmt.Sprintf(" · [source](%s/%s) · [help](%s/%s)",
		base, tabDir(u, "source"), base, tabDir(u, "help"))
}

// tabDir is the snapshot directory holding one tab of a realm's page, asked of
// urlToFile so the comment's links cannot drift from what the crawler wrote.
func tabDir(u, query string) string {
	return path.Dir(urlToFile(u+"$"+query)) + "/"
}
```

<details>
<summary>the `Index` half</summary>

```diff
@@ -171,9 +178,8 @@ func Index(p *Plan, c *Crawler) string {
 			continue
 		}
 		rows.WriteString(fmt.Sprintf(
-			`    <li><a href="%s/"><code>gno.land%s</code></a> <a href="%s/_t/source/">source</a> <a href="%s/_t/help/">help</a></li>`+"\n",
-			strings.TrimPrefix(path.Dir(urlToFile(u)), "/"), u,
-			strings.TrimPrefix(path.Dir(urlToFile(u)), "/"), strings.TrimPrefix(path.Dir(urlToFile(u)), "/")))
+			`    <li><a href="%s/"><code>gno.land%s</code></a> <a href="%s">source</a> <a href="%s">help</a></li>`+"\n",
+			path.Dir(urlToFile(u)), u, tabDir(u, "source"), tabDir(u, "help")))
 	}
```

Both hunks applied at this head: emitted markup and HTML byte-identical, six lines added.

</details>

## misc/gnopreview/comment.go:157-162 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L157-L162) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L157) · Suggestion
Suggestion: `isSeed` takes its receiver last and spends six lines on one boolean and, while the plan's other predicates are methods, [`Empty`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L91) and [`Mode`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L94); as `func (p *Plan) isSeed(pkgPath string) bool` returning `p.Gnoweb && contains(gnowebSeedRealms, pkgPath)` it is three lines, with [the one call site](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L69) reading `p.isSeed(r)`.

## misc/gnopreview/comment.go:173-176 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L173-L176) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/comment.go#L173) · Suggestion
Suggestion: the row holds three byte-identical copies of the directory expression, so an edit to the slug rule that misses one points a row's source link at a different directory than its page link.

```suggestion
		dir := strings.TrimPrefix(path.Dir(urlToFile(u)), "/")
		rows.WriteString(fmt.Sprintf(
			`    <li><a href="%s/"><code>gno.land%s</code></a> <a href="%s/_t/source/">source</a> <a href="%s/_t/help/">help</a></li>`+"\n",
			dir, u, dir, dir))
```

## misc/gnopreview/crawl.go:70 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L70) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L70) · Suggestion
Suggestion: `order` is a parallel key list for `pages` with one reader, [`writePages`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L327), which can range over `slices.Sorted(maps.Keys(c.pages))` instead and take the field, [its append](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L144) and the reset [`Run`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L110) forgets with it, 3 lines out for 2 in.

<details>
<summary>patch</summary>

```diff
@@ misc/gnopreview/crawl.go
-	"io"
+	"io"
+	"maps"
@@
 	pages map[string]*page // url -> page
-	order []string
 }
@@ func (c *Crawler) Run() error
 		c.pages[u] = p
-		c.order = append(c.order, u)
 		fmt.Printf("  ✓ %s\n", u)
@@ func (c *Crawler) writePages(dir string) error
-	for _, u := range c.order {
+	for _, u := range slices.Sorted(maps.Keys(c.pages)) {
 		p := c.pages[u]
```

The written tree is byte-identical, so capture order is not observable in the output.

</details>

## SKIP misc/gnopreview/crawl.go:110 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L110) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L110) · Suggestion
Suggestion: `Run` resets `pages` and leaves `order` and `fileBudget` holding the previous call's entries, so a second crawl on one `Crawler` reaches [`c.pages[u]`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L328) for a URL it did not re-capture and dereferences nil. Nothing reuses a `Crawler` today, since [`renderBase`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L259) builds its own.

Skipped: deleting `order`, asked for at [crawl.go:70](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L70), removes the half-reset this rests on.

## misc/gnopreview/crawl.go:99-104 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L99-L104) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L99) · Suggestion
Suggestion: `dirs` is filled at [one place](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L96), inside the loop body the [`RenderOnly` continue](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L88-L90) skips, so the guard here protects an empty map, and the `seen` closure repeats the dedupe `Run` already does through [`visited`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L118-L123).

```suggestion
	return append(out, sortedKeys(dirs)...)
```

<details>
<summary>the rest of the rewrite</summary>

Dropping the closure with it takes `Seeds` from 31 lines to 18:

```diff
-	seen := map[string]bool{}
 	var out []string
-	add := func(u string) {
-		if !seen[u] {
-			seen[u] = true
-			out = append(out, u)
-		}
-	}
 	dirs := map[string]bool{}
 	for _, r := range c.Realms {
 		u := urlOf(r)
-		add(u)
+		out = append(out, u)
 		if c.RenderOnly {
 			continue
 		}
-		add(u + "$source")
-		add(u + "$help")
+		out = append(out, u+"$source", u+"$help")
```

One delta: `Seeds` returns a repeated entry when one selected realm is the directory of another, `gno.land/r/x` beside `gno.land/r/x/y`. `Run` absorbs it through `visited`, and it widens the `queue still had %d` count in the page-cap error.

</details>

## misc/gnopreview/crawl.go:112 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L112) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L112) · Suggestion
Suggestion: every seed sits in `queue` before the loop starts and is [marked visited on dequeue](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L125), so `seedSet[link]` in [the link filter](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L148) admits nothing the queue does not already hold, and dropping the map, its build and that term takes `Run` from 46 lines to 41.

<details>
<summary>patch</summary>

```diff
@@ func (c *Crawler) Run() error
 	c.pages = map[string]*page{}
-	seeds := c.Seeds()
-	seedSet := map[string]bool{}
-	for _, s := range seeds {
-		seedSet[s] = true
-	}
-
-	queue := append([]string(nil), seeds...)
+	queue := c.Seeds()
 	visited := map[string]bool{}
@@
 		for _, link := range links(body) {
-			if !visited[link] && (seedSet[link] || c.inScope(link)) {
+			if !visited[link] && c.inScope(link) {
```

</details>

## misc/gnopreview/crawl.go:148 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L148) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L148) · Suggestion
Suggestion: the append tests `visited`, which is [written only at dequeue](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L123), so the header, breadcrumb and footer links every gnoweb page repeats are queued once per referencing page and the queue grows as pages times links per page; a `queued` set written at append time bounds it by the distinct link set instead.

## misc/gnopreview/crawl.go:186 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L186) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L186) · Suggestion
Suggestion: render-argument pages carry no per-realm budget, only [`file=` queries do](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L209), so a realm whose render links one page per item enqueues them all under nothing but the [400-page cap](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L118). Charging the per-realm budget on the argument branch too, and truncating at the cap rather than [returning an error](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L127), keeps one wide realm from costing the whole preview.

## SKIP misc/gnopreview/crawl.go:209 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L209) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L209) · Suggestion
Suggestion: the budget call sits inside the query loop, which an argument-only URL never enters, so [`{"/r/gnoland/home:p/x", true}`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L87) holds for every argument a realm links and the argument tree is followed uncapped.

Skipped: the same edit is asked for at [crawl.go:186](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L186), the one gate on render arguments.

## misc/gnopreview/crawl.go:231 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L231) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L231) · Suggestion
Suggestion: the counter this reads is advanced by [`chargeFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L138) after [the fetch](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L129), while the test itself runs [at link-enqueue time](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L148), so a `$source` overview queues every file it links in one burst and gnodev renders each one whole, highlighting included, before the budget drops it. Re-testing the budget at dequeue, ahead of `getRetry`, is what turns the bound on what is written into a bound on what is rendered.

## misc/gnopreview/crawl.go:304-306 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L304-L306) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L304) · Suggestion
Suggestion: the empty `assets` sentinel routes one caller, [`renderBase`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L275), to the method three lines below it, so exporting `writePages` and calling it there drops the branch and leaves `Write`'s signature describing only what `Write` does.

<details>
<summary>the patch</summary>

```diff
--- a/misc/gnopreview/crawl.go
+++ b/misc/gnopreview/crawl.go
@@ -301,9 +301,6 @@ func canonicalURL(p string) string {
 // rewritten: to a relative path when we captured the target, to the live site
 // otherwise. assets is the repo's gnoweb public/ dir, copied verbatim.
 func (c *Crawler) Write(dir, assets string) error {
-	if assets == "" { // a prefixed crawl reuses the assets already written
-		return c.writePages(dir)
-	}
 	n, err := copyTree(assets, filepath.Join(dir, "public"))
@@ -320,10 +317,12 @@ func (c *Crawler) Write(dir, assets string) error {
-	return c.writePages(dir)
+	return c.WritePages(dir)
 }
 
-func (c *Crawler) writePages(dir string) error {
+// WritePages renders the captured pages into dir. A prefixed crawl calls it
+// directly: the assets are already on disk from the head pass.
+func (c *Crawler) WritePages(dir string) error {
--- a/misc/gnopreview/main.go
+++ b/misc/gnopreview/main.go
@@ -272,7 +272,7 @@ func renderBase(cfg config, plan *Plan, head *Crawler) (*Crawler, map[string]boo
-	if err := base.Write(cfg.out, ""); err != nil {
+	if err := base.WritePages(cfg.out); err != nil {
```

Write goes from 22 lines to 19, and `base.Write(out, "")` ran exactly `c.writePages(out)`.

</details>

## misc/gnopreview/crawl.go:314 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L314) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L314) · Suggestion
Suggestion: the count this guards is [`/public/` summed over every copied `.js` and `.css`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L573), while the invariant the message names is the `/public/js/controller-` prefix in `js/index.js`, so any other asset gaining an absolute reference keeps the warning silent when that prefix goes. Counting the named prefix in the named file, and returning an error instead of printing one line, is what stops a preview with dead interactive controls from shipping.

## SKIP misc/gnopreview/crawl.go:328 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L328) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L328) · Suggestion
Suggestion: the map read takes no `ok`, so any URL in `order` without an entry in `pages` dereferences nil on the next line and kills the render after gnodev has been booted.

Skipped: the same edit at [crawl.go:70](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L70) deletes `order` and with it the only way the two can disagree.

## misc/gnopreview/crawl.go:421 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L421) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L421) · Suggestion
Suggestion: `err != nil || code != http.StatusTooManyRequests` returns on the first attempt for a dial error and for any 5xx, so only 429 reaches the backoff, and both callers drop the page for good, [continuing on the error](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L130) and [on the non-200](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L134). Retrying a transport error and a 5xx alongside the 429 keeps a page the node was slow to serve.

<details>
<summary>mechanism</summary>

A page dropped here is absent from `c.pages`, so [`mapURL`'s fallback](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L405) rewrites every link to it as `c.Live` and the reviewer reads production, with one stderr line in a render log the sticky comment does not link.

</details>

## misc/gnopreview/crawl.go:444 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L444) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl.go#L444) · Suggestion
Suggestion: `io.ReadAll` takes a response whole with no byte cap anywhere in the crawler, and [every captured body is held](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L142) until `Write`, so peak memory is page count times page size with only [the count bounded](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L118). An `io.LimitReader` at a documented cap, counted against a total byte budget beside `MaxPages`, bounds the other half.

## misc/gnopreview/crawl_test.go:61-62 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L61-L62) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L61) · Suggestion
Test: these two rows pin `_dir` and `_root` output for paths [`Seeds`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L75) never emits and [`inScope`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L191-L193) refuses, so [the branches they cover](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L490-L498) are unreachable from a crawl and read as live behaviour. Say in the rows' comment that they cover a listing URL nothing queues.

<details>
<summary>observed</summary>

`urlToFile` has two non-test callers, [the dequeue in `Run`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L142) and [`comment.go`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L175-L176) on `urlOf(realm)`. A dequeued URL is a seed or passed `inScope`, `Seeds` builds its directory entries with `path.Dir` and stops at two slashes, and `urlOf` returns `"/" + rest`, so no input ends in `/` or is `/`. The branches stay: they are what keeps `/r/x/y` and `/r/x/y/` on separate files in [`TestURLToFileIsSafe`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L261-L271).

</details>

## SKIP misc/gnopreview/crawl_test.go:70 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L70) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L70) · Suggestion
Test: the directory page the `_dir` branch was written for arrives as `path.Dir` output, `/r/gnoland` for `/r/gnoland/home`, which carries no trailing slash, so that page is captured through the ordinary path into `r/gnoland/index.html` and the branch has no input.

Skipped: the same edit as the posted `crawl_test.go:61-62` section, which carries the call-site trace.

## misc/gnopreview/crawl_test.go:276 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L276) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L276) · Suggestion
Test: this bound allows a segment of `maxSlugLen+16`, 80 characters, while [`slug`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L532) emits at most `maxSlugLen+9`, 73, so widening the digest to `sum[:6]` puts a segment at 77 and the only assertion on the length cap still passes.

```suggestion
		if len(seg) > maxSlugLen+9 {
```

<details>
<summary>observed</summary>

With both `hex.EncodeToString(sum[:4])` calls replaced by `sum[:6]`, only [`TestURLToFile`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L49) goes red, on its hardcoded digests, and this test reports `ok`. The bound is written against `maxSlugLen` itself, so raising [`maxSlugLen`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L510) to 400 raises the bound with it. One line for one line.

</details>

## misc/gnopreview/crawl_test.go:297-301 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L297-L301) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/crawl_test.go#L297) · Suggestion
Test: this loop looks only for `/r` and `/p`, both absent whether the directory page was seeded or dropped, so moving [the seed depth bound](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L95) to `>= 3` sends every `/r/<namespace>` listing to the live site and the package stays green. One equality against the four seeds rejects the over-seed this names and the under-seed it cannot see, at four lines for five.

```suggestion
	want := []string{"/r/gnoland/home", "/r/gnoland/home$source", "/r/gnoland/home$help", "/r/gnoland"}
	if got := c.Seeds(); !reflect.DeepEqual(got, want) {
		t.Errorf("Seeds() = %v; want %v", got, want)
	}
```

<details>
<summary>observed</summary>

Under that mutation `Seeds()` returns three entries instead of four and the whole package reports `ok`. With the equality in place it fails on the missing `/r/gnoland`. `reflect` is already imported for [`TestLinks`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl_test.go#L117). Function 10 lines to 9.

</details>

## misc/gnopreview/main.go:63 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L63) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L63) · Suggestion
Suggestion: one flag set is built before the switch on the subcommand, so `plan` advertises and accepts the ten flags only [`render`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L105-L161) reads, and a render flag put on the plan step exits 0 with no warning. Register `-root`, `-changed` and `-max-realms` on both and the other ten on a render-only set built after the command is known.

## misc/gnopreview/main.go:102 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L102) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L102) · Suggestion
Suggestion: the command name is checked after [the changed list is read from stdin](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L87), which [defaults to `-`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L66), so `gnopreview help` on a terminal blocks until it is interrupted and never reaches this line. Validating `cmd` right after it is split off prints the usage instead.

## misc/gnopreview/main.go:77-79 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L77-L79) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L77) · Suggestion
Suggestion: [the flag set is built with `flag.ExitOnError`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L63), so `Parse` exits the process with its own usage on a bad flag and this branch cannot run, which leaves it reading as proof that a flag error travels [the same funnel every other error takes](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L49-L53).

```suggestion
	// ExitOnError: Parse never returns, it exits 2 with the usage on a bad flag.
	_ = fs.Parse(args)
```

## misc/gnopreview/main.go:131 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L131) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L131) · Suggestion
Suggestion: [`Plan.Empty`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L91) admits a gnoweb plan carrying no realms and this line indexes `plan.Realms[0]` on it, so a gnoweb change meeting a tree where [none of the four seed realms resolve](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L264-L266) ends on a panic rather than the `nothing to preview` message above. Giving [the guard](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L111) the realm count this line depends on keeps the two in step.

<details>
<summary>state the package's own tests assert</summary>

`plan_test.go`'s `gnoweb alone` case pins `wantGnoweb` true with `wantRealms` empty, and `BuildPlan` over a tree holding no seed realm returns `Gnoweb=true Realms=[] Empty()=false Mode()="gnoweb"`, from which `urlOf(plan.Realms[0])` panics with `index out of range [0] with length 0`.

</details>

## misc/gnopreview/main.go:147 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L147) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L147) · Suggestion
Suggestion: a pull request touching gnoweb and a realm takes this arm and leaves `plan.Shots` nil, so the change where the chrome moved gets [the gnoweb lead-in](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L48) with no sample of the chrome under it. Taking the screenshots whenever `plan.Gnoweb` holds, independently of the pairs, fills that slot.

## misc/gnopreview/main.go:167 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L167) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L167) · Suggestion
Suggestion: `root` and `port` arrive as parameters while the `cfg` beside them carries both fields, and [the base pass](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L252) leaves those fields naming the head tree and the head web port, so the first body line spelling `cfg.root` or `cfg.port` boots the base render against the head checkout. Setting `cfg.root, cfg.port = cfg.baseRoot, cfg.port+1` on the caller's copy drops both parameters and leaves one source for each value.

## misc/gnopreview/main.go:183 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L183) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L183) · Suggestion
Suggestion: the gnodev log is created inside `cfg.out`, the directory the workflow uploads as the artifact, so what keeps `gnodev.log` and `gnodev-base.log` off the published site is [three names hardcoded in another file](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L103). Give render a scratch directory of its own for both logs, so the output directory holds only what is meant to be served.

## misc/gnopreview/main.go:208 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L208) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L208) · Suggestion
Suggestion: this wait has no deadline and no `SIGKILL` escalation, so a gnodev that does not exit on [the SIGTERM above](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L207) hangs the process with every artifact already written, and no `timeout-minutes` on the preview job bounds what that costs.

## misc/gnopreview/main.go:267 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L267) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/main.go#L267) · Suggestion
Suggestion: this pass repeats [the head pass's boot, wait, crawl and write sequence](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L117-L140) step for step, and the two copies have already drifted, because the `realms[0]` probe is guarded by [an empty check](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L248) here and by [none](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L131) in the head pass, which a gnoweb-only plan whose four seed realms have all been renamed out of `examples/` reaches with an empty slice. One `crawlTree` holding the sequence and the guard once leaves the two callers differing only in their arguments.

<details>
<summary>measured</summary>

Folding the sequence into a 17-line `crawlTree` takes `render` from 58 lines to 46 and `renderBase` from 50 to 35: 108 lines carrying the sequence twice become 99 carrying it once, a net 4 lines off the file, `gofmt` clean and the package's tests still green.

</details>

## misc/gnopreview/plan.go:28-32 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L28-L32) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L28) · Suggestion
Suggestion: this list leaves out `misc/gnopreview/`, so a pull request that [triggers a run](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L24-L25) by touching only the preview tool sets `Gnoweb` false, selects no realm, and stops at [the skip gate](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L79-L82). Adding `"misc/gnopreview/"` here renders the seed sample against the changed crawler.

<details>
<summary>observed</summary>

`gnopreview plan` over a changed set of `misc/gnopreview/crawl.go` alone emits `"gnoweb": false` with `"realms": []`, which is the `no` branch of the skip gate. The binary is still built and `plan` still runs, so the tool is compile-checked and nothing is crawled, snapshotted or commented.

</details>

## misc/gnopreview/plan.go:58 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L58) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L58) · Suggestion
Refactor: `Draft` is [written for every package](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L142) and read nowhere, unlike [`Ignore`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220), so [`modFlags`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L155-L169) scans every `gnomod.toml` under `examples/` for a `draft = true` nothing consults: 12 lines to 8.

## misc/gnopreview/plan.go:55 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L55) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L55) · Suggestion
Suggestion: `Dir` is `examples/` plus `Path` for every package, since [the walk is rooted at examples/gno.land](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L109) and both fields come off one string at [plan.go:136-139](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L136-L139); dropping the field and the `byDir` index at [plan.go:199-201](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L199-L201) turns its two consumers into direct `pkgs` lookups and takes the file from 345 to 337 lines, with one fewer pair of indexes a reader has to prove in sync.

## SKIP misc/gnopreview/plan.go:75 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L75) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L75) · Suggestion
Suggestion: `Dirs` is documented as the package dirs handed to gnodev and says nothing about being index-aligned with `Realms`, which is how [main.go:241](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L241) reads it.

Skipped: the half that changes behaviour is the missing assertion, posted at plan.go:271-276, and what is left is a doc comment's own wording.

## misc/gnopreview/plan.go:91 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L91) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L91) · Suggestion
Suggestion: `Empty` returns false for a gnoweb plan holding no realms, and [`render` then evaluates `plan.Realms[0]`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L131), which a gnoweb-only change reaches once [the four seed realms](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L37-L42) stop resolving at [the guard that appends them](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L265).

## misc/gnopreview/plan.go:114 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L114) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L114) · Suggestion
Suggestion: returning `nil` from a `filepath.WalkDir` callback on a directory means carry on, so the dotted-directory test drops that directory's own package and walks every child anyway, where `fs.SkipDir` is what prunes the subtree.

```suggestion
		if !d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
```

## misc/gnopreview/plan.go:114 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L114) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L114) · Suggestion
Suggestion: [`filepath.WalkDir`](https://pkg.go.dev/path/filepath#WalkDir) prunes a directory only on an `fs.SkipDir` return, so this `return nil` drops the dot-prefixed directory entry and still descends into every package under it, of which `examples/gno.land` holds none today. Return `fs.SkipDir` for the directory case, keeping `return nil` for a non-directory.

<details>
<summary>observed</summary>

`find examples/gno.land -type d -name '.*'` returns nothing, so the guard protects nothing and misstates what it does. A package at `examples/gno.land/r/.staging/foo`, a `gnomod.toml` plus one `.gno` file, would be registered in `pkgs`, enter the dependents graph and be rendered.

</details>

## misc/gnopreview/plan.go:138 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L138) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L138) · Suggestion
Suggestion: `Pkg.Path` comes from the directory while [the reverse import graph](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L299-L300) is keyed by the import strings found in source, so a package whose `gnomod.toml` module line differs from its directory loses every dependent realm, and all 145 `gnomod.toml` files under `examples/gno.land` agree today. Read the module path out of `gnomod.toml` here, or assert the two are equal in `LoadPkgs`.

<details>
<summary>observed</summary>

Built as two temporary trees: with `module = gno.land/p/x/base` declared under `.../base/v2`, a realm importing `gno.land/p/x/base` gives `Realms == []`; with the module line matching the directory the same realm gives `Realms == [gno.land/r/x/leaf]`. Every fixture in [`plan_test.go`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L29) writes the two equal, so no test covers the day they stop matching.

</details>

## misc/gnopreview/plan.go:160-166 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L160-L166) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L160) · Suggestion
Suggestion: the switch compares each line with its spaces stripped against the exact string `ignore=true`, so `ignore = true # not deployable` and a tab-separated `ignore	=	true` both read as absent and the package whose author opted out is previewed; cutting each line at `#` and trimming on any whitespace covers both, and the repo's own gnomod TOML parser covers the rest.

## misc/gnopreview/plan.go:161-166 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L161-L166) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L161) · Suggestion
Suggestion: this comparison strips spaces and matches the whole line, so `ignore = true # quarantined` reads as `ignore=false` and the realm passes [both `!p.Ignore` guards](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) and is handed to gnodev, which is what the flag exists to prevent. Parse with [`gnovm/pkg/gnomod`](https://github.com/gnolang/gno/blob/ecf7af0/gnovm/pkg/gnomod/file.go#L21), whose `File` already declares `Ignore` with a `toml:"ignore,omitempty"` tag.

<details>
<summary>observed</summary>

| line | read as |
| --- | --- |
| `ignore = true` | `ignore=true` |
| `ignore = true # quarantined until X` | `ignore=false` |
| `draft = true # not ready` | `draft=false` |
| `ignore\t=\ttrue` | `ignore=false` |

No `gnomod.toml` under `examples/` carries a comment or a tab on either line today.

</details>

## misc/gnopreview/plan.go:206 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L206) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L206) · Suggestion
Suggestion: `changedFiles` is a map of maps deduping base names that repeat only when the changed-file list carries one path twice, so `map[string][]string` with an `append` at [plan.go:222-225](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L222-L225), sorted and compacted where [plan.go:274](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L274) calls `sortedKeys`, keeps the output byte for byte and drops the nil check, one allocation per changed package and two lines, 345 to 343.

## misc/gnopreview/plan.go:220 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L220) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L220) · Suggestion
Suggestion: the `byDir` lookup resolves a changed path against [the head tree](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L107-L110), so a realm the pull request removes has no package, falls through unrecorded, and reaches neither [`ChangedRealms`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L231) nor [`Dropped`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L257). Count a changed path under `examples/gno.land` that resolves to no package, and name those realms in the comment as removed.

<details>
<summary>observed</summary>

`BuildPlan(root, []string{"examples/gno.land/r/does/not/exist/lib.gno"}, 25)` returns `Empty() == true`, `ChangedRealms` empty and `Dropped == 0`, with no error. The same path pointed at a package that exists returns that realm in `ChangedRealms`. A pull request that removes `gno.land/r/x/a` while editing `gno.land/r/x/b` therefore comments on `b` alone.

</details>

## misc/gnopreview/plan.go:225 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L225) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L225) · Suggestion
Suggestion: a deleted file's base name is recorded here too, because its directory still resolves, and [wantFile](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L227-L230) renders a per-file `$source` page only for a name in that set, so a PR that only removes files gets the realm render and not one source page; skipping a changed path absent from the head tree, and falling back to the gnoweb file budget when a realm's set comes out empty, puts those pages back.

## misc/gnopreview/plan.go:253-255 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L253-L255) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L253) · Suggestion
Suggestion: the comparator carries one key, `changedPkgs`, so among the indirectly affected realms the cap keeps the alphabetic head: a change to a widely imported package previews the realms under `gno.land/r/archive/` and cuts every one under `gno.land/r/sys/`, which a second key sorting an archived prefix last would keep.

## misc/gnopreview/plan.go:263-268 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L263-L268) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L263) · Suggestion
Suggestion: this loop appends the seed realms after the cap and the `Dropped` count are settled at [plan.go:256-258](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L256-L258), so folding the seeds in above the truncation is what makes `-max-realms` a bound.

- A diff under any of the [gnoweb paths](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L28-L32) with more affected realms than the cap renders up to four realms over it, each one another gnodev load and another crawl against the shared page budget.
- `Dropped` keeps the pre-seed number, so a seed realm the cap cut and this loop put back is reported to the reader as not rendered.

## misc/gnopreview/plan.go:265 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L265) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L265) · Suggestion
Suggestion: this guard admits a seed realm on existence alone, where the test it wants is the `Ignore` filter every other realm takes at [plan.go:241](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L241) plus a prefix match on the version.

- An `ignore = true` seed realm passes it, and is loaded into gnodev and rendered.
- A seed whose path does not resolve is dropped with no log line, and the list pins [gno.land/r/gnoland/boards2/v0](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L40), the realm the `$help` forms are photographed from, so a version renumber shrinks the sample a gnoweb PR shows and the comment reads as the full one.

## misc/gnopreview/plan.go:284 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L284) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L284) · Suggestion
Refactor: the `_test.gno` and `_filetest.gno` pair is spelled again at [plan.go:128](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L128), where it decides a package's deployable file list, so a third test suffix added to one site alone triggers a realm preview for a file that realm's own file list omits.

## misc/gnopreview/plan.go:335 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L335) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L335) · Suggestion
Refactor: `contains` is a one-line alias of `slices.Contains`, which [`wantFile` calls directly](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L229), so a reader grepping the module for `slices.Contains` finds one of its four call sites: 7 lines to 5 across three files.

## misc/gnopreview/plan.go:335-337 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L335-L337) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L335) · Suggestion
Suggestion: `contains` is a one-line alias of `slices.Contains`, which [crawl.go:229](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L229) already calls under its own name, so deleting it and calling `slices.Contains` at [main.go:238](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L238), [plan.go:265](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L265) and [comment.go:161](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L161) leaves one name for the operation and takes the three files from 880 to 878 lines.

## misc/gnopreview/plan.go:339 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L339) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan.go#L339) · Suggestion
Refactor: `mustRel` panics for [one caller](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L136) inside a `WalkDir` callback that already returns error, and [its panic message](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L342) is the file's only use of `fmt`: 10 lines to 5, and the import goes with it.

## misc/gnopreview/plan_test.go:54 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L54) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L54) · Suggestion
Suggestion: [`BuildPlan`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L193) calls `LoadPkgs` itself, so every graph case needs a directory tree on disk here, and extracting the derivation under it into `planFrom(pkgs, changed, maxRealms)` costs 5 lines in `plan.go` and lets a draft package, a cycle, a source that does not parse or a realm count above the cap be a `map[string]*Pkg` literal.

<details><summary>the seam, and one case rewritten on it</summary>

The split is mechanical: `BuildPlan` keeps the `LoadPkgs` call and returns `planFrom(pkgs, changed, maxRealms), nil`, and the rest of its body moves into `planFrom` with `return plan, nil` becoming `return plan`. `plan.go` goes from 345 to 350 lines, `go vet` stays silent and the package stays green.

```go
// planFrom derives the plan from an already-loaded package graph.
func planFrom(pkgs map[string]*Pkg, changed []string, maxRealms int) *Plan {
```

The transitive-dependents case and a cap case on that seam, with no files on disk:

```go
func TestPlanFrom(t *testing.T) {
	t.Parallel()
	pkgs := map[string]*Pkg{
		"gno.land/p/x/base/v0": {Path: "gno.land/p/x/base/v0", Dir: "examples/gno.land/p/x/base/v0"},
		"gno.land/p/x/mid/v0":  {Path: "gno.land/p/x/mid/v0", Dir: "examples/gno.land/p/x/mid/v0", Imports: []string{"gno.land/p/x/base/v0"}},
		"gno.land/r/x/leaf":    {Path: "gno.land/r/x/leaf", Dir: "examples/gno.land/r/x/leaf", Realm: true, Imports: []string{"gno.land/p/x/mid/v0"}},
		"gno.land/r/x/other":   {Path: "gno.land/r/x/other", Dir: "examples/gno.land/r/x/other", Realm: true, Imports: []string{"gno.land/p/x/base/v0"}},
	}
	changed := []string{"examples/gno.land/p/x/base/v0/lib.gno"}
	got := planFrom(pkgs, changed, defaultMaxRealms)
	want := []string{"gno.land/r/x/leaf", "gno.land/r/x/other"}
	if !reflect.DeepEqual(got.Realms, want) {
		t.Errorf("Realms = %#v; want %#v", got.Realms, want)
	}
	if capped := planFrom(pkgs, changed, 1); len(capped.Realms) != 1 || capped.Dropped != 1 {
		t.Errorf("cap 1: Realms = %v, Dropped = %d", capped.Realms, capped.Dropped)
	}
}
```

The payoff is expressiveness rather than time: `fakeRepo` is built once per `TestBuildPlan` and all ten subtests run in 0.005s.

</details>

## misc/gnopreview/plan_test.go:132-136 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L132-L136) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/plan_test.go#L132) · Suggestion
Test: `cmp.Or` folds the four-line default into the [`BuildPlan`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L193) call, and one `mustContain` helper folds the three copies of the same loop at [`plan_test.go:195-208`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L195-L208), [`plan_test.go:223-231`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L223-L231) and [`plan_test.go:245-255`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan_test.go#L245-L255), taking the file from 278 lines to 273 and reporting a missing string at the calling test's line. The block below needs `cmp` added to the import list.

```suggestion
			got, err := BuildPlan(root, tc.changed, cmp.Or(tc.maxRealms, defaultMaxRealms))
```

<details>
<summary>the helper and one converted call site</summary>

```go
// mustContain fails the test for every want the rendered comment is missing.
func mustContain(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("comment missing %q\n---\n%s", w, got)
		}
	}
}

	got := Comment(p, "https://example.test/pr-9", "9")
	mustContain(t, got,
		"changes **gnoweb itself**",
		`<img src="https://example.test/pr-9/_shots/home.png"`,
		"https://example.test/pr-9/r/gnoland/home/",
	)
```

</details>

## misc/gnopreview/shots.go:24 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L24) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L24) · Suggestion
Suggestion: `_shots` and the [`_a`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L500) and [`_t`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L503) directories all open on an underscore, so every published image and tab page rests on a `.nojekyll` marker that [the README calls mandatory](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L96-L97) and [the publish step](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L138-L141) neither writes nor checks.

## SKIP misc/gnopreview/shots.go:27 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L27) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L27) · Suggestion
`beforeDir` names the [base crawl's output prefix](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L265) rather than a screenshot, and [the tree it writes](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L275) holds one page per changed realm while only the capped PNGs are ever read, all of it copied into the shared site and [counted against the size cap](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L128-L132).

Skipped: the same edit as the section on [shots.go:77](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L77) resolves it, and the declaration's home is a rename the author can fold into it.

## misc/gnopreview/shots.go:38 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L38) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L38) · Suggestion
Suggestion: `URL` re-derives `urlOf(Realm)` [through the head crawler's output file](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L77), and [`pairGrid`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L113-L121) emits the same comment bytes when both of its hrefs read `urlOf(p.Realm)`, the way [the realm list already builds a link](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L63) does, which takes the file from 213 lines to 212.

## misc/gnopreview/shots.go:51 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L51) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L51) · Suggestion
Suggestion: the fifteen lines from this browser lookup to the `_shots` `MkdirAll` are identical to [the same prologue in `Screenshot`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L117-L131) but for one word of one message, so a Chrome flag or a temp profile directory added for one pass silently skips the other and the pair disagrees with the gnoweb sample on the same runner. One `shotEnv(outDir, chrome, lost)` returning the binary, the origin and the closer holds it once.

<details>
<summary>measured</summary>

The helper is not shorter: 169 non-comment lines become 170, `gofmt` clean, `go vet` clean, package tests green. What it buys is one prologue instead of two.

</details>

## misc/gnopreview/shots.go:57-65 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L57-L65) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L57) · Suggestion
Suggestion: a listener the sandbox refuses and a `_shots` directory that cannot be created both return `nil`, the same answer as the missing browser one line up, and [`render` assigns that result without checking it](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L149), so the preview publishes and the comment posts with no images and nothing naming the failure.

## misc/gnopreview/shots.go:72-76 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L72-L76) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L72) · Suggestion
Suggestion: `urlOf(r)` is evaluated three times per realm in this loop, twice in these five lines and once at [the merge-base lookup](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L89), which one local removes for one added line.

```suggestion
		u := urlOf(r)
		afterFile, ok := head.FileOf(u)
		if !ok {
			continue
		}
		name := slug(strings.TrimPrefix(u, "/"))
```

## SKIP misc/gnopreview/shots.go:76 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L76) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L76) · Suggestion
The shot filename is a pure function of the realm path, and [the comment is edited in place](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L174) with [a per-PR URL carrying no sha](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview-publish.yml#L138-L140), so a second push republishes new bytes at a byte-identical image URL inside an unchanged comment body.

Skipped: whether GitHub's image proxy then serves the earlier push's PNG turns on a cache TTL this tree does not settle.

## misc/gnopreview/shots.go:77 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L77) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L77) · Suggestion
Suggestion: `ShotPair` keeps the after page's URL alone, so [the before cell ships as a bare image](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L118) where [the after cell links its page](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L121), and the [`_before/` tree](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/main.go#L265) is published with no link into it, against [the README's click-through to the before page](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L73-L74).

## SKIP misc/gnopreview/shots.go:78 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L78) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L78) · Suggestion
A URL the snapshot server answers 404 on is photographed and [captioned as the realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L121), since every `afterFile` reaching `chromeShot` here is trusted to be served, which holds only while a page's file name and its served path agree.

Skipped: the same edit as the section on [shots.go:187-189](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187-L189) resolves it.

## misc/gnopreview/shots.go:83 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L83) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L83) · Suggestion
Suggestion: each image path is spelled twice, [once for the browser](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L78-L79) and once here for the comment, at three sites, and [the only post-write check](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187) stats the path the browser was handed, so a naming change touching one spelling puts a broken image in the pull request comment. Build the relative path once and hand `filepath.Join(outDir, rel)` to the browser.

<details>
<summary>measured</summary>

With the prologue helper folded in, 169 non-comment lines become 170, `gofmt` clean, `go vet` clean, package tests green.

</details>

## SKIP misc/gnopreview/shots.go:87 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L87) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L87) · Suggestion
Suggestion: a realm present at the merge base whose before shot was not obtained keeps `New` false and `Before` empty, and [`pairGrid`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L106-L113) then publishes a lone linked image with no caption, so a run where `renderBase` returned nothing reads that way for every changed realm.

Skipped: the same edit as the section on line 94, which carries this finding.

## misc/gnopreview/shots.go:107 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L107) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L107) · Suggestion
Suggestion: `shotPlan` re-types three of [the four seed realms](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/plan.go#L37-L42) as raw URLs with no symbol shared between the two lists, and [a lookup miss is a bare `continue`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L134-L136), so a renamed realm costs the sample one image and prints nothing.

## misc/gnopreview/shots.go:134-139 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L134-L139) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L134) · Suggestion
Suggestion: this reads the crawler's unexported page map directly where [`FileOf`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/crawl.go#L338-L344) is exactly that lookup and [the sibling pass](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L72) already calls it, so a later change to how a URL maps to a captured file reaches one pass and not the other.

```suggestion
		src, ok := c.FileOf(s.url)
		if !ok {
			continue
		}
		dst := filepath.Join(outDir, shotsDir, s.name+".png")
		if err := chromeShot(bin, base+"/"+src, dst); err != nil {
```

## misc/gnopreview/shots.go:167-170 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L167-L170) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L167) · Suggestion
Suggestion: `filepath.Abs` only cleans the path, since Chrome is started with no `cmd.Dir` and inherits the Go process's working directory, so a relative `--screenshot=` lands exactly where [`os.Stat`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187) looks for it, and passing `dst` at both sites instead takes the file from 213 lines to 209.

## misc/gnopreview/shots.go:174 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L174) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L174) · Suggestion
Suggestion: `--no-sandbox` is a constant of the argument list with no environment check, so [the README's local `render` command](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/README.md?plain=1#L51-L54) renders pull-request-authored pages unsandboxed on a maintainer's own machine, where the CI container's reason for the flag does not hold.

## misc/gnopreview/shots.go:162-163 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L162-L163) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L162) · Suggestion
Suggestion: `serve` returns the listener as its `io.Closer`, so the [`*http.Server`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L158) is unreachable after the call and the deferred `Close` in [`ScreenshotPairs`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L61) and [`Screenshot`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L127) shuts the socket without draining the server's connections, which costs nothing while the process exits moments later.

## misc/gnopreview/shots.go:180 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L180) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L180) · Suggestion
Suggestion: `--virtual-time-budget` bounds the page's virtual clock and not the browser process, which [`exec.Command`](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L171) starts with no deadline, so a Chrome wedging before it reaches the page holds [the render job](https://github.com/gnolang/gno/blob/ecf7af0/.github/workflows/pr-preview.yml#L36) to GitHub's six-hour default, no `timeout-minutes` being declared on it.

## misc/gnopreview/shots.go:187-189 [gh](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L187-L189) · [↗](../../../../../.worktrees/gno-review-6194/misc/gnopreview/shots.go#L187) · Suggestion
Suggestion: a non-empty file at the destination is the whole verdict, so Chrome's rendering of [the snapshot server's plain-text 404](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/shots.go#L158-L161) passes as a successful shot and [gets captioned as the realm](https://github.com/gnolang/gno/blob/ecf7af0/misc/gnopreview/comment.go#L121).
