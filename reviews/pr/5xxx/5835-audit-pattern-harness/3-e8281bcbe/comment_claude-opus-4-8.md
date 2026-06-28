# Review: PR #5835
Event: COMMENT

## Body
Ran the gno-compiling contract against the worktree on e8281bcbe: all eight fixed fixtures compile and all eight vulnerable fixtures flag.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/3-e8281bcbe/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

The `audit-pattern-harness` module is still absent from the [`ci-dir-misc.yml` matrix](https://github.com/gnolang/gno/blob/e8281bcbe/.github/workflows/ci-dir-misc.yml#L24), so its Go tests and [`TestAgentPatternContract`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L184) never run in CI. Add it to the matrix, with a gno toolchain on the job so the compile gate runs too.

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:257-262 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L257)
The new marker check ties each vulnerable hit to a loose substring like `*` or `realm`, but never to a file and line. A rule can still be rewritten to flag a different marker-bearing line and the suite stays green while it stops detecting its vulnerability. Record the expected file and line per vulnerable fixture and assert the hit content, not just the count and a substring.

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

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38)
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

The `\`` is the escape, but inside a code span the backslash is literal, so the span closes at the user's backtick and `b` lands outside it with a dangling backtick.
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:375-379 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L375)
After the whitespace-normalization fix, `render_map_iteration` still matches `range `+name as a substring with no right word boundary, so a map named `scores` flags `range scoresList` (an unrelated slice). A word boundary after the map name removes it.

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

## misc/audit-pattern-harness/internal/auditpattern/run.go:267 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L267)
`origin_caller_auth` flags every `OriginCaller()` read, including a benign `emit("actor", unsafe.OriginCaller().String())` with no comparison and no auth. `exported_pointer_leak` flags an idiomatic `func NewVault() *Vault` constructor yet misses a receiver-method getter `func (r *Registry) GetVault() *Vault`, because [the check](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L332) wants a capital right after `func `. Two of these have no [README "Known limitations"](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/README.md?plain=1#L87) entry. Tighten the heuristics or extend the caveats.

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

func TestReceiverGetterFN(t *testing.T) {
	src := "package x\n\ntype Vault struct{ B int }\ntype Registry struct{ v *Vault }\n\nfunc (r *Registry) GetVault() *Vault {\n\treturn r.v\n}\n"
	t.Logf("exported_pointer_leak receiver-getter hits (0 = missed): %+v", mustRun(t, "exported_pointer_leak", write(t, src)))
}

func mustRun(t *testing.T, rule, dir string) []Hit { h, _ := RunRule(rule, dir); return h }
GO
go test -count=1 -v -run 'TestOriginCallerBenignFP|TestConstructorFP|TestReceiverGetterFN' ./internal/auditpattern/
rm internal/auditpattern/zz_fpfn_test.go
```

```
    origin_caller_auth benign-read hits: [{File:a.gno Line:6 Text:emit("actor", unsafe.OriginCaller().String())}]
    exported_pointer_leak constructor hits: [{File:a.gno Line:5 Text:func NewVault() *Vault {}]
    exported_pointer_leak receiver-getter hits (0 = missed): []
```
</details>

## misc/audit-pattern-harness/internal/auditpattern/run.go:146 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L146)
The five scanners collapse into two shapes: a guard-before-use scan (`currentGuardHits`, [`paymentUserCallHits`](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/internal/auditpattern/run.go#L225)) and a block-scoped scan (`renderMarkdownEscapeHits`, `interfaceRealmParamHits`, `renderMapIterationHits`). The block-scoped three also repeat the `codeLines`/`orig` pairing line for line.

## misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno:4 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4)
The [interface-realm-param](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4) and [callback-param](https://github.com/gnolang/gno/blob/e8281bcbe/misc/audit-pattern-harness/fixtures/callback-param/vulnerable/hooks.gno#L6) fixtures show only the bad pattern, never the safe one of threading `cur` through your own concrete `/p/` functions, which [daokit's interrealm-v2 port](https://github.com/samouraiworld/gnodaokit/pull/64) needs. The danger is a caller-supplied `func` or `interface` value, since a realm token grants authority only while `cur.IsCurrent()` holds. Without one sentence saying so, readers avoid passing realms to `/p/` entirely and lose the safe threading pattern.

Repros run at e8281bcbe.
