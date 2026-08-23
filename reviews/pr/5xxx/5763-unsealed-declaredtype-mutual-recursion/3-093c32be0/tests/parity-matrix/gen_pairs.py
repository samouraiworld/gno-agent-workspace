import os, textwrap

# base shapes: (name, T1 declaration line(s), body statements)
shapes = {
 "structptr": ("type T1 struct {\n\tNext *T2\n\tVal  int\n}",
               "var a T1\n\ta.Next = &T2{Val: 9}\n\tPRINT(a.Next.Val)"),
 "slice":     ("type T1 []T2",
               "var a T1\n\ta = append(a, T2{})\n\tPRINT(len(a))"),
 "map":       ("type T1 map[string]T2",
               "a := T1{}\n\ta[\"x\"] = T2{}\n\tPRINT(len(a))"),
 "funct":     ("type T1 func(int) T2",
               "var a T1\n\ta = func(i int) T2 { return nil }\n\tPRINT(a(1) == nil)"),
 "iface":     ("type T1 interface {\n\tM() int\n}",
               "var a T1 = impl{}\n\tPRINT(a.M())"),
 "ptr":       ("type T1 *T2",
               "var a T1\n\tPRINT(a == nil)"),
 "arrayptr":  ("type T1 [2]*T2",
               "var a T1\n\ta[0] = &T2{}\n\tPRINT(len(a))"),
}

extra = {"iface": "type impl struct{}\n\nfunc (impl) M() int { return 42 }\n"}

out = "/tmp/rev5763/matrix"
cases = []
for sname, (decl, body) in shapes.items():
    for kind, deriv in (("def", "type T2 T1"), ("alias", "type T2 = T1")):
        for order in ("t1first", "t2first"):
            name = "%s_%s_%s" % (sname, kind, order)
            parts = [decl, deriv] if order == "t1first" else [deriv, decl]
            src_body = "\n\n".join(parts)
            ex = extra.get(sname, "")
            gno = "package main\n\n%s\n\n%s\nfunc main() {\n\t%s\n\tprintln(\"ok\")\n}\n" % (
                src_body, ex, body.replace("PRINT", "println"))
            go = "package main\n\nimport \"fmt\"\n\n%s\n\n%s\nfunc main() {\n\t%s\n\tfmt.Println(\"ok\")\n}\n" % (
                src_body, ex, body.replace("PRINT", "fmt.Println"))
            os.makedirs("%s/go/%s" % (out, name), exist_ok=True)
            open("%s/go/%s/main.go" % (out, name), "w").write(go)
            open("%s/go/%s/go.mod" % (out, name), "w").write("module m\ngo 1.21\n")
            open("%s/gno/%s.gno" % (out, name), "w").write(gno)
            cases.append(name)
open("%s/cases.txt" % out, "w").write("\n".join(cases) + "\n")
print(len(cases), "cases")
