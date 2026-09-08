# Review: [#6138](https://github.com/gnolang/gno/pull/6138)
Event: APPROVE

## Body

## examples/gno.land/p/demo/tokens/grc20/token_test.gno:207-212 [gh](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L207-L212) · [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L207-L212)
Missing test: [`TransferFrom`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L245-L273) returns seven errors and [`TestTransferFromAtomicity`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token_test.gno#L174) drives two, so a later refusal sliding back above [`SpendAllowance`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L264) reddens nothing.

<details><summary>test cases</summary>

The table fails on the `owner is recipient` row alone at the merge base and
passes at this head, so it is the net for the case this change closes and for
the five it leaves alone.

```gno
func TestTransferFromLeavesNoStateOnError(cur realm, t *testing.T) {
	owner := testutils.TestAddress("owner")
	spender := testutils.TestAddress("spender")
	to := testutils.TestAddress("to")

	const (
		balance   = int64(100)
		allowance = int64(50)
	)

	cases := []struct {
		name    string
		owner   address
		spender address
		to      address
		amount  int64
		want    error
	}{
		{"invalid to", owner, spender, address(""), 30, ErrInvalidAddress},
		{"invalid owner", address(""), spender, to, 30, ErrInvalidAddress},
		{"invalid spender", owner, address(""), to, 30, ErrInvalidAddress},
		{"owner is recipient", owner, spender, owner, 30, ErrCannotTransferToSelf},
		{"negative amount", owner, spender, to, -1, ErrInvalidAmount},
		{"amount over balance", owner, spender, to, balance + 1, ErrInsufficientBalance},
		{"amount over allowance", owner, spender, to, allowance + 1, ErrInsufficientAllowance},
	}

	for _, tc := range cases {
		// A fresh token per row, so one row's mutation cannot mask the next.
		tok, led := newTestToken("Atomic", "ATM", 6, 0, cur)
		urequire.NoError(t, led.Mint(owner, balance))
		urequire.NoError(t, led.Approve(owner, spender, allowance))

		err := led.TransferFrom(tc.owner, tc.spender, tc.to, tc.amount)
		uassert.Equal(t, tc.want.Error(), err.Error(), tc.name+": error")
		uassert.Equal(t, balance, tok.BalanceOf(owner), tc.name+": balance moved")
		uassert.Equal(t, allowance, tok.Allowance(owner, spender), tc.name+": allowance moved")
	}
}
```

Run it with:

```bash
go build -o /tmp/gno ./gnovm/cmd/gno
GNOROOT="$PWD" /tmp/gno test -C examples ./gno.land/p/demo/tokens/grc20 \
  -run TestTransferFromLeavesNoStateOnError
```

With `token.gno` reverted to the merge base:

```
--- FAIL: TestTransferFromLeavesNoStateOnError (0.02s)
uassert.Equal: same type but different value
	expected: 50
	actual:   20 - owner is recipient: allowance moved
failed: "TestTransferFromLeavesNoStateOnError"
FAIL    ./gno.land/p/demo/tokens/grc20 	3.84s
```
</details>

## SKIP examples/gno.land/p/demo/tokens/grc20/token.gno:262-263 [gh](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L262-L263) · [↗](../../../../../.worktrees/gno-review-6138/examples/gno.land/p/demo/tokens/grc20/token.gno#L262-L263)
Nit: two checks now guarantee that [`Transfer`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L196-L241) will succeed, the balance one and the [`owner == to`](https://github.com/junghoon-vans/gno/blob/fix/grc20-transferfrom-atomicity/examples/gno.land/p/demo/tokens/grc20/token.gno#L254-L256) one, and this comment credits one.

Skipped: the line sits outside both diff hunks, which end at `token.gno:260`, so a review cannot anchor it, and it changes no behaviour.
