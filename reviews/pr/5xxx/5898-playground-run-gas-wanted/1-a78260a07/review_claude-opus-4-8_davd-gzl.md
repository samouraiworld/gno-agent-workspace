# PR [#5898](https://github.com/gnolang/gno/pull/5898): chore(gnoweb): Playground Run: Increase default gas wanted

URL: https://github.com/gnolang/gno/pull/5898
Author: jefft0 | Base: playground2 | Files: 3 | +3 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: a78260a07 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5898 a78260a07`

**TL;DR:** The gnoweb Playground Run page pre-fills a "Gas wanted" box whose value is dropped into the copy-paste `gnokey maketx run` command. This bumps that default from `2000000` to `1_000_000_000` so pasted transactions stop running out of gas, matching what the Actions screen already emits.

**Verdict: APPROVE** — trivial default-value change, aligned with the existing Actions-screen default; the emitted underscore literal parses in gnokey and 1B stays under the block gas cap.

## Summary
The Run page's `gasWanted` input defaulted to `2000000`, which is too low for many realm calls, so pasted commands failed with out-of-gas. This raises the default to `1_000_000_000` (1B) in the three places that carry it: the HTML input value, the TypeScript source that builds the command, and the compiled JS bundle. The value matches the Actions screen and the form-command golden files, which already emit `-gas-wanted 1_000_000_000`.

## Fix
Three identical literal swaps: the `value=` attribute on the `run-gas-wanted` input at [`page.html:46`](https://github.com/gnolang/gno/blob/a78260a07/gno.land/pkg/gnoweb/feature/run/templates/page.html?plain=1#L46) · [↗](../../../../../.worktrees/gno-review-5898/gno.land/pkg/gnoweb/feature/run/templates/page.html#L46), the fallback in `_buildCmd` at [`controller-run.ts:70`](https://github.com/gnolang/gno/blob/a78260a07/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L70) · [↗](../../../../../.worktrees/gno-review-5898/gno.land/pkg/gnoweb/feature/run/frontend/controller-run.ts#L70), and the same fallback in the built bundle at [`controller-run.js:20`](https://github.com/gnolang/gno/blob/a78260a07/gno.land/pkg/gnoweb/public/js/controller-run.js#L20) · [↗](../../../../../.worktrees/gno-review-5898/gno.land/pkg/gnoweb/public/js/controller-run.js#L20). The bundle edit matches the source edit, so the served asset stays in sync.

## Verification
`gnokey maketx run` binds `-gas-wanted` with the standard library `flag.Int64Var` at [`maketx.go:76-81`](https://github.com/gnolang/gno/blob/a78260a07/tm2/pkg/crypto/keys/client/maketx.go#L76-L81) · [↗](../../../../../.worktrees/gno-review-5898/tm2/pkg/crypto/keys/client/maketx.go#L76-L81), whose `Set` calls `strconv.ParseInt(s, 0, 64)`. Base 0 permits Go integer-literal underscores, so the pasted `-gas-wanted 1_000_000_000` parses to `1000000000` rather than erroring. Confirmed by running an `Int64Var` parse of that exact string: `err <nil> val 1000000000`. The value sits under the block gas ceiling: default consensus `MaxBlockMaxGas` is 3B at [`params.go:28`](https://github.com/gnolang/gno/blob/a78260a07/tm2/pkg/bft/types/params.go#L28) · [↗](../../../../../.worktrees/gno-review-5898/tm2/pkg/bft/types/params.go#L28) and the in-memory dev node uses 30B at [`node_inmemory.go:59`](https://github.com/gnolang/gno/blob/a78260a07/gno.land/pkg/gnoland/node_inmemory.go#L59) · [↗](../../../../../.worktrees/gno-review-5898/gno.land/pkg/gnoland/node_inmemory.go#L59).

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. UI default-value change; the emitted-command shape is already covered by the `ext_forms` golden txtar files, which pin `-gas-wanted 1_000_000_000`.

## Suggestions
None.

## Open questions
- Base is `playground2`, not `master`. Almost certainly deliberate stacking onto the gnoweb playground feature branch; not a code concern, so not posted.
