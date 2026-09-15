#!/usr/bin/env bash
# Candidate #12: tm2/pkg/bft/version/version_test.go:16
#
# Claim: TestProtocolVersionsAgree's assert loop is unreachable, because each
# of the six protocol-version constants is transitively guarded by an init()
# that panics before the test binary reaches any assertion.
#
# from a local clone of gnolang/gno:
#   gh pr checkout 6177 -R gnolang/gno && git checkout 607942b78fa4fdf6f378fecce32bc1d1d984ab8e
#   bash 12-protocol-versions-agree-unreachable.sh
set -euo pipefail

echo "=== bump only tm2/pkg/p2p/version.Version ==="
sed -i 's/Version = "v1.0.0-rc.0"/Version = "v1.1.0"/' tm2/pkg/p2p/version/version.go
go test -run TestProtocolVersionsAgree -count=1 ./tm2/pkg/bft/version/ 2>&1 || true
git checkout -- tm2/pkg/p2p/version/version.go

echo
echo "=== bump only tm2/pkg/crypto.Version ==="
sed -i 's/Version = "v1.0.0-rc.0"/Version = "v1.1.0"/' tm2/pkg/crypto/version.go
go test -run TestProtocolVersionsAgree -count=1 ./tm2/pkg/bft/version/ 2>&1 || true
git checkout -- tm2/pkg/crypto/version.go
