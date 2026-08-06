# PR [#6043](https://github.com/gnolang/gno/pull/6043): feat(grc20): let a registry own token identity and event emission

URL: https://github.com/gnolang/gno/pull/6043
Author: moul | Base: master | Files: 9 | +479 -6
Reviewed by: davd-gzl | Model: claude-opus-5 (deep) | Commit: 9ce031429 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6043 9ce031429`

**TL;DR:** Two token contracts made by the same realm can announce the same identifier in their events today, so a reader of the chain cannot tell which one an event came from. This PR lets a token hand its identifier and its event stream to a registry contract, on the theory that the chain stamps the sending contract's name on every event and that name cannot be borrowed.

**Verdict: REQUEST CHANGES** — the registry hands its stamping capability to any realm that asks, so a third party emits token events and the registry's own `register` event under the registry's name, and one issued identifier still backs two tokens, so the ambiguity the PR exists to remove survives inside the mechanism built to remove it (3 Critical, 5 Warnings, 3 Missing tests, 6 Nits, 5 Suggestions).

## Verify first

- [`examples/gno.land/r/demo/defi/grc20reg/emitter.gno:67-69`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) — confirm `Emitter()` is meant to be callable by anyone. `IssueToken` carries no gate, so a realm needs one line to hold a live `TokenEmitter`, and every Critical reaches its result through that one line.
- [`examples/gno.land/r/demo/defi/grc20reg/emitter.gno:102`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L102) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L102) — confirm `Emit` is meant to forward both `kind` and `attrs` untouched. `t.id` is read by `TokenID` and nowhere else, so the handle's binding to one identifier lives in the doc comment and not in the code, and the event type is the caller's to pick.
- [`examples/gno.land/p/demo/tokens/grc20/token.gno:105-108`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L105-L108) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L105-L108) — confirm the namespace prefix is meant to be the only check on the returned identifier. Drop [`tests/reused_handle_filetest.gno`](tests/reused_handle_filetest.gno) into `examples/gno.land/r/demo/defi/grc20reg/filetests/`: two constructions come back holding one identifier.

## Summary

`Token.ID()` is what a token writes into the `token` attribute of its `Transfer`, `Approval`, `Mint` and `Burn` events, and on master its trailing component is a [`seqid.ID` the creating realm supplies](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L53). Nothing stops a realm supplying the same value twice, which is [issue 6026](https://github.com/gnolang/gno/issues/6026). The PR adds an opt-in second constructor, [`NewTokenWithEmitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L80) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L80), which takes a registry realm through the new [`Emitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L71-L73) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L71-L73) interface, asks it for an identifier, and routes every later ledger event back through it. [`grc20reg`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74-L91) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74-L91) implements it with a per-realm counter in package state, and because the VM stamps `pkg_path` from the package calling `chain.Emit`, those events carry `gno.land/r/demo/defi/grc20reg` instead of grc20's path.

The stamping works. The instruction built on it, ["consumers filter on pkg_path and need trust nothing in-band"](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L29) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L29), does not. [`Emitter()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) hands the capability to any realm, and [`Emit`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L102) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L102) passes the caller's event type and attributes straight through, so both what an event under the registry's label says and what kind of event it is are whatever the caller typed. That reaches the registry's own `register` event as readily as a token's `Transfer`. The same entry point defeats the identifier guarantee: [the prefix check](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L106) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L106) tests the string and never that the handle is fresh, so a realm that fetches one handle and returns it from two `IssueToken` calls gets two ledgers behind one identifier, both emitting under the registry's path.

