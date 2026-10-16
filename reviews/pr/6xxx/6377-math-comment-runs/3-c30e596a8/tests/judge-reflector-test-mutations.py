# Reflector check: revert each fix of c30e596a8 alone and run the tests the diff adds.
# Usage: python3 judge-reflector-test-mutations.py <gno worktree> gno.land/pkg/gnoweb/markdown/mathml
# Measured: all 8 reverts turn at least one added test red; baseline green.
import sys, subprocess, pathlib
W, P = sys.argv[1], sys.argv[2]
d = pathlib.Path(W) / P
M = {
 "M1 fixFences guard": ("tokenize.go", [("\t\tif set == 0 {\n\t\t\tout = append(out, toks[i])\n\t\t\tcontinue\n\t\t}\n", ""), ("if j == len(toks) {", "if set == 0 || j == len(toks) {")]),
 "M2 Empty last": ("tokenize.go", [("\tif b.idx >= b.last {\n\t\tb.idx = max(b.idx, len(b.Expr))\n\t\treturn true\n\t}\n\treturn false\n",
   "\tif b.idx >= len(b.Expr) {\n\t\treturn true\n\t}\n\ttemp := b.idx\n\tfor b.idx < len(b.Expr) && b.Expr[b.idx].Kind&(tokComment|tokWhitespace) > 0 {\n\t\tb.idx++\n\t}\n\tif b.idx >= len(b.Expr) {\n\t\treturn true\n\t}\n\tb.idx = temp\n\treturn false\n")]),
 "M3 colorArgMissing": ("commands.go", [("if name == \"color\" && colorArgMissing(b) {", "if false && colorArgMissing(b) {")]),
 "M4 cellEnd memo": ("tokenize.go", [("\tn := len(b.Expr)\n\tif b.cellEnds == nil {", "\tn := len(b.Expr)\n\tb.cellEnds = nil\n\tif b.cellEnds == nil {")]),
 "M5 matched bound": ("tokenize.go", [(" && i+t.MatchOffset <= len(b.Expr)", "")]),
 "M6 doFence nil": ("parse.go", [("\t\t\tn = converter.ProcessCommand(context&^ctxRoot, tok, b)\n\t\t\tif n == nil {\n\t\t\t\treturn nil\n\t\t\t}\n", "\t\t\tn = converter.ProcessCommand(context&^ctxRoot, tok, b)\n")]),
 "M7 class nil": ("commands.go", [("\tif n == nil {\n\t\t// \\class{a}{} has nothing to draw.\n\t\treturn nil\n\t}\n", "")]),
 "M8 GetNextToken matched": ("tokenize.go", [("\tif b.matched(b.idx, \"{\") {\n\t\tb.idx = temp\n\t\treturn result, &TokenBufferError{tbIsExprErr", "\tif b.Expr[b.idx].Kind&(tokEscaped|tokCurly|tokOpen) == (tokCurly | tokOpen) {\n\t\tb.idx = temp\n\t\treturn result, &TokenBufferError{tbIsExprErr")]),
}
run = "TestCommentKeepsTheWindow$|TestCommentRunIsLinear$|TestBareColorRunIsLinear$|TestSwitchEndsAtItsCell$|TestMalformedInputDoesNotCrash$|TestCrossingGroupIsRejected$|TestDelimiterAfterSpace$"
for name, (f, reps) in M.items():
    p = d / f; orig = p.read_text(); s = orig
    ok = True
    for a, b in reps:
        if s.count(a) != 1: print(name, "PATTERN COUNT", s.count(a)); ok = False; break
        s = s.replace(a, b)
    if not ok: continue
    p.write_text(s)
    r = subprocess.run(["go", "test", "-count=1", "-run", run, "-v", "./" + P + "/"], cwd=W, capture_output=True, text=True, timeout=600)
    fails = sorted({l.split()[2] for l in r.stdout.splitlines() if l.startswith("--- FAIL")})
    build = "build failed" if "build failed" in r.stdout + r.stderr or "[setup failed]" in r.stdout else ""
    print(f"{name}: {'RED ' + ','.join(fails) if fails else ('BUILD ' + r.stderr[-300:] if build else 'green')}")
    p.write_text(orig)
