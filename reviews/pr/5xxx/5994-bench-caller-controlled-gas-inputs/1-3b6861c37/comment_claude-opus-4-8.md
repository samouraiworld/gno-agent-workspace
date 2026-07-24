# Review: PR [#5994](https://github.com/gnolang/gno/pull/5994)
Event: COMMENT

## Body
Both mispricings reproduce with numbers. innerHash measured 1369 ns at 32 bytes per side against 75742 ns at 4096, a 55x span billed as a flat 7513, and the L32/R4096 vs L4096/R32 gap (37037 vs 48229) shows the two sides are independent axes a single total-bytes slope cannot capture. modExp measured 317486 ns against 9070185 ns at a fixed 256-byte modulus as the exponent grew, the whole 28.6x driven by the argument the old spec left free. The change is a fitter and benchmark change and does not move the shipped table: regenerating from the same benchmark data before and after the change produces identical output. The one substantive addition beyond the two spec fixes is extending the undercharge replay to the flat and one-parameter paths, which is the path that silently mispriced innerHash in the first place.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5994-bench-caller-controlled-gas-inputs/1-3b6861c37/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gnovm/cmd/calibrate/gen_native_table.py:239
The two-arg specs for innerHash `(0,LenBytes,1,LenBytes)` and modExp `(1,LenBytes,2,LenBytes)` are the right shape, and the runtime does consume the second axis (`Slope2` is summed in `chargeNativeGas`). Worth stating plainly in the PR body that the shipped `native_gas.go` is unchanged and needs a reference-hardware regeneration to pick these up, so a reviewer does not expect the table diff.

## gnovm/stdlibs/crypto/modexp/modexp.go:6
Note, not a blocker: `X_modExp` accepts slices of any length, and the additive model cannot express the `len(exp)·len(mod)²` product, so even the corrected entry undercharges large inputs. The fitter says so now (R² below zero, an 18x undercharge at a benched point). The durable fix is a length ceiling in `X_modExp` matching the range the entry is fit for. Merged in #5725, not yet deployed, so this is a heads-up rather than a live exposure.
