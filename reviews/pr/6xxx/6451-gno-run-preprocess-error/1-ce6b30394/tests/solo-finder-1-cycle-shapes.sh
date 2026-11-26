#!/bin/sh
# Repro: `gno run` still aborts with a Go panic (exit 2) on every shape of a
# package-level initialization cycle, not only the `var a = f(); func f() int
# { return a }` shape already reported on the pull request. The panic is the
# plain string raised by resolveEffectiveDeps (gnovm/pkg/gnolang/preprocess.go,
# "circular dependency: ..."), which runFiles' type switch sends to
# `default: panic(r)`.
#
# From a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6451/head && git checkout ce6b303940399f5fc07e1d2eea72ae03eb83ed4d
#   (cd gnovm && go build -o /tmp/gno-6451 ./cmd/gno)
#   GNO=/tmp/gno-6451 sh solo-finder-1-cycle-shapes.sh
#
# Measured at ce6b30394: every case exits 2 with
#   panic: main/<case>/main.gno:<line>:<col>: circular dependency: ... [recovered, repanicked]
# The non-cycle recursive shapes (const a = b; const b = a / var x = x /
# type A = B; type B = A) exit 1 with a (code=gnoPreprocessError) line, since
# they raise inside Preprocess and are wrapped as *gno.PreprocessError.
set -u
GNO=${GNO:?set GNO to a gno binary built from the PR head}
d=$(mktemp -d)
cd "$d"
mk() { mkdir -p "$1"; cat > "$1/main.gno"; }

# 1. the shape already reported: through a function.
mk cycle_func <<'EOF'
package main

var a = f()

func f() int { return a }

func main() { println(a) }
EOF
# 2. through a method.
mk cycle_method <<'EOF'
package main

type T struct{}

func (T) M() int { return a }

var a = T{}.M()

func main() { println(a) }
EOF
# 3. through a closure in the initializer.
mk cycle_closure <<'EOF'
package main

var a = func() int { return a }()

func main() { println(a) }
EOF
# 4. through two functions.
mk cycle_twohop <<'EOF'
package main

var a = f()

func f() int { return g() }

func g() int { return a }

func main() { println(a) }
EOF

fail=0
for c in cycle_func cycle_method cycle_closure cycle_twohop; do
	"$GNO" run "./$c/" >/dev/null 2>"$c.err"
	rc=$?
	# current behaviour (ce6b30394): Go panic, exit code 2.
	want=2
	# fixed behaviour: one "<file>:<line>:<col>: circular dependency ... (code=gnoPreprocessError)" line, exit code 1.
	# want=1
	if [ "$rc" -ne "$want" ]; then
		echo "FAIL $c: exit $rc, want $want"; fail=1
	else
		echo "ok   $c: exit $rc"
	fi
	head -1 "$c.err"
done
rm -rf "$d"
exit $fail
