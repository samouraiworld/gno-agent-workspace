/* Run: from a gno checkout:
gh pr checkout 5981 -R gnolang/gno && git checkout 0558015ac
curl -fsSL -o gnovm/tests/files/iota_identifier_forinit.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5981-avoid-shadowing-iota/1-0558015ac/tests/iota_identifier_forinit.gno
go test -v -run 'TestFiles/iota_identifier_forinit.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/iota_identifier_forinit.gno
*/

// A three-clause for loop renames its init-declared names to "<name>.loopvar"
// before initStaticBlocks reserves them, so StaticBlock.Reserve never sees
// "iota" and the guard does not fire.
// At 0558015ac this program runs and prints 0 and 1 instead of erroring.

package main

func main() {
	for iota := 0; iota < 2; iota++ {
		println(iota)
	}
}

// Error:
// main/iota_identifier_forinit.gno:18:6-14: builtin identifiers cannot be shadowed: iota
