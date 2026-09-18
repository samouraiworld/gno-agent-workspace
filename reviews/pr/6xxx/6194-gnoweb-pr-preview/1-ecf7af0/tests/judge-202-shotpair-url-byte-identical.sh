#!/usr/bin/env bash
# judge-202 — ShotPair.URL (misc/gnopreview/shots.go:38) is a second derivation
# of urlOf(Realm): dropping the field and spelling the link the way comment.go
# already spells it emits a byte-identical PR comment. shots.go 213 -> 212.
# Fails (diff non-empty, or a red test) if the two derivations ever differ.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   bash judge-202-shotpair-url-byte-identical.sh    # run from the repo root
#
# Measured at ecf7af0f2 with go1.25.9: comments identical, shots.go 213 -> 212,
# gofmt/vet silent, "ok github.com/gnolang/gno/misc/gnopreview".
set -eu
cd "$(git rev-parse --show-toplevel)/misc/gnopreview"
tmp=$(mktemp -d)
trap 'git checkout -- . ; rm -f zz_judge_emit_test.go ; rm -rf "$tmp"' EXIT

# An emitter over the same fixture TestCommentBeforeAfter uses, so the two
# phases are compared on the bytes a reviewer actually reads.
cat > zz_judge_emit_test.go <<'EOF'
package main

import (
	"os"
	"testing"
)

func TestJudgeEmitComment(t *testing.T) {
	out := os.Getenv("JUDGE_EMIT")
	if out == "" {
		t.Skip("emitter: set JUDGE_EMIT") // so the package's own suite stays green
	}
	p := &Plan{
		ChangedRealms: []string{"gno.land/r/x/leaf", "gno.land/r/x/fresh"},
		Realms:        []string{"gno.land/r/x/fresh", "gno.land/r/x/leaf"},
		Pairs: []ShotPair{
			{Realm: "gno.land/r/x/leaf", Before: "_shots/r-x-leaf-before.png", After: "_shots/r-x-leaf-after.png", URL: "r/x/leaf/"},
			{Realm: "gno.land/r/x/fresh", After: "_shots/r-x-fresh-after.png", URL: "r/x/fresh/", New: true},
		},
	}
	if err := os.WriteFile(out, []byte(Comment(p, "https://example.test/pr-5", "5")), 0o644); err != nil {
		t.Fatal(err)
	}
}
EOF

echo "== BEFORE: shots.go $(wc -l < shots.go) lines"
JUDGE_EMIT=$tmp/before.md go test -run TestJudgeEmitComment ./... >/dev/null
# The hrefs the rewrite must reproduce already hold at head.
grep -c '<a href="https://example.test/pr-5/r/x/leaf/">' "$tmp/before.md"

# The rewrite: drop the field, spell the link from the realm at both pairGrid
# sites, drop the hand-written URL: from every fixture.
python3 - <<'PY'
import re, pathlib
s = pathlib.Path("shots.go")
t = s.read_text()
t = t.replace('\tURL    string `json:"url"` // the after page, for the link behind the image\n', '')
t = t.replace('pair := ShotPair{Realm: r, URL: path.Dir(afterFile) + "/", New: newRealms[r]}',
              'pair := ShotPair{Realm: r, New: newRealms[r]}')
s.write_text(t)

c = pathlib.Path("comment.go")
t = c.read_text()
t = t.replace('`<a href="%s/%s"><img src="%s/%s" width="600" alt="%s"></a>`',
              '`<a href="%s%s/"><img src="%s/%s" width="600" alt="%s"></a>`')
t = t.replace('`<td width="50%%"><a href="%s/%s"><img src="%s/%s" width="100%%" alt="%s after">',
              '`<td width="50%%"><a href="%s%s/"><img src="%s/%s" width="100%%" alt="%s after">')
t = t.replace('base, p.URL, base, p.After, p.Realm, note))',
              'base, urlOf(p.Realm), base, p.After, p.Realm, note))')
t = t.replace('base, p.URL, base, p.After, p.Realm))',
              'base, urlOf(p.Realm), base, p.After, p.Realm))')
c.write_text(t)

for f in ("plan_test.go", "zz_judge_emit_test.go"):
    p = pathlib.Path(f)
    p.write_text(re.sub(r', URL: "[^"]*"|URL: "[^"]*", ', '', p.read_text()))
PY

echo "== AFTER: shots.go $(wc -l < shots.go) lines"
gofmt -l . && go vet ./... && go test ./...
JUDGE_EMIT=$tmp/after.md go test -run TestJudgeEmitComment ./... >/dev/null
if diff -u "$tmp/before.md" "$tmp/after.md"; then
	echo "PASS: comment byte-identical without ShotPair.URL"
else
	echo "FAIL: the two derivations differ"
	exit 1
fi
