/* Run: from a gno checkout:
gh pr checkout 5999 -R gnolang/gno && git checkout 29fb53a1e
curl -fsSL -o tm2/pkg/sdk/auth/zz_differential_sim_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5999-block-gas-price-ratchet/1-29fb53a1e/tests/differential_sim_test.go
go test -v -run 'TestZZ' ./tm2/pkg/sdk/auth/
rm tm2/pkg/sdk/auth/zz_differential_sim_test.go
*/

// oldCalc is master's calcBlockGasPrice at d14a03770; TestZZDifferential runs it
// against the branch's version over 43632 parameter combinations and classifies
// every divergence as the new non-positive-target guard, the new max(init,1)
// floor, or unexplained. At 29fb53a1e the unexplained count is 0, so a chain with
// a positive target and a non-zero initial price is output-identical to master.
// TestZZTrajectory reports the 5000-block price path under nine gas patterns.

package auth

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/std"
)

// oldCalc is master's calcBlockGasPrice at d14a03770, verbatim.
func oldCalc(lastGasPrice std.GasPrice, gasUsed int64, maxGas int64, params Params) std.GasPrice {
	if lastGasPrice.Price.Amount == 0 {
		return lastGasPrice
	}
	if params.TargetGasRatio == 0 {
		return lastGasPrice
	}
	var (
		num   = new(big.Int)
		denom = new(big.Int)
	)
	num.Mul(big.NewInt(maxGas), big.NewInt(params.TargetGasRatio))
	num.Div(num, big.NewInt(int64(100)))
	targetGasInt := new(big.Int).Set(num)

	gasUsedInt := big.NewInt(gasUsed)
	if targetGasInt.Cmp(gasUsedInt) == 0 {
		return lastGasPrice
	}

	c := params.GasPricesChangeCompressor
	lastPriceInt := big.NewInt(lastGasPrice.Price.Amount)

	bigOne := big.NewInt(1)
	if gasUsedInt.Cmp(targetGasInt) == 1 {
		num = num.Sub(gasUsedInt, targetGasInt)
		num.Mul(num, lastPriceInt)
		num.Div(num, targetGasInt)
		num.Div(num, denom.SetInt64(c))
		diff := maxBig(num, bigOne)
		num.Add(lastPriceInt, diff)
	} else {
		initPriceInt := big.NewInt(params.InitialGasPrice.Price.Amount)
		if lastPriceInt.Cmp(initPriceInt) == -1 {
			return params.InitialGasPrice
		}
		num.Sub(targetGasInt, gasUsedInt)
		num.Mul(num, lastPriceInt)
		num.Div(num, targetGasInt)
		num.Div(num, denom.SetInt64(c))
		diff := maxBig(num, bigOne)
		num.Sub(lastPriceInt, diff)
		num = maxBig(num, initPriceInt)
	}
	if !num.IsInt64() {
		panic("The min gas price is out of int64 range")
	}
	lastGasPrice.Price.Amount = num.Int64()
	return lastGasPrice
}

type outcome struct {
	amount   int64
	panicked bool
	msg      string
}

func runOld(gp std.GasPrice, gasUsed, maxGas int64, p Params) (o outcome) {
	defer func() {
		if r := recover(); r != nil {
			o.panicked = true
			o.msg = fmt.Sprint(r)
		}
	}()
	o.amount = oldCalc(gp, gasUsed, maxGas, p).Price.Amount
	return
}

func runNew(gp std.GasPrice, gasUsed, maxGas int64, p Params) (o outcome) {
	gk := GasPriceKeeper{}
	defer func() {
		if r := recover(); r != nil {
			o.panicked = true
			o.msg = fmt.Sprint(r)
		}
	}()
	o.amount = gk.calcBlockGasPrice(gp, gasUsed, maxGas, p).Price.Amount
	return
}

func mkParams(ratio, compressor, initAmount int64) Params {
	return Params{
		TargetGasRatio:            ratio,
		GasPricesChangeCompressor: compressor,
		InitialGasPrice:           std.GasPrice{Gas: 1000, Price: std.Coin{Amount: initAmount, Denom: "ugnot"}},
	}
}

func mkPrice(a int64) std.GasPrice {
	return std.GasPrice{Gas: 1000, Price: std.Coin{Amount: a, Denom: "ugnot"}}
}

