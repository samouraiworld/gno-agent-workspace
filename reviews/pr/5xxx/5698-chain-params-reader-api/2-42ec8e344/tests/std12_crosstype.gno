// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5698 -R gnolang/gno && git checkout 42ec8e344
curl -fsSL -o gnovm/tests/files/std12_crosstype.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5698-chain-params-reader-api/2-42ec8e344/tests/std12_crosstype.gno
go test -v -run 'TestFiles/std12_crosstype.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/std12_crosstype.gno
*/

// chain/params.SetBytes stores raw bytes (tm2/pkg/sdk/params/keeper.go SetBytes,
// no JSON). GetString routes through getIfExists -> amino.MustUnmarshalJSON,
// which on a real chain PANICS on the raw, non-JSON {0,1,2,255} payload.
// The in-memory test stub (gnovm/pkg/test/test.go testParams.GetString) instead
// does a typed map lookup and returns (", false). So this filetest passes under
// `gno test` (output below) but the same realm panics on chain — a test/prod
// divergence. The doc comment at test.go:159-161 claims the stub mirrors "the
// production keeper's amino-unmarshal" fail-safe; that claim is false.
//
// Observed under gno test: prints "false" then "done", test reports ok.
// To assert the production behavior instead, the contract must either document
// the panic or make GetXxx return (zero,false) on type mismatch on chain too.
package main

import "chain/params"

func main() {
	params.SetBytes("k.bytes", []byte{0, 1, 2, 255})
	_, ok := params.GetString("k.bytes")
	println("crosstype string ok:", ok)   // IS:     false (test stub) — chain PANICS in amino.MustUnmarshalJSON
	// println("crosstype string ok: PANIC") // SHOULD: same observable on both paths
	println("done")
}

// Output:
// crosstype string ok: false
// done
