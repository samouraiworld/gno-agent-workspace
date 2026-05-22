/* Run:
   cp reviews/pr/4xxx/4008-address-prefix-check/1-32f0c57e/tests/adv_bech32_prefix_test.go \
      gno/tm2/pkg/crypto/adv_prefix_test.go
   go test -C gno -v -run TestAddressFromBech32_WrongPrefix ./tm2/pkg/crypto/
   rm -f gno/tm2/pkg/crypto/adv_prefix_test.go
*/
package crypto_test

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/crypto"
)

func TestAddressFromBech32_WrongPrefix(t *testing.T) {
	// bech32-valid but non-"g" prefix — covers the OpenZeppelin finding
	// that prompted PR #4008. AddressFromBech32 must reject these because
	// the gno-side `address.IsValid()` (uverse.go) delegates to it.
	cases := []string{
		"cosmos1qxy2gn0hu4xnq03tmgfhfd7zvc2hr5dxztpmcd",
		"bc156tlrfxxxelwrmvu0v986psjln9ry60ef34yp2",
	}
	for _, c := range cases {
		_, err := crypto.AddressFromBech32(c)
		if err == nil {
			t.Fatalf("expected error for %q (non-g prefix), got nil", c)
		}
	}
}
