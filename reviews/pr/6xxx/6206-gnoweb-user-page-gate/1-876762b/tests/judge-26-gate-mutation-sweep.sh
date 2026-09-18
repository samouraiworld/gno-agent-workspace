#!/usr/bin/env bash
# What the whole gnoweb suite says when the /u/ registry gate is deleted.
# Measured: 1 of 7 GetUserView tests plus 1 of 3 new /u/ routes is blind to the
# deletion; the other 5 go red. Run at gnolang/gno head 876762bdf.
#
# Repro from a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
#   git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
#   bash <this file>
#
# Toolchain: go1.25.9, the version go.mod pins.
set -u

echo "== unmutated head =="
go test ./gno.land/pkg/gnoweb/ -run 'GetUserView' -count=1 -v 2>&1 |
  grep -E '^\s*--- (PASS|FAIL): '

# handler_http.go:618-627 is the whole gate:
#   if !isAddress && len(contribs) == 0 { ... userExists ... 404 }
sed -i '618,627d' gno.land/pkg/gnoweb/handler_http.go

echo "== gate deleted: unit =="
go test ./gno.land/pkg/gnoweb/ -run 'GetUserView' -count=1 -v 2>&1 |
  grep -E '^\s*--- (PASS|FAIL): '

echo "== gate deleted: integration routes =="
go test ./gno.land/pkg/gnoweb/ -run 'TestRoutes' -count=1 -v 2>&1 |
  grep -E '^\s*--- (PASS|FAIL): TestRoutes/test_route_/u/'

git checkout -- gno.land/pkg/gnoweb/handler_http.go
