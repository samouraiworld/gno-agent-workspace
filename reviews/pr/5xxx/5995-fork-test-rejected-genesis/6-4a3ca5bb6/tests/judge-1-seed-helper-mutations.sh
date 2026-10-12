#!/usr/bin/env bash
# Asserts seedCommittedChain still makes both committed-chain tests go red when the seed loses
# the height-0 marker or the committed height. Measured at 4a3ca5bb6: A reddens CommittedStatePreserved,
# B reddens CommittedStatePreserved and ProbeErrorKeepsCommittedState. Run from a gno checkout at that sha.
set -u
f=tm2/pkg/bft/node/node_test.go
run() { go test ./tm2/pkg/bft/node -run 'TestLoadStateFromDBOrGenesisDocProvider_(CommittedStatePreserved|ProbeErrorKeepsCommittedState)' -count=1 2>&1 | grep -E '^(--- |ok|FAIL)'; }
echo '== A: no height-0 marker'; sed -i '611{/SaveABCIResponses(db, 0/d}' $f; git diff --stat -- $f; run; git checkout -- $f
echo '== B: no LastBlockHeight = 500'; sed -i '612{/LastBlockHeight = 500/d}' $f; git diff --stat -- $f; run; git checkout -- $f
