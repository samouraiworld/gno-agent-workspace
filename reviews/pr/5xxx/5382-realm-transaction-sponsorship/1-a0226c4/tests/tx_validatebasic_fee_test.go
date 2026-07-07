/* Run: from a gno checkout:
gh pr checkout 5382 -R gnolang/gno && git checkout a0226c4
curl -fsSL -o tm2/pkg/std/zz_pr5382_validatebasic_fee_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5382-realm-transaction-sponsorship/1-a0226c4/tests/tx_validatebasic_fee_test.go
go test -v -run TestTxValidateBasicFeeMatrix ./tm2/pkg/std/
rm tm2/pkg/std/zz_pr5382_validatebasic_fee_test.go

Pins the zero-fee acceptance/rejection matrix the PR relaxes in Tx.ValidateBasic.
A canonical zero fee (empty Coin) or valid coin passes the fee gate; a malformed
coin (bad denom, empty denom with amount, negative amount) is rejected. Passes at
a0226c4; guards the relaxed check against a future regression to the looser form.
*/
package std

import (
	stderrors "errors"
	"testing"
)

// TestTxValidateBasicFeeMatrix exercises the zero-fee acceptance/rejection
// matrix introduced by PR 5382 (realm transaction sponsorship). ValidateBasic
// must accept a zero fee only in its canonical form (the empty zero-value Coin)
// or as an otherwise-valid coin, and must reject a malformed fee coin — notably
// a bad-denom / zero-amount coin, which the previous `!IsZero() && !IsValid()`
// check let through.
//
// The tx carries no signatures and no messages, so the fee gate (which runs
// first in ValidateBasic) decides the outcome: a rejected fee yields the
// "invalid fee" error; an accepted fee falls through to the later "no signers"
// error. Asserting on which error surfaces isolates the fee decision.
func TestTxValidateBasicFeeMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		fee        Coin
		wantReject bool // true = rejected AT the fee gate ("invalid fee")
	}{
		{"canonical zero-value coin", Coin{}, false},
		{"valid denom zero amount", Coin{Denom: "ugnot", Amount: 0}, false},
		{"valid denom positive amount", Coin{Denom: "ugnot", Amount: 100}, false},
		{"bad denom zero amount", Coin{Denom: "X", Amount: 0}, true},
		{"bad denom positive amount", Coin{Denom: "X", Amount: 100}, true},
		{"empty denom nonzero amount", Coin{Denom: "", Amount: 100}, true},
		{"negative amount valid denom", Coin{Denom: "ugnot", Amount: -5}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tx := Tx{Fee: Fee{GasWanted: 1, GasFee: tc.fee}}
			err := tx.ValidateBasic()
			if err == nil {
				t.Fatalf("expected a non-nil error (at least 'no signers'), got nil")
			}
			var feeErr InsufficientFeeError
			var noSigErr NoSignaturesError
			gotFeeReject := stderrors.As(err, &feeErr)
			if gotFeeReject != tc.wantReject {
				t.Fatalf("fee %+v: rejected-at-fee-gate=%v, want %v (err=%q)",
					tc.fee, gotFeeReject, tc.wantReject, err.Error())
			}
			// When accepted at the fee gate, the tx must fall through to the
			// signature checks — proving the fee was not the blocker.
			if !tc.wantReject && !stderrors.As(err, &noSigErr) {
				t.Fatalf("fee %+v accepted but did not reach signer check; err=%q",
					tc.fee, err.Error())
			}
		})
	}
}