No realm under `examples/gno.land/` opts in. [`wugnot`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L27) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L27), [`foo20`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/foo20/foo20.gno#L23) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/foo20/foo20.gno#L23), [`test20`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/tests/vm/test20/test20.gno#L23) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/tests/vm/test20/test20.gno#L23) and [`grc20factory`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L40) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L40) all still call `NewToken`. Nothing in the tree is exposed today, and no Critical needs an adopter: each triggers on any realm calling the exported `Emitter()`, which this diff adds.

## Diagram

```
  any realm                              /r/demo/defi/grc20reg
  ┌──────────────────────┐               ┌────────────────────────────────┐
  │                      │  ① Emitter()  │  canonicalEmitter              │  no gate
  │  h := ───────────────┼──────────────▶│  IssueToken(cur, symbol)       │
  │        .IssueToken() │               │    sequences[caller]++         │
  │                      │               │    issued[id] = caller         │
  │                      │               └────────────────────────────────┘
  │  ② h.Emit("Transfer",│
  │       "token", <any>)│──────────────▶  chain.Emit, called from grc20reg
  │                      │                 ⇒ pkg_path: …/grc20reg   ◀── forged
  │                      │
  │  ③ NewTokenWithEmitter(…, relay{h}, cur)   ×2
  └──────────────────────┘                 ⇒ two Tokens, one identifier
```

Edge ① is the leak. Everything a token realm gets from the registry, a third party gets from the same call, and the only check on the way back is that the identifier starts with the caller's own path.

## Fix

`NewToken` is untouched. `NewTokenWithEmitter` repeats its five validation checks, calls `reg.IssueToken(cross(rlm), symbol)`, adopts the returned identifier when it starts with the calling realm's path, and stores the handle in the new unexported [`Token.emitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L95) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L95) field. The four `chain.Emit` call sites in `PrivateLedger` become [`led.token.emit`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L126-L132) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L126-L132), which dispatches to the handle when there is one. The load-bearing constraint is that `chain.Emit` takes its `pkg_path` from the calling frame's package, so an event routed through `grc20reg` carries `grc20reg`'s path whoever triggered it, which is also why edge ① is enough to forge one.

## Examples

| Written as | Identifier | `pkg_path` on its events |
|---|---|---|
| `NewToken("Legacy", "LEG", 6, 0, cur)` | `gno.land/r/demo/x.LEG.0000000` | `gno.land/p/demo/tokens/grc20` |
| `NewTokenWithEmitter("Owned", "OWN", 6, grc20reg.Emitter(), cur)` | `gno.land/r/demo/x.OWN.1` | `gno.land/r/demo/defi/grc20reg` |
| `grc20reg.Emitter().IssueToken(cross(cur), "X")` then `Emit` | caller's choice, event type and every attribute | `gno.land/r/demo/defi/grc20reg` |

## Critical (must fix)

- **[the registry's label is available to anyone who asks for it]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:101-103` — `Emit` forwards its attributes to `chain.Emit` untouched, and `Emitter()` hands a live handle to any realm, so a third party emits a `Transfer` naming any token under `pkg_path: gno.land/r/demo/defi/grc20reg`.
  <details><summary>details</summary>

  [`tokenEmitter.Emit`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L101-L103) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L101-L103) never reads `t.id`. The `token` attribute of every event it sends is supplied by whoever calls it, and [`Emitter()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L67-L69) exports `IssueToken` to every realm with no gate. Keeping the handle out of `Token`'s accessors, which [`types.gno:92-95`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L95) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L95) calls deliberate, protects nothing when a fresh one is one call away.

  This is the property the whole design rests on. [`emitter.gno:26-29`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L26-L29) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L26-L29) tells consumers to filter on `pkg_path` and trust nothing in-band; that filter now admits forged rows for any registry-owned token, and a consumer reconstructing balances from them credits the attacker's address. Verified: an unrelated realm emits a `Transfer` of 1000000 from a victim address, stamped with the registry's path, [repro](comment_claude-opus-5.md). The filetest asserting the post-fix state is [`tests/forged_token_event_filetest.gno`](tests/forged_token_event_filetest.gno). Fix: bind the event to the identifier the handle was issued for, so a holder cannot name another token.
  </details>

- **[one identifier still backs two tokens]** `examples/gno.land/p/demo/tokens/grc20/token.gno:105-108` — `NewTokenWithEmitter` checks the namespace of the identifier it gets back and never that the handle is fresh, so one registry-issued identifier is consumed twice and the ambiguity the PR exists to remove reappears wearing the registry's label.
  <details><summary>details</summary>

  A realm calls [`Emitter().IssueToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74) once, wraps the handle in a four-line `Emitter` that returns it every time, and constructs twice. Both tokens hold `gno.land/r/demo/grc20reuse.DUP.1`, both mint under the registry's `pkg_path`, and [`IssuedTo`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L107-L113) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L107-L113) confirms the registry issued it, so `Register` stamps one of the pair as registry-owned. This is worse than master, where the colliding pair at least carries grc20's path and no provenance claim.

  The gap is that [`sequences` and `issued`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L45-L56) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L45-L56) record issuance and nothing records consumption. [`NewTokenWithEmitter`'s doc](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L68-L69) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L68-L69) states the opposite, that the caller "never holds a TokenEmitter and cannot reuse one across two tokens", and that sentence is the justification for invoking `IssueToken` inside the constructor at all. Verified: two constructions return `gno.land/r/demo/grc20reuse.DUP.1` twice, [repro](comment_claude-opus-5.md). The filetest asserting the post-fix state is [`tests/reused_handle_filetest.gno`](tests/reused_handle_filetest.gno). Fix: make an identifier single-use, so the second construction on the same handle is rejected.
  </details>

- **[the registry's own discovery feed is forgeable too]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:102` — `Emit` forwards `kind` as well as `attrs`, so a handle emits a `register` event byte-identical to one `Register` would produce, for a token in a namespace the caller does not own.
  <details><summary>details</summary>

  [`Register`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L42-L48) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L42-L48) is the on-chain gate, rejecting a token from another realm and refusing a second registration under one key. The `register` event is its off-chain mirror, and a forged row detaches the mirror from the gate: no state is written, so neither guard fires, and an indexer keyed on the event sees a registration the contract would have rejected. The forged row can carry a `token_path` already registered, which `Register` itself answers with `token already registered`.

  The `emitter` attribute is the sharpest part, since it is the attestation this PR adds so [a consumer learns at registration time which kind of token it is looking at](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51-L56) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51-L56), and on a forged row it is caller-chosen. Binding the `token` attribute to the handle's identifier does not close this, because `kind` stays caller-chosen. On master the package has one `chain.Emit` call site, inside `Register`, and no exported function lets a caller pick an event type. Verified: a `register` row naming `gno.land/r/gnoland/wugnot` from an unrelated realm, with `registry has wugnot: false`, [repro](comment_claude-opus-5.md). The filetest asserting the post-fix state is [`tests/forged_register_event_filetest.gno`](tests/forged_register_event_filetest.gno). Fix: constrain what a handle may emit, so `register` stays the registry's alone.
  </details>

