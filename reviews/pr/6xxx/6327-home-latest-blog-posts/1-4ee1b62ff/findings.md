# Findings in posting order, from round assemble: 8 to post, 2 SKIP, 2 refuted kept out

## gno.land/pkg/gnoweb/alias_static.go:187 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L187) · Warning
State: CONFIRMED, band: Warning, angle: lines
TL;DR: a static file gnoweb cannot read is dropped without a word: every request keeps serving the old page with a 200, and the next restart fails at boot on that same file
Check: go test ./gno.land/pkg/gnoweb -run TestZZSoloFinder1UnreadableFileIsSilent -count=1 -v as uid 1000, tests/solo-finder-1-unreadable-silent_test.go copied in as zz_solofinder1_test.go
Details: alias_static.go:187-188 run os.Stat and os.ReadFile under `err == nil &&` with no else branch, and staticFile carries no logger (grep -c 'Logger|slog' alias_static.go: 0), so a permission-denied read reaches no surface. Enumerated: readable file, reloaded (both); unreadable file, stale and silent at head, boot failure at base; missing file, stale at head (designed, ADR), boot failure at base; at boot, both fail. At the merge base parseAliases called os.ReadFile and returned 'failed to read static file' (main.go:81-84 of the diff), and the script restarted gnoweb on every change, so the bad file failed at once. At head the failure waits for the next unrelated restart, where it takes gnoweb down.
Evidence: request 0..2: Value="# Old" Kind=1; restart on the same file: err=open /tmp/TestZZSoloFinder1UnreadableFileIsSilent922376377/001/home.md: permission denied; --- PASS
Artifact: tests/solo-finder-1-unreadable-silent_test.go

## gno.land/pkg/gnoweb/alias_static_test.go:254 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static_test.go#L254) · Missing test
State: CONFIRMED, band: Missing test, angle: claims
TL;DR: no test pins the 2 s settle window: TestNewStaticFileAliasReloads stays green with the zeroing block at alias_static.go:190-192 deleted
Check: delete alias_static.go:190-192 and run TestNewStaticFileAliasReloads, then tests/judge-2-settle-window_test.go at head and with the same lines deleted
Details: Every replacement in TestNewStaticFileAliasReloads is backdated a minute, so the mtime or size comparison catches each reload whatever the settle code does. tests/judge-2-settle-window_test.go rewrites the file in place with the same size and the first read's mtime: it passes at head and fails with lines 190-192 deleted.
Evidence: 190-192 deleted: TestNewStaticFileAliasReloads ok 0.038s; TestNewStaticFileAliasRereadsUnsettledFile at head ok 0.049s, with 190-192 deleted FAIL expected: "# Two" actual: "# One"
Artifact: tests/judge-2-settle-window_test.go

## gno.land/pkg/gnoweb/alias_static.go:161 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L161) · Nit
State: CONFIRMED, band: Nit, angle: lines
TL;DR: NewStaticFileAlias reads and parses the file, then current() reads and parses it again, since the staticFile it builds has a zero modTime
Check: read alias_static.go:160-166 and 184-197
Details: Line 165 builds staticFile with modTime and size unset, so `!info.ModTime().Equal(f.modTime)` at line 187 is true for any real file and line 188 reads it a second time; the first read's target survives only when the second read fails. Each static alias, ten in the compose setup, is read and parsed twice at boot.
Evidence: alias_static.go:165 f := &staticFile{path: path, target: NewStaticAlias(string(content))}; 166 return f.current(), nil

## gno.land/pkg/gnoweb/alias_static.go:190 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L190) · Nit
State: CONFIRMED, band: Nit, angle: claims
TL;DR: a modification time ahead of gnoweb's clock keeps time.Since negative, so modTime is zeroed after every read and the file is re-read and re-parsed on every request until the clock passes it
Check: go test ./gno.land/pkg/gnoweb -run TestZZJudge5FutureMtimeRereads -count=1 -v (tests/judge-5-future-mtime_test.go copied in)
Details: alias_static.go:190 compares the file's own mtime with now; for an mtime an hour ahead the test's three same-size, same-mtime rewrites are each returned by the next resolve with modTime zero, which shows a ReadFile and a NewStaticAlias per request. The trigger is a writer whose clock runs ahead or a copy that keeps a future timestamp; the cost is one read and parse per request, under f.mu.
Evidence: resolve 0: Value="# B" modTimeZero=true; resolve 1: Value="# C" modTimeZero=true; resolve 2: Value="# D" modTimeZero=true
Artifact: tests/judge-5-future-mtime_test.go

## SKIP misc/deployments/home-alias/gnoalias.sh:36 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L36) · Nit
State: PLAUSIBLE, band: Nit, angle: page
TL;DR: the date sits under a title of one to three lines, so the three dates of the row land at different heights on desktop
Check: serve the rendered home.md with head gnoweb (--aliases=/=static:home.md) and measure each date paragraph's top at 1280 and 1440
Details: gnoalias.sh:36 prints each title as a bold paragraph above the rest of the column, the date included, so a column's date follows its title's wrapped height. The finder measured 48 px of offset at 1280 and 1440; this judge did not render the page.

