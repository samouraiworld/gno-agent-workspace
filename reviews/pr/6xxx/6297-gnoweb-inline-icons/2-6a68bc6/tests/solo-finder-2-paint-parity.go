// solo-finder-2: dumps each outline icon of icons/*.svg as its source symbol and as the
// registry glyph, for solo-finder-2-paint-parity.sh; repro commands are in that file.
package markdown

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestZZPaintDump writes, for every icon of icons/*.svg that the registry
// holds, the source symbol as a standalone SVG and the registry glyph as the
// renderer writes it, both colored black, into $PAINT_DUMP_DIR.
func TestZZPaintDump(t *testing.T) {
	dir := os.Getenv("PAINT_DUMP_DIR")
	if dir == "" {
		t.Skip("PAINT_DUMP_DIR unset")
	}
	re := regexp.MustCompile(`(?s)<symbol id="ico-([^"]+)"([^>]*)>(.*?)</symbol>`)
	n := 0
	for _, file := range []string{"drawn.svg", "vendored.svg"} {
		data, err := os.ReadFile("icons/" + file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllSubmatch(data, -1) {
			name := string(m[1])
			g, ok := iconRegistry[name]
			if !ok {
				continue
			}
			const open = `<svg xmlns="http://www.w3.org/2000/svg" color="black" `
			src := open + string(m[2]) + ">" + string(m[3]) + "</svg>"
			gen := open + g.head + ">" + g.body + "</svg>"
			if err := os.WriteFile(filepath.Join(dir, name+".src.svg"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name+".gen.svg"), []byte(gen), 0o644); err != nil {
				t.Fatal(err)
			}
			n++
		}
	}
	t.Logf("dumped %d icons", n)
}
