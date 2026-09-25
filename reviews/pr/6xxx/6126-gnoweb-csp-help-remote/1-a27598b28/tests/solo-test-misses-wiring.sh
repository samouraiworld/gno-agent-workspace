#!/usr/bin/env bash
# Repro: the test added by gnolang/gno#6126 stays green when setupWeb's call is
# reverted to the NodeRemote it replaced, so the regression it names is not pinned.
#
# From a plain clone:
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin a27598b2867c226470b01c23f48ccdf5089c4e87 && git checkout --detach a27598b2867c226470b01c23f48ccdf5089c4e87
#   bash <this file>
#
# Expected if the test pinned the fix: FAIL after the revert. Observed: ok.
set -euo pipefail
cd gno.land
sed -i 's|newSecureHeadersMiddleware(app, !cfg.noStrict, appcfg)|SecureHeadersMiddleware(app, !cfg.noStrict, appcfg.NodeRemote)|' cmd/gnoweb/main.go
git diff --stat
go test ./cmd/gnoweb -run 'TestSecureHeadersMiddleware|TestSetupWeb' -count=1 -v 2>&1 | grep -E '^(=== RUN|--- |ok|FAIL)'
git checkout -- cmd/gnoweb/main.go
