# Findings in posting order, from round assemble: 1 to post, 0 SKIP, 0 refuted kept out

## gno.land/pkg/gnoweb/redirect.go:10 [gh](https://github.com/gnolang/gno/blob/c2e4e0207310376989b2b8d781b002a738fafcb2/gno.land/pkg/gnoweb/redirect.go#L10) · Suggestion
State: CONFIRMED, band: Suggestion, angle: removed
TL;DR: /newsletter ends on a 404 at base and at head, while both homepages link the Mailchimp form that a redirect could point at
Check: curl -s -o /dev/null -w '%{http_code}' https://gno.land/newsletter (302) and https://gno.land/r/gnoland/pages:p/newsletter (404); grep -c list-manage misc/deployments/home-alias/home.mainnet.md (2)
Details: Base maps /newsletter to /r/gnoland/pages:p/newsletter, which 404s on gno.land and onyx; head drops the entry, so the path 404s directly. The visitor outcome is the same 404 either way; the Redirects map at redirect.go:10 could instead map /newsletter to the Mailchimp URL both home files link twice, which answers 200. RedirectMiddleware passes the value to http.Redirect unchanged, which accepts an absolute URL. Whether the list is still wanted is the author's open question.
Evidence: curl: gno.land/newsletter 302 then 404 with -L; onyx.testnets.gno.land/newsletter 302 then 404; /r/gnoland/pages:p/newsletter 404 on both; Mailchimp subscribe URL 200; grep -c list-manage: home.mainnet.md 2, home.testnet.md 2; redirect.go:10 `var Redirects = map[string]string{`