## Warnings (should fix)

- **[a keypair is enough to be a token issuer]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:75-78` — the "must be a realm" gate tests for the empty package path, which only a direct `maketx call` has, so a `maketx run` script issues identifiers with no deployed code and no namespace.
  <details><summary>details</summary>

  `rlmPath == ""` is [`IsUserCall()`](https://github.com/gnolang/gno/blob/9ce031429/gnovm/stdlibs/chain/runtime/frame.gno#L103-L107) · [↗](../../../../../.worktrees/gno-review-6043/gnovm/stdlibs/chain/runtime/frame.gno#L103-L107) spelled out, and that predicate is true for `maketx call` alone. A `maketx run` frame carries `gno.land/e/<addr>/run`, so it walks through; [`IsUser`](https://github.com/gnolang/gno/blob/9ce031429/gnovm/stdlibs/chain/runtime/frame.gno#L78-L84) · [↗](../../../../../.worktrees/gno-review-6043/gnovm/stdlibs/chain/runtime/frame.gno#L78-L84) is the predicate that covers both and is the one a realms-only gate rejects on. Each address then owns a `sequences` key, and the per-realm counter the design rests on becomes per-address.

  Registering such a token stamps it `emitter: gno.land/r/demo/defi/grc20reg`, the strongest attribution signal the PR defines, for an issuing path that holds no code anyone can read. Verified: a `maketx run` script took `gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run.EPH.1` and `.EPH.2` out of the shared counter, [repro](comment_claude-opus-5.md).
  </details>

- **[the provenance stamp is forgeable]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:57-60` — `IssuedTo` matches the identifier string, not the handle the token holds, so a realm registers a token stamped `emitter: gno.land/r/demo/defi/grc20reg` whose events never touch the registry.
  <details><summary>details</summary>

  Take an identifier from [`IssueToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74), then construct the token on a counterfeit `Emitter` returning that same string. The [namespace check](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L106) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L106) passes, since the identifier really is in the caller's namespace, and [`Register`'s lookup](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58) hits.

  [The comment on the check](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51-L53) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L51-L53) says the registry reads its own record "rather than against anything the token says about itself". The record proves an identifier was handed to a realm; it does not prove this object routes its events anywhere. A consumer told at registration time that the token is attributable then receives no events from it at all. Verified: `Register` reported the registry as emitter for a token whose `Mint` emitted nothing, [repro](comment_claude-opus-5.md). The filetest asserting the post-fix state is [`tests/forged_provenance_filetest.gno`](tests/forged_provenance_filetest.gno). Fix: record consumption at construction, so the stamp names what the token actually holds.
  </details>

- **[the identifier need not name the token's own symbol]** `examples/gno.land/p/demo/tokens/grc20/token.gno:106` — the check tests the realm prefix only, so a token reporting symbol `BBB` emits under the identifier `<realm>.AAA.1`.
  <details><summary>details</summary>

  [`NewToken` builds the identifier itself](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L53) as `pkgPath + "." + symbol + "." + id`, so its middle component is always the token's own symbol. `NewTokenWithEmitter` lets the `Emitter` choose everything after the realm path. [Its doc](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L71-L74) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L71-L74) claims the check "preserves the namespace guarantee NewToken gives, no matter what reg is"; the realm half is preserved and the symbol half is not.

  [`Register`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L40-L45) rebuilds its key from `token.GetSymbol()` and rejects such a token, so the mismatch is caught at registration and never before it. An unregistered token emits `Transfer` events naming a symbol it does not have for as long as it likes. [`filetests/emitter_foreign_namespace_filetest.gno`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/filetests/emitter_foreign_namespace_filetest.gno#L18) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/filetests/emitter_foreign_namespace_filetest.gno#L18) covers the cross-realm direction only, which is why this passes. Verified: `symbol: BBB` with `id: gno.land/r/demo/grc20sym.AAA.1`, [repro](comment_claude-opus-5.md). The filetest asserting the post-fix state is [`tests/unbound_symbol_filetest.gno`](tests/unbound_symbol_filetest.gno).
  </details>

- **[a token can now emit nothing at all]** `examples/gno.land/p/demo/tokens/grc20/token.gno:126-132` — whether a ledger write emits is the `Emitter` implementer's choice, so supply and balances move against an empty event stream and a consumer cannot tell that token from one that emits under grc20's path.
  <details><summary>details</summary>

  Before the diff the four ledger mutations reached `chain.Emit` unconditionally. `emit` now dispatches to a caller-supplied implementation whenever the token has one, and an `Emit` with an empty body clears every nil check and the namespace check. At registration `IssuedTo` misses, so the token is stamped `emitter: ""`, the same stamp a `NewToken` token gets, and that one does emit.

  The accidental route reaches the same place. [`if reg == nil`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L97-L99) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L97-L99) and [`if emitter == nil`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L102-L104) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L102-L104) catch an untyped nil literal only. A typed-nil pointer in the interface passes both, so `tok.emitter != nil` is true forever and every event dispatches to a nil receiver. Verified: a typed-nil emitter gave `mint err: true` with no event recorded, and an empty-bodied `Emit` swallowed four ledger mutations, [repro](comment_claude-opus-5.md). Fix: make silence detectable, either by rejecting an emitter that does not emit or by giving a consumer a way to tell the two cases apart.
  </details>

- **[the test named for the replay attack cannot fail]** `examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno:42-53` — `emitter` is a zero-field struct, so `snapshot := *canonicalEmitter` copies nothing and the test would pass with the copy deleted.
  <details><summary>details</summary>

  [`type emitter struct{}`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L41) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L41) holds no state, so `&snapshot` dispatches to the same package-level `sequences` tree as `Emitter()` does. The assertion left standing is `NotEqual(first.ID(), second.ID())`, which [`TestEmitterNeverReissuesAnIdentifier`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L15-L24) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L15-L24) already makes.

  The copyable value in this design is the returned `TokenEmitter`, not the stateless `emitter`, and copying that one succeeds. The PR body cites this test as evidence for its first design property, so it reads as a demonstration where it is a tautology. Fix: assert that the copy draws the next number from the shared counter, and add the handle-reuse case the identifier Critical describes.
  </details>

## Nits

- **[one doc claim survives both fixes]** `examples/gno.land/p/demo/tokens/grc20/token.gno:68-69` — ["the caller never holds a TokenEmitter and cannot reuse one across two tokens"](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L68-L69) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L68-L69) is false on its first half through the exported `Emitter()`, and no Critical's fix closes that route. The other doc claims resolve with the code: ["consumers filter on pkg_path and are done"](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L38-L42) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L38-L42) becomes true once `Emit` is bound to the handle's identifier, and ["two Tokens cannot collide"](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L44-L46) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L44-L46) becomes true once an identifier is single-use.

- **[the compatibility section describes an operation the design does not offer]** `examples/gno.land/p/demo/tokens/grc20/token.gno:116` — [line 116](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L116) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L116) is the only assignment to `Token.emitter` and there is no setter, so moving a token onto a registry emitter later, which the body defers as "a separate, deliberate decision per token", cannot be done without a new package path. The benign half is that the added field never meets a persisted five-field `*grc20.Token`, so there is no struct-shape hazard.

- **[a written rule the diff does not follow]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:74-75` — [`IssueToken`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74-L75) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L74-L75) reads `cur.Previous()` with no `cur.IsCurrent()` first, which [`AGENTS.md`](https://github.com/gnolang/gno/blob/9ce031429/AGENTS.md?plain=1#L101) · [↗](../../../../../.worktrees/gno-review-6043/AGENTS.md#L101) states without qualification, and the doc comment two lines up leans on that frame read as the authentication. Not posted, no change needed: the VM already rejects the only path that reaches the read with a forgeable value, `cannot cur-call to external realm function`, so the guard is unreachable protection rather than a missing one. [`Register`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L39) has the same shape and predates the branch.

- **[a constant written as a frame read]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:59` — [`emitterPath = cur.PkgPath()`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L59) is always `gno.land/r/demo/defi/grc20reg`, since `Register` is this realm's own crossing function. Written as a frame read, it suggests the value varies with the caller.

- **[the counter is per realm, and the identifier reads as though it were per symbol]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:85` — [`sequences`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L85) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter.gno#L85) is keyed by realm path, so a realm's first two tokens come out `.AAA.1` and `.BBB.2`. The identifier puts the symbol beside a number that is not the symbol's own.

- **[the doc block above `Register` now contradicts the comment inside it]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:17-31` — [line 28](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L28-L29) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L28-L29) says the trailing sequence id "keeps token identities/events unique"; [line 54](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L54-L55) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L54-L55), added by this diff, says two ledgers behind one identifier remain possible. [Line 17](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L17) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L17) also still says construction lives in `NewToken`.

## Missing Tests

- **[the attribute the compatibility story rests on]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:69` — nothing asserts the `emitter` attribute on the register event.
  <details><summary>details</summary>

  [`TestRegisterReportsWhetherTheRegistryIssuedTheIdentifier`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L57-L71) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/emitter_test.gno#L57-L71) calls `IssuedTo` and never reads the event, so the test named for the report does not test the report. The PR's backward-compatibility section makes this attribute the thing a consumer learns from at registration time.

  <details><summary>test cases</summary>

  ```go
  // PKGPATH: gno.land/r/demo/grc20regattr

  // Register's emitter attribute is the only thing that tells a consumer, at
  // registration time, which kind of token it is looking at. One realm registers
  // one token of each kind, so the two rows sit side by side.
  package grc20regattr

  import (
  	"gno.land/p/demo/tokens/grc20"
  	"gno.land/r/demo/defi/grc20reg"
  )

  func main(cur realm) {
  	owned, _ := grc20.NewTokenWithEmitter("Owned", "OWN", 6, grc20reg.Emitter(), cur)
  	grc20reg.Register(cross(cur), owned, "")

  	legacy, _ := grc20.NewToken("Legacy", "LEG", 6, 0, cur)
  	grc20reg.Register(cross(cur), legacy, "")
  }
  ```

  The golden carries `emitter: gno.land/r/demo/defi/grc20reg` on the first register event and `emitter: ""` on the second.
  </details>
  </details>

- **[the premise of the design is untested]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:80-88` — every test runs inside one transaction, so nothing exercises the counter surviving a transaction boundary.
  <details><summary>details</summary>

  [`emitter.gno:19-20`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L19-L20) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L19-L20) names cross-transaction survival as the reason a `/p/` package cannot own the counter. Unit tests and filetests both run in one transaction, so the property holds untested. It does hold: a two-call `.txtar` returns `.AAA.1` then `.BBB.2` across two `maketx call` transactions.

  <details><summary>test cases</summary>

  ```
  # The registry counter has to survive a transaction boundary, which no unit
  # test or filetest crosses.

  loadpkg gno.land/r/demo/defi/grc20reg
  loadpkg gno.land/r/demo/seqtok $WORK

  gnoland start

  gnokey maketx call -pkgpath gno.land/r/demo/seqtok -func New -args AAA -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
  stdout OK!
  stdout '\("gno.land/r/demo/seqtok.AAA.1" string\)'

  gnokey maketx call -pkgpath gno.land/r/demo/seqtok -func New -args BBB -gas-fee 1000000ugnot -gas-wanted 20000000 -broadcast -chainid=tendermint_test test1
  stdout OK!
  stdout '\("gno.land/r/demo/seqtok.BBB.2" string\)'

  -- gnomod.toml --
  module = "gno.land/r/demo/seqtok"
  gno = "0.9"
  -- seqtok.gno --
  package seqtok

  import (
  	"gno.land/p/demo/tokens/grc20"
  	"gno.land/r/demo/defi/grc20reg"
  )

  func New(cur realm, symbol string) string {
  	tok, _ := grc20.NewTokenWithEmitter("T", symbol, 6, grc20reg.Emitter(), cur)
  	return tok.ID()
  }
  ```
  </details>
  </details>

- **[three of the four rerouted call sites]** `examples/gno.land/p/demo/tokens/grc20/token.gno:126-132` — only `Mint` is exercised through a non-nil emitter, so a routing regression on `Transfer`, `Approve` or `Burn` passes CI.
  <details><summary>details</summary>

  [`filetests/emitter_pkgpath_filetest.gno`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/r/demo/defi/grc20reg/filetests/emitter_pkgpath_filetest.gno#L24-L25) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/r/demo/defi/grc20reg/filetests/emitter_pkgpath_filetest.gno#L24-L25) mints on both tokens and stops. The other three call sites are [`Transfer`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L281) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L281), [`Approve`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L331) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L331) and [`Burn`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L396) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L396). Adding an `Approve` and a `Burn` to that filetest and regenerating the golden closes it.

  The two new errors are unreached too. [`ErrNilEmitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L132) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/types.gno#L132) and [`ErrNilTokenEmitter`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/types.gno#L133) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/types.gno#L133) have no assertion anywhere. `uassert.PanicsContains` needs a plain `func()` closure here; a `func(cur realm)` is invoked through `cross(rlm)` and the panic arrives as an abort.
  </details>

