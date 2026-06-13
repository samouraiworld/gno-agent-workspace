# Review: PR #5221
Event: COMMENT

## Body
This boundary should behave like Go, and that is the lens worth settling the PR on. Go accepts NaN and ±Inf as float arguments (so does `strconv.ParseFloat`), and the GnoVM produces and stores both itself via `math.NaN()`/`math.Inf()` and unguarded float division, so rejecting them here is the divergence rather than a fix. Verified on `28383f0`: every spelling of NaN/Inf parses to one fixed canonical bit pattern and the signature commits to the arg string, not the parsed float, so there is nothing malleable or non-deterministic to close. The one part that does match Go is the `-0`→`+0` fold (Go folds the literal `-0.0` to `+0`); that can stay.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5221-fix-float-malleability/1-28383f0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert.go:214-218 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214)
Go accepts NaN and ±Inf as float arguments, and the VM already produces and stores both, so rejecting them only on `maketx call` diverges from Go for no gain. The malleability rationale doesn't hold: the signature commits to the arg string, not the parsed float, and every spelling of NaN/Inf parses to one fixed bit pattern. Drop the two panics so NaN/Inf are accepted, matching Go.

<details><summary>repro</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5221 -R gnolang/gno
cat > gno.land/pkg/sdk/vm/zz_parity_test.go <<'EOF'
package vm
import ("fmt"; "math"; "strconv"; "testing"; "github.com/gnolang/gno/gnovm/pkg/gnolang")
func TestZZParity(t *testing.T) {
	for _, s := range []string{"NaN","nan","Inf","+Inf","Infinity","inf"} {
		f,_ := strconv.ParseFloat(s,64)
		fmt.Printf("parse %-9q bits=%016X\n", s, math.Float64bits(f))
	}
	for _, s := range []string{"NaN","Inf"} {
		func(){ defer func(){ fmt.Printf("convertArgToGno(%q) panics: %v\n", s, recover()) }()
			convertArgToGno(s, gnolang.Float64Type) }()
	}
}
EOF
go test -run TestZZParity -v ./gno.land/pkg/sdk/vm/ 2>&1 | grep -E "parse |panics"
rm gno.land/pkg/sdk/vm/zz_parity_test.go
```

```
parse "NaN"     bits=7FF8000000000001
parse "nan"     bits=7FF8000000000001
parse "Inf"     bits=7FF0000000000000
parse "+Inf"    bits=7FF0000000000000
parse "Infinity" bits=7FF0000000000000
parse "inf"     bits=7FF0000000000000
convertArgToGno("NaN") panics: float64 does not accept NaN
convertArgToGno("Inf") panics: float64 does not accept Inf
```
</details>

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert.go:220-224 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L220)
The `-0`→`+0` fold is fine and matches Go (the literal `-0.0` folds to `+0`), so it can stay, but its comment blames "malleability," which isn't the reason. Reword it to "match Go's `-0.0` literal folding."

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert_test.go:117 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117)
If the `-0` fold stays, no test covers a value that underflows to float32 `-0` (e.g. `"-1e-50"` as `Float32Type`). Add that case so a refactor touching only literal `-0` can't silently regress the float32 underflow path.

*(AI Agent)*

## gno.land/pkg/integration/testdata/maketx_call_float_args.txtar:38-39 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38)
The float64 `-0.0` case only asserts the formatted output is `"0"`, which `strconv.FormatFloat` also prints for true `+0`, so a float64 fold regression would slip through. Only the float32 path has a sign-bit assertion; add a `CheckSignBitFloat64` for parity.

*(AI Agent)*
