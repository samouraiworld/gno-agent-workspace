# PR #4494: feat(txtar): txtar file options & formating

URL: https://github.com/gnolang/gno/pull/4494
Author: gfanton | Base: master | Files: 60 | +782 -351
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 99dca9441 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-4494 99dca9441`

**TL;DR:** Adds a `setopts` directive line to the top of an integration `.txtar` test file so each file can set its own options (skip formatting, run sequentially, skip the file, custom timeout), and adds a `go test -ts-fmt` mode that auto-formats the `.gno` source embedded inside `.txtar` archives. Most of the 60-file diff is that auto-formatter run applied to existing test files.

**Verdict: REQUEST CHANGES** — `doc.go` still documents the old `# txtar:opts` comment syntax that maintainers asked be replaced; the code now uses the `setopts` command but the package doc, the PR description, and one in-code comment never followed. No correctness issue in the harness; the txtar reformatting is behavior-preserving.

## Summary
The harness gains per-file test options. The original design used a magic comment (`# txtar:opts <flags>`); moul and thehowl asked for a real command instead, and gfanton agreed and reimplemented it as a `setopts <flags>` line parsed before the test runs ([testdata_test.go:84-89](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L84-L89) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L84)). Four flags exist: `-no-fmt`, `-no-parallel`, `-skip`, `-timeout`. `-no-parallel` now skips `t.Parallel()` natively instead of the previous `sequentialMu` mutex, which this PR removes. A separate `-ts-fmt` flag runs a gno formatter over the `.gno` blocks inside each archive; the bulk of the diff is that formatter applied across the testdata corpus. The package doc was never updated to the agreed syntax: it still teaches `# txtar:opts`, which does nothing.

## Examples
| File top | Effect |
|---|---|
| `setopts -no-parallel` | file's subtest does not call `t.Parallel()` |
| `setopts -no-fmt` | `-ts-fmt` leaves this file's `.gno` blocks untouched |
| `setopts -skip` | file's subtest is `t.Skip`ped |
| `setopts -timeout 90s` | file-level node context deadline is 90s |
| `# txtar:opts -no-fmt` | nothing — treated as a plain comment, silently ignored |

## Glossary
- txtar: testscript-based integration tests under `gno.land/pkg/integration/testdata/`.

## Critical (must fix)
None.

## Warnings (should fix)
- **[doc teaches a syntax the code dropped]** `gno.land/pkg/integration/doc.go:12-30` — package doc still documents `# txtar:opts <flags>`; the implementation uses a `setopts` command and ignores that comment.
  <details><summary>details</summary>

  The "Txtar Test File Options" section ([doc.go:12-30](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/doc.go#L12-L30) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/doc.go#L12)) tells a reader to set options with `# txtar:opts <flags>` at the top of the file. The parser keys off the `setopts` prefix and treats any line starting with `#` as a skippable comment ([testdata_test.go:214-220](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L214-L220) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L214)), so a `# txtar:opts -skip` line does nothing and raises no error. This is the exact syntax moul and thehowl asked be replaced ([moul](https://github.com/gnolang/gno/pull/4494#issuecomment-3057024895), [thehowl](https://github.com/gnolang/gno/pull/4494#issuecomment-3175223015)) and that gfanton agreed to change. The three in-tree fixtures already use the new `setopts` form; only the doc lags. Confirmed behaviorally: a `# txtar:opts` line returns empty args while `setopts` returns the parsed flags (see Missing Tests repro). Fix: rewrite the doc section to `setopts <flags>`, and add `setopts` to the "Additional Command Overview" list.
  </details>

## Nits
- `gno.land/pkg/integration/testdata_test.go:206` — comment names a non-existent function and the old `#`-prefixed syntax. It reads `// ParseTopLevelFlags parses top-level lines starting with # <prefix> <flags>.` above `func captureTopLevelLineArgs`; the function was renamed and no longer keys off `#`. [testdata_test.go:206](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L206) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L206)
- `gno.land/pkg/integration/testdata_test.go:219` — typo in the break comment: `// setopts as to be the top level commands` (should be "has to be"). [testdata_test.go:219](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L219) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L219)

