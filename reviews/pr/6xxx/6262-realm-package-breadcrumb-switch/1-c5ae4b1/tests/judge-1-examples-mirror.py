# judge-1: mirrors counterpartTarget (counterpart.go:75-117 at c5ae4b13a) over every
# package of examples/gno.land and counts the pages whose switch reads "N matching"
# and the ones whose target is itself a package. Measured: 1 and 0.
# Run from a gno checkout: python3 judge-1-examples-mirror.py
import os, posixpath as pp
root = 'examples/gno.land'
pk = {'/' + os.path.relpath(d, root) for d, _, f in os.walk(root) if 'gnomod.toml' in f}
def under(p, d): return p == d or p.startswith(d + '/')
def target(twin, rt, paths):
    m = [p for p in paths if under(p, rt)]
    if twin in m:
        dd = pp.dirname(twin); n = sum(1 for x in m if pp.dirname(x) == dd)
        return (twin, 1) if n == 1 else (dd, n)
    d = twin
    while True:
        c = [x for x in m if under(x, d)]
        if len(c) == 1: return c[0], 1
        if len(c) > 1: return d, len(c)
        if d == rt: return '', 0
        d = pp.dirname(d)
total, hits = 0, []
for p in sorted(pk):
    k, _, rest = p[1:].partition('/'); segs = rest.split('/')
    if k not in ('r', 'p') or len(segs) < 2: continue
    o = 'p' if k == 'r' else 'r'
    t, n = target('/' + o + '/' + rest, '/' + o + '/' + '/'.join(segs[:2]), pk)
    if n > 1:
        total += 1
        if t in pk: hits.append((p, t, n))
print('pages with an N-matching link:', total, ' of which target is itself a package:', len(hits))
