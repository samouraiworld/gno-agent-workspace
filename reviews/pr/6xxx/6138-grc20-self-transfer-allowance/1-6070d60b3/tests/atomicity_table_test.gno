// Asserts the invariant the fix rests on: every input that makes TransferFrom
// return an error leaves the owner balance and the spender allowance untouched.
// Seven rows, one per error the call or its callees can produce.
// Green at 6070d60b3; the "owner is recipient" row reads an allowance of 20
// against 50 at the merge base c29ca2629.
//
//   gh pr checkout 6138 -R gnolang/gno && git checkout 6070d60b3
//   cp atomicity_table_test.gno examples/gno.land/p/demo/tokens/grc20/
//   go build -o /tmp/gno ./gnovm/cmd/gno
//   GNOROOT="$PWD" /tmp/gno test -C examples ./gno.land/p/demo/tokens/grc20 \
//     -run TestTransferFromLeavesNoStateOnError

package grc20

import (
	"testing"

	"gno.land/p/nt/testutils/v0"
	"gno.land/p/nt/uassert/v0"
	"gno.land/p/nt/urequire/v0"
)

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
