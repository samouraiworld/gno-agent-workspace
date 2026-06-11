// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a gno checkout:
gh pr checkout 5808 -R gnolang/gno && git checkout 17b76f841
curl -fsSL -o gnovm/tests/files/nilmap_delete_key_once.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/nilmap_delete_key_once.gno
go test -v -run 'TestFiles/nilmap_delete_key_once.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/nilmap_delete_key_once.gno
*/
// Mechanism: delete's key argument is evaluated before the builtin runs, so
// the nil-map early return (uverse.go:978-980) must not skip key side
// effects. The key() call prints once; Go evaluates call arguments exactly
// once regardless of the nil no-op, and gno matches.
// Observed at 17b76f841: PASS — "key evaluated" printed exactly once.
// Flip: duplicate the "key evaluated" Output line (asserting double
// evaluation) — fails, the key expression runs once.
package main

var m map[string]int

func key() string {
	println("key evaluated")
	return "k"
}

func main() {
	delete(m, key())
	println("done")
}

// Output:
// key evaluated
// done
