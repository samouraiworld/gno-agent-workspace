# PR [#5983](https://github.com/gnolang/gno/pull/5983): feat: add token-bound-account realm example using realm.Sub()

URL: https://github.com/gnolang/gno/pull/5983
Author: zeycan1 | Base: master | Files: 2 | +157 -0
Reviewed by: davd-gzl | Model: claude-opus-4-8 | Commit: 7d9a11104 (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5983 7d9a11104`

**TL;DR:** Adds a demo realm where every NFT comes with its own on-chain wallet. Anyone can send coins to that wallet, only the person holding the NFT can take them out, and selling the NFT hands the wallet over with it.

**Verdict: REQUEST CHANGES** — the vault mechanics work end to end, but an unauthorized transfer is reported as a successful transaction, the vault address has two independent derivations that nothing keeps in sync, and the realm ships with no tests (3 Warnings, 1 Missing test, 4 Nits, 2 Suggestions).

## Summary

The realm mints GRC721 tokens and gives each one a [sub-realm token](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/pkg/gnolang/uverse.go#L1715-L1770) · [↗](../../../../../.worktrees/gno-review-5983/gnovm/pkg/gnolang/uverse.go#L1715-L1770) at subpath `vault/<id>`, whose address holds ugnot. [`Withdraw`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L73-L86) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L73-L86) mints the sub-token, builds a `BankerTypeRealmSend` [banker](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/stdlibs/chain/banker/banker.gno#L166-L175) · [↗](../../../../../.worktrees/gno-review-5983/gnovm/stdlibs/chain/banker/banker.gno#L166-L175) on it, and sends. The owner gate holds: a non-owner is rejected, and after a transfer the old owner loses access while the new owner gains it, balance intact. Amount and recipient validation is entirely the banker's: a negative amount aborts with `invalid coins error`, an overdraw with `insufficient coins error`, a malformed recipient with a bech32 error. Zero is the one value that gets through, as a no-op costing 5.06M gas.

Two structural problems sit above that. [`TransferFrom`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L96-L99) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L96-L99) returns the GRC721 error instead of panicking, so an unauthorized transfer commits a successful transaction that moved nothing, while every other guard in the file panics. And the vault address is derived twice from different sources: [`VaultAddress`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L64-L66) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L64-L66) hashes the hand-written [`selfPath`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L15) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L15) constant, `Withdraw` hashes the realm's real package path through `cur.Sub`. They agree at this path and nothing checks that they still do at another.

## Glossary

- addpkg: the transaction (`maketx addpkg`) that uploads a package or realm to the chain.
- banker: stdlib API (`chain/banker`) for issuing, sending, and burning coins from a realm.
- crossing / `cross`: a call into a crossing function (`func F(cur realm, ...)`), where the callee identifies its caller through `cur.Previous()`.
- realm: a stateful on-chain package under `r/`; also the VM builtin threaded as a `cur realm` parameter.
- sub-realm token: a realm value minted by `cur.Sub(subpath)`, with a synthesized `host#subpath` pkgpath and an address hashed from it; `chain.DerivePkgSubAddr` computes the same address off-chain.

## Examples

| Call | Outcome on chain |
|---|---|
| `Withdraw(tid, to, -100)` | aborts, `invalid coins error` from the banker native |
| `Withdraw(tid, to, 5000)` on a 1000 balance | aborts, `insufficient coins error` |
| `Withdraw(tid, "", 100)` | aborts, `bech32.ErrInvalidLength` |
| `Withdraw(tid, to, 0)` | succeeds, moves nothing, 5,063,313 gas |
| `TransferFrom(from, to, tid)` by a stranger | succeeds, moves nothing, returns `caller is not token owner or approved` |
| `Mint(...)` before `Setup()` | aborts, `runtime error: nil pointer dereference` inside grc721 |

## Warnings (should fix)

- **[a rejected transfer looks like a completed one]** `examples/gno.land/r/zeycan1/tba/tba.gno:96-99` — an unauthorized `TransferFrom` commits a successful transaction and moves nothing.
  <details><summary>details</summary>

  [`nft.TransferFrom`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L214-L225) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L214-L225) returns `ErrCallerIsNotOwnerOrApproved` when [`isApprovedOrOwner`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L405-L422) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L405-L422) rejects the caller, and the wrapper hands that error back to the caller instead of panicking. A returned error does not fail a MsgCall, so `gnokey` prints `OK!` with the error as the result value and the token stays put. The three other guards in the file, [`Setup`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L39-L41) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L39-L41), [`Mint`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L47-L49) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L47-L49) and [`Withdraw`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L79-L81) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L79-L81), all panic, so a caller reading transaction status has no reason to expect this one to be different. Observed on chain, [repro](comment_claude-opus-4-8.md). Fix: panic on the returned error.
  </details>

