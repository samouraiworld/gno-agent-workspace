// Design harness, parent: one parse per shape per design of the details
// closer. Apply parent-design-harness.patch to ext_details.go at c05644643,
// copy this file beside it, then: go test -run TestZZDesign -v .
// zzDesign: Z no exemption (8ab095f49), A the descendant walk (c05644643),
// B LastOpenedBlock is foreign, C LastOpenedBlock is fenced code, HTML or
// foreign, E a separate closer parser matching only under its own details.
package markdown

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

type zzShape struct {
	src  string
	ok   func(out string) bool
	what string
}

func zzOutside(out, s string) bool { // s rendered after the last </details>
	i := strings.LastIndex(out, "</details>")
	return i >= 0 && strings.Contains(out[i:], s)
}
func zzInside(out, s string) bool {
	i := strings.Index(out, s)
	j := strings.LastIndex(out, "</details>")
	return i >= 0 && j > i
}

func TestZZDesign(t *testing.T) {
	shapes := map[string]zzShape{
		"01 fence holds :::":        {":::details A\n```\n:::\n```\nafter\n:::\n\nfooter\n", func(o string) bool { return zzInside(o, "after") && zzOutside(o, "footer") }, "after inside, footer outside"},
		"02 html block holds :::":   {":::details A\n<div>\n:::\n</div>\n\n:::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") && !strings.Contains(o, "<p>:::</p>") }, "footer outside, no stray :::"},
		"03 foreign holds :::":      {":::details A\n\n<gno-foreign>\nhi\n:::\n<gno-form exec=\"W\">\n</gno-form>\n</gno-foreign>\n\n:::\n\nfooter\n", func(o string) bool { return !strings.Contains(o, "<form") && zzOutside(o, "footer") }, "no form, footer outside"},
		"04 foreign in list :::":    {":::details A\n\n- <gno-foreign>\n  hi\n  :::\n  <gno-form exec=\"W\">\n  </gno-form>\n  </gno-foreign>\n\n:::\n\nfooter\n", func(o string) bool { return !strings.Contains(o, "<form") && zzOutside(o, "footer") }, "no form, footer outside"},
		"05 stale foreign quote":    {":::details A\n> <gno-foreign>\n> text\n\n:::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") }, "footer outside"},
		"06 stale foreign quote nb": {":::details A\n> <gno-foreign>\n> text\n:::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") }, "footer outside"},
		"07 stale foreign list":     {":::details A\n- <gno-foreign>\n  text\n\n:::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") }, "footer outside"},
		"08 equal nest (pandoc)":    {":::details A\n:::details B\nx\n:::\ny\n:::\n\nfooter\n", func(o string) bool { return zzInside(o, "<p>y</p>") && zzOutside(o, "footer") }, "y inside A"},
		"09 longer outer fence":     {"::::details A\n:::details B\nx\n:::\ny\n::::\n\nfooter\n", func(o string) bool { return zzInside(o, "<p>y</p>") && zzOutside(o, "footer") }, "y inside A"},
		"10 paragraph then :::":     {":::details A\ntext\n:::\nfooter\n", func(o string) bool { return zzOutside(o, "footer") && !strings.Contains(o, ":::") }, "closes"},
		"11 list then :::":          {":::details A\n- item\n:::\nfooter\n", func(o string) bool { return zzOutside(o, "footer") && !strings.Contains(o, ":::") }, "closes"},
		"12 quote then :::":         {":::details A\n> q\n:::\nfooter\n", func(o string) bool { return zzOutside(o, "footer") && !strings.Contains(o, ":::") }, "closes"},
		"13 unclosed fence":         {":::details A\n```\ncode\n:::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") }, "footer outside (trade-off)"},
		"14 in alert":               {"> [!NOTE]\n> :::details S\n> body\n> :::\n> after\n", func(o string) bool { return zzOutside(o, "after") }, "after outside"},
		"15 in list item":           {"- item\n  :::details S\n  body\n  :::\n  after\n", func(o string) bool { return zzOutside(o, "after") }, "after outside"},
		"16 list item ::: indented": {":::details A\n- item\n\n  :::\n\nfooter\n", func(o string) bool { return zzOutside(o, "footer") }, "list ::: is the list's text; footer outside?"},
		"17 tab close in quote":     {"> :::details S\n> body\n> \t:::\n> after\n", func(o string) bool { return zzOutside(o, "after") }, "after outside"},
		"18 indented 4 not closer":  {":::details A\n    :::\nx\n:::\n\nfooter\n", func(o string) bool { return zzInside(o, "<p>x</p>") && zzOutside(o, "footer") }, "x inside"},
	}
	names := make([]string, 0, len(shapes))
	for n := range shapes {
		names = append(names, n)
	}
	sort.Strings(names)
	designs := []string{"Z", "A", "B", "C", "E"}
	var b strings.Builder
	fmt.Fprintf(&b, "| shape | %s |\n|---|%s\n", strings.Join(designs, " | "), strings.Repeat("---|", len(designs)))
	pass := map[string]int{}
	for _, n := range names {
		fmt.Fprintf(&b, "| %s |", n)
		for _, d := range designs {
			zzDesign = d
			_, out, depth := parseDetails(t, shapes[n].src)
			r := "ok"
			if !shapes[n].ok(out) || depth != 0 {
				r = "FAIL"
			} else {
				pass[d]++
			}
			fmt.Fprintf(&b, " %s |", r)
			if os.Getenv("ZZV") == d+n[:2] {
				t.Logf("%s %s:\n%s", d, n, out)
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "| pass |")
	for _, d := range designs {
		fmt.Fprintf(&b, " %d/%d |", pass[d], len(names))
	}
	b.WriteString("\n| parse 120KB / plain |")
	big := strings.Repeat(":::details a\n", 3) + strings.Repeat("- item\n  > q\n\n", 8000)
	plain := strings.Repeat("- item\n  > q\n\n", 8000)
	for _, d := range designs {
		zzDesign = d
		t0 := time.Now()
		parseDetails(t, plain)
		p := time.Since(t0)
		t0 = time.Now()
		parseDetails(t, big)
		fmt.Fprintf(&b, " %.1fx |", float64(time.Since(t0))/float64(p))
	}
	b.WriteString("\n")
	t.Log("\n" + b.String())
	zzDesign = "E"
}
