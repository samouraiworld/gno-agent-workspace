// solo-finder-location-delta: which BlockNode Locations gnolang/gno#6107 moves.
//
// Repro from a plain clone of gnolang/gno:
//   git worktree add --detach ../h 98ba8a8e4a1981b9aad23c5487387300d6eda15e
//   git worktree add --detach ../b 26c0a7b32a1ec0c194cb97df893ef77778e1a3e5
//   cp solo-finder-location-delta.go ../h/gnovm/pkg/gnolang/zz_locdump_test.go
//   cp solo-finder-location-delta.go ../b/gnovm/pkg/gnolang/zz_locdump_test.go
//   ROOT=$PWD/../h/examples:$PWD/../h/gnovm/stdlibs:$PWD/../h/gnovm/tests/files
//   (cd ../h/gnovm/pkg/gnolang && LOCDUMP_ROOT=$ROOT LOCDUMP_OUT=/tmp/head.tsv go test -count=1 -run TestSoloFinderLocDump -v .)
//   (cd ../b/gnovm/pkg/gnolang && LOCDUMP_ROOT=$ROOT LOCDUMP_OUT=/tmp/base.tsv go test -count=1 -run TestSoloFinderLocDump -v .)
//   paste /tmp/base.tsv /tmp/head.tsv | awk -F'\t' '$4!=$8 {print $3"\tbase="gsub(/'"'"'/,"",$4)"\thead="gsub(/'"'"'/,"",$8)}' | sort | uniq -c
//
// Measured 2026-09-25, go1.25.9, same inputs (the head tree's files) at both commits:
//   head: files=4253 blocknodes=66204 duplicate_locations=0
//   base: files=4253 blocknodes=66204 duplicate_locations=254
//   changed Locations (Num counted by trailing apostrophes):
//     254 *gnolang.IfStmt     base=0 head=2   (the else-if's own IfStmt)
//     254 *gnolang.IfCaseStmt base=0 head=1   (the synthetic else wrapper)
//     136 *gnolang.IfCaseStmt base=1 head=3   (empty else of that IfStmt)
//   no FuncDecl, FuncLitExpr or FileNode Location changes (the kinds
//   DeclaredType.ParentLoc and persisted RefNode sources carry); 52 files
//   under examples/ carry a moved Location.
package gnolang

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSoloFinderLocDump(t *testing.T) {
	out := os.Getenv("LOCDUMP_OUT")
	if out == "" {
		t.Skip("LOCDUMP_OUT unset")
	}
	var lines []string
	var m *Machine
	nfiles, dups := 0, 0
	for _, root := range strings.Split(os.Getenv("LOCDUMP_ROOT"), ":") {
		filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".gno") {
				return nil
			}
			bz, _ := os.ReadFile(p)
			var fn *FileNode
			func() {
				defer func() { recover() }()
				f, err := m.ParseFile(filepath.Base(p), string(bz))
				if err == nil {
					fn = f
				}
			}()
			if fn == nil {
				return nil
			}
			nfiles++
			setNodeLines(fn)
			setNodeLocations("x", filepath.Base(p), fn)
			idx := 0
			seen := map[Location]bool{}
			Transcribe(fn, func(ns []Node, ftype TransField, index int, n Node, stage TransStage) (Node, TransCtrl) {
				if stage != TRANS_ENTER {
					return n, TRANS_CONTINUE
				}
				if bn, ok := n.(BlockNode); ok {
					loc := bn.GetLocation()
					if seen[loc] {
						dups++
					}
					seen[loc] = true
					lines = append(lines, fmt.Sprintf("%s\t%d\t%T\t%s", p, idx, bn, loc.Span.String()))
					idx++
				}
				return n, TRANS_CONTINUE
			})
			return nil
		})
	}
	os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
	t.Logf("files=%d blocknodes=%d duplicate_locations=%d", nfiles, len(lines), dups)
}
