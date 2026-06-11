// NOT AUDITED — AI-generated adversarial test artifact. Verify before executing in any privileged context.
/* Run: from any machine with the Go toolchain (no gno checkout needed):
curl -fsSL -o /tmp/opus2_go_oracle_nilmap_delete.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/5xxx/5808-pin-nil-map-delete/1-17b76f841/tests/opus2_go_oracle_nilmap_delete.go
go run /tmp/opus2_go_oracle_nilmap_delete.go
*/
// Go gc oracle for the gno<->gc divergence delete1.gno pins. Confirms that the
// reference toolchain hashes the key BEFORE the nil-map short-circuit, so
// delete(nilmap, []int{1}) panics "hash of unhashable type: []int" in gc, while
// gno no-ops it (the gno guard returns before ComputeMapKey). Also confirms the
// gc panic is itself recoverable via Go defer/recover — relevant context for
// ADR consequence 2's recoverability framing.
//
// Observed output:
//   A nil-map slice-key delete: PANIC: hash of unhashable type: []int
//   B non-nil-map slice-key delete: PANIC: runtime error: hash of unhashable type []int
package main

import "fmt"

func main() {
	// Case A: NIL map, unhashable (slice) interface key.
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("A nil-map slice-key delete: PANIC:", r)
			} else {
				fmt.Println("A nil-map slice-key delete: no-op (no panic)")
			}
		}()
		var mi map[interface{}]int // nil
		delete(mi, []int{1})
		fmt.Println("A returned normally")
	}()

	// Case B: NON-nil map, unhashable (slice) interface key.
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("B non-nil-map slice-key delete: PANIC:", r)
			}
		}()
		mi := map[interface{}]int{1: 1}
		delete(mi, []int{1})
		fmt.Println("B returned normally")
	}()
}
