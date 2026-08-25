# PR [#5871](https://github.com/gnolang/gno/pull/5871): feat(examples/r/docs): revive and correct r/docs

URL: https://github.com/gnolang/gno/pull/5871
Author: gfanton | Base: master | Files: 117 | +1261 -1290
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 7c41a297d (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-5871 7c41a297d`
Overview: [visual overview](../overview.html)

## Overview

Thirty documentation realms sat in `examples/quarantined/`, where they compile and run their tests but are not part of the package set a chain deploys. This change moves them back under `examples/gno.land/r/docs/` with `git mv`, so the pages a newcomer reads first are live again, and corrects them on the way. Two corrections carry most of the diff. The runtime mints the first `cur realm` of a crossing function per frame, so an `IsCurrent()` check on it can never fail, and every such check is deleted from the revived realms and rewritten in `AGENTS.md` and three files under `docs/resources/`. Four statements about how gnoweb renders markdown were wrong and are replaced with what the extensions do. Three realms are folded into siblings, four are dropped, and one case in gnoweb's own route test moves off a realm this change removes.

**Verdict: REQUEST CHANGES** — the caller-identity rewrite says what the ADR and `zrealm_iscurrent.gno` say and the `ownable` demo stops teaching a stack-walk auth bug, but the branch takes `r/docs/routing` live with a sub-page that answers "Error: internal error", and picks the wrong sanitize helper for a code span on a line it adds (2 Warnings, 2 Missing tests, 4 Nits, 1 Suggestion).

## Verify first

- [`gnovm/adr/interrealm_v2.md:336-339`](https://github.com/gnolang/gno/blob/7c41a297d/gnovm/adr/interrealm_v2.md?plain=1#L336-L339) · [↗](../../../../../.worktrees/gno-review-5871/gnovm/adr/interrealm_v2.md#L336-L339) — confirm the premise the whole sweep rests on, that the runtime keeps the first `cur` current. Run `go test -run 'TestFiles/zrealm_iscurrent.gno$' ./gnovm/pkg/gnolang/`: the filetest prints `cur.IsCurrent(): true` and `cur.Previous().IsCurrent(): false`, and passing `cur` through a helper keeps the first answer.
- [`examples/gno.land/r/docs/crossing/crossing.gno:35-43`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/crossing/crossing.gno#L35-L43) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/crossing/crossing.gno#L35-L43) — confirm this is the one check worth keeping in the tree. The realm value arrives in second position from a caller who could send `cur.Previous()` instead, which is the case the deleted checks were not.
- [`examples/gno.land/r/docs/soliditypatterns/banker/banker.gno:23`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L23) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L23) — confirm the payment guard is the one you want taught. `IsUserCall()` admits only a direct `maketx call`, where `IsUser()` would also admit a `maketx run` script that has already consumed the send envelope.

## Summary

The deletions and the doc rewrites are the same claim in four documents and twelve realm functions: the token a crossing frame receives is minted by the runtime, so it is the live one by construction, and only a realm value the caller chose is worth testing. The claim holds, stated in [the ADR](https://github.com/gnolang/gno/blob/7c41a297d/gnovm/adr/interrealm_v2.md?plain=1#L336-L339) · [↗](../../../../../.worktrees/gno-review-5871/gnovm/adr/interrealm_v2.md#L336-L339) and asserted by [`zrealm_iscurrent.gno`](https://github.com/gnolang/gno/blob/7c41a297d/gnovm/tests/files/zrealm_iscurrent.gno#L11-L34) · [↗](../../../../../.worktrees/gno-review-5871/gnovm/tests/files/zrealm_iscurrent.gno#L11-L34). Alongside it the branch fixes two real defects: [`ownable`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/soliditypatterns/ownable/demo.gno#L20) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/ownable/demo.gno#L20) stops authenticating through `unsafe.PreviousRealm()` in a non-crossing helper, which named a realm above the immediate caller, and [MiniSocial v2's edit window](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v2/posts.gno#L86) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/minisocial/v2/posts.gno#L86) now compares the block clock against the post instead of the post against itself.

The four corrected gnoweb statements were each checked against the extension that implements them: [the textarea row clamp](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/markdown/ext_forms.go#L29-L30) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/markdown/ext_forms.go#L29-L30), [`parseSelect`](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/markdown/ext_forms.go#L401-L424) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/markdown/ext_forms.go#L401-L424), [`parseAlertType`](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/markdown/ext_alert.go#L124-L139) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/markdown/ext_alert.go#L124-L139), and [the image validator](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/markdown/ext_imgvalidator.go#L35-L36) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/markdown/ext_imgvalidator.go#L35-L36) paired with [`allowSvgDataImage`](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/render_config.go#L38-L40) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/render_config.go#L38-L40). The host list the page prints matches [`cspImgHost`](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/cmd/gnoweb/main.go#L23-L43) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/cmd/gnoweb/main.go#L23-L43) entry for entry, and the new mention rules match [the two regexes](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28-L29) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/markdown/ext_mentions.go#L28-L29).

Reading order: `AGENTS.md` and the three files under `docs/resources/`, then `crossing`, `soliditypatterns/ownable`, `soliditypatterns/banker`, then `markdown`, then `home` and the realms it lists, then the sixteen new test files, then `gno.land/pkg/gnoweb/app_test.go`.

## Examples

| Realm value | Where it comes from | `IsCurrent()` |
|---|---|---|
| the first `cur` of a crossing function | minted by the runtime for this frame | always true |
| `cur` handed on to a helper in the same realm | the same token, passed by value | true, identity survives the pass |
| a secondary `rlm realm` parameter | whatever the caller supplied | the check that matters |
| `cur.Previous()` | derived, a different token | false |

| The markdown page said | What the renderer does |
|---|---|
| textarea `rows` spans 4 to 10 | clamps to 2 to 10, default 4 |
| `<gno-select>` takes `text=` for the option label | no `text` attribute exists; the option label is the value |
| five alert types | six, matched case-insensitively, anything else falling back to INFO |
| a short list of providers is supported for images | the renderer erases every non-SVG `data:` URI first, then the browser applies the host allowlist |

## Fix

`InitOwner` and `ChangeOwner` in the `ownable` demo now derive the owner inside the crossing function, and the realm carries a warning saying why the natural `msg.sender` port is not equivalent. MiniSocial v2 gains a 1000-byte cap on post text, an empty-text guard on `UpdatePost`, and the corrected window comparison. Post bodies in both MiniSocial versions run through [`sanitize.Block`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v1/types.gno#L22) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/minisocial/v1/types.gno#L22) before they reach the feed, and [`GetRealmBalance`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L102) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/soliditypatterns/banker/banker.gno#L102) reads its single denomination with `GetCoin` rather than walking every denomination a third party can add.

## Warnings (should fix)

- **[a live sub-page cannot render]** `examples/gno.land/r/docs/routing/routing.gno:29` — `/r/docs/routing:wildcard`, the parent of the URL this realm's own index links, answers "Error: internal error" instead of the wildcard handler.
  <details><summary>details</summary>

  [`mux.Router.Render`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/nt/mux/v0/router.gno#L44) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/nt/mux/v0/router.gno#L44) walks the pattern's segments and reads `reqParts[i]` for each one, and the wildcard case only breaks out of the loop after that read. A one-segment request against the two-segment `wildcard/*` pattern therefore indexes past the end: `runtime error: slice index out of bounds: 1 (len=1)`. The same happens with master's `wildcard/*/` spelling, so the change on this line is not the cause, but the realm was quarantined and is now the only live realm registering a wildcard route, which is what puts the URL in front of a reader. Driving it through gnoweb's own `TestRoutes` gives `unable to fetch realm ... RPC node response error: runtime error: slice index out of bounds: 1 (len=1)`, and the page body reads "Error: internal error. Something went wrong." Repro in [`tests/wildcard_bare_path_filetest.gno`](tests/wildcard_bare_path_filetest.gno), which fails at this head and passes once the bare pattern is registered. Fix: register `router.HandleFunc("wildcard", wildcardHandler)` ahead of the wildcard pattern, or guard the index in `mux`.
  </details>

- **[the sanitize sweep picks the wrong helper for a code span]** `examples/gno.land/r/docs/complexargs/complexargs.gno:74` — a backtick in the value closes the code span early, and the stray one left over pairs with the next backtick further along the sentence, so caller-controlled text rewrites the shape of the line.
  <details><summary>details</summary>

  `InlineText` escapes a backtick to `` \` ``, and a backslash is literal inside a code span, so the escape neither hides the backtick nor renders as one. The sanitize package's own [helper table](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L53) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L53) names `InlineCode` for this slot, whose fence is one backtick longer than the longest run in the content. This line is new in the branch, where the merge base wrote `myObject.Name` raw, and `SetMyObject` is documented as unguarded so any account can set the value through `MsgRun`. The same pairing predates the branch at [`registry.gno:149`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/registry/registry.gno#L149) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/registry/registry.gno#L149) and [`userprofile.gno:123`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/userprofile/userprofile.gno#L123) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/userprofile/userprofile.gno#L123), where the value is the URL path itself, and the sweep walked past both. Neither line is inside a diff hunk, so GitHub offers no anchor for them and one comment carries all three. Through the repo's own sanitize golden harness, a backtick in the value moves both span boundaries onto the wrong words, on the anchored line and on registry's sentence alike:

  ```html
  <!-- complexargs.gno:74, InlineText then InlineCode -->
  <p>Value of myObject: <code>CustomType{Name: x\</code>y, Numbers: 1}`</p>
  <p>Value of myObject: <code>CustomType{Name: x`y, Numbers: 1}</code></p>

  <!-- registry.gno:149, tests/inline-code-span-breakout.txtar then -contained.txtar -->
  <p>No service with address <code>g1\</code>evil<code>and name</code>svc` is registered.</p>
  <p>No service with address <code>g1`evil</code> and name <code>svc</code> is registered.</p>
  ```

  The realm-level strings match: the branch emits `` `CustomType{Name: x\`y, Numbers: 1}` `` and the replacement emits a two-backtick fence around the same text. Fix: use `sanitize.InlineCode` and drop the hand-written backticks at all three sites.
  </details>

## Missing Tests

- **[the repaired edit window is the one fix nothing asserts]** `examples/gno.land/r/docs/minisocial/v2/posts.gno:85-86` — put master's comparison back and the branch's own suite stays green, so the fix can be undone by a merge with nothing to notice.
  <details><summary>details</summary>

  Master compared `post.updatedAt.After(post.createdAt.Add(time.Minute * 10))`, and a post that was never edited carries `updatedAt == createdAt`, so it passed the check however old it was. The branch adds `TestUpdatePostRejectsNonAuthor` and `TestDeletePostRejectsNonAuthor` beside this change, and neither advances the clock. [`tests/minisocial_v2_update_window_test.gno`](tests/minisocial_v2_update_window_test.gno) advances 240 heights, twenty minutes of block time at the five seconds a height `SkipHeights` uses, and asserts `ErrUpdateWindowExpired`; it passes at this head and fails on the reverted comparison with `error mismatch, expected update window expired, got %!s(<nil>)`. Fix: add it to `minisocial/v2`.
  </details>

- **[the two realms that move coins are the untested ones]** `examples/gno.land/r/docs/soliditypatterns/banker/banker.gno:23` — nothing asserts that the `IsUserCall()` guard added to `Deposit` and `Withdraw` refuses a realm caller.
  <details><summary>details</summary>

  The branch adds tests to `soliditypatterns/counter`, `ownable` and `statelock`, and leaves `banker` and `reentrancy` with no `_test.gno` at all, so the guard that is the whole lesson of the banker page is the one thing a reader cannot see asserted. The distinction the page teaches is the narrow one: `IsUser()` would also admit a `maketx run` script that consumed the send envelope before calling in, and only a test that calls `Deposit` from a realm frame shows which of the two is in force. Fix: add a test where a non-user caller is refused.
  </details>

## Nits

- **[one test duplicates a stronger neighbour]** `examples/gno.land/r/docs/minisocial/v1/posts_test.gno:36-45` — `TestResetPostsRejectsNonOwner` asserts what [`admin_test.gno`'s `TestResetPostsUnauthorized`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/minisocial/v1/admin_test.gno#L13-L28) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/minisocial/v1/admin_test.gno#L13-L28) already asserts and adds a survival check on top, and `resetPostsForTest` exists for this one call while its comment promises an isolation the file's other two tests do not take.
- **[the guard is weaker than its five siblings]** `examples/gno.land/r/docs/home/home_test.gno:31` — the closing paren in `"/r/docs/avl_pager)"` lets a link written with a sub-path, `/r/docs/avl_pager:2`, walk through the assertion.
- **[a live package points at a realm this change deletes]** `examples/gno.land/p/samcrew/piechart/README.md:40` — the README links `/r/docs/charts:piechart`, and `charts` is removed here rather than revived. The stated reason, packages absent from `examples/`, holds for the gauge half, which imports `p/samcrew/gauge`; the piechart half imports [`p/samcrew/piechart`](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/p/samcrew/piechart/piechart.gno#L1) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/p/samcrew/piechart/piechart.gno#L1), which is in the tree.
- **[one twin mirrors the sanitizer and the other does not]** `examples/gno.land/r/docs/minisocial/v2/posts_test.gno:67` — v1's equivalent assertion was updated to `sanitize.Block(p.text)` alongside the same change to `Post.String()`, and v2's still reads `p.text`. Both pass today because `Block` leaves these strings alone, so this is consistency rather than a defect, and it is not posted.

## Suggestions

- **[the docs realm teaches a check its own docs now call dead code]** `examples/gno.land/r/docs/security_patterns/security_patterns.gno:61` — `assertAdmin` panics unless `cur.IsCurrent()` holds and [the page advertises that as defence one](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L37-L39) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/security_patterns/security_patterns.gno#L37-L39), while [`gno-ai-contract-review.md`](https://github.com/gnolang/gno/blob/7c41a297d/docs/resources/gno-ai-contract-review.md?plain=1#L24-L27) · [↗](../../../../../.worktrees/gno-review-5871/docs/resources/gno-ai-contract-review.md#L24-L27) now calls that check dead code.
  <details><summary>details</summary>

  `assertAdmin` is unexported and every call site inside the realm passes the frame's own `cur`, so the check is the same always-true one the sweep deleted twelve times elsewhere. The realm landed on master separately and is not in this diff, but it sits in the tree this change curates and [the new index promotes it](https://github.com/gnolang/gno/blob/7c41a297d/examples/gno.land/r/docs/home/home.gno#L49) · [↗](../../../../../.worktrees/gno-review-5871/examples/gno.land/r/docs/home/home.gno#L49) as the security page, so a reader who follows the learning path meets the rewritten rule and its counterexample in the same session. Fix: delete the check and the Render bullet describing it, in this change or the next one.
  </details>

## Verified

- `/r/docs/routing:wildcard` returns an error page while `/r/docs/routing:wildcard/foo` renders: two cases added to `TestRoutes` in [`gno.land/pkg/gnoweb/app_test.go:107`](https://github.com/gnolang/gno/blob/7c41a297d/gno.land/pkg/gnoweb/app_test.go#L107) · [↗](../../../../../.worktrees/gno-review-5871/gno.land/pkg/gnoweb/app_test.go#L107), the first failing with `unable to fetch realm`, the second passing.
- A backtick in a caller-controlled value escapes its code span under `InlineText` and stays inside it under `InlineCode`: [`tests/inline-code-span-breakout.txtar`](tests/inline-code-span-breakout.txtar) and [`tests/inline-code-span-contained.txtar`](tests/inline-code-span-contained.txtar), both run green through `TestSanitizeIntegration`, each carrying the HTML the renderer produced.
- The MiniSocial v2 window fix is load-bearing: reverting the comparison to master's and running [`tests/minisocial_v2_update_window_test.gno`](tests/minisocial_v2_update_window_test.gno) gives `error mismatch, expected update window expired, got %!s(<nil>)`.
- The moved `TestRoutes` case still exercises the file-listing fallback: `/r/demo/disperse` declares no `Render`, and the case asserts the listing rather than a rendered page.
- Green at 7c41a297d with the go1.25.9 the tree pins: `gno test ./gno.land/r/docs/...` over 30 packages, `gno lint ./gno.land/r/docs/...`, and `go test ./gno.land/pkg/gnoweb/...`.

## Open questions

- Storing the `mux.Router` in a realm global moves the route table and its handler values into persisted state, where before it was rebuilt per query. The filetest reports `storage: gno.land/r/docs/routing:+11542b` either way, so nothing measurable changed, and the realm now teaches a shape whose cost lands at deploy time rather than per call. Not posted: no number separates the two.
- `Deposit` credits only the ugnot in the send envelope, and any other denomination in the same envelope reaches the realm's address with no record and no way out. Not posted: the page is about ugnot throughout and says so.
