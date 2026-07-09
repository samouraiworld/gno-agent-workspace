# PR [#5911](https://github.com/gnolang/gno/pull/5911): fix(grc20reg): enforce maximum slug length

URL: https://github.com/gnolang/gno/pull/5911
Author: notJoon | Base: master | Files: 2 | +15 -3
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 6c20a2f7b (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5911 6c20a2f7b`

**TL;DR:** The `grc20reg` realm lets any realm register a token under a short name (a "slug"). Nothing limited how long that name could be, so a caller could register with a very long slug. This PR caps the slug at 128 bytes.

**Verdict: APPROVE** — minimal, correct, well-tested cap that matches the linked issue [#5909](https://github.com/gnolang/gno/issues/5909); one optional doc nit.

## Summary
`grc20reg.Register` builds the registry key as `pkgpath + "." + slug` ([`fqname.Construct`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/p/nt/fqname/v0/fqname.gno#L51-L57) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/p/nt/fqname/v0/fqname.gno#L51)) and stores it in the AVL registry; the same key drives the rendered gnoweb path and the `register` event's `slug` attribute. Nothing bounded the slug, so the key, the rendered path, and the event attribute could grow arbitrarily. The VM already caps each event attribute at 4096 bytes ([`MaxEventAttrLen`](https://github.com/gnolang/gno/blob/6c20a2f7b/gnovm/stdlibs/chain/emit_event.go#L37) · [↗](../../../../../.worktrees/gno-review-5911/gnovm/stdlibs/chain/emit_event.go#L37)), so the event was never truly unbounded, but the registry key and the rendered path were. The fix adds `maxSlugLen = 128` and panics in `validateSlug` when `len(slug)` exceeds it, keeping the slug at ~3% of the event-attribute cap and bounding the stored key.

## Fix
`validateSlug` now checks `len(slug) > maxSlugLen` before its character loop and panics `grc20reg: slug too long` ([`grc20reg.gno:97-106`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L97-L106) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L97)). `Register` calls `validateSlug` at the top, before `registry.Set` and `chain.Emit` ([`grc20reg.gno:23-35`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L23-L35) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L23)), so an over-long slug aborts before any state write or event. Validation runs only for non-empty slugs and only on `Register`, so existing entries stay readable through `Get`/`Render`; this matches the compatibility note in the issue.

## Critical (must fix)
None.

## Warnings (should fix)
None.

## Nits
None.

## Missing Tests
None. Coverage is complete: [`grc20reg_test.gno:58`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L58) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L58) accepts a slug of exactly `maxSlugLen`, and [`TestValidateSlugPanicsOnTooLong`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64-L68) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64) rejects `maxSlugLen+1`, so the boundary is pinned on both sides.

## Suggestions
- [`grc20reg.gno:88`](https://github.com/gnolang/gno/blob/6c20a2f7b/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L88) · [↗](../../../../../.worktrees/gno-review-5911/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L88) — name the unit on the constant.
  <details><summary>details</summary>

  `maxSlugLen = 128` has no unit comment, and the limit is applied as `len(slug)`, which is bytes. The character loop that follows admits only single-byte ASCII, so an accepted slug always has byte length equal to rune count, but the length check runs first, so the unit that actually gates the panic is bytes. The issue's own proposal wrote `const maxSlugLen = 128 // bytes`; restoring that comment removes the ambiguity. Fix: add `// bytes`.
  </details>

## Verified
- Revert-proof: removing the `len(slug) > maxSlugLen` panic makes `TestValidateSlugPanicsOnTooLong` fail with `should have panicked`, so the new test genuinely guards the check, not a vacuous pass.
- All 10 package tests pass at 6c20a2f7b, including the 128-byte-accepted and 129-byte-rejected boundary cases (`go run ./gnovm/cmd/gno test -v .` from the package dir with `GNOROOT` set to the worktree).

## Open questions
- The registry key also embeds the unbounded caller `pkgpath`, so bounding the slug alone does not fully bound the key or rendered path. Out of scope here: realm paths are constrained elsewhere and the 4096-byte event-attribute cap is the protocol backstop. Not posted.
</content>
</invoke>
