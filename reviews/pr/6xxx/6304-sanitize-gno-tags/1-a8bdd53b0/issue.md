Target: new issue on gnolang/gno
Status: draft, not posted. Searched open gnolang/gno pull requests by the user and the full text for "GFM table pipe sanitize" on 2026-11-27: no open fix. Same output at the merge base d25b308c1 and at a8bdd53b0, per tests/writer-repros-base.out and tests/writer-repros-head.out.

## Title

sanitize: strict Block lets a GFM table without a leading pipe through

## Body

## Description

Strict `sanitize.Block` escapes only a line-leading `|`, and a GFM table needs none, so `Block("a | b\n--- | ---\nc | d")` renders a live `<table>` between the realm's own lines. The [package doc](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L161-L162) says strict `Block` keeps user content from injecting `<table>`-shaped structure. The escape sits in [`escapeLineLeader`](https://github.com/gnolang/gno/blob/a8bdd53b08e988227136f8b74f4cd6571956e5f4/gnovm/stdlibs/chain/markdown/markdown.go#L1123), whose `|` case fires on the first byte of the line only. The `\n\n` envelope still bounds the table, so the realm's text after it renders.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
cat > gno.land/pkg/gnoweb/markdown/zz_repro_test.go <<'EOF'
package markdown

import (
	"strings"
	"testing"
)

func TestRepro(t *testing.T) {
	out := callSanitizeGno(t, "Block", "a | b\n--- | ---\nc | d\n", "", "")
	html := renderMarkdown(t, "REALM BEFORE\n"+out+"\nREALM AFTER\n")
	t.Logf("table=%v", strings.Contains(html, "<table>"))
}
EOF
go test ./gno.land/pkg/gnoweb/markdown -run TestRepro -v -count=1 | grep table=
rm gno.land/pkg/gnoweb/markdown/zz_repro_test.go
```

```
zz_repro_test.go:11: table=true
```

</details>

- [ ] Strict `Block` escapes the first `|` of a GFM delimiter row, or of every line holding an unescaped `|`.
- [ ] A golden case pins `a | b` / `--- | ---` as paragraph text.
