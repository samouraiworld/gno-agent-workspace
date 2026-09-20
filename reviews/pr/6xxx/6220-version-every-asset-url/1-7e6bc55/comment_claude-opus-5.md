# PR [#6220](https://github.com/gnolang/gno/pull/6220): fix(gnoweb): version every asset URL the head emits
Posted: https://github.com/gnolang/gno/pull/6220#pullrequestreview-5261200966
Verdict: REQUEST CHANGES, on one defect this branch causes: the two font preloads ask for a URL the stylesheet never requests, so the browser discards both preloads and downloads Intervar a second time on every cold load.
Event: COMMENT
Model: claude-opus-5, trivial review
Commit: 7e6bc5514
Overview: [overview](../overview.md)
Open the code: [github.dev](https://github.dev/gnolang/gno/blob/7e6bc551447b80122b90e3b66f172e2c84bf0414) · [vscode.dev](https://vscode.dev/github/gnolang/gno/blob/7e6bc551447b80122b90e3b66f172e2c84bf0414)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6220 7e6bc5514`
Round: 1. No finder stage: one pass found, ran and judged, booting gnoweb from source at both the head and the merge base and comparing the two in a browser.

## gno.land/pkg/gnoweb/components/layouts/head.html:9-10 [gh](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/components/layouts/head.html#L9-L10) · [↗](../../../../../.worktrees/gno-review-6220/gno.land/pkg/gnoweb/components/layouts/head.html#L9) · Warning [posted](https://github.com/gnolang/gno/pull/6220#discussion_r4057545325)

A preload is matched by URL: these two carry `?v={{ .BuildTime }}` and [the `@font-face` rules naming the same files](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/frontend/css/04-elements.css#L12-L23) carry none, so `Intervar.woff2` downloads twice on a cold load. The stamp belongs in the stylesheet, where it fixes the match and keeps the cache key.

<details>
<summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 6220 -R gnolang/gno
go build -o /tmp/gnoweb ./gno.land/cmd/gnoweb
/tmp/gnoweb --bind 127.0.0.1:18899 --remote https://rpc.gno.land:443 >/tmp/gnoweb.log 2>&1 &
pid=$!
until curl -sS -o /dev/null --max-time 5 http://127.0.0.1:18899/; do sleep 2; done
curl -sS http://127.0.0.1:18899/ | grep -o 'href="[^"]*Intervar[^"]*"'
curl -sS http://127.0.0.1:18899/public/main.css | grep -o 'url([^)]*Intervar[^)]*)'
kill $pid
rm /tmp/gnoweb
```

The first line is the URL the preload asks for and the second the URL the stylesheet asks for; a preload is matched by URL, so these two never meet.

```
href="/public/fonts/intervar/Intervar.woff2?v=20260920235923"
url(fonts/intervar/Intervar.woff2)
```

Loading that page in Chromium records three font requests where the merge base records two:

| Server | Font requests the page makes |
| --- | --- |
| head | `Intervar.woff2?v=20260920235649`, `roboto-mono-normal.woff2?v=20260920235649`, `Intervar.woff2` |
| merge base | `Intervar.woff2`, `roboto-mono-normal.woff2` |

Both spellings serve the same file, 73,080 bytes each, so the third request is the duplicate. The roboto file is [declared as `Roboto` at weight 900](https://github.com/gnolang/gno/blob/7e6bc5514/gno.land/pkg/gnoweb/frontend/css/04-elements.css#L7-L15) and no rule on this page asks for that weight, so its preload goes unclaimed at both ends and the count stays at one.
</details>
