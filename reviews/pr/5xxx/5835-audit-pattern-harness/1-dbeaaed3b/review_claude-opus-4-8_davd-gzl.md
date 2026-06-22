# PR #5835: feat: add audit pattern harness and security patterns example

URL: https://github.com/gnolang/gno/pull/5835
Author: moul | Base: master | Files: 59 | +2005 -73
Reviewed by: davd-gzl | Model: claude-opus-4-8 (deep) | Commit: dbeaaed3b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5835 dbeaaed3b`

**TL;DR:** Adds a small Go tool under `misc/audit-pattern-harness` that runs `gno test` on paired vulnerable/fixed `.gno` fixtures and checks each against a text-pattern rule, encoding eight recurring smart-contract audit lessons as runnable checks. Alongside it ships a reference realm (`r/docs/security_patterns`) modelling authenticated mutators plus safe `Render` output, and a batch of security/storage doc updates including a new Community Packages page.

**Verdict: REQUEST CHANGES** — three pieces of published-as-authoritative material need a fix: the reference realm escapes `Render` output with a deprecated helper its own guide tells you not to use ([W1](#warnings-should-fix)), the §5.8 anti-pattern snippet uses an undefined symbol and does not compile ([W3](#warnings-should-fix)), and the harness the docs call "executable" never runs in CI ([W4](#warnings-should-fix)). None are security holes; the auth content itself is verified sound.

## Summary

The harness is a separate Go module: each `expected/<slice>.yaml` names a rule and a vulnerable/fixed fixture pair, and `internal/auditpattern` runs `gno test` on each fixture plus a line-based pattern rule, asserting the vulnerable side flags and the fixed side stays clean. The rules are deliberately heuristic (text, not AST), and `README.md` documents the known false-positive/negative classes honestly. The reference realm and the security-guide additions (§5.8 `OriginCaller`-as-auth, §5.9 raw text in `Render`) are the builder-facing payload; the storage docs reframe "prefer AVL" into "choose storage by access pattern." The PR is fully additive and touches no consensus, VM, or stdlib runtime code, so blast radius is confined to docs, examples, and a dev tool. The findings below are correctness defects in material whose whole value is being correct, not runtime risk.

## Glossary

- crossing function — a `.gno` function whose first parameter is `cur realm`; entered via `f(cross, ...)`, which shifts `PreviousRealm`.
- `cur.IsCurrent()` — true only when `cur` is the topmost crossing frame's live realm token; rejects stale or forged realm values.
- `cur.Previous()` — the realm that crossed into the current frame, i.e. the immediate caller.
- `IsUserCall()` — true only for a direct EOA `maketx call`; false for code realms and for ephemeral `maketx run` (`/e/.../run`) realms.
- `OriginCaller()` — the transaction signer; same across intermediate realms, so unsafe as an immediate-caller identity. Lives in `chain/runtime/unsafe`.
- ephemeral realm — a `gno.land/e/<addr>/run` realm created by `maketx run`; `IsUser()` accepts it, `IsUserCall()` does not.

## Critical (must fix)
None.

## Warnings (should fix)

- **[example teaches a deprecated escaper]** [`security_patterns.gno:56-58`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L56-L58) · [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L56) — `escapeRenderText` routes through `md.EscapeText`, which the package marks deprecated; this PR's own §5.9 recommends `sanitize.InlineText`.
  <details><summary>details</summary>

  The realm is published as the reference for "safe Render output," yet [`md.gno:397`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/p/moul/md/md.gno#L397) reads `Deprecated: use sanitize.InlineText directly`, and `EscapeText` now just delegates to `sanitize.InlineText` ([`md.gno:414`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/p/moul/md/md.gno#L414)). Meanwhile [`gno-security-guide.md:393`](https://github.com/gnolang/gno/blob/dbeaaed3b/docs/resources/gno-security-guide.md#L393) (§5.9) recommends `gno.land/p/nt/markdown/sanitize/v0` and `sanitize.InlineText`. The flagship example contradicts the same PR's guidance and models a deprecated call. It compiles and tests pass (the delegation keeps it functionally safe), so this is consistency, not a runtime hole. The author's own thread on this PR asked for the new stdlib/sanitize markdown demo. Fix: import `gno.land/p/nt/markdown/sanitize/v0` and call `sanitize.InlineText` directly.
  </details>

- **[guide snippet does not compile]** [`gno-security-guide.md:340`](https://github.com/gnolang/gno/blob/dbeaaed3b/docs/resources/gno-security-guide.md#L340) · [↗](../../../../../.worktrees/gno-review-5835/docs/resources/gno-security-guide.md#L340) — §5.8's anti-pattern block calls `runtime.OriginCaller()`, undefined under gno 0.9.
  <details><summary>details</summary>

  With the standard `import "chain/runtime"`, `runtime.OriginCaller` does not exist; the symbol lives only in `chain/runtime/unsafe` ([`unsafe.gno:51`](https://github.com/gnolang/gno/blob/dbeaaed3b/gnovm/stdlibs/chain/runtime/unsafe/unsafe.gno#L51)). The same section's table and the deploy checklist use bare `OriginCaller()`, so only the code block carries the stale `runtime.` qualifier. An anti-pattern block should still build, and naming the `unsafe` package actually strengthens the lesson: the dangerous primitive is gated behind `unsafe`. Verified with `gno lint`: `undefined: runtime.OriginCaller (code=gnoTypeCheckError)`. Fix: `import "chain/runtime/unsafe"` and call `unsafe.OriginCaller()`.
  </details>

- **[harness never runs in CI]** [`.github/workflows/ci-dir-misc.yml:24`](https://github.com/gnolang/gno/blob/dbeaaed3b/.github/workflows/ci-dir-misc.yml#L24) · [↗](../../../../../.worktrees/gno-review-5835/.github/workflows/ci-dir-misc.yml#L24) — `audit-pattern-harness` is absent from the fixed misc matrix, so its Go tests and the agent-contract guarantee never execute in CI.
  <details><summary>details</summary>

  `misc/audit-pattern-harness` is its own module, and `ci-dir-misc.yml` (the only workflow matching `misc/**`) runs a hardcoded matrix of `autocounterd, genproto, genstd, goscan, loop`. No workflow runs `go test` in the harness (`grep -rln 'audit-pattern-harness\|auditpattern' .github/` is empty). So the 293-line `run_test.go` suite and `TestAgentPatternContract` (the "every slice flags vulnerable, leaves fixed clean, and is documented" contract) never run. Yet [`README.md:8`](https://github.com/gnolang/gno/blob/dbeaaed3b/misc/audit-pattern-harness/README.md#L8) calls it the "executable audit pattern harness" and [`gno-security-guide.md:619`](https://github.com/gnolang/gno/blob/dbeaaed3b/docs/resources/gno-security-guide.md#L619) says it keeps "recurring audit lessons executable." Nothing executes them, so a later edit to a rule or fixture breaks the contract silently. Fix: add `- audit-pattern-harness` to the matrix. The pure-Go rule/record tests run anywhere; the gno-compiling variant (`TestAgentPatternContractWithGNO`) self-skips without a gno binary, so the job needs a gno toolchain on PATH (or `GNO_BIN`) to cover the fixtures too.
  </details>

## Nits

- [`security_patterns.gno:38`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38) · [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L38) — `path` is wrapped in a manual code span but escaped with inline-text escaping, so a backtick in `path` breaks the span.
  <details><summary>details</summary>

  `"\nPath: ` `` ` `` `" + escapeRenderText(path) + "` `` ` `` `\n"` treats the slot as a code span, but `escapeRenderText` is inline-text escaping. Inside a CommonMark code span a backslash is literal and a raw backtick still closes the span, so the escape is the wrong primitive here. Verified: `Render("a` `` ` `` `b")` emits `Path: ` `` ` ``` a\ ``` ` `` `b` `` ` `` (the user backtick closes early, the added backslash renders literally). Not an injection — `InlineText` still neutralizes `[ ] ( ) < &` — and the realm's own test never feeds a backtick, so it passes. `md.InlineCode(path)` ([`md.gno:214`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/p/moul/md/md.gno#L214)) picks a safe fence length and is the right primitive. The `admin.String()` code span on line 35 is fine (bech32 cannot contain a backtick).
  </details>

- [`gnomod.toml:2`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/r/docs/security_patterns/gnomod.toml#L2) · [↗](../../../../../.worktrees/gno-review-5835/examples/gno.land/r/docs/security_patterns/gnomod.toml#L2) — `gno = ""`; it is the only example realm without `gno = "0.9"` (136 others pin it). `gno mod tidy` does not rewrite it, so CI stays green, but set it for consistency.

- [`render-map-iteration/vulnerable/leaderboard.gno`](https://github.com/gnolang/gno/blob/dbeaaed3b/misc/audit-pattern-harness/fixtures/render-map-iteration/vulnerable/leaderboard.gno) · [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/fixtures/render-map-iteration/vulnerable/leaderboard.gno) — labelled "vulnerable," but gno map iteration is insertion-order deterministic today, so unlike the other seven slices this is a forward-compat/contract lesson, not a live exploit. The docs already nuance this ([`effective-gno.md:720`](https://github.com/gnolang/gno/blob/dbeaaed3b/docs/resources/effective-gno.md#L720)); a one-line note in the fixture or YAML would keep the framing honest. Review-file only.

## Missing Tests
- The reference realm's `Render` test ([`security_patterns_test.gno:27`](https://github.com/gnolang/gno/blob/dbeaaed3b/examples/gno.land/r/docs/security_patterns/security_patterns_test.gno#L27)) never feeds a backtick, so the broken-code-span case in the Nit above is uncaught. Adding a backtick case would have surfaced it. Low priority; covered by the Nit's fix.

## Suggestions
- [`run.go:369`](https://github.com/gnolang/gno/blob/dbeaaed3b/misc/audit-pattern-harness/internal/auditpattern/run.go#L369) · [↗](../../../../../.worktrees/gno-review-5835/misc/audit-pattern-harness/internal/auditpattern/run.go#L369) — rule precision: substring and comment false positives.
  <details><summary>details</summary>

  `render_map_iteration` matches `range `+name as a substring, so a map named `scores` also flags `range scoresList` (a slice). `interface_realm_param` ([`run.go:302`](https://github.com/gnolang/gno/blob/dbeaaed3b/misc/audit-pattern-harness/internal/auditpattern/run.go#L302)) matches the word `realm` on any line inside an `interface {` block, including doc comments, so a comment mentioning "realm" in a realm-free interface flags. Both reproduce as spurious hits. `README.md` carries a blanket "heuristic, expect false positives/negatives" disclaimer, so this is optional polish, not a defect: a word-boundary check and skipping comment lines would cut the most likely false positives on real code.
  </details>

## Open questions
- The harness is one module among the `misc/` tools but is framed (README, security guide, the spec-corpus test) as an enforceable contract rather than a one-off script. If the intent is enforcement, W4's CI wiring is the gap; if it is a manual auditor aid, the "executable/enforceable" wording overstates it. Either resolution is fine; the author should pick one. Not posted as a separate finding — folded into W4.
