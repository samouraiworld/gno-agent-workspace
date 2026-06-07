// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from a local clone of gnolang/gno:
gh pr checkout 5699 -R gnolang/gno && git checkout 577362f8b
curl -fsSL -o gnovm/tests/files/params_byte_key_alias.gno \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5699-raw-byte-key-api/2-577362f8b/tests/params_byte_key_alias.gno
go test -v -run 'TestFiles/params_byte_key_alias.gno$' ./gnovm/pkg/gnolang/
rm gnovm/tests/files/params_byte_key_alias.gno
*/

// Probes the silent keyspace overlap between the existing string-keyed
// chain/params.SetBytes and the PR's new raw byte-keyed GetBytesKey.
// pkey("foo") and pkeyRaw([]byte("foo")) both build "vm:<realm>:foo", so a
// value written under the string API is readable under the raw API and vice
// versa whenever the raw key is valid UTF-8 without a leading-colon hazard.
//
// Observed: a write via SetBytes("foo", ...) is returned by
// GetBytesKey([]byte("foo")) with ok=true, and a delete via SetBytesKey(...,nil)
// removes the slot that SetBytes wrote. The two APIs are one keyspace.
//
// To assert a (hypothetical) namespaced design instead, flip the Output block:
// the cross-API read should miss (ok=false) if pkeyRaw used a distinct prefix.
package main

import "chain/params"

func main() {
	// Write under the string API.
	params.SetBytes("foo", []byte("via-string"))

	// Read under the raw API with a byte-identical key.
	v, ok := params.GetBytesKey([]byte("foo"))
	println("string->raw read ok:", ok, "val:", string(v))

	// Delete under the raw API.
	params.SetBytesKey([]byte("foo"), nil)
	_, ok2 := params.GetBytesKey([]byte("foo"))
	println("after raw delete ok:", ok2)
}

// Output:
// string->raw read ok: true val: via-string
// after raw delete ok: false
