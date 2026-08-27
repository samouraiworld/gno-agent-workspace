# gnolang/gno [#6074](https://github.com/gnolang/gno/pull/6074): feat(grc721): add collection package with read-only Collection and NFT handle (3/4)

URL: https://github.com/gnolang/gno/pull/6074
Author: jinoosss | Base: master | Files: 11 | +562 -26
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh) | Commit: 710cdabae (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6074 710cdabae`
Overview: [visual overview](../overview.html)

Open the code: [github.dev](https://github.dev/jinoosss/gno/tree/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection) · [vscode.dev](https://vscode.dev/github/jinoosss/gno/tree/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection) · `./scripts/review-worktrees.sh gno 6074`

Round 3, a recheck of the round 2 findings after five commits. Every one of the eight comments posted on the branch is closed, including the return type davd-gzl raised at [r3869492724](https://github.com/gnolang/gno/pull/6074#discussion_r3869492724). `gno test ./gno.land/p/demo/tokens/grc721 ./gno.land/p/demo/tokens/grc721/collection` is green at this head.

## Overview

The fix that closed the copied-registry Warning changed what `Collection` is. `ExtensionKinds` and `ExtensionView` moved from `PrivateLedger` to [`Token`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/token.gno#L137-L154), the tree and `sync` are gone, and [`Collection`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L11-L13) is now a single `token` field whose four methods read through it.

**Verdict: COMMENT** — nothing left from round 2, and one thing the fixes created: the read model is now a wrapper with nothing in it (1 Suggestion).

## Closed since 6350eccaf

| Posted | Finding | Closed by |
| --- | --- | --- |
| [r3869322204](https://github.com/gnolang/gno/pull/6074#discussion_r3869322204) | `RegisterExtension` files a view without asking its kind or its token | [`token.gno:190-198`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/token.gno#L190-L198), both comparisons, both panics |
| [r3869322215](https://github.com/gnolang/gno/pull/6074#discussion_r3869322215) | `Collection` copies the ledger's views instead of reading them | the tree and `sync` are gone; [`Extension`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L34) reads through the token |
| [r3869322228](https://github.com/gnolang/gno/pull/6074#discussion_r3869322228) | `NewCollection` checks the pointer, not the pair | [`collection.gno:23-25`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L23-L25) panics on a tokenless ledger |
| [r3869322235](https://github.com/gnolang/gno/pull/6074#discussion_r3869322235) | no test pins a mismatched view | [`TestRegisterExtensionRejectsAForeignView`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/token_test.gno#L751) asserts both messages |
| [r3869322242](https://github.com/gnolang/gno/pull/6074#discussion_r3869322242) | the sorted-kinds assertion cannot fail | royalty registers first at [`collection_test.gno:190`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L190-L199), and the token's order is asserted against the collection's |
| [r3869322255](https://github.com/gnolang/gno/pull/6074#discussion_r3869322255) | two `HasExtension`, two answers | `PrivateLedger.HasExtension` deleted |
| [r3869322260](https://github.com/gnolang/gno/pull/6074#discussion_r3869322260) | a key derived on every read | `kind` is stored on `registeredExtension` and every lookup keys off it |
| [r3869492724](https://github.com/gnolang/gno/pull/6074#discussion_r3869492724) | the comma-ok return | `Collection.Extension`, `NFT.Extension` and `Token.ExtensionView` all return a bare view |

## Suggestions

- **[a wrapper with nothing left in it]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:11` — `Collection` is one `*grc721.Token` field and four methods, three of which forward a single call to that token, so the read model the registry stores could be the token itself.
  <details><summary>details</summary>

  Measured at this head: [`Token()`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L30) returns the field, [`Extension`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L34) is `token.ExtensionView(kind)` under another name, and [`HasExtension`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L38) is the same call compared against nil. [`Kinds`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L43-L54) is the only behaviour the type adds, and it is the sort and the nil-view filter over `token.ExtensionKinds()`. `*grc721.Token` is already the read-only handle: its twelve methods are all reads, it publishes no ledger, and [#6075](https://github.com/gnolang/gno/pull/6075) reaches for `Kinds`, `Extension` and `Token` only, each of which the token answers or would with `Kinds` moved beside `ExtensionKinds` in the core. What the type still buys is a name for the published half and a place to add read behaviour later; what it costs today is a package and an indirection on the registry's storage type.
  </details>

## Verified

- `gno test ./gno.land/p/demo/tokens/grc721 ./gno.land/p/demo/tokens/grc721/collection` green at 710cdabae, and green at 6350eccaf before it.
- `Token` exposes no mutator: `GetName`, `GetSymbol`, `ID`, `TotalSupply`, `KnownAccounts`, `BalanceOf`, `OwnerOf`, `GetApproved`, `ExtensionKinds`, `ExtensionView`, `IsApprovedForAll`, `RenderHome`.
- A published `*Collection` sees a later registration, now because it holds no copy rather than because a handle refreshed it: [`collection_test.gno:216-224`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L216-L224).

## Open questions

- [`NFT.Collection()`](https://github.com/jinoosss/gno/blob/710cdabae/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L48) dereferences `n` with no `n == nil` guard. Not posted: measured at 6350eccaf, a nil `*NFT` aborts with `runtime error: nil pointer dereference` from `Collection()`, `Token()` and `Burn()` alike, and `&collection.NFT{}` aborts identically, so a guard on one method changes one abort message and leaves the rest. Neither constructor can hand back a nil.
- `Kinds` calls `ExtensionView` once per kind and each call scans the extension slice, so the listing is quadratic in the number of extensions. Not posted: the count is the number of extensions on one token, and the fix belongs in the core rather than here.
- `New` and `Wrap` build `&Collection{token: token}` directly rather than through `NewCollection`, so the constructor's two panics guard nothing on the paths that actually build collections. Not posted: `New` takes its token from `NewToken` and `Wrap` checks the pair itself, so neither path can reach the states `NewCollection` rejects.
