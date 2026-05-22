# PR #2357: feat(gnovm): Executable Markdown

**URL:** https://github.com/gnolang/gno/pull/2357
**Author:** notJoon | **Base:** master | **Files:** 10 | **+1479 -1**
**Reviewed by:** davd-gzl | **Model:** claude-opus-4-7 (1M context)

## Summary

Adds a `gno doctest` subcommand that parses fenced code blocks from a Markdown file and executes Gno blocks against the GnoVM, optionally checking expected output/error and panic behavior. Code lives in two places:

- `gnovm/cmd/gno/doctest.go` — CLI: `-path`, `-run` (regex over block names), `-timeout` (default 30s).
- `gnovm/pkg/doctest/` — parser (goldmark fenced blocks → `codeBlock`) and executor (per-block `gno.Machine` over a `test.ProdStore`).

A code block is treated as Gno when its language tag is `gno` or contains `gnodoctest` (e.g. `go,gnodoctest`, so GitHub still applies Go syntax highlighting). The directive grammar is borrowed from `gnovm/pkg/test/filetest.go`:

| Directive | Form | Purpose |
| --- | --- | --- |
| `Output:` | multi-line, captures following `// ...` lines until next directive / bare `//` / non-comment line | expected stdout |
| `Error:` | multi-line | expected error/panic text |
| `NAME: <name>` | single-line | block name used by `-run`; defaults to `block_<index>` |
| `IGNORE:` | single-line | skip execution |
| `SHOULD_PANIC:` / `SHOULD_PANIC: <msg>` | single-line | block must panic, optionally containing substring |

Expected values prefixed with `regex:` are matched as a Go regexp.

Compared to earlier rounds of the PR (visible in review threads), the implementation has been substantially reduced: the AST-based code-block name generator, the LRU result cache, and the auto-import analyzer are all gone; sdk/vm keeper plumbing has been replaced with `gnovm/pkg/test.ProdStore`; and `ExecuteMatchingCodeBlock` shares a single stdlib store across blocks (each block still runs in its own `BeginTransaction(CacheWrap, ...)`, so package state is isolated per block — verified, see Test Results). What remains is a focused, self-contained tool.

## Test Results

