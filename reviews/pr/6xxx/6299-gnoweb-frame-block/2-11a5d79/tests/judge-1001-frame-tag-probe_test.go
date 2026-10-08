package markdown

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestJudgeProbe(t *testing.T) {
	cases := map[string]string{
		"c1_no_grid_opener":   "<gno-frame>\nintro\n<!--\n<gno-frame>\nsecret\n-->\n</gno-frame>\nafter\n",
		"c1_commented_card":   "<gno-frame>\n<gno-columns>\n<!--\n<gno-frame>\nx\n</gno-frame>\n-->\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\nafter\n",
		"c1002_refused_unclosed": "<gno-frame>\n<gno-columns>\n<gno-frame x=\"1\">\ncard\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\nafter\n",
		"c1001_frame_in_card": "<gno-frame>\n<gno-columns>\n<gno-frame>\n<gno-frame>\nx\n</gno-frame>\n</gno-frame>\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\nafter\n",
		"c1001_two_refused":   "<gno-frame>\n<gno-columns>\n<gno-frame x=\"1\">\na\n</gno-frame>\n<gno-frame x=\"2\">\nb\n</gno-frame>\n<gno-columns-sep>\nc\n</gno-columns>\n</gno-frame>\nafter\n",
		"c2_nested":           "<gno-frame>\n<gno-frame>\ninner\n</gno-frame>\nouter rest\n</gno-frame>\nafter\n",
		"c3_selfclosing":      "<gno-frame>\n<gno-columns>\n<gno-frame/>\na\n</gno-frame>\nb\n</gno-columns>\nafter\n",
		"c3_selfclosing_closed": "<gno-frame>\n<gno-columns>\n<gno-frame/>\na\n<gno-columns-sep>\nb\n</gno-columns>\n</gno-frame>\nafter\n",
	}
	names := make([]string, 0, len(cases))
	for k := range cases {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		var buf bytes.Buffer
		if err := convertGno(newGnoMarkdown(), []byte(cases[k]), &buf); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("=== %s\n%s", k, buf.String())
	}
}

func TestJudgeGridScanTimes(t *testing.T) {
	body := strings.Repeat("x\n\n", 50)
	pages := map[string]string{
		"half inside": "<gno-frame>\n<gno-columns>\n" + body + "</gno-frame>\n" + body + "</gno-columns>\n",
		"inside":      "<gno-frame>\n<gno-columns>\n<gno-frame>\n" + body + "</gno-frame>\n<gno-columns-sep>\n" + body + "</gno-columns>\n</gno-frame>\n",
		"unclosed":    "<gno-frame>\n<gno-columns>\n" + body,
	}
	m := newGnoMarkdown()
	render := func(src []byte) time.Duration {
		best := time.Duration(1<<63 - 1)
		for range 5 {
			var buf bytes.Buffer
			runtime.GC()
			start := time.Now()
			_ = convertGno(m, src, &buf)
			best = min(best, time.Since(start))
		}
		return best
	}
	for _, name := range []string{"half inside", "inside", "unclosed"} {
		small := render([]byte(strings.Repeat(pages[name], 400)))
		large := render([]byte(strings.Repeat(pages[name], 1600)))
		fmt.Printf("TIMES %s small=%v large=%v ratio=%.2f checked=%v\n", name, small, large, float64(large)/float64(small), large >= 20*time.Millisecond)
	}
}
