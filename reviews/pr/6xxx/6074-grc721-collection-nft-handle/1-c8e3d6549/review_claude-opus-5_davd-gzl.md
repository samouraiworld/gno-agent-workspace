# gnolang/gno [#6074](https://github.com/gnolang/gno/pull/6074): feat(grc721): add collection package with read-only Collection and NFT handle (3/4)

URL: https://github.com/gnolang/gno/pull/6074
Author: jinoosss | Base: master | Files: 43 | +3770 -1606, of which 4 files and +396 are this PR's own work over [#6073](https://github.com/gnolang/gno/pull/6073)
Reviewed by: davd-gzl | Model: claude-opus-5 (xhigh, deep) | Commit: c8e3d6549 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6074 c8e3d6549`
Overview: [visual overview](../overview.html)

Open the code: [github.dev](https://github.dev/jinoosss/gno/tree/c8e3d6549/examples/gno.land/p/demo/tokens/grc721/collection) · [vscode.dev](https://vscode.dev/github/jinoosss/gno/tree/c8e3d6549/examples/gno.land/p/demo/tokens/grc721/collection) · `./scripts/review-worktrees.sh gno 6074`

## Overview

A realm running an NFT collection after [#6072](https://github.com/gnolang/gno/pull/6072) and [#6073](https://github.com/gnolang/gno/pull/6073) holds a token, a private ledger, and a read view plus a ledger for every extension it attached. This PR bundles them into two handles: a `*Collection`, which is the token's read side plus a tree of extension read views and has no write method, and an `*NFT`, which is that collection plus the ledger plus `Mint` and `Burn`. The realm keeps the `*NFT` unexported and publishes the `*Collection`, which is what the registry in [#6075](https://github.com/gnolang/gno/pull/6075) stores. 130 lines of code, 264 of tests.

**Verdict: REQUEST CHANGES** — the split itself is right, and the three defects are all in the checks around it: `Wrap` pairs any token with any ledger, the attach guard compares a token id two collections can share, and a `*Collection` arriving from elsewhere proves nothing about what it carries, which is the property the next PR's registry rests on (3 Warnings, 2 Nits, 2 Suggestions, 1 missing test).

## Verify first

- [`nft.gno:26`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26) — run [`collection_handle_probes_test.gno`](tests/collection_handle_probes_test.gno) and read the first log line: an NFT wrapped around a mismatched pair mints into one collection while publishing the other.
- [`collection.gno:35`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L35) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L35) — the second log line of the same file prints two independent ledgers carrying one token id, which is what the attach guard compares.
- [`collection.gno:8`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L8) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L8) — the third builds a lookalike collection around the genuine token, advertising a royalty view of its own.

## Summary

`NewCollection` and `attach` do the whole job: reject a nil token, skip a nil view, reject a view whose `TokenID()` does not match, reject a kind already present. `NFT` adds construction, the ledger accessor, `Attach`, and mint and burn passthroughs. Every defect below is a case those four checks let through, and each one matters more in [#6075](https://github.com/gnolang/gno/pull/6075) than here, since that PR takes these values across a realm boundary and hands them to strangers. Read [`collection.gno`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L28-L45) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L28) first, then [`nft.gno`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L20-L32) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L20).

## Warnings (should fix)

- **[a pair nothing checks is a pair]** `examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:26` — `Wrap` rejects nils and nothing else, so a transposed argument produces an NFT that mints into one collection and publishes another, with no error at either call.
  <details><summary>details</summary>

  [`Wrap`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26-L32) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L26) never compares `ledger.ReadToken()` against `token`, though that accessor is public and free. Measured at this head with [`collection_handle_probes_test.gno`](tests/collection_handle_probes_test.gno): after one mint through the wrapped handle, the published token reports supply 0 while the ledger's own token reports 1, and the published collection answers `invalid token id` for the token it just minted. A realm hits this by ordering two constructor results wrongly, and nothing tells it until a reader complains. Fix: `if ledger.ReadToken() != token { panic(...) }` beside the nil check.
  </details>
- **[a guard comparing a string that is not an identity]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:35` — `attach` accepts any view whose `TokenID()` string matches, and two collections from one realm can carry the same id, so a view wired to another ledger passes a check whose panic message claims it cannot.
  <details><summary>details</summary>

  A token id is [`pkgPath + "." + symbol + "." + id`](https://github.com/gnolang/gno/blob/master/examples/gno.land/p/demo/tokens/grc721/token.gno#L41) and nothing enforces uniqueness; the core's own `NewToken` doc describes the duplicate case and tells indexers to detect it. Measured at this head: two tokens built in one realm with the same symbol and the same sequence id share `…collection.FOO.0000007`, and attaching the second collection's royalty view to the first is accepted, after which the first collection's royalty answers come from the second's storage while its own mints never reach that extension's hooks. The existing test at [`collection_test.gno:105`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L105) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno#L105) varies both the symbol and the id, which is why the guard looks stronger than it is. `ExtensionView` carries no handle for a stronger check, so either it grows one or the message stops claiming ownership. Fix: say the check is on the advertised id, or add the identity the message promises.
  </details>
- **[a handle that vouches for nothing]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:8` — the doc calls a `*Collection` safe to hand across realms and says nothing about receiving one: `NewCollection` is exported and `Token()` returns the genuine token, so anyone holding a published collection can hand on a lookalike carrying real token and forged views.
  <details><summary>details</summary>

  `grc721.ExtensionView` is two exported methods, so any package satisfies it, and `attach` admits it once the id matches. Measured at this head: a collection built from `published.Token()` and a hand-written royalty view is indistinguishable by token pointer, id and symbol, answers `HasExtension("royalty")` true, and is separated from the real one only by the concrete type assert. The type assert is the right advice and [line 49](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L49-L50) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/collection.gno#L49) gives it, but the comma-ok form a reader naturally writes turns a forged royalty into no royalty rather than an abort. This is the invariant catalog's canonical-assert shape, and the core package beside this one already answers it for another type with [`IsCanonicalTeller`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/types.gno#L121) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/types.gno#L121). Fix: say on the type that a received collection is only as trustworthy as its source, and give consumers a canonicity check to call.
  </details>

## Nits

- **[a family the doc tool cannot see]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:1` — the three packages this one composes each open with a `// Package …` comment stating their contract; neither file here has one, so `gno doc` on the package that ties them together prints nothing.
- **[two nils, two outcomes, neither the package's own]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:31` — a nil interface is skipped in silence while a typed nil pointer walks past the guard and faults: measured at this head, `NewCollection(tok, (*royalty.Royalty)(nil))` aborts with `runtime error: nil pointer dereference` where every other bad input to this file yields a `collection:` message. Every sibling constructor and the core's `RegisterExtension` panic on a nil argument, so the skip is also the family's odd one out.

## Missing Tests

- **[the upgrade path]** `examples/gno.land/p/demo/tokens/grc721/collection/collection_test.gno:166` — no test attaches an extension after `Collection()` was handed out, which is both the documented hazard and the only mechanism a realm has for adding an extension after publication.
  <details><summary>details</summary>

  [`nft.gno:34-35`](https://github.com/jinoosss/gno/blob/refactor/grc721-collection/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L34-L35) · [↗](../../../../../.worktrees/gno-review-6074/examples/gno.land/p/demo/tokens/grc721/collection/nft.gno#L34) states that `Collection()` returns the live object and that a later `Attach` mutates it too. Nothing pins that, so a later change returning a defensive copy would pass the suite and silently break every realm that upgrades this way. It also has a ceiling worth asserting: a kind can be added but never replaced, since `Attach` panics on a duplicate. Fix: add the test that holds the pointer, attaches, and asserts both.
  </details>

## Suggestions

- **[two registries, one of them invisible]** `examples/gno.land/p/demo/tokens/grc721/collection/nft.gno:48` — an extension is registered on the ledger by its own constructor and recorded in the collection by `Attach`, and a realm that does the first without the second gets an extension that is fully live, hooks and events included, while every consumer reads a collection that has none.
  <details><summary>details</summary>

  `PrivateLedger.extensions` is unexported with no accessor, so neither side can detect the drift, and re-running the extension constructor to recover the lost view panics on the duplicate kind, which makes the loss permanent for the life of the token. Fix: have `Attach` take the constructor's pair, or return the view list from the core so the two can be reconciled.
  </details>
- **[a shape the siblings do not use]** `examples/gno.land/p/demo/tokens/grc721/collection/collection.gno:11` — the three extension packages split `types.gno` and `token.gno`; this one splits `collection.gno` and `nft.gno`. Cosmetic, and worth one decision for the family rather than a change here.

## Verified

- `Collection` has no path to a write: it reaches the ledger only through `Token`'s unexported field, which has no accessor.
- `Kinds()` iterates an `avl.Tree`, so discovery order is deterministic and does not depend on registration order.
- The mint and burn passthroughs return the core's errors unchanged, and a burn through the handle fans out to the extension hooks, which the package's own tests assert.
- `New` forwards its realm argument to the core, which asserts the realm is current and carries a package path, so a collection cannot be built from a user call.
- Type-punning a read view into its admin ledger is blocked at the realm boundary: the view and ledger structs share a layout in all three extension packages, so the conversion compiles, and the VM refuses it with `illegal conversion of readonly or externally stored value` for any value the calling realm does not own.
- `New` forwarding a realm that is not the current frame is refused at runtime with `rlm does not match the current crossing frame`, so the constructor cannot be handed a stale realm value.
