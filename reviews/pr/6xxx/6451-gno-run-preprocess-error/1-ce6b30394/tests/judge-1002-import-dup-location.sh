#!/bin/sh
# Asserts: a PreprocessError with no location in its message prints `.:0:` at the head.
# Needs a gno root whose examples hold gno.land/p/zz/dup with Helper declared in a.gno and b.gno.
# usage: sh judge-1002.sh <gno binary> <gno root>
G=$1; ROOT=$2; T=$(mktemp -d); cd "$T"
mkdir -p i1 other; printf 'package main\n\nimport "gno.land/p/zz/dup"\n\nfunc main() {\n\tprintln(dup.Helper())\n}\n' > i1/main.gno
"$G" run -root-dir "$ROOT" ./i1/ >out 2>err; echo "i1 rc=$?"; grep -m1 -v '^$' err | cut -c1-200; grep -m1 'stack 1' err | cut -c1-120
# absolute path to a file with a preprocess error, run from another directory
printf 'package main\n\nfunc main() {\n\tvar u uint8 = -1\n\tprintln(u)\n}\n' > i1/bad.gno; mkdir -p abs && mv i1/bad.gno abs/main.gno
cd other; "$G" run "$T/abs/main.gno" >/dev/null 2>../err2; echo "abs-from-other rc=$?"; head -1 ../err2 | cut -c1-200
rm -rf "$T"
