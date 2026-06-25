# Review: PR #5835
Event: REQUEST_CHANGES

## Body
This is authoritative guidance plus a reference realm, so correctness of the published material is the product, not only the runtime behavior. The auth content is sound. Verified on 34ac1e7cd: the realm's admin guard rejects an intermediate realm that cross-calls `SetMessage`/`TransferAdmin`, so there is no confused-deputy path. Dropping the `IsCurrent()` check from `current-guard/fixed` lets a forged realm value whose `IsCurrent()` is false pass an address-only guard, so that lesson is load-bearing. The vulnerable `IsUser()` payment guard accepts an ephemeral `/e/<addr>/run` realm that the fixed `IsUserCall()` guard rejects.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5835-audit-pattern-harness/2-34ac1e7cd/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## AGENTS.md:90 [↗](../../../../../.worktrees/gno-review-5835/AGENTS.md#L90)
The caller-identity row prescribes `cur.Previous().IsUserCall()` for auth. That primitive rejects realm-mediated calls, so a realm meant to be callable by other realms locks out every legitimate realm caller, and the row also drops the `cur.IsCurrent()` guard this PR's own guide requires before reading `cur.Previous()`. Scope `IsUserCall()` to the payment row; ordinary realm authorization is `cur.Previous()` under `cur.IsCurrent()`.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:56-58 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L56)
`md.EscapeText` is [deprecated](https://github.com/gnolang/gno/blob/34ac1e7cd/examples/gno.land/p/moul/md/md.gno#L397). The package comment says use `sanitize.InlineText` directly, and [§5.9](https://github.com/gnolang/gno/blob/34ac1e7cd/docs/resources/gno-security-guide.md#L376) of this PR's own guide recommends `gno.land/p/nt/markdown/sanitize/v0`. The reference example for safe Render output should not model the deprecated call.

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
The `audit-pattern-harness` module is not in this matrix, so its Go tests and the `TestAgentPatternContract` agent contract never run in CI (`grep -rln 'audit-pattern-harness' .github/` is empty). The README, the security guide, and the new AGENTS.md section call these lessons executable, but nothing executes them, so a later edit to a rule or fixture breaks the contract silently. Add the module to the matrix, with a gno toolchain on PATH so the `.gno` fixtures compile too.

## misc/audit-pattern-harness/internal/auditpattern/run_test.go:189-203 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run_test.go#L189)
`TestAgentPatternContract` checks only `len(hits)` against the want-count, never which file or line matched, and six of the eight rules have no location assertion anywhere. So a rule can be rewritten to flag a coincidental line and the suite stays green while it stops detecting its vulnerability. Record the expected `{file, line}` per vulnerable fixture and assert the hit content, not just the count.

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
The page says packages under `examples/gno.land/p/...` "may be deployed on public networks," then recommends seven that exist only under `examples/quarantined/` (`moul/collection`, `jeronimoalbi/bitset` with a runnable import block, `nt/pausable/v0`, `lou/query`, `agherasie/forms`, `lou/blog`, `morgan/chess`), with no caveat. `examples/README.md` marks that subtree not shipped to genesis and not audited, so a reader importing `gno.land/p/jeronimoalbi/bitset` will not find it on-chain. Label them quarantined / unaudited or drop them.

## examples/gno.land/r/docs/security_patterns/security_patterns.gno:38 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38)
Nit: `path` sits in a manual code span but is escaped with inline-text escaping, so a backtick in `path` closes the span early. Confirmed behaviorally: `Render` of a path containing a backtick emits an unbalanced code span, though not an injection. `md.InlineCode(path)` is the code-span-safe primitive.

## examples/gno.land/r/docs/security_patterns/gnomod.toml:2 [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/gnomod.toml#L2)
Nit: `gno = ""` here, while every other example realm pins `gno = "0.9"`. Set it for consistency.

## misc/audit-pattern-harness/internal/auditpattern/run.go:166 [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L166)
Optional: brace-depth tracking counts `{`/`}` inside string literals and comments, so a `}` in a string flips a correctly guarded function to a false positive. Confirmed: `current_guard` flags a guarded `cur.Previous()` when an earlier line in the same function is `msg := "}"`. AGENTS.md now points agents at this harness for unfamiliar realm code, where braces-in-strings are routine; stripping string and comment spans before counting fixes the class, and the five scanners also reduce to two shared helpers. The separate substring matches (`range scoresList`, `realm` in a comment) are a smaller false-positive source in the same engine.

Repros run at 34ac1e7cd.
