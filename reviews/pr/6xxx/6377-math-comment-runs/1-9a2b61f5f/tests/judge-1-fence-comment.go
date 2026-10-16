// Asserts that \left, \middle, \right and \big skip a comment before their
// delimiter, as TeX does. Passes at 9a2b61f5f; turns red once tokComment is
// dropped from the fixFences skip mask, which every existing test survives.
//
//   cp <this file> gno.land/pkg/gnoweb/markdown/mathml/zz_fence_comment_test.go
//   go test -count=1 -run TestZZFenceComment -v ./gno.land/pkg/gnoweb/markdown/mathml/
package mathml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZZFenceComment(t *testing.T) {
	for _, p := range [][2]string{
		{"\\left%c\n(x\\right)", "\\left(x\\right)"},
		{"\\left(x\\middle%c\n|y\\right)", "\\left(x\\middle|y\\right)"},
		{"\\left(x\\right%c\n)", "\\left(x\\right)"},
		{"\\big%c\n(x\\big)", "\\big(x\\big)"},
		{"\\left %c\n  ( x \\right)", "\\left(x\\right)"},
	} {
		assert.Equal(t, mathBody(t, p[1]), mathBody(t, p[0]), p[0])
	}
}