// TestZZDifferential compares master vs PR head over a wide parameter sweep and
// classifies every divergence.
func TestZZDifferential(t *testing.T) {
	maxGases := []int64{-1, 0, 1, 2, 3, 10, 100, 142, 143, 1000, 100_000_000, 3_000_000_000, 30_000_000_000, math.MaxInt64}
	ratios := []int64{1, 7, 50, 70, 99, 100}
	compressors := []int64{1, 10, 1000}
	inits := []int64{0, 1, 2, 1000}
	prices := []int64{1, 2, 3, 10, 1000, 1_000_000}

	type key struct {
		maxGas int64
		ratio  int64
	}
	divergent := map[key]int{}
	// Classify every divergence by cause.
	causeGuard := 0    // target <= 0, the new early return
	causeFloor := 0    // target > 0 and initPrice == 0, the new max(init,1) floor
	causeUnknown := 0  // anything else: would break upgrade safety
	var unknownEx []string
	oldPanics, newPanics := 0, 0
	total := 0

	for _, mg := range maxGases {
		for _, r := range ratios {
			for _, c := range compressors {
				for _, ip := range inits {
					for _, lp := range prices {
						p := mkParams(r, c, ip)
						// gasUsed samples: 0, around target, at maxGas.
						target := int64(0)
						if mg > 0 && mg < math.MaxInt64/100 {
							target = mg * r / 100
						} else if mg == math.MaxInt64 {
							target = mg / 100 * r
						}
						gus := []int64{0, 1}
						for _, d := range []int64{-2, -1, 0, 1, 2} {
							if target+d >= 0 {
								gus = append(gus, target+d)
							}
						}
						if mg > 0 {
							gus = append(gus, mg)
						}
						for _, gu := range gus {
							total++
							o := runOld(mkPrice(lp), gu, mg, p)
							n := runNew(mkPrice(lp), gu, mg, p)
							if o.panicked {
								oldPanics++
							}
							if n.panicked {
								newPanics++
							}
							if o.panicked != n.panicked || (!o.panicked && o.amount != n.amount) {
								divergent[key{mg, r}]++
								tgtBig := new(big.Int).Mul(big.NewInt(mg), big.NewInt(r))
								tgtBig.Div(tgtBig, big.NewInt(100))
								switch {
								case tgtBig.Sign() <= 0:
									causeGuard++
								case ip == 0:
									causeFloor++
								default:
									causeUnknown++
									if len(unknownEx) < 10 {
										unknownEx = append(unknownEx, fmt.Sprintf(
											"maxGas=%d ratio=%d c=%d init=%d last=%d gasUsed=%d old=%v/%v new=%v/%v",
											mg, r, c, ip, lp, gu, o.amount, o.panicked, n.amount, n.panicked))
									}
								}
							}
						}
					}
				}
			}
		}
	}
	t.Logf("cases=%d oldPanics=%d newPanics=%d", total, oldPanics, newPanics)
	t.Logf("divergences by cause: newGuard(target<=0)=%d newFloor(init==0)=%d UNEXPLAINED=%d",
		causeGuard, causeFloor, causeUnknown)
	for _, e := range unknownEx {
		t.Logf("  UNEXPLAINED: %s", e)
	}
	_ = divergent
}

