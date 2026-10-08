# Review: [#6293](https://github.com/gnolang/gno/pull/6293)

Event: APPROVE
Verdict: APPROVE. Every link on both home files resolves on its network and the two removed redirects pointed at 404s; the one open concern is a Suggestion to map `/newsletter` to the Mailchimp form, which depends on the newsletter question the description leaves open.
Model: claude-opus-5-5, standard review (finder xhigh, judge and writer high)
Commit: c2e4e0207310376989b2b8d781b002a738fafcb2
Overview: [overview](../overview.md)
Open the code: [head](https://github.com/gnolang/gno/tree/c2e4e0207310376989b2b8d781b002a738fafcb2) · `git -C gno worktree add ../.worktrees/gno-review-6293 c2e4e02`
Round: 1. One finder, one reflector filing no candidate of its own, one candidate, judged from scratch by an agent that was not its finder; no Nit batch.

## Body

## gno.land/pkg/gnoweb/redirect.go:10 [gh](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/gno.land/pkg/gnoweb/redirect.go#L10) · [↗](../../../../../.worktrees/gno-review-6293/gno.land/pkg/gnoweb/redirect.go#L10) · Suggestion

Suggestion: `Redirects` has no `/newsletter` entry, so old `gno.land/newsletter` links answer 404; if the Mailchimp list stays, pointing `/newsletter` at its subscribe URL keeps them working.

<details><summary>status codes</summary>

```bash
for h in https://gno.land https://onyx.testnets.gno.land; do
  for p in /newsletter /r/gnoland/pages:p/newsletter; do
    printf '%s%s ' "$h" "$p"; curl -s -o /dev/null -L -w '%{http_code}\n' "$h$p"
  done
done
curl -s -o /dev/null -w '%{http_code}\n' 'https://land.us18.list-manage.com/subscribe?u=8befe3303cf82796d2c1a1aff&id=271812000b'
```

The shortlink ends on a 404 on both networks, through a 302 to a page that does not exist; the form it would reach answers 200.

```
https://gno.land/newsletter 404
https://gno.land/r/gnoland/pages:p/newsletter 404
https://onyx.testnets.gno.land/newsletter 404
https://onyx.testnets.gno.land/r/gnoland/pages:p/newsletter 404
200
```

</details>
