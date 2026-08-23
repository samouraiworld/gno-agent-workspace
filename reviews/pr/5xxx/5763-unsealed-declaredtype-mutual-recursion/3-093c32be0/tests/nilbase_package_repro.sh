#!/usr/bin/env bash
# Run: from a local clone of gnolang/gno:
#   gh pr checkout 5763 -R gnolang/gno && git checkout 093c32be0
#   bash reviews/.../tests/nilbase_package_repro.sh
#
# A package carrying `type T2 T1; type T1 struct{Next *T3; ...}; type T3 T2`
# leaves T3 with a nil underlying type. `gno lint` reports nothing; `gno test`
# dies in copyTypeWithRefs, the realm type-persistence path, with no file or
# line. 0397fc87f rejects the same package with a positioned
# "should not happen (code=gnoPreprocessError)".
set -u
ROOT=$(pwd)
mkdir -p examples/gno.land/p/demo/nilbase
cat > examples/gno.land/p/demo/nilbase/gnomod.toml <<'EOF'
module = "gno.land/p/demo/nilbase"
gno = "0.9"
EOF
cat > examples/gno.land/p/demo/nilbase/nilbase.gno <<'EOF'
package nilbase

type T2 T1

type T1 struct {
	Next *T3
	Val  int
}

type T3 T2

func Hello() string { return "hello" }
EOF
cat > examples/gno.land/p/demo/nilbase/nilbase_test.gno <<'EOF'
package nilbase

import "testing"

func TestHello(t *testing.T) {
	if Hello() != "hello" {
		t.Fatal("bad")
	}
}
EOF
cd examples/gno.land/p/demo/nilbase
echo "### gno lint:"
GNOROOT="$ROOT" go run "$ROOT/gnovm/cmd/gno" lint . 2>&1 | head -5
echo "### gno lint exit: ${PIPESTATUS[0]}"
echo "### gno test:"
GNOROOT="$ROOT" go run "$ROOT/gnovm/cmd/gno" test . 2>&1 | head -12
cd "$ROOT"
rm -rf examples/gno.land/p/demo/nilbase
