/* Run:
TEST_DIR=$(mktemp -d)
cp go_zerosize_reference_test.go "$TEST_DIR/main_test.go"
cd "$TEST_DIR" && go mod init zsize >/dev/null && go test -v -run Test ./...
rm -rf "$TEST_DIR"
*/

// Reference: Go's gc compiler treats [N]struct{} and [N][0]int as
// zero-sized. unsafe.Sizeof returns 0 for both. PR #5708 only
// short-circuits ArrayType when Len==0, so it misses these.

package zsize

import (
	"testing"
	"unsafe"
)

func TestSizes(t *testing.T) {
	var a [10]struct{}
	if got := unsafe.Sizeof(a); got != 0 {
		t.Errorf("unsafe.Sizeof([10]struct{}) = %d, want 0", got)
	}
	var b [5][0]int
	if got := unsafe.Sizeof(b); got != 0 {
		t.Errorf("unsafe.Sizeof([5][0]int) = %d, want 0", got)
	}
}

func TestZerobaseSharing(t *testing.T) {
	// Note: gc constant-folds &x==&y to false, but new() goes through
	// runtime.mallocgc -> runtime.zerobase, so two new() calls share
	// the same backing storage. p == q is FALSE under gc only because
	// of the SSA fold on &-of-locals; under the spec semantics Gno
	// claims to follow, they should compare equal.
	p := new([10]struct{})
	q := new([10]struct{})
	t.Logf("new([10]struct{}) -> p=%p q=%p (Go: ptr addresses may equal under spec)", p, q)
	_ = p
	_ = q
}
