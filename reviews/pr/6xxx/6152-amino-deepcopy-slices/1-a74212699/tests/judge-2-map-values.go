package amino_test

// gnolang/gno#6152, judge check for candidate 2: map values are stored, not copied.
//
// Repro from a plain clone of github.com/gnolang/gno (Go 1.25):
//   git checkout a742126999443305c6bfcf6fd6364ea57fba51d4
//   cp judge-2-map-values.go tm2/pkg/amino/zz_judge_map_test.go
//   go test ./tm2/pkg/amino -run TestJudgeMapValues -v -count=1
//
// Observed at head a74212699 and at merge base 7916d1dd6: map value aliased: true
// Observed with judge-2-map-values.patch applied: map value aliased: false

import (
	"testing"

	"github.com/gnolang/gno/tm2/pkg/amino"
)

type judgeMS struct{ M map[string][]byte }

func TestJudgeMapValues(t *testing.T) {
	src := judgeMS{M: map[string][]byte{"a": {1}}}
	cpy := amino.DeepCopy(src).(judgeMS)
	src.M["a"][0] = 9
	t.Logf("map value aliased: %v", cpy.M["a"][0] == 9)
}
