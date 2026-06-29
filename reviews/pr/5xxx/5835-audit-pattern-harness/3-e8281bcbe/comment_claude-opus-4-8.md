# Review: PR #5835
Posted: https://github.com/gnolang/gno/pull/5835#pullrequestreview-4588350692
Event: COMMENT

## Body
Verified on e8281bcbe.

Follow-up, not blocking: AGENTS.md:98 positions the harness as a regular sanity check, but the docs name it only in one hard-to-spot bullet. Surface it somewhere discoverable like the quickstart.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/3-e8281bcbe/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## SKIP misc/audit-pattern-harness/internal/auditpattern/run_test.go:257-262 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L257)
A rule can still be rewritten to a coincidental match while the contract test stays green. The marker check ties each vulnerable hit to a loose substring like `*` or `realm`, not to a file and line, so a rule that flags a different marker-bearing line still passes while it stops detecting its vulnerability. Record the expected file and line per vulnerable fixture, and assert the hit content, not just the count and a substring.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
cd misc/audit-pattern-harness
# Gut exported_pointer_leak so it never inspects the getter-return shape; it now
# matches any line spelling "*Vault" (the var decl and the getter signature),
# keeping count 2 on the vulnerable fixture, 0 on the fixed, and the "*" marker.
perl -0pi -e 's/func exportedPointerLeakHits\(dir string\) \(\[\]Hit, error\) \{.*?\n\}/func exportedPointerLeakHits(dir string) ([]Hit, error) {\n\treturn lineContainsHits(dir, func(line string) bool { return strings.Contains(line, "*Vault") })\n}/s' internal/auditpattern/run.go
go test -count=1 -run 'TestAgentPatternContract$|TestExportedPointerLeakRule' ./internal/auditpattern/
git checkout -- internal/auditpattern/run.go
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.004s
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:433-443 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L433) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799157)
On code that is not already gofmt-clean, the reported `file:line` and text come from the gofmt-reformatted buffer, not the on-disk file, so a hit can point at the wrong line. Committed and editor-formatted realms are already gofmt-clean, so this only bites on unformatted input such as generated or pasted code.

## SKIP AGENTS.md:98 [↗](../../../../../.worktrees/gno-review-5835/AGENTS.md#L98)
No shipped command runs a rule against arbitrary realm code, though this line asks agents to. The rules scan any directory, but the only command, [`auditpattern`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/cmd/auditpattern/main.go#L19), runs just the bundled fixtures, and the directory-scanning function [`RunRule`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L123) is internal.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:33-41 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L33) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799158)
The patterns are implicit: `Render` prints only the admin, message, and path, and only `assertAdmin` has a comment. Add a short explanation in `Render`, which gnoweb renders as the page body, or comments on the code, saying what each pattern is and why it is safer: the `cur.IsCurrent()` guard, `cur.Previous().Address()` over `OriginCaller()`, and `InlineText` on user text.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799159)

