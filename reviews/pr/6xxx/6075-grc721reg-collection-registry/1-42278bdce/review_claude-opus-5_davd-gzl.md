# gnolang/gno [#6075](https://github.com/gnolang/gno/pull/6075): feat(grc721reg): add NFT collection registry at r/demo/defi/grc721reg (4/4)

URL: https://github.com/gnolang/gno/pull/6075
Author: jinoosss | Base: master | Files: 47 | +4382 -1606, of which 4 files and +612 are this PR's own work over [#6074](https://github.com/gnolang/gno/pull/6074)
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh, deep) | Commit: 42278bdce (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6075 42278bdce`
Overview: [visual overview](../overview.html)

Open the code: [github.dev](https://github.dev/jinoosss/gno/tree/42278bdce/examples/gno.land/r/demo/defi/grc721reg) · [vscode.dev](https://vscode.dev/github/jinoosss/gno/tree/42278bdce/examples/gno.land/r/demo/defi/grc721reg) · `./scripts/review-worktrees.sh gno 6075`

## Overview

The last PR of the GRC721 series adds a realm where any other realm lists its NFT collection so wallets, marketplaces and other realms can find it without importing it. It is the counterpart of the fungible-token registry `r/demo/grc20reg` and follows that file closely: an AVL tree keyed by the registering realm's path, a `Register` that checks the collection was issued by the caller, a handful of getters, and two rendered pages. What it does differently is the value it stores. `grc20reg` stores a concrete `*grc20.Token`; this stores a `*collection.Collection` from [#6074](https://github.com/gnolang/gno/pull/6074), which carries the caller's extension read views as interface values so the registry never imports a concrete extension. The package doc states that trade and pushes the type check onto the consumer.

**Verdict: REQUEST CHANGES** — the openness the design is built on reaches the rendered page: an extension kind is a string the registering realm chooses, and it lands in `Render` unescaped, so **one registration puts a heading and a link of the registrant's choosing on the index page every visitor reads**, permanently, since nothing removes an entry. The rest follow the same theme, a caller-controlled value the page presents as the registry's own (1 Critical, 4 Warnings, 4 Nits, 1 Suggestion).

## Verify first

- [`grc721reg.gno:152`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L152) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L152) — run [`render_defacement_filetest.gno`](tests/render_defacement_filetest.gno) and read the index page it prints: a heading and a link the registering realm wrote, inside the registry's own list.
- [`grc721reg.gno:42`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L42) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L42) — compare against [`grc20reg.gno:41`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L41), whose key is the realm plus the symbol and whose comment calls that an overwrite and alias guard. This key is the realm plus a caller-chosen slug, so the guard is gone: [`alias_filetest.gno`](tests/alias_filetest.gno) lists one collection twice.
- [`grc721reg.gno:47`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) — `Register` builds its own `Collection` rather than storing the caller's, so an extension attached afterwards never reaches the registry: [`stale_extensions_filetest.gno`](tests/stale_extensions_filetest.gno).

## Summary

`Register` takes the caller's realm path from `cur.Previous().PkgPath()`, checks the token id starts with it, refuses a taken key, and stores a fresh `Collection` built from the token and the views passed in. Every getter reads that tree; `Render` walks it for the index and reads one entry for the detail page. The realm surface is small and the ownership check is the right shape, so the defects are all about what a caller controls after that check passes: the kind strings it advertises, how many keys it takes for one collection, and the fact that the stored entry is a copy taken at one instant with no way to refresh it.

## Critical (must fix)

- **[caller text the page presents as its own]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:152` — an extension kind is whatever the registering realm's view returns, and `extensionBadges` wraps it in backticks without escaping, so one registration writes a heading and a link into the index page every visitor reads, with no removal call to take it down.
  <details><summary>details</summary>

  [`extensionBadges`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L142-L156) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L142) builds `"` + k + "` "` while the collection name and symbol beside it go through [`md.EscapeText`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L116) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L116). The three in-tree kinds are constants, but the package doc's premise is that the registry accepts kinds it does not import, so the string is caller-chosen by design. Measured at this head with [`render_defacement_filetest.gno`](tests/render_defacement_filetest.gno): a kind carrying a newline and a link renders as a second-level heading and a working link inside the list, and the same text lands on the collection's own page. The entry cannot be removed, since there is no removal call and no owner. Fix: `md.InlineCode` is already in the imported package and sizes the fence to contain internal backticks while folding newlines, so it replaces the hand-built span in one line; a charset check on the kind at `Register`, matching `validateSlug`, closes the rest.
  </details>

## Warnings (should fix)

- **[a page that stops rendering]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:109` — the index walks every entry, and registration is open, so the home page has a hard ceiling: measured at 54.8M gas for 100 entries and 165.4M for 300, near 553,000 gas per entry against a query ceiling of 3,000,000,000.
  <details><summary>details</summary>

  [`registry.Iterate`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L109) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L109) is unbounded, and the sibling carries `// TODO: add pagination` on the same line. Measured with paired filetests that differ only in whether `main` renders: 54,792,437 gas at 100 entries and 165,405,897 at 300, so the slope is near 553,000 per entry and the page passes the [query ceiling](https://github.com/gnolang/gno/blob/master/gno.land/pkg/sdk/vm/keeper.go#L52) at roughly 5,400 entries. Registration costs near 559,000 gas per entry, and the Warning below means one collection can take as many entries as it likes, so filling it is one loop rather than one collection per entry. Nothing removes an entry once written. Fix: page the index off the render path.
  </details>
- **[a test that never leaves the registry realm]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:114` — `testing.SetRealm` inside a `t.Run` closure does not reach the `cur` the subtest closes over, so all three cases of the file's largest test register from the registry realm itself and the `realmPath` they carry is dead.
  <details><summary>details</summary>

  The realm value the case passes to `Register` is [`TestRegisterAndLookup`'s own `cur`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L113-L117) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L113), fixed before the closure runs. Measured at this head with the same two shapes side by side: a registration inside the closure keys under `gno.land/r/demo/defi/grc721reg.inside`, while the identical call outside it keys under `gno.land/r/demo/zprobe_outside.outside`. `TestRegisterAborts` escapes this by accident, since its case function takes `cur realm` itself, which is why the cross-realm rejection is genuinely covered while the happy path is not. Fix: hoist the `SetRealm` and the build above `t.Run`, and assert the key equals the case's own realm path plus its slug, in place of the `NotEqual(t, "", key)` that passes for any string.
  </details>
- **[the guard the sibling documents, dropped]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:42` — the key is the realm path plus a caller-chosen slug, so one collection can hold as many entries as it likes, each advertising a different extension set, and no entry is canonical.
  <details><summary>details</summary>

  [`grc20reg`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L41) keys on the realm plus the token symbol and says in the line above that this is what stops a realm registering two tokens under one symbol, calling it an overwrite and alias guard; its test suite pins it with `TestRegisterRejectsAliasedTokenPaths`. Here the slug is the key, so `registry.Has(key)` never sees the second registration of the same collection. Measured with [`alias_filetest.gno`](tests/alias_filetest.gno): one collection, keys `.first` and `.second`, two lines on the index. Two consequences follow. The extension views are supplied per registration, so a marketplace resolving the first key and one resolving the second read different royalty policy for the same NFT. And the key's suffix is where the sibling puts the token's own symbol, so a reader treats it as identity: measured at this head, a collection whose symbol is `SCAM` renders as `gno.land/r/demo/zdeface.BAYC` on both pages. Fix: reject a token id that is already registered, whatever the slug, or key on the symbol as the sibling does.
  </details>
- **[a copy taken once, with no way to refresh it]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:47` — `Register` builds its own `Collection` from the views it was passed, so an extension the issuer attaches afterwards is invisible to every consumer, and a second `Register` is refused.
  <details><summary>details</summary>

  [#6074](https://github.com/gnolang/gno/pull/6074)'s `NFT.Collection()` returns the live object the issuer keeps attaching to, and its doc says a later `Attach` mutates it too. That is not what the registry holds: [line 47](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L47) calls `collection.NewCollection` and stores that. Measured with [`stale_extensions_filetest.gno`](tests/stale_extensions_filetest.gno): after the issuer attaches royalty, its own handle lists one kind and the registry still lists zero, and the collection's page still reads `_core only_`. The realm cannot correct it, since the key is taken and there is no update or removal call. Fix: store the caller's `*collection.Collection` rather than building a new one, or add a call that replaces the entry for its own realm.
  </details>

## Nits

- **[a page that faults instead of answering]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:128` — `Render` on an unknown key goes through `MustGet` and panics, so a reader who mistypes a key gets an error page rather than a message. Measured: `Render("gno.land/r/nobody/here")` aborts with `grc721reg: unknown collection`. Not posted: [`grc20reg`](https://github.com/gnolang/gno/blob/master/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L130) does the same, so this is the sibling's shape rather than this branch's choice.
- **[a kind with no ceiling]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:57` — nothing bounds the kind's length or charset; the only limit is the 4096-byte cap `chain.Emit` puts on an attribute, and hitting it aborts `Register` with a message naming the event rather than the registry. An empty kind renders as a bare pair of backticks.
- **[two names for one field]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:52` — the key goes out as `token_key` where the sibling emits `token_path`, though [`consts.gno:3`](https://github.com/jinoosss/gno/blob/refactor/grc721-registry/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L3) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/consts.gno#L3) says the event value mirrors grc20reg for cross-registry consistency. An indexer subscribed to the shared `register` type reads one field under two names.
- **[an event field that cannot be split]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:57` — the `extensions` attribute is `strings.Join(col.Kinds(), ",")`, and a kind may contain a comma, so an indexer splitting on it reads one kind as two. Same caller-controlled string as the first Warning, and the same fix closes it.

## Suggestions

- **[a lookup a consumer cannot derive]** `examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:42` — `grc20reg` keys on the symbol, so a consumer that knows the realm and the symbol can build the key; here the slug is free text, so the key has to be learned out of band. Where the alias Warning above is fixed by keying on the symbol, this goes with it.

## Verified

- The ownership check holds: the token id prefix must match `cur.Previous().PkgPath()`, so a realm cannot register another realm's collection, and a direct user call has an empty path and is refused.
- Registration and every getter are deterministic: the store is an `avl.Tree`, `Kinds()` iterates it in order, and no map, clock or randomness appears in the diff.
- `GetRegistry` hands back `rotree.Wrap`, so a consumer walking the tree cannot write to it.
- `validateSlug` rejects every character outside letters, digits, underscore and hyphen, including the dot that would make a key ambiguous, and caps the length at 128.

## Open questions

- Each registration bills the registry realm about 3.6 KB of storage, measured from the filetest summaries, and there is no removal call, so the deposit is locked for as long as the chain lives. `grc20reg` has the same property, so it is a question for the pair rather than for this branch.
