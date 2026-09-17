#!/usr/bin/env bash
# Asserts installInheritedCur's `fr` is always m.PeekCallFrame(1), the
# precondition harnessSeedsCur's PeekCallFrame(2) rests on.
# Measurement: 47 calls across 5 filetests, 0 assertion failures at 6d88deba7.
# It PASSES at the reviewed head: no live defect behind candidate #23.
set -e
# from a local clone of gnolang/gno:
gh pr checkout 6196 -R gnolang/gno && git checkout 6d88deba7

python3 - <<'EOF'
p = 'gnovm/pkg/gnolang/op_call.go'
s = open(p).read()
old = 'func (m *Machine) installInheritedCur(fr *Frame, fv *FuncValue, b *Block) {\n'
assert s.count(old) == 1
# Panic if the caller ever hands a frame that is not the top call frame; the
# println counts the calls so a silent zero-execution run is visible.
new = old + '\tif fr != m.PeekCallFrame(1) {\n\t\tpanic("JUDGE-ASSERT-FAIL: fr is not PeekCallFrame(1)")\n\t}\n\tprintln("JUDGE-ASSERT-OK")\n'
open(p, 'w').write(s.replace(old, new))
EOF

go test ./gnovm/pkg/gnolang/ -v \
  -run 'TestFiles/(access2|func32|heap_item_value|issue-2449|alias_crosspkg_realm).gno$' \
  > /tmp/judge-23-assert.log 2>&1 || true
echo "assertion executions: $(grep -c JUDGE-ASSERT-OK /tmp/judge-23-assert.log)"
echo "assertion failures:   $(grep -c JUDGE-ASSERT-FAIL /tmp/judge-23-assert.log)"
grep -E '^(ok|FAIL)' /tmp/judge-23-assert.log

git checkout -- gnovm/pkg/gnolang/op_call.go
