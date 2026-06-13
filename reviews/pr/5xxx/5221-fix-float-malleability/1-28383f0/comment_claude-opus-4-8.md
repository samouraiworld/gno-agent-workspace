# Review: PR #5221
Event: COMMENT

## Body
The conversion logic is correct, but the change's premise needs settling before merge: there is no malleability or non-determinism at this boundary to fix, so rejecting NaN/Inf is a policy choice about the MsgCall ABI. Verified on `28383f0`: every spelling of NaN/Inf (`NaN`, `nan`, `Inf`, `+Inf`, `Infinity`) parses to one fixed canonical bit pattern, and the signature commits to the arg string rather than the parsed float, so no equivalent inputs can diverge after signing. The VM itself produces and stores both NaN and Inf via `math.NaN()`/`math.Inf()` and unguarded float division, so this restriction does not match how floats behave elsewhere.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5221-fix-float-malleability/1-28383f0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert.go:214-218 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214)
The signature commits to the arg string, not the parsed float, and every spelling of NaN/Inf already parses to one fixed bit pattern, so there is no malleability or non-determinism here to fix. Rejecting NaN/Inf is a policy choice that also contradicts the rest of the VM, which produces and stores both via `math.NaN()`/`math.Inf()` and unguarded float division, so the same function can receive these values from another realm or `maketx run` but not from a direct `maketx call`. Either justify the ABI restriction as a deliberate policy or drop it.

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
The `-0` to `+0` step is fine, but its comment ("prevent malleability") is the wrong reason, and the step is redundant at the only consensus-relevant place since `MapKeyBytes` already normalizes `-0` to `0`. The real justification is Go parity: a Go source literal `-0.0` folds to `+0`, so the arg boundary matching that is sensible. Reword the comment to say that.

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert.go:215 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L215)
`"float%d does not accept NaN"` reads like a parser-internal spec; `"float%d argument cannot be NaN"` is clearer to the realm caller who sees it as a transaction error.

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert_test.go:117 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert_test.go#L117)
The canonicalization test covers float64 `-0.0`/`-0` but not a value that underflows to float32 `-0` (e.g. `"-1e-50"` for `Float32Type`, which rounds to float32 `-0` and is then canonicalized). Add that case so a refactor touching only literal `-0` cannot regress the float32 underflow path invisibly.

*(AI Agent)*

## gno.land/pkg/integration/testdata/maketx_call_float_args.txtar:38-39 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/integration/testdata/maketx_call_float_args.txtar#L38)
The float64 `-0.0` case only asserts the formatted output is `"0"`, which `strconv.FormatFloat` also prints for true `+0`, so a float64 canonicalization regression would slip through. Only the float32 path has a sign-bit assertion; add a `CheckSignBitFloat64` for parity.

*(AI Agent)*
