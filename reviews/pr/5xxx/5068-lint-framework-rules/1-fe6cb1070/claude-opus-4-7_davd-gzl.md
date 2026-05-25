# PR #5068: feat(gnovm): add extensible linting framework with AVL001 and GLOBAL001 rules

URL: https://github.com/gnolang/gno/pull/5068
Author: mvallenet | Base: master | Files: 50 | +2777 -234
Reviewed by: davd-gzl | Model: claude-opus-4-7

**Verdict: REQUEST CHANGES** — solid framework, three real bugs in `//nolint` parsing and `--disable-rules` flag handling silently mis-suppress or fail to suppress; redundant sort and a dead `Issue.Pos` field already flagged by @notJoon are still open.

## Summary

Replaces the ad-hoc lint logic in `gno lint` with a pluggable framework under [`gnovm/pkg/lint/`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/), shipping three built-in rules (AVL001 unbounded iteration, GLOBAL001 exported package-level vars, RENDER001 invalid Render signature), three reporters (text, json, direct), `//nolint[:RULE,...]` suppression, three modes (`default`/`strict`/`warn-only`), and `--list-rules` / `--disable-rules` flags. Output format changes from `file:l:c: msg (code=X)` to `file:l:c: severity: msg (X)` — breaks any external tooling that parses lint output. The framework is well-shaped (rule self-registration via `init()`, `RuleContext` stack, severity adjustment per mode), but the `//nolint` regex and `--disable-rules` parser have rough edges that silently change behavior, and several findings from prior reviews remain open.

## Glossary

- `Engine.Run` — traverses gnolang AST via `Transcribe`, invoking each rule's `Check` on every node.
- `NolintParser` — regex-driven scan of source for `//nolint[:RULE,...]` directives, keyed by line.
- `RuleContext` — handed to each `Check`, carries `PkgPath`, `IsTest`, `File`, `Source`, and parent-node stack.
- `baseReporter` — buffered, deduplicating impl shared by `TextReporter` and `JSONReporter`; `DirectReporter` streams without buffering.
- `TCLatestRelaxed` — typecheck mode used by lint (vs `TCLatestStrict` when `--auto-gnomod=false`).

## Fix