- **[two derivations of one address, only one of them checked]** `examples/gno.land/r/zeycan1/tba/tba.gno:15` — `VaultAddress` hashes the `selfPath` constant while `Withdraw` hashes the realm's real package path, and nothing ties the two together.
  <details><summary>details</summary>

  [`chain.DerivePkgSubAddr`](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/stdlibs/chain/address.gno#L28-L31) · [↗](../../../../../.worktrees/gno-review-5983/gnovm/stdlibs/chain/address.gno#L28-L31) takes the host path as an argument, so [`VaultAddress`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L64-L66) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L64-L66) and [`Balance`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L68-L71) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L68-L71) answer for whatever `selfPath` says. `cur.Sub` in [`Withdraw`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/tba.gno#L83-L85) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/tba.gno#L83-L85) derives from the realm's own pkgpath and cannot be pointed elsewhere. Deployed as-is the two match. A copy of this file deployed at another path with the constant untouched advertises the original realm's vault: `VaultAddress("1")` returns `g13aazlmjgmalp0r9q72jhv6f0t82vj0q76d8cez`, `Balance("1")` reports the deposit, and `Withdraw` aborts with `insufficient coins error` while the coins sit in the original realm's vault, spendable by whoever holds token 1 there. Reproduced with an addpkg of this file at `gno.land/r/<addr>/tba`, [repro](comment_claude-opus-4-8.md). Fix: make `Setup` reject a mismatch between `selfPath` and the realm's own path.
  </details>

- **[the deployer, not the author, ends up holding the realm]** `examples/gno.land/r/zeycan1/tba/tba.gno:30-32` — `admin` is whoever signs the deployment, which for a package shipped in `examples/` is the chain's genesis key.
  <details><summary>details</summary>

  At [`StageAdd`](https://github.com/gnolang/gno/blob/7d9a11104/gnovm/stdlibs/internal/execctx/realm.go#L58-L69) · [↗](../../../../../.worktrees/gno-review-5983/gnovm/stdlibs/internal/execctx/realm.go#L58-L69) `unsafe.PreviousRealm()` resolves to the transaction's origin caller, so `admin` is the addpkg signer. Packages under `examples/` are deployed by [the genesis key](https://github.com/gnolang/gno/blob/7d9a11104/gno.land/cmd/gnoland/start.go#L453) · [↗](../../../../../.worktrees/gno-review-5983/gno.land/cmd/gnoland/start.go#L453) unless the manifest names a creator, which [genesis loading honours](https://github.com/gnolang/gno/blob/7d9a11104/gno.land/pkg/gnoland/genesis.go#L199-L207) · [↗](../../../../../.worktrees/gno-review-5983/gno.land/pkg/gnoland/genesis.go#L199-L207) and 20 of the 58 realm manifests under `examples/gno.land/r/` set, e.g. [devrels/events](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/devrels/events/gnomod.toml#L4-L5) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/devrels/events/gnomod.toml#L4-L5). [`gnomod.toml`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/zeycan1/tba/gnomod.toml#L1-L2) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/zeycan1/tba/gnomod.toml#L1-L2) sets none. Verified by loading the realm through genesis and calling `Setup` as the genesis key, which succeeds. The import also trips [checklist item 9](https://github.com/gnolang/gno/blob/7d9a11104/docs/resources/gno-ai-contract-review.md?plain=1#L140-L161) · [↗](../../../../../.worktrees/gno-review-5983/docs/resources/gno-ai-contract-review.md#L140-L161), which flags `chain/runtime/unsafe` in any realm that also threads `cur realm`; the convention in `examples/` is a literal address, as in [r/gnoland/home](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/gnoland/home/home.gno#L16) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/gnoland/home/home.gno#L16). Fix: name the admin address in the source and drop the `unsafe` import.
  </details>

## Nits

- **[a no-op transaction that still bills]** `examples/gno.land/r/zeycan1/tba/tba.gno:73-86` — `Withdraw` accepts an amount of zero, committing a transaction that moves nothing for 5,063,313 gas. Confirmed behaviorally against a funded vault.
- **[the error names grc721, not the missing step]** `examples/gno.land/r/zeycan1/tba/tba.gno:52` — calling `Mint` before `Setup` aborts with `runtime error: nil pointer dereference` raised inside [`BasicNFT.exists`](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L425-L427) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/p/demo/tokens/grc721/basic_nft.gno#L425-L427). Only the admin can reach it. Confirmed behaviorally on chain.
- **[the feature the realm exists to show is not on the page]** `examples/gno.land/r/zeycan1/tba/tba.gno:131-136` — `Render` discards `path` and returns `nft.RenderHome()`, so a `vm/qrender` of `:token/1` shows the collection header and no vault, address, or balance. Confirmed behaviorally.
- **[a reference realm read from its signatures alone]** `examples/gno.land/r/zeycan1/tba/tba.gno:34` — none of the fourteen exported functions carries a doc comment, and the admin-only and owner-only contracts live only in panic strings.

## Missing Tests

- **[nothing pins the two gates that hold the coins]** `examples/gno.land/r/zeycan1/tba/tba.gno:1` — the realm ships with no `_test.gno` file, so the admin gate, the owner gate, and vault control following the token are unasserted.
  <details><summary>details</summary>

  `gno test ./gno.land/r/zeycan1/tba` reports `[no test files]`, and the examples suite passes over the package for that reason alone. Writing them also surfaces the `init()` binding: under `gno test` the origin caller is empty, so `admin` is the empty address and no caller can match it; a test has to assign `admin` before any admin-gated call. The suite in [`tests/tba_test.gno`](tests/tba_test.gno) covers the admin gate, the owner gate, vault control across a transfer, the `selfPath`-versus-`cur.Sub` agreement, plus the two behaviors above; four pass at this commit and `TestWithdrawRejectsNonPositive` and `TestTransferFromAbortsWhenUnauthorized` fail.
  </details>

## Suggestions

- **[the balance a buyer sees is not the balance they get]** `examples/gno.land/r/zeycan1/tba/tba.gno:116-129` — `Withdraw` and `TransferFrom` are separate transactions, so the seller can empty the vault after a buyer reads `TokenInfo` and before the transfer lands.
  <details><summary>details</summary>

  Inherent to the pattern rather than a defect in this code: ERC-6551 has the same property. It is the one thing a reader of a token-bound-account example has to know before copying it, and neither the source nor the PR description mentions it. A sentence on `Withdraw` saying the vault is drainable by the current owner up to the moment of transfer covers it.
  </details>

- **[an example teaching a linear scan]** `examples/gno.land/r/zeycan1/tba/tba.gno:101-114` — `allTokens` only grows, and `TokensOf` runs one AVL `OwnerOf` lookup per minted token.
  <details><summary>details</summary>

  Minting is admin-gated, so the list is bounded by the admin's own behavior and no user can inflate it. The cost is that a realm published as a reference pattern shows a linear owner scan where a per-owner `avl.Tree` is the idiomatic shape. Not posted, no change needed for correctness.
  </details>

## Verified

- The owner gate holds across a transfer: on a live node the previous owner's `Withdraw` aborts with `caller is not the current owner of this token` after `TransferFrom`, and the new owner's succeeds against the same balance.
- Deposits reach the vault the realm advertises: a plain `maketx send` to `VaultAddress("1")` shows up in `Balance("1")` and is spendable by the token owner, so `chain.DerivePkgSubAddr(selfPath, ...)` and `cur.Sub(...)` name the same account at this path.
- A copy of this file deployed at a different package path strands deposits: `Balance` reports them, `Withdraw` aborts, and the coins sit in the original realm's vault.
- An unauthorized `TransferFrom` returns `caller is not token owner or approved` inside a transaction that reports `OK!`, with `OwnerOf` unchanged.
- Amount and recipient validation is the banker's alone: negative, overdraw, and empty-recipient calls all abort and revert; zero commits.
- Returning `allTokens` from `AllTokens()` is not a write handle: an external realm assigning into the returned slice is rejected by the readonly taint.
- `gno lint -C examples ./...` and `gno test -C examples ./gno.land/r/...` are green at this commit, and `gno mod tidy --recursive` leaves the manifest unchanged.

## Open questions

- `Setup`, `Mint`, `Withdraw` and `TransferFrom` all read `cur.Previous()` with no preceding `cur.IsCurrent()`. Not posted: the preprocessor rejects any realm value other than `cur` or `cross(rlm)` in that position and no forgeable path was demonstrated, matching the rest of `examples/`.
- `examples/gno.land/r/zeycan1/` is a valid namespace: [the `r/` README](https://github.com/gnolang/gno/blob/7d9a11104/examples/gno.land/r/README.md?plain=1#L10) · [↗](../../../../../.worktrees/gno-review-5983/examples/gno.land/r/README.md#L10) allows personal namespaces. Not posted: nothing for the author to change.
- `VaultAddress` and `Balance` panic on a token id that is not a valid subpath, e.g. an uppercase or empty string, rather than returning an error. Not posted: ids are generated by `Mint` as decimal digits, so only a hand-written query reaches it.