- **Existing tests:**
  - `gnovm/pkg/doctest`: PASS (`go test -count=1 ./pkg/doctest/...`)
  - `gnovm/cmd/gno`: **FAIL** — `TestDoctest/ChainRuntime` errors with `unknown import path chain/runtime` (or `getSessionInfo does not have a body but is not natively defined`, depending on `GNOROOT` resolution). The test imports `chain/runtime`, but `test.ProdStore(rootDir, ...)` only provides `gnovm/stdlibs/...`; `chain/runtime` natives live in `gno.land/pkg/sdk/vm` and are not registered in this store. So the test was either passing on stale gno-skills/gno submodule state or has been broken for a while. See [doctest_test.go:91-93](.worktrees/gno-review-2357/gnovm/cmd/gno/doctest_test.go#L91-L93).
- **CI:** `main / build` and `main / lint` are red on the PR. The actual failure in both is gofmt/goimports complaining about a trailing blank line in [doctest.go:89](.worktrees/gno-review-2357/gnovm/cmd/gno/doctest.go#L89). Build then short-circuits on the dirty diff.
- **Edge-case tests:** 7 written, all PASS (saved under `tests/edge_test.go`). They pin/probe: empty fenced block, package-isolation between shared-store blocks, empty `// Output:` directive, invalid `-run` regex, `regex:` literal collision, conflicting `Output:`/`Error:`, `IGNORE` vs `SHOULD_PANIC` precedence.

## Critical (must fix)

- [ ] `gnovm/cmd/gno/doctest_test.go:56-69, 91-93` — `TestDoctest/ChainRuntime` is broken. The block imports `chain/runtime`, which is not resolvable inside `test.ProdStore`. Either drop the `ChainRuntime` test case, or switch the example to a stdlib-only import (e.g. `strings`, `strconv`), or extend the doctest store setup to include `chain/runtime` natives. As-is the package fails `go test ./gnovm/cmd/gno/...`.
- [ ] `gnovm/cmd/gno/doctest.go:89` — trailing blank line breaks gofmt/goimports → `main / lint` red. Remove the empty line at EOF.

## Warnings (should fix)

- [ ] `gnovm/pkg/doctest/exec.go:223-232` — `matchPattern` silently swallows regex compile errors and treats invalid patterns as "no match." A user running `gno doctest -path foo.md -run "[oops"` gets `No code blocks matched the pattern` with no hint that the pattern is malformed. Compile the pattern once up front in `ExecuteMatchingCodeBlock` and surface the error to the caller. Confirmed by `tests/edge_test.go:TestInvalidPatternSilentlySkips`.
- [ ] `gnovm/pkg/doctest/parser.go:115-169` — `parseBlockMetadata` ignores `scanner.Err()` and uses `bufio.Scanner`'s default 64K line buffer. A pathological code block with a single >64K line (entirely possible in generated/minified examples) would silently truncate metadata. Either check `scanner.Err()` and surface it, or use a larger buffer / a different reader.
- [ ] `gnovm/pkg/doctest/exec.go:180-203` — semantic gap: a block with `// Output:` and no following lines parses as `expectedOutput == ""`, which `executeBlock` treats as "no expectation" and prints whatever was emitted. So an author who wrote `// Output:` to assert *no output* gets the opposite. Either reject empty `Output:`/`Error:` directives at parse time, or treat them as `expected == ""` strictly (i.e. require empty actual). Confirmed by `tests/edge_test.go:TestEmptyOutputDirective`.
- [ ] `gnovm/pkg/doctest/exec.go:54-67, 167-178` — when both `// Output:`/`// Error:` and `SHOULD_PANIC` are set on the same block, the Output/Error expectations are silently dropped (only the panic message is checked). Either reject this combination or document the precedence in the README. Same pattern for `IGNORE` + anything else (IGNORE wins, others ignored — at least it's intuitive, but worth a one-line note).
- [ ] `gnovm/pkg/doctest/parser.go:118-119` — `outputs []string` shadows the package-level concept just fine, but the parallel slice `errors []string` shadows the `errors` import in any future change to this file. Rename to `outputLines`/`errorLines` to remove the foot-gun.

## Nits

- [ ] `gnovm/pkg/doctest/exec_test.go:429` — typo in test name `TestShowingPropoerType` → `TestShowingProperType`.
- [ ] `gnovm/cmd/gno/README.md:21` — `doctest` is inserted between `env` and `fix`; the rest of the list is alphabetical. Move it between `doc` and `env`.
- [ ] `gnovm/pkg/doctest/exec.go:25-27` — `maxAllocBytes = 500_000_000` is hardcoded. Filetests / `gno test` expose this; doctest could expose a `-max-alloc` flag (or just inherit `MaxAllocOverride`). Low priority but easy to hit on benchmarks.
- [ ] `gnovm/pkg/doctest/parser.go:131-135` — `bodyTrim == ""` ends a section; the comment at line 19-20 says "a bare `//`" — accurate, but the implementation also ends on `//   ` (whitespace-only after `//`). Either spell that out in the README or trim only what filetest does.
- [ ] `gnovm/pkg/doctest/exec.go:107-113` — `unsupportedLangResult` re-runs the comma split that `isGnoDoctest` already does. Combining the two would be a one-liner.
- [ ] `gnovm/pkg/doctest/README.md:9` — usage line omits `-timeout`; mention it for parity with `--help`.
- [ ] `gnovm/pkg/doctest/exec.go:38-50` — `ExecuteCodeBlock` is exported but takes the unexported `codeBlock`, so external callers can't construct one. Either unexport `ExecuteCodeBlock` (only `ExecuteMatchingCodeBlock` is callable from outside) or export `CodeBlock`. As-is the asymmetry is confusing.

## Missing Tests

- [ ] Empty Output / Error directives — see Warning above. `tests/edge_test.go:TestEmptyOutputDirective` documents current (surprising) behavior; needs a real test once semantics are decided. (`gnovm/pkg/doctest/exec_test.go`)
- [ ] Invalid `-run` regex — should error rather than silently match nothing. (`gnovm/pkg/doctest/exec_test.go`)
- [ ] `Output` + `SHOULD_PANIC` and `Error` + `SHOULD_PANIC` interactions — currently uncovered. (`gnovm/pkg/doctest/exec_test.go`)
- [ ] CLI: missing `-path` already returns a clear error, but no test covers it. Trivial to add in `gnovm/cmd/gno/doctest_test.go`.
- [ ] CLI: `-timeout` actually firing — no test currently exercises a block that exceeds the timeout. (`gnovm/cmd/gno/doctest_test.go`)

## Suggestions

- The PR description still mentions a "feature to cache execution results" (LRU). That cache was removed in `chore(doctest): remove dead codes`; update the PR description to match what's shipping.
- Consider an ADR under `gnovm/adr/` per `AGENTS.md` ("Every non-trivial AI-assisted PR must include an ADR"). This PR introduces a new subcommand and a new public package — it qualifies. Document the directive grammar choice (filetest reuse vs. a doctest-specific dialect) and the decision to drop the AST name generator / LRU cache.
- `ExecuteMatchingCodeBlock` short-circuits on the first failing block ([exec.go:155-158](.worktrees/gno-review-2357/gnovm/pkg/doctest/exec.go#L155-L158)). For doctest UX it's friendlier to collect all results and report all failures at the end, like `go test`. Otherwise a single broken block in a long tutorial hides every later block. Worth changing before the API stabilizes.
- The two-form language tag (`gno` vs. `go,gnodoctest`) is well-motivated by GitHub highlighting — but a third path emerges for embedmd-driven docs: `gno` blocks that come from `_assets/` and aren't directly authored. Make sure those flow through unchanged; not a blocker, just a thing to confirm with @moul before merging.
- The PR has been open since 2024 and rebased against a much newer master several times. Worth confirming the current `chain/runtime` path actually exists in the same form for the test (and others) on master before re-requesting review.

## Questions for Author

- Why was the LRU result cache dropped? For repeated runs over a large doc, caching by block-content hash seems valuable — was it a complexity/scope cut, or an explicit design call?
- The `ChainRuntime` test imports `chain/runtime`. Was this passing locally for you, and if so, against which `GNOROOT`? It fails in a clean worktree because `test.ProdStore` doesn't bind chain natives.
- Is there an intended interaction with embedmd-style docs (where snippets are extracted from `_assets/`)? Right now doctest reads the rendered Markdown only; if blocks are embedmd-included from elsewhere, doctest sees them but errors don't reference the original source location.
- Should `-run` accept a glob/substring like `go test -run` (which is regex but commonly used as substring), or strictly regex? Current behavior is strictly regex — and silently no-matches on bad syntax (see Warnings).

## Verdict

**REQUEST CHANGES** — The design is sound and the simplification since the earlier rounds is a clear improvement, but two blockers must land before merge: `TestDoctest/ChainRuntime` fails in a clean checkout, and CI is red on a one-line gofmt issue. Once those are fixed and the empty-Output / invalid-regex semantics are tightened, this is close to mergeable.
