# gnolang/gno [#6075](https://github.com/gnolang/gno/pull/6075): feat(grc721reg): add NFT collection registry at r/demo/defi/grc721reg (4/4)

URL: https://github.com/gnolang/gno/pull/6075
Author: jinoosss | Base: master | Files: 8 | +1240 -0
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh, deep) | Commit: 953cd82e5 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6075 953cd82e5`
Overview: [visual overview](../overview.html)

Open the code: [github.dev](https://github.dev/gnolang/gno/tree/953cd82e5/examples/gno.land/r/demo/defi/grc721reg) · [vscode.dev](https://vscode.dev/github/gnolang/gno/tree/953cd82e5/examples/gno.land/r/demo/defi/grc721reg)

Round 2, deep. The head moved from 42278bdce to 953cd82e5, whose tip is a merge of master that resolved no conflicts. All seven round 1 findings were answered on the branch: the key is the token's own symbol, the entry is the caller's live collection, the index pages, `Unregister` exists, the event mirrors grc20reg, and an extension kind reaches the page through `md.InlineCode`. The findings below are on the code that replaced them, and none of them repeats a round 1 finding.

## Overview

A realm where any other realm lists its NFT collection so wallets and marketplaces find it by name rather than by import, the counterpart of [`grc20reg`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32) for fungible tokens. An entry is filed under the listing realm's path and its token's symbol, neither of which the caller supplies: the path comes from the live crossing frame and the symbol from the token, and one prefix check on the token id covers both. What is stored is the caller's live `*collection.Collection`, so a page reads whatever the collection holds when it is asked for, and the listing realm can take its own entry back down.

**Verdict: REQUEST CHANGES** — the round 1 fixes all hold and two of them are measurably right, and the surface they left is who may list and what a listing costs everyone else: an ephemeral `gnokey maketx run` realm satisfies the only origin gate, and one page of entries grown after listing puts the index above the query gas ceiling (1 Critical, 3 Warnings, 1 Nit, 1 missing test, 1 refactor, 3 suggestions).

## Verify first

- [`grc721reg.gno:57`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L57) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L57) — run [`run_realm_gate.txtar`](tests/run_realm_gate.txtar) and read the rendered index: a `maketx run` script is listed, and the realm the row links to does not exist.
- [`grc721reg.gno:237`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L237) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L237) — the two `TestZFillRender` figures in [`render_kind_growth_test.gno`](tests/render_kind_growth_test.gno), each minus its `TestZFillOnly` twin, against the 3,000,000,000 ceiling.
- [`grc721reg_test.gno:57`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L57) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L57) — build this branch with [#6074](https://github.com/gnolang/gno/pull/6074)'s `collection` package in place and read which file the errors land in.

## Summary

`Register` is the whole admission surface and `extensionBadges` is the whole cost surface, and each is missing one check the other's fix does not supply. `Register` asks whether the token's id starts with the key, which proves the collection was minted in the frame that is listing it and says nothing about whether that frame is a deployed realm. `extensionBadges` bounds what it prints and not what it reads, so the ceiling the constants promise is a ceiling on bytes rather than on gas. Read [`grc721reg.gno:44-59`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L44-L59) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L44) first, then [`grc721reg.gno:236-260`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L236-L260) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L236).

## Critical (must fix)

- **[the gate proves the frame, not the realm]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:57` — a `gnokey maketx run` script mints a token and lists it in the same frame, so any funded address puts a row of its own choosing on the public index, pointing at a realm that does not exist.
  <details><summary>details</summary>

  The token id prefix check is the only origin gate, and the ephemeral `gno.land/e/<addr>/run` realm satisfies it by construction: the token is minted in that frame, so its `origRealm` is that path and the key is built from the same value. Measured with [`run_realm_gate.txtar`](tests/run_realm_gate.txtar), three transactions against a running node: the run succeeds, the register event carries `"pkgpath":"gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run"`, `vm/qrender` on the index returns the row `- **Definitely Not A Scam** - [gno.land/e/g1jg8…/run](/e/g1jg8…/run).GNOT - _core only_`, and `vm/qfile` on that path answers `package "gno.land/e/…/run" is not available`. The entry is also frozen: the `PrivateLedger` died with the ephemeral package, so nobody can ever mint into it, and only another `maketx run` from the same address reaches `Unregister`. This is the [invariant catalog](../../../../../skills/invariant-catalog.md)'s `realm-only-gate` row, and the predicate it names is `IsUser()`, not `IsUserCall()`, which is false for a run realm. Fix: reject `cur.Previous().IsUser()` at the top of `Register` and `Unregister`.
  </details>

