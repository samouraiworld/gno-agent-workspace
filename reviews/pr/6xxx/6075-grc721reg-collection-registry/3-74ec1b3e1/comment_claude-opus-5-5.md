# Review: [#6075](https://github.com/gnolang/gno/pull/6075)
Event: REQUEST_CHANGES
Verdict: REQUEST CHANGES. Two confirmed Warnings in the new realm: `publishedKinds` applies its limit before dropping view-less kinds, so `Register` and the index page under-report kinds with nothing marking the list partial.
Model: claude-opus-5-5, standard review
Commit: 74ec1b3e1ad9406d645443fffbca8ae277b8edf7
Overview: [overview.md](../overview.md)
Open the code: [grc721reg.gno](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno)
Round: 3. 8 finders, one reflector, 21 candidates, the Criticals and Warnings run by their finders and judged by an agent that was not the finder, the rest judged by read; 0 refuted, two Warnings among the confirmed.

## Body

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:278 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L278) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L278) · Warning

`publishedKinds` takes the first `limit` kinds from `ExtensionKindsUpTo` before dropping those registered with no view. One such hook registered first lets a collection with 17 views past the 16-kind cap, and its `register` event omits a published kind.

<details><summary>repro</summary>

The test asserts the under-report, so PASS is the defect: the event lists 16 kinds while 17 are published.

```
gno test -v -run TestB2Viewless ./gno.land/r/demo/defi/grc721reg/
event kinds=16 published kinds=17 maxEventKinds=16
badges=8 elided=false published=10
```

Harness: `tests/b2-lines-viewless-cap_test.gno` in this round.
</details>

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:246 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L246) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L246) · Warning

`extensionBadges` asks `publishedKinds` for the first 9 registered kinds, so one view-less hook among them leaves 8 of 9 published kinds on the page with no `…`. Past nine kinds, registration order picks the badges, which the `publishedKinds` godoc says sorting rules out.

<details><summary>repro</summary>

Both tests fail: the page drops a kind silently, and `kind0` never shows when kinds are registered in reverse.

```
gno test -v -run 'TestCatalogBadgesHideKindsBehindHook|TestCatalogBadgeSetDependsOnOrder' ./gno.land/r/demo/defi/grc721reg/
badges: published 9 shown 8 elided false
order: kind0 shown false
=== RUN   TestCatalogBadgesHideKindsBehindHook
Render shows 8 of 9 published kinds with no elision mark
--- FAIL: TestCatalogBadgesHideKindsBehindHook (0.00s)
=== RUN   TestCatalogBadgeSetDependsOnOrder
sorted-first kind0 missing from badges: # Ord
- symbol: **ORD**
- realm: [gno.land/r/demo/cat_order](/r/demo/cat_order).ORD
- total supply: 0
- extensions: `kind1` `kind2` `kind3` `kind4` `kind5` `kind6` `kind7` `kind8` …

--- FAIL: TestCatalogBadgeSetDependsOnOrder (0.00s)
```

Harness: `tests/b3-catalog-viewless-window_test.gno` in this round.
</details>

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:146 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L146) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L146) · Missing test

Missing test: no test pins the `limit` break in `ExtensionKindsUpTo`, so deleting it leaves every `grc721` and `grc721reg` test green. PLAUSIBLE, no verifier ran the mutation.

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:208 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L208) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L208) · Missing test

Missing test: a rejected view leaving `led.extensions` empty is pinned by no test. PLAUSIBLE, no verifier ran the mutation.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:252 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L252) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L252) · Missing test

Missing test: the persistent-realm gate is pinned against `NewUserRealm` only, never against a `maketx run` realm. PLAUSIBLE, no verifier ran the mutation.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:351 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L351) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L351) · Missing test

Missing test: no test registers through a sub-identity from `cur.Sub`, which `Register` accepts. PLAUSIBLE, no verifier ran it.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:555 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L555) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L555) · Missing test

Missing test: no txtar registers in `init`, mints in a later transaction and reads `GetExtension` from a consumer realm. PLAUSIBLE, no verifier ran it.

## examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno:20 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L20) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/enumerable/token.gno#L20) · Nit