## Suggestions

- **[registry state grows from unauthenticated calls and never shrinks]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:85-88` — every `IssueToken` call writes two permanent tree entries whether or not a token is ever built, and no path in the realm removes either.
  <details><summary>details</summary>

  Measured: twenty calls from one `maketx run`, creating no token, took `grc20reg` from 15,134 to 58,599 bytes, so 43,465 bytes and 4,346,500 ugnot of deposit locked and unreclaimable. The caller pays, so this is priced rather than free, but a shared public registry nearly quadrupled in one transaction and nobody can shrink it back.

  The honest path pays too: registering one token costs `grc20reg` 3,944 bytes on the `NewToken` path against 7,025 bytes on the `NewTokenWithEmitter` path, so opting in locks +3,081 bytes per token forever. Recording on consumption rather than on issuance would cut the wasted half and close the second Critical in the same move.
  </details>

- **[`IssueToken` validates nothing about `symbol`]** `examples/gno.land/r/demo/defi/grc20reg/emitter.gno:87` — reached directly, the symbol is arbitrary bytes, so the identifier's `<realm>.<symbol>.<seq>` shape stops parsing.
  <details><summary>details</summary>

  [`validSymbol`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L152-L162) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L152-L162) restricts the charset and the length, and `NewTokenWithEmitter` runs it, but the direct call bypasses it. Dots, slashes, quotes, newlines and unbounded length all land in the `issued` key and in `IssuedTo`'s return: `gno.land/r/demo/x.A.1.2` reads as symbol `A` sequence `1.2` or symbol `A.1` sequence `2`.

  No collision follows. Within a realm the sequence strictly increments and the final dot always precedes decimal digits, and a package path element cannot contain a dot, so two `issued` keys cannot be made equal. What is left is an unparseable identifier and a caller-sized key in permanent state.
  </details>

- **[the interface cannot be wrapped, and the panic blames the wrong party]** `examples/gno.land/p/demo/tokens/grc20/emitter.gno:71-73` — `IssueToken` takes the namespace from `cur.Previous()`, so only an implementation entered straight from `NewTokenWithEmitter`'s frame can return an accepted identifier.
  <details><summary>details</summary>

  Any adapter, versioned wrapper, multi-registry aggregator, or v2-delegating-to-v1 registry is structurally impossible: the forwarded call makes the wrapper the previous realm, so the identifier lands in the wrapper's namespace and the caller sees `emitter issued an id outside the calling realm's namespace`. The message names the emitter as the offender when the shape is the constraint. Nothing in the doc block mentions it, and the interface is the extension point the design offers.
  </details>