// TestZZTrajectory simulates block sequences and reports the price path.
func TestZZTrajectory(t *testing.T) {
	gk := GasPriceKeeper{}
	const maxGas = int64(3_000_000_000)
	p := mkParams(70, 10, 1)
	target := maxGas * 70 / 100

	report := func(name string, blocks int, start int64, gasFor func(i int, price int64) int64) {
		price := mkPrice(start)
		minP, maxP := start, start
		samples := []string{}
		halted := -1
		for i := range blocks {
			var panicked bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				price = gk.calcBlockGasPrice(price, gasFor(i, price.Price.Amount), maxGas, p)
			}()
			if panicked {
				halted = i + 1
				break
			}
			a := price.Price.Amount
			if a < minP {
				minP = a
			}
			if a > maxP {
				maxP = a
			}
			if i < 6 || i == blocks/2 || i >= blocks-3 {
				samples = append(samples, fmt.Sprintf("b%d=%d", i+1, a))
			}
		}
		t.Logf("%-34s start=%d final=%d min=%d max=%d halted_at_block=%d  %v", name, start, price.Price.Amount, minP, maxP, halted, samples)
	}

	report("idle 5000 blocks", 5000, 1_000_000, func(i int, _ int64) int64 { return 0 })
	report("full 5000 blocks", 5000, 1, func(i int, _ int64) int64 { return maxGas })
	report("exactly at target", 5000, 1000, func(i int, _ int64) int64 { return target })
	report("alternate full/empty", 5000, 1000, func(i int, _ int64) int64 {
		if i%2 == 0 {
			return maxGas
		}
		return 0
	})
	report("target+1 every block", 5000, 1000, func(i int, _ int64) int64 { return target + 1 })
	report("target-1 every block", 5000, 1000, func(i int, _ int64) int64 { return target - 1 })
	report("adversarial: 1 full, 9 empty", 5000, 1000, func(i int, _ int64) int64 {
		if i%10 == 0 {
			return maxGas
		}
		return 0
	})
	report("adversarial: 9 full, 1 empty", 5000, 1000, func(i int, _ int64) int64 {
		if i%10 == 9 {
			return 0
		}
		return maxGas
	})
	rng := rand.New(rand.NewSource(1))
	report("random uniform [0,maxGas]", 5000, 1000, func(i int, _ int64) int64 { return rng.Int63n(maxGas + 1) })
	report("keep just above target", 5000, 1000, func(i int, _ int64) int64 { return target + target/1000 })

	// int64 overflow reachability: how many all-full blocks to blow the int64 guard.
	price := mkPrice(1)
	n := 0
	for {
		next := gk.calcBlockGasPrice(price, maxGas, maxGas, p)
		n++
		if next.Price.Amount < price.Price.Amount || n > 5_000_000 {
			t.Logf("overflow probe: stalled at %d after %d blocks", next.Price.Amount, n)
			break
		}
		price = next
		if price.Price.Amount > math.MaxInt64/2 {
			t.Logf("overflow probe: reached %d after %d consecutive full blocks", price.Price.Amount, n)
			break
		}
	}

	// Same probe with a small target ratio (fast multiplier).
	p2 := mkParams(1, 1, 1)
	price = mkPrice(1)
	n = 0
	for n < 200 {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("overflow probe ratio=1 compressor=1: PANIC %q after %d full blocks (last price %d)", fmt.Sprint(r), n, price.Price.Amount)
					n = 1 << 30
				}
			}()
			price = gk.calcBlockGasPrice(price, maxGas, maxGas, p2)
			n++
		}()
	}
}

// TestZZUnboundedFreeze checks what an elevated price does once MaxGas flips to
// the unbounded spelling.
func TestZZUnboundedFreeze(t *testing.T) {
	gk := GasPriceKeeper{}
	p := mkParams(70, 10, 1)
	price := mkPrice(1_000_000)
	for range 1000 {
		price = gk.calcBlockGasPrice(price, 0, -1, p)
	}
	t.Logf("MaxGas=-1, 1000 idle blocks, start 1000000 -> %d", price.Price.Amount)
	price = mkPrice(1_000_000)
	for range 1000 {
		price = gk.calcBlockGasPrice(price, 0, 0, p)
	}
	t.Logf("MaxGas=0,  1000 idle blocks, start 1000000 -> %d", price.Price.Amount)
}

// TestZZBoundaries walks the specific boundaries the review asks about.
func TestZZBoundaries(t *testing.T) {
	gk := GasPriceKeeper{}
	p := mkParams(70, 10, 1)
	const maxGas = int64(3_000_000_000)
	target := maxGas * 70 / 100
	for _, tc := range []struct {
		name   string
		gas    int64
		maxGas int64
	}{
		{"gasUsed=0", 0, maxGas},
		{"gasUsed=target-1", target - 1, maxGas},
		{"gasUsed=target", target, maxGas},
		{"gasUsed=target+1", target + 1, maxGas},
		{"gasUsed=maxGas", maxGas, maxGas},
		{"gasUsed=maxGas+1 (over-limit)", maxGas + 1, maxGas},
		{"gasUsed=MaxInt64", math.MaxInt64, maxGas},
		{"maxGas=MaxInt64,gasUsed=0", 0, math.MaxInt64},
		{"maxGas=MaxInt64,gasUsed=MaxInt64", math.MaxInt64, math.MaxInt64},
		{"maxGas=-1", 0, -1},
		{"maxGas=0", 5, 0},
		{"maxGas=1", 5, 1},
		{"maxGas=2", 5, 2},
	} {
		o := runNew(mkPrice(1000), tc.gas, tc.maxGas, p)
		od := runOld(mkPrice(1000), tc.gas, tc.maxGas, p)
		t.Logf("%-32s new=%v(panic=%v %s) old=%v(panic=%v %s)", tc.name, o.amount, o.panicked, o.msg, od.amount, od.panicked, od.msg)
	}
	// ratio=100 boundary: target == maxGas.
	p100 := mkParams(100, 10, 1)
	t.Logf("ratio=100 maxGas=3e9 gasUsed=3e9 -> %+v", runNew(mkPrice(1000), maxGas, maxGas, p100))
	// compressor = 0 (Params.Validate rejects, but is it reachable?)
	p0 := mkParams(70, 0, 1)
	t.Logf("compressor=0 -> %+v", runNew(mkPrice(1000), 0, maxGas, p0))
	_ = gk
}
