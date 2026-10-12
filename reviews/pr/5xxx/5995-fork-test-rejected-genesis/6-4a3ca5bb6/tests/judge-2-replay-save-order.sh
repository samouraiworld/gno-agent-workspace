#!/usr/bin/env bash
# Asserts TestHandshakeInitChainError pins "leaves the genesis ABCI responses unsaved": saving the
# height-0 responses before the res.IsErr() return must redden both subtests. Measured at 4a3ca5bb6:
# both subtests FAIL 'genesis ABCI responses must not be saved after a rejected InitChain'.
set -u
p=tm2/pkg/bft/consensus/replay.go
python3 - <<'PY'
p='tm2/pkg/bft/consensus/replay.go'; s=open(p).read(); old='\t\tif res.IsErr() {'
assert s.count(old)==1
open(p,'w').write(s.replace(old,'\t\tsm.SaveABCIResponses(h.stateDB, 0, sm.NewABCIResponsesFromNum(0))\n'+old,1))
PY
go test ./tm2/pkg/bft/consensus -run 'TestHandshakeInitChainError' -count=1 2>&1 | grep -E '^\s*(--- |ok|FAIL)|Messages'
git checkout -- $p
