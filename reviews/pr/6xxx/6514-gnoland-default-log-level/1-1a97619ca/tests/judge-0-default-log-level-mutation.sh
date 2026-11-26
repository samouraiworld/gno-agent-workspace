#!/usr/bin/env bash
# Asserts TestStart_DefaultLogLevel goes red when the -log-level default is put back to debug.
# Measurement: PASS at the reviewed head, FAIL (expected "info", actual "debug") with start.go:155 reverted.
# from a local clone of gnolang/gno:
#   gh pr checkout 6514 -R gnolang/gno
#   bash judge-0-default-log-level-mutation.sh
set -u
go test -count=1 -run 'TestStart_DefaultLogLevel$' ./gno.land/cmd/gnoland/ || exit 1
sed -i 's/zapcore.InfoLevel.String(),/zapcore.DebugLevel.String(),/' gno.land/cmd/gnoland/start.go
if go test -count=1 -run 'TestStart_DefaultLogLevel$' ./gno.land/cmd/gnoland/; then echo "MUTANT SURVIVED"; rc=1; else echo "MUTANT KILLED"; rc=0; fi
git checkout -- gno.land/cmd/gnoland/start.go
exit $rc