## gno.land/pkg/gnoweb/alias_static.go:185 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/gno.land/pkg/gnoweb/alias_static.go#L185) · Suggestion
State: CONFIRMED, band: Suggestion, angle: lines
TL;DR: f.mu is held across os.ReadFile, so one read that blocks stalls every request to the page: a FIFO renamed over home.md leaves 0 of 10 concurrent resolves answered within 1 s
Check: go test ./gno.land/pkg/gnoweb -run TestZZJudge3FifoBlocksAll -count=1 -v (tests/judge-3-fifo-blocks_test.go copied in)
Details: alias_static.go:185-186 lock f.mu for the whole of current(), and line 188 reads the file under it. os.Stat on a FIFO succeeds and its size differs, so the first resolve opens it and blocks until a writer appears; the nine others wait on f.mu. At the merge base the page was served from memory with no I/O. The trigger needs a file whose read blocks (a FIFO, a hung mount), and a hung mount would block os.Stat as well, outside any lock, which the ADR's alternatives already weigh for a slow network filesystem.
Evidence: zz_judge3_test.go:41: served within 1s: 0 of 10
Artifact: tests/judge-3-fifo-blocks_test.go

## SKIP misc/deployments/home-alias/docker-compose.yml:28 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/docker-compose.yml#L28) · Suggestion
State: PLAUSIBLE, band: Suggestion, angle: two versions
TL;DR: the compose file can no longer run a reload command, so a host whose gnoweb predates the change keeps its boot page until gnoweb is recreated
Check: build gnoweb at f915442bf, serve --aliases=/=static:home/home.md, run one head gnoalias.sh round with a changed home-override.md and no reload_cmd, curl /
Details: docker-compose.yml:28 passes no reload_cmd and the container no longer has docker or its socket, while the ADR says a reload is needed until the deployed gnoweb re-reads static files. Not run against a base-built gnoweb. The PR body's Ops section names the order, gnoweb first, and the state heals at the next gnoweb recreate.

## misc/deployments/home-alias/docker-compose.yml:31 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/docker-compose.yml#L31) · Suggestion
State: CONFIRMED, band: Suggestion, angle: page
TL;DR: BLOG_LINK_BASE ships empty, so a testnet serving home.testnet.md links its three posts to its own blog, which answers 404, while the button beside them already points at https://gno.land
Check: gnokey query vm/qrender -remote https://rpc.onyx.testnets.gno.land -data 'gno.land/r/gnoland/blog:p/onyx-1'; read gnoalias.sh:35 with BLOG_LINK_BASE empty
Details: gnoalias.sh:35 rewrites `](/r/gnoland/blog` to `](" base "/r/gnoland/blog`, a no-op with the empty value docker-compose.yml:31 ships. latest_blogposts always queries https://rpc.gno.land, so the posts are mainnet's whatever the network, and home.testnet.md's button hard-codes https://gno.land. A default of https://gno.land in the script would make the links right on both networks with no setting; the README's step to set it is the only guard.
Evidence: onyx gno.land/r/gnoland/blog:p/onyx-1 -> 'height: 0' 'data: 404'; gnoalias.sh:35 { gsub(/\]\(\/r\/gnoland\/blog/, "](" base "/r/gnoland/blog") }

## misc/deployments/home-alias/gnoalias.sh:11 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L11) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: SLEEPING_TIME=3600 paces every round, so an edit to home-override.md and the retry after a failed first blog query each wait up to an hour, where the base waited 60 s
Check: sh tests/solo-finder-2-hourly-rounds.sh <gnoalias.sh> at head and at the merge base
Details: gnoalias.sh:11 sets one interval for the blog refresh, the override copy and the chain-home poll. A round with no change already ends at cmp without a write (lines 839-842 of the diff) and no longer restarts gnoweb, so a short interval costs one vm/qrender per round. Round 1 with a failed query writes the page with the marker stripped: the 'Latest from the blog' heading with only its button for an hour. Banded Suggestion: the hourly pace is a stated choice of the PR body, and the page recovers on its own at the next round.
Evidence: head: round 1: home.md first line '# Home v1', blog posts in it: 0; next round after sleep 3600 / round 2: '# Edited by the operator', blog posts in it: 1; next round after sleep 3600. base: next round after sleep 60 on both rounds
Artifact: tests/solo-finder-2-hourly-rounds.sh

## misc/deployments/home-alias/gnoalias.sh:63 [gh](https://github.com/gnolang/gno/blob/4ee1b62ff5fa327ac9817b4b0d6f08251765a354/misc/deployments/home-alias/gnoalias.sh#L63) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: while the blog query fails, every edit to home-override.md is held back with the posts, since the round returns before any write
Check: read gnoalias.sh:58-66 at head and the base's override branch (diff lines 825-828)
Details: At head, render_override returns 1 when the marker is present and latest_blogposts prints nothing; check_gnoland_home then runs `[ -f home.md ] && return 1` (line 63), so the operator's edit reaches the page only on the first round whose mainnet query succeeds, an hour apart. At the merge base the branch ran `cp home-override.md home.md` whenever the two differed, whatever the network did. Only an override carrying the marker is affected; one without it never queries the blog.
Evidence: sed -n 63p gnoalias.sh: '      [ -f home.md ] && return 1'; base: if cmp -s home-override.md home.md; then return 1; else cp home-override.md home.md
