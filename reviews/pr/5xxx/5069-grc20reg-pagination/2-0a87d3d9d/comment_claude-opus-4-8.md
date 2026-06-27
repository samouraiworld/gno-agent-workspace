# Review: PR #5069
Event: APPROVE

## Body
Looks good. Confirmed on 0a87d3d9d that `md.EscapeText` neutralizes markdown-meaningful bytes in a token name while leaving plain names unchanged, and `validSymbol` closes the symbol path.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5069-grc20reg-pagination/2-0a87d3d9d/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:78 [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L78)
`validName` allows `[`, `]`, `(`, `)`, and backticks, so a registering realm can inject a link or code span into the listing. The baseline wrapped the name in `md.EscapeText`; wrap `token.GetName()` here too.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5069 -R gnolang/gno
cat > examples/gno.land/r/demo/defi/grc20reg/zz_escape_test.gno <<'EOF'
package grc20reg

import (
	"strings"
	"testing"

	"gno.land/p/demo/tokens/grc20"
)

func TestNameMarkdownInjection(cur realm, t *testing.T) {
	testing.SetRealm(testing.NewCodeRealm("gno.land/r/demo/evilname"))
	token, _ := grc20.NewToken(0, cur, "Evil](https://x.com) `code`", "EVIL", 6)
	Register(cross(cur), token, "")
	home := Render("")
	if !strings.Contains(home, "Evil](https://x.com) `code`") {
		t.Errorf("listing escaped the name (good):\n%s", home)
	}
	detail := Render("gno.land/r/demo/evilname")
	if !strings.Contains(detail, "# Evil](https://x.com) `code`") {
		t.Errorf("detail escaped the name (good):\n%s", detail)
	}
}
EOF
gno test -v -run TestNameMarkdownInjection ./examples/gno.land/r/demo/defi/grc20reg/
rm examples/gno.land/r/demo/defi/grc20reg/zz_escape_test.gno
```

```
=== RUN   TestNameMarkdownInjection
--- PASS: TestNameMarkdownInjection
ok      ./examples/gno.land/r/demo/defi/grc20reg/
```
The test passing means the raw markdown survived into both render outputs.
</details>

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:92 [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L92)
Same unescaped name on the detail page title. Wrap `token.GetName()` in `md.EscapeText` here as well, re-adding the `gno.land/p/moul/md` import.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:66 [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L66)
`MustGetPageByPath(req.RawPath)` re-parses a URL the router already parsed into `req.Query`, and panics on inputs `url.Parse` rejects: `Render("?\x7f")` routes home and panics `invalid path`. Read page and size from `req.Query` via `GetPageWithSize` to drop the second parse.

## examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:55-73 [↗](../../../../../.worktrees/gno-review-5069/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L55)
`TestPagination` checks only the `Page X of Y` substring. It would still pass if `page.Items` were empty. Add out-of-range `?page=999`, custom `?size=3`, and per-page item membership.
