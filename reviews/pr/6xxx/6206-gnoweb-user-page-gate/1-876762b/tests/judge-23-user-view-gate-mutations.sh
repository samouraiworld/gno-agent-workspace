#!/usr/bin/env bash
# NOT AUDITED — AI-generated tooling. Review before executing in any privileged context.
#
# Equivalence harness for candidate #23: does the one-table rewrite of the five
# user-view tests detect what the five originals detect? Five mutations of
# gno.land/pkg/gnoweb/handler_http.go, each run against both test files.
# Measured at 876762bdf with go1.25.9: M1-M4 fail on both, M5 fails only on the
# table — an address's contribution list silently stops being fetched and the
# five originals stay green.
#
# Repro from a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6206/head && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
#   curl -fsSL -o /tmp/table.go https://raw.githubusercontent.com/samouraiworld/gno-agent-workspace/main/reviews/pr/6xxx/6206-gnoweb-user-page-gate/1-876762b/tests/b1-refactor-r2-user-view-gate-table.go
#   bash <this file> /tmp/table.go
set -eu
TABLE_SRC=${1:?path to b1-refactor-r2-user-view-gate-table.go}
T=gno.land/pkg/gnoweb/handler_http_test.go
H=gno.land/pkg/gnoweb/handler_http.go
D=$(mktemp -d)
cp "$T" "$D/orig_test.go"; cp "$H" "$D/h.bak"
python3 - "$D" "$TABLE_SRC" "$T" <<'PY'
import sys
d, src, t = sys.argv[1:4]
new = open(src).read().split("---- replacement block ----\n", 1)[1]
open(d + "/table_test.go", "w").writelines(open(t).readlines()[:2003] + [new])
PY
run() { go test ./gno.land/pkg/gnoweb/ -run TestHTTPHandler_GetUserView -count=1 -v 2>&1; }
for m in M1 M2 M3 M4 M5; do
  cp "$D/h.bak" "$H"
  case $m in
    # the registry lookup stops being skipped for an address
    M1) sed -i 's/if !isAddress && len(contribs) == 0 {/if len(contribs) == 0 {/' "$H" ;;
    # an unresolvable name is served a page instead of a 404
    M2) sed -i 's/^\t\tif !exists {$/\t\tif false \&\& !exists {/' "$H" ;;
    # the length cap stops rejecting an over-long name before the chain call
    M3) sed -i 's/len(username) > maxUsernameLen || //' "$H" ;;
    # ResolveName's "(false bool)" answer stops being recognised
    M4) sed -i 's/case "(false bool)":/case "(false boolX)":/' "$H" ;;
    # an address's contributions stop being listed
    M5) perl -0pi -e 's/\tcontribs, realmCount, err := h\.buildContributions\(ctx, username\)\n/\tvar contribs []components.UserContribution\n\tvar realmCount int\n\tif !isAddress {\n\t\tcontribs, realmCount, err = h.buildContributions(ctx, username)\n\t}\n/' "$H" ;;
  esac
  for v in orig table; do
    cp "$D/${v}_test.go" "$T"
    echo "$m/$v -> $(run | grep -E '^(ok|FAIL)' | tail -1)"
  done
done
cp "$D/h.bak" "$H"; cp "$D/orig_test.go" "$T"; rm -rf "$D"