## Warnings (should fix)

- **[a ceiling on bytes standing in for a ceiling on gas]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:237` — `v.Kinds()` walks and materialises the whole extension set before the loop that stops at [`maxBadges`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L14) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L14), and the stored collection is live, so one page of entries grown after listing costs more gas than a query may spend.
  <details><summary>details</summary>

  [`maxEventKinds`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L20) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L20) is checked at listing time only, and [`TestRegisteredCollectionStaysLive`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L491) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L491) is the branch's own proof that the set keeps growing afterwards. Measured at this head with [`render_kind_growth_test.gno`](tests/render_kind_growth_test.gno), one full page of 20 entries, each render figure minus its fill-only twin: 8,858,374 gas with plain entries and 3,536,584,412 with each grown to 2,000 kinds, for 3,350 bytes of page either way. [`maxGasQuery`](https://github.com/gnolang/gno/blob/953cd82e5/gno.land/pkg/sdk/vm/keeper.go#L52) · [↗](../../../../../.worktrees/gno-review-6075/gno.land/pkg/sdk/vm/keeper.go#L52) is 3,000,000,000, so the index stops being servable, and only each listing realm can remove its own row. The registry cannot reach `Collection.exts`, so the bound belongs on the `/p/` type this stack owns. Fix: give `*collection.Collection` a kinds accessor that stops after a limit, and call it with `maxBadges+1`.
  </details>
- **[a page past the last one renders nothing at all]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:185` — the empty-registry gate reads `page.TotalItems`, which counts the whole tree, so a page number past the end falls through it and the item loop writes nothing.
  <details><summary>details</summary>

  [`GetPageWithSize`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L77-L81) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/nt/avl/v0/pager/pager.gno#L77) returns no items with `TotalItems` intact when the requested number is past the last page, and the fallback above only fires when `GetPageByPath` errors, which a large page number does not. Measured at this head: `Render("?page=999")` on a one-entry registry renders zero rows and an empty body. Where the registry has more than one page the body is a picker built from the requested number, so it offers `[997] | [998] | _999_`. The realm that already solved this in tree gates on the other value: [`impl/render.gno:77`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/gov/dao/v3/impl/render.gno#L77) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/gov/dao/v3/impl/render.gno#L77) reads `len(page.Items)` after the same fallback. Fix: keep the `TotalItems` gate for an empty registry and add a `len(page.Items)` gate saying the page is past the end.
  </details>
- **[twenty call sites written against a constructor the next PR replaces]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:57` — this branch carries an older copy of the `collection` package than [#6074](https://github.com/gnolang/gno/pull/6074)'s head, and under that head the registry compiles clean while its test file does not.
  <details><summary>details</summary>

  Measured by building the merged state in a scratch worktree, #6074's whole `grc721` tree at 6350eccaf over this branch: zero errors in `grc721reg.gno` and twenty in `grc721reg_test.gno`, nineteen on `collection.NewCollection`, which #6074 narrows to a single `*PrivateLedger` argument, and one on `nft.Attach`, which #6074 removes. A twentieth `NewCollection` site passes `nil` and still type-checks under the narrower signature. The behaviour the tests assert survives: written in the new flow, `collection.New` then `metadata.NewMetadata(led)` then `nft.Collection()`, the registry entry still reports the later extension, so the port is mechanical. Fix: port the twenty sites when #6074 lands, and keep the live-entry assertion, which is the one that would go quiet if the port took a snapshot instead.
  </details>

## Nits

- **[an entry the registry refused]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:65` — `registry.Set` runs above the two kind bounds, so a registration `Register` aborts stays in the tree wherever the abort is caught, which is this PR's own suite: measured at this head, after `urequire.AbortsContains` catches `extension kind too long`, `Get` still resolves the key and the index still renders the row. On chain the abort takes the transaction with it, so the ordering costs nothing there. Fix: move the `Set` down to sit beside `chain.Emit`.

## Missing Tests