## Missing Tests
- **[the unused options can rot unnoticed]** `gno.land/pkg/integration/testdata/` — `-skip` and `-timeout` have zero in-tree coverage; only `-no-fmt` and `-no-parallel` are exercised by any fixture.
  <details><summary>details</summary>

  Across the whole testdata corpus only `gc.txtar` (`-no-parallel`), `addpkg_invalid.txtar` and `err_metadata.txtar` (`-no-fmt`) carry a `setopts` line. `-skip` (wired at [testdata_test.go:155-157](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L155-L157) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L155)) and `-timeout` (wired at [testscript_gnoland.go:175](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testscript_gnoland.go#L175) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testscript_gnoland.go#L175)) never run, so a regression in either would pass CI. The flag-parsing layer also has no direct unit test: `captureTopLevelLineArgs` and `ParseDirFlags` are only reached transitively. A table test over `captureTopLevelLineArgs` would pin the parse contract, including the silent-ignore behavior in the Suggestion below. Repro of the current parse behavior:

  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 4494 -R gnolang/gno
  cat > gno.land/pkg/integration/zz_probe_test.go <<'EOF'
  package integration

  import (
  	"strings"
  	"testing"
  )

  func TestProbeSetopts(t *testing.T) {
  	for _, tc := range []struct{ name, body string }{
  		{"old-comment-syntax", "# txtar:opts -skip\ngnoland start\n"},
  		{"setopts-after-cmd", "gnoland start\nsetopts -skip\n"},
  		{"setopts-first", "setopts -no-parallel\ngnoland start\n"},
  	} {
  		got, err := captureTopLevelLineArgs(strings.NewReader(tc.body), "setopts")
  		t.Logf("%-20s -> %#v err=%v", tc.name, got, err)
  	}
  }
  EOF
  go test ./gno.land/pkg/integration/ -run TestProbeSetopts -v 2>&1 | grep '\->'
  rm gno.land/pkg/integration/zz_probe_test.go
  ```

  ```
  old-comment-syntax   -> []string{} err=<nil>
  setopts-after-cmd    -> []string{} err=<nil>
  setopts-first        -> []string{"-no-parallel"} err=<nil>
  ```
  </details>

## Suggestions
- `gno.land/pkg/integration/testdata_test.go:207-232` — a `setopts` line placed after any non-comment line is silently dropped; consider erroring or warning instead.
  <details><summary>details</summary>

  `captureTopLevelLineArgs` accumulates `setopts` args only until the first line that is neither a comment nor a `setopts` line, then `break`s ([testdata_test.go:218-220](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L218-L220) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/testdata_test.go#L218)). A `setopts` line below a real command (e.g. after `gnoland start`) is then ignored with no error, so a misplaced `setopts -skip` quietly does nothing. The repro above shows `setopts-after-cmd -> []`. The position requirement is sensible; the silent failure is the footgun. A directive line found below the top block could fail the parse instead.
  </details>
- `gno.land/pkg/integration/utils.go:78` — `splitArgs` and the existing `unquote` are two near-duplicate quote-aware splitters; a comment cross-referencing why both exist would help.
  <details><summary>details</summary>

  `splitArgs` ([utils.go:78](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/utils.go#L78) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/utils.go#L78)) parses `setopts` lines; `unquote` ([utils.go:12](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/utils.go#L12) · [↗](../../../../../.worktrees/gno-review-4494/gno.land/pkg/integration/utils.go#L12)) parses `gnokey`/`patchpkg` args. They differ in single-quote and escape handling, so a merge is non-trivial and probably not worth it, but the two sitting side by side with no note invites a future reader to assume one is dead. A one-line comment on each pointing at its call site would settle it.
  </details>
- `gnovm/pkg/gnofmt/package.go:66-68` — `memPackage.Read` dereferences the result of `GetFile` without a nil check.
  <details><summary>details</summary>

  `m.MemPackage.GetFile(filename)` returns `*MemFile`, nilable, and the next line reads `f.Body` ([package.go:66-68](https://github.com/gnolang/gno/blob/99dca9441/gnovm/pkg/gnofmt/package.go#L66-L68) · [↗](../../../../../.worktrees/gno-review-4494/gnovm/pkg/gnofmt/package.go#L66)). In the formatter path `Files()` and `Read()` are always called in lockstep over the same MemPackage (`ReadWalkPackage`, [package.go:29-44](https://github.com/gnolang/gno/blob/99dca9441/gnovm/pkg/gnofmt/package.go#L29-L44) · [↗](../../../../../.worktrees/gno-review-4494/gnovm/pkg/gnofmt/package.go#L29)), so the name always resolves and this can't panic today. It's a latent nil-deref for any other caller of the new exported `NewPackage` adapter. Confirmed by reading the signature: `GetFile` returns `*MemFile` ([tm2/pkg/std/memfile.go:173](https://github.com/gnolang/gno/blob/99dca9441/tm2/pkg/std/memfile.go#L173) · [↗](../../../../../.worktrees/gno-review-4494/tm2/pkg/std/memfile.go#L173)).
  </details>

## Open questions
- A malformed `setopts` flag in any single `.txtar` fails the entire `TestTestdata` suite, not just that file's subtest, because `ParseDirFlags` runs over every file up front and the caller does `require.NoError` ([testdata_test.go:88-89](https://github.com/gnolang/gno/blob/99dca9441/gno.land/pkg/integration/testdata_test.go#L88-L89)). Fail-fast on a bad directive is defensible, so not posted; noting in case per-file isolation is preferred later.
- The PR also reformats the testdata corpus via `-ts-fmt`. A parallel pass confirmed every `.txtar` change is formatting-only (import alphabetization, indentation, trailing-newline, plus the expected `setopts` lines); the two non-obvious cases (`gno "0.9"` → `gno = "0.9"`, neutralized because the loader regenerates `gnomod.toml`; and fixed gas assertions, which sit only on packages the formatter never touched) were verified safe. Not a finding, recorded for the next reviewer.

## Verification

Verified on 99dca9441, beyond what CI shows:
- The agreed `setopts` command works end-to-end while the documented `# txtar:opts` form is a no-op: ran `gc.txtar` (logs "parallel testing is disable for this test", so `-no-parallel` took effect) and confirmed `captureTopLevelLineArgs` returns `[]` for a `# txtar:opts` line but the parsed flags for a `setopts` line.
- The corpus reformatting is behavior-preserving: a whitespace-normalized multiset diff of all 51 changed `.txtar` files reduces to import-ordering, indentation, EOF-newline, and `setopts`/directive lines only; no command, assertion value, gas number, address, or flag changed. Representative reformatted files (`interrealm_mix_call`, `storage_deposit`, `params_sysparams2`, `grc20_registry`, `infinite_loop`) pass.
- An unknown `setopts` flag is rejected (parse error surfaces as a test failure), so a typo'd flag fails loudly; only a misplaced (post-command) `setopts` line fails silently.
