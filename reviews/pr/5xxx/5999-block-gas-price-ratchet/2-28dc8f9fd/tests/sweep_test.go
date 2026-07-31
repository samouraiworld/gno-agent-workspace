/* Run: from a gno checkout:
gh pr checkout 5999 -R gnolang/gno && git checkout 28dc8f9fd
curl -fsSL -o tm2/pkg/sdk/auth/zz_sweep_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/2-28dc8f9fd/tests/sweep_test.go
ZZ_SWEEP_OUT=/tmp/sweep_head.txt go test -run TestZZSweep ./tm2/pkg/sdk/auth/
git checkout d1a33f574 -- tm2/pkg/sdk/auth/keeper.go
ZZ_SWEEP_OUT=/tmp/sweep_base.txt go test -run TestZZSweep ./tm2/pkg/sdk/auth/
git checkout HEAD -- tm2/pkg/sdk/auth/keeper.go
diff /tmp/sweep_base.txt /tmp/sweep_head.txt | head
rm tm2/pkg/sdk/auth/zz_sweep_test.go
*/

// Walks calcBlockGasPrice over a parameter grid and writes one line per case,
// recording a panic as a result like any other. Run once at the branch and once
// with keeper.go at the merge base, then diff: every differing line is a case the
// PR intends to change, and the diff is the whole list of them.

package auth

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
)

func zzOne(gk GasPriceKeeper, last std.GasPrice, gasUsed, maxGas int64, params Params) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fmt.Sprintf("PANIC(%v)", r)
		}
	}()
	got := gk.calcBlockGasPrice(last, gasUsed, maxGas, params)
	return fmt.Sprintf("%d/%d%s", got.Price.Amount, got.Gas, got.Price.Denom)
}

func TestZZSweep(t *testing.T) {
	path := os.Getenv("ZZ_SWEEP_OUT")
	if path == "" {
		t.Skip("set ZZ_SWEEP_OUT to the output path")
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gk := GasPriceKeeper{}
	maxGases := []int64{-1, 0, 1, 2, 3, 10, 100, 142, 143, 1_000, 100_000_000, 3_000_000_000, 30_000_000_000, math.MaxInt64}
	ratios := []int64{0, 1, 7, 50, 70, 99, 100}
	comps := []int64{1, 2, 10, 1000}
	inits := []int64{0, 1, 2, 1000}
	lasts := []int64{0, 1, 2, 3, 10, 1000, 1_000_000, math.MaxInt64 - 1, math.MaxInt64}

	cases := 0
	for _, maxGas := range maxGases {
		for _, ratio := range ratios {
			for _, c := range comps {
				for _, init := range inits {
					params := Params{
						TargetGasRatio:            ratio,
						GasPricesChangeCompressor: c,
						InitialGasPrice:           std.GasPrice{Gas: 1000, Price: std.Coin{Amount: init, Denom: "ugnot"}},
					}
					// target, computed the way the function does, then probed around.
					target := int64(0)
					if maxGas > 0 && maxGas < math.MaxInt64/100 {
						target = maxGas * ratio / 100
					}
					gasUseds := []int64{0, 1, target - 1, target, target + 1, maxGas, math.MaxInt64}
					for _, last := range lasts {
						for _, used := range gasUseds {
							if used < 0 {
								continue
							}
							lastGP := std.GasPrice{Gas: 1000, Price: std.Coin{Amount: last, Denom: "ugnot"}}
							fmt.Fprintf(f, "maxGas=%d ratio=%d c=%d init=%d last=%d used=%d -> %s\n",
								maxGas, ratio, c, init, last, used,
								zzOne(gk, lastGP, used, maxGas, params))
							cases++
						}
					}
				}
			}
		}
	}
	t.Logf("cases: %d", cases)
}
