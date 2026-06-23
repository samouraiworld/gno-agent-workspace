# Review: PR #4494
Event: REQUEST_CHANGES

## Body
Checked the parse contract on 99dca9441: a `# txtar:opts` line yields empty args while a `setopts` line returns the flags, so the dropped comment syntax is now inert. Diffed all 51 reformatted archives as a whitespace-normalized multiset: the changes reduce to import ordering, indentation, EOF newlines, and the new directive lines, with no command, assertion, or gas number altered.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/4xxx/4494-txtar-file-options-formatting/1-99dca9441/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/integration/doc.go:12-30 [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/doc.go#L12)
blocking: this section still tells readers to set options with `# txtar:opts <flags>`, which the parser now ignores as a comment, so it documents a syntax that silently does nothing. This is the form reviewers asked be replaced. Update it to `setopts <flags>` and add `setopts` to the command overview.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4494 -R gnolang/gno
cat > gno.land/pkg/integration/zz_probe_test.go <<'EOF'
package integration

import (
	"strings"
	"testing"
)

func TestProbeDoc(t *testing.T) {
	old, _ := captureTopLevelLineArgs(strings.NewReader("# txtar:opts -skip\n"), "setopts")
	new, _ := captureTopLevelLineArgs(strings.NewReader("setopts -skip\n"), "setopts")
	t.Logf("# txtar:opts -> %#v ; setopts -> %#v", old, new)
}
EOF
go test ./gno.land/pkg/integration/ -run TestProbeDoc -v 2>&1 | grep '\->'
rm gno.land/pkg/integration/zz_probe_test.go
```

```
# txtar:opts -> []string{} ; setopts -> []string{"-skip"}
```
</details>

## gno.land/pkg/integration/testdata_test.go:206
Nit: the comment names `ParseTopLevelFlags` and the old `# <prefix>` syntax, but the function is `captureTopLevelLineArgs` and no longer keys off `#`.

## gno.land/pkg/integration/testdata_test.go:219
Nit: typo, "setopts as to be the top level commands" should read "has to be".

## gno.land/pkg/integration/testdata_test.go:207-232 [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L207)
Optional: a `setopts` line below any real command is dropped with no error, so a misplaced directive quietly does nothing. The top-of-file requirement is fine; the silent failure is the trap.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 4494 -R gnolang/gno
cat > gno.land/pkg/integration/zz_probe_test.go <<'EOF'
package integration

import (
	"strings"
	"testing"
)

func TestProbeOrder(t *testing.T) {
	got, _ := captureTopLevelLineArgs(strings.NewReader("gnoland start\nsetopts -skip\n"), "setopts")
	t.Logf("setopts after command -> %#v", got)
}
EOF
go test ./gno.land/pkg/integration/ -run TestProbeOrder -v 2>&1 | grep '\->'
rm gno.land/pkg/integration/zz_probe_test.go
```

```
setopts after command -> []string{}
```
</details>

## gno.land/pkg/integration/testdata_test.go:155-157 [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L155)
Optional: `-skip` (wired here) and `-timeout` have no fixture, so a regression in either passes CI. Only `-no-fmt` and `-no-parallel` are exercised today. A table test over the flag parser would pin the parse contract and cover them.

## gnovm/pkg/gnofmt/package.go:66-68 [↗](../../../../../.worktrees/gno-review-4494/gnovm/pkg/gnofmt/package.go#L66)
Optional: `GetFile` can return nil and the result is dereferenced with no check. Safe in the formatter path where `Files` and `Read` stay in lockstep, but a latent nil-deref for other callers of the newly exported `NewPackage`.
