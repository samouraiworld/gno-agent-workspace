# PR #5608: feat(gnokey): print pkgpath after `maketx addpkg`

URL: https://github.com/gnolang/gno/pull/5608
Author: davd-gzl | Base: master | Files: 5 | +37 -0
Reviewed by: davd-gzl | Model: claude-opus-4-7
Local worktree: `git -C gno worktree add .worktrees/gno-review-5608 5c57764` (then `gh -R gnolang/gno pr checkout 5608` inside it)

**Verdict: APPROVE** — small, well-scoped UX win for `addpkg`; only fixable items are a stale per-command callback override (now dead-equivalent to the root callback) and a docs-output mismatch missing the `INFO:` line.

## Summary

After a successful `gnokey maketx addpkg --broadcast` (or `gnokey broadcast` of a pre-signed tx), `PrintTxInfo` now iterates `tx.Msgs` and prints one `PKGPATH:    <path>` line per `MsgAddPackage`. Single source of truth: the tx itself. Correct for multi-msg broadcasts (`gnokey broadcast` of a hand-signed tx with several addpkgs), silent for `call`/`run`/`send`, and consistent across every entry point that funnels through `PrintTxInfo`. The change is six lines of code (one type-assert loop) plus a new txtar covering the multi-msg shape. The PR converged on the right design after iteration (previous draft hooked into `OnTxSuccess` from `addpkg.go`, this lives in `PrintTxInfo`).

## Glossary

- `PrintTxInfo`: single helper in `gno.land/pkg/keyscli/root.go` that formats the post-broadcast result (TX HASH, EVENTS, etc.). Wired as `OnTxSuccess` from `NewRootCmd`.
- `OnTxSuccess`: `BaseCfg` callback fired by `broadcast.go` and `maketx.go` when `!DeliverTx.IsErr()`.
- `MsgAddPackage`: the only `vm.Msg` type that creates a new on-chain package; registered as the amino concrete type `m_addpkg`.

## Fix

`PrintTxInfo` previously printed `OK!` / `GAS WANTED` / `…` / `TX HASH` for every successful broadcast and stopped there. With this PR it walks `tx.Msgs`, type-asserts each as `vm.MsgAddPackage`, and on a hit emits `PKGPATH:    <addPkg.Package.Path>` — preserving message order. See [`gno.land/pkg/keyscli/root.go:87-91`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L87-L91) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L87-L91). All other entry points (`call`, `run`, `send`) are untouched because their messages don't carry a freshly created path.

## Critical (must fix)

None.

## Warnings (should fix)