- **[the authority a lookup does not grant]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:138` — nothing asserts that finding a collection in the registry grants no authority over its tokens, which is the property the sibling registry spends two tests on.
  <details><summary>details</summary>

  `GetToken` hands back a `*grc721.Token` carrying [`RealmTeller`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L51) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/tellers.gno#L51), which mints a teller bound to the calling realm's own address. That is the right behaviour and it is what makes the registry safe to read from another realm, and 538 lines of tests never construct one. [`grc20reg_test.gno:64`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L64) pins exactly this for fungible tokens. Fix: add the test that approves a consumer realm for one token, moves it through a teller taken from `GetToken`, and asserts the unapproved one does not move.
  </details>

## Refactor

- **[a field set six times and read never]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:220` — `abortCase.substr` is filled in every case and read in none, since each `run` hardcodes its own substring. Dropping it is 538 lines to 531, suite green.

## Suggestions

- **[three lines that never change an outcome]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:40` — `validateSlug("")` already passes, since a zero length clears the bound and the range loop has nothing to walk, and the branch's own `TestValidateSlug` proves it. Dropping the guard is 276 lines to 274, suite green.
- **[two names used once each]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:193` — `fqname.Parse`'s two results feed nothing but `fqname.RenderLink`, at both call sites, so `RenderLink(fqname.Parse(key))` carries it. 274 lines to 272, suite green, and `Render` output byte-identical across the empty path and seven page queries.
- **[the key is a prefix match where equality is available]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:57` — `key + "."` being a prefix of the token id lets a token minted under a sub-identity list under an ancestor sub-identity's key, since a subpath may contain a dot. Both belong to the same host realm, so no realm reaches another's namespace; the id's last dot is always the sequence boundary, so comparing everything before it against the key is exact.

## Verified

- The index's cost no longer tracks the number of entries, which was round 1's finding. Page one measured flat in registry size, 9,374,904 gas at 25 entries against 10,034,786 at 300, and flat in the requested offset, since the tree walk skips whole subtrees by size.
- A visitor cannot widen a page: the page-size query parameter is left unset, so `Render("?size=1000")` and `Render("")` return the same bytes.
- An extension kind cannot break out of its code span. `sanitize.InlineCode` [strips bidi](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1430-1444) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/nt/markdown/sanitize/v0/sanitize.gno#L1430) and zero-width characters, folds newlines and separators to a space, and sizes the fence to outscan any run of backticks in the content.
- A realm cannot name another realm's key, in either direction: both `Register` and `Unregister` build the key from `cur.Previous().PkgPath()` behind an `IsCurrent()` check, and `Unregister`'s unvalidated symbol only ever composes a key under the caller's own path.
- Nothing published grants a write. `rotree.ReadOnlyTree` [panics on `Set` and `Remove`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/p/nt/avl/v0/rotree/rotree.gno#L167) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/nt/avl/v0/rotree/rotree.gno#L167), `*collection.Collection` has no exported mutator, and the tellers `Token` hands out are readonly or bound to the caller's own address.
- Guard for guard against `grc20reg`, this registry carries everything the sibling has and adds five it does not: the `IsCurrent()` check before [`cur.Previous()`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39), pagination, the render bounds, `Unregister`, and a not-found message where the sibling's `Render` aborts on a bad URL. What the sibling has and this does not is the test above.
- The tip merge of master resolved no conflicts, and every check run at this head is green.

## Open questions

- [`grc721reg.gno:44-47`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L44-L47) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L44) describes a stale realm value reaching `Register`, and no caller shape produces one: the preprocessor refuses anything but `cur` or `cross(rlm)` in that position. Keep the guard, which gno's own `AGENTS.md` mandates in every crossing function. Not posted: a finding about a comment's own wording stays here.
- `fqname.Parse` splits at the first dot after the last slash, so a key whose subpath carries a dot renders the wrong realm and the wrong symbol, and a `#` in a subpath makes the link a fragment on the host page. Not posted: it needs a subpath a realm has to opt into, and the refactor above touches the same two lines.
- `slug` is validated and emitted, never keyed and never rendered. Not posted: round 1 asked for it to leave the key, and the author kept it as an event field on purpose, mirroring the sibling.
- The weakest assertion in the suite is the page-two check at [`grc721reg_test.gno:533`](https://github.com/gnolang/gno/blob/953cd82e5/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L533) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L533), which counts more than zero rows where the remainder is sixteen. Not posted: it is a real assertion, just a loose one.
