# PR [#6028](https://github.com/gnolang/gno/pull/6028): feat(grc20): token identity via a registry-owned id generator

URL: https://github.com/gnolang/gno/pull/6028
Author: jinoosss | Base: master | Files: 23 | +383 -133
Reviewed by: davd-gzl | Model: claude-opus-5 | Commit: 37cd55419 (latest)
Local worktree: `git -C gno worktree add ../.worktrees/gno-review-6028 37cd55419`

Round 2. Head advanced 37182a315 to 37cd55419: seven branch commits and a clean merge of master, `git show 37cd55419 --cc` printing no conflict-resolution hunk. The design changed rather than the details. `p/onbloc/identifier`, its sha256 plus cford32 code and the registry rekey are all gone; the generator now carries the registry's counter as a function value, and the issuer's package path is spliced into `Token.ID()`. Round 1 findings resolved: the doc claiming a `gen.PackagePath() == rlm.PkgPath()` guard is deleted; the `Token.ID()`-as-registry-key doc is corrected; the registry key is back to master's `rlmPath.symbol`, so the migration finding and the collision with [PR 6027](https://github.com/gnolang/gno/pull/6027) are moot, 6027 having been closed unmerged on 2026-08-03. Carried: the write handle on the registry's counter is narrowed from a copyable pointer to a call and stays open, the shared counter still makes an id depend on chain-wide mint order, and a registrable token still needs a `grc20reg` import. Two doc findings recur in new places.

**TL;DR:** A GRC20 token carries a text identifier that goes into every one of its events, and until now the creating realm picked the tail of that identifier itself, so one realm could hand two different tokens the same one. This change makes the token registry hand out the numbers instead, stamps the issuer's name into the identifier, and makes the registry accept only tokens numbered by itself.

**Verdict: NEEDS DISCUSSION** — the mechanism now survives the copy-replay attack behind the standing changes-requested, verified live and against fresh attempts, but [PR 6042](https://github.com/gnolang/gno/pull/6042) merged detection into master while [PR 6043](https://github.com/gnolang/gno/pull/6043) was closed unmerged, and the branch still has to merge a master that rewrote the same `NewToken` (5 Warnings, 3 Nits, 1 Missing test, 3 Suggestions).

## Verify first

- [`examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:32-34`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) — confirm that letting every realm drive the registry's counter is intended. Deploy a four-line realm that calls `IdentifierGenerator(cross(cur))` and loops `NextID()`, then read the tx receipt: `grc20reg` takes the storage diff for a write it never authorised.
- [`examples/gno.land/p/demo/tokens/grc20/token.gno:59`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) — confirm this is the identifier you want on every GRC20 event. Run `go test ./gno.land/pkg/integration/ -run 'TestTestdata/grc20_registry_emit' -v` and read the `Transfer`: the `token` attribute is `gno.land/r/demo/defi/foo20.FOO.gno.land/r/demo/defi/grc20reg:0000001`, 68 characters against master's 38.
- [`examples/gno.land/p/demo/tokens/grc20/token.gno:32`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L32) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L32) — decide what master's [`newtoken_event_filetest.gno`](https://github.com/gnolang/gno/blob/18018c6a3/examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_event_filetest.gno#L20-L21) becomes. It calls `NewToken` with the `seqid.ID` this signature deletes, and its whole premise is a caller reusing `0`.

## Summary

`Token.ID()` is the value a token puts in its `Transfer`, `Approval`, `Mint` and `Burn` events, and on master its trailing component is a `seqid.ID` the creating realm supplies, so a realm passing `0` twice gets two tokens announcing one identity. This round replaces the branch's hash-based scheme with [`grc20.IDGenerator`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L11-L14) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L11-L14), which holds a package path and a `func() string`, never a counter. [`grc20reg.IdentifierGenerator`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L32-L34) hands out a generator backed by [`nextIDSeq`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L38-L40) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L38-L40), so a copy of the struct copies the function and advances the same `idSeq`. That is what kills the copy-replay reproduction: two tokens built from `a, b := *gen, *gen` come back with different ids.

