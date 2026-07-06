# Review: PR [#5864](https://github.com/gnolang/gno/pull/5864)
Event: COMMENT

## Body
The negative-zero fold is correct. Verified on 662cbc5ba through a live `maketx call` that `-0.0`, `-0`, and the float32-underflow `-1e-50` reach a realm with the sign bit cleared. Verified that reverting the fold restores the negative zero. The same live path still accepts `NaN` and `Inf`; see the inline note on convert.go:204.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5864-fold-negzero-float-args/1-662cbc5ba/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

## gno.land/pkg/sdk/vm/convert.go:204-224 [↗](../../../../../.worktrees/gno-review-5864/gno.land/pkg/sdk/vm/convert.go#L204)
`NaN`, `Inf`, `-Inf`, and `Infinity` parse through and reach realm code as real floats, since the fold only touches zero. [#5221](https://github.com/gnolang/gno/pull/5221) rejected them for the determinism reason this fold shares. Reject them here too, or say why admitting them is acceptable while folding `-0` is not.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5864 -R gnolang/gno

cat > gno.land/pkg/integration/testdata/probe_float_naninf.txtar <<'EOF'
loadpkg gno.land/r/test/floatprobe $WORK/realm
gnoland start
gnokey maketx call -pkgpath gno.land/r/test/floatprobe -func ClassF64 -args 'NaN' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stdout '("NaN" string)'
gnokey maketx call -pkgpath gno.land/r/test/floatprobe -func ClassF64 -args 'Inf' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stdout '("Inf" string)'
-- realm/realm.gno --
package floatprobe
import "math"
func ClassF64(cur realm, x float64) string {
	if math.IsNaN(x) { return "NaN" }
	if math.IsInf(x, 0) { return "Inf" }
	return "finite"
}
EOF

go test -run 'TestTestdata/probe_float_naninf' ./gno.land/pkg/integration/
rm gno.land/pkg/integration/testdata/probe_float_naninf.txtar
```

```
--- PASS: TestTestdata/probe_float_naninf
# realm returns ("NaN" string) and ("Inf" string): both args accepted, not rejected
```
</details>

## gno.land/pkg/sdk/vm/convert_test.go:1385 [↗](../../../../../.worktrees/gno-review-5864/gno.land/pkg/sdk/vm/convert_test.go#L1385)
Missing test: no assertion pins whether `NaN`/`Inf` args are accepted or rejected, so the scope decision behind this PR is invisible to the suite.

<details><summary>test cases</summary>

A txtar through the real MsgCall path covering both the fold (passes now) and NaN/Inf rejection (fails now, passes once rejection lands). If the decision is to keep admitting NaN/Inf, invert the four `!`/`stderr` assertions to lock the accept behavior instead.

```
loadpkg gno.land/r/test/floatargs $WORK/realm
gnoland start

gnokey maketx call -pkgpath gno.land/r/test/floatargs -func SignF64 -args '-0.0' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stdout '("false" string)'
gnokey maketx call -pkgpath gno.land/r/test/floatargs -func SignF32 -args '-1e-50' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stdout '("false" string)'

! gnokey maketx call -pkgpath gno.land/r/test/floatargs -func ClassF64 -args 'NaN' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stderr 'float64 does not accept NaN'
! gnokey maketx call -pkgpath gno.land/r/test/floatargs -func ClassF64 -args 'Inf' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stderr 'float64 does not accept Inf'
! gnokey maketx call -pkgpath gno.land/r/test/floatargs -func ClassF64 -args '-Inf' -gas-fee 1000000ugnot -gas-wanted 10000000 -broadcast -chainid=tendermint_test test1
stderr 'float64 does not accept Inf'

-- realm/realm.gno --
package floatargs
import ("math"; "strconv")
func SignF64(cur realm, x float64) string { return strconv.FormatBool(math.Signbit(x)) }
func SignF32(cur realm, x float32) string { return strconv.FormatBool(math.Signbit(float64(x))) }
func ClassF64(cur realm, x float64) string {
	if math.IsNaN(x) { return "NaN" }
	if math.IsInf(x, 0) { return "Inf" }
	return "finite"
}
```
</details>

## gno.land/pkg/sdk/vm/convert.go:219-221 [↗](../../../../../.worktrees/gno-review-5864/gno.land/pkg/sdk/vm/convert.go#L219)
`if f64 == 0 { f64 = 0 }` reads as a no-op; the sign-bit clear lives only in the comment. Write the zero case so the code shows it clears the sign, instead of relying on the reader trusting the comment.