Related nit: passing `view` here publishes an enumerable attached after minting, whose `TotalSupply` skips every token minted before it: 0 against a core supply of 2. The missing guard predates the diff.

<details><summary>repro</summary>

Two tokens minted, then the enumerable attached: the published view disagrees with the core.

```
gno test -v -run TestLateEnumerable ./gno.land/r/demo/defi/grc721reg/
core TotalSupply: 2 | published enumerable TotalSupply: 0
TokenByIndex(0) err: true | TokenOfOwnerByIndex(carol,0): 1 true | TokenOfOwnerByIndex(bob,0) err: true
```

Harness: `tests/b1-lines-removed-reach-catalog-late-enumerable_test.gno` in this round.
</details>

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno:314 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L314) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg_test.gno#L314) · Nit

Nit: the comment names `collection.KindsUpTo`, which no longer exists at the head. UNVERIFIED, on the finder's read only.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:122 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L122) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L122) · Nit

Nit: the pull request description still describes the closed gnolang/gno#6074 design: it registers `*collection.Collection`, lists a `GetToken` lookup and waits on #6074 to merge. UNVERIFIED, kept in claims.md.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:204 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L204) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L204) · Nit

Nit: `infoLink` pastes the raw key, so for a sub-realm key, whose realm path holds `#`, the browser drops the rest as a fragment and the link opens "Unknown collection".

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:286 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L286) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L286) · Nit

Nit: the Warning on line 246 carries this order-dependence.

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:142 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L142) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L142) · Suggestion

Suggestion: the Suggestion on line 156 carries this quadratic walk.

## SKIP examples/gno.land/p/demo/tokens/grc721/token.gno:155 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L155) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L155) · Suggestion

Suggestion: the Suggestion on line 198 carries this hook-as-view exposure.

## examples/gno.land/p/demo/tokens/grc721/token.gno:156 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L156) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L156) · Suggestion

Suggestion: `lookupExtension` re-scans `led.extensions` for each kind `publishedKinds` walks, so `GetExtensionKinds` costs quadratic gas in the issuer's own extension count, 28.0M at 200 hooks and 107.3M at 400.

<details><summary>repro</summary>

Read gas minus setup gas grows about 4x when the hook count doubles.

```
gno test -v -print-runtime-metrics -run TestB2Cost ./gno.land/r/demo/defi/grc721reg/
Setup200 GAS 33585552, Read200 61596876
Setup400 GAS 118351897, Read400 225629763
```

Only calls on that issuer's own key pay it. Harness: `tests/b2-lines-kinds-quadratic_test.gno` in this round.
</details>

## examples/gno.land/p/demo/tokens/grc721/token.gno:193 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L193) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L193) · Suggestion

Suggestion: `ext.ExtensionKind()` is accepted at any length, so a 4096-byte kind attached after `Register` comes back whole from `GetExtensionKinds`, past the 64-byte check `Register` runs.

## examples/gno.land/p/demo/tokens/grc721/token.gno:198 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/p/demo/tokens/grc721/token.gno#L198) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/p/demo/tokens/grc721/token.gno#L198) · Suggestion

Suggestion: `view` may be the hook itself, so any realm reading `GetExtension` can assert it to `grc721.Extension` and call `OnTransfer` or `OnBurn` on the issuer's state.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:33 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L33) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L33) · Suggestion

Suggestion: `slug` is validated and emitted in the event but never stored or read, since the key comes from the token's symbol.

## examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:51 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L51) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L51) · Suggestion

Suggestion: `prev.IsUser()` lets a sub-identity from `cur.Sub` register, so a collection is keyed on `host#subpath` with no test and no doc saying that is intended.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:79 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L79) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L79) · Suggestion

Suggestion: the Warning on line 278 carries this window-before-filter defect.

## SKIP examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno:158 [gh](https://github.com/gnolang/gno/blob/74ec1b3e1ad9406d645443fffbca8ae277b8edf7/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L158) · [↗](../../../../../.worktrees/gno-review-6075/examples/gno.land/r/demo/defi/grc721reg/grc721reg.gno#L158) · Suggestion

Suggestion: the Suggestion on `token.gno` line 156 carries this quadratic walk.
