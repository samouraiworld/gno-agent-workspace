#!/usr/bin/env bash
# Asserts the whole diff is one address substitution: added and removed lines are
# equal as multisets once both addresses map to the same token. Measured at
# f2bdb07b0: three gen-genesis.sh comment lines differ, every other line matches.
#
# Run: from a gno checkout:
# gh pr checkout 6131 -R gnolang/gno && git checkout f2bdb07b0
# curl -fsSL -o /tmp/normalised-diff.sh \
#   https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6131-govdao-t1-multisig-address/1-f2bdb07b0/tests/normalised-diff.sh
# bash /tmp/normalised-diff.sh

set -u
OLD=g1rp7cmetn27eqlpjpc4vuusf8kaj746tysc0qgh
NEW=g1sze988ga0a7sj5583cu3xt6m4vkxru4uwh6dmf
base=$(git merge-base origin/master HEAD)
echo "merge base: $base"

git diff -U0 "$base" HEAD | grep '^+' | grep -v '^+++' | cut -c2- | sed "s/$NEW/ADDR/g" | sort >/tmp/6131-added
git diff -U0 "$base" HEAD | grep '^-' | grep -v '^---' | cut -c2- | sed "s/$OLD/ADDR/g" | sort >/tmp/6131-removed
printf 'added %s lines, removed %s lines\n' "$(wc -l </tmp/6131-added)" "$(wc -l </tmp/6131-removed)"

echo
echo "lines that are not a pure address swap:"
diff /tmp/6131-removed /tmp/6131-added && echo "  none"

echo
echo "occurrences: old $(git grep -o -e "$OLD" HEAD | wc -l), new $(git grep -o -e "$NEW" HEAD | wc -l)"
echo "files holding both addresses:"
comm -12 \
  <(git grep -l -e "$OLD" HEAD | sed 's/^[^:]*://' | sort) \
  <(git grep -l -e "$NEW" HEAD | sed 's/^[^:]*://' | sort) | sed 's/^/  /'

echo
echo "encoded forms of the old address anywhere in the tree:"
git grep -In -e 187d8de57357b20f8641c559ce4127b765eae964 \
             -e 'GH2N5XNXsg+GQcVZzkEnt2Xq6WQ=' HEAD || echo "  none"
