# gnoweb re-reads static alias files, and the homepage lists the latest blog posts
Written by claude-opus-5-5.
PR: [gnolang/gno#6327](https://github.com/gnolang/gno/pull/6327)

## TLDR

gnoweb used to read each `--aliases=<path>=static:<file>` page once, at startup, so `gnoalias.sh` restarted gnoweb through the Docker socket whenever it changed `home.md`. With this change `NewStaticFileAlias` keeps the file's path, and each request to the page stats the file and reads it again when its size or modification time differs from the last read. That lets `gnoalias.sh` stop restarting anything: every hour it renders `home-override.md` with the three newest posts of `r/gnoland/blog` in place of a `<!-- latest-blogposts -->` line, and renames the result over `home.md` so gnoweb never reads half a file. The compose file drops the Docker install and the socket mount, which gave the script's container root-equivalent access to its host.

## Words used here

| Name | What it is |
| --- | --- |
| `NewStaticFileAlias` | Builds a static alias from a file path; a file it cannot read fails gnoweb at startup, as before. |
| `staticFile.current()` | Stats the file under a mutex and reads it again when its size or mtime changed; on any error it serves the last page read. |
| `staticFileSettle` | 2 s. A file whose mtime is less than this before the read is read again on the next request, so a write in the same timestamp tick is not cached. |
| `AliasTarget.resolve()` | Returns a static page as the file is now; `prepareIndexBodyView` calls it once per request. |
| `<!-- latest-blogposts -->` | The line of `home-override.md` that `gnoalias.sh` replaces with the first row of the blog index, queried on mainnet with `vm/qrender`. |
| `BLOG_LINK_BASE` | Prefix the script puts before the posts' `/r/gnoland/blog` links: empty by default, `https://gno.land` on a testnet. |
| `SLEEPING_TIME` | Seconds between two rounds of `gnoalias.sh`: 3600. |

<details><summary>Read the code in this order</summary>

`hot` is the risky code to read closely, `warm` other code, `cold` a test or a generated file. Every excerpt is the code after the change.

### 1. `warm` · The re-read, [`gno.land/pkg/gnoweb/alias_static.go`](https://github.com/gnolang/gno/pull/6327/files#diff-d38e3f8ddb82048370ef8ca2667f11194129c8ab3491f4c02a726ec154f2f632R184-R198)
Decides on each request whether the file is read again, and keeps the last page when a stat or a read fails.
**If wrong:** the homepage serves a stale page, or one caught halfway through a write.
```go
func (f *staticFile) current() AliasTarget {
	f.mu.Lock()
	defer f.mu.Unlock()
	if info, err := os.Stat(f.path); err == nil && (!info.ModTime().Equal(f.modTime) || info.Size() != f.size) {
		if content, err := os.ReadFile(f.path); err == nil {
			f.target, f.size, f.modTime = NewStaticAlias(string(content)), info.Size(), info.ModTime()
			if time.Since(f.modTime) < staticFileSettle {
				f.modTime = time.Time{}
			}
		}
	}
	t := f.target
	t.file = f
	return t
}
```

### 2. `warm` · Where a request resolves it, [`gno.land/pkg/gnoweb/handler_http.go`](https://github.com/gnolang/gno/pull/6327/files#diff-14c9af5c4d6d337ded17f989428cb916ecd9ba3db4013cec6e949b7a5555cb8dR638-R639)
Resolves the alias before the page is rendered, so the title and description from the front matter follow the file too.
**If wrong:** the page body updates while its head keeps the startup title.
```go
	aliasTarget, aliasExists := h.Aliases[r.URL.Path]
	aliasTarget = aliasTarget.resolve()
```

### 3. `warm` · Startup, [`gno.land/cmd/gnoweb/main.go`](https://github.com/gnolang/gno/pull/6327/files#diff-a123dd3737a47dbac1001d53f411dfb5034817ca1f4a4cd89e3dffcc003c5a0cR437-R442)
Builds each `static:` alias from its path rather than from its content.
**If wrong:** a missing file no longer stops gnoweb at startup.
```go
			target, err := gnoweb.NewStaticFileAlias(staticFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read static file %s: %w", staticFilePath, err)
			}

			aliases[parts[0]] = target
```

### 4. `hot` · One round of the script, [`misc/deployments/home-alias/gnoalias.sh`](https://github.com/gnolang/gno/pull/6327/files#diff-b832400050fdbd81ae77dbade022bc4fe5472eae50bb6db899f85765905581bbR57-R75)
Renders the override with the posts, keeps the current page when the blog query fails, and renames a new file over `home.md` only when it differs.
**If wrong:** the homepage loses its posts, or its content, on a failed query.
```sh
  if [ -f home-override.md ]; then
    echo "Home page overriden by home-override.md"
    # a failed blog query keeps the current page rather than dropping the posts
    if ! render_override > out.md; then
      echo "Warn: failed to read the latest blog posts."
      rm -f out.md
      [ -f home.md ] && return 1
      # first round: gnoweb cannot start without home.md, so serve the page
      # without the posts until the next round
      grep -vxF "$BLOG_MARKER" home-override.md > home.md
      return 0
    fi
    if cmp -s out.md home.md; then # nothing changed since the last round
      rm out.md
      return 1
    fi
    mv out.md home.md
    return 0
  fi
```

</details>
