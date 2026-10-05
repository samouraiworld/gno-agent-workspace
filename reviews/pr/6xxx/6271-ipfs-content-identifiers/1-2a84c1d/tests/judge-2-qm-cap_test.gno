// Judge measurement for candidate 2 of gnolang/gno PR 6271 at 2a84c1dcd:
// gas of Parse on an over-long "Qm" string, with and without the
// `len(s) == 46` cap in Parse; run with gno test -v -print-runtime-metrics.
package cid

import (
	"strings"
	"testing"
)

func TestJudgeQmRepeatOnly(t *testing.T) { _ = "Qm" + strings.Repeat("z", 510) }
func TestJudgeQm512(t *testing.T)        { Parse("Qm" + strings.Repeat("z", 510)) }