The second half is the identifier itself. [`NewToken` splices `gen.packagePath` into the id](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) ahead of the code, and [`validID`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L57-L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L57-L59) keeps `:` out of the code, so a realm that reads a registry-issued code off chain state and replays it verbatim still lands on a different id. That closes the last in-band forgery: a package path is built from [`Re_domain` and `Re_name`](https://github.com/gnolang/gno/blob/37cd55419/gnovm/pkg/gnolang/mempackage.go#L67-L70) · [↗](../../../../../.worktrees/gno-review-6028/gnovm/pkg/gnolang/mempackage.go#L67-L70), both lowercase `[a-z0-9]` plus `.`, `_` and `-`, and a symbol runs through the same [`validSlugText`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L98-L108) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L98-L108) charset the code does, so nothing a caller controls can introduce the separator. The registry key goes back to master's `rlmPath.symbol` and `slug` is now emitted and nothing else.

Reading order: `p/demo/tokens/grc20/idgenerator.gno`, `p/demo/tokens/grc20/token.gno`, `r/demo/defi/grc20reg/grc20reg.gno`, then the six call-site realms, then the tests, filetests and txtars.

## Diagram

```
  /r/probe                              /r/demo/defi/grc20reg
  ┌──────────────────────┐              ┌────────────────────────────┐
  │ IdentifierGenerator( │              │ idSeq seqid.ID             │  never leaves
  │   cross(cur))  ──────┼─────────────▶│ nextIDSeq func            │
  │                      │              └────────────┬───────────────┘
  │ gen = {packagePath:  │◀── returns ───────────────┘
  │        "…/grc20reg", │      the function value, not the counter
  │        nextID: ──────┼──┐
  │                    } │  │
  │                      │  │ ① g.NextID() from /r/probe's frame
  │ a, b := *gen, *gen   │  └──▶ declaring-realm borrow: idSeq++ commits on grc20reg
  │   both hold the same │
  │   function           │
  └──────────────────────┘
```

Edge ① is what round 1 flagged as a raw pointer and what this round still leaves open: it no longer forks the counter, and it still writes `grc20reg`.

## Fix

`NewToken` swaps its `id seqid.ID` parameter for `gen *IDGenerator`, rejects both `nil` and the externally constructible `&IDGenerator{}` by [checking the function field](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L49-L53) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L49-L53), and records `gen.packagePath` in the new [`identifierPath`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L93) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/types.gno#L92-L93) field. `Register` keeps master's origin prefix check and adds [`token.IdentifierPath() != cur.PkgPath()`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69). The load-bearing constraint is that only `grc20reg` can build a generator carrying `grc20reg`'s path, because [`NewIDGenerator`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L24-L39) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L24-L39) reads the path off a live `IsCurrent` frame. [`&Token{`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L61) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L61) appears once in the package, so `NewToken` is the only way an id comes into existence.

## Examples

| Written as | Before this PR | After |
|---|---|---|
| `foo20` token id | `gno.land/r/demo/defi/foo20.FOO.0000000` | `gno.land/r/demo/defi/foo20.FOO.gno.land/r/demo/defi/grc20reg:0000001` |
| second token from the same realm, self-issued | `gno.land/r/probe.DUP.0000000`, duplicate reachable | `gno.land/r/probe.DUP.gno.land/r/probe:0000001` |
| `foo20` registry key | `gno.land/r/demo/defi/foo20.FOO` | unchanged |
| `fqname.Parse` of a token id | `gno.land/r/demo/defi/foo20` and `FOO.0000000` | whole string, empty name |

## Warnings (should fix)