Before: `gno lint` had a hardcoded set of checks driven by string codes (`gnoTypeCheckError`, etc.) printed via a single format. After: lint is two stages — preprocessing/typecheck issues funnel through `reportError` → `reporter.Report(Issue)` (same path as before), then a second stage walks the preprocessed AST and runs each enabled `Rule.Check`. Mode adjustment runs in `Engine.runOnFile` ([`engine.go:75`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/engine.go#L75)) and exit code is driven by the post-adjustment error count ([`lint.go:398-401`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L398-L401)). `Issue.Pos` plumbing was added but is never read.

## Critical (must fix)

- **[silent over-suppression: any `//nolint*` prefix suppresses everything]** [`gnovm/pkg/lint/nolint.go:8`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/nolint.go#L8) — `^//\s*nolint(?::([A-Za-z0-9_,]+))?` has no word boundary, so `//nolintbar` or `//nolint-typo` matches as `//nolint` with empty rule list, suppressing every rule on the next line.
  <details><summary>details</summary>

  The regex anchors at `//` but doesn't terminate `nolint`. `//nolintbar` produces match `//nolint` with capture group empty, which `addDirective` stores as a directive with `Rules == nil`. `matchesRule` then returns `true` for every rule ID. Repro:

  ```
  $ cat > main.gno <<'EOF'
  package main

  //nolintbar
  var Counter int

  func Render(string) string { return "" }
  EOF
  $ gno lint .
  # (nothing — GLOBAL001 silently suppressed)
  ```

  Fix: anchor with `\b` (`^//\s*nolint\b(?::([A-Za-z0-9_,]+))?`). Verified locally that `//nolintbar`/`//nolint_typo` then no longer match while `//nolint`, `//nolint:AVL001`, `//nolint:A,B` still do.
  </details>

- **[silent failure: `//nolint:A, B` truncates at first space]** [`gnovm/pkg/lint/nolint.go:8`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/nolint.go#L8) — the character class `[A-Za-z0-9_,]+` stops at whitespace, so `//nolint:AVL001, GLOBAL001` (one space after the comma) captures only `AVL001` and silently suppresses just AVL001 while GLOBAL001 fires.
  <details><summary>details</summary>

  The intent of [`nolint.go:50-54`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/nolint.go#L50-L54) trimming each rule after `Split(",")` is to tolerate spaces around commas — but the regex captures only up to the first whitespace, so the trim loop never sees the trailing entries. Repro:

  ```
  //nolint:AVL001 , GLOBAL001
  var Counter int   // GLOBAL001 still fires
  ```

  Fix: widen the capture class to include spaces — `[A-Za-z0-9_, ]+` (or `[A-Za-z0-9_,\t ]+`) — then the existing TrimSpace loop already handles it. Alternative: accept the whole tail with `(?::(\S.*?))?$` and rely on TrimSpace.
  </details>

- **[silent failure: `--disable-rules="GLOBAL001 "` doesn't disable]** [`gnovm/cmd/gno/lint.go:367-371`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L367-L371) — flag value is split by `,` but each rule is never trimmed, so any whitespace around a rule ID makes it land in the disable map under the wrong key and the rule still fires.
  <details><summary>details</summary>

  ```go
  for _, rule := range strings.Split(cmd.disableRules, ",") {
      lintCfg.Disable[rule] = true
  }
  ```

  `IsRuleEnabled` does `!c.Disable[ruleID]` — a key `" GLOBAL001 "` never matches `"GLOBAL001"`. Repro: `gno lint --disable-rules=" GLOBAL001 " .` still emits GLOBAL001. Fix: trim each rule (`strings.TrimSpace(rule)`) before insertion, and consider warning on unknown rule IDs since unknowns are also silently accepted (`gno lint --disable-rules=TYPO ./...` exits success with no diagnostic).
  </details>

## Warnings (should fix)

- **[redundant sort, already in `Registry.All()`]** [@notJoon](https://github.com/gnolang/gno/pull/5068#discussion_r3090937263) [`gnovm/cmd/gno/lint.go:467-469`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L467-L469) — `listLintRules` re-sorts `ruleInfos` by `ID`, but `Registry.All()` already returns rules sorted by `Info().ID` ([`registry.go:45-47`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/registry.go#L45-L47)). Drop the second sort.

- **[dead field: `Issue.Pos` set but never read]** [@notJoon](https://github.com/gnolang/gno/pull/5068#discussion_r3090943043) [`gnovm/pkg/lint/issue.go:16`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/issue.go#L16) — `Pos` is populated by `NewIssue` and JSON-skipped, but no consumer ever reads it (verified via grep). Remove the field and the parameter shape on `NewIssue` (`Line`/`Column` are already extracted from `pos`).

- **[mode-string parsing in CLI, not in lint package]** [@notJoon](https://github.com/gnolang/gno/pull/5068#discussion_r3090887879) [`gnovm/cmd/gno/lint.go:355-365`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L355-L365) — the `switch` mapping `"default"/"strict"/"warn-only"` → `Mode` belongs next to the `Mode` constants in [`config.go`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/config.go#L5-L9). A `lint.ParseMode(string) (Mode, error)` helper would keep the CLI thin and let third-party callers (gnodev, lsp) reuse the parser.
  <details><summary>details</summary>

  Today the CLI is the only place that knows the string form. If another consumer wants to honor `--mode=strict` (e.g. gnopls), it has to duplicate this switch. Centralizing the string ⇆ enum mapping in the package that owns the enum is the standard pattern.
  </details>

- **[`GLOBAL001` false-positive on `errors.New()` package vars]** [`gnovm/pkg/lint/rules/global001.go:25-62`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/rules/global001.go#L25-L62) — pattern `var ErrFoo = errors.New(...)` is idiomatic Go/Gno (used heavily in `grc1155`, `grc20`, `grc721`); flagging it forces every such package to `//nolint:GLOBAL001` per line or sit with permanent noise. Pre-existing concern raised on this PR (line 51 of docs), the resolution proposed was "use nolint" — fine as a stopgap but the rule lacks an exception for `var X = errors.New(...)` / similar immutable-by-construction patterns.
  <details><summary>details</summary>

  Repro (`examples/gno.land/p/demo/microblog/microblog.gno`):

  ```
  $ gno lint examples/gno.land/p/demo/microblog
  microblog.gno:15:2: warning: exported package-level variable: ErrNotFound (GLOBAL001)
  microblog.gno:16:2: warning: exported package-level variable: StatusNotFound (GLOBAL001)
  ```

  Suggestion: skip vars whose RHS is a single CallExpr to `errors.New` / `fmt.Errorf`, or whose declared type is `error`. Treat the conservative version as a follow-up rule, but at minimum document the limitation in [`gno-lint.md`](../../../../../.worktrees/gno-review-5068/docs/resources/gno-lint.md) under "Available Rules" so users aren't surprised.
  </details>

- **[`reportError` dedup may swallow distinct same-position errors]** [`gnovm/pkg/lint/reporters/base.go:27-32`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/reporters/base.go#L27-L32) — dedup key is `filename:line:column:ruleID`. Two type-check errors at the same `file:line:col` with the same ruleID (`gnoTypeCheckError`) but different messages collapse to one. The existing `file_error.txtar` test happens to have different columns; pathological cases (e.g. an `undefined: X` plus a follow-on error at column 0) will be hidden.
  <details><summary>details</summary>

  Either include the message in the key, or scope dedup only to rule-driven issues (where `(file,line,col,ruleID)` uniquely identifies the finding) and skip dedup for typecheck/parser errors. Practical impact today: low; latent for the future.
  </details>

## Nits

- [`gnovm/pkg/lint/nolint.go:60-62`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/nolint.go#L60-L62) — `IsSuppressed` only checks `line-1`. A multi-line decl whose first issue line is N+2 below the comment won't be suppressed. Document the one-line scope in [`gno-lint.md`](../../../../../.worktrees/gno-review-5068/docs/resources/gno-lint.md#L71-L75) (currently says "on the line above") with explicit "applies only to the next single line; does not cover the rest of a multi-line statement."

- [`gnovm/pkg/lint/rules/avl001.go:9-13`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/rules/avl001.go#L9-L13) — hardcoded `gno.land/p/nt/avl/v0` already has a TODO referencing PR #5048 for version-suffix stripping. Track via an issue so this doesn't decay silently when v1 lands.

- [`gnovm/pkg/lint/rule.go:8-11`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/rule.go#L8-L11) — category constants live in `rule.go` but consumers will keep adding (`CategoryAVL`, `CategoryGeneral`, `CategoryRender`). Consider documenting in [`gno-lint.md`](../../../../../.worktrees/gno-review-5068/docs/resources/gno-lint.md#L106-L137) how to add a new category, or move to a registry pattern.

- [`gnovm/pkg/lint/rules/avl001.go:104-113`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/rules/avl001.go#L104-L113) — `isEmptyStringLiteral` checks `ConstExpr.V.(gnolang.StringValue)` without a type-assert ok-check; if the const value is somehow not a `StringValue` despite `Kind() == StringKind`, the function panics. Defensive `if sv, ok := e.V.(gnolang.StringValue); ok` would be safer.

- [`gnovm/cmd/gno/common.go:111-119`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/common.go#L111-L119) — `reportIssue` builds an `Issue` literal without `Pos`. Consistent with the "Pos is dead" finding above; either populate or remove the field.

## Missing Tests

- **[`//nolint` regex edge cases]** [`gnovm/pkg/lint/nolint_test.go:153-171`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/nolint_test.go#L153-L171) — `TestNolintParser_EdgeCases` (flagged by @notJoon) only checks for non-panic. Add assertions that `//nolintbar`, `//nolint_typo`, `//nolint:A B` are either rejected entirely or behave as documented. Today these silently pass through with surprising semantics (see Critical findings).

- **[`--disable-rules` whitespace and unknown IDs]** [`gnovm/cmd/gno/testdata/lint/`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/testdata/lint/) — no txtar covers `--disable-rules=" RULE "`, `--disable-rules="TYPO001"`, or `--disable-rules=""`. Each currently has a different silent behavior; pin the contract.

- **[AVL001 rule unit tests cover only `hasEmptyStringBounds`]** [@notJoon](https://github.com/gnolang/gno/pull/5068#discussion_r3090900332) [`gnovm/pkg/lint/rules/avl001_test.go`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/rules/avl001_test.go) — no test exercises `Check` against an `*gnolang.CallExpr` with realistic `SelectorExpr`/`DeclaredType` shapes. Currently the only coverage of full-rule behavior lives in the txtar; a unit test on `Check` would catch regressions when the AST helpers shift.

## Suggestions

- [`gnovm/pkg/lint/reporter.go:3-5`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/reporter.go#L3-L5) — the comment already says "consider moving Reporter to a standalone package." Worth doing now since `gno test` and `gno run` already import `lint` solely for `Reporter` ([`run.go:14`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/run.go#L14), [`test.go:19`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/test.go#L19)) — split to `gnovm/pkg/report` would let `gno run` not pull in rule definitions.

- [`gnovm/pkg/lint/engine.go:11`](../../../../../.worktrees/gno-review-5068/gnovm/pkg/lint/engine.go#L11) — TODO "handle verbose mode for linting engine" — tracking issue or remove? Verbose mode at engine level (which rule fired on which node) would help debug rule false-positives.

- [`gnovm/cmd/gno/lint.go:147-154`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L147-L154) — comment notes "Currently the linter only supports linting directories." Pre-existing limitation; not for this PR, but worth tracking as a follow-up since the new framework makes per-file lint plausible.

## Questions for Author

- Does the `cleanup` between `engine.Flush()` and `sortAndReset()` matter for callers that call `Engine.Run()` multiple times across packages? Today `Flush()` calls `sortAndReset()` which clears counts; `Summary()` after `Flush` returns zeros, so the `lintErrors > 0` check at [`lint.go:398`](../../../../../.worktrees/gno-review-5068/gnovm/cmd/gno/lint.go#L398) is computed before `Flush()` — correct, but fragile if the order ever flips.

- For `GLOBAL001`, was an exception for `var X = errors.New(...)` considered before settling on "use //nolint"? Many existing packages will need annotation, and the false-positive rate hurts the rule's signal-to-noise.

- Is there a tracking issue for the breaking output format change (old `(code=X)` → new `severity: msg (X)`)? Tools like gnodev, gnoweb, IDE integrations parse this output; PR body lists it as breaking but doesn't reference a downstream-impact issue.
