#!/usr/bin/env bash
# The keyed proof's acceptance criteria are not pinned by any test. Both guards
# mutated below can be deleted and grc20routes_test.gno stays green, because
# every one of its 20 Test funcs proves the one realm it runs inside (selfPath =
# gno.land/r/nt/grc20routes/v0, which declares no Approve at all: the grep below
# counts 0). The suite's oracle is "two ledger allowances moved" — the
# mechanism — not "the realm routes by symbol", the claim BeginKeyedProof's
# doc at grc20routes.gno:336-337 makes.
# Measured at head 3aa06ef75, this file's own run: baseline exit=0 ok 10.69s,
# mutation 1 exit=0 ok 20.92s, mutation 2 exit=0 ok 59.38s. Nothing goes red.
#
# # from a local clone of github.com/gnolang/gno:
# git checkout 3aa06ef757b87d731a3e554e96c7540a209fd9a3
# go build -o /tmp/gno-6187 ./gnovm/cmd/gno
# bash <this file> /tmp/gno-6187
#
# Runs in a throwaway copy of examples/; the clone is never modified.

set -u
GNO="${1:-gno}"
PKG=gno.land/r/nt/grc20routes/v0
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

cp -r examples "$WORK/examples"
SRC="$WORK/examples/$PKG/grc20routes.gno"
cp "$SRC" "$WORK/pristine.gno"

run() { # $1 = label
	local out
	out="$(cd "$WORK/examples" && "$GNO" test "./$PKG" 2>&1)"
	printf '%-42s exit=%d  %s\n' "$1" "$?" "$(printf '%s' "$out" | tail -1)"
}

# The realm under proof exposes no Approve of any arity, so the flag the suite
# asserts true at grc20routes_test.gno:195 describes a MsgCall nobody can send.
printf 'func Approve on the realm every test proves: '
grep -c '^func Approve' "$SRC"

run "baseline (unmutated)"

# --- mutation 1 -------------------------------------------------------------
# Delete the same-realm guard: the two tokens no longer have to be hosted by the
# realm the proof marks. No test owns a second realm, so none can notice.
python3 - "$SRC" <<'PY'
import sys
p = sys.argv[1]
s = open(p).read()
old = '''	if pathA != pathB {
		panic("grc20routes: both tokens must live in the same realm")
	}
'''
assert old in s, "guard not found"
open(p, "w").write(s.replace(old, '	_ = pathB // MUTATION 1\n'))
PY
run "mutation 1: same-realm guard deleted"
cp "$WORK/pristine.gno" "$SRC"

# --- mutation 2 -------------------------------------------------------------
# Loosen what the source itself labels "The load-bearing check" (:404-406) from
# "carries exactly the nonce claimed" to "carries anything at all". The one test
# that exercises Finish's rejection path, TestKeyedProofFailsWhenOnlyOneTokenMoves
# (:211), leaves the second allowance at 0, so the weakened form still aborts
# there and the test still passes.
python3 - "$SRC" <<'PY'
import sys
p = sys.argv[1]
s = open(p).read()
for key in ("A", "B"):
	old = '\tif allowanceOf(p.key%s, prover) != p.nonce%s {' % (key, key)
	assert old in s, old
	s = s.replace(old, '\tif allowanceOf(p.key%s, prover) == 0 { // MUTATION 2' % key)
open(p, "w").write(s)
PY
run "mutation 2: exact nonce no longer required"
cp "$WORK/pristine.gno" "$SRC"

cat <<'EOF'

The missing test, which cannot be written as a package test (gno package tests
cannot deploy a second realm) and belongs beside
gno.land/pkg/integration/testdata/grc20routes_keyed_proof.txtar: a realm that
hosts two registered tokens and exposes no keyed Approve must still read
source=convention after any sequence of calls. That txtar covers the true
positive (grc20factory, a real keyed realm) and one realm that cannot be proven
at all (wugnot, single token). Neither it nor the package suite holds a
multi-token realm that must NOT come out keyed.
EOF
