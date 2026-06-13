# Review: PR #5221
Event: COMMENT

## Body
This boundary should behave like Go, and that is the lens worth settling the PR on. Go accepts NaN and ±Inf as float arguments and the GnoVM already produces and stores both itself, so rejecting them here is the divergence rather than a fix. Verified on `28383f0`: every spelling of NaN/Inf parses to one fixed canonical bit pattern, so there is nothing non-deterministic to canonicalize. The one part that does match Go is the `-0`→`+0` fold (Go folds the literal `-0.0` to `+0`); that can stay.

Full review: https://github.com/samouraiworld/gno-agent-workspace/blob/main/reviews/pr/5xxx/5221-fix-float-malleability/1-28383f0/review_claude-opus-4-8_davd-gzl.md [↗](review_claude-opus-4-8_davd-gzl.md)

*(AI Agent)*

## gno.land/pkg/sdk/vm/convert.go:214-218 [↗](../../../../../.worktrees/gno-review-5221/gno.land/pkg/sdk/vm/convert.go#L214)
Go accepts NaN and ±Inf as float arguments, and the VM already produces and stores both, so rejecting them only on `maketx call` diverges from Go for no gain. The malleability rationale doesn't hold: the signature commits to the arg string, not the parsed float, and every spelling of NaN/Inf parses to one fixed bit pattern. Drop the two panics so NaN/Inf are accepted, matching Go.

The same source, run in Go and in the GnoVM, then the same values at the `maketx call` arg boundary:

| input | Go | Gno (VM) | `maketx call` arg (this PR) |
|---|---|---|---|
| `NaN` | `echo=NaN isNaN=true` | `echo=NaN isNaN=true` | panic: `float64 does not accept NaN` |
| `Inf` | `echo=+Inf isInf=true` | `echo=+Inf isInf=true` | panic: `float64 does not accept Inf` |
| `-0.0` | `signbit=false` | `signbit=false` | `signbit=false` (bits `0x0`, folded) |

<details><summary>Go</summary>

```bash
# from a local clone of gnolang/gno:
gh pr checkout 5221 -R gnolang/gno
cat > /tmp/fc.go <<'EOF'
package main
import ("math"; "strconv")
func echo(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
func main() {
	println("NaN  echo=" + echo(math.NaN()) + " isNaN=" + strconv.FormatBool(math.IsNaN(math.NaN())))
	println("Inf  echo=" + echo(math.Inf(1)) + " isInf=" + strconv.FormatBool(math.IsInf(math.Inf(1), 1)))
	println("-0.0 signbit=" + strconv.FormatBool(math.Signbit(-0.0)))
}
EOF
go run /tmp/fc.go; rm /tmp/fc.go
```

```
NaN  echo=NaN isNaN=true
Inf  echo=+Inf isInf=true
-0.0 signbit=false
```
</details>

<details><summary>Gno</summary>

Same source in the GnoVM (the language agrees with Go), then the same values at the `maketx call` arg boundary (this PR):

```bash
# from a local clone of gnolang/gno (run from the repo root):
gh pr checkout 5221 -R gnolang/gno
cat > /tmp/fc.gno <<'EOF'
package main
import ("math"; "strconv")
func echo(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }
func main() {
	println("NaN  echo=" + echo(math.NaN()) + " isNaN=" + strconv.FormatBool(math.IsNaN(math.NaN())))
	println("Inf  echo=" + echo(math.Inf(1)) + " isInf=" + strconv.FormatBool(math.IsInf(math.Inf(1), 1)))
	println("-0.0 signbit=" + strconv.FormatBool(math.Signbit(-0.0)))
}
EOF
echo "== gno run =="; GNOROOT=$PWD go run ./gnovm/cmd/gno run /tmp/fc.gno 2>/dev/null; rm /tmp/fc.gno
cat > gno.land/pkg/sdk/vm/zz_parity_test.go <<'EOF'
package vm
import ("fmt"; "testing"; "github.com/gnolang/gno/gnovm/pkg/gnolang")
func TestZZParity(t *testing.T) {
	for _, s := range []string{"NaN", "Inf", "-0.0"} {
		func() {
			defer func() { if r := recover(); r != nil { fmt.Printf("arg %-5q -> panic: %v\n", s, r) } }()
			tv := convertArgToGno(s, gnolang.Float64Type)
			fmt.Printf("arg %-5q -> bits=%016X\n", s, tv.GetFloat64())
		}()
	}
}
EOF
echo "== maketx call arg (this PR) =="; go test -run TestZZParity -v ./gno.land/pkg/sdk/vm/ 2>&1 | grep "arg "
rm gno.land/pkg/sdk/vm/zz_parity_test.go
```

```
== gno run ==
NaN  echo=NaN isNaN=true
Inf  echo=+Inf isInf=true
-0.0 signbit=false
== maketx call arg (this PR) ==
arg "NaN" -> panic: float64 does not accept NaN
arg "Inf" -> panic: float64 does not accept Inf
arg "-0.0" -> bits=0000000000000000
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