- **[docs misalign the actual output]** [`docs/users/interact-with-gnokey.md:217-225`](https://github.com/gnolang/gno/blob/5c57764/docs/users/interact-with-gnokey.md#L217-L225) · [↗](../../../../../.worktrees/gno-review-5608/docs/users/interact-with-gnokey.md#L217-L225) — sample output skips the `INFO:` line that `PrintTxInfo` actually prints between `EVENTS:` and `TX HASH:`.
  <details><summary>details</summary>

  The real output emitted by `PrintTxInfo` (see [`root.go:84-89`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L84-L89) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L84-L89)) is `EVENTS → INFO → TX HASH → PKGPATH`. The docs example shows `EVENTS → TX HASH → PKGPATH` (no `INFO`). The annotated bullet list at [interact-with-gnokey.md:229-234](https://github.com/gnolang/gno/blob/5c57764/docs/users/interact-with-gnokey.md#L229-L234) · [↗](../../../../../.worktrees/gno-review-5608/docs/users/interact-with-gnokey.md#L229-L234) inherits the same gap. The mismatch predates this PR but the same hunk that adds `PKGPATH` is the natural place to fix it. The other doc page, [`deploy-packages.md:97-105`](https://github.com/gnolang/gno/blob/5c57764/docs/builders/deploy-packages.md#L97-L105) · [↗](../../../../../.worktrees/gno-review-5608/docs/builders/deploy-packages.md#L97-L105), has the same omission. Fix: add an `INFO:       ` line to both example blocks (and the bullet list) — the simplest variant is a blank value, matching the integration test output where INFO is empty for non-dry-run broadcasts. Also flagged by [@davd-gzl in round 1](../1-211a804/claude-sonnet-4-6_davd-gzl.md#L40-L46).

  </details>

- **[per-command callback override is now dead-equivalent code]** [`gno.land/pkg/keyscli/addpkg.go:143-145`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/addpkg.go#L143-L145) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/addpkg.go#L143-L145) (and the identical blocks in `call.go:150-152`, `run.go:160-162`) reassigns `OnTxSuccess` to the same lambda already set by `NewRootCmd`.
  <details><summary>details</summary>

  [`root.go:38-40`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L38-L40) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L38-L40) already wires `cfg.OnTxSuccess = func(tx, res) { PrintTxInfo(tx, res, io) }` on every `NewRootCmd` invocation, so every subcommand (`maketx addpkg`, `maketx call`, `maketx run`, `broadcast`) inherits it. The per-command overrides in [`addpkg.go:143`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/addpkg.go#L143) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/addpkg.go#L143), [`call.go:150`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/call.go#L150) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/call.go#L150), and [`run.go:160`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/run.go#L160) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/run.go#L160) re-set the same lambda. Before this PR the `addpkg.go` override differed (it accepted an extra responsibility); now all three are byte-identical to the root assignment and serve only as a future foot-gun: a contributor editing `PrintTxInfo`'s callsite signature must remember to update four places, and a contributor refactoring the root callback away may forget the three subcommand overrides shadowing it. Fix: drop the three `if cfg.RootCfg.Broadcast { cfg.RootCfg.RootCfg.OnTxSuccess = … }` blocks; trust the root-level assignment. Also raised by [@davd-gzl in round 1](../1-211a804/claude-sonnet-4-6_davd-gzl.md#L64) (suggestion, but the duplication is concrete enough to warrant fixing).

  </details>

## Nits

- [`docs/users/interact-with-gnokey.md:234`](https://github.com/gnolang/gno/blob/5c57764/docs/users/interact-with-gnokey.md#L234) · [↗](../../../../../.worktrees/gno-review-5608/docs/users/interact-with-gnokey.md#L234) — "only printed for `addpkg`" is slightly imprecise: PKGPATH prints for every `MsgAddPackage` in a tx, including multi-msg `gnokey broadcast` of pre-signed txs that did not go through `maketx addpkg`. Tighter: "printed once for each `MsgAddPackage` in the transaction".
- [`gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar:23-24`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar#L23-L24) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar#L23-L24) — hardcoded `g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5` matches `$test1_user_addr` (set in [`testscript_gnoland.go:175`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/integration/testscript_gnoland.go#L175) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/integration/testscript_gnoland.go#L175) from [`node_testing.go:26`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/integration/node_testing.go#L26) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/integration/node_testing.go#L26)). A one-line comment above the JSON would save the next reader the cross-reference. Lower-priority than the `OnTxSuccess` cleanup because heredoc literals can't do `$var` substitution in testscript anyway. Also surfaced in [round 1](../1-211a804/claude-sonnet-4-6_davd-gzl.md#L54).
- [`gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar:13`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar#L13) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar#L13) — `-quiet=false` is the default for `BaseCfg.Quiet` ([`tm2/pkg/crypto/keys/client/root.go:74-79`](https://github.com/gnolang/gno/blob/5c57764/tm2/pkg/crypto/keys/client/root.go#L74-L79) · [↗](../../../../../.worktrees/gno-review-5608/tm2/pkg/crypto/keys/client/root.go#L74-L79)); the flag is redundant. Drop it. Surfaced in [round 1](../1-211a804/claude-sonnet-4-6_davd-gzl.md#L52).

## Missing Tests

- **[mixed-message tx not exercised]** [`gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/integration/testdata/addpkg_multi_msg.txtar) — the multi-msg txtar covers two `MsgAddPackage` only; the type-assert filter at [`root.go:88`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L88) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L88) is the load-bearing logic and a tx mixing `MsgAddPackage` + `MsgCall` (or + `MsgSend`) would prove the filter actually skips non-addpkg msgs.
  <details><summary>details</summary>

  Without it, a future change that, say, ANDs the assertion with `addPkg.Package != nil` and forgets to keep the type filter could regress to printing `PKGPATH:    ` for non-addpkg msgs, and no integration test would catch it. A txtar variant that pre-deploys a realm, then broadcasts a hand-crafted tx with `[MsgAddPackage, MsgCall]` and asserts exactly one PKGPATH line in the output would close the gap. Optional — the type assertion is straightforward enough that the maintenance burden of a third txtar may not be worth it.

  </details>

- **[unit test for `PrintTxInfo`]** [`gno.land/pkg/keyscli/root.go:65-92`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L65-L92) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L65-L92) — `PrintTxInfo` has no unit test. The PKGPATH output is exercised only via two slow txtar tests that boot a full node.
  <details><summary>details</summary>

  A table-driven test with a fake `ctypes.ResultBroadcastTxCommit` and an in-memory `commands.IO` could cover: (a) tx with one `MsgAddPackage` → one PKGPATH line, (b) tx with two `MsgAddPackage` → two lines in order, (c) tx with only `MsgCall` → no PKGPATH, (d) mixed `[MsgAddPackage, MsgCall, MsgAddPackage]` → two PKGPATH lines for the addpkg msgs only, in order, (e) storage-event present → `STORAGE DELTA` block appears before `PKGPATH`. Fast feedback, no node dependency. Lower priority than the dead-code cleanup. Echoes [round 1](../1-211a804/claude-sonnet-4-6_davd-gzl.md#L57-L58).

  </details>

## Suggestions

- [`gno.land/pkg/keyscli/root.go:87-91`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/root.go#L87-L91) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/root.go#L87-L91) — micro-style: the loop variable could be named `m` instead of `msg` for parity with the rest of the codebase (most `tx.Msgs` iterations in `gno.land/` use single-letter), and the type-assertion could use the more compact `addPkg, ok := msg.(vm.MsgAddPackage)` (already does — keep). Pure style, no action needed.

## Questions for Author

- Should `PKGPATH` be emitted by `maketx addpkg` even without `--broadcast` (i.e. in the JSON-dump path at [`addpkg.go:154`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/keyscli/addpkg.go#L154) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/keyscli/addpkg.go#L154))? The use case is offline review of a tx file before signing — the operator wants to see the deployed path one more time. The tx JSON already contains it, but a one-liner makes it copy-pasteable. Same question for `gnokey sign` (which emits "Tx successfully signed and saved to …" with no preview). Out of scope for this PR; flagging for a follow-up.
- Round 1 raised a nil-`Package` deref concern at `root.go:88`. I verified the path is unreachable: any tx with `Package=nil` panics in [`MsgAddPackage.ValidateBasic` at `msgs.go:57`](https://github.com/gnolang/gno/blob/5c57764/gno.land/pkg/sdk/vm/msgs.go#L57) · [↗](../../../../../.worktrees/gno-review-5608/gno.land/pkg/sdk/vm/msgs.go#L57), gets caught by the [`runTx` recover at `baseapp.go:760-789`](https://github.com/gnolang/gno/blob/5c57764/tm2/pkg/sdk/baseapp.go#L760-L789) · [↗](../../../../../.worktrees/gno-review-5608/tm2/pkg/sdk/baseapp.go#L760-L789), returns as `DeliverTx.Error`, and bails out of [`broadcast.go:84-86`](https://github.com/gnolang/gno/blob/5c57764/tm2/pkg/crypto/keys/client/broadcast.go#L84-L86) · [↗](../../../../../.worktrees/gno-review-5608/tm2/pkg/crypto/keys/client/broadcast.go#L84-L86) before reaching `OnTxSuccess`. No nil-check needed at the call site — `ValidateBasic` is the invariant. (Defensive `if addPkg.Package != nil` is cheap, but I'd argue it muddies which layer owns the invariant.)
