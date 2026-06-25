# Review: PR #5835
Event: REQUEST_CHANGES

## Body
Ran the reference realm and seven of the eight harness families on 34ac1e7cd; each vulnerable case is exploitable and its fix blocks the same attack. The eighth, render-map-iteration, is forward-compat, not a live exploit.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/2-34ac1e7cd/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## AGENTS.md:90 [↗](../../../../../.worktrees/gno-review-5835/AGENTS.md#L90)
The caller-identity row prescribes `IsUserCall()` for auth, but that rejects realm-mediated calls, so a realm meant to be called by other realms locks out its legitimate callers. It also drops the `cur.IsCurrent()` guard the guide requires before `cur.Previous()`. The general primitive is `cur.Previous()` under `cur.IsCurrent()`; keep `IsUserCall()` to the payment row.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:56-58 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L56)
`md.EscapeText` is [deprecated](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/moul/md/md.gno#L397); the package says use `sanitize.InlineText`, and [§5.9](https://github.com/gnolang/gno/blob/34ac1e7cd/docs/resources/gno-security-guide.md#L376) of this guide recommends `gno.land/p/nt/markdown/sanitize/v0`. The reference example for safe Render output should not model a deprecated call.

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
The `audit-pattern-harness` module is absent from this matrix, so its Go tests and `TestAgentPatternContract` never run in CI.

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
The page says packages under `examples/gno.land/p/...` may be deployed on public networks, then recommends seven that exist only under `examples/quarantined/`, one with a runnable import block. `examples/README.md` marks that subtree unshipped to genesis and unaudited, so a reader importing `gno.land/p/jeronimoalbi/bitset` will not find it on-chain.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38)
Nit: `path` sits in a manual code span but uses inline-text escaping, so a backtick in `path` closes the span early. Not an injection. `md.InlineCode(path)` is the code-span-safe primitive.

## examples/gno.land/r/docs/security_patterns/gnomod.toml:2 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/gnomod.toml#L2)
Nit: `gno = ""` here, while every other example realm pins `gno = "0.9"`. Set it for consistency.

## misc/audit-pattern-harness/internal/auditpattern/run.go:166 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L166)
Optional: brace-depth tracking counts `{`/`}` inside strings and comments, so a `}` in a string flips a correctly guarded function to a false positive. AGENTS.md now points agents at this harness for unfamiliar realm code, where braces-in-strings are routine. Stripping string and comment spans before counting fixes it; the substring matches and the five-scanners-to-two refactor are smaller items in the same engine.

## misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno:4 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/interface-realm-param/vulnerable/hook.gno#L4)
Optional: the interface-realm-param and callback-param slices teach the realm-into-caller-supplied-code leak, but nothing documents the safe counterpart, threading `cur` through your own concrete `/p/` functions, which is what daokit's interrealm-v2 port needs. A realm value is a frame-scoped capability: `cross(rlm)` works only while `rlm.IsCurrent()` holds, so the only escape is handing the live token to code the realm did not audit. One line drawing that boundary, the danger is a caller-supplied `func`/`interface` value, keeps a reader from concluding never to give a realm to `/p/`.

Repros run at 34ac1e7cd.
