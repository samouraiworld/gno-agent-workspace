# Review: [#6327](https://github.com/gnolang/gno/pull/6327)
Posted: https://github.com/gnolang/gno/pull/6327#pullrequestreview-5551620165
Event: COMMENT
Verdict: APPROVE. Nothing blocks: the reload keeping the last page on a failed read is the documented behaviour, so its missing log is a Nit; the rest is one missing test, Nits and Suggestions.
Model: claude-opus-5-5, standard review; finders xhigh, judge and writer high
Commit: 4ee1b62ff
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/alexiscolin/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354) · [vscode.dev](https://vscode.dev/github/alexiscolin/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6327 4ee1b62ff`
Round: 1. Scope is the pull request's own commits 73a1fc81f and c1ceeae48; the rest of the diff is #6293, merged. Two finders and one judge that also ran the reflector pass, 12 candidates (10 from the finders, 2 from the reflector), each judged from scratch by an agent that was not its finder: 8 confirmed, 2 plausible and skipped, 2 refuted. The parent rebanded the unlogged-read Warning to a Nit, since the doc comment on `NewStaticFileAlias` states that keep-last behaviour, and dropped the unrun fix sentences from two Suggestions, now Nits.

## Body

> AI review, claude-opus-5-5, standard review, reviewed at [4ee1b62ff](https://github.com/gnolang/gno/pull/6327/commits/4ee1b62ff5fa327ac9817b4b0d6f08251765a354), [skills](https://github.com/davd-gzl/skills) · [overview](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6327-home-latest-blog-posts/overview.md) · [claims](https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/6xxx/6327-home-latest-blog-posts/1-4ee1b62ff/claims.md) · Status: APPROVE

## gno.land/pkg/gnoweb/alias_static_test.go:254 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static_test.go#L254) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static_test.go#L254) · Missing test [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641004)

Missing test: `TestNewStaticFileAliasReloads` stays green with the zeroing block at [`alias_static.go:190-192`](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L190-L192) deleted, since every file it renames in is backdated a minute.

<details><summary>test cases</summary>

The case below passes on the branch and fails with lines 190-192 deleted (`expected: "# Two"`, `actual: "# One"`).

```go
func TestNewStaticFileAliasRereadsUnsettledFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	require.NoError(t, os.WriteFile(path, []byte("# One"), 0o644))
	info, err := os.Stat(path)
	require.NoError(t, err)

	target, err := gnoweb.NewStaticFileAlias(path)
	require.NoError(t, err)

	// Rewritten within the same timestamp tick: same size, same mtime.
	require.NoError(t, os.WriteFile(path, []byte("# Two"), 0o644))
	require.NoError(t, os.Chtimes(path, info.ModTime(), info.ModTime()))
	assert.Equal(t, "# Two", gnoweb.ResolveAlias(target).Value)
}
```

</details>

## gno.land/pkg/gnoweb/alias_static.go:161 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L161) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L161) · Nit [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641014)

Nit: `f.current()` reads and parses the file a second time right after this read, since the new `staticFile` starts with a zero `modTime`.

## gno.land/pkg/gnoweb/alias_static.go:185 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L185) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L185) · Nit [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641026)

Nit: `f.mu` is held across `os.ReadFile`, so one blocking read, a FIFO renamed over `home.md`, stalls all 10 concurrent requests past a second.

<details><summary>repro</summary>

```go
// in package gnoweb: 0 of 10 concurrent resolves return within 1 s
func TestZZFifoBlocksAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	os.WriteFile(path, []byte("# Old"), 0o644)
	past := time.Now().Add(-time.Minute)
	os.Chtimes(path, past, past)
	target, _ := NewStaticFileAlias(path)
	syscall.Mkfifo(path+".new", 0o644)
	os.Rename(path+".new", path)
	done := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func() { done <- target.resolve().Value }()
	}
	served, timeout := 0, time.After(time.Second)
loop:
	for served < 10 {
		select {
		case <-done:
			served++
		case <-timeout:
			break loop
		}
	}
	t.Logf("served within 1s: %d of 10", served)
}
```

```
zz_judge3_test.go:41: served within 1s: 0 of 10
```

</details>

## gno.land/pkg/gnoweb/alias_static.go:187 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L187) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L187) · Nit [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641037)

Nit: these `os.Stat` and `os.ReadFile` errors go unlogged, so an unreadable `home.md` keeps serving the old page silently until a restart fails on it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6327 -R gnolang/gno
cat > gno.land/pkg/gnoweb/zz_unreadable_test.go <<'EOF'
package gnoweb_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnolang/gno/gno.land/pkg/gnoweb"
	"github.com/stretchr/testify/require"
)

// Run as a non-root user: root reads a 0000 file.
func TestZZUnreadableFileIsSilent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home.md")
	require.NoError(t, os.WriteFile(path, []byte("# Old"), 0o644))
	target, err := gnoweb.NewStaticFileAlias(path)
	require.NoError(t, err)

	next := path + ".new"
	require.NoError(t, os.WriteFile(next, []byte("# New, longer"), 0o000))
	earlier := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(next, earlier, earlier))
	require.NoError(t, os.Rename(next, path))

	for i := 0; i < 3; i++ {
		t.Logf("request %d: %q", i, gnoweb.ResolveAlias(target).Value)
	}
	_, err = gnoweb.NewStaticFileAlias(path)
	t.Logf("restart: %v", err)
}
EOF
go test ./gno.land/pkg/gnoweb -run TestZZUnreadableFileIsSilent -count=1 -v
rm gno.land/pkg/gnoweb/zz_unreadable_test.go
```

Three requests return the old page and nothing reports the failed read, until a restart on the same file fails:

```
zz_unreadable_test.go:27: request 0: "# Old"
zz_unreadable_test.go:27: request 1: "# Old"
zz_unreadable_test.go:27: request 2: "# Old"
zz_unreadable_test.go:30: restart: open /tmp/…/home.md: permission denied
--- PASS: TestZZUnreadableFileIsSilent (0.00s)
ok  	github.com/gnolang/gno/gno.land/pkg/gnoweb	0.029s
```

</details>

## gno.land/pkg/gnoweb/alias_static.go:190 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L190) · [↗](../../../../../.worktrees/gno-review-6327/gno.land/pkg/gnoweb/alias_static.go#L190) · Nit [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641052)

Nit: with an mtime ahead of gnoweb's clock, `time.Since(f.modTime)` stays negative and each request re-reads and re-parses the page.

## SKIP misc/deployments/home-alias/gnoalias.sh:36 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L36) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L36) · Nit

Nit: each title becomes a bold paragraph above its date, so titles wrapping to different line counts put the three dates at different heights on desktop.

Not posted: one agent alone took the date offsets, and no render of the page ran here.

## misc/deployments/home-alias/gnoalias.sh:63 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L63) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L63) · Nit [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641063)

Nit: this `return 1` blocks every edit to `home-override.md` until the mainnet blog query succeeds again.

## SKIP misc/deployments/home-alias/docker-compose.yml:28 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/docker-compose.yml#L28) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/docker-compose.yml#L28) · Suggestion

Suggestion: this command runs the script without a reload command or `docker`, so a gnoweb built before the re-read keeps its boot page. The README could say to update gnoweb first.

Not posted: the description's Ops section already orders the gnoweb deploy first, and no gnoweb built at the base was run against the script.

## misc/deployments/home-alias/docker-compose.yml:31 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/docker-compose.yml#L31) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/docker-compose.yml#L31) · Suggestion [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641076)

Suggestion: `BLOG_LINK_BASE` ships empty, so on a testnet each mainnet post links to the testnet's own blog and gets a `404`. Defaulting it to `https://gno.land` makes the links right on both networks.

## misc/deployments/home-alias/gnoalias.sh:11 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L11) · [↗](../../../../../.worktrees/gno-review-6327/misc/deployments/home-alias/gnoalias.sh#L11) · Suggestion [posted](https://github.com/gnolang/gno/pull/6327#discussion_r4297641085)

Suggestion: this one interval paces every round, so an edit to `home-override.md` or the retry after a failed first blog query waits up to an hour. A shorter interval costs one `vm/qrender` per round, since a round with nothing new ends at `cmp` without writing.