- **[a realm that takes an `Emitter` from its own caller hands out a permanent hook]** `examples/gno.land/p/demo/tokens/grc20/token.gno:101` — this is the worst thing an honest author can get wrong, and the capability note warns about the opposite direction.
  <details><summary>details</summary>

  A factory writing `func Deploy(cur realm, name, symbol string, reg grc20.Emitter)` lets its caller choose the emitter. The identifier is built in the factory's namespace, so the prefix check passes, and the attacker holds a permanent hook on every ledger write of a token carrying the factory's path: silence it, retitle it, or emit alongside it. [The capability note](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L48-L57) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/emitter.gno#L48-L57) covers handing a `TokenEmitter` out and says nothing about taking an `Emitter` in.

  This matches two rows of the realm audit patterns at once, `callback-param` and `interface-realm-param`. Neither is a privilege escalation on its own: inside `IssueToken` the implementer's own realm is current, so it cannot act as the creating realm, and `cur.Previous()` only reveals who called.
  </details>

- **[the validation block is duplicated verbatim]** `examples/gno.land/p/demo/tokens/grc20/token.gno:81-96` — five checks, byte-identical to [`NewToken:35-49`](https://github.com/gnolang/gno/blob/9ce031429/examples/gno.land/p/demo/tokens/grc20/token.gno#L35-L49) · [↗](../../../../../.worktrees/gno-review-6043/examples/gno.land/p/demo/tokens/grc20/token.gno#L35-L49), with nothing linking them, so a check added to one constructor is silently absent from the other. The shared part is a pure function of `(name, symbol, decimals, rlm)`.

## Verified

- Any realm emits a `Transfer` naming another realm's token id under `pkg_path: gno.land/r/demo/defi/grc20reg`, and a `register` row naming `gno.land/r/gnoland/wugnot` while the registry holds no such key, and two constructions on one reused handle both come back holding `gno.land/r/demo/grc20reuse.DUP.1`.
- A `gnokey maketx run` script, with no deployed realm anywhere, took `gno.land/e/g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5/run.EPH.1` and `.EPH.2` out of the shared counter.
- `Register` stamped `emitter: gno.land/r/demo/defi/grc20reg` on a token whose `Mint` emitted nothing, and a typed-nil emitter reached the same silence with `Mint` returning no error.
- The balance retunes in [`storage_deposit_price_change.txtar`](https://github.com/gnolang/gno/blob/9ce031429/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L37) · [↗](../../../../../.worktrees/gno-review-6043/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L37) and [line 72](https://github.com/gnolang/gno/blob/9ce031429/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) · [↗](../../../../../.worktrees/gno-review-6043/gno.land/pkg/integration/testdata/storage_deposit_price_change.txtar#L72) move by the same 1,176,600 ugnot, which at the default 100 ugnot per byte is 11,766 bytes against 11,471 bytes of added non-test source in the two genesis-loaded packages, so the retune is deployment-size drift and not a change in storage accounting. The realm-storage assertions in the same file are untouched.
- `gno test ./...` over `examples/` gives 222 packages ok and zero failures at 9ce031429 and at the merge base alike, and `gno fmt -diff` and `gno lint` over both changed packages are clean.

## Open questions

- The `register` event gained `token_id` and `emitter`. No consumer in the tree reads it positionally and no txtar golden asserts its attribute list, so the shape change alone breaks nothing; worth one line in the PR body since the event is the registry's public interface. Forgery of that event is the third Critical, not this. Not posted, no change needed.
- [PR 6042](https://github.com/gnolang/gno/pull/6042) and [PR 6028](https://github.com/gnolang/gno/pull/6028) are both open against the same issue, and this PR's body argues against 6028 on a copyable-generator ground that the second Critical shows this design shares in another shape. Which of the three lands first is a maintainer decision, not a review finding.
