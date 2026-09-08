# grc20 `TransferFrom`, and the allowance it spends before it transfers

Model: claude-opus-5

[`TransferFrom`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L245-L273)
is the delegated-spend call of the grc20 token package. An owner grants a
spender an allowance with
[`Approve`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L276-L295), and
the spender later moves the owner's tokens by calling
`TransferFrom(owner, spender, to, amount)`. The call does two things that must
either both happen or neither: it reduces the allowance, and it moves the
balance.

## An error is not a rollback

On most chains a failed token call reverts every state change it made. Gno
splits the two. A panic aborts the transaction and rolls the store back, while a
returned `error` is an ordinary value, so every write made before it stands. A
Gno function that mutates state and then returns an error has committed that
mutation, and only a caller who panics on the error undoes it.

Every in-tree realm calling this method panics on it today.
[`wugnot`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L101),
[`foo20`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/demo/defi/foo20/foo20.gno#L50),
[`grc20factory`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/demo/defi/grc20factory/grc20factory.gno#L99)
and
[`grc20reg`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/demo/defi/grc20reg/grc20reg.gno#L138)
route it through a
[`checkErr`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/gnoland/wugnot/wugnot.gno#L110-L114)
helper,
[`atomicswap`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/r/demo/defi/atomicswap/atomicswap.gno#L117-L118)
wraps it in `require`, and
[`eventix`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/quarantined/gno.land/r/jjoptimist/eventix/eventix.gno#L118-L120)
panics with the error's own text. A realm that logs the error and carries on is
what the ordering below costs.

## The order of operations

`TransferFrom` delegates the two mutations to
[`SpendAllowance`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L165-L193)
and
[`Transfer`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L196-L241),
in that order, and validates ahead of them so neither can fail once the first
has run. Its own
[comment](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L262-L263)
states the contract: the checks above guarantee that `Transfer` will succeed.

Before, at merge base `c29ca2629`:

```
amount < 0              -> ErrInvalidAmount
owner or to invalid     -> ErrInvalidAddress
balance(owner) < amount -> ErrInsufficientBalance
SpendAllowance(owner, spender, amount)   <- the allowance drops here
Transfer(owner, to, amount)              <- and this can still fail
```

After, at head `6070d60b3`, with
[one check](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256)
inserted before the pair:

```
amount < 0              -> ErrInvalidAmount
owner or to invalid     -> ErrInvalidAddress
owner == to             -> ErrCannotTransferToSelf
balance(owner) < amount -> ErrInsufficientBalance
SpendAllowance(owner, spender, amount)
Transfer(owner, to, amount)              <- cannot fail
```

## Which of Transfer's refusals the caller had already excluded

`Transfer` refuses five ways. The `from == to` row is the one that reached the
mutating pair before this change, because nothing upstream ruled it out.

| `Transfer` refuses when | Excluded before `SpendAllowance` at `c29ca2629` | At `6070d60b3` |
| --- | --- | --- |
| `from` is not a valid address | yes | yes |
| `to` is not a valid address | yes | yes |
| `from == to` | no | yes |
| `amount < 0` | yes | yes |
| `balance(from) < amount` | yes | yes |

`SpendAllowance` writes
[allowances](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L183-L190)
and never balances, so the balance the caller measured is still the balance
`Transfer` reads. One check on the third row therefore closes the gap, rather
than the whole table having to move.

## What the owner and the spender saw

The owner holds 100, the spender holds an allowance of 50, and the spender calls
`TransferFrom(owner, spender, owner, 30)`.

| | `c29ca2629` | `6070d60b3` |
| --- | --- | --- |
| returned error | `ErrCannotTransferToSelf` | `ErrCannotTransferToSelf` |
| `BalanceOf(owner)` | 100 | 100 |
| `Allowance(owner, spender)` | 20 | 50 |

The call reports failure both times. Only the allowance differs, and only for a
caller that reads the error without panicking.
