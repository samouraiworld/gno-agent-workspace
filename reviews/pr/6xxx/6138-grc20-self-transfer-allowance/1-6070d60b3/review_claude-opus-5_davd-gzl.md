# PR [#6138](https://github.com/gnolang/gno/pull/6138): fix(grc20): preserve allowance on failed self transfer

URL: https://github.com/gnolang/gno/pull/6138
Author: junghoon-vans | Base: master | Files: 2 | +13 -0
Reviewed by: davd-gzl | Model: claude-opus-5, effort xhigh | Commit: `6070d60b3` (latest)
Local checkout: `git -C gno worktree add ../.worktrees/gno-review-6138 origin/master && cd ../.worktrees/gno-review-6138 && gh pr checkout 6138 -R gnolang/gno`
Overview: [overview](../overview.md)

## Overview

grc20's `TransferFrom` spends the spender's allowance and then moves the owner's
balance, and it validates ahead of both so that the second step cannot fail once
the first has run. One refusal was missing from that validation: `Transfer`
rejects a transfer whose sender and recipient are the same address, and nothing
upstream ruled that case out. A returned error does not roll a Gno transaction
back, so a caller that read the error without panicking kept a reduced allowance
against a transfer that never happened. This change inserts the missing check
above the two mutations and extends the package's atomicity test to cover it.

**Verdict: APPROVE** - the added check closes the last input for which
`SpendAllowance` ran ahead of a `Transfer` that would refuse, measured over the
whole case space, and no in-tree caller distinguishes the error whose precedence
it changes. 1 Missing test, 1 Nit.

## Verify first

- [`examples/gno.land/p/demo/tokens/grc20/token.gno:254-256`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256) - [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256) - the property is that no input reaching `SpendAllowance` can make `Transfer` refuse. Confirm it with [`tests/atomicity_table_test.gno`](tests/atomicity_table_test.gno), which drives all seven errors the call can produce and asserts the balance and the allowance are unmoved on each.

## Summary

At the merge base, `TransferFrom(owner, spender, owner, 30)` against a balance of
100 and an allowance of 50 returned `ErrCannotTransferToSelf` and left the
allowance at 20. The four lines added at
[`token.gno:254-256`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256) - [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256)
move that refusal above
[`SpendAllowance`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L264) - [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token.gno#L264),
so it now leaves the allowance at 50. The sibling ledger
[`grc721`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc721/token.gno#L313-L315)
already ordered the same rule this way, and grc20's
[`Mint`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L298-L326) and
[`Burn`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L329-L360)
validate fully before their first write, so the branch leaves no other call in
the package that mutates and then returns an error.

## Numbers

Seven inputs make `TransferFrom` return an error. The table is the allowance
read back after each call, against a starting balance of 100 and a starting
allowance of 50, from [`tests/atomicity_table_test.gno`](tests/atomicity_table_test.gno).

| Input | Error returned | Allowance at `c29ca2629` | At `6070d60b3` |
| --- | --- | --- | --- |
| `to` is the empty address | `ErrInvalidAddress` | 50 | 50 |
| `owner` is the empty address | `ErrInvalidAddress` | 50 | 50 |
| `spender` is the empty address | `ErrInvalidAddress` | 50 | 50 |
| `to == owner` | `ErrCannotTransferToSelf` | **20** | 50 |
| `amount` is -1 | `ErrInvalidAmount` | 50 | 50 |
| `amount` is 101 | `ErrInsufficientBalance` | 50 | 50 |
| `amount` is 51 | `ErrInsufficientAllowance` | 50 | 50 |

## Missing Tests

- **[the invariant is asserted for two of its seven inputs]** [`examples/gno.land/p/demo/tokens/grc20/token_test.gno:207-212`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L207-L212) - [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L207-L212) - `TestTransferFromAtomicity` drives the invalid recipient and the self transfer, so the next refusal that drifts above the mutations is caught by nothing.
  <details><summary>details</summary>

  The property the fix establishes is that every error `TransferFrom` returns
  leaves both the balance and the allowance where they were, and it holds for all
  seven inputs at this head. Two of them are asserted. A table over the set fails
  on exactly the `to == owner` row at the merge base and passes here, so it is the
  regression net for the case this change closes and for the five it does not
  touch. Fix: replace the two ad-hoc blocks with the table in
  [`tests/atomicity_table_test.gno`](tests/atomicity_table_test.gno).
  </details>

## Nits

- **[the comment counts one check where there are now two]** [`examples/gno.land/p/demo/tokens/grc20/token.gno:262-263`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L262-L263) - [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token.gno#L262-L263) - "The check above guarantees that Transfer will succeed" now rests on the balance check and the self-transfer check together, and a reader tracing the guarantee back finds one of the two.
  <details><summary>details</summary>

  The line sits outside both diff hunks, which end at `token.gno:260`, so it
  cannot be anchored in a review and its section in comment.md is `SKIP`ped.
  It changes no behaviour. Fix: make the subject plural, "The checks above".
  </details>

## Verified

Every run below was made in the reviewed worktree at `6070d60b3`, restored clean
afterwards, with the toolchain `go.mod` pins, go1.25.9.

- The branch's own test fails at the merge base and passes at the head, so it
  catches the bug it was written for. Reverting
  [`token.gno`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno) to
  `c29ca2629` and keeping the branch's test file reports
  `expected: 50 actual: 20 - allowance should not be reduced when self-transfer fails`.
- No input remains for which `SpendAllowance` runs ahead of a `Transfer` that
  refuses. [`tests/atomicity_table_test.gno`](tests/atomicity_table_test.gno)
  drives all seven and is green at this head; at `c29ca2629` exactly one row
  fails, `owner is recipient: allowance moved`, reading 20 against 50.
- No in-tree caller can see the error precedence change. Seven realms call a
  grc20 teller's `TransferFrom` and none inspects which error came back:
  `grep -rn 'ErrCannotTransferToSelf\|ErrInsufficientBalance' --include=*.gno
  examples/gno.land/r examples/quarantined/gno.land/r` returns nothing, so no
  realm in the tree branches on either name.
- The refactor pass rewrote `TransferFrom` to check the spender and the
  allowance, delegate every remaining refusal to `Transfer`, and spend the
  allowance last, 27 lines down to 17. Rejected: it reports
  `ErrInsufficientAllowance` where the current code reports
  `ErrInsufficientBalance` when both are short, measured with a balance of 10, an
  allowance of 5 and an amount of 20, so it does not carry the same behaviour.
- The static pass over the incoming branch found no non-ASCII byte in the diff,
  so no bidirectional override, zero-width character or homoglyph, and no change
  to the build, dependency, workflow, lockfile, manifest or container surface.
  The author is a `FIRST_TIME_CONTRIBUTOR`.
- `gno test -C examples ./gno.land/p/demo/tokens/grc20` is green at the head,
  3.06s.

## Open questions

- A caller passing `owner == to` together with a second failure now reads
  `ErrCannotTransferToSelf` where it read `ErrInsufficientBalance` before, since
  the new check sits above the balance check. Not posted: nothing in the tree
  branches on either value, and the call fails and mutates nothing either way.
