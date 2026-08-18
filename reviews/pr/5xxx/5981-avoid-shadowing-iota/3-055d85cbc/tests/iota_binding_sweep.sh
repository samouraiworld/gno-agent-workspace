#!/usr/bin/env bash
# Runs every gno binding form that can name an identifier "iota" against the
# merge base and against PR 5981's head, and prints one row per form.
#
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 5981 -R gnolang/gno && git checkout 055d85cbc
#   bash <this file>
#
# The PR rejects "iota" at every name-binding site. Nine of the rows run at the
# merge base and are rejected at the head; those are the forms an already
# deployed package could be using today. RUNS on both sides is a form neither
# side binds, so the guard does not reach it.
set -u

root=$(git rev-parse --show-toplevel)
base=$(git merge-base origin/master HEAD)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo ":: building gno at merge base $base"
git -C "$root" worktree add -q "$tmp/base" "$base"
(cd "$tmp/base" && go build -o "$tmp/gno-base" ./gnovm/cmd/gno)
echo ":: building gno at HEAD $(git rev-parse --short HEAD)"
(cd "$root" && go build -o "$tmp/gno-head" ./gnovm/cmd/gno)

mkdir -p "$tmp/src"
w() { printf '%s\n' "$2" > "$tmp/src/$1.gno"; }

w shortvar            'package main
func main() { iota := 5; _ = iota }'
w var_local           'package main
func main() { var iota int; _ = iota }'
w const_local         'package main
func main() { const iota = 1; _ = iota }'
w type_local          'package main
func main() { type iota int; var x iota; _ = x }'
w func_decl           'package main
func iota() {}
func main() { iota() }'
w param_used          'package main
func f(iota int) int { return iota }
func main() { println(f(3)) }'
w param_unused        'package main
func f(iota int) { println("hi") }
func main() { f(3) }'
w param_second        'package main
func f(a, iota int) { println("hi") }
func main() { f(1, 2) }'
w result_named        'package main
func f() (iota int) { return 5 }
func main() { println(f()) }'
w recv                'package main
type T int
func (iota T) M() { println("m") }
func main() { var t T; t.M() }'
w funclit_param       'package main
func main() { g := func(iota int) { println("hi") }; g(1) }'
w range_key           'package main
func main() { s := []int{1, 2}; for iota := range s { _ = iota } }'
w range_value         'package main
func main() { s := []int{1, 2}; for _, iota := range s { _ = iota } }'
w forinit             'package main
func main() { for iota := 0; iota < 2; iota++ { println(iota) } }'
w forinit_nocond      'package main
func main() { for iota := 0; ; { println(iota); break } }'
w forinit_multi       'package main
func main() { for iota, j := 0, 1; iota < 2; iota++ { println(iota + j) } }'
w forinit_unused      'package main
func main() { for iota := 0; ; { println("hi"); break } }'
w if_init             'package main
func main() { if iota := 0; iota == 0 { println("z") } }'
w switch_init         'package main
func main() { switch iota := 0; iota { case 0: println("z") } }'
w typeswitch          'package main
func main() { var x interface{} = 5; switch iota := x.(type) { case int: println(iota) } }'
w label               'package main
func main() { iota: for { break iota } }'
w struct_field        'package main
type T struct{ iota int }
func main() { t := T{iota: 3}; println(t.iota) }'
w method_name         'package main
type T int
func (t T) iota() int { return 7 }
func main() { var t T; println(t.iota()) }'
w const_block         'package main
const ( a = iota
 b
 c )
func main() { println(a + b + c) }'
w const_in_for        'package main
func main() { for i := 0; i < 2; i++ { const ( a = iota
 b ); println(a + b) } }'
w var_pkg             'package main
var iota int
func main() { println(iota) }'
w type_pkg            'package main
type iota int
func main() { var x iota; println(int(x)) }'
w import_alias        'package main
import iota "strings"
func main() { println(iota.ToUpper("a")) }'
w complit_key         'package main
type T struct{ iota int }
func main() { t := T{iota: 1}; println(t.iota) }'
w len_param           'package main
func f(len int) int { return len }
func main() { println(f(3)) }'

classify() {
	out=$("$1" run "$tmp/src/$2.gno" 2>&1)
	if printf '%s' "$out" | grep -q 'builtin identifiers cannot be shadowed'; then
		echo REJECT
	elif printf '%s' "$out" | grep -qi 'panic\|error'; then
		echo OTHER-ERR
	else
		echo RUNS
	fi
}

printf '\n%-22s %-10s %-10s\n' FORM BASE HEAD
newly=0
for f in "$tmp"/src/*.gno; do
	n=$(basename "$f" .gno)
	b=$(classify "$tmp/gno-base" "$n")
	h=$(classify "$tmp/gno-head" "$n")
	printf '%-22s %-10s %-10s\n' "$n" "$b" "$h"
	[ "$b" = RUNS ] && [ "$h" = REJECT ] && newly=$((newly + 1))
done
printf '\nran at the merge base, rejected at the head: %d\n' "$newly"

git -C "$root" worktree remove --force "$tmp/base"
