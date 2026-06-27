fix(gnovm): fold -0 to +0 for float call args

`convertFloat` turns MsgCall string args into typed floats with `strconv.ParseFloat`, which keeps the sign bit, so a `-0.0` or `-0` argument arrives as negative zero. The equivalent Go source literal `-0.0` is a constant that folds to `+0` at compile time, so the argument path diverges from Go for no reason. Clearing the sign bit on zero makes the two agree.

Behavior at the `convertArgToGno` boundary, before and after the fix (float bits):

```
arg          before (master)      after
f64 "-0.0"   0x8000000000000000   0x0000000000000000
f64 "-0"     0x8000000000000000   0x0000000000000000
f32 "-0.0"   0x80000000           0x00000000
f32 "-1e-50" 0x80000000           0x00000000   underflows to -0 at float32
f64 "-1e-50" 0xB58DEE7A4AD4B81F   unchanged    normal nonzero negative
```

Verified in Go: the source literal `-0.0` has bits `0x0` (sign bit clear), while `strconv.ParseFloat("-0.0", 64)` returns bits `0x8000000000000000` (sign bit set). The fix brings the arg path in line with the literal.

Salvaged from #5221. That PR also rejected `NaN` and `Inf` at this boundary; that half is dropped, since Go accepts both as float values and the GnoVM produces and stores them itself, so rejecting them only on `maketx call` is a divergence, not a fix.
