// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5609 -R gnolang/gno && git checkout 443885d1b
curl -fsSL -o gnovm/tests/files/addressable_gap_ptr_method_recv.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5609-gnovm-addressability-check/2-443885d1b/tests/addressable_gap_ptr_method_recv.gno
go test -v -run 'TestFiles/addressable_gap_ptr_method_recv.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/addressable_gap_ptr_method_recv.gno
*/

// Calling a pointer-receiver method on a non-addressable value auto-takes the
// receiver's address. PR #5609 synthesizes that `&RefExpr` at preprocess.go:2404/2432
// and marks it setPreprocessed, so the TRANS_LEAVE *RefExpr addressability gate
// never runs on it. `m["k"]` is a map element (non-addressable), so Go rejects
// the call; gno accepts it and even mutates the stored element. Observed:
// `// Output: 2` passes (the increment persisted into the map).

package main

type T struct{ n int }

func (t *T) Inc() { t.n++ }

func main() {
	m := map[string]T{"k": {1}}
	m["k"].Inc()
	println(m["k"].n)
}

// Output:
// 2

// TypeCheckError:
// main/addressable_gap_ptr_method_recv.gno:25:9: cannot call pointer method Inc on T
