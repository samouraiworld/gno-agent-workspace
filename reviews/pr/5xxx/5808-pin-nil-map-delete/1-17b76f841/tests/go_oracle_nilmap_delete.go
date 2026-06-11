// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: anywhere with a Go toolchain (no gno checkout needed):
curl -fsSL -o /tmp/go_oracle_nilmap_delete.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/go_oracle_nilmap_delete.go
go run /tmp/go_oracle_nilmap_delete.go
rm /tmp/go_oracle_nilmap_delete.go
*/
// Mechanism: gc oracle for the semantics pinned by delete1.gno. Mirrors the
// five hashable nil-map delete forms (all no-op under gc, matching gno) plus
// the unhashable-key cases where gc and gno diverge: gc panics "hash of
// unhashable type" on delete and read with a slice key even when the map is
// nil; gno no-ops both (nil guard runs before any hashing).
// Observed with go1.26.4: five "ok" lines, then three recovered panics
// (delete nil unhashable, delete nonnil unhashable, read nil unhashable),
// then "lens: 0 0 0 0 1". All gc panics are recoverable runtime panics.
// Flip: remove the recover wrapper on any unhashable case — the program
// crashes, showing the gc panic is real.
package main

import "fmt"

type S struct{ M map[string]int }

var pkgM map[string]int

func ret() map[string]int { return nil }

func try(label string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%s: panic: %v\n", label, r)
			return
		}
		fmt.Printf("%s: ok\n", label)
	}()
	f()
}

func main() {
	var m map[string]int
	try("delete nil local", func() { delete(m, "k") })
	try("delete nil pkgvar", func() { delete(pkgM, "k") })
	var s S
	try("delete nil field", func() { delete(s.M, "k") })
	try("delete nil ret()", func() { delete(ret(), "k") })
	try("delete nil conversion", func() { delete(map[string]int(nil), "k") })
	var mi map[interface{}]int
	try("delete nil unhashable", func() { delete(mi, []int{1}) })
	nm := map[interface{}]int{"a": 1}
	try("delete nonnil unhashable", func() { delete(nm, []int{1}) })
	try("read nil unhashable", func() { _ = mi[[]int{1}] })
	fmt.Println("lens:", len(m), len(pkgM), len(s.M), len(mi), len(nm))
}
