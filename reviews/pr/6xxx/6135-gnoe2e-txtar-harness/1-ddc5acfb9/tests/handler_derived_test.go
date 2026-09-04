// Asserts that a Handler derived through WithAttrs or WithGroup keeps the
// colour the parent was built with, which is how every cluster line is
// rendered (cmd_run.go's logger.With("component", "cluster")).
// Measured: at ddc5acfb9 the derived handler drops h.color, so the composed
// line is run through plain.Replace and every escape sequence is stripped.
// Fails at the reviewed head; passes once WithAttrs and WithGroup copy color.
/* Run: from a gno checkout:
gh pr checkout 6135 -R gnolang/gno && git checkout ddc5acfb9
curl -fsSL -o misc/gnoe2e/internal/termlog/handler_derived_test.go \
  https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6135-gnoe2e-txtar-harness/1-ddc5acfb9/tests/handler_derived_test.go
go test -C misc/gnoe2e -v -run 'TestHandlerKeepsColourThroughWith|TestHandlerKeepsColourThroughGroup' ./internal/termlog/
rm misc/gnoe2e/internal/termlog/handler_derived_test.go
*/

package termlog

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// The baseline and the defect in one run: the undecorated logger is the
// existing TestHandlerColoursALineForATerminal, and the derived one is what
// every cluster and serve line actually goes through.
func TestHandlerKeepsColourThroughWith(t *testing.T) {
	var out bytes.Buffer
	// newHandler with color=true stands in for a run attached to a terminal,
	// which a bytes.Buffer can never be.
	root := slog.New(newHandler(&out, false, true))

	root.Info("cluster ready")
	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"baseline: the undecorated logger colours its level")

	out.Reset()
	root.With("component", "cluster").Info("cluster ready")
	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"a logger derived with With() must colour its level too")
	require.Contains(t, out.String(), colorBold+"cluster"+colorReset,
		"and must render the component prefix in bold")
}

func TestHandlerKeepsColourThroughGroup(t *testing.T) {
	var out bytes.Buffer
	slog.New(newHandler(&out, false, true)).WithGroup("cluster").Info("cluster ready")

	require.Contains(t, out.String(), colorGreen+"INF"+colorReset,
		"a logger derived with WithGroup() must colour its level too")
}
