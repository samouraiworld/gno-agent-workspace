#!/usr/bin/env bash
# Asserts: a narrow `userResolver interface { Eval(...) }` beside userExists cannot
# replace the Eval method on the exported ClientAdapter — h.Client is typed
# ClientAdapter, so the assignment fails to compile once Eval leaves that interface.
# Measured: one build error at handler_http.go, 0 lines saved, at the reviewed head.
#
# from a local clone of gnolang/gno:
#   gh pr checkout 6206 -R gnolang/gno && git checkout 876762bdf2ea6635b27e2b0a42f26f9fdc54af24
#   bash judge-22-userresolver-subset.sh
set -u

python3 - <<'PY'
p = 'gno.land/pkg/gnoweb/client.go'
s = open(p).read()
decl = '''
	// Eval evaluates a read-only Gno expression (`vm/qeval`) and returns the
	// raw result, one line per return value. The node splits pkgPath from expr
	// on the first dot, so pkgPath carries none, and expr is the caller's to
	// keep safe.
	Eval(ctx context.Context, pkgPath, expr string) ([]byte, error)
'''
assert decl in s, 'Eval declaration not found in ClientAdapter'
open(p, 'w').write(s.replace(decl, ''))   # the finding's ask: Eval off the wide interface

p2 = 'gno.land/pkg/gnoweb/handler_http.go'
t = open(p2).read()
old = '	res, err := h.Client.Eval(ctx, "/r/sys/users", fmt.Sprintf("ResolveName(%q)", username))'
assert old in t
t = t.replace(old, '''	// the proposed house form: a narrow consumer subset at the call site
	var resolver userResolver = h.Client
	res, err := resolver.Eval(ctx, "/r/sys/users", fmt.Sprintf("ResolveName(%q)", username))''')
t = t.replace('const maxUsernameLen = 64', '''const maxUsernameLen = 64

// userResolver is the subset of ClientAdapter the user page depends on.
type userResolver interface {
	Eval(ctx context.Context, pkgPath, expr string) ([]byte, error)
}''')
open(p2, 'w').write(t)
PY

# -run matches nothing: the package still compiles, which is the whole assertion.
go test -run 'TestNothingZZZ$' ./gno.land/pkg/gnoweb/ 2>&1 |
	grep -E 'gnoweb/handler_http\.go|^ok|^FAIL' | head -5

git checkout -- gno.land/pkg/gnoweb/client.go gno.land/pkg/gnoweb/handler_http.go
