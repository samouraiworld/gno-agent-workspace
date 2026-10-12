// Repro: a <gno-details> opener left unclosed inside a blockquote or a list
// item keeps its level of the shared gno-* depth cap until the end of the
// page, so four of them refuse every gno-* block the realm renders after
// them. BlockRich passes `> <gno-details>` and `- <gno-details>` through.
//
// From a plain clone:
//
//	git clone https://github.com/gnolang/gno && cd gno
//	git checkout 5ad81a10214a0379b6099159e98431176732d1f7
//	cp <this file> gno.land/pkg/gnoweb/markdown/zz_depth_leak_test.go
//	cd gno.land/pkg/gnoweb/markdown
//	go test -run 'TestDetailsDepthLeak' -count=1 -v .
//
// Expected at 5ad81a102: both tests FAIL on the 4-opener cases (columns,
// alert and the realm's own details refused); the 1-opener and control
// cases pass. ext_details.go does not exist at the merge base 02606904e.
package markdown

import (
	"strings"
	"testing"
)

const leakRealmTail = "\n\n<gno-columns>\nA\n|||\nB\n</gno-columns>\n\n> [!WARNING]\n> Do not send funds\n\n<gno-details summary=\"Admin\">\nhidden\n</gno-details>\n"

func checkRealmTail(t *testing.T, out string) {
	t.Helper()
	if !strings.Contains(out, `class="gno-columns"`) {
		t.Errorf("realm grid refused (its A|||B body is swallowed as raw HTML)")
	}
	if !strings.Contains(out, "gno-alert") {
		t.Errorf("realm alert refused (renders as a plain quote reading [!WARNING])")
	}
	if !strings.Contains(out, "<span>Admin</span>") {
		t.Errorf("realm details refused (its body renders unfolded)")
	}
}

func TestDetailsDepthLeakRaw(t *testing.T) {
	for name, src := range map[string]string{
		"control":       "comment" + leakRealmTail,
		"quote_x1":      "> <gno-details>\n" + leakRealmTail,
		"quote_x4":      strings.Repeat("> <gno-details>\n", 4) + leakRealmTail,
		"list_items_x4": strings.Repeat("- <gno-details>\n  b\n", 4) + leakRealmTail,
	} {
		t.Run(name, func(t *testing.T) {
			_, out, _ := parseDetails(t, src)
			checkRealmTail(t, out)
		})
	}
}

func TestDetailsDepthLeakBlockRich(t *testing.T) {
	for name, in := range map[string]string{
		"control":  "nice post",
		"quote_x4": strings.Repeat("> <gno-details>\n", 4),
		"list_x4":  strings.Repeat("- <gno-details>\n", 4),
	} {
		t.Run(name, func(t *testing.T) {
			san := callSanitizeGno(t, "BlockRich", in, "", "")
			checkRealmTail(t, renderMarkdown(t, "# Board\n\n"+san+leakRealmTail))
		})
	}
}
