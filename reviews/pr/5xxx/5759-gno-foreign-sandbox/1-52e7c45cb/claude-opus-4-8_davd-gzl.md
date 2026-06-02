# PR #5759: feat(gnoweb,boards2): gno-foreign markdown sandbox + rich, paginated comments

URL: https://github.com/gnolang/gno/pull/5759
Author: jaekwon | Base: master | Files: 88 | +3283 -276
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 52e7c45cb (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5759 52e7c45cb`

**Verdict: REQUEST CHANGES** — the feature is well-built and the sandbox holds under adversarial probing, but `TestAppHashCrossrealm38` fails: adding the `chain/markdown.MaxForeignBlocksPerConvert` native shifts the on-chain stdlib Merkle root and the pinned consensus hash was not updated. That is the one blocker; everything else is a nit or a question.

## Summary

Adds a render-time sandbox, `<gno-foreign>`, for untrusted markdown (comments, foreign-realm bodies). gnoweb collects the block body opaquely, renders it in its own isolated goldmark instance under safe mode (no raw HTML, dangerous-URL guard, links marked `rel="noopener nofollow ugc"` with first-party trust icons stripped), enforces a 256-block-per-render cap and a depth-4 cross-family nesting cap, and fails closed (opaque capture + budget marker) past either. A realm-side helper `p/nt/markdown/foreign` builds the wrapper and neutralizes every sentinel-shaped body line so hand-rolled envelopes can't escape; the cap is exposed to realms via a new `chain/markdown` native so renderer and realm agree on one number. boards2 then renders every comment body through the sandbox (dropping its old write-time markdown blacklist), paginates top-level comments (10/page), caps inline replies at 10 with a re-rooted paginated "View all N" view, adds a `?flat=1` all-comments view, carries sort order through every view, and fixes a few collateral bugs (frozen-thread action links, ghost reply on delete, escaped flag-reason/invalid-ID echoes).

The blocker is mechanical: the new native changes stdlib content, so the multistore root hash in `gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go` no longer matches.

## Glossary
- `<gno-foreign>` — render-time sandbox block; body rendered by an isolated inner goldmark instance.
- foreign helper — `p/nt/markdown/foreign`: builds the wrapper, escapes body sentinels, blank-line-frames the opener.
- nestdepth — shared cross-family gno-* nesting counter (cap 4), seeded across the inner/outer boundary.
- block-count cap — monotonic per-Convert `<gno-foreign>` admission limit (256); refused blocks captured opaquely + budget marker.
- untrusted link — link parsed under a foreign-origin context: `rel="...ugc"`, no first-party icons.
- `maxRenderedBodies()` — boards2 per-page foreign-block budget = native cap − 10.
- AppHash test — `TestAppHashCrossrealm38`, a determinism guard pinning the committed multistore Merkle root.

## Fix

Before: realms had no safe way to embed untrusted markdown — escape everything (lose formatting) or trust it (injection/XSS). After: untrusted bytes go through `foreign.Foreign`/`ForeignWithLabel` → `<gno-foreign>` → an isolated inner goldmark render with safe-mode HTML stripping, dangerous-URL neutralization, untrusted-link chrome, and hard caps. The load-bearing constraint is that the body is opaque to every outer block parser (`Continue` returns `parser.Continue | NoChildren` for all body lines, only an unescaped outer `</gno-foreign>` closes it — see [`ext_foreign.go:245-290`](https://github.com/gnolang/gno/blob/52e7c45cb/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L245-L290) · [↗](../../../../../.worktrees/gno-review-5759/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L245-L290)), and the helper escapes every sentinel-shaped body line so attacker bytes can never reach the parser as a tag.

## Critical (must fix)

- **[pinned consensus hash not updated → CI red]** [`gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go:116`](https://github.com/gnolang/gno/blob/52e7c45cb/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L116) · [↗](../../../../../.worktrees/gno-review-5759/gno.land/pkg/sdk/vm/apphash_crossrealm38_test.go#L116) — adding the `chain/markdown.MaxForeignBlocksPerConvert` native changes stdlib content; the determinism guard's pinned root no longer matches.
  <details><summary>details</summary>

  This is the only failing job in `main / test` (the `Merge Requirements` red is just the gnoweb-codeowner approval gate, not a test). The PR adds a native binding ([`generated.go`](https://github.com/gnolang/gno/blob/52e7c45cb/gnovm/stdlibs/generated.go#L699-L716) · [↗](../../../../../.worktrees/gno-review-5759/gnovm/stdlibs/generated.go#L699-L716), [`native_gas.go`](https://github.com/gnolang/gno/blob/52e7c45cb/gnovm/stdlibs/native_gas.go#L145) · [↗](../../../../../.worktrees/gno-review-5759/gnovm/stdlibs/native_gas.go#L145)) plus the `.gno`/`.go` source in `chain/markdown`. That changes the bytes of an on-chain stdlib package, which shifts the iavlStore Merkle root the test pins. The test itself is not touched by the PR and its message spells out the required action: "Verify this is an intentional consensus-breaking change before updating the pinned value." Reproduced locally (see below). Fix: confirm the stdlib change is the intended consensus delta, then update `expectedCrossrealm38Hash` to the observed value `0fbdbf8ff64fd5b851a030229304d95c9196c80a449c47da89ed76ff1f3c0bb4`. Worth a maintainer sanity-check that no OTHER unintended save-set change is folded into the same hash shift.

  **Repro:**
  ```bash
  # from a local clone of gnolang/gno:
  gh pr checkout 5759 -R gnolang/gno
  go test ./gno.land/pkg/sdk/vm/ -run TestAppHashCrossrealm38
  ```
  ```
  --- FAIL: TestAppHashCrossrealm38 (3.82s)
      apphash_crossrealm38_test.go:116:
          expected: "e37075fb6a103445adc4d83ecb95e1bd3ba839a709eac873db2f95d56f9010ac"
          actual  : "0fbdbf8ff64fd5b851a030229304d95c9196c80a449c47da89ed76ff1f3c0bb4"
          Messages: multistore commit hash changed — the save set ... shifted.
  FAIL	github.com/gnolang/gno/gno.land/pkg/sdk/vm	3.857s
  ```
  </details>

## Warnings (should fix)

- **[helper-dependent safety not enforced in code]** [`ext_foreign.go:11-20`](https://github.com/gnolang/gno/blob/52e7c45cb/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L11-L20) · [↗](../../../../../.worktrees/gno-review-5759/gno.land/pkg/gnoweb/markdown/ext_foreign.go#L11-L20) — the sandbox is safe only if realms emit blocks through `foreign.Foreign`; a hand-built envelope with an attribute-bearing close in the body escapes the sandbox.
  <details><summary>details</summary>

  The boundary depends entirely on the helper escaping sentinel-shaped body lines and blank-line-framing the opener. A realm that hand-assembles `<gno-foreign>`…`</gno-foreign>` and concatenates an attacker body containing `</gno-foreign bogus="y">` will have the outer block terminated early by that line — trailing content renders at top level, outside the sandbox. I confirmed this directly (test 3 below): a hand-built attr-close terminates the outer block and `AFTER-ESCAPED` renders outside `gno-foreign__body`. This is by design and documented loudly in the package doc and `foreign.gno`, but nothing in the type system or API prevents a realm from bypassing the helper (the block parser is exported and any string can be written to a render). boards2 uses the helper correctly everywhere ([`format.gno:56-59`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/format.gno#L56-L59) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/format.gno#L56-L59), [`render_post.gno:535`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L535) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L535)). Fix: nothing required for this PR's safety, but consider whether a future hardening (e.g. only honoring `<gno-foreign>` when preceded by an internal sentinel the helper injects) is warranted, or at minimum keep the "MUST use the helper" warning prominent in any realm-facing docs that ship later.
  </details>

- **[ADR missing for an AI-assisted feature]** [`gno.land/adr/`](https://github.com/gnolang/gno/tree/52e7c45cb/gno.land/adr) · [↗](../../../../../.worktrees/gno-review-5759/gno.land/adr) — `gno/AGENTS.md` requires an ADR for every non-trivial AI-assisted PR; this 3.3k-line feature has none, and the commit log references "scratch planning docs" that were deliberately dropped.
  <details><summary>details</summary>

  `AGENTS.md` ("Architecture Decision Records"): "Every non-trivial AI-assisted PR must include an ADR." Commit `de30f92e7 docs: drop references to scratch planning docs from code` confirms design docs existed and were removed rather than distilled into an ADR. The sandbox model (opacity invariant, dual cap design, helper-escaper/parser case-fold lockstep, untrusted-link policy, mention carve-out) is exactly the kind of reasoning the ADR rule exists to preserve for reviewers and future contributors. Fix: add an ADR under `gnovm/adr/` (stdlib native) and/or `gno.land/adr/` (gnoweb sandbox) capturing the threat model, the two caps, and the helper-is-mandatory contract. (Severity is a judgment call — flagged because it's a repo rule, not a style preference.)
  </details>

## Nits

- [`render_post.gno:90-117`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L90-L117) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L90-L117) — a repost charges two foreign-block budget units (its own body via `indentForeignBody` plus the source body via `renderSourcePost`→`indentForeignBody`). `maxRenderedBodies()` = native−10 leaves headroom, but the truncation accounting in `renderTopLevelReplies` counts one comment as one unit; a page full of reposts consumes the budget ~2× faster than the comment count suggests. Harmless (fails safe to a truncation notice) but the budget-vs-comment-count relationship is non-obvious.
- [`public.gno:776-786`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/public.gno#L776-L786) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/public.gno#L776-L786) — `assertReplyBodyIsValid` now only checks empty + length; the markdown/`gno-form` blacklist removal is correct (sandbox contains structure, inner instance omits forms) but the function name now overstates what it validates. Consider renaming or folding into the length check.
- [`render_thread.gno:112-124`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/render_thread.gno#L112-L124) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/render_thread.gno#L112-L124) — `flatIndent` walks parents per reply (up to `maxFlatIndentDepth`=6 `getReply` lookups each) for every row on a flat page (up to `pageSizeFlat`=50). Bounded (≤300 tree lookups/page) so fine today, but it's O(page × depth) avltree gets layered on top of the render budget.

## Missing Tests

- **[no boards2 filetest for the repost double-budget path]** [`render_post.gno:90-117`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L90-L117) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/render_post.gno#L90-L117) — pagination/truncation tests cover plain comment threads; a thread dominated by reposts (two foreign blocks each) isn't exercised against the budget.
  <details><summary>details</summary>

  The 12 new boards2 filetests cover pagination, reply caps, flat view, frozen-thread actions, ghost-delete, and escaped echoes. They don't cover a page of reposts where each post wraps both its own body and the source body in `<gno-foreign>`, which is the case most likely to hit `maxRenderedBodies()` earlier than the visible comment count implies. Low priority — the path fails safe — but it's the untested corner of the budget logic.
  </details>

## Suggestions

- `chain/markdown` native — the value 256 lives in Go (`maxForeignBlocksPerConvert`) and is read by both gnoweb and realms, which is the right single-source-of-truth design. Worth a one-line note in any realm-facing docs that this is a per-*render* cap (monotonic, never decremented), not a per-page or per-thread cap, so realm authors size their own budgets correctly (boards2 does this at [`render.gno:41-43`](https://github.com/gnolang/gno/blob/52e7c45cb/examples/gno.land/r/gnoland/boards2/v1/render.gno#L41-L43) · [↗](../../../../../.worktrees/gno-review-5759/examples/gno.land/r/gnoland/boards2/v1/render.gno#L41-L43)).

## Questions for Author

- `data:image/svg+xml` links: goldmark's `IsDangerousURL` whitelists svg data URIs, but I confirmed the inner safe-mode strips the inline `<svg>` raw HTML so the markdown link doesn't even form (test 2 below shows `[x](data:image/svg+xml,<!-- raw HTML omitted -->)` as literal text). Is the reliance on that raw-HTML-stripping side effect intentional, or should the sandbox link transformer explicitly reject `data:` schemes regardless of goldmark's carve-out? Today it's safe; the safety is incidental to how the svg payload is written, not a deliberate `data:` block.
- Was an ADR intentionally omitted, or is one planned as a follow-up? (See warning above.)
- The CLAUDE.md change ([`CLAUDE.md:7`](https://github.com/gnolang/gno/blob/52e7c45cb/CLAUDE.md#L7) · [↗](../../../../../.worktrees/gno-review-5759/CLAUDE.md#L7), `-run txtar` → `-run TestTestdata`) is unrelated to the feature — intentional drive-by, or stray?

---

Adversarial probes run against HEAD (saved under `tests/foreign_adversarial_test.go`): (1) a `javascript:` link nested inside an inner `<gno-columns>` block is neutralized to `href=""`; (2) a `data:image/svg+xml` link does not survive (inner safe-mode strips the svg payload, link doesn't form); (3) a hand-built attribute-bearing close in the body DOES terminate the outer block — confirming the helper is load-bearing, not optional. The sandbox boundary held in every probe; the existing 18 gnoweb tests + 12 boards2 filetests are unusually thorough (sentinel escaping incl. `</gno-foreign/>`, case-folding, attr-bearing forms, cap fail-closed under `WithUnsafe`, mention carve-out, depth cap at 5). gnoweb markdown, boards2, and foreign-helper test suites all pass locally.
