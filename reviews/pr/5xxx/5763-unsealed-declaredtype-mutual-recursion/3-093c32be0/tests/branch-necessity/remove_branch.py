#!/usr/bin/env python3
# Deletes one kind's case from fillTypeInPlace (gnovm/pkg/gnolang/types.go),
# builds gno from that tree, and runs the fixture that depends on that branch.
# Every branch present at 093c32be0 is load-bearing: each removal breaks its
# own fixture and nothing else here.
#
#   # from a local clone of gnolang/gno:
#   gh pr checkout 5763 -R gnolang/gno
#   ./remove_branch.py SliceType /path/to/clone
import subprocess, sys, os

kind, root = sys.argv[1], sys.argv[2]
p = os.path.join(root, "gnovm/pkg/gnolang/types.go")
s = open(p).read()
block = "\tcase *%s:\n\t\tif src, ok := src.(*%s); ok {\n\t\t\t*dst = *src\n\t\t\treturn true\n\t\t}\n" % (kind, kind)
if block not in s:
    sys.exit("anchor not found for %s" % kind)
open(p, "w").write(s.replace(block, "", 1))
print("removed", kind, "- now rebuild and run the matching fixture")