A backtick in `path` closes the manual code span, and `path` reaches `Render` as arbitrary bytes. Backslash-escaping does not help, since a backslash is literal inside a code span. [`md.InlineCode(path)`](https://github.com/gnolang/gno/blob/e8281bcbe/examples/gno.land/p/moul/md/md.gno#L214) is the safe primitive.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
go build -o /tmp/gno ./gnovm/cmd/gno
export GNOROOT=$(pwd)
cd examples/gno.land/r/docs/security_patterns
cat > zz_bt_test.gno <<'GO'
package securitypatterns

import "testing"

func TestBacktickPath(t *testing.T) { t.Logf("%q", Render("a`b")) }
GO
/tmp/gno test -v -run TestBacktickPath .
rm zz_bt_test.gno
```

```
zz_bt_test.go:5: "# Security Patterns\n\n...\n\nPath: `a\`b`\n"
```

The ``\``` is the escape, but inside a code span the backslash is literal, so the span closes at the user's backtick and `b` lands outside it with a dangling backtick.
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:375-379 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L375) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799161)
After the whitespace-normalization fix, `render_map_iteration` still matches `range `+name as a substring with no right word boundary, so a map named `scores` flags `range scoresList`, an unrelated slice. A word boundary after the map name removes it.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
cd misc/audit-pattern-harness
cat > internal/auditpattern/zz_mapfp_test.go <<'GO'
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMapIterSubstringFP(t *testing.T) {
	d := t.TempDir()
	src := "package x\n\nvar scores = map[string]int{}\nvar scoresList = []int{}\n\nfunc Render(path string) string {\n\tfor _, v := range scoresList {\n\t\t_ = v\n\t}\n\treturn \"\"\n}\n"
	os.WriteFile(filepath.Join(d, "a.gno"), []byte(src), 0o644)
	hits, _ := RunRule("render_map_iteration", d)
	if len(hits) != 0 {
		t.Fatalf("unrelated slice flagged: %+v", hits)
	}
}
GO
go test -count=1 -run TestMapIterSubstringFP ./internal/auditpattern/
rm internal/auditpattern/zz_mapfp_test.go
```

```
--- FAIL: TestMapIterSubstringFP (0.00s)
    unrelated slice flagged: [{File:a.gno Line:7 Text:for _, v := range scoresList {}]
FAIL
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:267 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L267) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799162)
`origin_caller_auth` flags every `OriginCaller()` read, including a benign `emit("actor", unsafe.OriginCaller().String())` with no comparison and no auth. [`exported_pointer_leak`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L321) also flags `func NewVault() *Vault`, a normal constructor whose returned pointer is to a fresh object, not shared state, so there is nothing to leak. Neither rule has a [README "Known limitations"](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/README.md?plain=1#L87) entry, so tighten the heuristics or extend the caveats.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
cd misc/audit-pattern-harness
cat > internal/auditpattern/zz_fpfn_test.go <<'GO'
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, src string) string {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "a.gno"), []byte(src), 0o644)
	return d
}

func TestOriginCallerBenignFP(t *testing.T) {
	src := "package x\n\nimport \"chain/runtime/unsafe\"\n\nfunc Log() {\n\temit(\"actor\", unsafe.OriginCaller().String())\n}\n"
	t.Logf("origin_caller_auth benign-read hits: %+v", mustRun(t, "origin_caller_auth", write(t, src)))
}

func TestConstructorFP(t *testing.T) {
	src := "package x\n\ntype Vault struct{ B int }\n\nfunc NewVault() *Vault {\n\treturn &Vault{}\n}\n"
	t.Logf("exported_pointer_leak constructor hits: %+v", mustRun(t, "exported_pointer_leak", write(t, src)))
}

func mustRun(t *testing.T, rule, dir string) []Hit { h, _ := RunRule(rule, dir); return h }
GO
go test -count=1 -v -run 'TestOriginCallerBenignFP|TestConstructorFP' ./internal/auditpattern/
rm internal/auditpattern/zz_fpfn_test.go
```

```
    origin_caller_auth benign-read hits: [{File:a.gno Line:6 Text:emit("actor", unsafe.OriginCaller().String())}]
    exported_pointer_leak constructor hits: [{File:a.gno Line:5 Text:func NewVault() *Vault {}]
```
</details>

## SKIP misc/audit-pattern-harness/internal/auditpattern/run.go:146 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L146)
The five scanners are two shapes. [`currentGuardHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L146) and [`paymentUserCallHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L225) are one guard-before-use scan; [`renderMarkdownEscapeHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L188), [`interfaceRealmParamHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L285), and [`renderMapIterationHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L338) are one block-scoped scan, and each of the three repeats the [`codeLines`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L454)/`orig` pairing line for line.

## docs/resources/gno-security-guide.md:618 [↗](../../../../../.worktrees/gno-review-5835/docs/resources/gno-security-guide.md#L618) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799164)
The docs point a human at the harness but never document it: how to run it, what it outputs, how to read a result. This matters even agent-first: when a human does not understand a flagged result, the agent needs human docs to redirect them to. Document it in the harness README and link this bullet to it.

## misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno:6 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/current-guard/vulnerable/admin.gno#L6) [posted](https://github.com/gnolang/gno/pull/5835#discussion_r3488799166)
None of the 16 fixtures carry a comment, so a human reading a vulnerable/fixed pair learns nothing about what is wrong or what the fix changed. Add a one-line comment marking the vulnerable construct and the fix, here the `cur.Previous()` read with no `cur.IsCurrent()` guard.

## SKIP misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno:4 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4)
Posted in round 2 (https://github.com/gnolang/gno/pull/5835#discussion_r3475424457); the author accepted it ("not done yet ... I'll add it to the security guide alongside the callback/interface families"). Not reposting.

The [interface-realm-param](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4) and [callback-param](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/fixtures/callback-param/vulnerable/hooks.gno#L6) fixtures show only the bad pattern, never the safe one of threading `cur` through your own concrete `/p/` functions, which [daokit's interrealm-v2 port](https://github.com/samouraiworld/gnodaokit/pull/64) needs. The danger is a caller-supplied `func` or `interface` value, since a realm token grants authority only while `cur.IsCurrent()` holds. Without one sentence saying so, readers avoid passing realms to `/p/` entirely and lose the safe threading pattern.
