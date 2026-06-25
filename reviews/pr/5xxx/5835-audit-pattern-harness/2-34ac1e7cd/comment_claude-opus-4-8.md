# Review: PR #5835
Event: REQUEST_CHANGES

## Body
Ran the reference realm and the harness guards on 34ac1e7cd; each rejects the attacker case it claims to.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/2-34ac1e7cd/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## AGENTS.md:90 [↗](../../../../../.worktrees/gno-review-5835/AGENTS.md#L90)
The caller-identity row prescribes `cur.Previous().IsUserCall()` for auth, but `IsUserCall()` rejects realm-mediated calls, locking out a realm's legitimate realm callers. It also skips the `cur.IsCurrent()` check [the guide](https://github.com/gnolang/gno/blob/34ac1e7cd/docs/resources/gno-security-guide.md?plain=1#L366) requires before `cur.Previous()`. General auth is `cur.Previous()` under `cur.IsCurrent()`; `IsUserCall()` is for the payment row only.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:56-58 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L56)
`md.EscapeText` is [deprecated](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/moul/md/md.gno#L397); the package says use [`sanitize.InlineText`](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno), and [§5.9](https://github.com/gnolang/gno/blob/34ac1e7cd/docs/resources/gno-security-guide.md?plain=1#L376) of this guide recommends [`gno.land/p/nt/markdown/sanitize/v0`](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/nt/markdown/sanitize/v0). The reference example for safe Render output should not model a deprecated call.

## docs/resources/gno-security-guide.md:340 [↗](../../../../../.worktrees/gno-review-5835/docs/resources/gno-security-guide.md#L340)
This block does not compile under gno 0.9. With the standard `import "chain/runtime"`, `runtime.OriginCaller` is undefined; the symbol lives in [`chain/runtime/unsafe`](https://github.com/gnolang/gno/blob/34ac1e7cd/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L51). The section's table and checklist already use bare `OriginCaller()`, so only this snippet carries the stale `runtime.` qualifier.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
go build -o /tmp/gno ./gnovm/cmd/gno
export GNOROOT=$(pwd)
D=examples/gno.land/r/docs/tmp_oc; mkdir -p "$D"
printf 'module = "gno.land/r/docs/tmp_oc"\ngno = "0.9"\n' > "$D/gnomod.toml"
cat > "$D/a.gno" <<'EOF'
package tmp_oc

import "chain/runtime"

var owner = address("g125em6arxsnj49vx35f0n0z34putv5ty3376fg5")

func F(cur realm) bool { return runtime.OriginCaller() == owner }
EOF
(cd "$D" && /tmp/gno lint .); rm -rf "$D"
```

```
a.gno:7:41: undefined: runtime.OriginCaller (code=gnoTypeCheckError)
```
</details>

## .github/workflows/ci-dir-misc.yml:24 [↗](../../../../../.worktrees/gno-review-5835/.github/workflows/ci-dir-misc.yml#L24)
The `audit-pattern-harness` module is absent from this matrix, so its Go tests and [`TestAgentPatternContract`](https://github.com/gnolang/gno/blob/34ac1e7cd/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L189) never run in CI.

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:189-203 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L189)
`TestAgentPatternContract` checks only the hit count, never which line matched, and six of the eight rules have no location assertion anywhere. So a rule can be rewritten to flag a coincidental line and the suite stays green while it stops detecting its vulnerability.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
cd misc/audit-pattern-harness
# Gut origin_caller_auth so it never inspects OriginCaller(); it now matches the
# vulnerable fixture's import line (present once) and nothing in the fixed fixture.
perl -0pi -e 's/func originCallerAuthHits\(dir string\) \(\[\]Hit, error\) \{.*?\n\}/func originCallerAuthHits(dir string) ([]Hit, error) {\n\treturn lineContainsHits(dir, func(line string) bool { return strings.Contains(line, "chain\/runtime\/unsafe") })\n}/s' internal/auditpattern/run.go
go test -count=1 -run 'TestAgentPatternContract$|TestOriginCallerAuthRule' ./internal/auditpattern/
git checkout -- internal/auditpattern/run.go
```

```
ok  	github.com/gnolang/gno/misc/audit-pattern-harness/internal/auditpattern	0.007s
```
</details>

## docs/resources/community-packages.md:3 [↗](../../../../../.worktrees/gno-review-5835/docs/resources/community-packages.md#L3)
The page recommends seven packages that currently live only under [`examples/quarantined/`](https://github.com/gnolang/gno/tree/34ac1e7cd/examples/quarantined), one with a runnable import block. A reader who imports one, like [`gno.land/p/jeronimoalbi/bitset`](https://github.com/gnolang/gno/tree/34ac1e7cd/examples/quarantined/gno.land/p/jeronimoalbi/bitset), may be confused when it is not yet on-chain. Flag them as quarantined.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38)
A backtick in `path` breaks the code span. gnoweb [decodes the URL args](https://github.com/gnolang/gno/blob/34ac1e7cd/gno.land/pkg/gnoweb/weburl/url.go#L248) without re-escaping, so visiting `...security_patterns:x%60y` makes `Render` see `x`y`. The escape is a backslash, which is literal inside a code span, so the backtick still closes it. Not an injection; [`md.InlineCode(path)`](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/moul/md/md.gno#L214) is the safe primitive.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
go build -o /tmp/gno ./gnovm/cmd/gno
export GNOROOT=$(pwd)
D=examples/gno.land/r/docs/security_patterns
cat > "$D/zz_bt_test.gno" <<'GO'
package securitypatterns

import "testing"

func TestBacktickPath(t *testing.T) { t.Logf("%q", Render("a`b")) }
GO
(cd "$D" && /tmp/gno test -v -run TestBacktickPath .)
rm "$D/zz_bt_test.gno"
```

```
zz_bt_test.go:5: "# Security Patterns\n\n...\n\nPath: `a\`b`\n"
```

The `\`` is the escape, but inside a code span the backslash is literal, so the span closes at the user's backtick and `b` lands outside it with a dangling backtick.
</details>

## examples/gno.land/r/docs/security_patterns/gnomod.toml:2 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/gnomod.toml#L2)
`gno = ""` here, while every other example realm pins `gno = "0.9"`. Set it for consistency.

## misc/audit-pattern-harness/internal/auditpattern/run.go:166 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L166)
Brace-depth tracking counts `{`/`}` inside strings and comments, so a `}` in a string flips a correctly guarded function to a false positive. [AGENTS.md](https://github.com/gnolang/gno/blob/34ac1e7cd/AGENTS.md?plain=1#L98) now points agents at this harness for unfamiliar realm code, where braces-in-strings are routine. Stripping string and comment spans before counting fixes it; the substring matches and the five-scanners-to-two refactor are smaller items in the same engine.

<details><summary>failing test</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5835 -R gnolang/gno
cd misc/audit-pattern-harness
cat > internal/auditpattern/zz_brace_test.go <<'GO'
package auditpattern

import (
	"os"
	"path/filepath"
	"testing"
)

// Correctly guarded: cur.IsCurrent() before cur.Previous(). The "}" in a
// string literal must not be read as the function's closing brace.
func TestBraceInStringFalsePositive(t *testing.T) {
	d := t.TempDir()
	src := "package x\n\nfunc F(cur realm) {\n\tif !cur.IsCurrent() {\n\t\tpanic(\"no\")\n\t}\n\tmsg := \"}\"\n\t_ = cur.Previous()\n\t_ = msg\n}\n"
	os.WriteFile(filepath.Join(d, "a.gno"), []byte(src), 0o644)
	if hits, _ := RunRule("current_guard", d); len(hits) != 0 {
		t.Fatalf("guarded function flagged: %+v", hits)
	}
}
GO
go test -run TestBraceInStringFalsePositive ./internal/auditpattern/
rm internal/auditpattern/zz_brace_test.go
```

```
--- FAIL: TestBraceInStringFalsePositive (0.00s)
    guarded function flagged: [{File:a.gno Line:8 Text:_ = cur.Previous()}]
FAIL
```
</details>

## misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno:4 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4)
The [interface-realm-param](https://github.com/gnolang/gno/blob/34ac1e7cd/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4) and [callback-param](https://github.com/gnolang/gno/blob/34ac1e7cd/misc/audit-pattern-harness/fixtures/callback-param/vulnerable/hooks.gno#L6) slices show the bad pattern, a realm handed to caller-supplied code, but not the safe one: threading `cur` through your own concrete `/p/` functions, which [daokit's interrealm-v2 port](https://github.com/samouraiworld/gnodaokit/pull/64) needs. The danger is specifically a caller-supplied `func` or `interface` value, because a realm token grants authority only while `cur.IsCurrent()` holds. Spell that out in one sentence, or readers avoid passing realms to `/p/` at all and lose the safe threading pattern.
