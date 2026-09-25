package amino_test

// gnolang/gno#6152, judge check for candidate 1: a nil behind a pointer, per kind.
//
// Repro from a plain clone of github.com/gnolang/gno (Go 1.25):
//   git checkout a742126999443305c6bfcf6fd6364ea57fba51d4
//   cp judge-1-nil-behind-pointer.go tm2/pkg/amino/zz_judge_nil_test.go
//   go test ./tm2/pkg/amino -run TestJudgeNilBehindPointer -v -count=1
//
// Observed at head a74212699 and at merge base 7916d1dd6:
//   *[]int: copy nil=true; *map: copy nil=false
//   **int: PANIC unsupported type invalid
//   *any: PANIC reflect: call of reflect.Value.Type on zero Value
// Observed with judge-fix.patch (one isNil guard at the top of _deepCopy,
// the Slice-only guard removed): every line copy nil=true.

import (
	"fmt"
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

func TestJudgeNilBehindPointer(t *testing.T) {
	try := func(name string, f func() bool) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("%s: PANIC %v", name, r)
			}
		}()
		t.Logf("%s: copy nil=%v", name, f())
	}
	var s []int
	try("*[]int", func() bool { return *amino.DeepCopy(&s).(*[]int) == nil })
	var m map[string]int
	try("*map", func() bool { return *amino.DeepCopy(&m).(*map[string]int) == nil })
	var p *int
	try("**int", func() bool { return *amino.DeepCopy(&p).(**int) == nil })
	var i any
	try("*any", func() bool { return *amino.DeepCopy(&i).(*any) == nil })
	_ = fmt.Sprint
}
