/* Run: from a gno checkout:
gh pr checkout 5975 -R gnolang/gno && git checkout e3b7a7934
curl -fsSL -o /tmp/go_parity.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5975-parimutuel-payout-math/1-e3b7a7934/tests/go_parity.go
go run /tmp/go_parity.go
*/

// Go port of parimutuel.gno at e3b7a7934, used to cross-check three things
// against the gno run: the wrapped values match Go's int64 arithmetic, the
// panic threshold for a raked payout is winningPool > netPool, and summed
// payouts never exceed the pool once the product stays in range.
package main

import "fmt"

func assertNonNegative(v int64, name string) {
	if v < 0 {
		panic("parimutuel: " + name + " must be non-negative")
	}
}

func mulDiv(a, b, c int64) int64 {
	if c == 0 {
		panic("parimutuel: division by zero")
	}
	return (a * b) / c
}

func CalculatePayout(userBet, totalPool, winningPool int64) int64 {
	assertNonNegative(userBet, "userBet")
	assertNonNegative(totalPool, "totalPool")
	if winningPool <= 0 {
		panic("parimutuel: winningPool must be positive")
	}
	if winningPool > totalPool {
		panic("parimutuel: winningPool cannot exceed totalPool")
	}
	if userBet > winningPool {
		panic("parimutuel: userBet cannot exceed winningPool")
	}
	return mulDiv(userBet, totalPool, winningPool)
}

func CalculatePayoutWithRake(userBet, totalPool, winningPool, rakeBps int64) int64 {
	if rakeBps < 0 || rakeBps > 10000 {
		panic("parimutuel: rakeBps must be between 0 and 10000")
	}
	netPool := totalPool - mulDiv(totalPool, rakeBps, 10000)
	if netPool <= 0 {
		return 0
	}
	return CalculatePayout(userBet, netPool, winningPool)
}

func ImpliedProbabilityBps(outcomePool, totalPool int64) int64 {
	if totalPool <= 0 {
		panic("parimutuel: totalPool must be positive")
	}
	assertNonNegative(outcomePool, "outcomePool")
	if outcomePool > totalPool {
		panic("parimutuel: outcomePool cannot exceed totalPool")
	}
	return mulDiv(outcomePool, 10000, totalPool)
}

func try(label string, f func() int64) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%-46s PANIC %v\n", label, r)
		}
	}()
	fmt.Printf("%-46s %d\n", label, f())
}

func main() {
	try("payout 4e9/4e9/4e9", func() int64 { return CalculatePayout(4e9, 4e9, 4e9) })
	try("payout 5e9/1e10/5e9", func() int64 { return CalculatePayout(5e9, 1e10, 5e9) })
	try("impliedProb 1e15/1e15", func() int64 { return ImpliedProbabilityBps(1e15, 1e15) })
	try("rake 1000/1000/1000 @100bps", func() int64 { return CalculatePayoutWithRake(1000, 1000, 1000, 100) })
	try("rake 960/1000/960 @500bps", func() int64 { return CalculatePayoutWithRake(960, 1000, 960, 500) })
	try("rake 950/1000/950 @500bps", func() int64 { return CalculatePayoutWithRake(950, 1000, 950, 500) })
	try("rake 100/-1000/400 @0bps", func() int64 { return CalculatePayoutWithRake(100, -1000, 400, 0) })

	// truncation: three equal outcomes over a pool of 3 do not sum to 10000
	a, b, c := ImpliedProbabilityBps(1, 3), ImpliedProbabilityBps(1, 3), ImpliedProbabilityBps(1, 3)
	fmt.Printf("%-46s %d\n", "impliedProb thirds sum", a+b+c)

	// no overpay once the product stays inside int64
	worstDust, over := int64(0), 0
	for total := int64(1); total <= 120; total++ {
		for win := int64(1); win <= total; win++ {
			for x := int64(0); x <= win; x++ {
				for y := int64(0); y <= win-x; y++ {
					z := win - x - y
					sum := CalculatePayout(x, total, win) + CalculatePayout(y, total, win) + CalculatePayout(z, total, win)
					if sum > total {
						over++
					}
					if total-sum > worstDust {
						worstDust = total - sum
					}
				}
			}
		}
	}
	fmt.Printf("%-46s overpaying splits=%d worst dust=%d\n", "3-way split sweep total<=120", over, worstDust)
}
