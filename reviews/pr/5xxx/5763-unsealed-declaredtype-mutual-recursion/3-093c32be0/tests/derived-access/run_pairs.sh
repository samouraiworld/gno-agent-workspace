#!/usr/bin/env bash
# Runs every <case>.go / <case>.gno pair here and prints Go vs gno accept/reject
# plus stdout, for a gno binary built from the branch under test.
#
#   # from a local clone of gnolang/gno:
#   gh pr checkout 5763 -R gnolang/gno
#   go build -o /tmp/gno-head ./gnovm/cmd/gno
#   ./run_pairs.sh /tmp/gno-head
set -u
GNO=${1:?usage: run_pairs.sh <path-to-gno-binary>}
here=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
printf '%-34s %-24s %-24s\n' CASE GO GNO
for f in "$here"/*.gno; do
  n=$(basename "$f" .gno)
  mkdir -p "$tmp/$n"
  cp "$here/$n.go" "$tmp/$n/main.go"
  printf 'module m\ngo 1.21\n' > "$tmp/$n/go.mod"
  g=$( (cd "$tmp/$n" && go run . 2>&1) | tr '\n' '|' | cut -c1-22)
  h=$("$GNO" run "$f" 2>&1 | head -1 | cut -c1-22)
  printf '%-34s %-24s %-24s\n' "$n" "$g" "$h"
done
rm -rf "$tmp"
