#!/bin/sh
# PR 6061 round 2: TestReadMemInfo fails inside a cgroup v2 scope under 256 MiB.
# ReadMemInfo now narrows Total to the cgroup's memory.max, and the test still
# asserts Total > 256 MiB as a check that the figure is in bytes.
#
# Repro from a plain clone of gnolang/gno at 115d118c7, on a cgroup v2 host with
# a systemd user manager, from the repository root:
#   go test -c -o /tmp/tu.test ./tm2/pkg/testutils
#   systemd-run --user --scope -q -p MemoryMax=200M /tmp/tu.test -test.run 'TestReadMemInfo$' -test.v
#
# Observed with go1.25.9 on linux/amd64:
#   Error:      "209715200" is not greater than "268435456"
#   Messages:   total looks like it is not in bytes
#   --- FAIL: TestReadMemInfo (0.00s)
set -e
go test -c -o /tmp/tu.test ./tm2/pkg/testutils
systemd-run --user --scope -q -p MemoryMax=200M /tmp/tu.test -test.run 'TestReadMemInfo$' -test.v
