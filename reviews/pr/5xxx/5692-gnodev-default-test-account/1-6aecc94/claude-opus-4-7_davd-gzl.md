# PR #5692: docs(gnodev): default test account in local dev with gnodev doc

URL: https://github.com/gnolang/gno/pull/5692
Author: jefft0 | Base: master | Files: 1 | +28 -3
Reviewed by: davd-gzl | Model: claude-opus-4-7 | Commit: `6aecc94` (latest)
Local worktree: `git -C gno worktree add .worktrees/gno-review-5692 6aecc94`

**Verdict: APPROVE** — small, correct docs-only addition that surfaces the publicly-known test mnemonic in `docs/builders/local-dev-with-gnodev.md`; minor redundancy between the new `Security note` subsection and the existing `[^1]` footnote, and the `(registered in r/sys/users)` parenthetical is not load-bearing for a fresh local dev node — both nits, not blockers.

## Summary

Closes [issue #5077](https://github.com/gnolang/gno/issues/5077). Follow-on to closed [PR #5110](https://github.com/gnolang/gno/pull/5110) by @moonia, picked up by @jefft0 with the review feedback already applied (warning consolidated into the footnote, "testnet GNOT" -> "locally usable GNOT", `test1` keybase name added, "let's dive into a practical example" moved into the `Practical example` section). The diff adds a `Default test account` section between `Hot reload` and `Practical example` listing the address, mnemonic, keybase name, and a security note; the existing footnote `[^1]` is also reworded to make the warning explicit.

Mnemonic, address, and name all match the canonical constants in [`gno.land/pkg/integration/node_testing.go:25-27`](https://github.com/gnolang/gno/blob/6aecc94/gno.land/pkg/integration/node_testing.go#L25-L27) · [↗](../../../../../.worktrees/gno-review-5692/gno.land/pkg/integration/node_testing.go#L25-L27) (`DefaultAccount_Name = "test1"`, `DefaultAccount_Address = "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5"`, `DefaultAccount_Seed = "source bonus chronic ..."`).

## Fix

Before: the doc only acknowledged "the mnemonic phrase for this address is publicly known" in a footnote — readers had to look elsewhere (e.g. `gno.land/pkg/integration/node_testing.go`) for the actual mnemonic. After: address, mnemonic, and `test1` name are inline in a dedicated section ([`docs/builders/local-dev-with-gnodev.md:98-120`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L98-L120) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L98-L120)), and the footnote is upgraded to a strong "never on production / real funds" warning ([`docs/builders/local-dev-with-gnodev.md:234-236`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L234-L236) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L234-L236)).

## Critical (must fix)

None.

## Warnings (should fix)

None.

## Nits

- [`docs/builders/local-dev-with-gnodev.md:116-120`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L116-L120) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L116-L120) — `Security note` subsection duplicates the `[^1]` footnote one line above the same page; either drop the subsection or drop the footnote. Author already raised this in [discussion r3273738164](https://github.com/gnolang/gno/pull/5692#discussion_r3273738164). I'd keep the subsection (it's visible in the body where readers will copy-paste the mnemonic) and shorten the footnote to just the address-as-deployer reference.
- [`docs/builders/local-dev-with-gnodev.md:108`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L108) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L108) — `Name: test1 (registered in r/sys/users)` is misleading for a fresh `gnodev` run. The `test1` name comes from the local `gnokey` keybase if the user has it imported (see [`contribs/gnodev/setup_address_book.go:43-52`](https://github.com/gnolang/gno/blob/6aecc94/contribs/gnodev/setup_address_book.go#L43-L52) · [↗](../../../../../.worktrees/gno-review-5692/contribs/gnodev/setup_address_book.go#L43-L52)); otherwise gnodev synthesizes `_default#<hash>`. And `test1` is not registered in `r/sys/users` by default — `gno.land/genesis/genesis_txs.jsonl` registers names like `gfanton123`, `moul001`, etc., but not `test1`. Suggest: `Name: test1 (the canonical keybase name across gno tooling)`.
- [`docs/builders/local-dev-with-gnodev.md:101`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L101) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L101) — "automatically pre-funded with locally usable GNOT" reads slightly awkward. "automatically pre-funded with GNOT on the local node" or "pre-funded with test GNOT on the local dev node" scans cleaner without dragging back the "testnet" word.

## Missing Tests

None — docs-only change, no test coverage applicable.

## Suggestions

- [`docs/builders/local-dev-with-gnodev.md:104-108`](https://github.com/gnolang/gno/blob/6aecc94/docs/builders/local-dev-with-gnodev.md#L104-L108) · [↗](../../../../../.worktrees/gno-review-5692/docs/builders/local-dev-with-gnodev.md#L104-L108) — link `test1` to the source constants so future drift is caught at review time: `Name: test1 — defined as DefaultAccount_Name in gno.land/pkg/integration/node_testing.go`. Optional; the mnemonic is already long enough to grep for.

## Questions for Author

- The author already asked in [r3273738164](https://github.com/gnolang/gno/pull/5692#discussion_r3273738164) whether the `Security note` subsection should be dropped given the footnote already carries the warning. My take: keep the subsection (it surfaces inline at the point of copy-paste), shorten the footnote to just the deployer-address reference. Either way, pick one — not both.
