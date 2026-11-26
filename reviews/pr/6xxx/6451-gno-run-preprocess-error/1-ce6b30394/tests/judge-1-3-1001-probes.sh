#!/bin/sh
# judge probes, run once per binary: $1 = gno binary
G=$1; T=$(mktemp -d); cd "$T"
mk() { mkdir -p "$1"; cat > "$1/main.gno"; }
mk p1_missing_import <<'X'
package main

import "gno.land/p/nope/x"

func main() { x.F() }
X
mk p2_abs <<'X'
package main

func main() {
	var u uint8 = -1
	println(u)
}
X
mk p5_builtinrecv <<'X'
package main

func (int) M() {}

func main() {}
X
mk p8_rel <<'X'
package main

func main() {
	var u uint8 = -1
	println(u)
}
X
mkdir -p p6_lint; printf 'module = "gno.land/p/f1/l2"\ngno = "0.9"\n' > p6_lint/gnomod.toml
cat > p6_lint/l2.gno <<'X'
package l2

func F(cur realm) {}
X
mkdir -p p7_dup; printf 'package main\n\nfunc helper() int { return 1 }\n\nfunc main() { println(helper()) }\n' > p7_dup/a.gno; printf 'package main\n\nfunc helper() int { return 2 }\n' > p7_dup/b.gno
run() { n=$1; shift; "$@" >out 2>err; rc=$?; echo "== $n rc=$rc"; head -c 300 out; head -3 err | cut -c1-260; }
run p1 "$G" run ./p1_missing_import/
run p2 "$G" run "$T/p2_abs/main.gno"
run p5 "$G" run ./p5_builtinrecv/
mkdir -p sub && cd sub
run p8 "$G" run -pkgpath gno.land/r/x/y ../p8_rel/main.gno
cd ..
run p6lint sh -c "cd p6_lint && $G lint ."
run p7dir "$G" run ./p7_dup/
run p7pkgpath "$G" run -pkgpath example.com/main ./p7_dup/
rm -rf "$T"