- **[any realm can still write registry state]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:32-34` — `IdentifierGenerator` hands every realm a call into `grc20reg`'s own counter, so a realm that builds no token and registers nothing advances `idSeq` and bills the storage diff to `grc20reg`.
  <details><summary>details</summary>

  The generator no longer carries the counter, so the copy-replay path is gone, but the write itself is unchanged in kind. [`NextID`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L45-L52) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L45-L52) calls the function value, `nextIDSeq` was declared in `grc20reg`, and the declaring-realm borrow puts the write on `grc20reg` wherever the call happens. The [doc for `IdentifierGenerator`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30-L31) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L30-L31) says holding a generator grants no more than calling the function again would, which is true and is also the point: the grant is unconditional and the registry cannot rate-limit, meter or revoke it.

  Verified live: a four-line realm calling `IdentifierGenerator(cross(cur))` and looping `NextID()` fifty times, creating no token and registering nothing, succeeds, and the receipt carries `{"bytes_delta":10,…,"pkg_path":"gno.land/r/demo/defi/grc20reg"}`, [repro](comment_claude-opus-5.md). The reachable damage is bounded, since the only state is a monotonic `uint64` and the attacker pays the gas and the deposit, so the cost is a burnt id range rather than a collision. What is unbounded is the shape: every future method on `IDGenerator` inherits the same grant, and the counter is now the single thing standing between two registered tokens and one identity. Fix: issue codes through a crossing function on `grc20reg` that returns the code itself, so no call into the registry's counter is left in a caller's hands.
  </details>

- **[the doc's own example id is not the id the code builds]** `examples/gno.land/p/demo/tokens/grc20/token.gno:15-16` — the `NewToken` doc says `Token.ID()` reads `gno.land/r/demo/defi/foo20.FOO.0000001`; the line that builds it splices the issuer's package path in, and the live event for that exact realm reads `gno.land/r/demo/defi/foo20.FOO.gno.land/r/demo/defi/grc20reg:0000001`.
  <details><summary>details</summary>

  [The doc block](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L15-L16) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L15-L16) and [line 59](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) sit forty lines apart in the same function and disagree, and the example is the concrete `foo20` case the branch's own txtar pins. Round 1 raised two findings of this class against the previous design, both since deleted; this is the same defect in the rewritten text, and the id shape is the one thing every event consumer has to get right. Confirmed behaviorally against the `grc20_registry_emit` golden at 37cd55419. Fix: put the id the code produces in the example.
  </details>

- **[master already moved under this branch]** `examples/gno.land/p/demo/tokens/grc20/token.gno:32` — master merged the `NewToken` event and new `grc20reg` surface after this branch's base, and one of the files it added is written against the `seqid.ID` parameter this signature deletes.
  <details><summary>details</summary>

  The base here is [`abcd20dad`](https://github.com/gnolang/gno/commit/abcd20dad). Two commits landed after it on the same files. [`18018c6a3`](https://github.com/gnolang/gno/commit/18018c6a3) adds a `NewToken` event emitted at the end of `NewToken`, updates the `token_identity_filetest.gno` golden this branch rewrites, and adds [`newtoken_event_filetest.gno`](https://github.com/gnolang/gno/blob/18018c6a3/examples/gno.land/p/demo/tokens/grc20/filetests/newtoken_event_filetest.gno#L20-L21), a file absent from this worktree, whose two calls pass `0` where this branch takes a generator, and whose whole premise is that a caller can reuse an id. [`96c3cee24`](https://github.com/gnolang/gno/commit/96c3cee24) adds 39 lines to `grc20reg.gno`, 85 to `grc20reg_test.gno` and retunes the same two coin values in `storage_deposit_price_change.txtar` this branch retunes.

  So the next merge is not mechanical: it has to decide what the duplicate-id filetest becomes under a design that removes the caller-supplied id, keep the `NewToken` emission inside the rewritten constructor, and re-derive the storage numbers from both retunes rather than pick one. Fix: merge master and settle the filetest before the next review pass, since its outcome is the clearest statement of what this design leaves reachable.
  </details>

- **[a test change that does not fix the test]** `gno.land/pkg/gnoland/node_initial_height_test.go:26` — this raises the genesis initial height from 100 to 101 in a Go node-boot test the rest of the branch does not touch, and the flake it targets is master's own and survives the change.
  <details><summary>details</summary>

  The failure is real: `main / test` was red at [`bbc45d8e8`](https://github.com/gnolang/gno/commit/bbc45d8e8) with `expected: 100, actual: 101`. It is not this branch's. `TestNodeBootWithInitialHeight` also fails on master, at [`1bf8b2826`](https://github.com/gnolang/gno/commit/1bf8b2826) and [`fe2a4b8e9`](https://github.com/gnolang/gno/commit/fe2a4b8e9), and the test boots from `DefaultGenState` with no examples loaded, so no `examples/` diff can reach it.

  The cause is that [the height read](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/gnoland/node_initial_height_test.go#L77-L79) · [↗](../../../../../.worktrees/gno-review-6028/gno.land/pkg/gnoland/node_initial_height_test.go#L77-L79) races block production: `Ready()` returns [`firstBlockSignal`](https://github.com/gnolang/gno/blob/37cd55419/tm2/pkg/bft/node/node.go#L758-L760) · [↗](../../../../../.worktrees/gno-review-6028/tm2/pkg/bft/node/node.go#L758-L760) and the node keeps committing afterwards, so a loaded runner reads one block too many. Inserting `time.Sleep(3 * time.Second)` before the read at the branch's own 101 returns 3002. Locally the original 100 passes ten times alone and again across the whole package. Fix: take it out of this branch, since a grc20 change squash-merges into one commit and this would land in `git blame` on a node test it has nothing to do with.
  </details>

- **[a token's id depends on what the rest of the chain minted first]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:19` — one counter serves every realm, so `foo20.FOO.…:0000001` means the first token the registry ever issued rather than `foo20`'s first, and the same realm deployed on two chains gets two different ids.
  <details><summary>details</summary>

  Round 1 raised this against the hashed form. The plain sequence makes it easier to misread, because a per-realm counter is what the shape suggests. The branch already pays for it: [`grc20factory_test.gno`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20factory/grc20factory_test.gno#L41-L44) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20factory/grc20factory_test.gno#L41-L44) gave up asserting exact ids and now asserts a prefix and a difference, with a comment saying the suffix depends on creation order. The consequence is that nothing off chain, from docs to indexer fixtures to front-end config, can key on a token id without also pinning the chain, which is a defensible trade for one global sequence and is stated nowhere. Fix: say in the `grc20reg` doc that an id is assigned at deployment and is specific to the chain.
  </details>

## Nits

- **[the comment names a guarantee the next line spends]** `examples/gno.land/p/demo/tokens/grc20/idgenerator.gno:54-56` — [`validID`'s comment](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L54-L56) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/idgenerator.gno#L54-L56) says banning `.` and `/` keeps ids unambiguous for downstream parsers, and [`NewToken`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L59) then concatenates a package path carrying both. The ban is still doing real work, on `:` and on the code's own shape, and it is not the work the comment claims. Confirmed behaviorally: [`fqname.Parse`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/nt/fqname/v0/fqname.gno#L17-L43) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/nt/fqname/v0/fqname.gno#L17-L43) splits master's `gno.land/r/demo/defi/foo20.FOO.0000000` into `gno.land/r/demo/defi/foo20` and `FOO.0000000`, and returns the branch's id whole with an empty name.
- **[the id shape given without the part that carries the provenance]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:43` — [the `Register` doc](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L43) and [the comment above the prefix check](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L60-L61) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L60-L61) both write the id as `rlmPath.symbol.<id>`, leaving out the issuer segment that is the reason the check two lines below exists.
- **[a function described as a closure]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:36-37` — [`nextIDSeq`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36-L40) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L36-L40) is a package-level function reading a package-level var, and both it and [the `idSeq` comment](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L19) call it a closure over the counter. What makes the write land on `grc20reg` is where the function was declared, which is what the second half of the same sentence says.

## Missing Tests

- **[the property everything rests on is pinned only at genesis]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno:73-89` — no test shows the registry's counter surviving a transaction boundary, so the one thing that makes two registry-issued ids differ is unasserted outside genesis.
  <details><summary>details</summary>

  [`TestCopyingTheGeneratorCannotReplayAnID`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73-L89) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg_test.gno#L73-L89) is the reproduction of [issue 6026](https://github.com/gnolang/gno/issues/6026) and every other test in the file runs inside one transaction, where `idSeq` never has to persist. The only multi-transaction coverage is [`grc20_registry_emit.txtar`](https://github.com/gnolang/gno/blob/37cd55419/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19) · [↗](../../../../../.worktrees/gno-review-6028/gno.land/pkg/integration/testdata/grc20_registry_emit.txtar#L19), whose only token is minted at genesis by `foo20`'s `init`. The write is cross-realm and lands on `grc20reg` through the declaring-realm borrow, which is exactly the kind of thing a VM change can alter without any unit test noticing. The ready-to-add test is [`tests/idgen_across_transactions.txtar`](tests/idgen_across_transactions.txtar); it mints from a foreign realm in two separate transactions and passes at 37cd55419.
  </details>

## Suggestions

- **[every event pays 30 characters for a constant]** `examples/gno.land/p/demo/tokens/grc20/token.gno:59` — a registered token's id grows from 38 to 68 characters, and the added `gno.land/r/demo/defi/grc20reg:` is by construction the same for every registered token, since [`Register`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L67-L69) rejects anything else.
  <details><summary>details</summary>

  [The comment above it](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/p/demo/tokens/grc20/token.gno#L54-L58) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/p/demo/tokens/grc20/token.gno#L54-L58) gives the reason and it holds: events carry only the id, so the issuer has to travel in it or a self-issued token replaying a registry code is indistinguishable. The information it carries is one bit for the registered population, registry-issued against self-issued, and it is paid on every `Transfer`, `Approval`, `Mint` and `Burn` of every token forever. The alternative is a second event attribute, which master's newly merged `NewToken` event makes cheaper than it was: the issuer could be announced once at construction rather than repeated per transfer.
  </details>

- **[a guard the same function's other reader has]** `examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno:58` — [`Register`](https://github.com/gnolang/gno/blob/37cd55419/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58) · [↗](../../../../../.worktrees/gno-review-6028/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L58) reads `cur.Previous().PkgPath()` with no `cur.IsCurrent()` before it, while the new check nine lines down reads `cur.PkgPath()` and depends on the same frame being genuine.
  <details><summary>details</summary>

  This matches master and no exploit path exists: the preprocessor rejects any realm value other than `cur` or `cross(rlm)` in that position and `cross()` validates `IsCurrent` itself, which is why [the invariant catalog's `current-guard` rule](https://github.com/samouraiworld/gno-agent-workspace/blob/main/skills/invariant-catalog.md?plain=1#L54-L56) treats a bare hit as unproven. It is worth a line now rather than later because the frame is what the trusted-issuer check reduces to.
  </details>

- **[building a registrable token needs a registry import]** `examples/gno.land/p/demo/tokens/grc20/token.gno:27` — the only way to get a token `grc20reg` will accept is to import `/r/demo/defi/grc20reg` and fetch its generator, so a `/p/` standard's useful contract runs through one named realm.
  <details><summary>details</summary>

  Raised by [@moul](https://github.com/gnolang/gno/pull/6028#pullrequestreview-3583044847) and unchanged by the rewrite, which addressed the other half of the same point by moving `IDGenerator` into `grc20` itself rather than importing `p/onbloc/identifier`. No realm gains an import it did not already have, since each was already calling `grc20reg.Register`, and `grc20` stays compilable without `grc20reg`. Where it shows is [tokenhub's how-to-register snippet](https://github.com/gnolang/gno/blob/37cd55419/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/render.gno#L59-L70) · [↗](../../../../../.worktrees/gno-review-6028/examples/quarantined/gno.land/r/matijamarjanovic/tokenhub/render.gno#L59-L70), which now tells a reader to import a registry in order to build a token.
  </details>

## Verified

- The copy-replay reproduction is dead on a live node, not only under `gno test`. Two `Mint` calls from a foreign realm in two separate transactions returned `gno.land/r/idprobe.AAA.gno.land/r/demo/defi/grc20reg:0000001` and `gno.land/r/idprobe.BBB.gno.land/r/demo/defi/grc20reg:0000002`, so the counter both persists across a transaction boundary and advances from another realm's frame, [`tests/idgen_across_transactions.txtar`](tests/idgen_across_transactions.txtar).
- A realm holding no token and registering nothing advanced `grc20reg`'s sequence fifty times and put `{"bytes_delta":10,…,"pkg_path":"gno.land/r/demo/defi/grc20reg"}` on the receipt, so the function value is a live write handle rather than a read-only draw.
- Reverting the branch's own `TestNodeBootWithInitialHeight` constant to master's 100 passes ten consecutive runs and again across the whole `gno.land/pkg/gnoland` package, while a three-second sleep before the height read at 101 returns 3002.
- The `gno lint` run over `p/demo/tokens/grc20`, `r/demo/defi/grc20reg` and `r/demo/defi/grc20factory` really typechecks rather than exiting quietly, confirmed by replacing `MaxIDLen` with a bogus constant and seeing `undefined: MaxIDLenBogusXYZ (code=gnoTypeCheckError)`.
- Green at 37cd55419: `p/demo/tokens/grc20`, `r/demo/defi/grc20reg`, `r/demo/defi/grc20factory`, `p/nt/treasury/v0`, `r/gov/dao/v3/treasury/test`, the three quarantined consumers, the `grc20_registry_emit` and `storage_deposit_price_change` txtars, and the test under [`tests/`](tests/).

## Existing threads

- [@moul](https://github.com/gnolang/gno/pull/6028#pullrequestreview-3583044847), CHANGES_REQUESTED, open. Five objections, four now answered by the rewrite: the copy-replay defeat, the registry rekey, the hash carrying no weight, and the two incompatible usage patterns. The fifth, that a token realm should not need a hard dependency on `grc20reg`, is unchanged and overlaps the third Suggestion. The review closes by redirecting to [PR 6042](https://github.com/gnolang/gno/pull/6042) and [PR 6043](https://github.com/gnolang/gno/pull/6043); since then 6042 merged as [`18018c6a3`](https://github.com/gnolang/gno/commit/18018c6a3) and 6043 was closed unmerged by its own author.
- [@Villaquiranm](https://github.com/gnolang/gno/pull/6028#discussion_r3726703656), answered by the author, no overlap with these findings.

## Open questions

- Detection is now on master and prevention is not, so the question this PR has to answer is no longer the one it was opened with. `NewTokenEvent` makes every construction announceable and gives an indexer a complete rule; this branch stops the duplicate from existing for registered tokens and leaves it fully reachable for self-issued ones, which the merged filetest continues to demonstrate. Not posted: which of the two the project wants is a maintainer call, and the author cannot act on it alone.
- `slug` survives as a parameter that is validated and emitted and keys nothing, which is master's behaviour too. Worth folding into whatever settles the registry's lookup surface, together with the wrappers [PR 5962](https://github.com/gnolang/gno/pull/5962) just added. Deferred scope, no decision needed here.
