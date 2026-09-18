#!/usr/bin/env python3
# Asserts that `gnopreview plan -changed -` reads its list from whatever stdin
# is, not only from a stdin that /dev/stdin can be reopened on. Measured at
# ecf7af0f2: a pipe works, a socket fails with "open /dev/stdin: no such device
# or address", because readLines does os.ReadFile("/dev/stdin") (main.go:310)
# instead of io.ReadAll(os.Stdin). Fails at the reviewed head on the socket case.
#
# From a plain clone:
#
#   git clone https://github.com/gnolang/gno && cd gno
#   git fetch origin pull/6194/head && git checkout ecf7af0f29abe4737a52803d672bc5a33c17cc60
#   (cd misc/gnopreview && go build -o /tmp/gnopreview .)
#   python3 <this file> /tmp/gnopreview .
#
import socket, subprocess, sys

binary, root = sys.argv[1], sys.argv[2]
line = b"gno.land/pkg/gnoweb/gnoweb.go\n"
cmd = [binary, "plan", "-changed", "-", "-root", root]

def report(name, rc, out, err):
    print(f"--- {name}: exit={rc}")
    print("    stdout:", (out.decode().strip().replace("\n", " ")[:120] or "(empty)"))
    print("    stderr:", (err.decode().strip()[:120] or "(empty)"))

p = subprocess.run(cmd, input=line, capture_output=True)   # stdin is a pipe
report("pipe stdin", p.returncode, p.stdout, p.stderr)
pipe_ok = p.returncode == 0

a, b = socket.socketpair()                                  # stdin is a socket
b.sendall(line); b.shutdown(socket.SHUT_WR)
p = subprocess.run(cmd, stdin=a, capture_output=True)
report("socket stdin", p.returncode, p.stdout, p.stderr)
sock_ok = p.returncode == 0

# IS:     the socket case fails at ecf7af0f2 (os.ReadFile("/dev/stdin"))
# SHOULD: both read the same one-line list (io.ReadAll(os.Stdin))
print("RESULT:", "PASS" if (pipe_ok and sock_ok) else "FAIL")
sys.exit(0 if (pipe_ok and sock_ok) else 1)
